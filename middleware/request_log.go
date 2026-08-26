package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

// RequestLog 使用结构化日志记录每个 HTTP 请求的方法、路径、状态码、字节数与耗时。
func RequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := NewStatusRecorder(w)
		next.ServeHTTP(rec, r)
		slog.Info("http_request",
			"request_id", RequestIDFromContext(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"status", rec.Status(),
			"bytes", rec.Bytes(),
			"duration_ms", time.Since(start).Seconds()*1000,
			"client_ip", clientIP(r),
		)
	})
}

// clientIP 提取客户端 IP，兼容 X-Forwarded-For / X-Real-IP 与裸 RemoteAddr。
func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(firstForwardedIP(r.Header.Get("X-Forwarded-For"))); ip != "" {
		return ip
	}
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func firstForwardedIP(s string) string {
	if s == "" {
		return ""
	}
	if idx := strings.IndexByte(s, ','); idx >= 0 {
		return s[:idx]
	}
	return s
}
