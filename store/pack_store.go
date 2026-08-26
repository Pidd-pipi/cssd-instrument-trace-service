package store

import (
	"sort"
	"strings"

	"example.com/cssd-instrument-trace-service/domain"
)

// PackFilter 器械包查询过滤条件。
type PackFilter struct {
	Stage    string
	PackType string
	Keyword  string // 按条码/名称模糊匹配
	Limit    int
	Offset   int
}

// SavePack 新增或更新器械包并持久化。
func (s *Store) SavePack(p *domain.InstrumentPack) error {
	return s.mutate(func() error {
		old, existed := s.packs[p.ID]
		if existed && old.Barcode != p.Barcode {
			delete(s.packBarcodeIndex, old.Barcode)
		}
		s.packs[p.ID] = p
		s.packBarcodeIndex[p.Barcode] = p.ID
		return nil
	})
}

// GetPack 按 ID 获取器械包。
func (s *Store) GetPack(id string) (*domain.InstrumentPack, error) {
	var out *domain.InstrumentPack
	s.view(func() {
		if p, ok := s.packs[id]; ok {
			out = p.Copy()
		}
	})
	if out == nil {
		return nil, domain.ErrNotFound
	}
	return out, nil
}

// GetPackByBarcode 按唯一条码获取器械包。
func (s *Store) GetPackByBarcode(barcode string) (*domain.InstrumentPack, error) {
	var out *domain.InstrumentPack
	s.view(func() {
		if id, ok := s.packBarcodeIndex[barcode]; ok {
			if p, ok := s.packs[id]; ok {
				out = p.Copy()
			}
		}
	})
	if out == nil {
		return nil, nil
	}
	return out, nil
}

// UpdatePack 在写锁内按 ID 更新器械包；fn 接收副本，返回错误则回滚。
func (s *Store) UpdatePack(id string, fn func(p *domain.InstrumentPack) error) error {
	return s.mutate(func() error {
		p, ok := s.packs[id]
		if !ok {
			return domain.ErrNotFound
		}
		cp := p.Copy()
		if err := fn(cp); err != nil {
			return err
		}
		s.packs[id] = cp
		s.packBarcodeIndex[cp.Barcode] = cp.ID
		return nil
	})
}

// ListPacks 按过滤条件查询器械包，按创建时间倒序。
func (s *Store) ListPacks(f PackFilter) []*domain.InstrumentPack {
	out := make([]*domain.InstrumentPack, 0)
	s.view(func() {
		for _, p := range s.packs {
			if f.Stage != "" && string(p.Stage) != f.Stage {
				continue
			}
			if f.PackType != "" && string(p.PackType) != f.PackType {
				continue
			}
			if f.Keyword != "" && !strings.Contains(strings.ToLower(p.Barcode), strings.ToLower(f.Keyword)) &&
				!strings.Contains(strings.ToLower(p.Name), strings.ToLower(f.Keyword)) {
				continue
			}
			out = append(out, p.Copy())
		}
	})
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
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

// CountPacksByStage 统计各环节在途器械包数量，供工作台总览使用。
func (s *Store) CountPacksByStage() map[domain.PackStage]int {
	counts := make(map[domain.PackStage]int)
	s.view(func() {
		for _, p := range s.packs {
			counts[p.Stage]++
		}
	})
	return counts
}

// CountPacks 返回器械包总数。
func (s *Store) CountPacks() int {
	var n int
	s.view(func() { n = len(s.packs) })
	return n
}
