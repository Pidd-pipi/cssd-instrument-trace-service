package store

import (
	"testing"

	"example.com/cssd-instrument-trace-service/domain"
)

// TestListCyclesByPackReturnsCopiedParams 环节记录列表必须返回参数快照的副本：
// 修改返回值中的参数 map，库内记录必须保持不变。
func TestListCyclesByPackReturnsCopiedParams(t *testing.T) {
	st := newTestStore(t)
	pack := domain.NewPack(domain.RegisterPackInput{Barcode: "LS-001", Name: "列表污染测试包", PackType: domain.TypeSurgical})
	if err := st.SavePack(pack); err != nil {
		t.Fatalf("保存器械包失败: %v", err)
	}
	rec := domain.NewCycleRecord(pack, domain.StageToCollect, domain.StageCollected, "测试员", "", "回收", map[string]any{
		"tempC": 134.0,
		"note":  "原参数",
	})
	if err := st.SaveCycle(rec); err != nil {
		t.Fatalf("保存环节记录失败: %v", err)
	}

	cycles := st.ListCyclesByPack(pack.ID)
	if len(cycles) != 1 {
		t.Fatalf("期望 1 条环节记录，实际 %d", len(cycles))
	}
	cycles[0].Params["tempC"] = 1.0
	cycles[0].Params["note"] = "被篡改"

	again := st.ListCyclesByPack(pack.ID)
	if len(again) != 1 {
		t.Fatalf("二次读取环节记录异常: %d", len(again))
	}
	if again[0].Params["tempC"] != 134.0 || again[0].Params["note"] != "原参数" {
		t.Fatalf("ListCyclesByPack 与库内共享参数 map: %v", again[0].Params)
	}
}
