package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func TestInsertAndGetProbeWithTurns(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	c := newCompetency(t, q, g, "concurrency", "Concorrência")

	p := domain.NewProbe(mustID(t), g.ID, userID)
	if err := InsertProbe(context.Background(), q, p); err != nil {
		t.Fatalf("InsertProbe: %v", err)
	}

	turn := domain.ProbeTurn{ID: mustID(t), ProbeID: p.ID, CompetencyID: &c.ID, Question: "Já mexeu com goroutines?"}
	if err := p.AskTurn(turn); err != nil {
		t.Fatalf("AskTurn: %v", err)
	}
	if err := InsertProbeTurn(context.Background(), q, &p.Turns[0]); err != nil {
		t.Fatalf("InsertProbeTurn: %v", err)
	}
	if err := UpdateProbe(context.Background(), q, p); err != nil {
		t.Fatalf("UpdateProbe: %v", err)
	}

	fetched, err := GetProbeByGoal(context.Background(), q, g.ID)
	if err != nil {
		t.Fatalf("GetProbeByGoal: %v", err)
	}
	if fetched.AskedCount != 1 || len(fetched.Turns) != 1 {
		t.Fatalf("probe recuperada não bate: %+v", fetched)
	}
	if fetched.Turns[0].CompetencyID == nil || *fetched.Turns[0].CompetencyID != c.ID {
		t.Fatalf("competencyId do turno não bate: %+v", fetched.Turns[0])
	}

	now := time.Now()
	answer := "sim"
	fetched.Turns[0].Answer = &answer
	fetched.Turns[0].AnsweredAt = &now
	if err := UpdateProbeTurnAnswer(context.Background(), q, &fetched.Turns[0]); err != nil {
		t.Fatalf("UpdateProbeTurnAnswer: %v", err)
	}

	refetched, err := GetProbeByGoal(context.Background(), q, g.ID)
	if err != nil {
		t.Fatalf("GetProbeByGoal (2): %v", err)
	}
	if refetched.Turns[0].Answer == nil || *refetched.Turns[0].Answer != "sim" {
		t.Fatalf("resposta não persistiu: %+v", refetched.Turns[0])
	}
}

func TestGetProbeByGoalNotFound(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta sem sondagem")
	if _, err := GetProbeByGoal(context.Background(), q, g.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("erro = %v, want ErrNotFound", err)
	}
}
