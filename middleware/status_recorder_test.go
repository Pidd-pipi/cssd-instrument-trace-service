package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestStatusRecorderPreservesFirstStatus 响应提交后再次写头不得覆盖首个状态码。
func TestStatusRecorderPreservesFirstStatus(t *testing.T) {
	rec := NewStatusRecorder(httptest.NewRecorder())
	rec.WriteHeader(http.StatusOK)
	rec.WriteHeader(http.StatusInternalServerError)
	if got := rec.Status(); got != http.StatusOK {
		t.Fatalf("首个状态码 200 被覆盖为 %d", got)
	}
}

// TestStatusRecorderCommitted 写头或写体后应标记响应已提交。
func TestStatusRecorderCommitted(t *testing.T) {
	rec := NewStatusRecorder(httptest.NewRecorder())
	if rec.Committed() {
		t.Fatal("未写任何内容时不应视为已提交")
	}
	rec.WriteHeader(http.StatusOK)
	if !rec.Committed() {
		t.Fatal("写头后应视为已提交")
	}
	rec2 := NewStatusRecorder(httptest.NewRecorder())
	_, _ = rec2.Write([]byte("body"))
	if !rec2.Committed() {
		t.Fatal("写体后应视为已提交")
	}
}
