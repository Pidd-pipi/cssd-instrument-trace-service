package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRecoveredAfterPartialWriteKeepsStatus 响应已部分提交后再 panic，
// 不得追加 500 JSON、不得覆盖状态码，日志状态码必须与客户端一致。
func TestRecoveredAfterPartialWriteKeepsStatus(t *testing.T) {
	handler := RequestLog(ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial-body"))
		panic("boom-after-write")
	})))

	req := httptest.NewRequest(http.MethodGet, "/partial", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("已提交 200 的响应状态码被改写为 %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.HasPrefix(body, "partial-body") {
		t.Fatalf("响应体被破坏: %q", body)
	}
	if strings.Contains(body, `"code":500`) {
		t.Fatalf("已提交响应后仍追加了 500 JSON: %q", body)
	}
	if body != "partial-body" {
		t.Fatalf("响应体被追加了额外内容: %q", body)
	}
}

// TestRecoveredUncommittedWrites500 未提交任何响应时 panic，仍须返回统一 500。
func TestRecoveredUncommittedWrites500(t *testing.T) {
	handler := RequestLog(ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom-before-write")
	})))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("未提交响应 panic 应返回 500，实际 %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":500`) {
		t.Fatalf("应返回统一 500 JSON: %q", rec.Body.String())
	}
}
