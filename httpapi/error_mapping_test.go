package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/cssd-instrument-trace-service/domain"
	"example.com/cssd-instrument-trace-service/service"
)

// TestCollectBlockedReturns422 回收被业务拦截时必须返回 422 与拦截原因，
// 不能当成服务器内部错误 500。
func TestCollectBlockedReturns422(t *testing.T) {
	srv := newTestServer(t)

	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	resp := doJSON(t, http.MethodPost, srv.URL+"/api/packs", `{"barcode":"CL-001","name":"回收测试包","packType":"surgical","operator":"测试员"}`, &created)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("登记器械包失败: %d", resp.StatusCode)
	}

	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	resp = doJSON(t, http.MethodPost, srv.URL+"/api/packs/"+created.Data.ID+"/collect",
		`{"operator":"测试员","collector":"测试员"}`, &body)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("回收拦截应返回 422，实际 %d，body=%+v", resp.StatusCode, body)
	}
	if body.Message == "" || body.Message == "服务器内部错误" {
		t.Fatalf("回收拦截应返回具体原因，实际 %q", body.Message)
	}
}

// TestIssueBlockedReturns422 发放被守卫拦截时必须返回 422 与拦截原因。
func TestIssueBlockedReturns422(t *testing.T) {
	srv := newTestServer(t)

	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	resp := doJSON(t, http.MethodPost, srv.URL+"/api/packs", `{"barcode":"IS-001","name":"发放测试包","packType":"dressing","operator":"测试员"}`, &created)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("登记器械包失败: %d", resp.StatusCode)
	}

	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	resp = doJSON(t, http.MethodPost, srv.URL+"/api/packs/"+created.Data.ID+"/issue",
		`{"department":"手术一科","operatingRoom":"1号手术间","issuer":"测试员","operator":"测试员"}`, &body)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("发放拦截应返回 422，实际 %d，body=%+v", resp.StatusCode, body)
	}
}

// TestCreateBatchMaintenanceSterilizer422 使用维护中的灭菌器创建批次必须返回 422。
func TestCreateBatchMaintenanceSterilizer422(t *testing.T) {
	srv := newTestServer(t)

	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	resp := doJSON(t, http.MethodPost, srv.URL+"/api/packs", `{"barcode":"MT-001","name":"维护测试包","packType":"surgical","operator":"测试员"}`, &created)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("登记器械包失败: %d", resp.StatusCode)
	}

	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	// ster_003 为种子数据中的维护中灭菌器。
	resp = doJSON(t, http.MethodPost, srv.URL+"/api/sterilizations",
		fmt.Sprintf(`{"sterilizerId":"ster_003","operator":"测试员","tempC":134,"durationMin":5,"pressureKPa":208,"packIds":["%s"]}`, created.Data.ID), &body)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("维护中灭菌器创建批次应返回 422，实际 %d，body=%+v", resp.StatusCode, body)
	}
}

// TestWriteErrMapsBlockedError WriteErr 必须把 BlockedError 映射为 422。
func TestWriteErrMapsBlockedError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteErr(rec, &service.BlockedError{Kind: domain.ErrIssueBlocked, Reason: "器械包已过期，禁止发放"})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("BlockedError 应映射为 422，实际 %d", rec.Code)
	}
	var body Response
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if body.Message != "器械包已过期，禁止发放" {
		t.Fatalf("应返回拦截原因，实际 %q", body.Message)
	}
}

// TestWriteErrMapsBlockedSentinels 发放/回收拦截哨兵错误本身也必须映射为 422。
func TestWriteErrMapsBlockedSentinels(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteErr(rec, domain.ErrIssueBlocked)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ErrIssueBlocked 应映射为 422，实际 %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	WriteErr(rec, domain.ErrCollectBlocked)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ErrCollectBlocked 应映射为 422，实际 %d", rec.Code)
	}
}
