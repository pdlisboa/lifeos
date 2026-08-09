// Package http implementa os handlers chi para os endpoints do módulo Metas
// usados na Fatia 1 (backend/api/openapi.yaml): goals, probe, competencies,
// track, action, sessions, evidence, delta, consistency.
package http

import (
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/app"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/modules/goals/packs"
)

func strPtr(s string) *string { return &s }

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func formatDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

// levelDescriptor resolve o texto do pack para um nível — nunca é copiado
// para o banco (04-agentes.md §4.6), então é resolvido aqui em tempo de
// leitura, por (packId, packKey, level).
func levelDescriptor(reg *packs.Registry, packID, packKey string, level *int) *string {
	if level == nil {
		return nil
	}
	p, ok := reg.Get(packID)
	if !ok {
		return nil
	}
	c, ok := p.CompetencyByKey(packKey)
	if !ok {
		return nil
	}
	d, ok := c.Levels[*level]
	if !ok {
		return nil
	}
	return &d
}

type goalSummaryDTO struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Archetype      string  `json:"archetype"`
	PackID         string  `json:"packId"`
	Status         string  `json:"status"`
	ScopeMode      string  `json:"scopeMode"`
	DaysActive     int     `json:"daysActive"`
	LastActivityAt *string `json:"lastActivityAt"`
}

func toGoalSummaryDTO(g domain.Goal) goalSummaryDTO {
	return goalSummaryDTO{
		ID:             g.ID,
		Title:          g.Title,
		Archetype:      string(g.Archetype),
		PackID:         g.PackID,
		Status:         string(g.Status),
		ScopeMode:      g.ScopeMode,
		DaysActive:     g.DaysActive(time.Now()),
		LastActivityAt: formatTimePtr(g.LastActivityAt),
	}
}

func toGoalSummaryDTOs(goals []domain.Goal) []goalSummaryDTO {
	out := make([]goalSummaryDTO, len(goals))
	for i, g := range goals {
		out[i] = toGoalSummaryDTO(g)
	}
	return out
}

type goalDTO struct {
	goalSummaryDTO
	Why              *string         `json:"why"`
	DefinitionOfDone *string         `json:"definitionOfDone"`
	HorizonOn        *string         `json:"horizonOn"`
	PausedUntil      *string         `json:"pausedUntil"`
	Competencies     []competencyDTO `json:"competencies"`
	ReadyToActivate  bool            `json:"readyToActivate"`
	Blockers         []string        `json:"blockers"`
}

func toGoalDTO(g *domain.Goal, comps []domain.Competency, reg *packs.Registry) goalDTO {
	blockers := g.Blockers(len(comps))
	compDTOs := make([]competencyDTO, len(comps))
	for i, c := range comps {
		compDTOs[i] = toCompetencyDTO(reg, g.PackID, c)
	}
	if blockers == nil {
		blockers = []string{}
	}
	return goalDTO{
		goalSummaryDTO:   toGoalSummaryDTO(*g),
		Why:              g.Why,
		DefinitionOfDone: g.DefinitionOfDone,
		HorizonOn:        formatDatePtr(g.HorizonOn),
		PausedUntil:      formatDatePtr(g.PausedUntil),
		Competencies:     compDTOs,
		ReadyToActivate:  g.ReadyToActivate(len(comps)),
		Blockers:         blockers,
	}
}

type competencyDTO struct {
	ID              string  `json:"id"`
	PackKey         string  `json:"packKey"`
	Label           string  `json:"label"`
	Weight          float64 `json:"weight"`
	Level           *int    `json:"level"`
	LevelDescriptor *string `json:"levelDescriptor"`
	BaselineLevel   *int    `json:"baselineLevel"`
	Confidence      string  `json:"confidence"`
	LastEvidenceAt  *string `json:"lastEvidenceAt"`
	StaleDays       *int    `json:"staleDays"`
	Retired         bool    `json:"retired"`
}

func toCompetencyDTO(reg *packs.Registry, packID string, c domain.Competency) competencyDTO {
	return competencyDTO{
		ID:              c.ID,
		PackKey:         c.PackKey,
		Label:           c.Label,
		Weight:          c.Weight,
		Level:           c.CurrentLevel,
		LevelDescriptor: levelDescriptor(reg, packID, c.PackKey, c.CurrentLevel),
		BaselineLevel:   c.BaselineLevel,
		Confidence:      string(c.Confidence),
		LastEvidenceAt:  formatTimePtr(c.LastEvidenceAt),
		StaleDays:       c.StaleDays(time.Now()),
		Retired:         c.Retired(),
	}
}

