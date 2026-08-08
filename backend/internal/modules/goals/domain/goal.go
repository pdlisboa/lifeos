package domain

import (
	"fmt"
	"strings"
	"time"
)

type GoalArchetype string

const (
	ArchetypeSkill   GoalArchetype = "skill"
	ArchetypeHabit   GoalArchetype = "habit"
	ArchetypeProject GoalArchetype = "project"
	ArchetypeMetric  GoalArchetype = "metric"
)

func (a GoalArchetype) Valid() bool {
	switch a {
	case ArchetypeSkill, ArchetypeHabit, ArchetypeProject, ArchetypeMetric:
		return true
	}
	return false
}

type GoalStatus string

const (
	GoalDraft     GoalStatus = "draft"
	GoalActive    GoalStatus = "active"
	GoalAtRisk    GoalStatus = "at_risk"
	GoalStagnant  GoalStatus = "stagnant"
	GoalPaused    GoalStatus = "paused"
	GoalCompleted GoalStatus = "completed"
	GoalAbandoned GoalStatus = "abandoned"
)

// MaxActiveGoals é RN-02: dispersão é uma das causas de abandono, e é a
// única que dá para prevenir por regra de negócio (00-negocio.md P7).
const MaxActiveGoals = 3

// MinCompetenciesToActivate é parte de RN-01.
const MinCompetenciesToActivate = 3

type Goal struct {
	ID        string
	UserID    string
	Title     string
	Archetype GoalArchetype
	PackID    string
	Why       *string

	DefinitionOfDone *string
	HorizonOn        *time.Time

	Status      GoalStatus
	ScopeMode   string
	PausedUntil *time.Time

	ActivatedAt    *time.Time
	LastActivityAt *time.Time
	ClosedAt       *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewGoal cria uma meta em rascunho. A definição de pronto e as competências
// podem vir depois, pela sondagem — a fricção na criação é o que mata o
// entusiasmo (D2 em 00-negocio.md §14).
func NewGoal(id, userID, title string, archetype GoalArchetype, packID string, why *string, now time.Time) (*Goal, error) {
	title = strings.TrimSpace(title)
	if len(title) < 3 || len(title) > 120 {
		return nil, newRuleError("", "título deve ter entre 3 e 120 caracteres")
	}
	if !archetype.Valid() {
		return nil, newRuleError("", fmt.Sprintf("arquétipo inválido: %s", archetype))
	}
	if strings.TrimSpace(packID) == "" {
		return nil, newRuleError("", "packId é obrigatório")
	}
	return &Goal{
		ID:        id,
		UserID:    userID,
		Title:     title,
		Archetype: archetype,
		PackID:    packID,
		Why:       trimToNil(why),
		Status:    GoalDraft,
		ScopeMode: "normal",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Blockers é RN-01: o que falta para a meta poder ser ativada, em linguagem
// que a UI mostra direto (03-api.md §5, campo Goal.blockers) em vez de você
// descobrir no 409.
func (g *Goal) Blockers(competencyCount int) []string {
	var blockers []string
	if g.DefinitionOfDone == nil || strings.TrimSpace(*g.DefinitionOfDone) == "" {
		blockers = append(blockers, "falta a definição de pronto")
	}
	if competencyCount < MinCompetenciesToActivate {
		blockers = append(blockers, fmt.Sprintf("precisa de ao menos %d competências (tem %d)", MinCompetenciesToActivate, competencyCount))
	}
	return blockers
}

func (g *Goal) ReadyToActivate(competencyCount int) bool {
	return len(g.Blockers(competencyCount)) == 0
}

// Activate transiciona draft → active (RN-01). A checagem de RN-02 (máx. 3
// ativas) não é responsabilidade do domínio — depende de contar outras
// metas do usuário, e mora no caso de uso (app/activate_goal.go).
func (g *Goal) Activate(competencyCount int, now time.Time) error {
	if !g.ReadyToActivate(competencyCount) {
		return newRuleError("RN-01", strings.Join(g.Blockers(competencyCount), "; "))
	}
	if err := g.Transition(GoalActive, now); err != nil {
		return err
	}
	g.ActivatedAt = &now
	g.LastActivityAt = &now
	return nil
}

// Pause é a pausa consciente de §7.6 — estado de primeira classe, não
// fracasso, e libera o slot de RN-02.
func (g *Goal) Pause(reviewOn time.Time, now time.Time) error {
	if err := g.Transition(GoalPaused, now); err != nil {
		return err
	}
	g.PausedUntil = &reviewOn
	return nil
}

// Close é RN-10: concluir ou abandonar exige debrief no mesmo comando.
// debriefProvided vem do caso de uso, que grava os dois juntos ou nenhum.
func (g *Goal) Close(outcome GoalStatus, debriefProvided bool, now time.Time) error {
	if outcome != GoalCompleted && outcome != GoalAbandoned {
		return newRuleError("", "outcome deve ser completed ou abandoned")
	}
	if !debriefProvided {
		return newRuleError("RN-10", "encerrar uma meta exige debrief")
	}
	return g.Transition(outcome, now)
}

func (g *Goal) TouchActivity(now time.Time) {
	g.LastActivityAt = &now
	g.UpdatedAt = now
}

func (g *Goal) DaysActive(now time.Time) int {
	if g.ActivatedAt == nil {
		return 0
	}
	return int(now.Sub(*g.ActivatedAt).Hours() / 24)
}

func trimToNil(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}
