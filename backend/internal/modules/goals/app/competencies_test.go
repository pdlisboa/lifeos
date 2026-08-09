package app

import (
	"context"
	"errors"
	"testing"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
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

// TestSetCompetencyLevelWithEvidenceLinksTheEvent cobre §5.3 da UX: o
// gráfico temporal precisa poder linkar o ponto à evidência que sustentou a
// mudança.
func TestSetCompetencyLevelWithEvidenceLinksTheEvent(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta com evidência")
	detail, err := svc.GetGoal(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetGoal falhou: %v", err)
	}
	compID := detail.Competencies[0].ID

	body := "func main() {}"
	ev, err := svc.CreateEvidence(context.Background(), CreateEvidenceInput{
		UserID: userID, GoalID: active.Goal.ID, Kind: domain.EvidenceCodeSnippet, Body: &body,
	})
	if err != nil {
		t.Fatalf("CreateEvidence falhou: %v", err)
	}

	_, err = svc.SetCompetencyLevel(context.Background(), SetCompetencyLevelInput{
		UserID: userID, CompetencyID: compID, Level: 3, Rationale: "essa evidência mostra nível 3",
		EvidenceID: &ev.ID,
	})
	if err != nil {
		t.Fatalf("SetCompetencyLevel falhou: %v", err)
	}

	history, err := svc.CompetencyHistory(context.Background(), userID, compID)
	if err != nil {
		t.Fatalf("CompetencyHistory falhou: %v", err)
	}
	if len(history) != 1 || history[0].EvidenceID == nil || *history[0].EvidenceID != ev.ID {
		t.Fatalf("histórico = %+v, esperava 1 evento linkado a %s", history, ev.ID)
	}
}

func TestSetCompetencyLevelWithEvidenceFromOtherGoalFails(t *testing.T) {
	svc, userID := newFixture(t)
	a := readyGoal(t, svc, userID, "Meta A")
	b := createDraftGoal(t, svc, userID, "Meta B")

	body := "func main() {}"
	ev, err := svc.CreateEvidence(context.Background(), CreateEvidenceInput{
		UserID: userID, GoalID: a.Goal.ID, Kind: domain.EvidenceCodeSnippet, Body: &body,
	})
	if err != nil {
		t.Fatalf("CreateEvidence falhou: %v", err)
	}

	_, err = svc.SetCompetencyLevel(context.Background(), SetCompetencyLevelInput{
		UserID: userID, CompetencyID: b.Competencies[0].ID, Level: 3, Rationale: "evidência de outra meta",
		EvidenceID: &ev.ID,
	})
	if err == nil {
		t.Fatal("evidência de outra meta deveria ser rejeitada")
	}
}

func TestSetCompetencyLevelWithUnknownEvidenceFails(t *testing.T) {
	svc, userID := newFixture(t)
	created := createDraftGoal(t, svc, userID, "Meta com competências")
	unknown := "00000000-0000-0000-0000-000000000000"

	_, err := svc.SetCompetencyLevel(context.Background(), SetCompetencyLevelInput{
		UserID: userID, CompetencyID: created.Competencies[0].ID, Level: 3, Rationale: "evidência inexistente",
		EvidenceID: &unknown,
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