type levelEventDTO struct {
	ID           string  `json:"id"`
	FromLevel    *int    `json:"fromLevel"`
	ToLevel      int     `json:"toLevel"`
	Confidence   string  `json:"confidence"`
	Source       string  `json:"source"`
	Rationale    string  `json:"rationale"`
	AssessmentID *string `json:"assessmentId"`
	EvidenceID   *string `json:"evidenceId"`
	OccurredAt   string  `json:"occurredAt"`
}

func toLevelEventDTO(e domain.LevelEvent) levelEventDTO {
	return levelEventDTO{
		ID:           e.ID,
		FromLevel:    e.FromLevel,
		ToLevel:      e.ToLevel,
		Confidence:   string(e.Confidence),
		Source:       string(e.Source),
		Rationale:    e.Rationale,
		AssessmentID: e.AssessmentID,
		EvidenceID:   e.EvidenceID,
		OccurredAt:   e.OccurredAt.Format(time.RFC3339),
	}
}

type projectionDTO struct {
	Available      bool     `json:"available"`
	Reason         *string  `json:"reason"`
	MinutesPerWeek *float64 `json:"minutesPerWeek"`
	NextMilestone  *string  `json:"nextMilestone"`
	WeeksToNextMin *int     `json:"weeksToNextMin"`
	WeeksToNextMax *int     `json:"weeksToNextMax"`
	IfYouDouble    *string  `json:"ifYouDouble"`
}

func toProjectionDTO(p domain.Projection) projectionDTO {
	return projectionDTO{
		Available:      p.Available,
		Reason:         p.Reason,
		MinutesPerWeek: p.MinutesPerWeek,
		NextMilestone:  p.NextMilestone,
		WeeksToNextMin: p.WeeksToNextMin,
		WeeksToNextMax: p.WeeksToNextMax,
		IfYouDouble:    p.IfYouDouble,
	}
}

type deltaCompetencyDTO struct {
	competencyDTO
	Delta           *int `json:"delta"`
	DeltaWindowDays *int `json:"deltaWindowDays"`
	RisesLast90d    int  `json:"risesLast90d"`
}

type totalsDTO struct {
	EvidenceCount  int `json:"evidenceCount"`
	MilestonesDone int `json:"milestonesDone"`
	SessionMinutes int `json:"sessionMinutes"`
}

type deltaPanelDTO struct {
	GoalID       string               `json:"goalId"`
	DaysActive   int                  `json:"daysActive"`
	Competencies []deltaCompetencyDTO `json:"competencies"`
	Totals       totalsDTO            `json:"totals"`
	Headline     *string              `json:"headline"`
}

func toDeltaPanelDTO(reg *packs.Registry, d *app.DeltaResult) deltaPanelDTO {
	now := time.Now()
	comps := make([]deltaCompetencyDTO, 0, len(d.Competencies))
	for _, c := range d.Competencies {
		if c.Retired() {
			continue
		}
		var windowDays *int
		if since, ok := d.BaselineSetAt[c.ID]; ok {
			days := int(now.Sub(since).Hours() / 24)
			windowDays = &days
		}
		comps = append(comps, deltaCompetencyDTO{
			competencyDTO:   toCompetencyDTO(reg, d.Goal.PackID, c),
			Delta:           c.Delta(),
			DeltaWindowDays: windowDays,
			RisesLast90d:    d.RisesLast90d[c.ID],
		})
	}
	return deltaPanelDTO{
		GoalID:       d.Goal.ID,
		DaysActive:   d.Goal.DaysActive(now),
		Competencies: comps,
		Totals: totalsDTO{
			EvidenceCount:  d.EvidenceCount,
			MilestonesDone: d.MilestonesDone,
			SessionMinutes: d.SessionMinutes,
		},
		Headline: nil, // sem agente na Fatia 1 — degrada honestamente para "sem relato" (04-agentes.md §5)
	}
}

type consistencyDTO struct {
	ActiveDays int    `json:"activeDays"`
	WindowDays int    `json:"windowDays"`
	TodayDone  bool   `json:"todayDone"`
	Label      string `json:"label"`
}

