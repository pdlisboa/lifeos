package domain

import (
	"fmt"
	"strings"
	"time"
)

type ActionStatus string

const (
	ActionPending   ActionStatus = "pending"
	ActionCompleted ActionStatus = "completed"
	ActionSkipped   ActionStatus = "skipped"
)

type GeneratedBy string

const (
	GeneratedByAgent    GeneratedBy = "agent"
	GeneratedByUser     GeneratedBy = "user"
	GeneratedByFallback GeneratedBy = "fallback"
)

type OriginKind string

const (
	OriginPractice     OriginKind = "practice"
	OriginRevalidation OriginKind = "revalidation"
	OriginRecovery     OriginKind = "recovery"
)

type SkipReason string

const (
	SkipTooHard     SkipReason = "too_hard"
	SkipTooEasy     SkipReason = "too_easy"
	SkipNoTime      SkipReason = "no_time"
	SkipNotRelevant SkipReason = "not_relevant"
	SkipOther       SkipReason = "other"
)

func (r SkipReason) Valid() bool {
	switch r {
	case SkipTooHard, SkipTooEasy, SkipNoTime, SkipNotRelevant, SkipOther:
		return true
	}
	return false
}

const (
	MinEstimatedMin      = 5
	MaxEstimatedMin      = 30
	MinMinimalVariantLen = 15
)

type NextAction struct {
	ID     string
	GoalID string
	UserID string

	MilestoneID  *string
	CompetencyID *string

	Title          string
	Detail         *string
	PracticeFormat *string
	EstimatedMin   int
	MinimalVariant string

	DifficultyHint *string
	GeneratedBy    GeneratedBy
	OriginKind     *OriginKind

	Status     ActionStatus
	SkipReason *SkipReason
	ResolvedAt *time.Time
	CreatedAt  time.Time
}

// NewNextAction garante P4 (ação pequena, 5-30 min) e a mecânica do dia
// mínimo (§7.5): minimalVariant é obrigatório, não opcional. Se fosse
// opcional, a mecânica anti-quebra-de-streak morreria sem ninguém perceber
// (04-agentes.md §4.2).
func NewNextAction(id, goalID, userID, title string, estimatedMin int, minimalVariant string, generatedBy GeneratedBy, now time.Time) (*NextAction, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, newRuleError("", "título da ação é obrigatório")
	}
	if estimatedMin < MinEstimatedMin || estimatedMin > MaxEstimatedMin {
		return nil, newRuleError("P4", fmt.Sprintf("ação deve durar entre %d e %d minutos", MinEstimatedMin, MaxEstimatedMin))
	}
	minimalVariant = strings.TrimSpace(minimalVariant)
	if len(minimalVariant) < MinMinimalVariantLen {
		return nil, newRuleError("", "minimalVariant é obrigatório e precisa descrever a versão de 5 minutos")
	}
	return &NextAction{
		ID:             id,
		GoalID:         goalID,
		UserID:         userID,
		Title:          title,
		EstimatedMin:   estimatedMin,
		MinimalVariant: minimalVariant,
		GeneratedBy:    generatedBy,
		Status:         ActionPending,
		CreatedAt:      now,
	}, nil
}

func (a *NextAction) Complete(now time.Time) error {
	if a.Status != ActionPending {
		return newRuleError("", "só uma ação pendente pode ser concluída")
	}
	a.Status = ActionCompleted
	a.ResolvedAt = &now
	return nil
}

// Skip exige motivo — é o insumo da calibragem de dificuldade (§7.3). Sem
// ele o sistema não distingue "difícil demais" de "sem tempo", que pedem
// respostas opostas.
func (a *NextAction) Skip(reason SkipReason, now time.Time) error {
	if a.Status != ActionPending {
		return newRuleError("", "só uma ação pendente pode ser pulada")
	}
	if !reason.Valid() {
		return newRuleError("", fmt.Sprintf("motivo de skip inválido: %s", reason))
	}
	a.Status = ActionSkipped
	a.SkipReason = &reason
	a.ResolvedAt = &now
	return nil
}

// NextDifficultyHint é a metade barata da calibragem de §7.3 que dá para
// fazer sem histórico nenhum: o motivo do skip já é um sinal direto.
func NextDifficultyHint(reason SkipReason) string {
	switch reason {
	case SkipTooHard:
		return "easier"
	case SkipTooEasy:
		return "harder"
	default:
		return "same"
	}
}
