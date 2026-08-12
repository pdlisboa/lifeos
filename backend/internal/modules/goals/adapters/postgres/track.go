package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres/sqlcgen"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func InsertTrack(ctx context.Context, q Querier, t *domain.Track) error {
	err := sqlcgen.New(q).InsertTrack(ctx, sqlcgen.InsertTrackParams{
		ID:           t.ID,
		GoalID:       t.GoalID,
		UserID:       t.UserID,
		Version:      int32(t.Version),
		GeneratedBy:  t.GeneratedBy,
		AcceptedAt:   t.AcceptedAt,
		SupersededAt: t.SupersededAt,
	})
	if err != nil {
		return fmt.Errorf("inserir track: %w", err)
	}
	return nil
}

func InsertMilestone(ctx context.Context, q Querier, m *domain.Milestone) error {
	err := sqlcgen.New(q).InsertMilestone(ctx, sqlcgen.InsertMilestoneParams{
		ID:                 m.ID,
		TrackID:            m.TrackID,
		GoalID:             m.GoalID,
		UserID:             m.UserID,
		Ordinal:            int32(m.Ordinal),
		Title:              m.Title,
		CompletionCriteria: m.CompletionCriteria,
		Status:             string(m.Status),
		CompletedAt:        m.CompletedAt,
	})
	if err != nil {
		return fmt.Errorf("inserir milestone: %w", err)
	}
	return nil
}

func InsertMilestoneCompetency(ctx context.Context, q Querier, milestoneID, competencyID string) error {
	err := sqlcgen.New(q).InsertMilestoneCompetency(ctx, sqlcgen.InsertMilestoneCompetencyParams{
		MilestoneID:  milestoneID,
		CompetencyID: competencyID,
	})
	if err != nil {
		return fmt.Errorf("inserir milestone_competency: %w", err)
	}
	return nil
}

func UpdateMilestone(ctx context.Context, q Querier, m *domain.Milestone) error {
	err := sqlcgen.New(q).UpdateMilestone(ctx, sqlcgen.UpdateMilestoneParams{
		ID:          m.ID,
		Status:      string(m.Status),
		CompletedAt: m.CompletedAt,
	})
	if err != nil {
		return fmt.Errorf("atualizar milestone: %w", err)
	}
	return nil
}

// GetCurrentTrack devolve a trilha vigente de uma meta (a mais recente que
// não foi superseded), com seus marcos e as competências que cada um exerce.
func GetCurrentTrack(ctx context.Context, q Querier, goalID string) (*domain.Track, error) {
	r, err := sqlcgen.New(q).GetCurrentTrack(ctx, goalID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("buscar track: %w", err)
	}
	t := &domain.Track{
		ID:           r.ID,
		GoalID:       r.GoalID,
		UserID:       r.UserID,
		Version:      int(r.Version),
		GeneratedBy:  r.GeneratedBy,
		AcceptedAt:   r.AcceptedAt,
		SupersededAt: r.SupersededAt,
	}
	milestones, err := listMilestonesByTrack(ctx, q, t.ID)
	if err != nil {
		return nil, err
	}
	t.Milestones = milestones
	return t, nil
}

func listMilestonesByTrack(ctx context.Context, q Querier, trackID string) ([]domain.Milestone, error) {
	rows, err := sqlcgen.New(q).ListMilestonesByTrack(ctx, trackID)
	if err != nil {
		return nil, fmt.Errorf("listar milestones: %w", err)
	}

	milestones := make([]domain.Milestone, len(rows))
	ids := make([]string, len(rows))
	for i, r := range rows {
		milestones[i] = domain.Milestone{
			ID:                 r.ID,
			TrackID:            r.TrackID,
			GoalID:             r.GoalID,
			UserID:             r.UserID,
			Ordinal:            int(r.Ordinal),
			Title:              r.Title,
			CompletionCriteria: r.CompletionCriteria,
			Status:             domain.MilestoneStatus(r.Status),
			CompletedAt:        r.CompletedAt,
		}
		ids[i] = r.ID
	}

	compByMilestone, err := listCompetencyIDsByMilestones(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	for i := range milestones {
		milestones[i].CompetencyIDs = compByMilestone[milestones[i].ID]
	}
	return milestones, nil
}

func listCompetencyIDsByMilestones(ctx context.Context, q Querier, milestoneIDs []string) (map[string][]string, error) {
	out := make(map[string][]string)
	if len(milestoneIDs) == 0 {
		return out, nil
	}
	rows, err := sqlcgen.New(q).ListMilestoneCompetenciesByMilestones(ctx, milestoneIDs)
	if err != nil {
		return nil, fmt.Errorf("listar milestone_competency: %w", err)
	}
	for _, r := range rows {
		out[r.MilestoneID] = append(out[r.MilestoneID], r.CompetencyID)
	}
	return out, nil
}

// SupersedeTrack fecha a trilha vigente ao aplicar uma nova versão (aceite
// de proposta do A1) — a mesma trilha nunca some, só deixa de ser a atual
// (GetCurrentTrack filtra por superseded_at IS NULL).
func SupersedeTrack(ctx context.Context, q Querier, trackID string) error {
	if err := sqlcgen.New(q).SupersedeTrack(ctx, trackID); err != nil {
		return fmt.Errorf("superseder track: %w", err)
	}
	return nil
}

func GetMilestone(ctx context.Context, q Querier, userID, milestoneID string) (*domain.Milestone, error) {
	r, err := sqlcgen.New(q).GetMilestone(ctx, sqlcgen.GetMilestoneParams{ID: milestoneID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("buscar milestone: %w", err)
	}
	return &domain.Milestone{
		ID:                 r.ID,
		TrackID:            r.TrackID,
		GoalID:             r.GoalID,
		UserID:             r.UserID,
		Ordinal:            int(r.Ordinal),
		Title:              r.Title,
		CompletionCriteria: r.CompletionCriteria,
		Status:             domain.MilestoneStatus(r.Status),
		CompletedAt:        r.CompletedAt,
	}, nil
}
