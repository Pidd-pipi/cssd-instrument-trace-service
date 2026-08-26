package service

import (
	"example.com/cssd-instrument-trace-service/config"
	"example.com/cssd-instrument-trace-service/store"
)

// Actor 表示一次业务操作的操作人与来源 IP。
type Actor struct {
	Operator string
	IP       string
}

// Services 聚合全部业务服务，作为依赖注入的根容器。
type Services struct {
	Store          *store.Store
	Rules          config.Rules
	Packs          *PackService
	Sterilizations *SterilizationService
	Issues         *IssueService
	Trace          *TraceService
	Audit          *AuditService
	Stats          *StatsService
}

// New 构建服务容器并装配各服务之间的依赖。
func New(st *store.Store, rules config.Rules) *Services {
	audit := &AuditService{store: st}
	return &Services{
		Store:          st,
		Rules:          rules,
		Packs:          NewPackService(st, audit),
		Sterilizations: NewSterilizationService(st, audit, rules),
		Issues:         NewIssueService(st, audit, rules),
		Trace:          NewTraceService(st),
		Audit:          audit,
		Stats:          NewStatsService(st),
	}
}
