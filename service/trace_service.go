package service

import (
	"example.com/cssd-instrument-trace-service/domain"
	"example.com/cssd-instrument-trace-service/store"
)

// TraceService 追溯聚合：按条码/器械包拉取完整循环，按批次查询器械包去向。
type TraceService struct {
	store *store.Store
}

// NewTraceService 构建追溯服务。
func NewTraceService(st *store.Store) *TraceService {
	return &TraceService{store: st}
}

// PackTraceView 器械包完整循环追溯视图。
type PackTraceView struct {
	Pack      *domain.InstrumentPack     `json:"pack"`
	Cycles    []*domain.CycleRecord      `json:"cycles"`
	Issues    []*domain.IssueRecord      `json:"issues"`
	LastBatch *domain.SterilizationBatch `json:"lastBatch,omitempty"`
}

// PackTrace 按器械包 ID 拉取完整循环记录。
func (s *TraceService) PackTrace(packID string) (*PackTraceView, error) {
	pack, err := s.store.GetPack(packID)
	if err != nil {
		return nil, err
	}
	view := &PackTraceView{
		Pack:   pack,
		Cycles: s.store.ListCyclesByPack(packID),
		Issues: s.store.ListIssues(store.IssueFilter{PackID: packID}),
	}
	if pack.LastBatchID != "" {
		if batch, err := s.store.GetBatch(pack.LastBatchID); err == nil {
			view.LastBatch = batch
		}
	}
	return view, nil
}

// TraceByBarcode 按条码定位器械包并拉取完整循环记录。
func (s *TraceService) TraceByBarcode(barcode string) (*PackTraceView, error) {
	pack, err := s.store.GetPackByBarcode(barcode)
	if err != nil {
		return nil, err
	}
	return s.PackTrace(pack.ID)
}

// BatchPackView 批次内器械包去向视图。
type BatchPackView struct {
	Pack        *domain.InstrumentPack `json:"pack"`
	LatestIssue *domain.IssueRecord    `json:"latestIssue,omitempty"`
}

// BatchPacksView 灭菌批次器械包去向聚合视图。
type BatchPacksView struct {
	Batch *domain.SterilizationBatch `json:"batch"`
	Packs []BatchPackView            `json:"packs"`
}

// BatchPacks 查询灭菌批次内全部器械包及其去向。
func (s *TraceService) BatchPacks(batchID string) (*BatchPacksView, error) {
	batch, err := s.store.GetBatch(batchID)
	if err != nil {
		return nil, err
	}
	view := &BatchPacksView{Batch: batch, Packs: make([]BatchPackView, 0, len(batch.PackIDs))}
	for _, pid := range batch.PackIDs {
		pack, err := s.store.GetPack(pid)
		if err != nil {
			continue
		}
		item := BatchPackView{Pack: pack}
		// 最新去向：取最近一条发放记录（含已回收闭环），比仅查未回收记录信息更完整。
		if issues := s.store.ListIssues(store.IssueFilter{PackID: pid, Limit: 1}); len(issues) > 0 {
			item.LatestIssue = issues[0]
		}
		view.Packs = append(view.Packs, item)
	}
	return view, nil
}
