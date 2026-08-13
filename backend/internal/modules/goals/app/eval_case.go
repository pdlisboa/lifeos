package app

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

type MarkEvidenceEvalCaseInput struct {
	UserID     string
	EvidenceID string
	Note       string
	Scores     []domain.EvalCaseScore
}

// MarkEvidenceAsEvalCase captura o gabarito humano numa evidência já
// registrada, pra alimentar o conjunto de eval do A3 quando ele existir
// (04-agentes.md §6.1). Puro dado — nenhum agente entra aqui, e a marcação
// pode ser refeita (não é imutável como a evidência em si, RN-06).
func (s *Service) MarkEvidenceAsEvalCase(ctx context.Context, in MarkEvidenceEvalCaseInput) (*domain.EvalCase, error) {
	var out *domain.EvalCase
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		ev, err := postgres.GetEvidence(ctx, tx, in.UserID, in.EvidenceID)
		if err != nil {
			return err
		}
		comps, err := postgres.ListCompetenciesByGoal(ctx, tx, ev.GoalID)
		if err != nil {
			return err
		}
		valid := make(map[string]bool, len(comps))
		for _, c := range comps {
			valid[c.ID] = true
		}
		for _, sc := range in.Scores {
			if !valid[sc.CompetencyID] {
				return domain.NewRuleError("", "competência não pertence a esta meta")
			}
		}
		ec, err := domain.NewEvalCase(ev.ID, in.Note, in.Scores)
		if err != nil {
			return err
		}
		if err := postgres.UpsertEvalCase(ctx, tx, ec, time.Now()); err != nil {
			return err
		}
		out = ec
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UnmarkEvidenceEvalCase remove a marcação — ownership é checado via a
// mesma leitura de evidence usada em GetEvidence (só o dono vê e mexe).
func (s *Service) UnmarkEvidenceEvalCase(ctx context.Context, userID, evidenceID string) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		if _, err := postgres.GetEvidence(ctx, tx, userID, evidenceID); err != nil {
			return err
		}
		return postgres.DeleteEvalCase(ctx, tx, evidenceID)
	})
}

// GetEvalCase devolve nil sem erro quando a evidência existe mas não foi
// marcada — estado normal, não exceção.
func (s *Service) GetEvalCase(ctx context.Context, userID, evidenceID string) (*domain.EvalCase, error) {
	if _, err := postgres.GetEvidence(ctx, s.Pool, userID, evidenceID); err != nil {
		return nil, err
	}
	return postgres.GetEvalCase(ctx, s.Pool, evidenceID)
}
