package store

import (
	"sort"

	"example.com/cssd-instrument-trace-service/domain"
)

// SaveSterilizer 新增或更新灭菌器并持久化。
func (s *Store) SaveSterilizer(st *domain.Sterilizer) error {
	return s.mutate(func() error {
		s.sterilizers[st.ID] = st
		return nil
	})
}

// GetSterilizer 按 ID 获取灭菌器。
func (s *Store) GetSterilizer(id string) (*domain.Sterilizer, error) {
	var out *domain.Sterilizer
	s.view(func() {
		if st, ok := s.sterilizers[id]; ok {
			cp := *st
			out = &cp
		}
	})
	if out == nil {
		return nil, domain.ErrNotFound
	}
	return out, nil
}

// ListSterilizers 返回全部灭菌器，可用状态优先。
func (s *Store) ListSterilizers() []*domain.Sterilizer {
	out := make([]*domain.Sterilizer, 0)
	s.view(func() {
		for _, st := range s.sterilizers {
			cp := *st
			out = append(out, &cp)
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
