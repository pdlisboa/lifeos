package app

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/platform/idgen"
)

type CreateSessionInput struct {
	UserID      string
	GoalID      string
	StartedAt   time.Time
	DurationMin int
	Note        *string
	Energy      *int
	ActionID    *string
	Timezone    *time.Location
}

// CreateSession é insumo, não progresso (P1) — sessão só alimenta ritmo e
// consistência, nunca sobe nível de competência sozinha. LocalOn é calculado
// aqui, no servidor, a partir do timezone do usuário (CLAUDE.md, armadilhas).
func (s *Service) CreateSession(ctx context.Context, in CreateSessionInput) (*domain.Session, error) {
	var created *domain.Session
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		g, err := postgres.GetGoal(ctx, tx, in.UserID, in.GoalID)
		if err != nil {
			return err
		}
		loc := in.Timezone
		if loc == nil {
			loc = time.UTC
		}
		id, err := idgen.NewUUIDv7()
		if err != nil {
			return err
		}
		now := time.Now()
		sess, err := domain.NewSession(id, g.ID, in.UserID, in.ActionID, in.StartedAt, in.DurationMin, in.Note, in.Energy, loc, now)
		if err != nil {
			return err
		}
		if err := postgres.InsertSession(ctx, tx, sess); err != nil {
			return err
		}
		g.TouchActivity(now)
		if err := postgres.UpdateGoal(ctx, tx, g); err != nil {
			return err
		}
		created = sess
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Service) ListSessions(ctx context.Context, userID, goalID string, limit int) ([]domain.Session, error) {
	if _, err := postgres.GetGoal(ctx, s.Pool, userID, goalID); err != nil {
		return nil, err
	}
	return postgres.ListSessionsByGoal(ctx, s.Pool, goalID, limit)
}
