package httpapi

import (
	"net/http"
	"testing"
)

// TestRegisterNewBarcodeSucceeds 全新条码登记必须成功，不能被误判为重复条码。
func TestRegisterNewBarcodeSucceeds(t *testing.T) {
	srv := newTestServer(t)
	var body struct {
		Code int `json:"code"`
	}
	resp := doJSON(t, http.MethodPost, srv.URL+"/api/packs",
		`{"barcode":"NEW-001","name":"全新器械包","packType":"surgical","operator":"测试员"}`, &body)
	if resp.StatusCode != http.StatusOK || body.Code != 0 {
		t.Fatalf("全新条码登记应成功，实际 %d body=%+v", resp.StatusCode, body)
	}
}

// TestTraceUnknownBarcodeReturns404 追溯不存在的条码必须返回 404，不能 panic 成 500。
func TestTraceUnknownBarcodeReturns404(t *testing.T) {
	srv := newTestServer(t)
	var body struct {
		Code int `json:"code"`
	}
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/trace?barcode=NO-SUCH-BARCODE", "", &body)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("追溯不存在条码应返回 404，实际 %d body=%+v", resp.StatusCode, body)
	}
}

// TestTraceMissingBarcodeParam400 缺少条码参数必须返回 400，不能 panic 成 500。
func TestTraceMissingBarcodeParam400(t *testing.T) {
	srv := newTestServer(t)
	var body struct {
		Code int `json:"code"`
	}
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/trace", "", &body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("缺少条码参数应返回 400，实际 %d body=%+v", resp.StatusCode, body)
	}
}
