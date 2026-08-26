package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"example.com/cssd-instrument-trace-service/domain"
)

// snapshot 是仓储全量数据的可序列化视图。
type snapshot struct {
	Packs       map[string]*domain.InstrumentPack     `json:"packs"`
	Batches     map[string]*domain.SterilizationBatch `json:"batches"`
	Cycles      map[string]*domain.CycleRecord        `json:"cycles"`
	Issues      map[string]*domain.IssueRecord        `json:"issues"`
	Sterilizers map[string]*domain.Sterilizer         `json:"sterilizers"`
	Audits      map[string]*domain.AuditLog           `json:"audits"`
}

// persistLocked 在已持有写锁的前提下序列化并原子写入磁盘。
func (s *Store) persistLocked() error {
	return s.persist()
}

// persist 在调用方已持有写锁的前提下持久化当前快照。
func (s *Store) persist() error {
	data, err := json.MarshalIndent(s.snapshot(), "", "  ")
	if err != nil {
		return fmt.Errorf("序列化仓储数据失败: %w", err)
	}
	return atomicWriteFile(s.file, data)
}

func (s *Store) snapshot() snapshot {
	return snapshot{
		Packs:       s.packs,
		Batches:     s.batches,
		Cycles:      s.cycles,
		Issues:      s.issues,
		Sterilizers: s.sterilizers,
		Audits:      s.audits,
	}
}

// load 从磁盘加载仓储快照；文件不存在时视为首次启动。
// 数据文件损坏时备份为 .bak 并降级为空库启动，避免服务不可用。
func (s *Store) load() error {
	data, err := os.ReadFile(s.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("读取持久化文件失败: %w", err)
	}
	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		s.backupCorruptFile(err)
		return nil
	}
	s.applySnapshot(snap)
	return nil
}

func (s *Store) applySnapshot(snap snapshot) {
	if snap.Packs != nil {
		s.packs = snap.Packs
	}
	if snap.Batches != nil {
		s.batches = snap.Batches
	}
	if snap.Cycles != nil {
		s.cycles = snap.Cycles
	}
	if snap.Issues != nil {
		s.issues = snap.Issues
	}
	if snap.Sterilizers != nil {
		s.sterilizers = snap.Sterilizers
	}
	if snap.Audits != nil {
		s.audits = snap.Audits
	}
	// 重建条码索引，保证重启后仍能按条码查询。
	for id, p := range s.packs {
		s.packBarcodeIndex[p.Barcode] = id
	}
}

// backupCorruptFile 将损坏数据文件重命名为 .bak，并记录告警。
func (s *Store) backupCorruptFile(cause error) {
	backup := s.file + ".bak"
	if _, err := os.Stat(backup); err == nil {
		backup = s.file + ".bak." + time.Now().Format("20060102150405.000000000")
	}
	if err := os.Rename(s.file, backup); err != nil {
		slog.Error("持久化文件损坏且备份失败", "file", s.file, "backup", backup, "error", err)
		return
	}
	slog.Warn("持久化文件损坏，已备份并降级为空库启动", "file", s.file, "backup", backup, "error", cause)
}

// atomicWriteFile 使用「临时文件 → fsync → rename → fsync 目录」的原子替换策略，
// 避免进程中断或写一半导致数据文件损坏。
func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".store-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("设置临时文件权限失败: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("同步临时文件失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时文件失败: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("重命名持久化文件失败: %w", err)
	}
	syncDir(dir)
	return nil
}

// syncDir 尽力同步目录元数据，确保 rename 后目录项落盘。
func syncDir(dir string) {
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	defer d.Close()
	_ = d.Sync()
}
