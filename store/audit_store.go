package store

import (
	"sort"

	"example.com/cssd-instrument-trace-service/domain"
)

// SaveAudit 新增审计日志并持久化。
func (s *Store) SaveAudit(a *domain.AuditLog) error {
	return s.mutate(func() error {
		s.audits[a.ID] = a
		return nil
	})
}

// ListAudits 返回审计日志，按时间倒序，支持 limit 截断。保留旧签名便于测试与既有调用。
func (s *Store) ListAudits(limit int) []*domain.AuditLog {
	return s.ListAuditsPage(limit, 0)
}

// ListAuditsPage 返回审计日志分页结果，按时间倒序。
func (s *Store) ListAuditsPage(limit, offset int) []*domain.AuditLog {
	out := make([]*domain.AuditLog, 0)
	s.view(func() {
		for _, a := range s.audits {
			cp := *a
			out = append(out, &cp)
		}
	})
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if offset > len(out) {
		offset = len(out)
	}
	out = out[offset:]
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// CountAudits 返回审计日志总数。
func (s *Store) CountAudits() int {
	var n int
	s.view(func() { n = len(s.audits) })
	return n
}
