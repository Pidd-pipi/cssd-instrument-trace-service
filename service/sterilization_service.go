package service

import (
	"fmt"
	"strings"
	"time"

	"example.com/cssd-instrument-trace-service/config"
	"example.com/cssd-instrument-trace-service/domain"
	"example.com/cssd-instrument-trace-service/store"
)

// SterilizationService 灭菌批次创建、参数判定与器械包状态联动。
type SterilizationService struct {
	store *store.Store
	audit *AuditService
	rules config.Rules
}

// NewSterilizationService 构建灭菌服务。
func NewSterilizationService(st *store.Store, audit *AuditService, rules config.Rules) *SterilizationService {
	return &SterilizationService{store: st, audit: audit, rules: rules}
}

// CreateBatch 创建灭菌批次：校验灭菌器可用、器械包均为「已清洗」，
// 批次创建后器械包推进到「灭菌中」。
func (s *SterilizationService) CreateBatch(in domain.CreateBatchInput, actor Actor) (*domain.SterilizationBatch, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	sterilizer, err := s.store.GetSterilizer(in.SterilizerID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidParam, err)
	}
	if !sterilizer.IsAvailable() {
		return nil, fmt.Errorf("%w: %s 处于维护中", domain.ErrSterilizerUnavailable, sterilizer.Name)
	}
	for _, pid := range in.PackIDs {
		pack, err := s.store.GetPack(pid)
		if err != nil {
			return nil, fmt.Errorf("%w: 器械包 %s 不存在", domain.ErrInvalidParam, pid)
		}
		if pack.Stage != domain.StageWashed {
			return nil, fmt.Errorf("%w: 器械包 %s 当前环节为 %s，仅「已清洗」器械包可装载灭菌", domain.ErrInvalidTransition, pack.Barcode, pack.Stage)
		}
	}
	batchNo := domain.NewBatchNo(time.Now(), s.store.CountBatches()+1)
	batch := domain.NewBatch(in, sterilizer.Name, batchNo)
	if err := s.store.SaveBatch(batch); err != nil {
		return nil, err
	}
	params := map[string]any{
		"tempC":       in.TempC,
		"durationMin": in.DurationMin,
		"pressureKPa": in.PressureKPa,
	}
	for _, pid := range in.PackIDs {
		pack, _ := s.store.GetPack(pid)
		from := pack.Stage
		if err := s.store.UpdatePack(pid, func(p *domain.InstrumentPack) error {
			if err := domain.ValidateTransition(p.Stage, domain.StageSterilizing); err != nil {
				return err
			}
			p.Stage = domain.StageSterilizing
			p.Touch()
			return nil
		}); err != nil {
			return nil, err
		}
		rec := domain.NewCycleRecord(pack, from, domain.StageSterilizing, actor.Operator, sterilizer.ID,
			fmt.Sprintf("装载灭菌批次 %s", batch.BatchNo), params)
		if err := s.store.SaveCycle(rec); err != nil {
			return nil, err
		}
	}
	s.audit.Record(domain.ActionSterilizationCreate, actor, "batch", batch.ID, map[string]any{
		"batchNo":     batch.BatchNo,
		"sterilizer":  sterilizer.Name,
		"tempC":       in.TempC,
		"durationMin": in.DurationMin,
		"pressureKPa": in.PressureKPa,
		"packCount":   len(in.PackIDs),
	})
	return s.store.GetBatch(batch.ID)
}

