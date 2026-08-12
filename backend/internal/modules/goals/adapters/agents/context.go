package agents

import (
	"strings"

	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/modules/goals/packs"
)

// BuildPlannerContext monta o contexto de planner.v1.md a partir do estado
// já carregado do banco. existingTrack nil sinaliza criação; não-nil,
// revisão (04-agentes.md §4.1).
func BuildPlannerContext(g *domain.Goal, pack packs.Pack, comps []domain.Competency, existingTrack *domain.Track, revisionReason string) PlannerContext {
	compViews, unknown := competencyViews(pack, comps)

	libItems := make([]PlannerMilestoneLibraryItem, 0, len(pack.MilestoneLibrary))
	for _, m := range pack.MilestoneLibrary {
		libItems = append(libItems, PlannerMilestoneLibraryItem{
			Title:        m.Title,
			TypicalLevel: m.TypicalLevel,
			Competencies: strings.Join(m.Competencies, ", "),
			Criteria:     m.Criteria,
		})
	}

	var existing *PlannerExistingTrack
	if existingTrack != nil {
		ms := make([]PlannerExistingMilestone, 0, len(existingTrack.Milestones))
		for _, m := range existingTrack.Milestones {
			ms = append(ms, PlannerExistingMilestone{
				Ordinal:            m.Ordinal,
				Status:             string(m.Status),
				Title:              m.Title,
				CompletionCriteria: m.CompletionCriteria,
			})
		}
		existing = &PlannerExistingTrack{Milestones: ms}
	}

	return PlannerContext{
		Goal:           PlannerGoalView{Title: g.Title, Why: derefStr(g.Why), DefinitionOfDone: derefStr(g.DefinitionOfDone)},
		Pack:           PlannerPackView{Label: pack.Label, MilestoneLibrary: libItems},
		Competencies:   compViews,
		UnknownCount:   unknown,
		ExistingTrack:  existing,
		RevisionReason: revisionReason,
	}
}

// BuildPracticeContext monta o contexto de practice.v1.md. recent já vem
// pronto (título/status/motivo de skip) — é o que ListRecentActionsByGoal
// devolve, sem precisar hidratar domain.NextAction inteiro só pra isso.
func BuildPracticeContext(g *domain.Goal, pack packs.Pack, milestone *domain.Milestone, comps []domain.Competency, recent []RecentActionView, difficultyHint, originKind string) PracticeContext {
	compViews, _ := competencyViews(pack, comps)

	var ms *PracticeMilestoneView
	if milestone != nil {
		ms = &PracticeMilestoneView{
			Title:              milestone.Title,
			CompletionCriteria: milestone.CompletionCriteria,
			CompetencyKeys:     strings.Join(milestoneCompetencyKeys(*milestone, comps), ", "),
		}
	}

	formats := make([]PracticeFormatView, 0, len(pack.PracticeFormats))
	for _, f := range pack.PracticeFormats {
		formats = append(formats, PracticeFormatView{
			Key: f.Key, Label: f.Label, Minutes: f.Minutes,
			GoodFor: strings.Join(f.GoodFor, ", "), Shape: f.Shape,
		})
	}

	titles := make([]string, 0, len(recent))
	for _, r := range recent {
		titles = append(titles, r.Title)
	}

	return PracticeContext{
		Goal:           PracticeGoalView{Title: g.Title, DefinitionOfDone: derefStr(g.DefinitionOfDone), ScopeMode: g.ScopeMode},
		Milestone:      ms,
		Competencies:   compViews,
		Pack:           PracticePackView{Label: pack.Label, PracticeFormats: formats},
		RecentActions:  recent,
		DifficultyHint: difficultyHint,
		OriginKind:     originKind,
		RecentTitles:   titles,
	}
}

func competencyViews(pack packs.Pack, comps []domain.Competency) ([]CompetencyView, int) {
	views := make([]CompetencyView, 0, len(comps))
	unknown := 0
	for _, c := range comps {
		var descriptor *string
		if c.CurrentLevel == nil {
			unknown++
		} else if pc, ok := pack.CompetencyByKey(c.PackKey); ok {
			if d, ok := pc.Levels[*c.CurrentLevel]; ok {
				descriptor = &d
			}
		}
		views = append(views, CompetencyView{
			PackKey: c.PackKey, Label: c.Label, Weight: c.Weight,
			Level: c.CurrentLevel, LevelDescriptor: descriptor, Confidence: string(c.Confidence),
		})
	}
	return views, unknown
}

func milestoneCompetencyKeys(m domain.Milestone, comps []domain.Competency) []string {
	byID := make(map[string]string, len(comps))
	for _, c := range comps {
		byID[c.ID] = c.PackKey
	}
	keys := make([]string, 0, len(m.CompetencyIDs))
	for _, id := range m.CompetencyIDs {
		if k, ok := byID[id]; ok {
			keys = append(keys, k)
		}
	}
	return keys
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
