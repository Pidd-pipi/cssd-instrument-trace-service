package service

import (
	"testing"
	"time"

	"example.com/cssd-instrument-trace-service/domain"
)

func TestExpireSterilizedPacks(t *testing.T) {
	svc := newTestServices(t)
	id := sterilizeAndGetReadyPack(t, svc, "PK-EXP")

	pack, _ := svc.Packs.Get(id)
	if pack.ExpiryAt == nil {
		t.Fatal("器械包应有有效期")
	}
	// 将有效期回拨到过去，模拟已过期。
	past := time.Now().Add(-time.Hour)
	if err := svc.Store.UpdatePack(id, func(p *domain.InstrumentPack) error {
		p.ExpiryAt = &past
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	sweeper := NewExpirySweeper(svc, 60)
	count, err := sweeper.ExpireSterilizedPacks(time.Now())
	if err != nil {
		t.Fatalf("过期扫描失败: %v", err)
	}
	if count != 1 {
		t.Errorf("应扫描出 1 个过期包，实际 %d", count)
	}
	pack, _ = svc.Packs.Get(id)
	if pack.Stage != domain.StageExpired {
		t.Errorf("过期包环节 = %s, want expired", pack.Stage)
	}
	// 二次扫描应为 0（不再重复置为过期）。
	count, _ = sweeper.ExpireSterilizedPacks(time.Now())
	if count != 0 {
		t.Errorf("二次扫描应为 0，实际 %d", count)
	}
}

func TestExpiredPackReentryToWashing(t *testing.T) {
	svc := newTestServices(t)
	id := sterilizeAndGetReadyPack(t, svc, "PK-REENTRY")
	past := time.Now().Add(-time.Hour)
	if err := svc.Store.UpdatePack(id, func(p *domain.InstrumentPack) error {
		p.ExpiryAt = &past
		p.MarkExpired(time.Now())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// 过期包 → 清洗中 重新处理。
	pack, err := svc.Packs.Cycle(id, domain.StageWashing, "dev_x", "过期重新处理", testActor)
	if err != nil {
		t.Fatalf("过期包重新清洗应允许: %v", err)
	}
	if pack.Stage != domain.StageWashing {
		t.Errorf("重新清洗后环节 = %s, want washing", pack.Stage)
	}
}
