package store

import (
	"testing"

	"example.com/cssd-instrument-trace-service/domain"
)

// newTestBatch 构造一个测试批次。
func newTestBatch(id string) *domain.SterilizationBatch {
	return domain.NewBatch(domain.CreateBatchInput{
		SterilizerID: "ster_001", Operator: "测试员",
		TempC: 134, DurationMin: 5, PressureKPa: 208,
		PackIDs: []string{"pack_a", "pack_b"},
	}, "高温高压灭菌器 A", "SB"+id)
}

// TestStoreBatchGettersReturnCopies 验证 GetBatch 返回独立副本：
// 修改返回值后再次读取，库内数据必须保持不变。
func TestStoreBatchGettersReturnCopies(t *testing.T) {
	st := newTestStore(t)
	b := newTestBatch("T1")
	if err := st.SaveBatch(b); err != nil {
		t.Fatalf("保存批次失败: %v", err)
	}
	got, err := st.GetBatch(b.ID)
	if err != nil {
		t.Fatalf("读取批次失败: %v", err)
	}
	got.Result = domain.ResultFail
	got.Status = domain.BatchCompleted
	got.FailReasons = append(got.FailReasons, "篡改原因")
	got.PackIDs = append(got.PackIDs, "pack_evil")

	again, err := st.GetBatch(b.ID)
	if err != nil {
		t.Fatalf("二次读取批次失败: %v", err)
	}
	if again.Result != "" || again.Status != domain.BatchPending {
		t.Fatalf("GetBatch 返回值与库内共享引用，篡改被污染: result=%q status=%q", again.Result, again.Status)
	}
	if len(again.FailReasons) != 0 || len(again.PackIDs) != 2 {
		t.Fatalf("GetBatch 切片被污染: reasons=%v packIDs=%v", again.FailReasons, again.PackIDs)
	}
}

// TestStoreBatchListReturnsCopies 验证 ListBatchesPage 返回独立副本：
// 修改列表元素后再次读取，库内数据必须保持不变。
func TestStoreBatchListReturnsCopies(t *testing.T) {
	st := newTestStore(t)
	b := newTestBatch("T2")
	if err := st.SaveBatch(b); err != nil {
		t.Fatalf("保存批次失败: %v", err)
	}
	list := st.ListBatchesPage(10, 0)
	if len(list) != 1 {
		t.Fatalf("期望 1 个批次，实际 %d", len(list))
	}
	list[0].TempC = 1
	list[0].Status = domain.BatchCompleted
	list[0].PackIDs = append(list[0].PackIDs, "pack_evil")

	again, err := st.GetBatch(b.ID)
	if err != nil {
		t.Fatalf("二次读取批次失败: %v", err)
	}
	if again.TempC != 134 || again.Status != domain.BatchPending {
		t.Fatalf("ListBatchesPage 返回值与库内共享引用: tempC=%v status=%q", again.TempC, again.Status)
	}
	if len(again.PackIDs) != 2 {
		t.Fatalf("ListBatchesPage 切片被污染: %v", again.PackIDs)
	}
}

// TestStoreSaveBatchStoresCopy 验证 SaveBatch 保存副本：
// 保存后修改原始对象，库内数据必须保持不变。
func TestStoreSaveBatchStoresCopy(t *testing.T) {
	st := newTestStore(t)
	b := newTestBatch("T3")
	if err := st.SaveBatch(b); err != nil {
		t.Fatalf("保存批次失败: %v", err)
	}
	b.Result = domain.ResultFail
	b.Status = domain.BatchCompleted
	b.FailReasons = append(b.FailReasons, "保存后篡改")
	b.PackIDs = append(b.PackIDs, "pack_evil")

	again, err := st.GetBatch(b.ID)
	if err != nil {
		t.Fatalf("二次读取批次失败: %v", err)
	}
	if again.Result != "" || again.Status != domain.BatchPending {
		t.Fatalf("SaveBatch 保存了调用方引用: result=%q status=%q", again.Result, again.Status)
	}
	if len(again.FailReasons) != 0 || len(again.PackIDs) != 2 {
		t.Fatalf("SaveBatch 切片被污染: reasons=%v packIDs=%v", again.FailReasons, again.PackIDs)
	}
}

// TestStoreUpdateBatchRollbackOnError 验证 UpdateBatch 失败回滚：
// fn 修改后返回错误时，库内数据必须保持原状。
func TestStoreUpdateBatchRollbackOnError(t *testing.T) {
	st := newTestStore(t)
	b := newTestBatch("T4")
	if err := st.SaveBatch(b); err != nil {
		t.Fatalf("保存批次失败: %v", err)
	}
	err := st.UpdateBatch(b.ID, func(cp *domain.SterilizationBatch) error {
		cp.Status = domain.BatchCompleted
		cp.Result = domain.ResultFail
		return domain.ErrConflict
	})
	if err == nil {
		t.Fatal("UpdateBatch 应返回错误")
	}
	again, err := st.GetBatch(b.ID)
	if err != nil {
		t.Fatalf("二次读取批次失败: %v", err)
	}
	if again.Status != domain.BatchPending || again.Result != "" {
		t.Fatalf("UpdateBatch 失败后未回滚: status=%q result=%q", again.Status, again.Result)
	}
}
