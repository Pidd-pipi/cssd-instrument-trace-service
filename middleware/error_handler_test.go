package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorHandlerRecoversPanic(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
	handler := ErrorHandler(panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("panic 应返回 500，实际 %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"code":500`) {
		t.Errorf("响应应为统一错误格式: %s", body)
	}
}

func TestErrorHandlerPassesThrough(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := ErrorHandler(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("正常请求不应被拦截，实际 %d", rec.Code)
	}
}
