package domain

import (
	"testing"
)

// TestNewCycleRecordCopiesParams 环节记录必须拷贝参数快照：
// 创建记录后修改原始 map，记录内参数必须保持不变。
func TestNewCycleRecordCopiesParams(t *testing.T) {
	params := map[string]any{"tempC": 134.0, "durationMin": 5}
	pack := NewPack(RegisterPackInput{Barcode: "CY-001", Name: "参数污染测试包", PackType: TypeInstrument})
	rec := NewCycleRecord(pack, StageWashed, StageSterilizing, "测试员", "dev_001", "装载灭菌批次", params)

	params["tempC"] = 1.0
	params["durationMin"] = 1
	params["extra"] = "多出来的参数"

	if rec.Params["tempC"] != 134.0 {
		t.Fatalf("NewCycleRecord 保存了入参 map 引用，输入修改被污染: %v", rec.Params)
	}
	if rec.Params["durationMin"] != 5 {
		t.Fatalf("NewCycleRecord 参数被污染: %v", rec.Params)
	}
	if _, ok := rec.Params["extra"]; ok {
		t.Fatalf("NewCycleRecord 参数混入调用方新增键: %v", rec.Params)
	}
}
