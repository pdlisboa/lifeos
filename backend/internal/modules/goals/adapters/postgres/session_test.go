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

func TestSessionPaceLast30dNoSessions(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")

	pace, err := SessionPaceLast30d(context.Background(), q, g.ID)
	if err != nil {
		t.Fatalf("SessionPaceLast30d: %v", err)
	}
	if pace.TotalMinutes != 0 || pace.ActiveDays != 0 || pace.SpanDays != 0 {
		t.Fatalf("sem sessão nenhuma, esperava tudo zerado, got %+v", pace)
	}
}

// TestSessionPaceLast30dWithHistory cobre o ritmo real de §7.2/§14.4: soma
// de minutos, dias distintos e o span até a sessão mais antiga da janela.
func TestSessionPaceLast30dWithHistory(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")

	now := time.Now()
	newSession(t, q, g, now.AddDate(0, 0, -25), 30)
	newSession(t, q, g, now.AddDate(0, 0, -10), 20)
	newSession(t, q, g, now, 15)

	pace, err := SessionPaceLast30d(context.Background(), q, g.ID)
	if err != nil {
		t.Fatalf("SessionPaceLast30d: %v", err)
	}
	if pace.TotalMinutes != 65 {
		t.Fatalf("TotalMinutes = %d, want 65", pace.TotalMinutes)
	}
	if pace.ActiveDays != 3 {
		t.Fatalf("ActiveDays = %d, want 3", pace.ActiveDays)
	}
	if pace.SpanDays < 24 || pace.SpanDays > 26 {
		t.Fatalf("SpanDays = %d, want ~25", pace.SpanDays)
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
