package httpapi

import (
	"net/http"

	"example.com/cssd-instrument-trace-service/service"
)

// AuditHandler 审计日志查询接口。
type AuditHandler struct {
	svc *service.Services
}

// NewAuditHandler 构建审计 handler。
func NewAuditHandler(svc *service.Services) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// List 审计日志列表：GET /api/audit-logs?limit=&offset=。
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := parseListParams(w, r.URL.Query())
	if !ok {
		return
	}
	total := len(h.svc.Audit.List(0))
	writeList(w, h.svc.Audit.ListPage(limit, offset), listMeta{Total: total, Limit: limit, Offset: offset})
}
