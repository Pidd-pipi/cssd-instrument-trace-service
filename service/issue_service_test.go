package service

import (
	"errors"
	"testing"
	"time"

	"example.com/cssd-instrument-trace-service/domain"
)

// sterilizeAndGetReadyPack 登记并完成一轮灭菌，返回处于「已灭菌」的器械包。
func sterilizeAndGetReadyPack(t *testing.T, svc *Services, barcode string) string {
	t.Helper()
	id, _ := registerTestPack(t, svc, barcode, "发放测试包", domain.TypeSurgical)
	forwardToWashed(t, svc, id)
	batch, err := svc.Sterilizations.CreateBatch(domain.CreateBatchInput{
		SterilizerID: "ster_001", TempC: 134, DurationMin: 5, PressureKPa: 208,
		PackIDs: []string{id},
	}, testActor)
	if err != nil {
		t.Fatalf("创建批次失败: %v", err)
	}
	if _, err := svc.Sterilizations.CompleteBatch(batch.ID, testActor); err != nil {
		t.Fatalf("完成批次失败: %v", err)
	}
	return id
}

func TestIssueValidations(t *testing.T) {
	svc := newTestServices(t)
	id := sterilizeAndGetReadyPack(t, svc, "PK-ISSUE")

	// 正常发放。
	rec, err := svc.Issues.Issue(id, domain.IssueInput{Department: "普外科", OperatingRoom: "3号手术间", Issuer: "王护士"}, testActor)
	if err != nil {
		t.Fatalf("发放失败: %v", err)
	}
	if rec.Status != domain.IssueOpen {
		t.Errorf("发放记录状态 = %s, want issued", rec.Status)
	}
	pack, _ := svc.Packs.Get(id)
	if pack.Stage != domain.StageIssued {
		t.Errorf("发放后环节 = %s, want issued", pack.Stage)
	}

	// 重复发放应冲突（已有未回收记录）。
	_, err = svc.Issues.Issue(id, domain.IssueInput{Department: "普外科", OperatingRoom: "3号", Issuer: "王护士"}, testActor)
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("重复发放应冲突，实际 err=%v", err)
	}
}

func TestIssueBlockedForUnsterilizedPack(t *testing.T) {
	svc := newTestServices(t)
	id, _ := registerTestPack(t, svc, "PK-RAW", "未灭菌包", domain.TypeSurgical)
	forwardToWashed(t, svc, id)
	_, err := svc.Issues.Issue(id, domain.IssueInput{Department: "普外科", OperatingRoom: "1号", Issuer: "护士"}, testActor)
	if !errors.Is(err, domain.ErrIssueBlocked) {
		t.Errorf("未灭菌包发放应拦截，实际 err=%v", err)
	}
}

func TestCollectClosedLoop(t *testing.T) {
	svc := newTestServices(t)
	id := sterilizeAndGetReadyPack(t, svc, "PK-LOOP")

	if _, err := svc.Issues.Issue(id, domain.IssueInput{Department: "骨科", OperatingRoom: "2号手术间", Issuer: "李护士"}, testActor); err != nil {
		t.Fatal(err)
	}
	// 已发放 → 使用中。
	if _, err := svc.Packs.Cycle(id, domain.StageInUse, "", "", testActor); err != nil {
		t.Fatal(err)
	}
	// 回收闭环。
	rec, err := svc.Issues.Collect(id, "回收员小张", testActor)
	if err != nil {
		t.Fatalf("回收失败: %v", err)
	}
	if rec.Status != domain.IssueReturned || rec.CollectedAt == nil {
		t.Errorf("回收记录未闭环: %+v", rec)
	}
	pack, _ := svc.Packs.Get(id)
	if pack.Stage != domain.StageToCollect {
		t.Errorf("回收后环节 = %s, want to_collect（回到循环起点）", pack.Stage)
	}
	// 回收后无法重复回收（已非使用中）。
	_, err = svc.Issues.Collect(id, "回收员小张", testActor)
	if !errors.Is(err, domain.ErrCollectBlocked) {
		t.Errorf("非使用中器械包回收应拦截，实际 err=%v", err)
	}
}

func TestCollectRejectsWithoutIssueRecord(t *testing.T) {
	svc := newTestServices(t)
	id := sterilizeAndGetReadyPack(t, svc, "PK-NOI")
	// 直接模拟数据不一致：器械包在使用中但没有任何发放记录。
	if err := svc.Store.UpdatePack(id, func(p *domain.InstrumentPack) error {
		p.Stage = domain.StageInUse
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Issues.Collect(id, "回收员", testActor)
	if !errors.Is(err, domain.ErrCollectBlocked) {
		t.Errorf("无发放记录回收应拦截，实际 err=%v", err)
	}
}

func TestLostListAfterTimeout(t *testing.T) {
	svc := newTestServices(t)
	id := sterilizeAndGetReadyPack(t, svc, "PK-LOST")
	rec, err := svc.Issues.Issue(id, domain.IssueInput{Department: "ICU", OperatingRoom: "5号", Issuer: "护士"}, testActor)
	if err != nil {
		t.Fatal(err)
	}
	// 模拟发放时间超过 24 小时。
	past := time.Now().Add(-25 * time.Hour)
	if err := svc.Store.UpdateIssue(rec.ID, func(r *domain.IssueRecord) error {
		r.IssuedAt = past
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	entries := svc.Issues.LostList(time.Now())
	if len(entries) != 1 {
		t.Fatalf("丢失待查应包含 1 条，实际 %d", len(entries))
	}
	if entries[0].OverdueHours < 24 {
		t.Errorf("超时小时数异常: %.1f", entries[0].OverdueHours)
	}
	after, _ := svc.Store.GetIssue(rec.ID)
	if after.Status != domain.IssueLost {
		t.Errorf("超时记录应标记为 lost，实际 %s", after.Status)
	}
}
