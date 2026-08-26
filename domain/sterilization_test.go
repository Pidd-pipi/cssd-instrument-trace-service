package domain

import (
	"testing"
	"time"
)

func TestJudgeParamsPass(t *testing.T) {
	b := NewBatch(CreateBatchInput{
		SterilizerID: "s1", TempC: 134, DurationMin: 6, PressureKPa: 210,
		PackIDs: []string{"p1"},
	}, "灭菌器A", "SB20260825-001")
	result, reasons := b.JudgeParams(SterilizationLimits{MinTempC: 134, MinDurationMin: 4, MinPressureKPa: 205})
	if result != ResultPass {
		t.Errorf("参数达标应判定 pass，实际 %s reasons=%v", result, reasons)
	}
	if len(reasons) != 0 {
		t.Errorf("参数达标不应有失败原因: %v", reasons)
	}
}

func TestJudgeParamsFailOnEachLimit(t *testing.T) {
	b := NewBatch(CreateBatchInput{
		SterilizerID: "s1", TempC: 130, DurationMin: 3, PressureKPa: 200,
		PackIDs: []string{"p1"},
	}, "灭菌器A", "SB20260825-002")
	result, reasons := b.JudgeParams(SterilizationLimits{MinTempC: 134, MinDurationMin: 4, MinPressureKPa: 205})
	if result != ResultFail {
		t.Errorf("任一参数不达标应判定 fail，实际 %s", result)
	}
	if len(reasons) != 3 {
		t.Errorf("应收集 3 条失败原因，实际 %d: %v", len(reasons), reasons)
	}
}

func TestBatchComplete(t *testing.T) {
	b := NewBatch(CreateBatchInput{
		SterilizerID: "s1", TempC: 134, DurationMin: 5, PressureKPa: 208,
		PackIDs: []string{"p1"},
	}, "灭菌器A", "SB20260825-003")
	b.Complete(ResultPass, nil, time.Now())
	if b.Status != BatchCompleted || b.Result != ResultPass {
		t.Errorf("批次完结状态异常: %+v", b)
	}
	if b.CompletedAt == nil {
		t.Error("完结时间未设置")
	}
}
