package app

import (
	"context"
	"strings"
	"testing"
)

// TestGetTodayAggregatesActiveGoals cobre a agregação de até 3 metas com sua
// ação pendente e a consistência agregada (03-api.md, endpoint /today).
func TestGetTodayAggregatesActiveGoals(t *testing.T) {
	svc, userID := newFixture(t)
	readyGoal(t, svc, userID, "Meta 1")
	readyGoal(t, svc, userID, "Meta 2")
	createDraftGoal(t, svc, userID, "Rascunho — não deveria aparecer")

	today, err := svc.GetToday(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetToday falhou: %v", err)
	}
	if len(today.Goals) != 2 {
		t.Fatalf("esperava 2 metas ativas em Hoje, got %d", len(today.Goals))
	}
	for _, g := range today.Goals {
		if g.Action == nil {
			t.Fatalf("meta %s sem ação pendente em Hoje (RN-03 quebrada)", g.Goal.ID)
		}
	}
	if today.Consistency.WindowDays != 30 {
		t.Fatalf("windowDays = %d, want 30", today.Consistency.WindowDays)
	}
}

func TestGetTodayNoActiveGoals(t *testing.T) {
	svc, userID := newFixture(t)
	createDraftGoal(t, svc, userID, "Só rascunho")

	today, err := svc.GetToday(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetToday falhou: %v", err)
	}
	if len(today.Goals) != 0 {
		t.Fatalf("esperava 0 metas em Hoje sem nenhuma ativa, got %d", len(today.Goals))
	}
}

// TestGetTodayRecentWin cobre o cálculo do ganho recente (05-ux.md §3): só
// conta subida de nível de fato (fromLevel < toLevel), nunca a primeira
// medição (fromLevel nil é o baseline, não um ganho).
func TestGetTodayRecentWin(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta com evolução")
	detail, err := svc.GetGoal(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetGoal falhou: %v", err)
	}
	compID := detail.Competencies[0].ID

	if _, err := svc.SetCompetencyLevel(context.Background(), SetCompetencyLevelInput{
		UserID: userID, CompetencyID: compID, Level: 2, Rationale: "baseline inicial",
	}); err != nil {
		t.Fatalf("primeiro SetCompetencyLevel falhou: %v", err)
	}

	today, err := svc.GetToday(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetToday falhou: %v", err)
	}
	if today.Goals[0].RecentWin != nil {
		t.Fatalf("baseline sozinho não deveria contar como ganho, got %v", *today.Goals[0].RecentWin)
	}

	if _, err := svc.SetCompetencyLevel(context.Background(), SetCompetencyLevelInput{
		UserID: userID, CompetencyID: compID, Level: 4, Rationale: "evoluí de verdade",
	}); err != nil {
		t.Fatalf("segundo SetCompetencyLevel falhou: %v", err)
	}

	today2, err := svc.GetToday(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetToday (2) falhou: %v", err)
	}
	win := today2.Goals[0].RecentWin
	if win == nil {
		t.Fatal("esperava recentWin depois de uma subida real de nível")
	}
	if !strings.Contains(*win, "de 2 para 4") {
		t.Fatalf("recentWin = %q, want conter \"de 2 para 4\"", *win)
	}
}
