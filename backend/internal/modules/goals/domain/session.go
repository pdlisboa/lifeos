package domain

import (
	"fmt"
	"time"
)

const (
	MinSessionDuration = 1
	MaxSessionDuration = 600
)

type Session struct {
	ID       string
	GoalID   string
	UserID   string
	ActionID *string

	StartedAt   time.Time
	DurationMin int
	Note        *string
	Energy      *int

	// LocalOn é o dia do usuário, calculado no servidor a partir do timezone
	// dele — nunca no navegador (CLAUDE.md, armadilhas específicas deste
	// domínio: deploy em UTC faria a consistência virar de dia às 21h).
	LocalOn time.Time

	CreatedAt time.Time
}

func NewSession(id, goalID, userID string, actionID *string, startedAt time.Time, durationMin int, note *string, energy *int, loc *time.Location, now time.Time) (*Session, error) {
	if durationMin < MinSessionDuration || durationMin > MaxSessionDuration {
		return nil, newRuleError("", fmt.Sprintf("duração deve estar entre %d e %d minutos", MinSessionDuration, MaxSessionDuration))
	}
	if energy != nil && (*energy < 1 || *energy > 5) {
		return nil, newRuleError("", "energy deve estar entre 1 e 5")
	}
	return &Session{
		ID:          id,
		GoalID:      goalID,
		UserID:      userID,
		ActionID:    actionID,
		StartedAt:   startedAt,
		DurationMin: durationMin,
		Note:        trimToNil(note),
		Energy:      energy,
		LocalOn:     DateOnly(startedAt.In(loc)),
		CreatedAt:   now,
	}, nil
}

// DateOnly zera a hora, mantendo só a data — usado para local_on e para
// comparar dias em ComputeConsistency sem o horário atrapalhar a igualdade.
func DateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
