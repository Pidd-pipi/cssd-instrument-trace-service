package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"

	"example.com/cssd-instrument-trace-service/config"
	"example.com/cssd-instrument-trace-service/domain"
	"example.com/cssd-instrument-trace-service/service"
	"example.com/cssd-instrument-trace-service/store"
)

// TestCompleteHandlerDoesNotSwallowError 完成含缺失器械包的批次时，
// 接口必须返回错误状态码，不能吞掉错误返回 200。
func TestCompleteHandlerDoesNotSwallowError(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "store.json"))
	if err != nil {
		t.Fatalf("创建仓储失败: %v", err)
	}
	svc := service.New(st, config.DefaultRules())

	pack := domain.NewPack(domain.RegisterPackInput{Barcode: "SW-001", Name: "吞错测试包", PackType: domain.TypeSurgical})
	pack.Stage = domain.StageSterilizing
	if err := st.SavePack(pack); err != nil {
		t.Fatalf("保存器械包失败: %v", err)
	}
	batch := domain.NewBatch(domain.CreateBatchInput{
		SterilizerID: "ster_001",
		Operator:     "测试员",
		TempC:        134,
		DurationMin:  5,
		PressureKPa:  208,
		PackIDs:      []string{pack.ID, "missing-pack-id"},
	}, "高温高压灭菌器 A", "SB20260826-777")
	if err := st.SaveBatch(batch); err != nil {
		t.Fatalf("保存批次失败: %v", err)
	}

	webFS := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<!DOCTYPE html><html><body>test</body></html>")}}
	srv := httptest.NewServer(Router(svc, webFS))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/sterilizations/"+batch.ID+"/complete", bytes.NewReader([]byte(`{"operator":"测试员"}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("完成含缺失器械包的批次应返回 409，实际 %d body=%s", resp.StatusCode, string(body))
	}
	var out Response
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if out.Code == 0 {
		t.Fatalf("失败响应 code 不应为 0: %s", string(body))
	}
}

// 保持 fs 引用避免未使用告警。
var _ fs.FS
