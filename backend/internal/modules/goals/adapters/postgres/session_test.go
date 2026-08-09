package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func newSession(t *testing.T, q Querier, g *domain.Goal, startedAt time.Time, durationMin int) *domain.Session {
	t.Helper()
	s, err := domain.NewSession(mustID(t), g.ID, g.UserID, nil, startedAt, durationMin, nil, nil, time.UTC, time.Now())
	if err != nil {
		t.Fatalf("domain.NewSession: %v", err)
	}
	if err := InsertSession(context.Background(), q, s); err != nil {
		t.Fatalf("InsertSession: %v", err)
	}
	return s
}

func TestInsertAndListSessionsByGoal(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	newSession(t, q, g, time.Now(), 15)
	newSession(t, q, g, time.Now(), 25)

	list, err := ListSessionsByGoal(context.Background(), q, g.ID, 10)
	if err != nil {
		t.Fatalf("ListSessionsByGoal: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("esperava 2 sessões, got %d", len(list))
	}

	total, err := SumSessionMinutes(context.Background(), q, g.ID)
	if err != nil {
		t.Fatalf("SumSessionMinutes: %v", err)
	}
	if total != 40 {
		t.Fatalf("total = %d, want 40", total)
	}
}

func TestSumSessionMinutesNoSessions(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	total, err := SumSessionMinutes(context.Background(), q, g.ID)
	if err != nil {
		t.Fatalf("SumSessionMinutes: %v", err)
	}
	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}
}

// TestActiveLocalDatesJoinsSessionsAndEvidence alimenta RN-11: dias distintos
// vindos de sessão e evidência contam juntos, sem duplicar o mesmo dia.
func TestActiveLocalDatesJoinsSessionsAndEvidence(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")

	newSession(t, q, g, time.Now(), 10)
	newEvidence(t, q, g, "evidência do mesmo dia")

	dates, err := ActiveLocalDates(context.Background(), q, g.ID, 30)
	if err != nil {
		t.Fatalf("ActiveLocalDates: %v", err)
	}
	if len(dates) == 0 {
		t.Fatal("esperava ao menos um dia ativo")
	}
	seen := map[string]bool{}
	for _, d := range dates {
		seen[d.Format("2006-01-02")] = true
	}
	if len(seen) != 1 {
		t.Fatalf("sessão e evidência no mesmo dia deveriam colapsar num só dia distinto, got %d: %v", len(seen), dates)
	}
}
