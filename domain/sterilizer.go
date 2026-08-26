package domain

import (
	"fmt"
	"strings"
	"time"
)

// Sterilizer 灭菌器设备：批次创建时校验可用状态。
type Sterilizer struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Model     string           `json:"model"`
	Status    SterilizerStatus `json:"status"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

// CreateSterilizerInput 灭菌器登记入参。
type CreateSterilizerInput struct {
	Name   string           `json:"name"`
	Model  string           `json:"model"`
	Status SterilizerStatus `json:"status"`
}

// Validate 校验灭菌器登记入参。
func (in CreateSterilizerInput) Validate() error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("%w: 灭菌器名称不能为空", ErrInvalidParam)
	}
	switch in.Status {
	case "", SterilizerAvailable:
		in.Status = SterilizerAvailable
	case SterilizerMaintenance:
	default:
		return fmt.Errorf("%w: 未知灭菌器状态 %q", ErrInvalidParam, in.Status)
	}
	return nil
}

// IsAvailable 判断灭菌器是否可创建批次。
func (s *Sterilizer) IsAvailable() bool {
	return s.Status == SterilizerAvailable
}

// NewSterilizer 构建灭菌器。
func NewSterilizer(in CreateSterilizerInput) *Sterilizer {
	now := time.Now()
	status := in.Status
	if status == "" {
		status = SterilizerAvailable
	}
	return &Sterilizer{
		ID:        NewID("ster"),
		Name:      strings.TrimSpace(in.Name),
		Model:     strings.TrimSpace(in.Model),
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
