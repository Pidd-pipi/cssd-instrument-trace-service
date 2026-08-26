package service

import (
	"fmt"
	"time"

	"example.com/cssd-instrument-trace-service/config"
	"example.com/cssd-instrument-trace-service/domain"
	"example.com/cssd-instrument-trace-service/store"
)

// IssueService 发放校验与回收闭环。
type IssueService struct {
	store *store.Store
	audit *AuditService
	rules config.Rules
}

// NewIssueService 构建发放回收服务。
func NewIssueService(st *store.Store, audit *AuditService, rules config.Rules) *IssueService {
	return &IssueService{store: st, audit: audit, rules: rules}
}

// Issue 发放器械包：强制校验「已灭菌 + 未过期 + 灭菌批次参数合格」三项。
func (s *IssueService) Issue(packID string, in domain.IssueInput, actor Actor) (*domain.IssueRecord, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	pack, err := s.store.GetPack(packID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if open := s.store.GetOpenIssueByPack(packID); open != nil {
		return nil, fmt.Errorf("%w: 器械包 %s 已有未回收的发放记录", domain.ErrConflict, pack.Barcode)
	}
	if ok, reason := pack.CanBeIssued(now); !ok {
		return nil, &BlockedError{Kind: domain.ErrIssueBlocked, Reason: reason}
	}
	// 二次核对灭菌批次本身参数合格，防止数据不一致。
	batch, err := s.store.GetBatch(pack.LastBatchID)
	if err != nil || batch.Result != domain.ResultPass {
		return nil, &BlockedError{Kind: domain.ErrIssueBlocked, Reason: "关联灭菌批次参数不合格或不存在，禁止发放"}
	}
	if err := s.store.UpdatePack(packID, func(p *domain.InstrumentPack) error {
		p.Stage = domain.StageIssued
		p.Touch()
		return nil
	}); err != nil {
		return nil, err
	}
	issue := domain.NewIssueRecord(pack, in, batch.ID)
	if err := s.store.SaveIssue(issue); err != nil {
		return nil, err
	}
	rec := domain.NewCycleRecord(pack, domain.StageSterilized, domain.StageIssued, actor.Operator, "",
		fmt.Sprintf("发放至 %s %s", in.Department, in.OperatingRoom), map[string]any{
			"department":    in.Department,
			"operatingRoom": in.OperatingRoom,
			"batchNo":       batch.BatchNo,
		})
	if err := s.store.SaveCycle(rec); err != nil {
		return nil, err
	}
	s.audit.Record(domain.ActionPackIssue, actor, "pack", packID, map[string]any{
		"barcode":       pack.Barcode,
		"department":    in.Department,
		"operatingRoom": in.OperatingRoom,
		"batchNo":       batch.BatchNo,
	})
	return s.store.GetIssue(issue.ID)
}

// Collect 回收器械包：必须处于「使用中」且有未回收发放记录，按条码闭环。
func (s *IssueService) Collect(packID, collector string, actor Actor) (*domain.IssueRecord, error) {
	pack, err := s.store.GetPack(packID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if pack.Stage != domain.StageInUse {
		return nil, &BlockedError{Kind: domain.ErrCollectBlocked,
			Reason: fmt.Sprintf("器械包 %s 当前环节为 %s，仅「使用中」器械包可回收", pack.Barcode, pack.Stage)}
	}
	open := s.store.GetOpenIssueByPack(packID)
	if open == nil {
		return nil, &BlockedError{Kind: domain.ErrCollectBlocked,
			Reason: fmt.Sprintf("器械包 %s 没有未回收的发放记录，无法闭环回收", pack.Barcode)}
	}
	if err := s.store.UpdatePack(packID, func(p *domain.InstrumentPack) error {
		if err := domain.ValidateTransition(p.Stage, domain.StageToCollect); err != nil {
			return err
		}
		p.Stage = domain.StageToCollect
		p.Touch()
		return nil
	}); err != nil {
		return nil, err
	}
	if err := s.store.UpdateIssue(open.ID, func(r *domain.IssueRecord) error {
		r.MarkReturned(collector, now)
		return nil
	}); err != nil {
		return nil, err
	}
	rec := domain.NewCycleRecord(pack, domain.StageInUse, domain.StageToCollect, actor.Operator, "",
		fmt.Sprintf("回收完成，归还 %s %s", open.Department, open.OperatingRoom), map[string]any{
			"department":    open.Department,
			"operatingRoom": open.OperatingRoom,
			"collector":     collector,
		})
	if err := s.store.SaveCycle(rec); err != nil {
		return nil, err
	}
	s.audit.Record(domain.ActionPackCollect, actor, "pack", packID, map[string]any{
		"barcode":   pack.Barcode,
		"collector": collector,
		"issueId":   open.ID,
	})
	return s.store.GetIssue(open.ID)
}

// ListIssues 查询发放记录。
func (s *IssueService) ListIssues(f store.IssueFilter) []*domain.IssueRecord {
	return s.store.ListIssues(f)
}

// LostEntry 丢失待查条目：发放超时未回收的器械包。
type LostEntry struct {
	Issue        *domain.IssueRecord    `json:"issue"`
	Pack         *domain.InstrumentPack `json:"pack"`
	OverdueHours float64                `json:"overdueHours"`
}

// ScanLost 扫描发放超时（默认 24 小时）未回收的记录，标记为「丢失待查」。
func (s *IssueService) ScanLost(now time.Time) []LostEntry {
	cut := now.Add(-s.rules.LostTimeout())
	records := s.store.ListOpenIssuesOlderThan(cut)
	entries := make([]LostEntry, 0, len(records))
	for _, r := range records {
		pack, err := s.store.GetPack(r.PackID)
		if err != nil {
			continue
		}
		if r.Status == domain.IssueOpen {
			_ = s.store.UpdateIssue(r.ID, func(rec *domain.IssueRecord) error {
				rec.MarkLost()
				return nil
			})
			r.Status = domain.IssueLost
		}
		entries = append(entries, LostEntry{
			Issue:        r,
			Pack:         pack,
			OverdueHours: r.HoursOutstanding(now),
		})
	}
	return entries
}

// LostList 返回丢失待查清单。
func (s *IssueService) LostList(now time.Time) []LostEntry {
	entries := s.ScanLost(now)
	// 补充仍为「丢失」状态但可能刚被标记的记录（避免依赖 ScanLost 副作用）。
	seen := make(map[string]bool, len(entries))
	for _, e := range entries {
		seen[e.Issue.ID] = true
	}
	for _, r := range s.store.ListIssues(store.IssueFilter{Status: string(domain.IssueLost), Limit: 100}) {
		if seen[r.ID] {
			continue
		}
		pack, err := s.store.GetPack(r.PackID)
		if err != nil {
			continue
		}
		entries = append(entries, LostEntry{
			Issue:        r,
			Pack:         pack,
			OverdueHours: r.HoursOutstanding(now),
		})
	}
	return entries
}
