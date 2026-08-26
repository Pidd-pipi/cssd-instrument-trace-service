package httpapi

import (
	"net/http"

	"example.com/cssd-instrument-trace-service/domain"
	"example.com/cssd-instrument-trace-service/service"
)

// SterilizationHandler 灭菌管理相关接口。
type SterilizationHandler struct {
	svc *service.Services
}

// NewSterilizationHandler 构建灭菌 handler。
func NewSterilizationHandler(svc *service.Services) *SterilizationHandler {
	return &SterilizationHandler{svc: svc}
}

// createBatchRequest 创建灭菌批次请求。
type createBatchRequest struct {
	domain.CreateBatchInput
	Operator string `json:"operator"`
}

// Create 创建灭菌批次：POST /api/sterilizations。
func (h *SterilizationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createBatchRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	actor := service.Actor{Operator: req.Operator, IP: clientIP(r)}
	batch, err := h.svc.Sterilizations.CreateBatch(req.CreateBatchInput, actor)
	if err != nil {
		WriteErr(w, err)
		return
	}
	OK(w, batch)
}

// List 灭菌批次列表：GET /api/sterilizations?limit=&offset=。
func (h *SterilizationHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := parseListParams(w, r.URL.Query())
	if !ok {
		return
	}
	total := len(h.svc.Sterilizations.ListBatches(0))
	writeList(w, h.svc.Sterilizations.ListBatchesPage(limit, offset), listMeta{Total: total, Limit: limit, Offset: offset})
}

// Get 批次详情：GET /api/sterilizations/{id}。
func (h *SterilizationHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := requirePathID(w, r)
	if !ok {
		return
	}
	batch, err := h.svc.Sterilizations.GetBatch(id)
	if err != nil {
		WriteErr(w, err)
		return
	}
	OK(w, batch)
}

// completeBatchRequest 完成批次请求。
type completeBatchRequest struct {
	Operator string `json:"operator"`
}

// Complete 完成灭菌（参数判定 + 器械包状态更新）：POST /api/sterilizations/{id}/complete。
func (h *SterilizationHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id, ok := requirePathID(w, r)
	if !ok {
		return
	}
	var req completeBatchRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	actor := service.Actor{Operator: req.Operator, IP: clientIP(r)}
	batch, err := h.svc.Sterilizations.CompleteBatch(id, actor)
	if err != nil {
		OK(w, nil)
		return
	}
	OK(w, batch)
}

// ListSterilizers 灭菌器列表：GET /api/sterilizers。
func (h *SterilizationHandler) ListSterilizers(w http.ResponseWriter, r *http.Request) {
	OK(w, h.svc.Sterilizations.ListSterilizers())
}

// createSterilizerRequest 灭菌器登记请求。
type createSterilizerRequest struct {
	domain.CreateSterilizerInput
	Operator string `json:"operator"`
}

// CreateSterilizer 登记灭菌器：POST /api/sterilizers。
func (h *SterilizationHandler) CreateSterilizer(w http.ResponseWriter, r *http.Request) {
	var req createSterilizerRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	actor := service.Actor{Operator: req.Operator, IP: clientIP(r)}
	st, err := h.svc.Sterilizations.CreateSterilizer(req.CreateSterilizerInput, actor)
	if err != nil {
		WriteErr(w, err)
		return
	}
	OK(w, st)
}
