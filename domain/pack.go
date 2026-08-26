package domain

import (
	"fmt"
	"strings"
	"time"
)

// InstrumentPack 器械包实体：器械包登记、循环流转、有效期管理的核心对象。
type InstrumentPack struct {
	ID               string          `json:"id"`
	Barcode          string          `json:"barcode"` // 唯一条码
	Name             string          `json:"name"`    // 包名称
	PackType         PackType        `json:"packType"`
	Instruments      []string        `json:"instruments"` // 内含器械清单
	Stage            PackStage       `json:"stage"`
	SterilizedAt     *time.Time      `json:"sterilizedAt,omitempty"`
	ExpiryAt         *time.Time      `json:"expiryAt,omitempty"` // 无菌有效期截止
	LastBatchID      string          `json:"lastBatchId,omitempty"`
	LastBatchResult  SterilizeResult `json:"lastBatchResult,omitempty"`
	LastFailedReason string          `json:"lastFailedReason,omitempty"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

// RegisterPackInput 器械包登记入参。
type RegisterPackInput struct {
	Barcode     string   `json:"barcode"`
	Name        string   `json:"name"`
	PackType    PackType `json:"packType"`
	Instruments []string `json:"instruments"`
}

// Validate 校验登记入参的合法性。
func (in RegisterPackInput) Validate() error {
	if strings.TrimSpace(in.Barcode) == "" {
		return fmt.Errorf("%w: 条码不能为空", ErrInvalidParam)
	}
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("%w: 包名称不能为空", ErrInvalidParam)
	}
	if !IsValidPackType(in.PackType) {
		return fmt.Errorf("%w: 未知包类型 %q", ErrInvalidParam, in.PackType)
	}
	return nil
}

// NewPack 依据登记入参构建处于「待回收」初始环节的器械包。
func NewPack(in RegisterPackInput) *InstrumentPack {
	now := time.Now()
	return &InstrumentPack{
		ID:          NewID("pack"),
		Barcode:     strings.TrimSpace(in.Barcode),
		Name:        strings.TrimSpace(in.Name),
		PackType:    in.PackType,
		Instruments: in.Instruments,
		Stage:       StageToCollect,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// IsExpired 判断器械包在当前时刻是否已过无菌有效期。
func (p *InstrumentPack) IsExpired(now time.Time) bool {
	return p.ExpiryAt != nil && !p.ExpiryAt.After(now)
}

// CanBeIssued 校验发放三要素：已灭菌 + 未过期 + 灭菌批次参数合格。
func (p *InstrumentPack) CanBeIssued(now time.Time) (bool, string) {
	if p.Stage != StageSterilized {
		return false, "器械包当前环节不是「已灭菌」，禁止发放"
	}
	if p.IsExpired(now) {
		return false, "器械包已过有效期，禁止发放"
	}
	if p.LastBatchResult != ResultPass {
		return false, "灭菌批次参数不合格，器械包已被拦截，禁止发放"
	}
	return true, ""
}

// MarkSterilized 标记器械包灭菌成功并计算有效期。
func (p *InstrumentPack) MarkSterilized(batchID string, expiryAt time.Time, now time.Time) {
	p.Stage = StageSterilized
	p.LastBatchID = batchID
	p.LastBatchResult = ResultPass
	p.LastFailedReason = ""
	ts := now
	p.SterilizedAt = &ts
	exp := expiryAt
	p.ExpiryAt = &exp
	p.UpdatedAt = now
}

// MarkSterilizationFailed 标记灭菌失败，器械包退回「已清洗」等待重新灭菌。
func (p *InstrumentPack) MarkSterilizationFailed(batchID, reason string, now time.Time) {
	p.Stage = StageWashed
	p.LastBatchID = batchID
	p.LastBatchResult = ResultFail
	p.LastFailedReason = reason
	p.SterilizedAt = nil
	p.ExpiryAt = nil
	p.UpdatedAt = now
}

// MarkExpired 由过期扫描任务调用，将器械包置为「已过期」。
func (p *InstrumentPack) MarkExpired(now time.Time) {
	p.Stage = StageExpired
	p.UpdatedAt = now
}

// Copy 返回器械包的深拷贝，避免调用方意外修改仓储内对象。
func (p *InstrumentPack) Copy() *InstrumentPack {
	cp := *p
	if p.SterilizedAt != nil {
		t := *p.SterilizedAt
		cp.SterilizedAt = &t
	}
	if p.ExpiryAt != nil {
		t := *p.ExpiryAt
		cp.ExpiryAt = &t
	}
	cp.Instruments = append([]string(nil), p.Instruments...)
	return &cp
}

// IsValidPackType 判断包类型是否合法。
func IsValidPackType(t PackType) bool {
	switch t {
	case TypeSurgical, TypeDressing, TypeInstrument, TypeImplant:
		return true
	default:
		return false
	}
}

// Touch 更新器械包更新时间戳。
func (p *InstrumentPack) Touch() {
	p.UpdatedAt = time.Now()
}
