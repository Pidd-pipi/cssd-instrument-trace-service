package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDSetsHeaderAndContext(t *testing.T) {
	var gotID string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get(RequestIDHeader) == "" {
		t.Fatal("响应缺少 X-Request-ID")
	}
	if gotID == "" || gotID != rec.Header().Get(RequestIDHeader) {
		t.Fatalf("上下文请求 ID = %q, 响应头 = %q", gotID, rec.Header().Get(RequestIDHeader))
	}
}

func TestSecurityHeadersSetForAPI(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/packs", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("缺少 nosniff")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("缺少 DENY")
	}
	if rec.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatal("缺少 Referrer-Policy")
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("API 应设置 no-store")
	}
}
