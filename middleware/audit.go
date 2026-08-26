package middleware

import (
	"net/http"
	"strings"

	"example.com/cssd-instrument-trace-service/domain"
	"example.com/cssd-instrument-trace-service/service"
)

// AuditLogger 请求级审计日志中间件：
// 对非静态、非健康检查的 API 请求记录一条 http.request 审计日志。
func AuditLogger(svc *service.Services) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec, ok := w.(*StatusRecorder)
			if !ok {
				rec = NewStatusRecorder(w)
				next.ServeHTTP(rec, r)
				return
			}
			next.ServeHTTP(rec, r)
			if !shouldAudit(r) {
				return
			}
			svc.Audit.RecordOperator(domain.ActionHTTPRequest, "system", "http", r.URL.Path, map[string]any{
				"method": r.Method,
				"status": rec.Status(),
				"path":   r.URL.Path,
			})
		})
	}
}

// shouldAudit 判断请求是否需要记录请求级审计。
func shouldAudit(r *http.Request) bool {
	if r.Method == http.MethodGet {
		return false // 读操作由业务留痕与访问日志覆盖，避免审计膨胀
	}
	if !strings.HasPrefix(r.URL.Path, "/api/") {
		return false
	}
	if strings.HasSuffix(r.URL.Path, "/healthz") {
		return false
	}
	return true
}
