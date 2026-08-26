package domain

import (
	"errors"
	"testing"
)

func TestStageTransitions(t *testing.T) {
	tests := []struct {
		from PackStage
		to   PackStage
		want error
	}{
		{StageToCollect, StageCollected, nil},
		{StageCollected, StageWashing, nil},
		{StageWashing, StageWashed, nil},
		{StageWashed, StageSterilizing, nil},
		{StageSterilizing, StageSterilized, nil},
		{StageSterilized, StageIssued, nil},
		{StageIssued, StageInUse, nil},
		{StageInUse, StageToCollect, nil},
		{StageExpired, StageWashing, nil},
		// 跳步必须拒绝
		{StageToCollect, StageWashing, ErrInvalidTransition},
		{StageCollected, StageSterilized, ErrInvalidTransition},
		{StageWashed, StageIssued, ErrInvalidTransition},
		{StageSterilized, StageInUse, ErrInvalidTransition},
		{StageInUse, StageCollected, ErrInvalidTransition},
		{StageExpired, StageIssued, ErrInvalidTransition},
	}
	for _, tt := range tests {
		err := ValidateTransition(tt.from, tt.to)
		if !errors.Is(err, tt.want) {
			t.Errorf("ValidateTransition(%s -> %s) = %v, want %v", tt.from, tt.to, err, tt.want)
		}
	}
}

func TestIsManualCycle(t *testing.T) {
	manual := []struct{ from, to PackStage }{
		{StageToCollect, StageCollected},
		{StageCollected, StageWashing},
		{StageWashing, StageWashed},
		{StageIssued, StageInUse},
		{StageExpired, StageWashing},
	}
	for _, m := range manual {
		if !IsManualCycle(m.from, m.to) {
			t.Errorf("IsManualCycle(%s -> %s) = false, want true", m.from, m.to)
		}
	}
	blocked := []struct{ from, to PackStage }{
		{StageWashed, StageSterilizing},
		{StageSterilizing, StageSterilized},
		{StageSterilized, StageIssued},
		{StageInUse, StageToCollect},
	}
	for _, m := range blocked {
		if IsManualCycle(m.from, m.to) {
			t.Errorf("IsManualCycle(%s -> %s) = true, want false（应走专用接口）", m.from, m.to)
		}
	}
}

func TestIsValidStage(t *testing.T) {
	if !IsValidStage(StageToCollect) || !IsValidStage(StageExpired) {
		t.Error("合法环节被误判为非法")
	}
	if IsValidStage("unknown_stage") {
		t.Error("非法环节被误判为合法")
	}
}

func TestNextStage(t *testing.T) {
	if next, ok := NextStage(StageSterilizing); !ok || next != StageSterilized {
		t.Errorf("NextStage(sterilizing) = %s, %v; want sterilized, true", next, ok)
	}
}
