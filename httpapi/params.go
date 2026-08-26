package httpapi

import (
	"net/http"
	"strings"
)

// requirePathID 读取路径参数 id，非法时返回 400。
func requirePathID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		Fail(w, http.StatusBadRequest, http.StatusBadRequest, "路径参数 id 不能为空")
		return "", false
	}
	return id, true
}

// requireString 校验字符串非空，非法时返回 400。
func requireString(w http.ResponseWriter, name, value string) bool {
	if strings.TrimSpace(value) == "" {
		Fail(w, http.StatusBadRequest, http.StatusBadRequest, name+" 不能为空")
		return false
	}
	return true
}
