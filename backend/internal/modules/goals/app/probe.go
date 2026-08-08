package app

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/modules/goals/packs"
	"github.com/phablo/lifeos/internal/platform/idgen"
)

func (s *Service) GetProbe(ctx context.Context, userID, goalID string) (*domain.Probe, error) {
	if _, err := postgres.GetGoal(ctx, s.Pool, userID, goalID); err != nil {
		return nil, err
	}
	return postgres.GetProbeByGoal(ctx, s.Pool, goalID)
}

type AnswerProbeInput struct {
	UserID string
	GoalID string
	TurnID string
	Answer string
}

type ProbeStepResult struct {
	Probe        *domain.Probe
	NextQuestion *domain.ProbeTurn
}

// AnswerProbe é o percorredor estático de 04-agentes.md §4.7: sem LLM, ele
// segue os probe_seeds do pack na ordem, usando o follow-up de sim/não
// quando a resposta bate com a heurística léxica do domínio. O que não foi
// perguntado fica desconhecido — nunca vira nível 0.
func (s *Service) AnswerProbe(ctx context.Context, in AnswerProbeInput) (*ProbeStepResult, error) {
	var result ProbeStepResult
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		g, err := postgres.GetGoal(ctx, tx, in.UserID, in.GoalID)
		if err != nil {
			return err
		}
		probe, err := postgres.GetProbeByGoal(ctx, tx, g.ID)
		if err != nil {
			return err
		}

		now := time.Now()
		answered, err := probe.Answer(in.TurnID, in.Answer, now)
		if err != nil {
			return err
		}
		if err := postgres.UpdateProbeTurnAnswer(ctx, tx, answered); err != nil {
			return err
		}

		pack, err := s.pack(g.PackID)
		if err != nil {
			return err
		}

		var next *domain.ProbeTurn
		if step := nextProbeStep(pack.ProbeSeeds, probe.Turns); step != nil {
			comps, err := postgres.ListCompetenciesByGoal(ctx, tx, g.ID)
			if err != nil {
				return err
			}
			var compID *string
			for _, c := range comps {
				if c.PackKey == step.CompetencyKey {
					id := c.ID
					compID = &id
				}
			}
			turnID, err := idgen.NewUUIDv7()
			if err != nil {
				return err
			}
			turn := domain.ProbeTurn{ID: turnID, ProbeID: probe.ID, CompetencyID: compID, Question: step.Question}
			if err := probe.AskTurn(turn); err != nil {
				return err
			}
			newTurn := &probe.Turns[len(probe.Turns)-1]
			if err := postgres.InsertProbeTurn(ctx, tx, newTurn); err != nil {
				return err
			}
			if err := postgres.UpdateProbe(ctx, tx, probe); err != nil {
				return err
			}
			next = newTurn
		} else {
			if err := probe.Close(domain.ProbeCompleted, now); err != nil {
				return err
			}
			if err := postgres.UpdateProbe(ctx, tx, probe); err != nil {
				return err
			}
		}

		result = ProbeStepResult{Probe: probe, NextQuestion: next}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) SkipProbe(ctx context.Context, userID, goalID string) (*domain.Probe, error) {
	var probe *domain.Probe
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		g, err := postgres.GetGoal(ctx, tx, userID, goalID)
		if err != nil {
			return err
		}
		p, err := postgres.GetProbeByGoal(ctx, tx, g.ID)
		if err != nil {
			return err
		}
		if err := p.Close(domain.ProbeSkipped, time.Now()); err != nil {
			return err
		}
		if err := postgres.UpdateProbe(ctx, tx, p); err != nil {
			return err
		}
		probe = p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return probe, nil
}

type probeStep struct {
	Question      string
	CompetencyKey string
}

// nextProbeStep reconstrói, a partir dos turnos já feitos, qual é a próxima
// pergunta da sondagem estática — sem guardar nenhum estado extra além dos
// próprios turnos. Assume que os turnos foram gerados exatamente por esta
// função (main, [follow-up], main, [follow-up]...), o que é garantido porque
// é o único ponto do sistema que cria um ProbeTurn novo.
func nextProbeStep(seeds []packs.ProbeSeed, turns []domain.ProbeTurn) *probeStep {
	ti := 0
	for _, seed := range seeds {
		if ti >= len(turns) {
			return &probeStep{Question: seed.Question, CompetencyKey: seed.Competency}
		}
		mainTurn := turns[ti]
		if mainTurn.Answer == nil {
			return nil // ainda esperando resposta do turno atual
		}
		ti++

		var followup string
		if domain.AnsweredYes(*mainTurn.Answer) {
			followup = seed.Followups["yes"]
		} else if domain.AnsweredNo(*mainTurn.Answer) {
			followup = seed.Followups["no"]
		}
		if followup == "" {
			continue
		}
		if ti >= len(turns) {
			return &probeStep{Question: followup, CompetencyKey: seed.Competency}
		}
		followupTurn := turns[ti]
		if followupTurn.Answer == nil {
			return nil
		}
		ti++
	}
	return nil // sementes esgotadas — teto de 5 também é respeitado por AskTurn
}
