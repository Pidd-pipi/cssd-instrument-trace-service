package domain

import (
	"testing"
	"time"
)

func TestRegisterPackValidation(t *testing.T) {
	cases := []RegisterPackInput{
		{Barcode: "", Name: "手术包", PackType: TypeSurgical},
		{Barcode: "PK001", Name: "", PackType: TypeSurgical},
		{Barcode: "PK001", Name: "手术包", PackType: "bogus"},
	}
	for i, in := range cases {
		if err := in.Validate(); err == nil {
			t.Errorf("case %d: 期望校验失败，实际通过", i)
		}
	}
}

func TestNewPackInitialStage(t *testing.T) {
	p := NewPack(RegisterPackInput{Barcode: "PK001", Name: "手术包", PackType: TypeSurgical})
	if p.Stage != StageToCollect {
		t.Errorf("新登记器械包初始环节 = %s, want to_collect", p.Stage)
	}
	if p.ID == "" || p.Barcode != "PK001" {
		t.Errorf("ID/条码未正确初始化: %+v", p)
	}
}

func TestCanBeIssued(t *testing.T) {
	now := time.Now()
	future := now.Add(48 * time.Hour)
	passed := now.Add(-time.Hour)

	p := NewPack(RegisterPackInput{Barcode: "PK002", Name: "敷料包", PackType: TypeDressing})
	p.Stage = StageSterilized
	p.LastBatchResult = ResultPass
	p.ExpiryAt = &future

	if ok, reason := p.CanBeIssued(now); !ok {
		t.Errorf("已灭菌未过期合格包应可发放，实际拒绝: %s", reason)
	}

	// 未灭菌
	p.Stage = StageWashed
	if ok, _ := p.CanBeIssued(now); ok {
		t.Error("未灭菌包不应可发放")
	}
	p.Stage = StageSterilized

	// 已过期
	expired := passed
	p.ExpiryAt = &expired
	if ok, _ := p.CanBeIssued(now); ok {
		t.Error("已过期包不应可发放")
	}
	p.ExpiryAt = &future

	// 批次不合格
	p.LastBatchResult = ResultFail
	if ok, _ := p.CanBeIssued(now); ok {
		t.Error("灭菌批次不合格包不应可发放")
	}
}

func TestMarkSterilizedSetsExpiry(t *testing.T) {
	now := time.Now()
	p := NewPack(RegisterPackInput{Barcode: "PK003", Name: "植入物包", PackType: TypeImplant})
	p.MarkSterilized("batch_1", now.Add(30*24*time.Hour), now)
	if p.Stage != StageSterilized || p.LastBatchResult != ResultPass {
		t.Errorf("灭菌成功状态未正确写入: %+v", p)
	}
	if p.ExpiryAt == nil || !p.ExpiryAt.After(now) {
		t.Error("有效期未正确设置")
	}
}

func TestMarkSterilizationFailed(t *testing.T) {
	now := time.Now()
	p := NewPack(RegisterPackInput{Barcode: "PK004", Name: "器械包", PackType: TypeInstrument})
	p.MarkSterilizationFailed("batch_2", "灭菌温度不达标", now)
	if p.Stage != StageWashed {
		t.Errorf("灭菌失败后应退回已清洗，实际 %s", p.Stage)
	}
	if p.LastBatchResult != ResultFail || p.ExpiryAt != nil {
		t.Errorf("失败标记或有效期清理异常: %+v", p)
	}
}
