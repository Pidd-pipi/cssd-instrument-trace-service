package service

import (
	"time"

	"example.com/cssd-instrument-trace-service/domain"
	"example.com/cssd-instrument-trace-service/store"
)

// Dashboard 工作台总览数据。
type Dashboard struct {
	TotalPacks          int                      `json:"totalPacks"`
	ByStage             map[string]int           `json:"byStage"`
	SterilizedAvailable int                      `json:"sterilizedAvailable"`
	FailedIntercepted   int                      `json:"failedIntercepted"`
	LostCount           int                      `json:"lostCount"`
	TodayRegistered     int                      `json:"todayRegistered"`
	TodayIssued         int                      `json:"todayIssued"`
	TodaySterilized     int                      `json:"todaySterilized"`
	RecentPacks         []*domain.InstrumentPack `json:"recentPacks"`
	RecentAudits        []*domain.AuditLog       `json:"recentAudits"`
}

// StatsService 汇总工作台统计指标。
type StatsService struct {
	store *store.Store
}

// NewStatsService 构建统计服务。
func NewStatsService(st *store.Store) *StatsService {
	return &StatsService{store: st}
}

// Dashboard 计算工作台总览指标。
func (s *StatsService) Dashboard() Dashboard {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	byStage := s.store.CountPacksByStage()
	d := Dashboard{
		TotalPacks:          s.store.CountPacks(),
		ByStage:             make(map[string]int, len(byStage)),
		SterilizedAvailable: byStage[domain.StageSterilized],
		LostCount:           len(s.store.ListIssues(store.IssueFilter{Status: string(domain.IssueLost)})),
		RecentPacks:         s.store.ListPacks(store.PackFilter{Limit: 8}),
		RecentAudits:        s.store.ListAudits(8),
	}
	for stage, count := range byStage {
		d.ByStage[string(stage)] = count
	}
	// 灭菌失败拦截：最近一次灭菌判定失败且退回「已清洗」的器械包。
	for _, p := range s.store.ListPacks(store.PackFilter{Stage: string(domain.StageWashed)}) {
		if p.LastBatchResult == domain.ResultFail {
			d.FailedIntercepted++
		}
	}
	// 今日统计。
	for _, p := range s.store.ListPacks(store.PackFilter{Limit: 10000}) {
		if p.CreatedAt.After(todayStart) {
			d.TodayRegistered++
		}
	}
	for _, r := range s.store.ListIssues(store.IssueFilter{Limit: 10000}) {
		if r.IssuedAt.After(todayStart) {
			d.TodayIssued++
		}
	}
	for _, b := range s.store.ListBatches(10000) {
		if b.CompletedAt != nil && b.CompletedAt.After(todayStart) {
			d.TodaySterilized++
		}
	}
	return d
}
