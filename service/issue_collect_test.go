package service

import (
	"errors"
	"strings"
	"testing"

	"example.com/cssd-instrument-trace-service/domain"
)

// TestCollectLostIssueBlocked 丢失待查的发放记录不能通过回收接口闭环为已归还，
// 必须返回「丢失待查」拦截原因。
func TestCollectLostIssueBlocked(t *testing.T) {
	svc := newTestServices(t)
	pack := domain.NewPack(domain.RegisterPackInput{Barcode: "CL-LOST", Name: "丢失回收测试包", PackType: domain.TypeSurgical})
	pack.Stage = domain.StageInUse
	if err := svc.Store.SavePack(pack); err != nil {
		t.Fatalf("保存器械包失败: %v", err)
	}
	issue := domain.NewIssueRecord(pack, domain.IssueInput{
		Department: "手术一科", OperatingRoom: "1号手术间", Issuer: "测试员",
	}, "batch_1")
	if err := svc.Store.SaveIssue(issue); err != nil {
		t.Fatalf("保存发放记录失败: %v", err)
	}
	if err := svc.Store.UpdateIssue(issue.ID, func(r *domain.IssueRecord) error {
		r.MarkLost()
		return nil
	}); err != nil {
		t.Fatalf("标记丢失失败: %v", err)
	}

	_, err := svc.Issues.Collect(pack.ID, "回收员", testActor)
	if err == nil {
		t.Fatal("丢失待查记录不应被回收闭环")
	}
	var be *BlockedError
	if !errors.As(err, &be) {
		t.Fatalf("应返回拦截类错误，实际 %v", err)
	}
	if !strings.Contains(be.Reason, "丢失") {
		t.Fatalf("拦截原因应说明丢失待查，实际 %q", be.Reason)
	}
	// 发放记录必须仍为丢失状态，不能被改成已归还。
	got, err := svc.Store.GetIssue(issue.ID)
	if err != nil {
		t.Fatalf("读取发放记录失败: %v", err)
	}
	if got.Status != domain.IssueLost {
		t.Fatalf("丢失记录被错误闭环: %+v", got)
	}
}