func toConsistencyDTO(w domain.ConsistencyWindow) consistencyDTO {
	return consistencyDTO{
		ActiveDays: w.ActiveDays,
		WindowDays: w.WindowDays,
		TodayDone:  w.TodayDone,
		Label:      w.Label(),
	}
}

type todayGoalDTO struct {
	Goal      goalSummaryDTO `json:"goal"`
	Action    nextActionDTO  `json:"action"`
	RecentWin *string        `json:"recentWin"`
}

type todayDTO struct {
	Goals            []todayGoalDTO `json:"goals"`
	Nudge            any            `json:"nudge"`
	Consistency      consistencyDTO `json:"consistency"`
	PendingProposals int            `json:"pendingProposals"`
}

func toTodayDTO(r *app.TodayResult) todayDTO {
	items := make([]todayGoalDTO, len(r.Goals))
	for i, g := range r.Goals {
		items[i] = todayGoalDTO{
			Goal:      toGoalSummaryDTO(g.Goal),
			Action:    toNextActionDTO(g.Action),
			RecentWin: g.RecentWin,
		}
	}
	return todayDTO{
		Goals:            items,
		Nudge:            nil, // sem agente na Fatia 1 — nudge chega na Fatia 6 (04-agentes.md §4.7)
		Consistency:      toConsistencyDTO(r.Consistency),
		PendingProposals: 0,
	}
}

type probeTurnDTO struct {
	ID            string  `json:"id"`
	Ordinal       int     `json:"ordinal"`
	Question      string  `json:"question"`
	Answer        *string `json:"answer"`
	CompetencyID  *string `json:"competencyId"`
	InferredLevel *int    `json:"inferredLevel"`
	AnsweredAt    *string `json:"answeredAt"`
}

func toProbeTurnDTO(t domain.ProbeTurn) probeTurnDTO {
	return probeTurnDTO{
		ID:            t.ID,
		Ordinal:       t.Ordinal,
		Question:      t.Question,
		Answer:        t.Answer,
		CompetencyID:  t.CompetencyID,
		InferredLevel: t.InferredLevel,
		AnsweredAt:    formatTimePtr(t.AnsweredAt),
	}
}

type probeDTO struct {
	ID              string         `json:"id"`
	Status          string         `json:"status"`
	AskedCount      int            `json:"askedCount"`
	MaxQuestions    int            `json:"maxQuestions"`
	CurrentQuestion *probeTurnDTO  `json:"currentQuestion"`
	Turns           []probeTurnDTO `json:"turns"`
	CanSkip         bool           `json:"canSkip"`
}

func toProbeDTO(p *domain.Probe) probeDTO {
	turns := make([]probeTurnDTO, len(p.Turns))
	for i, t := range p.Turns {
		turns[i] = toProbeTurnDTO(t)
	}
	var current *probeTurnDTO
	if c := p.Current(); c != nil {
		d := toProbeTurnDTO(*c)
		current = &d
	}
	return probeDTO{
		ID:              p.ID,
		Status:          string(p.Status),
		AskedCount:      p.AskedCount,
		MaxQuestions:    domain.MaxProbeQuestions,
		CurrentQuestion: current,
		Turns:           turns,
		CanSkip:         true,
	}
}

type milestoneDTO struct {
	ID                 string   `json:"id"`
	Ordinal            int      `json:"ordinal"`
	Title              string   `json:"title"`
	CompletionCriteria string   `json:"completionCriteria"`
	Status             string   `json:"status"`
	CompletedAt        *string  `json:"completedAt"`
	CompetencyIDs      []string `json:"competencyIds"`
}

type trackDTO struct {
	ID         string         `json:"id"`
	Version    int            `json:"version"`
	Milestones []milestoneDTO `json:"milestones"`
}

func toTrackDTO(t *domain.Track) trackDTO {
	milestones := make([]milestoneDTO, len(t.Milestones))
	for i, m := range t.Milestones {
		ids := m.CompetencyIDs
		if ids == nil {
			ids = []string{}
		}
		milestones[i] = milestoneDTO{
			ID:                 m.ID,
			Ordinal:            m.Ordinal,
			Title:              m.Title,
			CompletionCriteria: m.CompletionCriteria,
			Status:             string(m.Status),
			CompletedAt:        formatTimePtr(m.CompletedAt),
			CompetencyIDs:      ids,
		}
	}
	return trackDTO{ID: t.ID, Version: t.Version, Milestones: milestones}
}

