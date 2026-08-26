package httpapi

import (
	"net/http"
	"strings"

	"example.com/cssd-instrument-trace-service/service"
)

// TraceHandler 追溯查询相关接口。
type TraceHandler struct {
	svc *service.Services
}

// NewTraceHandler 构建追溯 handler。
func NewTraceHandler(svc *service.Services) *TraceHandler {
	return &TraceHandler{svc: svc}
}

// PackTrace 器械包循环追溯：GET /api/packs/{id}/trace。
func (h *TraceHandler) PackTrace(w http.ResponseWriter, r *http.Request) {
	id, ok := requirePathID(w, r)
	if !ok {
		return
	}
	view, err := h.svc.Trace.PackTrace(id)
	if err != nil {
		WriteErr(w, err)
		return
	}
	OK(w, view)
}

// TraceByBarcode 按条码追溯：GET /api/trace?barcode=。
func (h *TraceHandler) TraceByBarcode(w http.ResponseWriter, r *http.Request) {
	barcode := strings.TrimSpace(r.URL.Query().Get("barcode"))
	if barcode == "" {
		Fail(w, http.StatusBadRequest, http.StatusBadRequest, "缺少条码参数 barcode")
		return
	}
	view, err := h.svc.Trace.TraceByBarcode(barcode)
	if err != nil {
		WriteErr(w, err)
		return
	}
	OK(w, view)
}

// BatchPacks 批次器械包去向：GET /api/sterilizations/{id}/packs。
func (h *TraceHandler) BatchPacks(w http.ResponseWriter, r *http.Request) {
	id, ok := requirePathID(w, r)
	if !ok {
		return
	}
	view, err := h.svc.Trace.BatchPacks(id)
	if err != nil {
		WriteErr(w, err)
		return
	}
	OK(w, view)
}
