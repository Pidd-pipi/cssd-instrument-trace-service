package service

import (
	"context"
	"log/slog"
	"time"

	"example.com/cssd-instrument-trace-service/domain"
	"example.com/cssd-instrument-trace-service/store"
)

// ExpirySweeper 过期扫描定时任务：周期性将过期器械包置为「已过期」。
type ExpirySweeper struct {
	svc      *Services
	interval time.Duration
}

// NewExpirySweeper 构建过期扫描器。
func NewExpirySweeper(svc *Services, intervalMin int) *ExpirySweeper {
	interval := time.Duration(intervalMin) * time.Minute
	if interval <= 0 {
		interval = time.Hour
	}
	return &ExpirySweeper{svc: svc, interval: interval}
}

// Run 启动定时扫描，直到 ctx 被取消。
func (e *ExpirySweeper) Run(ctx context.Context) {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()
	slog.Info("过期扫描任务已启动", "interval", e.interval.String())
	for {
		select {
		case <-ctx.Done():
			slog.Info("过期扫描任务停止")
			return
		case now := <-ticker.C:
			expired, err := e.ExpireSterilizedPacks(now)
			if err != nil {
				slog.Error("过期扫描失败", "error", err)
				continue
			}
			lost := e.svc.Issues.ScanLost(now)
			if expired > 0 || len(lost) > 0 {
				slog.Info("过期扫描完成", "expired", expired, "lost", len(lost))
			}
		}
	}
}

// ExpireSterilizedPacks 扫描「已灭菌」且超过有效期的器械包并置为「已过期」。
// 返回本次置为过期的器械包数量。
func (e *ExpirySweeper) ExpireSterilizedPacks(now time.Time) (int, error) {
	packs := e.svc.Store.ListPacks(store.PackFilter{Stage: string(domain.StageSterilized)})
	count := 0
	for _, pack := range packs {
		if !pack.IsExpired(now) {
			continue
		}
		if err := e.svc.Store.UpdatePack(pack.ID, func(p *domain.InstrumentPack) error {
			if p.Stage != domain.StageSterilized {
				return nil
			}
			p.MarkExpired(now)
			return nil
		}); err != nil {
			return count, err
		}
		rec := domain.NewCycleRecord(pack, domain.StageSterilized, domain.StageExpired, "system", "",
			"无菌有效期到期，自动置为过期", map[string]any{
				"expiryAt": pack.ExpiryAt.Format(time.RFC3339),
			})
		if err := e.svc.Store.SaveCycle(rec); err != nil {
			return count, err
		}
		e.svc.Audit.RecordOperator(domain.ActionExpirySweep, "system", "pack", pack.ID, map[string]any{
			"barcode":  pack.Barcode,
			"expiryAt": pack.ExpiryAt.Format(time.RFC3339),
		})
		count++
	}
	return count, nil
}
