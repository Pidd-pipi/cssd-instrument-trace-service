package service

import (
	"testing"

	"example.com/cssd-instrument-trace-service/domain"
)

func TestDashboardStats(t *testing.T) {
	svc := newTestServices(t)
	id := sterilizeAndGetReadyPack(t, svc, "PK-DASH")

	d := svc.Stats.Dashboard()
	if d.TotalPacks != 1 {
		t.Errorf("器械包总数 = %d, want 1", d.TotalPacks)
	}
	if d.ByStage[string(domain.StageSterilized)] != 1 {
		t.Errorf("已灭菌计数异常: %+v", d.ByStage)
	}
	if d.SterilizedAvailable != 1 {
		t.Errorf("可发放计数 = %d, want 1", d.SterilizedAvailable)
	}
	if _, err := svc.Issues.Issue(id, domain.IssueInput{Department: "普外科", OperatingRoom: "1号", Issuer: "护士"}, testActor); err != nil {
		t.Fatal(err)
	}
	d = svc.Stats.Dashboard()
	if d.TodayIssued != 1 {
		t.Errorf("今日发放计数 = %d, want 1", d.TodayIssued)
	}
	if d.ByStage[string(domain.StageIssued)] != 1 {
		t.Errorf("已发放环节计数异常: %+v", d.ByStage)
	}
}
