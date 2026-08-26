// Package httpapi 提供 REST 接口层，与前端页面一一对应。
package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"example.com/cssd-instrument-trace-service/domain"
)

// maxJSONBodyBytes 限制 JSON 请求体大小，避免无界读取。
const maxJSONBodyBytes = 1 << 20

// Response 统一响应格式：code 为 0 表示成功，非 0 表示业务错误码。
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// OK 返回成功响应。
func OK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, Response{Code: 0, Message: "ok", Data: data})
}

// Fail 返回失败响应。
func Fail(w http.ResponseWriter, status, code int, message string) {
	writeJSON(w, status, Response{Code: code, Message: message})
}

// WriteErr 将领域/服务错误映射为统一错误响应。
func WriteErr(w http.ResponseWriter, err error) {
	status, message := mapError(err)
	writeJSON(w, status, Response{Code: status, Message: message})
}

// mapError 将业务错误映射为 HTTP 状态码与可读消息。
func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, domain.ErrInvalidParam):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, domain.ErrDuplicateBarcode):
		return http.StatusConflict, err.Error()
	case errors.Is(err, domain.ErrInvalidTransition):
		return http.StatusConflict, err.Error()
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, err.Error()
	default:
		return http.StatusInternalServerError, "服务器内部错误"
	}
}

// writeJSON 序列化并写入 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, resp Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("写响应失败", "error", err)
	}
}

// decodeJSON 解析请求体 JSON，失败时返回 400。仅允许一个 JSON 值，
// 拒绝未知字段与超长请求体，避免静默忽略错误输入。
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		Fail(w, http.StatusBadRequest, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		Fail(w, http.StatusBadRequest, http.StatusBadRequest, "请求体只能包含一个 JSON 对象")
		return err
	}
	return nil
}

// clientIP 提取客户端 IP。
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
