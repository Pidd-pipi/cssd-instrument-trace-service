package service

import (
	"fmt"

	"example.com/cssd-instrument-trace-service/domain"
	"example.com/cssd-instrument-trace-service/store"
)

// PackService 器械包登记与环节流转。
type PackService struct {
	store *store.Store
	audit *AuditService
}

// NewPackService 构建器械包服务。
func NewPackService(st *store.Store, audit *AuditService) *PackService {
	return &PackService{store: st, audit: audit}
}

// Register 登记器械包，初始环节为「待回收」。
func (s *PackService) Register(in domain.RegisterPackInput, actor Actor) (*domain.InstrumentPack, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if _, err := s.store.GetPackByBarcode(in.Barcode); err == nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrDuplicateBarcode, in.Barcode)
	}
	pack := domain.NewPack(in)
	if err := s.store.SavePack(pack); err != nil {
		return nil, err
	}
	// 登记即写入首条环节记录（从空环节进入「待回收」），保证追溯链完整。
	rec := domain.NewCycleRecord(pack, "", domain.StageToCollect, actor.Operator, "", "器械包登记完成", map[string]any{
		"packType": pack.PackType,
	})
	if err := s.store.SaveCycle(rec); err != nil {
		return nil, err
	}
	s.audit.Record(domain.ActionPackRegister, actor, "pack", pack.ID, map[string]any{
		"barcode":  pack.Barcode,
		"name":     pack.Name,
		"packType": pack.PackType,
		"stage":    pack.Stage,
	})
	return s.store.GetPack(pack.ID)
}

// Cycle 通用环节流转：仅允许状态机中的手动迁移，跳步直接拒绝。
func (s *PackService) Cycle(packID string, target domain.PackStage, deviceID, note string, actor Actor) (*domain.InstrumentPack, error) {
	if !domain.IsValidStage(target) {
		return nil, fmt.Errorf("%w: 未知环节 %q", domain.ErrInvalidParam, target)
	}
	pack, err := s.store.GetPack(packID)
	if err != nil {
		return nil, err
	}
	from := pack.Stage
	if from == domain.StageSterilized && target == domain.StageIssued {
		// 手动发放放行
	} else if from == domain.StageInUse && target == domain.StageToCollect {
		// 手动回收放行
	} else if !domain.IsManualCycle(from, target) {
		return nil, fmt.Errorf("%w: 环节 %s 不能通过通用流转推进到 %s", domain.ErrInvalidTransition, from, target)
	}
	if err := domain.ValidateTransition(from, target); err != nil {
		return nil, err
	}
	if err := s.store.UpdatePack(packID, func(p *domain.InstrumentPack) error {
		p.Stage = target
		p.Touch()
		return nil
	}); err != nil {
		return nil, err
	}
	params := map[string]any{}
	if deviceID != "" {
		params["deviceId"] = deviceID
	}
	if note != "" {
		params["note"] = note
	}
	rec := domain.NewCycleRecord(pack, target, target, actor.Operator, deviceID, note, params)
	if err := s.store.SaveCycle(rec); err != nil {
		return nil, err
	}
	s.audit.Record(domain.ActionPackCycle, actor, "pack", packID, map[string]any{
		"from":     from,
		"to":       target,
		"deviceId": deviceID,
		"note":     note,
	})
	return s.store.GetPack(packID)
}

// List 按过滤条件查询器械包。
func (s *PackService) List(filter store.PackFilter) []*domain.InstrumentPack {
	return s.store.ListPacks(filter)
}

// Get 按 ID 获取器械包。
func (s *PackService) Get(id string) (*domain.InstrumentPack, error) {
	return s.store.GetPack(id)
}

// GetByBarcode 按条码获取器械包。
func (s *PackService) GetByBarcode(barcode string) (*domain.InstrumentPack, error) {
	return s.store.GetPackByBarcode(barcode)
}
