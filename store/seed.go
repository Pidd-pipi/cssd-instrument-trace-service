package store

import (
	"time"

	"example.com/cssd-instrument-trace-service/domain"
)

// seedSterilizers 返回首次启动时预置的灭菌器设备。
func seedSterilizers() []*domain.Sterilizer {
	now := time.Now()
	return []*domain.Sterilizer{
		{
			ID:        "ster_001",
			Name:      "高温高压灭菌器 A",
			Model:     "YXQ-LS-100SII",
			Status:    domain.SterilizerAvailable,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "ster_002",
			Name:      "高温高压灭菌器 B",
			Model:     "YXQ-LS-75SII",
			Status:    domain.SterilizerAvailable,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "ster_003",
			Name:      "低温等离子灭菌器",
			Model:     "PS-100X",
			Status:    domain.SterilizerMaintenance,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}
