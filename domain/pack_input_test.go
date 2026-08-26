package domain

import (
	"testing"
	"time"
)

// TestRegisterPackCopiesInstrumentsInput 登记入参的器械清单必须被拷贝：
// 登记后修改原始切片，库内器械包清单必须保持不变。
func TestRegisterPackCopiesInstrumentsInput(t *testing.T) {
	instruments := []string{"手术剪", "止血钳"}
	pack := NewPack(RegisterPackInput{
		Barcode:     "IN-001",
		Name:        "输入污染测试包",
		PackType:    TypeSurgical,
		Instruments: instruments,
	})

	instruments[0] = "被篡改的器械"
	instruments = append(instruments, "多余器械")

	if len(pack.Instruments) != 2 || pack.Instruments[0] != "手术剪" {
		t.Fatalf("NewPack 保存了入参切片引用，输入修改被污染: %v", pack.Instruments)
	}
}

// TestPackCopyIsDeepForInstruments 器械包 Copy 必须是深拷贝：
// 修改副本的器械清单，原对象必须保持不变。
func TestPackCopyIsDeepForInstruments(t *testing.T) {
	now := time.Now()
	pack := &InstrumentPack{
		ID:          "pack_001",
		Barcode:     "CP-001",
		Name:        "深拷贝测试包",
		PackType:    TypeDressing,
		Instruments: []string{"镊子", "持针器"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	cp := pack.Copy()
	cp.Instruments[0] = "被改的器械"
	cp.Instruments = append(cp.Instruments, "新增器械")

	if len(pack.Instruments) != 2 || pack.Instruments[0] != "镊子" {
		t.Fatalf("Copy 与源对象共享器械清单: %v", pack.Instruments)
	}
}
