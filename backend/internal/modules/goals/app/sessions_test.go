package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
)

// TestCreateSessionComputesLocalOnFromTimezone cobre a armadilha do
// CLAUDE.md: localOn é calculado no servidor a partir do timezone do
// usuário, nunca no navegador.
func TestCreateSessionComputesLocalOnFromTimezone(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")

	// 2026-01-01 01:00 UTC é, em UTC-3 (America/Sao_Paulo), ainda 2025-12-31.
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatalf("LoadLocation falhou: %v", err)
	}
	startedAt := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	sess, err := svc.CreateSession(context.Background(), CreateSessionInput{
		UserID:      userID,
		GoalID:      active.Goal.ID,
		StartedAt:   startedAt,
		DurationMin: 20,
		Timezone:    loc,
	})
	if err != nil {
		t.Fatalf("CreateSession falhou: %v", err)
	}
	if got := sess.LocalOn.Format("2006-01-02"); got != "2025-12-31" {
		t.Fatalf("localOn = %s, want 2025-12-31 (calculado em America/Sao_Paulo)", got)
	}

	list, err := svc.ListSessions(context.Background(), userID, active.Goal.ID, 10)
	if err != nil {
		t.Fatalf("ListSessions falhou: %v", err)
	}
	if len(list) != 1 || list[0].ID != sess.ID {
		t.Fatalf("ListSessions não devolveu a sessão criada: %+v", list)
	}

	updated, err := svc.GetGoal(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetGoal falhou: %v", err)
	}
	if updated.Goal.LastActivityAt == nil {
		t.Fatal("criar sessão deveria atualizar LastActivityAt da meta")
	}
}

func TestCreateSessionInvalidDuration(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")

	_, err := svc.CreateSession(context.Background(), CreateSessionInput{
		UserID: userID, GoalID: active.Goal.ID, StartedAt: time.Now(), DurationMin: 0,
	})
	if err == nil {
		t.Fatal("esperava erro para duração fora de 1-600 minutos")
	}
}

func TestCreateSessionGoalNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.CreateSession(context.Background(), CreateSessionInput{
		UserID: userID, GoalID: "00000000-0000-0000-0000-000000000000", StartedAt: time.Now(), DurationMin: 10,
	})
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}

func TestListSessionsGoalNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.ListSessions(context.Background(), userID, "00000000-0000-0000-0000-000000000000", 10)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}
