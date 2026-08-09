package domain

import (
	"fmt"
	"time"
)

const DefaultConsistencyWindowDays = 30

// ProjectionReliableAfterDays é §7.2: só projeta com >= 3 semanas de
// histórico. Antes disso, "ainda coletando ritmo" é mais honesto que uma
// média sobre poucos dias.
const ProjectionReliableAfterDays = 21

// Projection é a projeção honesta (§7.2, 02-modelo-de-dados.md §14.4).
// NextMilestone/WeeksToNextMin/WeeksToNextMax/IfYouDouble ficam sempre nil
// nesta fatia: nenhuma tabela guarda esforço restante por marco, e "nunca
// invente previsão" (CLAUDE.md) vale tanto quanto a regra dos 21 dias — sem
// esforço estimado, não existe cálculo honesto de quantas semanas faltam.
// Chegam quando o agente de trilha (Fatia 3+) tiver como estimar isso.
type Projection struct {
	Available      bool
	Reason         *string
	MinutesPerWeek *float64
	NextMilestone  *string
	WeeksToNextMin *int
	WeeksToNextMax *int
	IfYouDouble    *string
}

// minutesPerWeekDivisor é o mesmo 4,3 de 02-modelo-de-dados.md §14.4 —
// média de semanas em 30 dias (30/7 arredondado a uma casa).
const minutesPerWeekDivisor = 4.3

// ComputeProjection nunca inventa previsão: sem sessão nenhuma na janela ou
// com menos de 3 semanas de span, devolve available:false com o motivo. Com
// dado suficiente, devolve o ritmo real — nunca uma data de chegada, porque
// não há estimativa de esforço restante no modelo (ver comentário em
// Projection).
func ComputeProjection(totalMinutes, activeDays, spanDays int) Projection {
	if activeDays == 0 {
		reason := "ainda não há sessão registrada nos últimos 30 dias"
		return Projection{Available: false, Reason: &reason}
	}
	if spanDays < ProjectionReliableAfterDays {
		weeksSoFar := spanDays / 7
		reason := fmt.Sprintf("ainda coletando ritmo (%d de 3 semanas)", weeksSoFar)
		return Projection{Available: false, Reason: &reason}
	}
	minutesPerWeek := float64(totalMinutes) / minutesPerWeekDivisor
	return Projection{Available: true, MinutesPerWeek: &minutesPerWeek}
}

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
