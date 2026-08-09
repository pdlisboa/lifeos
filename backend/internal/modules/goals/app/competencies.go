package app

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/platform/idgen"
)

type SetCompetencyLevelInput struct {
	UserID       string
	CompetencyID string
	Level        int
	Rationale    string
	EvidenceID   *string
}

// SetCompetencyLevel é o caminho manual de RN-04 quando não há agente: a
// "confirmação explícita sua" já é, sozinha, uma das duas formas válidas de
// mudar um nível — por isso entra com confiança alta, não baixa.
func (s *Service) SetCompetencyLevel(ctx context.Context, in SetCompetencyLevelInput) (*domain.Competency, error) {
	var updated *domain.Competency
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		c, err := postgres.GetCompetency(ctx, tx, in.UserID, in.CompetencyID)
		if err != nil {
			return err
		}
		if in.EvidenceID != nil {
			evidence, err := postgres.GetEvidence(ctx, tx, in.UserID, *in.EvidenceID)
			if err != nil {
				return err
			}
			if evidence.GoalID != c.GoalID {
				return domain.NewRuleError("", "evidência não pertence à meta desta competência")
			}
		}
		evID, err := idgen.NewUUIDv7()
		if err != nil {
			return err
		}
		now := time.Now()
		ev, err := domain.NewLevelEvent(evID, c.ID, in.UserID, c.CurrentLevel, in.Level, domain.ConfidenceHigh, domain.SourceSelf, in.EvidenceID, in.Rationale, now)
		if err != nil {
			return err
		}
		if err := c.ApplyLevelEvent(*ev); err != nil {
			return err
		}
		if err := postgres.InsertLevelEvent(ctx, tx, ev); err != nil {
			return err
		}
		if err := postgres.UpdateCompetencyState(ctx, tx, c); err != nil {
			return err
		}
		updated = c
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) CompetencyHistory(ctx context.Context, userID, competencyID string) ([]domain.LevelEvent, error) {
	if _, err := postgres.GetCompetency(ctx, s.Pool, userID, competencyID); err != nil {
		return nil, err
	}
	return postgres.ListLevelEvents(ctx, s.Pool, competencyID)
}
