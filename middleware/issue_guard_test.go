package middleware

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"example.com/cssd-instrument-trace-service/config"
	"example.com/cssd-instrument-trace-service/domain"
	"example.com/cssd-instrument-trace-service/service"
	"example.com/cssd-instrument-trace-service/store"
)

func newGuardTestServices(t *testing.T) *service.Services {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "store.json"))
	if err != nil {
		t.Fatal(err)
	}
	return service.New(st, config.DefaultRules())
}

func TestIssueGuardBlocksNonSterilized(t *testing.T) {
	svc := newGuardTestServices(t)
	pack, err := svc.Packs.Register(domain.RegisterPackInput{
		Barcode: "PK-G-1", Name: "未灭菌包", PackType: domain.TypeSurgical,
	}, service.Actor{Operator: "测试员"})
	if err != nil {
		t.Fatal(err)
	}

	guard := IssueGuard(svc)
	handler := guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/packs/"+pack.ID+"/issue", nil)
	req.SetPathValue("id", pack.ID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("守卫应返回 422，实际 %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestIssueGuardPassesSterilized(t *testing.T) {
	svc := newGuardTestServices(t)
	pack, err := svc.Packs.Register(domain.RegisterPackInput{
		Barcode: "PK-G-2", Name: "已灭菌包", PackType: domain.TypeSurgical,
	}, service.Actor{Operator: "测试员"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Store.UpdatePack(pack.ID, func(p *domain.InstrumentPack) error {
		p.Stage = domain.StageSterilized
		p.LastBatchResult = domain.ResultPass
		future := time.Now().Add(48 * time.Hour)
		p.ExpiryAt = &future
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	guard := IssueGuard(svc)
	called := false
	handler := guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/packs/"+pack.ID+"/issue", nil)
	req.SetPathValue("id", pack.ID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called || rec.Code != http.StatusOK {
		t.Fatalf("合格器械包应放行，called=%v status=%d", called, rec.Code)
	}
}
