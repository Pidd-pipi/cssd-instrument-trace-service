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

// IsOpen 判断发放记录是否尚未回收闭环。
func (r *IssueRecord) IsOpen() bool {
	return r.CollectedAt == nil
}

// HoursOutstanding 返回发放至今未回收的小时数。
func (r *IssueRecord) HoursOutstanding(now time.Time) float64 {
	return now.Sub(r.IssuedAt).Hours()
}

// MarkReturned 标记发放记录回收闭环。
func (r *IssueRecord) MarkReturned(collector string, now time.Time) {
	r.Collector = strings.TrimSpace(collector)
	ts := now
	r.CollectedAt = &ts
	r.Status = IssueReturned
}

// MarkLost 标记发放记录为丢失待查。
func (r *IssueRecord) MarkLost() {
	r.Status = IssueLost
}

// Copy 返回发放记录的深拷贝，指针字段与原对象隔离，
// 避免调用方在副本上修改时写穿仓储内对象。
func (r *IssueRecord) Copy() *IssueRecord {
	cp := *r
	if r.CollectedAt != nil {
		t := *r.CollectedAt
		cp.CollectedAt = &t
	}
	return &cp
}
