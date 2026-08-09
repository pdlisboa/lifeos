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

// ListEvidenceByGoalAndCompetency filtra o museu por competência tocada
// (§7.4) — a mesma junção com evidence_competency que alimenta o filtro do
// query param `competencyId` em GET /goals/{id}/evidence.
func ListEvidenceByGoalAndCompetency(ctx context.Context, q Querier, goalID, competencyID string, ascending bool, limit int) ([]domain.Evidence, error) {
	qs := sqlcgen.New(q)
	var rows []sqlcgen.Evidence
	var err error
	if ascending {
		rows, err = qs.ListEvidenceByGoalAndCompetencyAsc(ctx, sqlcgen.ListEvidenceByGoalAndCompetencyAscParams{GoalID: goalID, CompetencyID: competencyID, Limit: int32(limit)})
	} else {
		rows, err = qs.ListEvidenceByGoalAndCompetencyDesc(ctx, sqlcgen.ListEvidenceByGoalAndCompetencyDescParams{GoalID: goalID, CompetencyID: competencyID, Limit: int32(limit)})
	}
	if err != nil {
		return nil, fmt.Errorf("listar evidence por competência: %w", err)
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

// InsertEvidenceCompetency é RN-agnóstico: só grava a tag "essa evidência
// tocou essa competência" (§7 da UX — "Competências tocadas"), validada
// contra as competências da meta na camada app antes de chamar aqui.
func InsertEvidenceCompetency(ctx context.Context, q Querier, evidenceID, competencyID string) error {
	err := sqlcgen.New(q).InsertEvidenceCompetency(ctx, sqlcgen.InsertEvidenceCompetencyParams{
		EvidenceID:   evidenceID,
		CompetencyID: competencyID,
	})
	if err != nil {
		return fmt.Errorf("ligar evidence a competency: %w", err)
	}
	return nil
}

// ListCompetencyIDsForEvidences agrupa por evidência de uma vez só — evita
// N+1 ao montar o museu (§7.4).
func ListCompetencyIDsForEvidences(ctx context.Context, q Querier, evidenceIDs []string) (map[string][]string, error) {
	out := make(map[string][]string, len(evidenceIDs))
	if len(evidenceIDs) == 0 {
		return out, nil
	}
	rows, err := sqlcgen.New(q).ListCompetencyIDsForEvidences(ctx, evidenceIDs)
	if err != nil {
		return nil, fmt.Errorf("listar competências das evidências: %w", err)
	}
	for _, r := range rows {
		out[r.EvidenceID] = append(out[r.EvidenceID], r.CompetencyID)
	}
	return out, nil
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
