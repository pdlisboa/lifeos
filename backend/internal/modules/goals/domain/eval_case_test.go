package domain

import "testing"

func TestNewEvalCaseValid(t *testing.T) {
	ec, err := NewEvalCase("ev1", "  gabarito claro  ", []EvalCaseScore{{CompetencyID: "c1", Level: 3}})
	if err != nil {
		t.Fatalf("NewEvalCase: %v", err)
	}
	if ec.Note != "gabarito claro" {
		t.Fatalf("note = %q, want nota sem espaços nas pontas", ec.Note)
	}
	if len(ec.Scores) != 1 || ec.Scores[0].CompetencyID != "c1" || ec.Scores[0].Level != 3 {
		t.Fatalf("scores = %+v", ec.Scores)
	}
}

func TestNewEvalCaseRejectsEmptyNote(t *testing.T) {
	if _, err := NewEvalCase("ev1", "   ", []EvalCaseScore{{CompetencyID: "c1", Level: 3}}); err == nil {
		t.Fatal("nota vazia deveria falhar")
	}
}

func TestNewEvalCaseRejectsNoScores(t *testing.T) {
	if _, err := NewEvalCase("ev1", "nota", nil); err == nil {
		t.Fatal("gabarito vazio deveria falhar")
	}
}

func TestNewEvalCaseRejectsLevelOutOfRange(t *testing.T) {
	if _, err := NewEvalCase("ev1", "nota", []EvalCaseScore{{CompetencyID: "c1", Level: 6}}); err == nil {
		t.Fatal("nível 6 deveria falhar")
	}
	if _, err := NewEvalCase("ev1", "nota", []EvalCaseScore{{CompetencyID: "c1", Level: -1}}); err == nil {
		t.Fatal("nível -1 deveria falhar")
	}
}

func TestNewEvalCaseRejectsDuplicateCompetency(t *testing.T) {
	_, err := NewEvalCase("ev1", "nota", []EvalCaseScore{
		{CompetencyID: "c1", Level: 2},
		{CompetencyID: "c1", Level: 3},
	})
	if err == nil {
		t.Fatal("competência duplicada no gabarito deveria falhar")
	}
}
