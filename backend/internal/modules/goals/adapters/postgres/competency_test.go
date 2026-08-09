package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func TestInsertAndGetCompetencyRoundTrip(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	c := newCompetency(t, q, g, "concurrency", "Concorrência")

	fetched, err := GetCompetency(context.Background(), q, userID, c.ID)
	if err != nil {
		t.Fatalf("GetCompetency: %v", err)
	}
	if fetched.PackKey != "concurrency" || fetched.Label != "Concorrência" || fetched.Weight != 0.5 {
		t.Fatalf("round-trip não bate: %+v", fetched)
	}
	if fetched.CurrentLevel != nil {
		t.Fatalf("current_level deveria ser nil (desconhecido), got %v", *fetched.CurrentLevel)
	}
	if fetched.Confidence != domain.ConfidenceUnknown {
		t.Fatalf("confidence = %s, want unknown", fetched.Confidence)
	}
}

func TestGetCompetencyNotFound(t *testing.T) {
	q, userID := setup(t)
	if _, err := GetCompetency(context.Background(), q, userID, mustID(t)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("erro = %v, want ErrNotFound", err)
	}
}

func TestListAndCountCompetenciesByGoal(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	newCompetency(t, q, g, "concurrency", "Concorrência")
	newCompetency(t, q, g, "testing", "Testes")

	list, err := ListCompetenciesByGoal(context.Background(), q, g.ID)
	if err != nil {
		t.Fatalf("ListCompetenciesByGoal: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("esperava 2 competências, got %d", len(list))
	}

	n, err := CountCompetenciesByGoal(context.Background(), q, g.ID)
	if err != nil {
		t.Fatalf("CountCompetenciesByGoal: %v", err)
	}
	if n != 2 {
		t.Fatalf("count = %d, want 2", n)
	}
}

// TestUpdateCompetencyStateAndLevelEvents cobre o ciclo de RN-04 na camada de
// persistência: gravar o evento append-only e atualizar o cache da
// competência são duas escritas distintas.
func TestUpdateCompetencyStateAndLevelEvents(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	c := newCompetency(t, q, g, "testing", "Testes")
	proof := newEvidence(t, q, g, "evidência que sustentou a medição")

	ev, err := domain.NewLevelEvent(mustID(t), c.ID, userID, nil, 2, domain.ConfidenceHigh, domain.SourceSelf, &proof.ID, "primeira medição", time.Now())
	if err != nil {
		t.Fatalf("domain.NewLevelEvent: %v", err)
	}
	if err := c.ApplyLevelEvent(*ev); err != nil {
		t.Fatalf("ApplyLevelEvent: %v", err)
	}
	if err := InsertLevelEvent(context.Background(), q, ev); err != nil {
		t.Fatalf("InsertLevelEvent: %v", err)
	}
	if err := UpdateCompetencyState(context.Background(), q, c); err != nil {
		t.Fatalf("UpdateCompetencyState: %v", err)
	}

	fetched, err := GetCompetency(context.Background(), q, userID, c.ID)
	if err != nil {
		t.Fatalf("GetCompetency: %v", err)
	}
	if fetched.CurrentLevel == nil || *fetched.CurrentLevel != 2 {
		t.Fatalf("current_level = %v, want 2", fetched.CurrentLevel)
	}
	if fetched.BaselineLevel == nil || *fetched.BaselineLevel != 2 {
		t.Fatalf("baseline_level = %v, want 2", fetched.BaselineLevel)
	}

	events, err := ListLevelEvents(context.Background(), q, c.ID)
	if err != nil {
		t.Fatalf("ListLevelEvents: %v", err)
	}
	if len(events) != 1 || events[0].Rationale != "primeira medição" {
		t.Fatalf("eventos = %+v", events)
	}
	if events[0].EvidenceID == nil || *events[0].EvidenceID != proof.ID {
		t.Fatalf("EvidenceID = %v, want %q", events[0].EvidenceID, proof.ID)
	}

	forGoal, err := ListLevelEventsForGoal(context.Background(), q, g.ID)
	if err != nil {
		t.Fatalf("ListLevelEventsForGoal: %v", err)
	}
	if len(forGoal) != 1 {
		t.Fatalf("esperava 1 evento para a meta, got %d", len(forGoal))
	}
}
