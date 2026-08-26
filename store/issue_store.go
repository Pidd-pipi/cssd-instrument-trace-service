package store

import (
	"sort"
	"time"

	"example.com/cssd-instrument-trace-service/domain"
)

// IssueFilter 发放记录查询过滤条件。
type IssueFilter struct {
	Status string
	PackID string
	Limit  int
	Offset int
}

// SaveIssue 新增或更新发放记录并持久化。
func (s *Store) SaveIssue(r *domain.IssueRecord) error {
	return s.mutate(func() error {
		s.issues[r.ID] = r
		return nil
	})
}

// GetIssue 按 ID 获取发放记录。
func (s *Store) GetIssue(id string) (*domain.IssueRecord, error) {
	var out *domain.IssueRecord
	s.view(func() {
		if r, ok := s.issues[id]; ok {
			out = r.Copy()
		}
	})
	if out == nil {
		return nil, domain.ErrNotFound
	}
	return out, nil
}

// GetOpenIssueByPack 返回器械包当前未回收的发放记录。
func (s *Store) GetOpenIssueByPack(packID string) *domain.IssueRecord {
	var out *domain.IssueRecord
	s.view(func() {
		for _, r := range s.issues {
			if r.PackID == packID && r.IsOpen() {
				out = r.Copy()
				return
			}
		}
	})
	return out
}

// ListIssues 按过滤条件查询发放记录，按发放时间倒序。
func (s *Store) ListIssues(f IssueFilter) []*domain.IssueRecord {
	out := make([]*domain.IssueRecord, 0)
	s.view(func() {
		for _, r := range s.issues {
			if f.Status != "" && string(r.Status) != f.Status {
				continue
			}
			if f.PackID != "" && r.PackID != f.PackID {
				continue
			}
			out = append(out, r.Copy())
		}
	})
	sort.Slice(out, func(i, j int) bool {
		return out[i].IssuedAt.After(out[j].IssuedAt)
	})
	if f.Offset > len(out) {
		f.Offset = len(out)
	}
	out = out[f.Offset:]
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out
}

// UpdateIssue 在写锁内按 ID 更新发放记录；fn 接收副本，返回错误则回滚。
func (s *Store) UpdateIssue(id string, fn func(r *domain.IssueRecord) error) error {
	return s.mutate(func() error {
		r, ok := s.issues[id]
		if !ok {
			return domain.ErrNotFound
		}
		cp := r.Copy()
		if err := fn(cp); err != nil {
			return err
		}
		s.issues[id] = cp
		return nil
	})
}

// ListOpenIssuesOlderThan 返回发放时间早于 cut 且尚未回收的发放记录。
func (s *Store) ListOpenIssuesOlderThan(cut time.Time) []*domain.IssueRecord {
	out := make([]*domain.IssueRecord, 0)
	s.view(func() {
		for _, r := range s.issues {
			if r.IsOpen() && r.IssuedAt.Before(cut) {
				out = append(out, r.Copy())
			}
		}
	})
	return out
}

// CountIssues 返回发放记录总数。
func (s *Store) CountIssues() int {
	var n int
	s.view(func() { n = len(s.issues) })
	return n
}
