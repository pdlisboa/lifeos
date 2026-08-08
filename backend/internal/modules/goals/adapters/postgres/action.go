package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres/sqlcgen"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func InsertNextAction(ctx context.Context, q Querier, a *domain.NextAction) error {
	var originKind *string
	if a.OriginKind != nil {
		s := string(*a.OriginKind)
		originKind = &s
	}
	var skipReason *string
	if a.SkipReason != nil {
		s := string(*a.SkipReason)
		skipReason = &s
	}
	minimalVariant := a.MinimalVariant

	err := sqlcgen.New(q).InsertNextAction(ctx, sqlcgen.InsertNextActionParams{
		ID:             a.ID,
		GoalID:         a.GoalID,
		UserID:         a.UserID,
		MilestoneID:    a.MilestoneID,
		CompetencyID:   a.CompetencyID,
		Title:          a.Title,
		Detail:         a.Detail,
		PracticeFormat: a.PracticeFormat,
		EstimatedMin:   int16(a.EstimatedMin),
		MinimalVariant: &minimalVariant,
		DifficultyHint: a.DifficultyHint,
		GeneratedBy:    string(a.GeneratedBy),
		OriginKind:     originKind,
		Status:         string(a.Status),
		SkipReason:     skipReason,
		ResolvedAt:     a.ResolvedAt,
		CreatedAt:      a.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("inserir next_action: %w", err)
	}
	return nil
}

// ResolveNextAction grava a transição de pending → completed/skipped feita
// no domínio (a struct já validou a regra; aqui só persiste o resultado).
func ResolveNextAction(ctx context.Context, q Querier, a *domain.NextAction) error {
	var skipReason *string
	if a.SkipReason != nil {
		s := string(*a.SkipReason)
		skipReason = &s
	}
	err := sqlcgen.New(q).ResolveNextAction(ctx, sqlcgen.ResolveNextActionParams{
		ID:         a.ID,
		Status:     string(a.Status),
		SkipReason: skipReason,
		ResolvedAt: a.ResolvedAt,
	})
	if err != nil {
		return fmt.Errorf("resolver next_action: %w", err)
	}
	return nil
}

func GetPendingAction(ctx context.Context, q Querier, goalID string) (*domain.NextAction, error) {
	r, err := sqlcgen.New(q).GetPendingAction(ctx, goalID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("buscar next_action pendente: %w", err)
	}
	return toDomainNextAction(r), nil
}

func GetAction(ctx context.Context, q Querier, userID, actionID string) (*domain.NextAction, error) {
	r, err := sqlcgen.New(q).GetAction(ctx, sqlcgen.GetActionParams{ID: actionID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("buscar next_action: %w", err)
	}
	return toDomainNextAction(r), nil
}

func toDomainNextAction(r sqlcgen.NextAction) *domain.NextAction {
	a := &domain.NextAction{
		ID:             r.ID,
		GoalID:         r.GoalID,
		UserID:         r.UserID,
		MilestoneID:    r.MilestoneID,
		CompetencyID:   r.CompetencyID,
		Title:          r.Title,
		Detail:         r.Detail,
		PracticeFormat: r.PracticeFormat,
		EstimatedMin:   int(r.EstimatedMin),
		DifficultyHint: r.DifficultyHint,
		GeneratedBy:    domain.GeneratedBy(r.GeneratedBy),
		Status:         domain.ActionStatus(r.Status),
		ResolvedAt:     r.ResolvedAt,
		CreatedAt:      r.CreatedAt,
	}
	if r.MinimalVariant != nil {
		a.MinimalVariant = *r.MinimalVariant
	}
	if r.OriginKind != nil {
		ok := domain.OriginKind(*r.OriginKind)
		a.OriginKind = &ok
	}
	if r.SkipReason != nil {
		sr := domain.SkipReason(*r.SkipReason)
		a.SkipReason = &sr
	}
	return a
}
