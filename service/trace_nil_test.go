package service

import (
	"errors"
	"testing"

	"example.com/cssd-instrument-trace-service/domain"
)

// TestPackTraceDanglingBatchNoPanic 器械包关联的灭菌批次不存在时，追溯必须正常返回，
// LastBatch 为空，不能空指针崩溃。
func TestPackTraceDanglingBatchNoPanic(t *testing.T) {
	svc := newTestServices(t)
	id, _ := registerTestPack(t, svc, "DB-001", "悬空批次包", domain.TypeSurgical)

	// 人为把 LastBatchID 指向不存在的批次。
	if err := svc.Store.UpdatePack(id, func(p *domain.InstrumentPack) error {
		p.LastBatchID = "missing-batch-001"
		return nil
	}); err != nil {
		t.Fatalf("设置悬空批次失败: %v", err)
	}

	view, err := svc.Trace.PackTrace(id)
	if err != nil {
		t.Fatalf("追溯悬空批次不应报错: %v", err)
	}
	if view.Pack == nil || view.Pack.ID != id {
		t.Fatalf("追溯结果缺少器械包: %+v", view)
	}
	if view.LastBatch != nil {
		t.Fatalf("悬空批次应返回空 LastBatch，实际 %+v", view.LastBatch)
	}
}

// TestBatchPacksMissingPackNoPanic 批次内存在缺失器械包时，批次去向查询必须正常返回，
// 不能空指针崩溃。
func TestBatchPacksMissingPackNoPanic(t *testing.T) {
	svc := newTestServices(t)
	batch := domain.NewBatch(domain.CreateBatchInput{
		SterilizerID: "ster_001",
		Operator:     "测试员",
		TempC:        134,
		DurationMin:  5,
		PressureKPa:  208,
		PackIDs:      []string{"missing-pack-001"},
	}, "高温高压灭菌器 A", "SB20260826-001")
	if err := svc.Store.SaveBatch(batch); err != nil {
		t.Fatalf("保存批次失败: %v", err)
	}

	view, err := svc.Trace.BatchPacks(batch.ID)
	if err != nil {
		t.Fatalf("批次去向查询不应报错: %v", err)
	}
	if view.Batch == nil || view.Batch.ID != batch.ID {
		t.Fatalf("查询结果缺少批次: %+v", view)
	}
	if len(view.Packs) != 0 {
		t.Fatalf("缺失器械包不应出现在去向列表中，实际 %d 条", len(view.Packs))
	}
}

// TestTraceByBarcodeNotFound 按条码追溯不存在时返回 ErrNotFound，不能返回 nil,nil。
func TestTraceByBarcodeNotFound(t *testing.T) {
	svc := newTestServices(t)
	view, err := svc.Trace.TraceByBarcode("NO-SUCH")
	if err == nil {
		t.Fatalf("追溯不存在条码应返回错误，实际 view=%+v", view)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("应返回 ErrNotFound，实际 %v", err)
	}
}
