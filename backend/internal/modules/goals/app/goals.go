package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/modules/goals/packs"
	"github.com/phablo/lifeos/internal/platform/idgen"
)

type CreateGoalInput struct {
	UserID    string
	Title     string
	Archetype domain.GoalArchetype
	PackID    string
	Why       *string
}

type CreateGoalResult struct {
	Goal         *domain.Goal
	Competencies []domain.Competency
	Probe        *domain.Probe
}

// CreateGoal materializa as competências do pack (04-agentes.md §4.6, todas
// com nível NULL) e abre a sondagem com a primeira pergunta já pronta — a
// fricção na criação é o que mata o entusiasmo (D2).
func (s *Service) CreateGoal(ctx context.Context, in CreateGoalInput) (*CreateGoalResult, error) {
	pack, err := s.pack(in.PackID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	goalID, err := idgen.NewUUIDv7()
	if err != nil {
		return nil, err
	}
	g, err := domain.NewGoal(goalID, in.UserID, in.Title, in.Archetype, in.PackID, in.Why, now)
	if err != nil {
		return nil, err
	}

	comps := make([]domain.Competency, 0, len(pack.Competencies))
	keyToID := make(map[string]string, len(pack.Competencies))
	for _, pc := range pack.Competencies {
		cid, err := idgen.NewUUIDv7()
		if err != nil {
			return nil, err
		}
		c, err := domain.NewCompetency(cid, goalID, in.UserID, pc.Key, pc.Label, pc.Weight, now)
		if err != nil {
			return nil, err
		}
		comps = append(comps, *c)
		keyToID[pc.Key] = cid
	}

	probeID, err := idgen.NewUUIDv7()
	if err != nil {
		return nil, err
	}
	probe := domain.NewProbe(probeID, goalID, in.UserID)
	if len(pack.ProbeSeeds) > 0 {
		seed := pack.ProbeSeeds[0]
		turnID, err := idgen.NewUUIDv7()
		if err != nil {
			return nil, err
		}
		var compID *string
		if id, ok := keyToID[seed.Competency]; ok {
			compID = &id
		}
		if err := probe.AskTurn(domain.ProbeTurn{ID: turnID, ProbeID: probeID, CompetencyID: compID, Question: seed.Question}); err != nil {
			return nil, err
		}
	} else {
		if err := probe.Close(domain.ProbeCompleted, now); err != nil {
			return nil, err
		}
	}

	err = s.withTx(ctx, func(tx pgx.Tx) error {
		if err := postgres.InsertGoal(ctx, tx, g); err != nil {
			return err
		}
		for i := range comps {
			if err := postgres.InsertCompetency(ctx, tx, &comps[i]); err != nil {
				return err
			}
		}
		if err := postgres.InsertProbe(ctx, tx, probe); err != nil {
			return err
		}
		for i := range probe.Turns {
			if err := postgres.InsertProbeTurn(ctx, tx, &probe.Turns[i]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &CreateGoalResult{Goal: g, Competencies: comps, Probe: probe}, nil
}

type GoalDetail struct {
	Goal         *domain.Goal
	Competencies []domain.Competency
}

func (s *Service) GetGoal(ctx context.Context, userID, goalID string) (*GoalDetail, error) {
	g, err := postgres.GetGoal(ctx, s.Pool, userID, goalID)
	if err != nil {
		return nil, err
	}
	comps, err := postgres.ListCompetenciesByGoal(ctx, s.Pool, g.ID)
	if err != nil {
		return nil, err
	}
	return &GoalDetail{Goal: g, Competencies: comps}, nil
}

func (s *Service) ListGoals(ctx context.Context, userID string, statuses []string) ([]domain.Goal, error) {
	return postgres.ListGoals(ctx, s.Pool, userID, statuses)
}

type PatchGoalInput struct {
	Title            *string
	Why              *string
	DefinitionOfDone *string
	HorizonOn        *time.Time
}

func (s *Service) PatchGoal(ctx context.Context, userID, goalID string, in PatchGoalInput) (*domain.Goal, error) {
	var g *domain.Goal
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		g, err = postgres.GetGoal(ctx, tx, userID, goalID)
		if err != nil {
			return err
		}
		if in.Title != nil {
			title := strings.TrimSpace(*in.Title)
			if len(title) < 3 || len(title) > 120 {
				return domain.NewRuleError("", "título deve ter entre 3 e 120 caracteres")
			}
			g.Title = title
		}
		if in.Why != nil {
			g.Why = in.Why
		}
		if in.DefinitionOfDone != nil {
			g.DefinitionOfDone = in.DefinitionOfDone
		}
		if in.HorizonOn != nil {
			g.HorizonOn = in.HorizonOn
		}
		g.UpdatedAt = time.Now()
		return postgres.UpdateGoal(ctx, tx, g)
	})
	if err != nil {
		return nil, err
	}
	return g, nil
}

type ActivateGoalInput struct {
	UserID      string
	GoalID      string
	PauseGoalID *string
}

type ActivateGoalResult struct {
	Goal   *domain.Goal
	Action *domain.NextAction
}

// ActivateGoal cobre RN-01 (definição de pronto + competências), RN-02
// (máx. 3 ativas, com a saída de pausar outra na mesma transação) e RN-03
// (a meta nunca fica ativa sem uma próxima ação — aqui, o fallback
// determinístico da milestone_library, já que não há agente na Fatia 1).
func (s *Service) ActivateGoal(ctx context.Context, in ActivateGoalInput) (*ActivateGoalResult, error) {
	var result ActivateGoalResult
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		// serializa ativações concorrentes do mesmo usuário — RN-02 precisa
		// que "contar ativas" e "ativar mais uma" sejam atômicos.
		if err := postgres.LockUser(ctx, tx, in.UserID); err != nil {
			return err
		}

		g, err := postgres.GetGoal(ctx, tx, in.UserID, in.GoalID)
		if err != nil {
			return err
		}

		if in.PauseGoalID != nil {
			other, err := postgres.GetGoal(ctx, tx, in.UserID, *in.PauseGoalID)
			if err != nil {
				return fmt.Errorf("meta a pausar: %w", err)
			}
			reviewOn := time.Now().AddDate(0, 0, 14)
			if err := other.Pause(reviewOn, time.Now()); err != nil {
				return err
			}
			if err := postgres.UpdateGoal(ctx, tx, other); err != nil {
				return err
			}
		}

		activeCount, err := postgres.CountActiveGoals(ctx, tx, in.UserID)
		if err != nil {
			return err
		}
		if activeCount >= domain.MaxActiveGoals {
			active, err := postgres.ListGoals(ctx, tx, in.UserID, []string{"active", "at_risk", "stagnant"})
			if err != nil {
				return err
			}
			return &MaxActiveGoalsError{ActiveGoals: active}
		}

		compCount, err := postgres.CountCompetenciesByGoal(ctx, tx, in.GoalID)
		if err != nil {
			return err
		}

		now := time.Now()
		if err := g.Activate(compCount, now); err != nil {
			return err
		}
		if err := postgres.UpdateGoal(ctx, tx, g); err != nil {
			return err
		}

		// só monta a trilha do pack se ainda não existir uma — uma proposta
		// do A1 pode já ter sido aceita antes da ativação (sondagem fechou
		// com a meta em rascunho), e o fallback não deve apagar isso.
		if _, err := postgres.GetCurrentTrack(ctx, tx, g.ID); errors.Is(err, postgres.ErrNotFound) {
			if err := s.buildFallbackTrack(ctx, tx, g); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		action, err := s.generateNextAction(ctx, tx, g, now, nil)
		if err != nil {
			return err
		}
		if err := enqueueGenerateNextAction(ctx, tx, g.ID, g.UserID, action.ID); err != nil {
			return err
		}

		result = ActivateGoalResult{Goal: g, Action: action}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// buildFallbackTrack monta a trilha direto da milestone_library do pack
// (04-agentes.md §5, §4.7) — o fallback de A1 quando não há agente.
func (s *Service) buildFallbackTrack(ctx context.Context, tx pgx.Tx, g *domain.Goal) error {
	pack, err := s.pack(g.PackID)
	if err != nil {
		return err
	}
	comps, err := postgres.ListCompetenciesByGoal(ctx, tx, g.ID)
	if err != nil {
		return err
	}
	keyToID := make(map[string]string, len(comps))
	for _, c := range comps {
		keyToID[c.PackKey] = c.ID
	}

	specs := make([]domain.MilestoneSpec, 0, len(pack.MilestoneLibrary))
	for _, m := range pack.MilestoneLibrary {
		specs = append(specs, domain.MilestoneSpec{
			Title:              m.Title,
			CompletionCriteria: m.Criteria,
			TypicalLevel:       m.TypicalLevel,
			CompetencyKeys:     m.Competencies,
		})
	}
	milestones := domain.BuildFallbackTrack(specs, keyToID)

	trackID, err := idgen.NewUUIDv7()
	if err != nil {
		return err
	}
	track := &domain.Track{ID: trackID, GoalID: g.ID, UserID: g.UserID, Version: 1, GeneratedBy: "user"}
	if err := postgres.InsertTrack(ctx, tx, track); err != nil {
		return err
	}

	for i := range milestones {
		mid, err := idgen.NewUUIDv7()
		if err != nil {
			return err
		}
		milestones[i].ID = mid
		milestones[i].TrackID = trackID
		milestones[i].GoalID = g.ID
		milestones[i].UserID = g.UserID
		if err := postgres.InsertMilestone(ctx, tx, &milestones[i]); err != nil {
			return err
		}
		for _, cid := range milestones[i].CompetencyIDs {
			if err := postgres.InsertMilestoneCompetency(ctx, tx, mid, cid); err != nil {
				return err
			}
		}
	}
	return nil
}

// generateNextAction é o fallback de A2 (04-agentes.md §5): próximo marco
// não concluído → primeiro formato de prática compatível → example do
// formato. difficultyHint vem preenchido quando a geração é motivada por um
// skip (§7.3).
func (s *Service) generateNextAction(ctx context.Context, tx pgx.Tx, g *domain.Goal, now time.Time, difficultyHint *string) (*domain.NextAction, error) {
	pack, err := s.pack(g.PackID)
	if err != nil {
		return nil, err
	}
	track, err := postgres.GetCurrentTrack(ctx, tx, g.ID)
	if err != nil {
		return nil, fmt.Errorf("buscar trilha para gerar ação: %w", err)
	}
	comps, err := postgres.ListCompetenciesByGoal(ctx, tx, g.ID)
	if err != nil {
		return nil, err
	}

	current := domain.FirstOpen(track.Milestones)
	action, err := fallbackAction(g, pack, current, comps, now)
	if err != nil {
		return nil, err
	}
	action.DifficultyHint = difficultyHint

	if err := postgres.InsertNextAction(ctx, tx, action); err != nil {
		return nil, err
	}
	return action, nil
}

func fallbackAction(g *domain.Goal, pack packs.Pack, milestone *domain.Milestone, comps []domain.Competency, now time.Time) (*domain.NextAction, error) {
	if milestone == nil {
		return nil, domain.NewRuleError("", "trilha sem marco disponível para gerar a próxima ação")
	}
	comp, ok := firstMappedCompetency(milestone, comps)
	if !ok {
		return nil, domain.NewRuleError("", "marco sem competência mapeada para gerar a próxima ação")
	}
	format, ok := pack.PracticeFormatFor(comp.PackKey)
	if !ok {
		return nil, domain.NewRuleError("", "pack sem formato de prática para "+comp.PackKey)
	}

	estimated := format.Minutes[1]
	if estimated < domain.MinEstimatedMin {
		estimated = domain.MinEstimatedMin
	}
	if estimated > domain.MaxEstimatedMin {
		estimated = domain.MaxEstimatedMin
	}
	minimal := fmt.Sprintf("Versão de 5 min: releia %q e escreva só o primeiro passo, sem terminar.", format.Example)

	actionID, err := idgen.NewUUIDv7()
	if err != nil {
		return nil, err
	}
	a, err := domain.NewNextAction(actionID, g.ID, g.UserID, format.Example, estimated, minimal, domain.GeneratedByFallback, now)
	if err != nil {
		return nil, err
	}
	shape := format.Shape
	a.Detail = &shape
	a.MilestoneID = &milestone.ID
	a.CompetencyID = &comp.ID
	pf := format.Key
	a.PracticeFormat = &pf
	origin := domain.OriginPractice
	a.OriginKind = &origin
	return a, nil
}

func firstMappedCompetency(milestone *domain.Milestone, comps []domain.Competency) (domain.Competency, bool) {
	for _, id := range milestone.CompetencyIDs {
		for _, c := range comps {
			if c.ID == id {
				return c, true
			}
		}
	}
	return domain.Competency{}, false
}
