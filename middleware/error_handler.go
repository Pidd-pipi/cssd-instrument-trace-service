package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// ErrorHandler 统一错误/panic 处理中间件：
// 拦截 handler 内部 panic，返回统一 JSON 格式的 500 响应，避免进程崩溃。
func ErrorHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic_recovered",
					"request_id", RequestIDFromContext(r.Context()),
					"method", r.Method,
					"path", r.URL.Path,
					"panic", rec,
					"stack", string(debug.Stack()),
				)
				writeRecovered(w)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// writeRecovered 向客户端返回 500 统一错误响应，不暴露内部堆栈。
func writeRecovered(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(`{"code":500,"message":"服务器内部错误"}`))
}
