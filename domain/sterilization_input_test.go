package domain

import "testing"

// TestNewBatchCopiesPackIDsInput 验证 NewBatch 保存入参切片副本：
// 创建批次后修改原始 PackIDs 切片，批次内 PackIDs 必须保持不变。
func TestNewBatchCopiesPackIDsInput(t *testing.T) {
	packIDs := []string{"pack_1", "pack_2"}
	b := NewBatch(CreateBatchInput{
		SterilizerID: "ster_001",
		TempC:        134,
		DurationMin:  5,
		PressureKPa:  208,
		PackIDs:      packIDs,
	}, "高温高压灭菌器 A", "SB20260826-001")

	packIDs[0] = "pack_evil"
	packIDs = append(packIDs, "pack_extra")

	if len(b.PackIDs) != 2 || b.PackIDs[0] != "pack_1" {
		t.Fatalf("NewBatch 保存了入参切片引用，输入修改被污染: %v", b.PackIDs)
	}
}
