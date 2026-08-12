package domain

import (
	"testing"
	"time"
)

func TestBuildFallbackTrack_OrdersByTypicalLevelAndMapsCompetencies(t *testing.T) {
	specs := []MilestoneSpec{
		{Title: "Marco avançado", CompletionCriteria: "critério A", TypicalLevel: 3, CompetencyKeys: []string{"concurrency"}},
		{Title: "Marco inicial", CompletionCriteria: "critério B", TypicalLevel: 1, CompetencyKeys: []string{"testing", "unknown-key"}},
	}
	byKey := map[string]string{"concurrency": "comp-1", "testing": "comp-2"}

	milestones := BuildFallbackTrack(specs, byKey)

	if len(milestones) != 2 {
		t.Fatalf("esperava 2 marcos, teve %d", len(milestones))
	}
	if milestones[0].Title != "Marco inicial" {
		t.Errorf("primeiro marco = %q, esperava o de menor typical_level", milestones[0].Title)
	}
	if milestones[0].Status != MilestoneCurrent {
		t.Errorf("primeiro marco status = %s, esperava current", milestones[0].Status)
	}
	if milestones[1].Status != MilestoneLocked {
		t.Errorf("segundo marco status = %s, esperava locked", milestones[1].Status)
	}
	if got := milestones[0].CompetencyIDs; len(got) != 1 || got[0] != "comp-2" {
		t.Errorf("competencyIDs do primeiro marco = %v, esperava só [comp-2] (chave desconhecida ignorada)", got)
	}
	for i, m := range milestones {
		if m.Ordinal != i+1 {
			t.Errorf("marco %d: ordinal = %d, esperava %d", i, m.Ordinal, i+1)
		}
	}
}

func TestApplyTrackStatuses_SkipsCompletedAndSkipped(t *testing.T) {
	milestones := []Milestone{
		{Title: "concluído", Status: MilestoneCompleted},
		{Title: "pulado", Status: MilestoneSkipped},
		{Title: "próximo", Status: MilestoneLocked},
		{Title: "depois", Status: MilestoneLocked},
	}

	ApplyTrackStatuses(milestones)

	if milestones[0].Status != MilestoneCompleted {
		t.Errorf("marco concluído não deveria mudar de status, ficou %s", milestones[0].Status)
	}
	if milestones[1].Status != MilestoneSkipped {
		t.Errorf("marco pulado não deveria mudar de status, ficou %s", milestones[1].Status)
	}
	if milestones[2].Status != MilestoneCurrent {
		t.Errorf("primeiro marco aberto deveria virar current, ficou %s", milestones[2].Status)
	}
	if milestones[3].Status != MilestoneLocked {
		t.Errorf("marco seguinte deveria continuar locked, ficou %s", milestones[3].Status)
	}
}

func TestApplyTrackStatuses_AllCompleted(t *testing.T) {
	milestones := []Milestone{
		{Title: "a", Status: MilestoneCompleted},
		{Title: "b", Status: MilestoneCompleted},
	}
	ApplyTrackStatuses(milestones)
	for _, m := range milestones {
		if m.Status != MilestoneCompleted {
			t.Errorf("marco %s deveria continuar completed, ficou %s", m.Title, m.Status)
		}
	}
}

func TestFirstOpen(t *testing.T) {
	milestones := []Milestone{
		{Title: "concluído", Status: MilestoneCompleted},
		{Title: "atual", Status: MilestoneCurrent},
		{Title: "bloqueado", Status: MilestoneLocked},
	}
	m := FirstOpen(milestones)
	if m == nil || m.Title != "atual" {
		t.Fatalf("FirstOpen = %v, esperava o marco 'atual'", m)
	}
}

func TestFirstOpen_NoneOpen(t *testing.T) {
	milestones := []Milestone{{Title: "concluído", Status: MilestoneCompleted}}
	if m := FirstOpen(milestones); m != nil {
		t.Errorf("esperava nil quando todos os marcos estão concluídos, teve %v", m)
	}
}

func TestMilestone_Complete(t *testing.T) {
	m := &Milestone{Status: MilestoneCurrent}
	if err := m.Complete(time.Now()); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if m.Status != MilestoneCompleted {
		t.Errorf("status = %s, esperava completed", m.Status)
	}
	if m.CompletedAt == nil {
		t.Error("completedAt não foi gravado")
	}
}

func TestMilestone_Complete_RejectsNonCurrent(t *testing.T) {
	m := &Milestone{Status: MilestoneLocked}
	if err := m.Complete(time.Now()); err == nil {
		t.Fatal("esperava erro ao concluir marco que não está current")
	}
}
