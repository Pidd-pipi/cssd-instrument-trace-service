// Package middleware 提供 HTTP 横切关注点：审计、错误处理、发放守卫、请求日志。
package middleware

import (
	"bufio"
	"net"
	"net/http"
)

// StatusRecorder 捕获响应状态码与字节数，供日志与审计中间件读取。
type StatusRecorder struct {
	http.ResponseWriter
	// status 记录已向客户端提交的状态码；一旦提交即不可变，
	// 即使下游再次调用 WriteHeader（例如 panic 恢复时补写 500）也不覆盖，
	// 保证日志/审计记录的 status 与客户端实际收到的状态码一致。
	status    int
	committed bool
	bytes     int
}

// NewStatusRecorder 包装 ResponseWriter。
func NewStatusRecorder(w http.ResponseWriter) *StatusRecorder {
	return &StatusRecorder{ResponseWriter: w}
}

// WriteHeader 记录状态码后透传。
//
// 响应一旦提交（首次写头或写体），后续 WriteHeader 不再改变已记录的
// status、也不再向底层 ResponseWriter 透传——此时状态行已经发往客户端，
// 再写只会触发 "superfluous response.WriteHeader call" 并在响应体尾部
// 拼接乱码。这是导致「客户端拿到 200 + 乱码尾巴、日志却记 500」的根因。
func (r *StatusRecorder) WriteHeader(code int) {
	if r.committed {
		return
	}
	r.status = code
	r.committed = true
	r.ResponseWriter.WriteHeader(code)
}

// Write 记录响应字节数后透传。
func (r *StatusRecorder) Write(b []byte) (int, error) {
	if !r.committed {
		r.status = http.StatusOK
		r.committed = true
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Status 返回已向客户端提交的状态码，未写入时视为 200。
// 该值反映客户端实际收到的状态码，不受恢复流程后续覆盖影响。
func (r *StatusRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

// Committed 返回响应是否已提交（写头或写体）。
// 已提交时状态行已发往客户端，恢复逻辑不得再写头或写体。
func (r *StatusRecorder) Committed() bool { return r.committed }

// Bytes 返回已写入的响应字节数。
func (r *StatusRecorder) Bytes() int { return r.bytes }

// Unwrap 暴露底层 ResponseWriter，使 http.NewResponseController 能解析到底层
// 支持的接口。本类型已显式代理 Flush/Hijack/Push，这些方法的调用仍会走本类型
// 的包装（同步维护 committed），不会被 Unwrap 短路绕过状态追踪。
func (r *StatusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// Flush 实现 http.Flusher。刷新会把已缓冲的响应头/体发往客户端，等价于提交响应，
// 因此必须像 Write 那样锁定 status=200（若未显式写头）并置 committed；
// 否则后续 panic 恢复会误以为响应未提交而补写 500，造成乱码尾巴与日志状态码错乱。
func (r *StatusRecorder) Flush() {
	if !r.committed {
		r.status = http.StatusOK
		r.committed = true
	}
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack 实现 http.Hijacker，供 WebSocket 升级等场景取走底层连接。
// 劫持意味着原始响应已提交（状态行已发送），故同样锁定 committed，
// 避免 panic 恢复在已提交响应上追加写入造成乱码。
func (r *StatusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if !r.committed {
		r.status = http.StatusOK
		r.committed = true
	}
	hj, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hj.Hijack()
}

// Push 实现 http.Pusher（HTTP/2 服务器推送）。推送不提交主响应状态行，
// 故不改变 committed；底层不支持时返回 http.ErrNotSupported，与标准库行为一致。
func (r *StatusRecorder) Push(target string, opts *http.PushOptions) error {
	if p, ok := r.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}
