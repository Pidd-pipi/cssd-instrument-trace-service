package domain

import "testing"

// TestLostIssueNotOpen 标记丢失待查的发放记录不再视为未回收开放记录。
func TestLostIssueNotOpen(t *testing.T) {
	r := &IssueRecord{ID: "issue_1", Status: IssueLost, CollectedAt: nil}
	if r.IsOpen() {
		t.Fatalf("丢失待查记录不应被视为开放记录")
	}
	r2 := &IssueRecord{ID: "issue_2", Status: IssueOpen, CollectedAt: nil}
	if !r2.IsOpen() {
		t.Fatalf("正常发放记录应被视为开放记录")
	}
	r3 := &IssueRecord{ID: "issue_3", Status: IssueReturned, CollectedAt: nil}
	if r3.IsOpen() {
		t.Fatalf("已归还记录不应被视为开放记录")
	}
}
