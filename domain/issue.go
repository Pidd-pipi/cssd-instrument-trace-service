package domain

import (
	"fmt"
	"strings"
	"time"
)

// IssueRecord 发放记录：绑定使用科室与手术间，回收时按条码闭环。
type IssueRecord struct {
	ID                   string      `json:"id"`
	PackID               string      `json:"packId"`
	Barcode              string      `json:"barcode"`
	Department           string      `json:"department"`    // 使用科室
	OperatingRoom        string      `json:"operatingRoom"` // 手术间
	Issuer               string      `json:"issuer"`        // 发放人
	IssuedAt             time.Time   `json:"issuedAt"`
	Collector            string      `json:"collector,omitempty"` // 回收人
	CollectedAt          *time.Time  `json:"collectedAt,omitempty"`
	Status               IssueStatus `json:"status"`
	SterilizationBatchID string      `json:"sterilizationBatchId"`
}

// IssueInput 发放登记入参。
type IssueInput struct {
	Department    string `json:"department"`
	OperatingRoom string `json:"operatingRoom"`
	Issuer        string `json:"issuer"`
}

// Validate 校验发放入参。
func (in IssueInput) Validate() error {
	if strings.TrimSpace(in.Department) == "" {
		return fmt.Errorf("%w: 使用科室不能为空", ErrInvalidParam)
	}
	if strings.TrimSpace(in.OperatingRoom) == "" {
		return fmt.Errorf("%w: 手术间不能为空", ErrInvalidParam)
	}
	if strings.TrimSpace(in.Issuer) == "" {
		return fmt.Errorf("%w: 发放人不能为空", ErrInvalidParam)
	}
	return nil
}

// NewIssueRecord 构建一条发放记录。
func NewIssueRecord(pack *InstrumentPack, in IssueInput, batchID string) *IssueRecord {
	return &IssueRecord{
		ID:                   NewID("issue"),
		PackID:               pack.ID,
		Barcode:              pack.Barcode,
		Department:           strings.TrimSpace(in.Department),
		OperatingRoom:        strings.TrimSpace(in.OperatingRoom),
		Issuer:               strings.TrimSpace(in.Issuer),
		IssuedAt:             time.Now(),
		Status:               IssueOpen,
		SterilizationBatchID: batchID,
	}
}

// IsOpen 判断发放记录是否处于「已发放、未回收」的活跃状态（可被回收闭环）。
// 注意：丢失待查（IssueLost）是终态，不再视为「开放」，不会进入未回收列表，
// 也不会被回收接口当成正常归还覆盖；以 Status 为唯一判据，避免 CollectedAt 与
// Status 不一致带来的状态流转错乱。
func (r *IssueRecord) IsOpen() bool {
	return r.Status == IssueOpen
}

// IsUnclosed 判断发放记录是否尚未闭环（活跃开放 或 丢失待查）。
// 丢失包未结案，禁止重复发放，故未闭环包含丢失态。
func (r *IssueRecord) IsUnclosed() bool {
	return r.Status == IssueOpen || r.Status == IssueLost
}

// IsLost 判断发放记录是否已进入「丢失待查」终态。
func (r *IssueRecord) IsLost() bool {
	return r.Status == IssueLost
}

// HoursOutstanding 返回发放至今未回收的小时数。
func (r *IssueRecord) HoursOutstanding(now time.Time) float64 {
	return now.Sub(r.IssuedAt).Hours()
}

// MarkReturned 标记发放记录回收闭环。
// 丢失待查（IssueLost）是终态：器械包尚未找回，不可被当作正常归还回收，
// 否则丢失标记会被静默覆盖、统计把这笔又记成已闭环，丢失清单与开放记录还会
// 同时出现同一条。故对丢失记录调用回收直接返回错误，禁止状态回流。
func (r *IssueRecord) MarkReturned(collector string, now time.Time) error {
	if r.Status == IssueLost {
		return fmt.Errorf("%w: 发放记录 %s 已丢失待查，不可回收闭环", ErrIssueLost, r.ID)
	}
	r.Collector = strings.TrimSpace(collector)
	ts := now
	r.CollectedAt = &ts
	r.Status = IssueReturned
	return nil
}

// MarkLost 标记发放记录为丢失待查（终态）。
// 仅当记录仍处于活跃开放态时才置为丢失，避免对已闭环或已丢失记录重复改写，
// 保证丢失标记幂等、不被重复扫描反复刷写。
func (r *IssueRecord) MarkLost() {
	if r.Status != IssueOpen {
		return
	}
	r.Status = IssueLost
}
