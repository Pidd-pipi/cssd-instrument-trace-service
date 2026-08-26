package service

import (
	"errors"
	"testing"

	"example.com/cssd-instrument-trace-service/domain"
)

func TestRegisterDuplicateBarcodeRejected(t *testing.T) {
	svc := newTestServices(t)
	registerTestPack(t, svc, "PK-DUP", "手术包", domain.TypeSurgical)
	_, err := svc.Packs.Register(domain.RegisterPackInput{
		Barcode: "PK-DUP", Name: "重复包", PackType: domain.TypeSurgical,
	}, testActor)
	if !errors.Is(err, domain.ErrDuplicateBarcode) {
		t.Errorf("重复条码应拒绝，实际 err=%v", err)
	}
}

func TestCycleRejectsSkip(t *testing.T) {
	svc := newTestServices(t)
	packID, _ := registerTestPack(t, svc, "PK-SKIP", "器械包", domain.TypeSurgical)

	// 待回收 → 清洗中 属于跳步，必须拒绝。
	_, err := svc.Packs.Cycle(packID, domain.StageWashing, "", "", testActor)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("跳步应拒绝，实际 err=%v", err)
	}

	// 正常按序推进。
	if _, err := svc.Packs.Cycle(packID, domain.StageCollected, "", "", testActor); err != nil {
		t.Fatalf("待回收→已回收 应成功: %v", err)
	}
	pack, _ := svc.Packs.Get(packID)
	if pack.Stage != domain.StageCollected {
		t.Errorf("推进后环节 = %s, want collected", pack.Stage)
	}
}

func TestCycleRejectsReservedTransitions(t *testing.T) {
	svc := newTestServices(t)
	packID, _ := registerTestPack(t, svc, "PK-RES", "器械包", domain.TypeSurgical)
	for _, stage := range []domain.PackStage{domain.StageCollected, domain.StageWashing, domain.StageWashed} {
		if _, err := svc.Packs.Cycle(packID, stage, "", "", testActor); err != nil {
			t.Fatalf("推进到 %s 失败: %v", stage, err)
		}
	}
	// 已清洗 → 灭菌中 必须走批次创建接口。
	_, err := svc.Packs.Cycle(packID, domain.StageSterilizing, "", "", testActor)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("已清洗→灭菌中 不应走通用流转，实际 err=%v", err)
	}
}

func TestCycleRecordsAuditAndTrace(t *testing.T) {
	svc := newTestServices(t)
	packID, barcode := registerTestPack(t, svc, "PK-AUD", "器械包", domain.TypeSurgical)
	if _, err := svc.Packs.Cycle(packID, domain.StageCollected, "dev_9", "回收完成", testActor); err != nil {
		t.Fatal(err)
	}
	view, err := svc.Trace.PackTrace(packID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Cycles) != 2 { // 登记 + 流转
		t.Errorf("环节记录数 = %d, want 2", len(view.Cycles))
	}
	// 登记记录：空 → to_collect；流转记录：to_collect → collected。
	if view.Cycles[0].FromStage != "" || view.Cycles[0].Stage != domain.StageToCollect {
		t.Errorf("登记环节记录异常: %+v", view.Cycles[0])
	}
	if view.Cycles[1].FromStage != domain.StageToCollect || view.Cycles[1].Stage != domain.StageCollected {
		t.Errorf("流转环节记录异常: %+v", view.Cycles[1])
	}
	if view.Pack.Barcode != barcode {
		t.Errorf("追溯视图条码不一致: %s", view.Pack.Barcode)
	}
	if len(svc.Audit.List(100)) < 2 {
		t.Error("审计日志未正确写入")
	}
}
