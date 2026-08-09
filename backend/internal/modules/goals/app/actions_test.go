package app

import (
	"context"
	"errors"
	"testing"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func TestGetPendingActionRequiresOwnedGoal(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")

	if _, err := svc.GetPendingAction(context.Background(), "00000000-0000-0000-0000-000000000000", active.Goal.ID); !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound para usuário errado, got %v", err)
	}

	a, err := svc.GetPendingAction(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetPendingAction falhou: %v", err)
	}
	if a.ID != active.Action.ID {
		t.Fatalf("ação pendente = %s, want %s", a.ID, active.Action.ID)
	}
}

// TestCreateUserActionReplacesPending cobre o caminho manual de RN-03: você
// pode sempre escrever sua própria ação, e ela substitui a pendente
// (que vira skipped/other) em vez de duplicar.
func TestCreateUserActionReplacesPending(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")
	previousActionID := active.Action.ID

	created, err := svc.CreateUserAction(context.Background(), CreateUserActionInput{
		UserID: userID,
		GoalID: active.Goal.ID,
		Title:  "Minha própria ação de 20 minutos",
	})
	if err != nil {
		t.Fatalf("CreateUserAction falhou: %v", err)
	}
	if created.GeneratedBy != domain.GeneratedByUser {
		t.Fatalf("generatedBy = %s, want user", created.GeneratedBy)
	}

	pending, err := svc.GetPendingAction(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetPendingAction falhou: %v", err)
	}
	if pending.ID != created.ID {
		t.Fatalf("ação pendente = %s, want a criada manualmente (%s)", pending.ID, created.ID)
	}
	if pending.ID == previousActionID {
		t.Fatal("ação anterior deveria ter sido substituída, não continuar pendente")
	}
}

// TestCompleteActionRN03AlwaysGeneratesNext é o núcleo de RN-03: completar a
// ação atual nunca deixa a meta sem uma próxima, na mesma transação.
func TestCompleteActionRN03AlwaysGeneratesNext(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")
	firstActionID := active.Action.ID

	durationMin := 20
	result, err := svc.CompleteAction(context.Background(), CompleteActionInput{
		UserID:      userID,
		ActionID:    firstActionID,
		DurationMin: &durationMin,
	})
	if err != nil {
		t.Fatalf("CompleteAction falhou: %v", err)
	}
	if result.Completed.Status != domain.ActionCompleted {
		t.Fatalf("status da ação concluída = %s, want completed", result.Completed.Status)
	}
	if result.Next == nil {
		t.Fatal("RN-03: CompleteAction deveria sempre gerar a próxima ação")
	}
	if result.Next.ID == firstActionID {
		t.Fatal("a próxima ação deveria ser uma linha nova, não a mesma")
	}
	if result.Session == nil {
		t.Fatal("informar durationMin deveria criar uma sessão")
	}

	// documenta o comportamento conhecido (fatia-1-implementacao.md §3): sem
	// avanço de marco, o fallback repete o mesmo marco e a mesma competência.
	if result.Next.MilestoneID == nil || active.Action.MilestoneID == nil || *result.Next.MilestoneID != *active.Action.MilestoneID {
		t.Fatalf("fallback deveria repetir o mesmo marco (comportamento documentado), got next=%v first=%v", result.Next.MilestoneID, active.Action.MilestoneID)
	}

	pending, err := svc.GetPendingAction(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetPendingAction falhou: %v", err)
	}
	if pending.ID != result.Next.ID {
		t.Fatalf("ação pendente persistida (%s) difere da devolvida por CompleteAction (%s)", pending.ID, result.Next.ID)
	}

	updated, err := svc.GetGoal(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetGoal falhou: %v", err)
	}
	if updated.Goal.LastActivityAt == nil {
		t.Fatal("completar uma ação deveria atualizar LastActivityAt da meta")
	}
}

func TestCompleteActionWithoutDurationSkipsSession(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")

	result, err := svc.CompleteAction(context.Background(), CompleteActionInput{
		UserID:   userID,
		ActionID: active.Action.ID,
	})
	if err != nil {
		t.Fatalf("CompleteAction falhou: %v", err)
	}
	if result.Session != nil {
		t.Fatal("sem durationMin, não deveria criar sessão")
	}
}

func TestCompleteActionAlreadyResolvedFails(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")

	if _, err := svc.CompleteAction(context.Background(), CompleteActionInput{UserID: userID, ActionID: active.Action.ID}); err != nil {
		t.Fatalf("primeira conclusão falhou: %v", err)
	}
	if _, err := svc.CompleteAction(context.Background(), CompleteActionInput{UserID: userID, ActionID: active.Action.ID}); err == nil {
		t.Fatal("completar uma ação já resolvida de novo deveria falhar")
	}
}

// TestSkipActionCarriesDifficultyHint cobre §7.3: o motivo do skip vira
// difficultyHint direto na próxima ação gerada.
func TestSkipActionCarriesDifficultyHint(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")

	result, err := svc.SkipAction(context.Background(), SkipActionInput{
		UserID:   userID,
		ActionID: active.Action.ID,
		Reason:   domain.SkipTooHard,
	})
	if err != nil {
		t.Fatalf("SkipAction falhou: %v", err)
	}
	if result.Next == nil {
		t.Fatal("SkipAction deveria gerar a próxima ação (RN-03)")
	}
	if result.Next.DifficultyHint == nil || *result.Next.DifficultyHint != "easier" {
		t.Fatalf("difficultyHint = %v, want easier (too_hard -> easier)", result.Next.DifficultyHint)
	}
}

func TestCreateUserActionGoalNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.CreateUserAction(context.Background(), CreateUserActionInput{
		UserID: userID, GoalID: "00000000-0000-0000-0000-000000000000", Title: "Ação qualquer",
	})
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}

// TestCreateUserActionOnGoalWithoutPendingAction cobre o outro ramo de RN-03
// dentro de CreateUserAction: quando não há nenhuma ação pendente ainda
// (meta em rascunho, nunca ativada), o caminho de "achar e substituir" cai
// direto em ErrNotFound e segue para criar a primeira ação mesmo assim.
func TestCreateUserActionOnGoalWithoutPendingAction(t *testing.T) {
	svc, userID := newFixture(t)
	created := createDraftGoal(t, svc, userID, "Meta em rascunho")

	a, err := svc.CreateUserAction(context.Background(), CreateUserActionInput{
		UserID: userID, GoalID: created.Goal.ID, Title: "Primeira ação manual",
	})
	if err != nil {
		t.Fatalf("CreateUserAction sem ação pendente prévia falhou: %v", err)
	}
	if a.Status != domain.ActionPending {
		t.Fatalf("status = %s, want pending", a.Status)
	}
}

func TestCompleteActionNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.CompleteAction(context.Background(), CompleteActionInput{
		UserID: userID, ActionID: "00000000-0000-0000-0000-000000000000",
	})
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}

func TestSkipActionNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.SkipAction(context.Background(), SkipActionInput{
		UserID: userID, ActionID: "00000000-0000-0000-0000-000000000000", Reason: domain.SkipOther,
	})
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}

func TestSkipActionInvalidReason(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")

	_, err := svc.SkipAction(context.Background(), SkipActionInput{
		UserID:   userID,
		ActionID: active.Action.ID,
		Reason:   domain.SkipReason("nao_existe"),
	})
	if err == nil {
		t.Fatal("esperava erro para motivo de skip inválido")
	}
}
