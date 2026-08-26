package httpapi

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"

	"example.com/cssd-instrument-trace-service/config"
	"example.com/cssd-instrument-trace-service/service"
	"example.com/cssd-instrument-trace-service/store"
)

// newTestServer 构建带内存仓储（临时文件）的测试 HTTP 服务。
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "store.json"))
	if err != nil {
		t.Fatalf("创建仓储失败: %v", err)
	}
	svc := service.New(st, config.DefaultRules())
	webFS := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!DOCTYPE html><html><body>test</body></html>")},
	}
	srv := httptest.NewServer(Router(svc, webFS))
	t.Cleanup(srv.Close)
	return srv
}

// doJSON 发送 JSON 请求并解析统一响应。
func doJSON(t *testing.T, method, url, body string, out any) *http.Response {
	t.Helper()
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s 请求失败: %v", method, url, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if out != nil {
		data, _ := ioutil.ReadAll(resp.Body)
		if err := json.Unmarshal(data, out); err != nil {
			t.Fatalf("解析响应失败 %s: %v\nbody=%s", url, err, data)
		}
	}
	return resp
}

// unwrapData 提取统一响应中的 data 字段。
func unwrapData(t *testing.T, raw json.RawMessage, dst any) {
	t.Helper()
	if err := json.Unmarshal(raw, dst); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
}

func TestFullBusinessFlow(t *testing.T) {
	srv := newTestServer(t)
	base := srv.URL

	// 1. 健康检查。
	resp := doJSON(t, http.MethodGet, base+"/healthz", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz 状态码 = %d, want 200", resp.StatusCode)
	}

	// 2. 登记器械包。
	var created struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	resp = doJSON(t, http.MethodPost, base+"/api/packs",
		`{"barcode":"PK-E2E-001","name":"腔镜手术包","packType":"surgical","operator":"张三"}`, &created)
	if resp.StatusCode != http.StatusOK || created.Code != 0 {
		t.Fatalf("登记失败 status=%d resp=%+v", resp.StatusCode, created)
	}
	var pack struct {
		ID    string `json:"id"`
		Stage string `json:"stage"`
	}
	unwrapData(t, created.Data, &pack)
	if pack.Stage != "to_collect" {
		t.Fatalf("初始环节 = %s, want to_collect", pack.Stage)
	}

	// 3. 按序推进：已回收 → 清洗中 → 已清洗。
	for _, stage := range []string{"collected", "washing", "washed"} {
		resp := doJSON(t, http.MethodPost, base+"/api/packs/"+pack.ID+"/cycle",
			`{"stage":"`+stage+`","operator":"张三","deviceId":"washer_01"}`, &created)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("流转到 %s 失败: %d", stage, resp.StatusCode)
		}
	}

	// 4. 创建灭菌批次（参数达标）。
	var batchResp struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	resp = doJSON(t, http.MethodPost, base+"/api/sterilizations",
		`{"sterilizerId":"ster_001","operator":"李四","tempC":134,"durationMin":6,"pressureKPa":210,"packIds":["`+pack.ID+`"]}`, &batchResp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("创建批次失败: %d", resp.StatusCode)
	}
	var batch struct {
		ID      string `json:"id"`
		BatchNo string `json:"batchNo"`
		Status  string `json:"status"`
	}
	unwrapData(t, batchResp.Data, &batch)

	// 5. 完成灭菌（判定合格）。
	resp = doJSON(t, http.MethodPost, base+"/api/sterilizations/"+batch.ID+"/complete",
		`{"operator":"李四"}`, &batchResp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("完成批次失败: %d", resp.StatusCode)
	}
	var completed struct {
		Result string `json:"result"`
	}
	unwrapData(t, batchResp.Data, &completed)
	if completed.Result != "pass" {
		t.Fatalf("判定结果 = %s, want pass", completed.Result)
	}

	// 6. 发放。
	var issueResp struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	resp = doJSON(t, http.MethodPost, base+"/api/packs/"+pack.ID+"/issue",
		`{"department":"普外科","operatingRoom":"3号手术间","issuer":"王护士","operator":"王护士"}`, &issueResp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("发放失败: %d", resp.StatusCode)
	}

	// 7. 使用中 → 回收闭环。
	resp = doJSON(t, http.MethodPost, base+"/api/packs/"+pack.ID+"/cycle",
		`{"stage":"in_use","operator":"王护士"}`, &created)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("标记使用中失败: %d", resp.StatusCode)
	}
	var collectResp struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	resp = doJSON(t, http.MethodPost, base+"/api/packs/"+pack.ID+"/collect",
		`{"operator":"回收员","collector":"回收员小赵"}`, &collectResp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("回收失败: %d", resp.StatusCode)
	}

	// 8. 追溯查询：应包含完整环节记录与发放记录。
	var traceResp struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	resp = doJSON(t, http.MethodGet, base+"/api/packs/"+pack.ID+"/trace", "", &traceResp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("追溯查询失败: %d", resp.StatusCode)
	}
	var trace struct {
		Cycles []json.RawMessage `json:"cycles"`
		Issues []json.RawMessage `json:"issues"`
	}
	unwrapData(t, traceResp.Data, &trace)
	if len(trace.Cycles) < 7 { // 登记+3流转+装载+灭菌判定+发放+使用+回收
		t.Errorf("追溯环节记录数 = %d, 期望 >= 7", len(trace.Cycles))
	}
	if len(trace.Issues) != 1 {
		t.Errorf("追溯发放记录数 = %d, want 1", len(trace.Issues))
	}

	// 9. 批次器械包去向。
	resp = doJSON(t, http.MethodGet, base+"/api/sterilizations/"+batch.ID+"/packs", "", &traceResp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("批次去向查询失败: %d", resp.StatusCode)
	}

	// 10. 工作台与审计接口可用。
	resp = doJSON(t, http.MethodGet, base+"/api/dashboard", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("工作台接口失败: %d", resp.StatusCode)
	}
	resp = doJSON(t, http.MethodGet, base+"/api/audit-logs", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("审计接口失败: %d", resp.StatusCode)
	}
}

func TestIssueGuardBlocksInvalidIssue(t *testing.T) {
	srv := newTestServer(t)
	base := srv.URL

	var created struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	resp := doJSON(t, http.MethodPost, base+"/api/packs",
		`{"barcode":"PK-GUARD-001","name":"未灭菌包","packType":"surgical","operator":"张三"}`, &created)
	if resp.StatusCode != http.StatusOK {
		t.Fatal("登记失败")
	}
	var pack struct {
		ID string `json:"id"`
	}
	unwrapData(t, created.Data, &pack)

	// 未灭菌器械包发放应被守卫拦截（422）。
	resp = doJSON(t, http.MethodPost, base+"/api/packs/"+pack.ID+"/issue",
		`{"department":"普外科","operatingRoom":"1号","issuer":"护士"}`, &created)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("守卫应返回 422，实际 %d", resp.StatusCode)
	}
}

func TestSPAFallback(t *testing.T) {
	srv := newTestServer(t)
	for _, path := range []string{"/", "/packs", "/sterilization", "/issue", "/trace"} {
		resp := doJSON(t, http.MethodGet, srv.URL+path, "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("页面 %s 状态码 = %d, want 200", path, resp.StatusCode)
		}
	}
}

// 确保 fs.FS 被引用（Router 参数类型）。
var _ fs.FS = fstest.MapFS{}
