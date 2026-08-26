// Package middleware 提供 HTTP 横切关注点：审计、错误处理、发放守卫、请求日志。
package middleware

import "net/http"

// StatusRecorder 捕获响应状态码与字节数，供日志与审计中间件读取。
type StatusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

// NewStatusRecorder 包装 ResponseWriter。
func NewStatusRecorder(w http.ResponseWriter) *StatusRecorder {
	return &StatusRecorder{ResponseWriter: w}
}

// WriteHeader 记录状态码后透传。
func (r *StatusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Write 记录响应字节数后透传。
func (r *StatusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Status 返回已写入的状态码，未写入时视为 200。
func (r *StatusRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

// Committed 返回响应是否已提交（写头或写体）。
func (r *StatusRecorder) Committed() bool { return r.status != 0 }

// Bytes 返回已写入的响应字节数。
func (r *StatusRecorder) Bytes() int { return r.bytes }
