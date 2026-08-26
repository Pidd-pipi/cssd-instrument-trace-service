// Package store 提供内存仓储 + JSON 文件持久化。
package store

import (
	"sync"

	"example.com/cssd-instrument-trace-service/domain"
)

// Store 内存仓储：以 map 保存全部实体，每次变更后原子写入 JSON 文件。
// 各实体操作方法按文件拆分（pack_store.go 等），保持仓储层职责清晰。
type Store struct {
	mu sync.RWMutex

	file string

	packs            map[string]*domain.InstrumentPack
	packBarcodeIndex map[string]string // barcode -> packID
	batches          map[string]*domain.SterilizationBatch
	cycles           map[string]*domain.CycleRecord
	issues           map[string]*domain.IssueRecord
	sterilizers      map[string]*domain.Sterilizer
	audits           map[string]*domain.AuditLog
}

// New 创建仓储；若持久化文件存在则加载，否则写入种子数据（默认灭菌器）。
func New(file string) (*Store, error) {
	s := &Store{
		file:             file,
		packs:            make(map[string]*domain.InstrumentPack),
		packBarcodeIndex: make(map[string]string),
		batches:          make(map[string]*domain.SterilizationBatch),
		cycles:           make(map[string]*domain.CycleRecord),
		issues:           make(map[string]*domain.IssueRecord),
		sterilizers:      make(map[string]*domain.Sterilizer),
		audits:           make(map[string]*domain.AuditLog),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	if len(s.sterilizers) == 0 {
		for _, st := range seedSterilizers() {
			s.sterilizers[st.ID] = st
		}
		if err := s.persist(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// mutate 执行写操作并持久化；fn 在写锁内执行，返回错误时中止持久化。
func (s *Store) mutate(fn func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(); err != nil {
		return err
	}
	return s.persistLocked()
}

// view 在只读锁内执行查询。
func (s *Store) view(fn func()) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn()
}

// File 返回持久化文件路径。
func (s *Store) File() string { return s.file }
