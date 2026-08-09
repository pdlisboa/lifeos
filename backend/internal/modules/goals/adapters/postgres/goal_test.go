package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func TestInsertAndGetGoalRoundTrip(t *testing.T) {
	q, userID := setup(t)
	why := "porque sim"
	g, err := domain.NewGoal(mustID(t), userID, "Aprender Go", domain.ArchetypeSkill, "golang", &why, time.Now())
	if err != nil {
		t.Fatalf("domain.NewGoal: %v", err)
	}
	if err := InsertGoal(context.Background(), q, g); err != nil {
		t.Fatalf("InsertGoal: %v", err)
	}

	fetched, err := GetGoal(context.Background(), q, userID, g.ID)
	if err != nil {
		t.Fatalf("GetGoal: %v", err)
	}
	if fetched.Title != g.Title || fetched.PackID != "golang" || fetched.Why == nil || *fetched.Why != why {
		t.Fatalf("round-trip não bate: %+v", fetched)
	}
	if fetched.Status != domain.GoalDraft {
		t.Fatalf("status = %s, want draft", fetched.Status)
	}
}

func TestGetGoalNotFoundOrWrongUser(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Minha meta")

	if _, err := GetGoal(context.Background(), q, userID, mustID(t)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("id inexistente: erro = %v, want ErrNotFound", err)
	}

	otherUser := "00000000-0000-0000-0000-000000000000"
	if _, err := GetGoal(context.Background(), q, otherUser, g.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("usuário errado: erro = %v, want ErrNotFound (isolamento por dono)", err)
	}
}

func TestUpdateGoalPersistsAndReturnsNotFoundIfMissing(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Título original")

	g.Title = "Título atualizado"
	g.UpdatedAt = time.Now()
	if err := UpdateGoal(context.Background(), q, g); err != nil {
		t.Fatalf("UpdateGoal: %v", err)
	}
	fetched, err := GetGoal(context.Background(), q, userID, g.ID)
	if err != nil {
		t.Fatalf("GetGoal: %v", err)
	}
	if fetched.Title != "Título atualizado" {
		t.Fatalf("title = %q, want atualizado", fetched.Title)
	}

	ghost, _ := domain.NewGoal(mustID(t), userID, "Fantasma", domain.ArchetypeSkill, "golang", nil, time.Now())
	if err := UpdateGoal(context.Background(), q, ghost); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateGoal de meta inexistente: erro = %v, want ErrNotFound", err)
	}
}

func TestListGoalsAndByStatus(t *testing.T) {
	q, userID := setup(t)
	g1 := newGoal(t, q, userID, "Meta 1")
	g2 := newGoal(t, q, userID, "Meta 2")

	all, err := ListGoals(context.Background(), q, userID, nil)
	if err != nil {
		t.Fatalf("ListGoals: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("esperava 2 metas, got %d", len(all))
	}

	// promove g1 para completed sem passar por Activate — só queremos exercitar
	// o filtro de status na consulta, não as regras de transição do domínio.
	g1.Status = domain.GoalCompleted
	now := time.Now()
	g1.DefinitionOfDone = strPtrTest("qualquer")
	g1.ClosedAt = &now
	if err := UpdateGoal(context.Background(), q, g1); err != nil {
		t.Fatalf("UpdateGoal: %v", err)
	}

	drafts, err := ListGoals(context.Background(), q, userID, []string{"draft"})
	if err != nil {
		t.Fatalf("ListGoals(draft): %v", err)
	}
	if len(drafts) != 1 || drafts[0].ID != g2.ID {
		t.Fatalf("esperava só g2 em draft, got %+v", drafts)
	}

	completed, err := ListGoals(context.Background(), q, userID, []string{"completed"})
	if err != nil {
		t.Fatalf("ListGoals(completed): %v", err)
	}
	if len(completed) != 1 || completed[0].ID != g1.ID {
		t.Fatalf("esperava só g1 em completed, got %+v", completed)
	}
}

func TestCountActiveGoalsAndLockUser(t *testing.T) {
	q, userID := setup(t)
	if n, err := CountActiveGoals(context.Background(), q, userID); err != nil || n != 0 {
		t.Fatalf("count inicial = %d, err=%v, want 0", n, err)
	}

	g := newGoal(t, q, userID, "Meta a ativar")
	g.Status = domain.GoalActive
	g.DefinitionOfDone = strPtrTest("critério qualquer")
	activated := time.Now()
	g.ActivatedAt = &activated
	if err := UpdateGoal(context.Background(), q, g); err != nil {
		t.Fatalf("UpdateGoal: %v", err)
	}

	n, err := CountActiveGoals(context.Background(), q, userID)
	if err != nil {
		t.Fatalf("CountActiveGoals: %v", err)
	}
	if n != 1 {
		t.Fatalf("count = %d, want 1", n)
	}

	if err := LockUser(context.Background(), q, userID); err != nil {
		t.Fatalf("LockUser: %v", err)
	}
}

func strPtrTest(s string) *string { return &s }
