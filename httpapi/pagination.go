package httpapi

import (
	"net/http"
	"net/url"
	"strconv"
)

const (
	// DefaultListLimit list 接口默认返回条数。
	DefaultListLimit = 20
	// MaxListLimit list 接口单次返回上限。
	MaxListLimit = 200
)

// listMeta 分页元数据。
type listMeta struct {
	Total  int
	Limit  int
	Offset int
}

// parseListParams 解析 limit/offset 查询参数。
// 非法值（非数字、负数或超上限）通过 WriteErr/ Fail 返回 400。
func parseListParams(w http.ResponseWriter, q url.Values) (limit, offset int, ok bool) {
	limit = DefaultListLimit
	offset = 0

	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			Fail(w, http.StatusBadRequest, http.StatusBadRequest, "limit 必须是非负整数")
			return 0, 0, false
		}
		if n == 0 {
			n = DefaultListLimit
		}
		if n > MaxListLimit {
			n = MaxListLimit
		}
		limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			Fail(w, http.StatusBadRequest, http.StatusBadRequest, "offset 必须是非负整数")
			return 0, 0, false
		}
		offset = n
	}
	return limit, offset, true
}

// writeList 写入 list 接口成功响应，并通过响应头返回 total/limit/offset 分页元数据。
func writeList(w http.ResponseWriter, data any, meta listMeta) {
	w.Header().Set("X-Total-Count", strconv.Itoa(meta.Total))
	w.Header().Set("X-Limit", strconv.Itoa(meta.Limit))
	w.Header().Set("X-Offset", strconv.Itoa(meta.Offset))
	OK(w, data)
}
