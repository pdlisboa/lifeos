package domain

import (
	"fmt"
	"time"
)

// goalTransitions espelha o ciclo de vida de 00-negocio.md §10. É a única
// fonte de verdade sobre quais mudanças de estado existem; regras adicionais
// (RN-01 em Activate, RN-10 em Close) vivem nos métodos específicos, que
// sabem o contexto que falta aqui.
var goalTransitions = map[GoalStatus]map[GoalStatus]bool{
	GoalDraft:     {GoalActive: true},
	GoalActive:    {GoalAtRisk: true, GoalPaused: true, GoalCompleted: true, GoalAbandoned: true},
	GoalAtRisk:    {GoalActive: true, GoalStagnant: true, GoalPaused: true, GoalCompleted: true, GoalAbandoned: true},
	GoalStagnant:  {GoalActive: true, GoalPaused: true, GoalCompleted: true, GoalAbandoned: true},
	GoalPaused:    {GoalActive: true},
	GoalCompleted: {},
	GoalAbandoned: {},
}

// RequiresDebrief é RN-10: só completed/abandoned exigem debrief.
func (s GoalStatus) RequiresDebrief() bool {
	return s == GoalCompleted || s == GoalAbandoned
}

func (g *Goal) CanTransition(to GoalStatus) bool {
	next, ok := goalTransitions[g.Status]
	if !ok {
		return false
	}
	return next[to]
}

func (g *Goal) Transition(to GoalStatus, now time.Time) error {
	if !g.CanTransition(to) {
		return newRuleError("", fmt.Sprintf("transição inválida: %s → %s", g.Status, to))
	}
	g.Status = to
	g.UpdatedAt = now
	if to == GoalCompleted || to == GoalAbandoned {
		g.ClosedAt = &now
	}
	return nil
}
