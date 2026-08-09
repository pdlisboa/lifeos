package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func newAction(t *testing.T, q Querier, g *domain.Goal, title string) *domain.NextAction {
	t.Helper()
	a, err := domain.NewNextAction(mustID(t), g.ID, g.UserID, title, 20,
		"Versão de 5 min: releia e escreva só o primeiro passo, sem terminar.", domain.GeneratedByUser, time.Now())
	if err != nil {
		t.Fatalf("domain.NewNextAction: %v", err)
	}
	if err := InsertNextAction(context.Background(), q, a); err != nil {
		t.Fatalf("InsertNextAction: %v", err)
	}
	return a
}

func TestInsertAndGetActionRoundTrip(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	a := newAction(t, q, g, "Escrever um worker pool")

	fetched, err := GetAction(context.Background(), q, userID, a.ID)
	if err != nil {
		t.Fatalf("GetAction: %v", err)
	}
	if fetched.Title != a.Title || fetched.EstimatedMin != 20 || fetched.Status != domain.ActionPending {
		t.Fatalf("round-trip não bate: %+v", fetched)
	}

	pending, err := GetPendingAction(context.Background(), q, g.ID)
	if err != nil {
		t.Fatalf("GetPendingAction: %v", err)
	}
	if pending.ID != a.ID {
		t.Fatalf("pending.ID = %s, want %s", pending.ID, a.ID)
	}
}

func TestGetActionNotFoundOrWrongUser(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	a := newAction(t, q, g, "Ação qualquer")

	if _, err := GetAction(context.Background(), q, userID, mustID(t)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("erro = %v, want ErrNotFound", err)
	}
	other := "00000000-0000-0000-0000-000000000000"
	if _, err := GetAction(context.Background(), q, other, a.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("usuário errado: erro = %v, want ErrNotFound", err)
	}
}

func TestGetPendingActionNotFoundWhenNoneOpen(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	if _, err := GetPendingAction(context.Background(), q, g.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("erro = %v, want ErrNotFound sem ação nenhuma", err)
	}
}

// TestResolveNextActionPersistsCompletion cobre a metade de persistência de
// RN-03: resolver a ação (completed/skipped) grava status e motivo.
func TestResolveNextActionPersistsCompletion(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	a := newAction(t, q, g, "Ação a completar")

	now := time.Now()
	if err := a.Complete(now); err != nil {
		t.Fatalf("a.Complete: %v", err)
	}
	if err := ResolveNextAction(context.Background(), q, a); err != nil {
		t.Fatalf("ResolveNextAction: %v", err)
	}

	fetched, err := GetAction(context.Background(), q, userID, a.ID)
	if err != nil {
		t.Fatalf("GetAction: %v", err)
	}
	if fetched.Status != domain.ActionCompleted || fetched.ResolvedAt == nil {
		t.Fatalf("ação resolvida não persistiu corretamente: %+v", fetched)
	}

	// depois de resolvida, não é mais "a" pendente.
	if _, err := GetPendingAction(context.Background(), q, g.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("esperava ErrNotFound (nenhuma pendente) após resolver, got %v", err)
	}
}

func TestResolveNextActionSkippedPersistsReason(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	a := newAction(t, q, g, "Ação a pular")

	if err := a.Skip(domain.SkipTooHard, time.Now()); err != nil {
		t.Fatalf("a.Skip: %v", err)
	}
	if err := ResolveNextAction(context.Background(), q, a); err != nil {
		t.Fatalf("ResolveNextAction: %v", err)
	}

	fetched, err := GetAction(context.Background(), q, userID, a.ID)
	if err != nil {
		t.Fatalf("GetAction: %v", err)
	}
	if fetched.Status != domain.ActionSkipped || fetched.SkipReason == nil || *fetched.SkipReason != domain.SkipTooHard {
		t.Fatalf("skip não persistiu corretamente: %+v", fetched)
	}
}
