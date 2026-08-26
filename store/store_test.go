package store

import (
	"path/filepath"
	"testing"

	"example.com/cssd-instrument-trace-service/domain"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "store.json"))
	if err != nil {
		t.Fatalf("创建仓储失败: %v", err)
	}
	return st
}

func TestStorePersistenceRoundtrip(t *testing.T) {
	st := newTestStore(t)

	pack := domain.NewPack(domain.RegisterPackInput{
		Barcode: "PK-100", Name: "手术器械包", PackType: domain.TypeSurgical,
		Instruments: []string{"手术剪", "止血钳"},
	})
	if err := st.SavePack(pack); err != nil {
		t.Fatalf("保存器械包失败: %v", err)
	}
	batch := domain.NewBatch(domain.CreateBatchInput{
		SterilizerID: "ster_001", TempC: 134, DurationMin: 5, PressureKPa: 208,
		PackIDs: []string{pack.ID},
	}, "高温高压灭菌器 A", "SB20260825-001")
	if err := st.SaveBatch(batch); err != nil {
		t.Fatalf("保存批次失败: %v", err)
	}
	cycle := domain.NewCycleRecord(pack, domain.StageToCollect, domain.StageCollected, "张三", "", "", nil)
	if err := st.SaveCycle(cycle); err != nil {
		t.Fatalf("保存环节记录失败: %v", err)
	}

	// 重新加载，验证数据完整。
	st2, err := New(st.File())
	if err != nil {
		t.Fatalf("重新加载仓储失败: %v", err)
	}
	got, err := st2.GetPack(pack.ID)
	if err != nil {
		t.Fatalf("重载后查询器械包失败: %v", err)
	}
	if got.Barcode != pack.Barcode || got.Stage != domain.StageToCollect {
		t.Errorf("重载后器械包数据不一致: %+v", got)
	}
	if byBarcode, err := st2.GetPackByBarcode("PK-100"); err != nil || byBarcode.ID != pack.ID {
		t.Errorf("重载后条码索引异常: %+v, err=%v", byBarcode, err)
	}
	gotBatch, err := st2.GetBatch(batch.ID)
	if err != nil || gotBatch.BatchNo != batch.BatchNo {
		t.Errorf("重载后批次数据不一致: %+v, err=%v", gotBatch, err)
	}
	if len(st2.ListCyclesByPack(pack.ID)) != 1 {
		t.Errorf("重载后环节记录缺失")
	}
}

func TestStoreSeedSterilizers(t *testing.T) {
	st := newTestStore(t)
	if len(st.ListSterilizers()) != 3 {
		t.Errorf("首次启动应预置 3 台灭菌器，实际 %d", len(st.ListSterilizers()))
	}
}

func TestPackBarcodeUniqueIndex(t *testing.T) {
	st := newTestStore(t)
	p1 := domain.NewPack(domain.RegisterPackInput{Barcode: "B-1", Name: "包一", PackType: domain.TypeSurgical})
	p2 := domain.NewPack(domain.RegisterPackInput{Barcode: "B-1", Name: "包二", PackType: domain.TypeSurgical})
	if err := st.SavePack(p1); err != nil {
		t.Fatal(err)
	}
	if err := st.SavePack(p2); err != nil {
		t.Fatal(err)
	}
	// 条码索引应指向最后保存的器械包（业务层已拦截重复条码）。
	got, err := st.GetPackByBarcode("B-1")
	if err != nil || got.ID != p2.ID {
		t.Errorf("条码索引未更新: %+v, err=%v", got, err)
	}
}

func TestListPacksFilter(t *testing.T) {
	st := newTestStore(t)
	for _, tc := range []struct{ barcode, typ, stage string }{
		{"A-1", string(domain.TypeSurgical), string(domain.StageToCollect)},
		{"A-2", string(domain.TypeDressing), string(domain.StageWashed)},
		{"A-3", string(domain.TypeSurgical), string(domain.StageWashed)},
	} {
		pack := domain.NewPack(domain.RegisterPackInput{Barcode: tc.barcode, Name: tc.barcode, PackType: domain.PackType(tc.typ)})
		pack.Stage = domain.PackStage(tc.stage)
		if err := st.SavePack(pack); err != nil {
			t.Fatal(err)
		}
	}
	got := st.ListPacks(PackFilter{Stage: string(domain.StageWashed), PackType: string(domain.TypeSurgical)})
	if len(got) != 1 || got[0].Barcode != "A-3" {
		t.Errorf("过滤查询结果异常: %+v", got)
	}
}
