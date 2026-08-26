package httpapi

import (
	"net/http"
	"testing"
)

func TestPackListPaginationAndHeaders(t *testing.T) {
	srv := newTestServer(t)
	base := srv.URL

	for _, barcode := range []string{"PK-PAGE-001", "PK-PAGE-002", "PK-PAGE-003"} {
		resp := doJSON(t, http.MethodPost, base+"/api/packs",
			`{"barcode":"`+barcode+`","name":"分页包","packType":"surgical","operator":"测试"}`, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("登记 %s 失败: %d", barcode, resp.StatusCode)
		}
	}

	resp := doJSON(t, http.MethodGet, base+"/api/packs?limit=2&offset=1", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("分页列表失败: %d", resp.StatusCode)
	}
	if got := resp.Header.Get("X-Total-Count"); got != "3" {
		t.Fatalf("X-Total-Count = %q, want 3", got)
	}
	if got := resp.Header.Get("X-Limit"); got != "2" {
		t.Fatalf("X-Limit = %q, want 2", got)
	}
	if got := resp.Header.Get("X-Offset"); got != "1" {
		t.Fatalf("X-Offset = %q, want 1", got)
	}
}

func TestPackListRejectsInvalidPagination(t *testing.T) {
	srv := newTestServer(t)
	resp := doJSON(t, http.MethodGet, srv.URL+"/api/packs?limit=abc", "", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("非法 limit 应返回 400，实际 %d", resp.StatusCode)
	}
	resp = doJSON(t, http.MethodGet, srv.URL+"/api/packs?stage=bad_stage", "", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("非法 stage 应返回 400，实际 %d", resp.StatusCode)
	}
}
