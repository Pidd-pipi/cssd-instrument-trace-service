package service

import (
	"testing"

	"example.com/cssd-instrument-trace-service/domain"
)

// setStage 直接把器械包拨到指定环节。
func setStage(t *testing.T, svc *Services, packID string, stage domain.PackStage) {
	t.Helper()
	if err := svc.Store.UpdatePack(packID, func(p *domain.InstrumentPack) error {
		p.Stage = stage
		p.Touch()
		return nil
	}); err != nil {
		t.Fatalf("设置环节 %s 失败: %v", stage, err)
	}
}

// TestManualCycleRejectsSterilizing 已清洗→灭菌中必须走批次创建，通用流转必须拒绝。
func TestManualCycleRejectsSterilizing(t *testing.T) {
	svc := newTestServices(t)
	id, _ := registerTestPack(t, svc, "SM-001", "跳步灭菌测试包", domain.TypeSurgical)
	forwardToWashed(t, svc, id)
	if _, err := svc.Packs.Cycle(id, domain.StageSterilizing, "dev_001", "", testActor); err == nil {
		t.Fatal("已清洗→灭菌中不应允许通过通用流转推进")
	}
}

// TestManualCycleRejectsIssued 已灭菌→已发放必须走发放接口，通用流转必须拒绝。
func TestManualCycleRejectsIssued(t *testing.T) {
	svc := newTestServices(t)
	id, _ := registerTestPack(t, svc, "SM-002", "跳步发放测试包", domain.TypeSurgical)
	setStage(t, svc, id, domain.StageSterilized)
	if _, err := svc.Packs.Cycle(id, domain.StageIssued, "", "", testActor); err == nil {
		t.Fatal("已灭菌→已发放不应允许通过通用流转推进")
	}
}

// TestManualCycleRejectsSterilized 灭菌中→已灭菌必须走批次完结，通用流转必须拒绝。
func TestManualCycleRejectsSterilized(t *testing.T) {
	svc := newTestServices(t)
	id, _ := registerTestPack(t, svc, "SM-003", "跳步完结测试包", domain.TypeSurgical)
	setStage(t, svc, id, domain.StageSterilizing)
	if _, err := svc.Packs.Cycle(id, domain.StageSterilized, "", "", testActor); err == nil {
		t.Fatal("灭菌中→已灭菌不应允许通过通用流转推进")
	}
}

// TestManualCycleRejectsToCollect 使用中→待回收必须走回收接口闭环发放记录，通用流转必须拒绝。
func TestManualCycleRejectsToCollect(t *testing.T) {
	svc := newTestServices(t)
	id, _ := registerTestPack(t, svc, "SM-004", "跳步回收测试包", domain.TypeSurgical)
	setStage(t, svc, id, domain.StageInUse)
	if _, err := svc.Packs.Cycle(id, domain.StageToCollect, "", "", testActor); err == nil {
		t.Fatal("使用中→待回收不应允许通过通用流转推进")
	}
}

// TestExpiredPackCanReprocess 已过期器械包必须能通过通用流转重新进入清洗流程。
func TestExpiredPackCanReprocess(t *testing.T) {
	svc := newTestServices(t)
	id, _ := registerTestPack(t, svc, "EX-001", "过期重洗测试包", domain.TypeSurgical)
	setStage(t, svc, id, domain.StageExpired)
	if _, err := svc.Packs.Cycle(id, domain.StageWashing, "dev_001", "", testActor); err != nil {
		t.Fatalf("已过期器械包应能重新进入清洗流程: %v", err)
	}
}

// TestCycleRecordKeepsOriginalFromStage 环节记录必须保留流转前的原始环节，
// 不能把迁移前环节写成目标环节。
func TestCycleRecordKeepsOriginalFromStage(t *testing.T) {
	svc := newTestServices(t)
	id, _ := registerTestPack(t, svc, "TR-001", "追溯环节测试包", domain.TypeSurgical)
	if _, err := svc.Packs.Cycle(id, domain.StageCollected, "dev_001", "", testActor); err != nil {
		t.Fatalf("推进到已回收失败: %v", err)
	}
	cycles := svc.Store.ListCyclesByPack(id)
	if len(cycles) < 2 {
		t.Fatalf("期望至少 2 条环节记录，实际 %d", len(cycles))
	}
	last := cycles[len(cycles)-1]
	if last.FromStage != domain.StageToCollect || last.Stage != domain.StageCollected {
		t.Fatalf("环节记录前后环节错位: from=%s to=%s", last.FromStage, last.Stage)
	}
}
