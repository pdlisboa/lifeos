package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

// TestGetDeltaNeverShowsWhatsMissing é P2: o Painel de Delta soma sessões,
// evidências e subidas de nível contra o próprio baseline — nunca "quanto
// falta". Marcos concluídos ficam 0 na Fatia 1 (§3: nada avança marco
// sozinho ainda).
func TestGetDeltaAggregatesRisesEvidenceAndSessions(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta com delta")
	detail, err := svc.GetGoal(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetGoal falhou: %v", err)
	}
	compID := detail.Competencies[0].ID

	if _, err := svc.SetCompetencyLevel(context.Background(), SetCompetencyLevelInput{
		UserID: userID, CompetencyID: compID, Level: 1, Rationale: "baseline",
	}); err != nil {
		t.Fatalf("SetCompetencyLevel (baseline) falhou: %v", err)
	}
	if _, err := svc.SetCompetencyLevel(context.Background(), SetCompetencyLevelInput{
		UserID: userID, CompetencyID: compID, Level: 3, Rationale: "subiu",
	}); err != nil {
		t.Fatalf("SetCompetencyLevel (subida) falhou: %v", err)
	}

	body := "evidência qualquer"
	if _, err := svc.CreateEvidence(context.Background(), CreateEvidenceInput{
		UserID: userID, GoalID: active.Goal.ID, Kind: domain.EvidenceCodeSnippet, Body: &body,
	}); err != nil {
		t.Fatalf("CreateEvidence falhou: %v", err)
	}

	if _, err := svc.CreateSession(context.Background(), CreateSessionInput{
		UserID: userID, GoalID: active.Goal.ID, StartedAt: time.Now(), DurationMin: 25,
	}); err != nil {
		t.Fatalf("CreateSession falhou: %v", err)
	}

	delta, err := svc.GetDelta(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetDelta falhou: %v", err)
	}
	if delta.EvidenceCount != 1 {
		t.Fatalf("evidenceCount = %d, want 1", delta.EvidenceCount)
	}
	if delta.SessionMinutes != 25 {
		t.Fatalf("sessionMinutes = %d, want 25", delta.SessionMinutes)
	}
	if delta.RisesLast90d[compID] != 1 {
		t.Fatalf("risesLast90d[%s] = %d, want 1", compID, delta.RisesLast90d[compID])
	}
	if delta.MilestonesDone != 0 {
		t.Fatalf("milestonesDone = %d, want 0 (Fatia 1 não avança marco sozinho)", delta.MilestonesDone)
	}
	if _, ok := delta.BaselineSetAt[compID]; !ok {
		t.Fatal("esperava baselineSetAt preenchido para a competência avaliada")
	}
}

func TestGetDeltaGoalNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.GetDelta(context.Background(), userID, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}

// TestGetConsistencyNeverStreak é RN-11/P6: janela móvel de 30 dias, nunca
// streak binária que zera.
func TestGetConsistencyCountsDistinctActiveDays(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta com consistência")

	if _, err := svc.CreateSession(context.Background(), CreateSessionInput{
		UserID: userID, GoalID: active.Goal.ID, StartedAt: time.Now(), DurationMin: 10,
	}); err != nil {
		t.Fatalf("CreateSession falhou: %v", err)
	}

	c, err := svc.GetConsistency(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetConsistency falhou: %v", err)
	}
	if c.WindowDays != 30 {
		t.Fatalf("windowDays = %d, want 30", c.WindowDays)
	}
	if c.ActiveDays != 1 {
		t.Fatalf("activeDays = %d, want 1", c.ActiveDays)
	}
	if !c.TodayDone {
		t.Fatal("sessão criada hoje deveria marcar todayDone")
	}
	if c.Label() == "" {
		t.Fatal("Label() não deveria ser vazio")
	}
}

func TestGetProjectionUnavailableWithoutHistory(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta sem ritmo ainda")

	p, err := svc.GetProjection(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetProjection falhou: %v", err)
	}
	if p.Available {
		t.Fatal("Available deveria ser false sem sessão nenhuma")
	}
	if p.Reason == nil || *p.Reason == "" {
		t.Fatal("Reason deveria explicar a ausência de dado")
	}
}

// TestGetProjectionAvailableWithThreeWeeksOfSessions cobre §7.2: com >= 3
// semanas de histórico, o ritmo real aparece — nunca uma previsão inventada
// de chegada ao marco.
func TestGetProjectionAvailableWithThreeWeeksOfSessions(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta com ritmo")

	now := time.Now()
	for _, daysAgo := range []int{22, 12, 2} {
		if _, err := svc.CreateSession(context.Background(), CreateSessionInput{
			UserID: userID, GoalID: active.Goal.ID, StartedAt: now.AddDate(0, 0, -daysAgo), DurationMin: 30,
		}); err != nil {
			t.Fatalf("CreateSession (%d dias atrás) falhou: %v", daysAgo, err)
		}
	}

	p, err := svc.GetProjection(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetProjection falhou: %v", err)
	}
	if !p.Available {
		t.Fatalf("Available deveria ser true com >= 3 semanas de histórico, reason=%v", p.Reason)
	}
	if p.MinutesPerWeek == nil || *p.MinutesPerWeek <= 0 {
		t.Fatalf("MinutesPerWeek = %v, want > 0", p.MinutesPerWeek)
	}
}

func TestGetProjectionGoalNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.GetProjection(context.Background(), userID, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}

func TestGetConsistencyGoalNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.GetConsistency(context.Background(), userID, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}
