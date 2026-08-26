package httpapi

import (
	"net/http"
	"strings"
	"time"

	"example.com/cssd-instrument-trace-service/domain"
	"example.com/cssd-instrument-trace-service/service"
	"example.com/cssd-instrument-trace-service/store"
)

// IssueHandler 发放回收相关接口。
type IssueHandler struct {
	svc *service.Services
}

// NewIssueHandler 构建发放回收 handler。
func NewIssueHandler(svc *service.Services) *IssueHandler {
	return &IssueHandler{svc: svc}
}

// issueRequest 发放登记请求。
type issueRequest struct {
	domain.IssueInput
	Operator string `json:"operator"`
}

// Issue 发放器械包：POST /api/packs/{id}/issue。
func (h *IssueHandler) Issue(w http.ResponseWriter, r *http.Request) {
	id, ok := requirePathID(w, r)
	if !ok {
		return
	}
	var req issueRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	actor := service.Actor{Operator: req.Operator, IP: clientIP(r)}
	record, err := h.svc.Issues.Issue(id, req.IssueInput, actor)
	if err != nil {
		WriteErr(w, err)
		return
	}
	OK(w, record)
}

// collectRequest 回收请求。
type collectRequest struct {
	Operator  string `json:"operator"`
	Collector string `json:"collector"`
}

// Collect 回收器械包：POST /api/packs/{id}/collect。
func (h *IssueHandler) Collect(w http.ResponseWriter, r *http.Request) {
	id, ok := requirePathID(w, r)
	if !ok {
		return
	}
	var req collectRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	if strings.TrimSpace(req.Operator) == "" && strings.TrimSpace(req.Collector) == "" {
		Fail(w, http.StatusBadRequest, http.StatusBadRequest, "operator 与 collector 至少填写一项")
		return
	}
	collector := req.Collector
	if collector == "" {
		collector = req.Operator
	}
	actor := service.Actor{Operator: req.Operator, IP: clientIP(r)}
	record, err := h.svc.Issues.Collect(id, collector, actor)
	if err != nil {
		WriteErr(w, err)
		return
	}
	OK(w, record)
}

// List 发放记录列表：GET /api/issues?status=&packId=&limit=&offset=。
func (h *IssueHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := parseListParams(w, r.URL.Query())
	if !ok {
		return
	}
	q := r.URL.Query()
	status := q.Get("status")
	if status != "" && !isValidIssueStatus(status) {
		Fail(w, http.StatusBadRequest, http.StatusBadRequest, "status 参数不合法")
		return
	}
	base := store.IssueFilter{
		Status: status,
		PackID: q.Get("packId"),
	}
	total := len(h.svc.Issues.ListIssues(base))
	page := base
	page.Limit = limit
	page.Offset = offset
	writeList(w, h.svc.Issues.ListIssues(page), listMeta{Total: total, Limit: limit, Offset: offset})
}

// Lost 丢失待查清单：GET /api/lost。
func (h *IssueHandler) Lost(w http.ResponseWriter, r *http.Request) {
	OK(w, h.svc.Issues.LostList(time.Now()))
}

func isValidIssueStatus(status string) bool {
	switch domain.IssueStatus(status) {
	case domain.IssueOpen, domain.IssueReturned, domain.IssueLost:
		return true
	default:
		return false
	}
}
