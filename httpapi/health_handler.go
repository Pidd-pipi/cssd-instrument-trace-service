package httpapi

import (
	"net/http"
	"time"

	"example.com/cssd-instrument-trace-service/service"
)

// HealthHandler 健康检查。
type HealthHandler struct {
	svc *service.Services
}

// NewHealthHandler 构建健康检查 handler。
func NewHealthHandler(svc *service.Services) *HealthHandler {
	return &HealthHandler{svc: svc}
}

// Get 返回服务健康状态：仓储可用 + 服务就绪。
func (h *HealthHandler) Get(w http.ResponseWriter, r *http.Request) {
	OK(w, map[string]any{
		"status":  "ok",
		"service": "cssd-instrument-trace-service",
		"packs":   h.svc.Store.CountPacks(),
		"time":    time.Now().Format(time.RFC3339),
	})
}
