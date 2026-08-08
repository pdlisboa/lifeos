package domain

import (
	"fmt"
	"time"
)

const DefaultConsistencyWindowDays = 30

type ConsistencyWindow struct {
	ActiveDays int
	WindowDays int
	TodayDone  bool
}

// Label é o texto de §7.5: "18 dos últimos 30 dias", nunca uma streak que
// zera (RN-11, P6).
func (w ConsistencyWindow) Label() string {
	return fmt.Sprintf("%d dos últimos %d dias", w.ActiveDays, w.WindowDays)
}

// ComputeConsistency é RN-11: janela móvel, nunca streak binária. activeDates
// pode ter duplicatas e vir de fontes diferentes (sessão + evidência) — só os
// dias distintos contam (02-modelo-de-dados.md §14.3).
func ComputeConsistency(activeDates []time.Time, today time.Time, windowDays int) ConsistencyWindow {
	today = DateOnly(today)
	since := today.AddDate(0, 0, -(windowDays - 1))

	seen := make(map[time.Time]bool)
	todayDone := false
	for _, d := range activeDates {
		d = DateOnly(d)
		if d.Before(since) || d.After(today) {
			continue
		}
		seen[d] = true
		if d.Equal(today) {
			todayDone = true
		}
	}
	return ConsistencyWindow{
		ActiveDays: len(seen),
		WindowDays: windowDays,
		TodayDone:  todayDone,
	}
}

// RisesInWindow conta quantas vezes o nível subiu (to > from) desde `since`
// — alimenta o Painel de Delta (§7.1, campo risesLast90d).
func RisesInWindow(events []LevelEvent, since time.Time) int {
	n := 0
	for _, e := range events {
		if e.FromLevel != nil && e.ToLevel > *e.FromLevel && !e.OccurredAt.Before(since) {
			n++
		}
	}
	return n
}
