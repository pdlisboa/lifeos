package app

import (
	"context"
	"errors"
	"testing"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func createTestEvidence(t *testing.T, svc *Service, userID, goalID string) *domain.Evidence {
	t.Helper()
	body := "func worker() {}"
	ev, err := svc.CreateEvidence(context.Background(), CreateEvidenceInput{
		UserID: userID, GoalID: goalID, Kind: domain.EvidenceCodeSnippet, Body: &body,
	})
	if err != nil {
		t.Fatalf("CreateEvidence: %v", err)
	}
	return ev
}

func TestMarkEvidenceAsEvalCaseRoundTrip(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta")
	detail, err := svc.GetGoal(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetGoal: %v", err)
	}
	comp := detail.Competencies[0].ID
	ev := createTestEvidence(t, svc, userID, active.Goal.ID)

	ec, err := svc.MarkEvidenceAsEvalCase(context.Background(), MarkEvidenceEvalCaseInput{
		UserID: userID, EvidenceID: ev.ID, Note: "nível claro de goroutine",
		Scores: []domain.EvalCaseScore{{CompetencyID: comp, Level: 2}},
	})
	if err != nil {
		t.Fatalf("MarkEvidenceAsEvalCase: %v", err)
	}
	if ec.Note != "nível claro de goroutine" {
		t.Fatalf("note = %q", ec.Note)
	}

	fetched, err := svc.GetEvalCase(context.Background(), userID, ev.ID)
	if err != nil {
		t.Fatalf("GetEvalCase: %v", err)
	}
	if fetched == nil || fetched.Note != ec.Note {
		t.Fatalf("GetEvalCase = %+v, want %+v", fetched, ec)
	}
}

func TestMarkEvidenceAsEvalCaseRejectsCompetencyFromOtherGoal(t *testing.T) {
	svc, userID := newFixture(t)
	a := readyGoal(t, svc, userID, "Meta A")
	b := readyGoal(t, svc, userID, "Meta B")
	bDetail, err := svc.GetGoal(context.Background(), userID, b.Goal.ID)
	if err != nil {
		t.Fatalf("GetGoal: %v", err)
	}
	ev := createTestEvidence(t, svc, userID, a.Goal.ID)

	_, err = svc.MarkEvidenceAsEvalCase(context.Background(), MarkEvidenceEvalCaseInput{
		UserID: userID, EvidenceID: ev.ID, Note: "nota",
		Scores: []domain.EvalCaseScore{{CompetencyID: bDetail.Competencies[0].ID, Level: 2}},
	})
	if err == nil {
		t.Fatal("competência de outra meta deveria ser rejeitada")
	}
}

func TestMarkEvidenceAsEvalCaseRejectsOtherUsersEvidence(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta")
	detail, err := svc.GetGoal(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetGoal: %v", err)
	}
	ev := createTestEvidence(t, svc, userID, active.Goal.ID)

	_, otherUserID := newFixture(t)
	_, err = svc.MarkEvidenceAsEvalCase(context.Background(), MarkEvidenceEvalCaseInput{
		UserID: otherUserID, EvidenceID: ev.ID, Note: "nota",
		Scores: []domain.EvalCaseScore{{CompetencyID: detail.Competencies[0].ID, Level: 2}},
	})
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound (evidência de outro usuário), got %v", err)
	}
}

func TestUnmarkEvidenceEvalCase(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta")
	detail, err := svc.GetGoal(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetGoal: %v", err)
	}
	ev := createTestEvidence(t, svc, userID, active.Goal.ID)

	if _, err := svc.MarkEvidenceAsEvalCase(context.Background(), MarkEvidenceEvalCaseInput{
		UserID: userID, EvidenceID: ev.ID, Note: "nota",
		Scores: []domain.EvalCaseScore{{CompetencyID: detail.Competencies[0].ID, Level: 3}},
	}); err != nil {
		t.Fatalf("MarkEvidenceAsEvalCase: %v", err)
	}

	if err := svc.UnmarkEvidenceEvalCase(context.Background(), userID, ev.ID); err != nil {
		t.Fatalf("UnmarkEvidenceEvalCase: %v", err)
	}

	fetched, err := svc.GetEvalCase(context.Background(), userID, ev.ID)
	if err != nil {
		t.Fatalf("GetEvalCase: %v", err)
	}
	if fetched != nil {
		t.Fatalf("GetEvalCase depois de desmarcar = %+v, want nil", fetched)
	}
}

func TestGetEvalCaseNotMarked(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta")
	ev := createTestEvidence(t, svc, userID, active.Goal.ID)

	fetched, err := svc.GetEvalCase(context.Background(), userID, ev.ID)
	if err != nil {
		t.Fatalf("GetEvalCase: %v", err)
	}
	if fetched != nil {
		t.Fatalf("GetEvalCase = %+v, want nil (não marcada)", fetched)
	}
}

// TestListEvidenceIncludesEvalCase cobre o cartão do museu carregando a
// marcação junto, sem N+1 (mesmo padrão de CompetencyIDs/LevelsAtTime).
func TestListEvidenceIncludesEvalCase(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta")
	detail, err := svc.GetGoal(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetGoal: %v", err)
	}
	ev := createTestEvidence(t, svc, userID, active.Goal.ID)
	if _, err := svc.MarkEvidenceAsEvalCase(context.Background(), MarkEvidenceEvalCaseInput{
		UserID: userID, EvidenceID: ev.ID, Note: "caso de eval",
		Scores: []domain.EvalCaseScore{{CompetencyID: detail.Competencies[0].ID, Level: 1}},
	}); err != nil {
		t.Fatalf("MarkEvidenceAsEvalCase: %v", err)
	}

	cards, err := svc.ListEvidence(context.Background(), userID, active.Goal.ID, nil, true, 10)
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	var card *EvidenceCard
	for i := range cards {
		if cards[i].Evidence.ID == ev.ID {
			card = &cards[i]
		}
	}
	if card == nil {
		t.Fatal("evidência não apareceu na listagem")
	}
	if card.EvalCase == nil || card.EvalCase.Note != "caso de eval" {
		t.Fatalf("card.EvalCase = %+v, want nota preenchida", card.EvalCase)
	}
}
