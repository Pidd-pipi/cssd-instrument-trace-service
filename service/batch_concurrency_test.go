package service

import (
	"fmt"
	"sync"
	"testing"

	"example.com/cssd-instrument-trace-service/domain"
)

// TestConcurrentCompleteAndReadNoRace 并发完成灭菌批次（写）与持续读取批次引用（读），
// 必须无数据竞争；库内批次不允许出现半更新状态。
func TestConcurrentCompleteAndReadNoRace(t *testing.T) {
	svc := newTestServices(t)

	const batchCount = 8
	const packPerBatch = 2
	packIDs := make([]string, 0, batchCount*packPerBatch)
	for i := 0; i < batchCount*packPerBatch; i++ {
		id, _ := registerTestPack(t, svc, fmt.Sprintf("CC-%02d", i), "并发测试包", domain.TypeSurgical)
		forwardToWashed(t, svc, id)
		packIDs = append(packIDs, id)
	}

	batchIDs := make([]string, 0, batchCount)
	for i := 0; i < batchCount; i++ {
		batch, err := svc.Sterilizations.CreateBatch(domain.CreateBatchInput{
			SterilizerID: "ster_001",
			Operator:     "测试员",
			TempC:        134,
			DurationMin:  5,
			PressureKPa:  208,
			PackIDs:      packIDs[i*packPerBatch : (i+1)*packPerBatch],
		}, testActor)
		if err != nil {
			t.Fatalf("创建批次失败: %v", err)
		}
		batchIDs = append(batchIDs, batch.ID)
	}

	// 只读参与者：先各取一次批次引用（bug 环境下为库内共享引用），随后持续读字段。
	stop := make(chan struct{})
	refs := make([]*domain.SterilizationBatch, 0, batchCount)
	for _, bid := range batchIDs {
		b, err := svc.Sterilizations.GetBatch(bid)
		if err != nil {
			t.Fatalf("读取批次失败: %v", err)
		}
		refs = append(refs, b)
	}
	var rwg sync.WaitGroup
	rwg.Add(1)
	go func() {
		defer rwg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				for _, b := range refs {
					_ = b.Status
					_ = b.Result
					_ = len(b.FailReasons)
					_ = len(b.PackIDs)
					_ = b.CompletedAt
				}
			}
		}
	}()

	// 两个写参与者并发完成批次（真实 CompleteBatch 调用点）。
	start := make(chan struct{})
	var wg sync.WaitGroup
	half := batchCount / 2
	for w := 0; w < 2; w++ {
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			<-start
			for i := lo; i < hi; i++ {
				if _, err := svc.Sterilizations.CompleteBatch(batchIDs[i], testActor); err != nil {
					t.Errorf("CompleteBatch %s: %v", batchIDs[i], err)
					return
				}
			}
		}(w*half, (w+1)*half)
	}
	close(start)
	wg.Wait()
	close(stop)
	rwg.Wait()

	// 全部批次必须完成且结果合格，不能出现半更新。
	for _, bid := range batchIDs {
		batch, err := svc.Sterilizations.GetBatch(bid)
		if err != nil {
			t.Fatalf("读取批次失败: %v", err)
		}
		if batch.Status != domain.BatchCompleted || batch.Result != domain.ResultPass {
			t.Fatalf("批次 %s 状态异常: status=%q result=%q", batch.BatchNo, batch.Status, batch.Result)
		}
	}
}
