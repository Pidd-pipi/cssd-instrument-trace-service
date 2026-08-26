package store

import (
	"sort"

	"example.com/cssd-instrument-trace-service/domain"
)

// SaveCycle 新增环节记录并持久化。
// 存入仓储的是入参的拷贝，避免外部继续修改入参指针而串进仓储内部。
func (s *Store) SaveCycle(c *domain.CycleRecord) error {
	return s.mutate(func() error {
		s.cycles[c.ID] = c.Copy()
		return nil
	})
}

// ListCyclesByPack 返回器械包全部环节记录，按时间正序（追溯链顺序）。
// 每条记录均为仓储内对象的深拷贝，调用方修改不会影响仓储数据。
func (s *Store) ListCyclesByPack(packID string) []*domain.CycleRecord {
	out := make([]*domain.CycleRecord, 0)
	s.view(func() {
		for _, c := range s.cycles {
			if c.PackID == packID {
				out = append(out, c.Copy())
			}
		}
	})
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

// CountCyclesByPack 返回器械包环节记录数量。
func (s *Store) CountCyclesByPack(packID string) int {
	var n int
	s.view(func() {
		for _, c := range s.cycles {
			if c.PackID == packID {
				n++
			}
		}
	})
	return n
}
