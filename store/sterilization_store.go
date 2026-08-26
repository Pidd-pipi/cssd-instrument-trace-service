package store

import (
	"sort"

	"example.com/cssd-instrument-trace-service/domain"
)

// SaveBatch 新增或更新灭菌批次并持久化。
func (s *Store) SaveBatch(b *domain.SterilizationBatch) error {
	return s.mutate(func() error {
		s.batches[b.ID] = b
		return nil
	})
}

// GetBatch 按 ID 获取灭菌批次。
func (s *Store) GetBatch(id string) (*domain.SterilizationBatch, error) {
	var out *domain.SterilizationBatch
	s.view(func() {
		if b, ok := s.batches[id]; ok {
			out = b
		}
	})
	if out == nil {
		return nil, domain.ErrNotFound
	}
	return out, nil
}

// UpdateBatch 在写锁内按 ID 更新批次。
func (s *Store) UpdateBatch(id string, fn func(b *domain.SterilizationBatch) error) error {
	return s.mutate(func() error {
		b, ok := s.batches[id]
		if !ok {
			return domain.ErrNotFound
		}
		if err := fn(b); err != nil {
			return err
		}
		return nil
	})
}

// ListBatches 返回全部灭菌批次，按创建时间倒序。保留旧签名便于测试与既有调用。
func (s *Store) ListBatches(limit int) []*domain.SterilizationBatch {
	return s.ListBatchesPage(limit, 0)
}

// ListBatchesPage 返回灭菌批次分页结果，按创建时间倒序。
func (s *Store) ListBatchesPage(limit, offset int) []*domain.SterilizationBatch {
	out := make([]*domain.SterilizationBatch, 0)
	s.view(func() {
		for _, b := range s.batches {
			out = append(out, b)
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

// CountBatches 返回灭菌批次总数。
func (s *Store) CountBatches() int {
	var n int
	s.view(func() { n = len(s.batches) })
	return n
}
