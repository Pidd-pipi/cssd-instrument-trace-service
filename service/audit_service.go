package service

import (
	"example.com/cssd-instrument-trace-service/domain"
	"example.com/cssd-instrument-trace-service/store"
)

// AuditService 负责操作审计日志的写入与查询。
type AuditService struct {
	store *store.Store
}

// Record 记录一条审计日志；审计写入失败不阻断主业务流程。
func (s *AuditService) Record(action string, actor Actor, targetType, targetID string, detail map[string]any) {
	log := domain.NewAuditLog(action, actor.Operator, targetType, targetID, actor.IP, detail)
	_ = s.store.SaveAudit(log)
}

// RecordOperator 使用默认操作人记录审计（定时任务等场景）。
func (s *AuditService) RecordOperator(action, operator, targetType, targetID string, detail map[string]any) {
	s.Record(action, Actor{Operator: operator}, targetType, targetID, detail)
}

// List 返回最近 limit 条审计日志。保留旧签名便于测试与既有调用。
func (s *AuditService) List(limit int) []*domain.AuditLog {
	return s.store.ListAudits(limit)
}

// ListPage 返回审计日志分页结果。
func (s *AuditService) ListPage(limit, offset int) []*domain.AuditLog {
	return s.store.ListAuditsPage(limit, offset)
}
