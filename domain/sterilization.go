package domain

import (
	"fmt"
	"strings"
	"time"
)

// SterilizationLimits 灭菌参数合格判定的下限规则（由 config 层提供）。
type SterilizationLimits struct {
	MinTempC       float64
	MinDurationMin int
	MinPressureKPa float64
}

// SterilizationBatch 灭菌批次：记录灭菌器、关键参数、判定结果与批次内器械包。
type SterilizationBatch struct {
	ID             string          `json:"id"`
	BatchNo        string          `json:"batchNo"`
	SterilizerID   string          `json:"sterilizerId"`
	SterilizerName string          `json:"sterilizerName"`
	Operator       string          `json:"operator"`
	TempC          float64         `json:"tempC"`
	DurationMin    int             `json:"durationMin"`
	PressureKPa    float64         `json:"pressureKPa"`
	Status         BatchStatus     `json:"status"`
	Result         SterilizeResult `json:"result"`
	FailReasons    []string        `json:"failReasons,omitempty"`
	PackIDs        []string        `json:"packIds"`
	CreatedAt      time.Time       `json:"createdAt"`
	CompletedAt    *time.Time      `json:"completedAt,omitempty"`
}

// CreateBatchInput 创建灭菌批次入参。
type CreateBatchInput struct {
	SterilizerID string   `json:"sterilizerId"`
	Operator     string   `json:"operator"`
	TempC        float64  `json:"tempC"`
	DurationMin  int      `json:"durationMin"`
	PressureKPa  float64  `json:"pressureKPa"`
	PackIDs      []string `json:"packIds"`
}

// Validate 校验创建批次入参。
func (in CreateBatchInput) Validate() error {
	if strings.TrimSpace(in.SterilizerID) == "" {
		return fmt.Errorf("%w: 必须指定灭菌器", ErrInvalidParam)
	}
	if in.TempC <= 0 {
		return fmt.Errorf("%w: 灭菌温度必须大于 0", ErrInvalidParam)
	}
	if in.DurationMin <= 0 {
		return fmt.Errorf("%w: 灭菌时长必须大于 0", ErrInvalidParam)
	}
	if in.PressureKPa <= 0 {
		return fmt.Errorf("%w: 灭菌压力必须大于 0", ErrInvalidParam)
	}
	if len(in.PackIDs) == 0 {
		return fmt.Errorf("%w: 批次必须包含至少一个器械包", ErrInvalidParam)
	}
	return nil
}

// NewBatch 依据入参构建待判定的灭菌批次。
// 入参切片拷贝后归入批次，避免外部对入参切片的后续修改串进批次内部。
func NewBatch(in CreateBatchInput, sterilizerName string, batchNo string) *SterilizationBatch {
	now := time.Now()
	return &SterilizationBatch{
		ID:             NewID("batch"),
		BatchNo:        batchNo,
		SterilizerID:   in.SterilizerID,
		SterilizerName: sterilizerName,
		Operator:       strings.TrimSpace(in.Operator),
		TempC:          in.TempC,
		DurationMin:    in.DurationMin,
		PressureKPa:    in.PressureKPa,
		Status:         BatchPending,
		Result:         "",
		PackIDs:        append([]string(nil), in.PackIDs...),
		CreatedAt:      now,
	}
}

// Copy 返回灭菌批次的深拷贝，避免调用方意外修改仓储内对象。
func (b *SterilizationBatch) Copy() *SterilizationBatch {
	if b == nil {
		return nil
	}
	cp := *b
	cp.PackIDs = append([]string(nil), b.PackIDs...)
	cp.FailReasons = append([]string(nil), b.FailReasons...)
	if b.CompletedAt != nil {
		ts := *b.CompletedAt
		cp.CompletedAt = &ts
	}
	return &cp
}

// JudgeParams 依据规则判定灭菌参数是否合格，返回判定结果与不合格原因。
func (b *SterilizationBatch) JudgeParams(limits SterilizationLimits) (SterilizeResult, []string) {
	var reasons []string
	if b.TempC < limits.MinTempC {
		reasons = append(reasons, fmt.Sprintf("灭菌温度 %.1f℃ 低于下限 %.1f℃", b.TempC, limits.MinTempC))
	}
	if b.DurationMin < limits.MinDurationMin {
		reasons = append(reasons, fmt.Sprintf("灭菌时长 %d 分钟短于下限 %d 分钟", b.DurationMin, limits.MinDurationMin))
	}
	if b.PressureKPa < limits.MinPressureKPa {
		reasons = append(reasons, fmt.Sprintf("灭菌压力 %.1fkPa 低于下限 %.1fkPa", b.PressureKPa, limits.MinPressureKPa))
	}
	if len(reasons) > 0 {
		return ResultFail, reasons
	}
	return ResultPass, nil
}

// Complete 完成批次判定，写入结果与完结时间。
func (b *SterilizationBatch) Complete(result SterilizeResult, reasons []string, now time.Time) {
	b.Status = BatchCompleted
	b.Result = result
	b.FailReasons = append([]string(nil), reasons...)
	ts := now
	b.CompletedAt = &ts
}
