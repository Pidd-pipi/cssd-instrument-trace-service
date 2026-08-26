package store

import (
	"testing"
	"time"

	"example.com/cssd-instrument-trace-service/domain"
)

// saveLostIssue 保存一条「丢失待查」的发放记录。
func saveLostIssue(t *testing.T, st *Store, pack *domain.InstrumentPack) *domain.IssueRecord {
	t.Helper()
	issue := domain.NewIssueRecord(pack, domain.IssueInput{
		Department: "手术一科", OperatingRoom: "1号手术间", Issuer: "测试员",
	}, "batch_1")
	if err := st.SaveIssue(issue); err != nil {
		t.Fatalf("保存发放记录失败: %v", err)
	}
	if err := st.UpdateIssue(issue.ID, func(r *domain.IssueRecord) error {
		r.MarkLost()
		return nil
	}); err != nil {
		t.Fatalf("标记丢失失败: %v", err)
	}
	return issue
}

// TestGetOpenIssueByPackSkipsLost 开放记录查询不得返回丢失待查记录。
func TestGetOpenIssueByPackSkipsLost(t *testing.T) {
	st := newTestStore(t)
	pack := domain.NewPack(domain.RegisterPackInput{Barcode: "LS-001", Name: "丢失测试包", PackType: domain.TypeSurgical})
	if err := st.SavePack(pack); err != nil {
		t.Fatalf("保存器械包失败: %v", err)
	}
	saveLostIssue(t, st, pack)

	if open := st.GetOpenIssueByPack(pack.ID); open != nil {
		t.Fatalf("丢失待查记录不应被当作开放记录返回: %+v", open)
	}
}

// TestListOpenIssuesOlderThanSkipsLost 超时扫描列表不得包含丢失待查记录。
func TestListOpenIssuesOlderThanSkipsLost(t *testing.T) {
	st := newTestStore(t)
	pack := domain.NewPack(domain.RegisterPackInput{Barcode: "LS-002", Name: "丢失扫描测试包", PackType: domain.TypeSurgical})
	if err := st.SavePack(pack); err != nil {
		t.Fatalf("保存器械包失败: %v", err)
	}
	saveLostIssue(t, st, pack)

	records := st.ListOpenIssuesOlderThan(time.Now().Add(24 * time.Hour))
	for _, r := range records {
		if r.PackID == pack.ID {
			t.Fatalf("丢失待查记录不应出现在开放扫描列表: %+v", r)
		}
	}
}
