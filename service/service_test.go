package service

import (
	"path/filepath"
	"testing"

	"example.com/cssd-instrument-trace-service/config"
	"example.com/cssd-instrument-trace-service/domain"
	"example.com/cssd-instrument-trace-service/store"
)

// newTestServices 构建带临时仓储的测试服务容器。
func newTestServices(t *testing.T) *Services {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "store.json"))
	if err != nil {
		t.Fatalf("创建测试仓储失败: %v", err)
	}
	return New(st, config.DefaultRules())
}

// testActor 测试用操作人。
var testActor = Actor{Operator: "测试员", IP: "127.0.0.1"}

// registerTestPack 便捷登记器械包，返回 ID 与条码。
func registerTestPack(t *testing.T, svc *Services, barcode, name string, packType domain.PackType) (string, string) {
	t.Helper()
	pack, err := svc.Packs.Register(domain.RegisterPackInput{
		Barcode:  barcode,
		Name:     name,
		PackType: packType,
	}, testActor)
	if err != nil {
		t.Fatalf("登记器械包失败: %v", err)
	}
	return pack.ID, pack.Barcode
}

// forwardToWashed 将器械包推进到「已清洗」，为装载灭菌做准备。
func forwardToWashed(t *testing.T, svc *Services, packID string) {
	t.Helper()
	for _, stage := range []domain.PackStage{domain.StageCollected, domain.StageWashing, domain.StageWashed} {
		if _, err := svc.Packs.Cycle(packID, stage, "dev_001", "", testActor); err != nil {
			t.Fatalf("推进到 %s 失败: %v", stage, err)
		}
	}
}
