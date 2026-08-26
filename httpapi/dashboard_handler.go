package httpapi

import (
	"net/http"

	"example.com/cssd-instrument-trace-service/service"
)

// DashboardHandler 工作台总览接口。
type DashboardHandler struct {
	svc *service.Services
}

// NewDashboardHandler 构建工作台 handler。
func NewDashboardHandler(svc *service.Services) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// Get 工作台总览：GET /api/dashboard。
func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	OK(w, h.svc.Stats.Dashboard())
}
