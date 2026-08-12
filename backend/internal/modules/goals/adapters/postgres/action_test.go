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

// TestUpdatePendingActionContent cobre o "upgrade no lugar" do fallback
// determinístico pelo resultado do A2 (04-agentes.md §5): mesmo ID, conteúdo
// novo, generated_by passa a 'agent'.
func TestUpdatePendingActionContent(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	a := newAction(t, q, g, "Ação de fallback")

	updated := *a
	updated.Title = "Escreva um worker pool que lê 3 URLs em paralelo"
	updated.EstimatedMin = 25
	updated.MinimalVariant = "Versão de 5 min: escreva só a assinatura da função."
	updated.GeneratedBy = domain.GeneratedByAgent

	ok, err := UpdatePendingActionContent(context.Background(), q, &updated)
	if err != nil {
		t.Fatalf("UpdatePendingActionContent: %v", err)
	}
	if !ok {
		t.Fatal("esperava true — a ação ainda estava pending")
	}

	fetched, err := GetAction(context.Background(), q, userID, a.ID)
	if err != nil {
		t.Fatalf("GetAction: %v", err)
	}
	if fetched.Title != updated.Title || fetched.EstimatedMin != 25 || fetched.GeneratedBy != domain.GeneratedByAgent {
		t.Fatalf("conteúdo não foi trocado: %+v", fetched)
	}
	if fetched.Status != domain.ActionPending {
		t.Fatalf("status mudou inesperadamente: %s", fetched.Status)
	}
}

// TestUpdatePendingActionContent_NoOpWhenAlreadyResolved cobre a defesa
// contra corrida: se a ação já foi concluída/pulada enquanto o job do A2
// rodava, o resultado do agente é descartado, sem sobrescrever RN-03.
func TestUpdatePendingActionContent_NoOpWhenAlreadyResolved(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	a := newAction(t, q, g, "Ação já resolvida")

	if err := a.Complete(time.Now()); err != nil {
		t.Fatalf("a.Complete: %v", err)
	}
	if err := ResolveNextAction(context.Background(), q, a); err != nil {
		t.Fatalf("ResolveNextAction: %v", err)
	}

	updated := *a
	updated.Title = "Não deveria aparecer"
	ok, err := UpdatePendingActionContent(context.Background(), q, &updated)
	if err != nil {
		t.Fatalf("UpdatePendingActionContent: %v", err)
	}
	if ok {
		t.Fatal("esperava false — a ação já não estava mais pending")
	}

	fetched, err := GetAction(context.Background(), q, userID, a.ID)
	if err != nil {
		t.Fatalf("GetAction: %v", err)
	}
	if fetched.Title == updated.Title {
		t.Fatal("o conteúdo original foi sobrescrito indevidamente")
	}
}

func TestListRecentActionsByGoal(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")

	skip := newAction(t, q, g, "Ação pulada")
	if err := skip.Skip(domain.SkipTooHard, time.Now()); err != nil {
		t.Fatalf("Skip: %v", err)
	}
	if err := ResolveNextAction(context.Background(), q, skip); err != nil {
		t.Fatalf("ResolveNextAction: %v", err)
	}

	done := newAction(t, q, g, "Ação concluída")
	if err := done.Complete(time.Now()); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if err := ResolveNextAction(context.Background(), q, done); err != nil {
		t.Fatalf("ResolveNextAction: %v", err)
	}

	// a pendente atual nunca deve aparecer na lista de recentes.
	_ = newAction(t, q, g, "Ação ainda pendente")

	recent, err := ListRecentActionsByGoal(context.Background(), q, g.ID, 10)
	if err != nil {
		t.Fatalf("ListRecentActionsByGoal: %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("esperava 2 ações recentes (só as resolvidas), teve %d: %+v", len(recent), recent)
	}
	for _, r := range recent {
		if r.Title == "Ação ainda pendente" {
			t.Fatal("ação pendente não deveria aparecer no histórico recente")
		}
	}

	var skipped *RecentAction
	for i := range recent {
		if recent[i].Status == domain.ActionSkipped {
			skipped = &recent[i]
		}
	}
	if skipped == nil || skipped.SkipReason == nil || *skipped.SkipReason != domain.SkipTooHard {
		t.Fatalf("motivo de skip não veio junto: %+v", recent)
	}
}
