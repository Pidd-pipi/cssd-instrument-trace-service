package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// NewID 生成带业务前缀的唯一 ID，形如 pack_1756ab2c3d4e_8f3a。
// 前缀用于区分实体类型，便于追溯与排查。
func NewID(prefix string) string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand 失败时回退到时间戳，仍可保证进程内唯一。
		return fmt.Sprintf("%s_%x", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%x_%s", prefix, time.Now().UnixMilli(), hex.EncodeToString(buf))
}

// NewBatchNo 生成灭菌批次号，形如 SB20260825-003。
func NewBatchNo(now time.Time, seq int) string {
	return fmt.Sprintf("SB%s-%03d", now.Format("20060102"), seq)
}
