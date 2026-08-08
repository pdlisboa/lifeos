package app

import (
	"context"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func (s *Service) GetTrack(ctx context.Context, userID, goalID string) (*domain.Track, error) {
	if _, err := postgres.GetGoal(ctx, s.Pool, userID, goalID); err != nil {
		return nil, err
	}
	return postgres.GetCurrentTrack(ctx, s.Pool, goalID)
}
