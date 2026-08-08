package domain

import (
	"testing"
	"time"
)

func TestComputeConsistencyWindow(t *testing.T) {
	today := time.Date(2026, 8, 8, 15, 0, 0, 0, time.UTC)

	dates := []time.Time{
		today,
		today.AddDate(0, 0, -1),
		today.AddDate(0, 0, -1), // duplicata do mesmo dia — não deve contar duas vezes
		today.AddDate(0, 0, -29),
		today.AddDate(0, 0, -30), // fora da janela de 30 dias
		today.AddDate(0, 0, 1),   // futuro — não deve contar
	}

	w := ComputeConsistency(dates, today, 30)
	if w.ActiveDays != 3 {
		t.Fatalf("ActiveDays = %d, want 3", w.ActiveDays)
	}
	if w.WindowDays != 30 {
		t.Fatalf("WindowDays = %d, want 30", w.WindowDays)
	}
	if !w.TodayDone {
		t.Fatal("TodayDone deveria ser true")
	}

	label := w.Label()
	want := "3 dos últimos 30 dias"
	if label != want {
		t.Fatalf("Label() = %q, want %q", label, want)
	}
}

func TestComputeConsistencyNeverExposesStreak(t *testing.T) {
	// Regressão de propósito: a métrica é sempre "N dos últimos M dias",
	// nunca uma contagem de dias consecutivos que possa "zerar" (RN-11, P6).
	today := DateOnly(time.Now())
	dates := []time.Time{today, today.AddDate(0, 0, -5), today.AddDate(0, 0, -10)}
	w := ComputeConsistency(dates, today, 30)
	if w.ActiveDays != 3 {
		t.Fatalf("ActiveDays = %d, want 3 mesmo com lacunas entre os dias", w.ActiveDays)
	}
}

func TestRisesInWindow(t *testing.T) {
	now := time.Now()
	l1, l2, l3 := 1, 2, 3
	events := []LevelEvent{
		{FromLevel: &l1, ToLevel: 2, OccurredAt: now.AddDate(0, 0, -10)},  // subida, dentro
		{FromLevel: &l2, ToLevel: 1, OccurredAt: now.AddDate(0, 0, -5)},   // queda, não conta
		{FromLevel: &l3, ToLevel: 4, OccurredAt: now.AddDate(0, 0, -200)}, // subida, fora da janela
		{FromLevel: nil, ToLevel: 1, OccurredAt: now},                     // baseline inicial, não é "subida"
	}
	got := RisesInWindow(events, now.AddDate(0, 0, -90))
	if got != 1 {
		t.Fatalf("RisesInWindow = %d, want 1", got)
	}
}
