package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	"example.com/cssd-instrument-trace-service/domain"
	"example.com/cssd-instrument-trace-service/service"
)

// IssueGuard 发放前置校验守卫中间件：
// 在进入发放 handler 之前快速校验「已灭菌 + 未过期 + 批次参数合格」，
// 不满足则直接返回 422，避免无效发放请求进入业务层。
func IssueGuard(svc *service.Services) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.PathValue("id")
			if id == "" {
				next.ServeHTTP(w, r)
				return
			}
			pack, err := svc.Packs.Get(id)
			if err != nil {
				guardJSON(w, http.StatusNotFound, "资源不存在")
				return
			}
			if ok, reason := pack.CanBeIssued(time.Now()); !ok {
				guardJSON(w, http.StatusUnprocessableEntity, reason)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// guardJSON 以统一响应格式写出错误响应（避免与 httpapi 包产生循环依赖）。
func guardJSON(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    status,
		"message": message,
		"data":    domain.ErrIssueBlocked.Error(),
	})
}
