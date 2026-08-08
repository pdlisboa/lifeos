package app

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/platform/idgen"
)

type CreateEvidenceInput struct {
	UserID       string
	GoalID       string
	ActionID     *string
	Kind         domain.EvidenceKind
	Title        *string
	Body         *string
	BlobKey      *string
	ExternalURL  *string
	Language     *string
	SupersedesID *string
	Timezone     *time.Location
}

// CreateEvidence é P1: a moeda real do sistema. Imutável a partir daqui — o
// banco recusa UPDATE/DELETE (RN-06); correção é sempre uma evidência nova
// com SupersedesID apontando para a anterior.
func (s *Service) CreateEvidence(ctx context.Context, in CreateEvidenceInput) (*domain.Evidence, error) {
	var created *domain.Evidence
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
		ev, err := domain.NewEvidence(id, g.ID, in.UserID, in.ActionID, in.Kind, in.Title, in.Body, in.BlobKey, in.ExternalURL, in.Language, in.SupersedesID, loc, now)
		if err != nil {
			return err
		}
		if err := postgres.InsertEvidence(ctx, tx, ev); err != nil {
			return err
		}
		g.TouchActivity(now)
		if err := postgres.UpdateGoal(ctx, tx, g); err != nil {
			return err
		}
		created = ev
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Service) GetEvidence(ctx context.Context, userID, evidenceID string) (*domain.Evidence, error) {
	return postgres.GetEvidence(ctx, s.Pool, userID, evidenceID)
}

func (s *Service) ListEvidence(ctx context.Context, userID, goalID string, ascending bool, limit int) ([]domain.Evidence, error) {
	if _, err := postgres.GetGoal(ctx, s.Pool, userID, goalID); err != nil {
		return nil, err
	}
	return postgres.ListEvidenceByGoal(ctx, s.Pool, goalID, ascending, limit)
}
