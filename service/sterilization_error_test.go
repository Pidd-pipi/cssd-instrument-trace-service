package service

import (
	"errors"
	"testing"

	"example.com/cssd-instrument-trace-service/domain"
)

// TestCreateBatchUnknownSterilizerErrorsIsInvalidParam 创建批次使用不存在的灭菌器时，
// 错误必须能被 errors.Is 判定为参数不合法，不能断链成系统错误。
func TestCreateBatchUnknownSterilizerErrorsIsInvalidParam(t *testing.T) {
	svc := newTestServices(t)
	_, err := svc.Sterilizations.CreateBatch(domain.CreateBatchInput{
		SterilizerID: "ster_missing",
		Operator:     "测试员",
		TempC:        134,
		DurationMin:  5,
		PressureKPa:  208,
		PackIDs:      []string{"pack_1"},
	}, testActor)
	if err == nil {
		t.Fatal("应返回错误")
	}
	if !errors.Is(err, domain.ErrInvalidParam) {
		t.Fatalf("错误链应可判定为 ErrInvalidParam，实际 %v", err)
	}
}

// TestCreateBatchNotWashedPackErrorsIsInvalidTransition 装载非「已清洗」器械包时，
// 错误必须能被 errors.Is 判定为非法流转。
func TestCreateBatchNotWashedPackErrorsIsInvalidTransition(t *testing.T) {
	svc := newTestServices(t)
	id, _ := registerTestPack(t, svc, "NW-001", "未清洗测试包", domain.TypeSurgical)
	// 只推进到「已回收」，停留在非「已清洗」环节。
	if _, err := svc.Packs.Cycle(id, domain.StageCollected, "dev_001", "", testActor); err != nil {
		t.Fatalf("推进环节失败: %v", err)
	}
	_, err := svc.Sterilizations.CreateBatch(domain.CreateBatchInput{
		SterilizerID: "ster_001",
		Operator:     "测试员",
		TempC:        134,
		DurationMin:  5,
		PressureKPa:  208,
		PackIDs:      []string{id},
	}, testActor)
	if err == nil {
		t.Fatal("应返回错误")
	}
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("错误链应可判定为 ErrInvalidTransition，实际 %v", err)
	}
}

// TestCompleteBatchMissingPackFails 批次内器械包缺失时整批必须失败，
// 不能静默跳过并把批次判定为合格。
func TestCompleteBatchMissingPackFails(t *testing.T) {
	svc := newTestServices(t)
	id, _ := registerTestPack(t, svc, "MP-001", "缺失包测试", domain.TypeSurgical)
	forwardToWashed(t, svc, id)
	// 直接推进到「灭菌中」，并构造含缺失器械包的批次。
	if err := svc.Store.UpdatePack(id, func(p *domain.InstrumentPack) error {
		p.Stage = domain.StageSterilizing
		p.Touch()
		return nil
	}); err != nil {
		t.Fatalf("推进灭菌中失败: %v", err)
	}
	batch := domain.NewBatch(domain.CreateBatchInput{
		SterilizerID: "ster_001",
		Operator:     "测试员",
		TempC:        134,
		DurationMin:  5,
		PressureKPa:  208,
		PackIDs:      []string{id, "missing-pack-id"},
	}, "高温高压灭菌器 A", "SB20260826-888")
	if err := svc.Store.SaveBatch(batch); err != nil {
		t.Fatalf("保存批次失败: %v", err)
	}

	_, err := svc.Sterilizations.CompleteBatch(batch.ID, testActor)
	if err == nil {
		t.Fatal("包含缺失器械包的批次必须失败，不能静默跳过")
	}
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("错误链应可判定为 ErrConflict，实际 %v", err)
	}
	got, err := svc.Store.GetBatch(batch.ID)
	if err != nil {
		t.Fatalf("读取批次失败: %v", err)
	}
	if got.Status == domain.BatchCompleted {
		t.Fatalf("批次不应被判定完成: %+v", got)
	}
}

// TestCompleteBatchAtomicPrevalidation 批次内存在非「灭菌中」器械包时，
// 整批必须原子失败，前面合法的器械包不能被半途灭菌。
func TestCompleteBatchAtomicPrevalidation(t *testing.T) {
	svc := newTestServices(t)
	valid, _ := registerTestPack(t, svc, "AT-001", "合法灭菌中包", domain.TypeSurgical)
	forwardToWashed(t, svc, valid)
	bad, _ := registerTestPack(t, svc, "AT-002", "非灭菌中包", domain.TypeSurgical)
	forwardToWashed(t, svc, bad)
	// 合法包推进到灭菌中，坏包停留在已清洗。
	if err := svc.Store.UpdatePack(valid, func(p *domain.InstrumentPack) error {
		p.Stage = domain.StageSterilizing
		p.Touch()
		return nil
	}); err != nil {
		t.Fatalf("推进灭菌中失败: %v", err)
	}
	batch := domain.NewBatch(domain.CreateBatchInput{
		SterilizerID: "ster_001",
		Operator:     "测试员",
		TempC:        134,
		DurationMin:  5,
		PressureKPa:  208,
		PackIDs:      []string{valid, bad},
	}, "高温高压灭菌器 A", "SB20260826-999")
	if err := svc.Store.SaveBatch(batch); err != nil {
		t.Fatalf("保存批次失败: %v", err)
	}

	_, err := svc.Sterilizations.CompleteBatch(batch.ID, testActor)
	if err == nil {
		t.Fatal("包含非灭菌中器械包的批次必须失败")
	}
	// 合法包必须仍停留在「灭菌中」，不能被半途灭菌。
	got, err := svc.Store.GetPack(valid)
	if err != nil {
		t.Fatalf("读取器械包失败: %v", err)
	}
	if got.Stage != domain.StageSterilizing {
		t.Fatalf("合法器械包被半途灭菌，整批非原子: %+v", got)
	}
}
