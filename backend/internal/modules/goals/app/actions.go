package app

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/platform/idgen"
)

func (s *Service) GetPendingAction(ctx context.Context, userID, goalID string) (*domain.NextAction, error) {
	if _, err := postgres.GetGoal(ctx, s.Pool, userID, goalID); err != nil {
		return nil, err
	}
	return postgres.GetPendingAction(ctx, s.Pool, goalID)
}

type CreateUserActionInput struct {
	UserID       string
	GoalID       string
	Title        string
	Detail       *string
	EstimatedMin *int
	CompetencyID *string
}

// CreateUserAction é o caminho manual de RN-03: o sistema nunca fica
// travado por causa da IA, você pode sempre escrever a sua própria ação.
// Se já existe uma pendente, ela é substituída (04-agentes.md §4.7 — sem
// agente, próxima ação = você escreve).
func (s *Service) CreateUserAction(ctx context.Context, in CreateUserActionInput) (*domain.NextAction, error) {
	var created *domain.NextAction
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		if _, err := postgres.GetGoal(ctx, tx, in.UserID, in.GoalID); err != nil {
			return err
		}
		now := time.Now()

		existing, err := postgres.GetPendingAction(ctx, tx, in.GoalID)
		if err == nil {
			if err := existing.Skip(domain.SkipOther, now); err != nil {
				return err
			}
			if err := postgres.ResolveNextAction(ctx, tx, existing); err != nil {
				return err
			}
		} else if !errors.Is(err, postgres.ErrNotFound) {
			return err
		}

		estimated := 20
		if in.EstimatedMin != nil {
			estimated = *in.EstimatedMin
		}
		minimal := "Versão de 5 min: releia a ação e escreva só o primeiro passo, sem terminar."

		id, err := idgen.NewUUIDv7()
		if err != nil {
			return err
		}
		a, err := domain.NewNextAction(id, in.GoalID, in.UserID, in.Title, estimated, minimal, domain.GeneratedByUser, now)
		if err != nil {
			return err
		}
		a.Detail = in.Detail
		a.CompetencyID = in.CompetencyID
		origin := domain.OriginPractice
		a.OriginKind = &origin

		if err := postgres.InsertNextAction(ctx, tx, a); err != nil {
			return err
		}
		created = a
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

type CompleteActionInput struct {
	UserID             string
	ActionID           string
	UsedMinimalVariant bool
	DurationMin        *int
	Note               *string
	Timezone           *time.Location
}

type CompleteActionResult struct {
	Completed *domain.NextAction
	Next      *domain.NextAction
	Session   *domain.Session
}

// CompleteAction é RN-03 na prática: fecha a atual e gera a próxima na
// mesma transação — nunca existe um instante em que a meta ativa fica sem
// ação (03-api.md §4, Problema C do negócio).
func (s *Service) CompleteAction(ctx context.Context, in CompleteActionInput) (*CompleteActionResult, error) {
	var result CompleteActionResult
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		a, err := postgres.GetAction(ctx, tx, in.UserID, in.ActionID)
		if err != nil {
			return err
		}
		now := time.Now()
		if err := a.Complete(now); err != nil {
			return err
		}
		if err := postgres.ResolveNextAction(ctx, tx, a); err != nil {
			return err
		}

		g, err := postgres.GetGoal(ctx, tx, in.UserID, a.GoalID)
		if err != nil {
			return err
		}
		g.TouchActivity(now)
		if err := postgres.UpdateGoal(ctx, tx, g); err != nil {
			return err
		}

		var sess *domain.Session
		if in.DurationMin != nil {
			loc := in.Timezone
			if loc == nil {
				loc = time.UTC
			}
			sid, err := idgen.NewUUIDv7()
			if err != nil {
				return err
			}
			s2, err := domain.NewSession(sid, g.ID, in.UserID, &a.ID, now, *in.DurationMin, in.Note, nil, loc, now)
			if err != nil {
				return err
			}
			if err := postgres.InsertSession(ctx, tx, s2); err != nil {
				return err
			}
			sess = s2
		}

		next, err := s.generateNextAction(ctx, tx, g, now, nil)
		if err != nil {
			return err
		}

		result = CompleteActionResult{Completed: a, Next: next, Session: sess}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type SkipActionInput struct {
	UserID   string
	ActionID string
	Reason   domain.SkipReason
	Note     *string
}

type SkipActionResult struct {
	Next *domain.NextAction
}

// SkipAction exige motivo (§7.3): é o insumo mais barato de calibragem de
// dificuldade que dá para fazer sem histórico — o motivo vira difficultyHint
// direto na próxima ação gerada.
func (s *Service) SkipAction(ctx context.Context, in SkipActionInput) (*SkipActionResult, error) {
	var result SkipActionResult
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		a, err := postgres.GetAction(ctx, tx, in.UserID, in.ActionID)
		if err != nil {
			return err
		}
		now := time.Now()
		if err := a.Skip(in.Reason, now); err != nil {
			return err
		}
		if err := postgres.ResolveNextAction(ctx, tx, a); err != nil {
			return err
		}

		g, err := postgres.GetGoal(ctx, tx, in.UserID, a.GoalID)
		if err != nil {
			return err
		}

		hint := domain.NextDifficultyHint(in.Reason)
		next, err := s.generateNextAction(ctx, tx, g, now, &hint)
		if err != nil {
			return err
		}

		result = SkipActionResult{Next: next}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}
