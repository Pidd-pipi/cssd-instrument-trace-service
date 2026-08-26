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
	if open := s.store.GetUnclosedIssueByPack(packID); open != nil {
		// 已有未闭环记录：活跃开放（重复发放）或丢失待查（包未找回，禁止再发）。
		if open.Status == domain.IssueLost {
			return nil, fmt.Errorf("%w: 器械包 %s 上一笔发放已丢失待查，未结案前禁止重复发放", domain.ErrConflict, pack.Barcode)
		}
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

// Collect 回收器械包：必须处于「使用中」且有活跃开放的发放记录，按条码闭环。
// 丢失待查(IssueLost)是终态：器械包尚未找回，不能当作正常归还回收，否则丢失
// 标记会被覆盖、统计把这笔又记成已闭环、丢失与开放列表还会重复出现同一条。
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
		// 区分两种异常：仅有丢失记录（不可回收，应走「找回/结案」流程），
		// 还是根本没有发放记录。
		if unclosed := s.store.GetUnclosedIssueByPack(packID); unclosed != nil && unclosed.Status == domain.IssueLost {
			return nil, &BlockedError{Kind: domain.ErrCollectBlocked,
				Reason: fmt.Sprintf("器械包 %s 的发放记录已丢失待查，请先找回并结案后再闭环，不可直接回收", pack.Barcode)}
		}
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
		// MarkReturned 对丢失记录会拒绝回流（终态保护），双保险防止覆盖。
		return r.MarkReturned(collector, now)
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

// ScanLost 扫描发放超时（默认 24 小时）仍未回收的活跃开放记录，标记为「丢失待查」。
// ListOpenIssuesOlderThan 仅返回 IssueOpen 记录，故已丢失记录不会再次进入扫描，
// 丢失标记幂等：同一条记录不会被反复刷写，丢失清单也不会重复出现。
func (s *IssueService) ScanLost(now time.Time) []LostEntry {
	cut := now.Add(-s.rules.LostTimeout())
	records := s.store.ListOpenIssuesOlderThan(cut)
	entries := make([]LostEntry, 0, len(records))
	for _, r := range records {
		pack, err := s.store.GetPack(r.PackID)
		if err != nil {
			continue
		}
		// MarkLost 仅对 IssueOpen 生效（终态保护），此处记录必为开放态。
		_ = s.store.UpdateIssue(r.ID, func(rec *domain.IssueRecord) error {
			rec.MarkLost()
			return nil
		})
		r.Status = domain.IssueLost
		entries = append(entries, LostEntry{
			Issue:        r,
			Pack:         pack,
			OverdueHours: r.HoursOutstanding(now),
		})
	}
	return entries
}

// LostList 返回丢失待查清单。
// 先扫描将新超时记录置为丢失（副作用与扫描任务一致），再补齐历史丢失记录，
// 以 ID 去重，确保同一笔记录不会同时出现两次。
func (s *IssueService) LostList(now time.Time) []LostEntry {
	entries := s.ScanLost(now)
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
