package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// ErrorHandler 统一错误/panic 处理中间件：
// 拦截 handler 内部 panic，返回统一 JSON 格式的 500 响应，避免进程崩溃。
//
// 若 panic 发生在响应已提交之后（已写状态行或响应体），状态行已发往
// 客户端，此时无法再改写状态码；强行补写 500 会在响应体尾部拼接出
// 「半截 200 + 500 错误体」的乱码，且会让日志把客户端实际拿到的 200
// 记成 500。故此时只记录错误、不再向客户端写任何字节，使日志记录的
// status 与客户端实际收到的状态码保持一致。
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
					"response_committed", isCommitted(w),
				)
				writeRecovered(w)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// isCommitted 返回响应是否已提交（仅能识别 StatusRecorder，未包装时按未提交处理）。
func isCommitted(w http.ResponseWriter) bool {
	if rec, ok := w.(*StatusRecorder); ok {
		return rec.Committed()
	}
	return false
}

// writeRecovered 向客户端返回 500 统一错误响应，不暴露内部堆栈。
// 响应已提交时跳过写头/写体，避免在已发送的响应上拼接乱码。
func writeRecovered(w http.ResponseWriter) {
	if isCommitted(w) {
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(`{"code":500,"message":"服务器内部错误"}`))
}
