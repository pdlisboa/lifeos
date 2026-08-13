package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func TestUpsertAndGetEvalCaseRoundTrip(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	c := newCompetency(t, q, g, "concurrency", "Concorrência")
	e := newEvidence(t, q, g, "func worker() {}")

	ec, err := domain.NewEvalCase(e.ID, "exemplo claro de nível 2", []domain.EvalCaseScore{{CompetencyID: c.ID, Level: 2}})
	if err != nil {
		t.Fatalf("domain.NewEvalCase: %v", err)
	}
	if err := UpsertEvalCase(context.Background(), q, ec, time.Now()); err != nil {
		t.Fatalf("UpsertEvalCase: %v", err)
	}

	fetched, err := GetEvalCase(context.Background(), q, e.ID)
	if err != nil {
		t.Fatalf("GetEvalCase: %v", err)
	}
	if fetched == nil {
		t.Fatal("GetEvalCase = nil, want caso marcado")
	}
	if fetched.Note != "exemplo claro de nível 2" {
		t.Fatalf("note = %q, want %q", fetched.Note, "exemplo claro de nível 2")
	}
	if len(fetched.Scores) != 1 || fetched.Scores[0].CompetencyID != c.ID || fetched.Scores[0].Level != 2 {
		t.Fatalf("scores = %+v, want [{%s 2}]", fetched.Scores, c.ID)
	}
}

func TestGetEvalCaseNotMarkedReturnsNilWithoutError(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	e := newEvidence(t, q, g, "sem marcação")

	fetched, err := GetEvalCase(context.Background(), q, e.ID)
	if err != nil {
		t.Fatalf("GetEvalCase: %v", err)
	}
	if fetched != nil {
		t.Fatalf("GetEvalCase = %+v, want nil (evidência não marcada)", fetched)
	}
}

// TestUpsertEvalCaseReplacesPreviousMarking cobre a correção: marcar de novo
// troca nota e gabarito por completo, sem deixar sobra do gabarito anterior
// (diferente de evidence/RN-06, essa marcação não é imutável).
func TestUpsertEvalCaseReplacesPreviousMarking(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	c1 := newCompetency(t, q, g, "concurrency", "Concorrência")
	c2 := newCompetency(t, q, g, "testing", "Testes")
	e := newEvidence(t, q, g, "evidência revisitada")

	first, err := domain.NewEvalCase(e.ID, "primeira nota", []domain.EvalCaseScore{{CompetencyID: c1.ID, Level: 1}})
	if err != nil {
		t.Fatalf("domain.NewEvalCase: %v", err)
	}
	if err := UpsertEvalCase(context.Background(), q, first, time.Now()); err != nil {
		t.Fatalf("UpsertEvalCase (primeira): %v", err)
	}

	second, err := domain.NewEvalCase(e.ID, "nota corrigida", []domain.EvalCaseScore{{CompetencyID: c2.ID, Level: 4}})
	if err != nil {
		t.Fatalf("domain.NewEvalCase: %v", err)
	}
	if err := UpsertEvalCase(context.Background(), q, second, time.Now()); err != nil {
		t.Fatalf("UpsertEvalCase (segunda): %v", err)
	}

	fetched, err := GetEvalCase(context.Background(), q, e.ID)
	if err != nil {
		t.Fatalf("GetEvalCase: %v", err)
	}
	if fetched.Note != "nota corrigida" {
		t.Fatalf("note = %q, want %q", fetched.Note, "nota corrigida")
	}
	if len(fetched.Scores) != 1 || fetched.Scores[0].CompetencyID != c2.ID || fetched.Scores[0].Level != 4 {
		t.Fatalf("scores = %+v, want só [{%s 4}]", fetched.Scores, c2.ID)
	}
}

func TestDeleteEvalCaseUnmarks(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	c := newCompetency(t, q, g, "concurrency", "Concorrência")
	e := newEvidence(t, q, g, "vai ser desmarcada")

	ec, err := domain.NewEvalCase(e.ID, "nota", []domain.EvalCaseScore{{CompetencyID: c.ID, Level: 3}})
	if err != nil {
		t.Fatalf("domain.NewEvalCase: %v", err)
	}
	if err := UpsertEvalCase(context.Background(), q, ec, time.Now()); err != nil {
		t.Fatalf("UpsertEvalCase: %v", err)
	}
	if err := DeleteEvalCase(context.Background(), q, e.ID); err != nil {
		t.Fatalf("DeleteEvalCase: %v", err)
	}

	fetched, err := GetEvalCase(context.Background(), q, e.ID)
	if err != nil {
		t.Fatalf("GetEvalCase: %v", err)
	}
	if fetched != nil {
		t.Fatalf("GetEvalCase depois de DeleteEvalCase = %+v, want nil", fetched)
	}
}

func TestListEvalCasesForEvidencesGroupsCorrectlyAndSkipsUnmarked(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	c := newCompetency(t, q, g, "concurrency", "Concorrência")
	marked := newEvidence(t, q, g, "marcada")
	unmarked := newEvidence(t, q, g, "não marcada")

	ec, err := domain.NewEvalCase(marked.ID, "nota", []domain.EvalCaseScore{{CompetencyID: c.ID, Level: 3}})
	if err != nil {
		t.Fatalf("domain.NewEvalCase: %v", err)
	}
	if err := UpsertEvalCase(context.Background(), q, ec, time.Now()); err != nil {
		t.Fatalf("UpsertEvalCase: %v", err)
	}

	byEvidence, err := ListEvalCasesForEvidences(context.Background(), q, []string{marked.ID, unmarked.ID})
	if err != nil {
		t.Fatalf("ListEvalCasesForEvidences: %v", err)
	}
	if len(byEvidence) != 1 {
		t.Fatalf("len(byEvidence) = %d, want 1", len(byEvidence))
	}
	got, ok := byEvidence[marked.ID]
	if !ok {
		t.Fatalf("byEvidence[%s] ausente", marked.ID)
	}
	if got.Note != "nota" || len(got.Scores) != 1 || got.Scores[0].Level != 3 {
		t.Fatalf("byEvidence[%s] = %+v, inesperado", marked.ID, got)
	}
	if _, ok := byEvidence[unmarked.ID]; ok {
		t.Fatalf("byEvidence[%s] presente, want ausente (não marcada)", unmarked.ID)
	}
}

func TestListEvalCasesForEvidencesEmptyInput(t *testing.T) {
	q, _ := setup(t)
	byEvidence, err := ListEvalCasesForEvidences(context.Background(), q, nil)
	if err != nil {
		t.Fatalf("ListEvalCasesForEvidences com slice vazia: %v", err)
	}
	if len(byEvidence) != 0 {
		t.Fatalf("esperava mapa vazio, got %v", byEvidence)
	}
}
