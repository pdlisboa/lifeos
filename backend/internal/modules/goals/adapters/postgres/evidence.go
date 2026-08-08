package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres/sqlcgen"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func InsertEvidence(ctx context.Context, q Querier, e *domain.Evidence) error {
	err := sqlcgen.New(q).InsertEvidence(ctx, sqlcgen.InsertEvidenceParams{
		ID:           e.ID,
		GoalID:       e.GoalID,
		UserID:       e.UserID,
		ActionID:     e.ActionID,
		Kind:         string(e.Kind),
		Title:        e.Title,
		Body:         e.Body,
		BlobKey:      e.BlobKey,
		ExternalUrl:  e.ExternalURL,
		Language:     e.Language,
		SupersedesID: e.SupersedesID,
		LocalOn:      e.LocalOn,
		CreatedAt:    e.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("inserir evidence: %w", err)
	}
	return nil
}

func GetEvidence(ctx context.Context, q Querier, userID, evidenceID string) (*domain.Evidence, error) {
	r, err := sqlcgen.New(q).GetEvidence(ctx, sqlcgen.GetEvidenceParams{ID: evidenceID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("buscar evidence: %w", err)
	}
	return toDomainEvidence(r), nil
}

// ListEvidenceByGoal serve o museu (§7.4) — ordem crescente por padrão
// (você quer ver o começo primeiro).
func ListEvidenceByGoal(ctx context.Context, q Querier, goalID string, ascending bool, limit int) ([]domain.Evidence, error) {
	qs := sqlcgen.New(q)
	var rows []sqlcgen.Evidence
	var err error
	if ascending {
		rows, err = qs.ListEvidenceByGoalAsc(ctx, sqlcgen.ListEvidenceByGoalAscParams{GoalID: goalID, Limit: int32(limit)})
	} else {
		rows, err = qs.ListEvidenceByGoalDesc(ctx, sqlcgen.ListEvidenceByGoalDescParams{GoalID: goalID, Limit: int32(limit)})
	}
	if err != nil {
		return nil, fmt.Errorf("listar evidence: %w", err)
	}
	out := make([]domain.Evidence, len(rows))
	for i, r := range rows {
		out[i] = *toDomainEvidence(r)
	}
	return out, nil
}

func CountEvidenceByGoal(ctx context.Context, q Querier, goalID string) (int, error) {
	n, err := sqlcgen.New(q).CountEvidenceByGoal(ctx, goalID)
	if err != nil {
		return 0, fmt.Errorf("contar evidence: %w", err)
	}
	return int(n), nil
}

func toDomainEvidence(r sqlcgen.Evidence) *domain.Evidence {
	return &domain.Evidence{
		ID:           r.ID,
		GoalID:       r.GoalID,
		UserID:       r.UserID,
		ActionID:     r.ActionID,
		Kind:         domain.EvidenceKind(r.Kind),
		Title:        r.Title,
		Body:         r.Body,
		BlobKey:      r.BlobKey,
		ExternalURL:  r.ExternalUrl,
		Language:     r.Language,
		SupersedesID: r.SupersedesID,
		LocalOn:      r.LocalOn,
		CreatedAt:    r.CreatedAt,
	}
}
