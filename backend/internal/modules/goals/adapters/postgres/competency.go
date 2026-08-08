package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres/sqlcgen"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func InsertCompetency(ctx context.Context, q Querier, c *domain.Competency) error {
	err := sqlcgen.New(q).InsertCompetency(ctx, sqlcgen.InsertCompetencyParams{
		ID:             c.ID,
		GoalID:         c.GoalID,
		UserID:         c.UserID,
		PackKey:        c.PackKey,
		Label:          c.Label,
		Weight:         c.Weight,
		CurrentLevel:   i16ptr(c.CurrentLevel),
		Confidence:     string(c.Confidence),
		BaselineLevel:  i16ptr(c.BaselineLevel),
		LastEvidenceAt: c.LastEvidenceAt,
		RetiredAt:      c.RetiredAt,
		CreatedAt:      c.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("inserir competency: %w", err)
	}
	return nil
}

func ListCompetenciesByGoal(ctx context.Context, q Querier, goalID string) ([]domain.Competency, error) {
	rows, err := sqlcgen.New(q).ListCompetenciesByGoal(ctx, goalID)
	if err != nil {
		return nil, fmt.Errorf("listar competencies: %w", err)
	}
	out := make([]domain.Competency, len(rows))
	for i, r := range rows {
		out[i] = domain.Competency{
			ID:             r.ID,
			GoalID:         r.GoalID,
			UserID:         r.UserID,
			PackKey:        r.PackKey,
			Label:          r.Label,
			Weight:         r.Weight,
			CurrentLevel:   intptr(r.CurrentLevel),
			Confidence:     domain.Confidence(r.Confidence),
			BaselineLevel:  intptr(r.BaselineLevel),
			LastEvidenceAt: r.LastEvidenceAt,
			RetiredAt:      r.RetiredAt,
			CreatedAt:      r.CreatedAt,
		}
	}
	return out, nil
}

func CountCompetenciesByGoal(ctx context.Context, q Querier, goalID string) (int, error) {
	n, err := sqlcgen.New(q).CountCompetenciesByGoal(ctx, goalID)
	if err != nil {
		return 0, fmt.Errorf("contar competencies: %w", err)
	}
	return int(n), nil
}

func GetCompetency(ctx context.Context, q Querier, userID, competencyID string) (*domain.Competency, error) {
	r, err := sqlcgen.New(q).GetCompetency(ctx, sqlcgen.GetCompetencyParams{ID: competencyID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("buscar competency: %w", err)
	}
	return &domain.Competency{
		ID:             r.ID,
		GoalID:         r.GoalID,
		UserID:         r.UserID,
		PackKey:        r.PackKey,
		Label:          r.Label,
		Weight:         r.Weight,
		CurrentLevel:   intptr(r.CurrentLevel),
		Confidence:     domain.Confidence(r.Confidence),
		BaselineLevel:  intptr(r.BaselineLevel),
		LastEvidenceAt: r.LastEvidenceAt,
		RetiredAt:      r.RetiredAt,
		CreatedAt:      r.CreatedAt,
	}, nil
}

// UpdateCompetencyState grava o cache (current_level, confidence,
// baseline_level, last_evidence_at) depois de aplicar um LevelEvent — a
// tabela competency_level_event é que é a fonte da verdade (02-modelo-de-dados.md §1).
func UpdateCompetencyState(ctx context.Context, q Querier, c *domain.Competency) error {
	err := sqlcgen.New(q).UpdateCompetencyState(ctx, sqlcgen.UpdateCompetencyStateParams{
		ID:             c.ID,
		CurrentLevel:   i16ptr(c.CurrentLevel),
		Confidence:     string(c.Confidence),
		BaselineLevel:  i16ptr(c.BaselineLevel),
		LastEvidenceAt: c.LastEvidenceAt,
	})
	if err != nil {
		return fmt.Errorf("atualizar estado da competency: %w", err)
	}
	return nil
}

func InsertLevelEvent(ctx context.Context, q Querier, ev *domain.LevelEvent) error {
	err := sqlcgen.New(q).InsertLevelEvent(ctx, sqlcgen.InsertLevelEventParams{
		ID:           ev.ID,
		CompetencyID: ev.CompetencyID,
		UserID:       ev.UserID,
		FromLevel:    i16ptr(ev.FromLevel),
		ToLevel:      int16(ev.ToLevel),
		Confidence:   string(ev.Confidence),
		Source:       string(ev.Source),
		AssessmentID: ev.AssessmentID,
		Rationale:    ev.Rationale,
		OccurredAt:   ev.OccurredAt,
	})
	if err != nil {
		return fmt.Errorf("inserir level event: %w", err)
	}
	return nil
}

func ListLevelEvents(ctx context.Context, q Querier, competencyID string) ([]domain.LevelEvent, error) {
	rows, err := sqlcgen.New(q).ListLevelEvents(ctx, competencyID)
	if err != nil {
		return nil, fmt.Errorf("listar level events: %w", err)
	}
	return toDomainLevelEvents(rows), nil
}

// ListLevelEventsForGoal traz todos os eventos das competências de uma meta
// de uma vez — usado no Painel de Delta para contar subidas recentes sem
// N+1 (02-modelo-de-dados.md §14.2).
func ListLevelEventsForGoal(ctx context.Context, q Querier, goalID string) ([]domain.LevelEvent, error) {
	rows, err := sqlcgen.New(q).ListLevelEventsForGoal(ctx, goalID)
	if err != nil {
		return nil, fmt.Errorf("listar level events da meta: %w", err)
	}
	return toDomainLevelEvents(rows), nil
}

func toDomainLevelEvents(rows []sqlcgen.CompetencyLevelEvent) []domain.LevelEvent {
	out := make([]domain.LevelEvent, len(rows))
	for i, r := range rows {
		out[i] = domain.LevelEvent{
			ID:           r.ID,
			CompetencyID: r.CompetencyID,
			UserID:       r.UserID,
			FromLevel:    intptr(r.FromLevel),
			ToLevel:      int(r.ToLevel),
			Confidence:   domain.Confidence(r.Confidence),
			Source:       domain.LevelSource(r.Source),
			AssessmentID: r.AssessmentID,
			Rationale:    r.Rationale,
			OccurredAt:   r.OccurredAt,
		}
	}
	return out
}
