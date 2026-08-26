package httpapi

import (
	"io/fs"
	"net/http"
	"strings"

	"example.com/cssd-instrument-trace-service/middleware"
	"example.com/cssd-instrument-trace-service/service"
)

// Router 组装全部 REST 路由、静态资源与中间件链。
// webFS 为 go:embed 内嵌的前端静态资源（main 包传入）。
func Router(svc *service.Services, webFS fs.FS) http.Handler {
	mux := http.NewServeMux()

	packH := NewPackHandler(svc)
	sterilH := NewSterilizationHandler(svc)
	issueH := NewIssueHandler(svc)
	traceH := NewTraceHandler(svc)
	dashH := NewDashboardHandler(svc)
	auditH := NewAuditHandler(svc)
	healthH := NewHealthHandler(svc)

	// 发放守卫中间件仅包裹发放路由。
	issueGuard := middleware.IssueGuard(svc)

	// 器械包。
	mux.HandleFunc("POST /api/packs", packH.Create)
	mux.HandleFunc("GET /api/packs", packH.List)
	mux.HandleFunc("GET /api/packs/{id}", packH.Get)
	mux.HandleFunc("POST /api/packs/{id}/cycle", packH.Cycle)
	mux.Handle("POST /api/packs/{id}/issue", issueGuard(http.HandlerFunc(issueH.Issue)))
	mux.HandleFunc("POST /api/packs/{id}/collect", issueH.Collect)
	mux.HandleFunc("GET /api/packs/{id}/trace", traceH.PackTrace)

	// 灭菌。
	mux.HandleFunc("POST /api/sterilizations", sterilH.Create)
	mux.HandleFunc("GET /api/sterilizations", sterilH.List)
	mux.HandleFunc("GET /api/sterilizations/{id}", sterilH.Get)
	mux.HandleFunc("POST /api/sterilizations/{id}/complete", sterilH.Complete)
	mux.HandleFunc("GET /api/sterilizations/{id}/packs", traceH.BatchPacks)
	mux.HandleFunc("GET /api/sterilizers", sterilH.ListSterilizers)
	mux.HandleFunc("POST /api/sterilizers", sterilH.CreateSterilizer)

	// 发放回收 / 追溯 / 总览 / 审计。
	mux.HandleFunc("GET /api/issues", issueH.List)
	mux.HandleFunc("GET /api/lost", issueH.Lost)
	mux.HandleFunc("GET /api/trace", traceH.TraceByBarcode)
	mux.HandleFunc("GET /api/dashboard", dashH.Get)
	mux.HandleFunc("GET /api/audit-logs", auditH.List)

	// 健康检查（含 /healthz、/readyz 与 /api/healthz）。
	mux.HandleFunc("GET /healthz", healthH.Get)
	mux.HandleFunc("GET /readyz", healthH.Get)
	mux.HandleFunc("GET /api/healthz", healthH.Get)

	// 前端静态资源与 SPA 回退。
	mux.Handle("/", spaHandler(webFS))

	// 中间件链：requestID（最外层）→ 请求日志 → 错误恢复 → 安全头 → 审计。
	// RequestID 必须位于最外层，这样日志、恢复与业务 handler 都能读取到请求 ID。
	return middleware.RequestID(
		middleware.RequestLog(
			middleware.ErrorHandler(
				middleware.SecurityHeaders(
					middleware.AuditLogger(svc)(mux),
				),
			),
		),
	)
}

// spaHandler 提供内嵌静态资源；非 API 路径找不到文件时回退到 index.html，
// 以支持前端路径路由（/packs、/sterilization 等）。
func spaHandler(webFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(webFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(webFS, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		data, err := fs.ReadFile(webFS, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})
}
