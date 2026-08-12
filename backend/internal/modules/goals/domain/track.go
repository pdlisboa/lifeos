package domain

import (
	"sort"
	"time"
)

type MilestoneStatus string

const (
	MilestoneLocked    MilestoneStatus = "locked"
	MilestoneCurrent   MilestoneStatus = "current"
	MilestoneCompleted MilestoneStatus = "completed"
	MilestoneSkipped   MilestoneStatus = "skipped"
)

type Milestone struct {
	ID                 string
	TrackID            string
	GoalID             string
	UserID             string
	Ordinal            int
	Title              string
	CompletionCriteria string
	Status             MilestoneStatus
	CompletedAt        *time.Time
	CompetencyIDs      []string
}

func (m *Milestone) Complete(now time.Time) error {
	if m.Status != MilestoneCurrent {
		return newRuleError("", "só o marco atual pode ser concluído")
	}
	m.Status = MilestoneCompleted
	m.CompletedAt = &now
	return nil
}

type Track struct {
	ID     string
	GoalID string
	UserID string

	// Version, GeneratedBy: na Fatia 1 a trilha nasce sempre 'user' — é o
	// fallback determinístico da milestone_library, não o A1 (04-agentes.md §5).
	Version      int
	GeneratedBy  string
	AcceptedAt   *time.Time
	SupersededAt *time.Time
	Milestones   []Milestone
}

// MilestoneSpec é o suficiente da milestone_library do pack para montar uma
// trilha, sem o domínio depender do pacote packs (regra de fronteira: domain
// não importa nada do projeto).
type MilestoneSpec struct {
	Title              string
	CompletionCriteria string
	TypicalLevel       int
	CompetencyKeys     []string
}

// BuildFallbackTrack monta a trilha direto da milestone_library do pack,
// ordenada por typical_level — o fallback de A1 quando não há agente
// (04-agentes.md §5, §4.7). O primeiro marco nasce 'current', o resto
// 'locked'; IDs e chaves estrangeiras são preenchidos por quem persiste.
func BuildFallbackTrack(specs []MilestoneSpec, competencyIDByKey map[string]string) []Milestone {
	ordered := make([]MilestoneSpec, len(specs))
	copy(ordered, specs)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].TypicalLevel < ordered[j].TypicalLevel
	})

	milestones := make([]Milestone, 0, len(ordered))
	for i, spec := range ordered {
		var ids []string
		for _, key := range spec.CompetencyKeys {
			if id, ok := competencyIDByKey[key]; ok {
				ids = append(ids, id)
			}
		}
		milestones = append(milestones, Milestone{
			Ordinal:            i + 1,
			Title:              spec.Title,
			CompletionCriteria: spec.CompletionCriteria,
			Status:             MilestoneLocked,
			CompetencyIDs:      ids,
		})
	}
	ApplyTrackStatuses(milestones)
	return milestones
}

// ApplyTrackStatuses marca o primeiro marco ainda não concluído/pulado como
// 'current' e todo o resto como 'locked', sem tocar em marcos já
// 'completed'/'skipped'. Usado tanto pelo fallback determinístico quanto
// pela trilha aplicada ao aceitar uma proposta do A1 — marcos concluídos são
// intocáveis numa revisão (04-agentes.md §4.1).
func ApplyTrackStatuses(milestones []Milestone) {
	foundCurrent := false
	for i := range milestones {
		switch milestones[i].Status {
		case MilestoneCompleted, MilestoneSkipped:
			continue
		default:
			if !foundCurrent {
				milestones[i].Status = MilestoneCurrent
				foundCurrent = true
			} else {
				milestones[i].Status = MilestoneLocked
			}
		}
	}
}

// FirstOpen devolve o primeiro marco ainda não concluído/pulado — é o que o
// fallback de A2 usa como alvo (04-agentes.md §5: "próximo marco não
// concluído → primeiro formato de prática compatível → example").
func FirstOpen(milestones []Milestone) *Milestone {
	for i := range milestones {
		if milestones[i].Status == MilestoneCurrent || milestones[i].Status == MilestoneLocked {
			return &milestones[i]
		}
	}
	return nil
}
