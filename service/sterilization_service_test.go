package service

import (
	"errors"
	"testing"

	"example.com/cssd-instrument-trace-service/domain"
)

// prepareSterilizingPacks 登记并推进两个器械包到「已清洗」。
func prepareSterilizingPacks(t *testing.T, svc *Services, barcodes ...string) []string {
	t.Helper()
	ids := make([]string, 0, len(barcodes))
	for _, bc := range barcodes {
		id, _ := registerTestPack(t, svc, bc, "灭菌测试包", domain.TypeSurgical)
		forwardToWashed(t, svc, id)
		ids = append(ids, id)
	}
	return ids
}

func TestCreateBatchRejectsNonWashedPack(t *testing.T) {
	svc := newTestServices(t)
	id, _ := registerTestPack(t, svc, "PK-NW", "未清洗包", domain.TypeSurgical)
	_, err := svc.Sterilizations.CreateBatch(domain.CreateBatchInput{
		SterilizerID: "ster_001", TempC: 134, DurationMin: 5, PressureKPa: 208,
		PackIDs: []string{id},
	}, testActor)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("未清洗器械包装载应拒绝，实际 err=%v", err)
	}
}

func TestCreateBatchRejectsMaintenanceSterilizer(t *testing.T) {
	svc := newTestServices(t)
	id, _ := registerTestPack(t, svc, "PK-MT", "维护灭菌器包", domain.TypeSurgical)
	forwardToWashed(t, svc, id)
	_, err := svc.Sterilizations.CreateBatch(domain.CreateBatchInput{
		SterilizerID: "ster_003", TempC: 134, DurationMin: 5, PressureKPa: 208,
		PackIDs: []string{id},
	}, testActor)
	if !errors.Is(err, domain.ErrSterilizerUnavailable) {
		t.Errorf("维护中灭菌器应拒绝，实际 err=%v", err)
	}
}

func TestCompleteBatchPass(t *testing.T) {
	svc := newTestServices(t)
	ids := prepareSterilizingPacks(t, svc, "PK-P1", "PK-P2")

	batch, err := svc.Sterilizations.CreateBatch(domain.CreateBatchInput{
		SterilizerID: "ster_001", TempC: 134, DurationMin: 5, PressureKPa: 208,
		PackIDs: ids,
	}, testActor)
	if err != nil {
		t.Fatalf("创建批次失败: %v", err)
	}
	if batch.Result != "" {
		t.Errorf("创建后批次不应有判定结果: %s", batch.Result)
	}

	completed, err := svc.Sterilizations.CompleteBatch(batch.ID, testActor)
	if err != nil {
		t.Fatalf("完成批次失败: %v", err)
	}
	if completed.Result != domain.ResultPass {
		t.Errorf("参数达标应判定 pass，实际 %s reasons=%v", completed.Result, completed.FailReasons)
	}
	for _, pid := range ids {
		pack, err := svc.Packs.Get(pid)
		if err != nil {
			t.Fatal(err)
		}
		if pack.Stage != domain.StageSterilized {
			t.Errorf("器械包 %s 灭菌后环节 = %s, want sterilized", pack.Barcode, pack.Stage)
		}
		if pack.ExpiryAt == nil {
			t.Errorf("器械包 %s 未计算有效期", pack.Barcode)
		}
	}
	// 重复完结应冲突。
	if _, err := svc.Sterilizations.CompleteBatch(batch.ID, testActor); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("重复完结应冲突，实际 err=%v", err)
	}
}

func TestCompleteBatchFailInterceptsPacks(t *testing.T) {
	svc := newTestServices(t)
	ids := prepareSterilizingPacks(t, svc, "PK-F1")

	batch, err := svc.Sterilizations.CreateBatch(domain.CreateBatchInput{
		SterilizerID: "ster_001", TempC: 120, DurationMin: 2, PressureKPa: 180,
		PackIDs: ids,
	}, testActor)
	if err != nil {
		t.Fatalf("创建批次失败: %v", err)
	}
	completed, err := svc.Sterilizations.CompleteBatch(batch.ID, testActor)
	if err != nil {
		t.Fatalf("完成批次失败: %v", err)
	}
	if completed.Result != domain.ResultFail || len(completed.FailReasons) != 3 {
		t.Errorf("参数不达标应判定 fail 并收集 3 条原因，实际 %s %v", completed.Result, completed.FailReasons)
	}
	pack, _ := svc.Packs.Get(ids[0])
	if pack.Stage != domain.StageWashed {
		t.Errorf("灭菌失败器械包应退回已清洗，实际 %s", pack.Stage)
	}
	if pack.LastBatchResult != domain.ResultFail {
		t.Errorf("灭菌失败标记未写入: %+v", pack)
	}
	// 失败批次器械包不得发放。
	_, err = svc.Issues.Issue(ids[0], domain.IssueInput{Department: "手术室", OperatingRoom: "1号", Issuer: "护士"}, testActor)
	if !errors.Is(err, domain.ErrIssueBlocked) && !IsBlocked(err) {
		t.Errorf("灭菌失败器械包发放应拦截，实际 err=%v", err)
	}
}
