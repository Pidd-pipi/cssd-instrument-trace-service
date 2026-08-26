package httpapi

import (
	"net/http"

	"example.com/cssd-instrument-trace-service/domain"
	"example.com/cssd-instrument-trace-service/service"
	"example.com/cssd-instrument-trace-service/store"
)

// PackHandler 器械包相关接口。
type PackHandler struct {
	svc *service.Services
}

// NewPackHandler 构建器械包 handler。
func NewPackHandler(svc *service.Services) *PackHandler {
	return &PackHandler{svc: svc}
}

// createPackRequest 器械包登记请求。
type createPackRequest struct {
	domain.RegisterPackInput
	Operator string `json:"operator"`
}

// Create 登记器械包：POST /api/packs。
func (h *PackHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createPackRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	actor := service.Actor{Operator: req.Operator, IP: clientIP(r)}
	pack, err := h.svc.Packs.Register(req.RegisterPackInput, actor)
	if err != nil {
		WriteErr(w, err)
		return
	}
	OK(w, pack)
}

// List 器械包列表：GET /api/packs?stage=&type=&keyword=&limit=&offset=。
// 响应体保持数组以兼容现有前端；分页元数据通过 X-Total-Count/X-Limit/X-Offset 返回。
func (h *PackHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := parseListParams(w, r.URL.Query())
	if !ok {
		return
	}
	q := r.URL.Query()
	stage := q.Get("stage")
	if stage != "" && !domain.IsValidStage(domain.PackStage(stage)) {
		Fail(w, http.StatusBadRequest, http.StatusBadRequest, "stage 参数不合法")
		return
	}
	packType := q.Get("type")
	if packType != "" && !domain.IsValidPackType(domain.PackType(packType)) {
		Fail(w, http.StatusBadRequest, http.StatusBadRequest, "type 参数不合法")
		return
	}
	base := store.PackFilter{
		Stage:    stage,
		PackType: packType,
		Keyword:  q.Get("keyword"),
	}
	total := len(h.svc.Packs.List(base))
	page := base
	page.Limit = limit
	page.Offset = offset
	writeList(w, h.svc.Packs.List(page), listMeta{Total: total, Limit: limit, Offset: offset})
}

// Get 器械包详情：GET /api/packs/{id}。
func (h *PackHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := requirePathID(w, r)
	if !ok {
		return
	}
	pack, err := h.svc.Packs.Get(id)
	if err != nil {
		WriteErr(w, err)
		return
	}
	OK(w, pack)
}

// cycleRequest 环节流转请求。
type cycleRequest struct {
	Stage    domain.PackStage `json:"stage"`
	Operator string           `json:"operator"`
	DeviceID string           `json:"deviceId"`
	Note     string           `json:"note"`
}

// Cycle 环节流转：POST /api/packs/{id}/cycle。
func (h *PackHandler) Cycle(w http.ResponseWriter, r *http.Request) {
	id, ok := requirePathID(w, r)
	if !ok {
		return
	}
	var req cycleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	if !domain.IsValidStage(req.Stage) {
		Fail(w, http.StatusBadRequest, http.StatusBadRequest, "stage 参数不合法")
		return
	}
	actor := service.Actor{Operator: req.Operator, IP: clientIP(r)}
	pack, err := h.svc.Packs.Cycle(id, req.Stage, req.DeviceID, req.Note, actor)
	if err != nil {
		WriteErr(w, err)
		return
	}
	OK(w, pack)
}