type nextActionDTO struct {
	ID             string  `json:"id"`
	GoalID         string  `json:"goalId"`
	Title          string  `json:"title"`
	Detail         *string `json:"detail"`
	PracticeFormat *string `json:"practiceFormat"`
	EstimatedMin   int     `json:"estimatedMin"`
	MinimalVariant *string `json:"minimalVariant"`
	CompetencyID   *string `json:"competencyId"`
	MilestoneID    *string `json:"milestoneId"`
	GeneratedBy    string  `json:"generatedBy"`
	OriginKind     *string `json:"originKind"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"createdAt"`
}

func toNextActionDTO(a *domain.NextAction) nextActionDTO {
	var origin *string
	if a.OriginKind != nil {
		o := string(*a.OriginKind)
		origin = &o
	}
	return nextActionDTO{
		ID:             a.ID,
		GoalID:         a.GoalID,
		Title:          a.Title,
		Detail:         a.Detail,
		PracticeFormat: a.PracticeFormat,
		EstimatedMin:   a.EstimatedMin,
		MinimalVariant: strPtr(a.MinimalVariant),
		CompetencyID:   a.CompetencyID,
		MilestoneID:    a.MilestoneID,
		GeneratedBy:    string(a.GeneratedBy),
		OriginKind:     origin,
		Status:         string(a.Status),
		CreatedAt:      a.CreatedAt.Format(time.RFC3339),
	}
}

type sessionDTO struct {
	ID          string  `json:"id"`
	StartedAt   string  `json:"startedAt"`
	DurationMin int     `json:"durationMin"`
	LocalOn     string  `json:"localOn"`
	Note        *string `json:"note"`
	Energy      *int    `json:"energy"`
}

func toSessionDTO(s domain.Session) sessionDTO {
	return sessionDTO{
		ID:          s.ID,
		StartedAt:   s.StartedAt.Format(time.RFC3339),
		DurationMin: s.DurationMin,
		LocalOn:     s.LocalOn.Format("2006-01-02"),
		Note:        s.Note,
		Energy:      s.Energy,
	}
}

type evidenceDTO struct {
	ID           string  `json:"id"`
	Kind         string  `json:"kind"`
	Title        *string `json:"title"`
	Body         *string `json:"body"`
	BlobURL      *string `json:"blobUrl"`
	ExternalURL  *string `json:"externalUrl"`
	Language     *string `json:"language"`
	SupersedesID *string `json:"supersedesId"`
	CreatedAt    string  `json:"createdAt"`
}

func toEvidenceDTO(e *domain.Evidence) evidenceDTO {
	return evidenceDTO{
		ID:           e.ID,
		Kind:         string(e.Kind),
		Title:        e.Title,
		Body:         e.Body,
		ExternalURL:  e.ExternalURL,
		Language:     e.Language,
		SupersedesID: e.SupersedesID,
		CreatedAt:    e.CreatedAt.Format(time.RFC3339),
	}
}

type evidenceCardDTO struct {
	evidenceDTO
	CompetencyIDs     []string       `json:"competencyIds"`
	AssessmentSummary *string        `json:"assessmentSummary"`
	LevelsAtTime      map[string]int `json:"levelsAtTime"`
}

// toEvidenceCardDTO monta o cartão do museu (§7.4). AssessmentSummary
// continua nil — depende do agente avaliador (Fatia 4). CompetencyIDs e
// LevelsAtTime já não dependem de agente nenhum: a tag de competência é
// manual (§7 da UX) e o nível point-in-time vem direto do histórico de
// eventos, que existe desde a Fatia 1.
func toEvidenceCardDTO(c app.EvidenceCard) evidenceCardDTO {
	return evidenceCardDTO{
		evidenceDTO:       toEvidenceDTO(&c.Evidence),
		CompetencyIDs:     c.CompetencyIDs,
		AssessmentSummary: nil,
		LevelsAtTime:      c.LevelsAtTime,
	}
}

type pageDTO[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"nextCursor"`
	HasMore    bool    `json:"hasMore"`
}
