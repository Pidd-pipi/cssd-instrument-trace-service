package store

import (
	"sort"

	"example.com/cssd-instrument-trace-service/domain"
)

// SaveSterilizer 新增或更新灭菌器并持久化。
// 存入仓储的是入参的拷贝，避免外部继续修改入参指针而串进仓储内部。
func (s *Store) SaveSterilizer(st *domain.Sterilizer) error {
	return s.mutate(func() error {
		s.sterilizers[st.ID] = st.Copy()
		return nil
	})
}

// GetSterilizer 按 ID 获取灭菌器。
// 返回仓储内对象的深拷贝，调用方修改不会影响仓储数据。
func (s *Store) GetSterilizer(id string) (*domain.Sterilizer, error) {
	var out *domain.Sterilizer
	s.view(func() {
		if st, ok := s.sterilizers[id]; ok {
			out = st.Copy()
		}
	})
	if out == nil {
		return nil, domain.ErrNotFound
	}
	return out, nil
}

// ListSterilizers 返回全部灭菌器，可用状态优先。
// 每条记录均为仓储内对象的深拷贝，调用方修改不会影响仓储数据。
func (s *Store) ListSterilizers() []*domain.Sterilizer {
	out := make([]*domain.Sterilizer, 0)
	s.view(func() {
		for _, st := range s.sterilizers {
			out = append(out, st.Copy())
		}
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].Status != out[j].Status {
			return out[i].Status == domain.SterilizerAvailable
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}