// CompleteBatch 完成灭菌：依据参数下限判定合格/失败并联动更新批次内器械包。
//   - 合格：器械包推进到「已灭菌」并按包类型计算有效期；
//   - 失败：批次内器械包全部拦截，退回「已清洗」等待重新灭菌。
func (s *SterilizationService) CompleteBatch(batchID string, actor Actor) (*domain.SterilizationBatch, error) {
	batch, err := s.store.GetBatch(batchID)
	if err != nil {
		return nil, err
	}
	if batch.Status == domain.BatchCompleted {
		return nil, fmt.Errorf("%w: 批次 %s 已完成参数判定", domain.ErrConflict, batch.BatchNo)
	}
	result, reasons := batch.JudgeParams(s.rules.SterilizationLimits)
	now := time.Now()
	for _, pid := range batch.PackIDs {
		pack, err := s.store.GetPack(pid)
		if err != nil {
			continue
		}
		from := pack.Stage
		var params map[string]any
		switch result {
		case domain.ResultPass:
			expiryDays := s.rules.ExpiryDaysOf(pack.PackType)
			expiryAt := now.Add(time.Duration(expiryDays) * 24 * time.Hour)
			if err := s.store.UpdatePack(pid, func(p *domain.InstrumentPack) error {
				if p.Stage != domain.StageSterilizing {
					return fmt.Errorf("%w: 器械包 %s 不在灭菌中，无法完成灭菌", domain.ErrConflict, p.Barcode)
				}
				p.MarkSterilized(batchID, expiryAt, now)
				return nil
			}); err != nil {
				return nil, err
			}
			params = map[string]any{
				"tempC":       batch.TempC,
				"durationMin": batch.DurationMin,
				"pressureKPa": batch.PressureKPa,
				"result":      domain.ResultPass,
				"batchNo":     batch.BatchNo,
				"expiryAt":    expiryAt.Format(time.RFC3339),
			}
		case domain.ResultFail:
			reason := strings.Join(reasons, ";")
			if err := s.store.UpdatePack(pid, func(p *domain.InstrumentPack) error {
				if p.Stage != domain.StageSterilizing {
					return fmt.Errorf("%w: 器械包 %s 不在灭菌中，无法完成灭菌", domain.ErrConflict, p.Barcode)
				}
				p.MarkSterilizationFailed(batchID, reason, now)
				return nil
			}); err != nil {
				return nil, err
			}
			params = map[string]any{
				"tempC":       batch.TempC,
				"durationMin": batch.DurationMin,
				"pressureKPa": batch.PressureKPa,
				"result":      domain.ResultFail,
				"batchNo":     batch.BatchNo,
				"reason":      reason,
			}
		}
		to := domain.StageSterilized
		if result == domain.ResultFail {
			to = domain.StageWashed
		}
		rec := domain.NewCycleRecord(pack, from, to, actor.Operator, batch.SterilizerID,
			fmt.Sprintf("灭菌批次 %s 判定完成", batch.BatchNo), params)
		if err := s.store.SaveCycle(rec); err != nil {
			return nil, err
		}
	}
	// 批次状态翻转必须在写锁内原子完成：状态校验与写入同处一个临界区，
	// 避免两台客户端并发完成时都读到 pending 而重复联动器械包。
	if err := s.store.UpdateBatch(batch.ID, func(b *domain.SterilizationBatch) error {
		if b.Status == domain.BatchCompleted {
			return fmt.Errorf("%w: 批次 %s 已完成参数判定", domain.ErrConflict, b.BatchNo)
		}
		b.Complete(result, reasons, now)
		return nil
	}); err != nil {
		return nil, err
	}
	s.audit.Record(domain.ActionSterilizationComplete, actor, "batch", batch.ID, map[string]any{
		"batchNo":   batch.BatchNo,
		"result":    result,
		"reasons":   reasons,
		"packCount": len(batch.PackIDs),
	})
	return s.store.GetBatch(batch.ID)
}

// ListBatches 返回灭菌批次列表。保留旧签名便于测试与既有调用。
func (s *SterilizationService) ListBatches(limit int) []*domain.SterilizationBatch {
	return s.store.ListBatches(limit)
}

// ListBatchesPage 返回灭菌批次分页结果。
func (s *SterilizationService) ListBatchesPage(limit, offset int) []*domain.SterilizationBatch {
	return s.store.ListBatchesPage(limit, offset)
}

// GetBatch 按 ID 获取灭菌批次。
func (s *SterilizationService) GetBatch(id string) (*domain.SterilizationBatch, error) {
	return s.store.GetBatch(id)
}

// ListSterilizers 返回全部灭菌器。
func (s *SterilizationService) ListSterilizers() []*domain.Sterilizer {
	return s.store.ListSterilizers()
}

// CreateSterilizer 登记灭菌器。
func (s *SterilizationService) CreateSterilizer(in domain.CreateSterilizerInput, actor Actor) (*domain.Sterilizer, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	st := domain.NewSterilizer(in)
	if err := s.store.SaveSterilizer(st); err != nil {
		return nil, err
	}
	s.audit.Record(domain.ActionSterilizerCreate, actor, "sterilizer", st.ID, map[string]any{
		"name":   st.Name,
		"model":  st.Model,
		"status": st.Status,
	})
	return s.store.GetSterilizer(st.ID)
}
