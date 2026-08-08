package domain

import (
	"fmt"
	"strings"
	"time"
)

type Confidence string

const (
	ConfidenceUnknown Confidence = "unknown"
	ConfidenceLow     Confidence = "low"
	ConfidenceMedium  Confidence = "medium"
	ConfidenceHigh    Confidence = "high"
)

func (c Confidence) Valid() bool {
	switch c {
	case ConfidenceUnknown, ConfidenceLow, ConfidenceMedium, ConfidenceHigh:
		return true
	}
	return false
}

type LevelSource string

const (
	SourceProbe               LevelSource = "probe"
	SourceSelf                LevelSource = "self"
	SourceAssessment          LevelSource = "assessment"
	SourceWeeklyConsolidation LevelSource = "weekly_consolidation"
	SourceDecay               LevelSource = "decay"
	SourceRevalidation        LevelSource = "revalidation"
)

func (s LevelSource) Valid() bool {
	switch s {
	case SourceProbe, SourceSelf, SourceAssessment, SourceWeeklyConsolidation, SourceDecay, SourceRevalidation:
		return true
	}
	return false
}

const (
	MinLevel = 0
	MaxLevel = 5
)

type Competency struct {
	ID     string
	GoalID string
	UserID string

	PackKey string
	Label   string
	Weight  float64

	CurrentLevel   *int // nil = desconhecido (nunca 0 — 04-agentes.md §4.6)
	Confidence     Confidence
	BaselineLevel  *int
	LastEvidenceAt *time.Time

	RetiredAt *time.Time

	CreatedAt time.Time
}

// NewCompetency materializa uma competência copiada do pack (04-agentes.md
// §4.6). Nasce sempre com nível desconhecido — a sondagem e as evidências
// preenchem o resto; RN-01 só exige a quantidade, nunca o valor.
func NewCompetency(id, goalID, userID, packKey, label string, weight float64, now time.Time) (*Competency, error) {
	if strings.TrimSpace(packKey) == "" || strings.TrimSpace(label) == "" {
		return nil, newRuleError("", "pack_key e label são obrigatórios")
	}
	if weight <= 0 || weight > 1 {
		return nil, newRuleError("", "weight deve estar em (0, 1]")
	}
	return &Competency{
		ID:         id,
		GoalID:     goalID,
		UserID:     userID,
		PackKey:    packKey,
		Label:      label,
		Weight:     weight,
		Confidence: ConfidenceUnknown,
		CreatedAt:  now,
	}, nil
}

func (c *Competency) Retired() bool { return c.RetiredAt != nil }

// LevelEvent é o registro append-only de 02-modelo-de-dados.md §4.3 — "o
// coração" do modelo. Nunca é editado; toda correção é um novo evento.
type LevelEvent struct {
	ID           string
	CompetencyID string
	UserID       string
	FromLevel    *int
	ToLevel      int
	Confidence   Confidence
	Source       LevelSource
	AssessmentID *string
	Rationale    string
	OccurredAt   time.Time
}

func NewLevelEvent(id, competencyID, userID string, fromLevel *int, toLevel int, confidence Confidence, source LevelSource, rationale string, now time.Time) (*LevelEvent, error) {
	if toLevel < MinLevel || toLevel > MaxLevel {
		return nil, newRuleError("", fmt.Sprintf("nível deve estar entre %d e %d", MinLevel, MaxLevel))
	}
	if !confidence.Valid() || confidence == ConfidenceUnknown {
		return nil, newRuleError("", "confiança inválida para um evento de nível")
	}
	if !source.Valid() {
		return nil, newRuleError("", fmt.Sprintf("origem inválida: %s", source))
	}
	rationale = strings.TrimSpace(rationale)
	if rationale == "" {
		// Nível que muda sem explicação é nível em que você não confia
		// (02-modelo-de-dados.md §4.3) — rationale nunca é decoração.
		return nil, newRuleError("", "rationale é obrigatório")
	}
	return &LevelEvent{
		ID:           id,
		CompetencyID: competencyID,
		UserID:       userID,
		FromLevel:    fromLevel,
		ToLevel:      toLevel,
		Confidence:   confidence,
		Source:       source,
		Rationale:    rationale,
		OccurredAt:   now,
	}, nil
}

// ApplyLevelEvent atualiza o cache da competência a partir de um evento já
// validado. Na Fatia 1 (sem agente) só as origens 'probe' e 'self' chegam
// aqui — RN-04 (duas avaliações concordantes) só vale para 'assessment',
// que entra na Fatia 4 junto do avaliador.
func (c *Competency) ApplyLevelEvent(ev LevelEvent) error {
	if ev.CompetencyID != c.ID {
		return newRuleError("", "evento não pertence a esta competência")
	}
	first := c.CurrentLevel == nil
	level := ev.ToLevel
	c.CurrentLevel = &level
	c.Confidence = ev.Confidence
	occurredAt := ev.OccurredAt
	c.LastEvidenceAt = &occurredAt
	// baseline_level é congelado na primeira vez que o nível deixa de ser
	// desconhecido — é o "de" em "de 2 para 4" (02-modelo-de-dados.md §4.2).
	if first {
		baseline := ev.ToLevel
		c.BaselineLevel = &baseline
	}
	return nil
}

// Delta é o número que importa no Painel de Delta (§7.1): nível atual menos
// baseline. nil quando qualquer um dos dois é desconhecido — nunca 0.
func (c *Competency) Delta() *int {
	if c.CurrentLevel == nil || c.BaselineLevel == nil {
		return nil
	}
	d := *c.CurrentLevel - *c.BaselineLevel
	return &d
}

// StaleDays alimenta o aviso de "não verificado" de RN-05. Sem evidência
// ainda, retorna nil — não há o que decair.
func (c *Competency) StaleDays(now time.Time) *int {
	if c.LastEvidenceAt == nil {
		return nil
	}
	days := int(now.Sub(*c.LastEvidenceAt).Hours() / 24)
	return &days
}
