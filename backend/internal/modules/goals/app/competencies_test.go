package app

import (
	"context"
	"errors"
	"testing"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
)

// TestSetCompetencyLevelRN04ManualPath cobre o caminho manual de RN-04: sem
// agente, a confirmação explícita (source: self) já muda o nível sozinha, e
// congela o baseline na primeira vez.
func TestSetCompetencyLevelRN04ManualPath(t *testing.T) {
	svc, userID := newFixture(t)
	created := createDraftGoal(t, svc, userID, "Meta com competências")
	compID := created.Competencies[0].ID

	updated, err := svc.SetCompetencyLevel(context.Background(), SetCompetencyLevelInput{
		UserID:       userID,
		CompetencyID: compID,
		Level:        2,
		Rationale:    "Consigo fazer isso sem consultar exemplos",
	})
	if err != nil {
		t.Fatalf("SetCompetencyLevel falhou: %v", err)
	}
	if updated.CurrentLevel == nil || *updated.CurrentLevel != 2 {
		t.Fatalf("nível atual = %v, want 2", updated.CurrentLevel)
	}
	if updated.BaselineLevel == nil || *updated.BaselineLevel != 2 {
		t.Fatalf("baseline = %v, want 2 (primeira medição)", updated.BaselineLevel)
	}

	// segunda medição: baseline não deve mudar, current sim.
	updated2, err := svc.SetCompetencyLevel(context.Background(), SetCompetencyLevelInput{
		UserID:       userID,
		CompetencyID: compID,
		Level:        4,
		Rationale:    "Evoluí bastante nas últimas semanas",
	})
	if err != nil {
		t.Fatalf("segundo SetCompetencyLevel falhou: %v", err)
	}
	if updated2.CurrentLevel == nil || *updated2.CurrentLevel != 4 {
		t.Fatalf("nível atual = %v, want 4", updated2.CurrentLevel)
	}
	if updated2.BaselineLevel == nil || *updated2.BaselineLevel != 2 {
		t.Fatalf("baseline deveria continuar 2, got %v", updated2.BaselineLevel)
	}

	history, err := svc.CompetencyHistory(context.Background(), userID, compID)
	if err != nil {
		t.Fatalf("CompetencyHistory falhou: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("esperava 2 eventos de nível, got %d", len(history))
	}
}

func TestSetCompetencyLevelInvalidLevel(t *testing.T) {
	svc, userID := newFixture(t)
	created := createDraftGoal(t, svc, userID, "Meta com competências")

	_, err := svc.SetCompetencyLevel(context.Background(), SetCompetencyLevelInput{
		UserID:       userID,
		CompetencyID: created.Competencies[0].ID,
		Level:        9,
		Rationale:    "nível fora da escala",
	})
	if err == nil {
		t.Fatal("esperava erro para nível fora de 0-5")
	}
}

func TestSetCompetencyLevelNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.SetCompetencyLevel(context.Background(), SetCompetencyLevelInput{
		UserID:       userID,
		CompetencyID: "00000000-0000-0000-0000-000000000000",
		Level:        2,
		Rationale:    "qualquer",
	})
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}

func TestCompetencyHistoryRequiresOwnedCompetency(t *testing.T) {
	svc, userID := newFixture(t)
	created := createDraftGoal(t, svc, userID, "Meta com competências")

	other := "00000000-0000-0000-0000-000000000000"
	_, err := svc.CompetencyHistory(context.Background(), userID, other)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}

	history, err := svc.CompetencyHistory(context.Background(), userID, created.Competencies[0].ID)
	if err != nil {
		t.Fatalf("CompetencyHistory falhou: %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("esperava histórico vazio antes de qualquer avaliação, got %d", len(history))
	}
}
