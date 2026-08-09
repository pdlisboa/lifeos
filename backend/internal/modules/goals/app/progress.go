package app

import (
	"context"
	"errors"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

type DeltaResult struct {
	Goal           *domain.Goal
	Competencies   []domain.Competency
	RisesLast90d   map[string]int       // por competencyID
	BaselineSetAt  map[string]time.Time // por competencyID — 1º evento de nível
	EvidenceCount  int
	MilestonesDone int
	SessionMinutes int
}

// GetDelta monta o Painel de Delta (§7.1): nunca "quanto falta", sempre
// quanto você andou contra o seu próprio baseline (P2). Sem projeção nem
// museu com comparação — isso fica para a Fatia 2.
func (s *Service) GetDelta(ctx context.Context, userID, goalID string) (*DeltaResult, error) {
	g, err := postgres.GetGoal(ctx, s.Pool, userID, goalID)
	if err != nil {
		return nil, err
	}
	comps, err := postgres.ListCompetenciesByGoal(ctx, s.Pool, goalID)
	if err != nil {
		return nil, err
	}
	events, err := postgres.ListLevelEventsForGoal(ctx, s.Pool, goalID)
	if err != nil {
		return nil, err
	}
	evidenceCount, err := postgres.CountEvidenceByGoal(ctx, s.Pool, goalID)
	if err != nil {
		return nil, err
	}
	sessionMin, err := postgres.SumSessionMinutes(ctx, s.Pool, goalID)
	if err != nil {
		return nil, err
	}

	milestonesDone := 0
	track, err := postgres.GetCurrentTrack(ctx, s.Pool, goalID)
	if err == nil {
		for _, m := range track.Milestones {
			if m.Status == domain.MilestoneCompleted {
				milestonesDone++
			}
		}
	} else if !errors.Is(err, postgres.ErrNotFound) {
		return nil, err
	}

	since := time.Now().AddDate(0, 0, -90)
	rises := make(map[string]int, len(comps))
	baselineSetAt := make(map[string]time.Time, len(comps))
	for _, c := range comps {
		var compEvents []domain.LevelEvent
		for _, e := range events {
			if e.CompetencyID == c.ID {
				compEvents = append(compEvents, e)
			}
		}
		rises[c.ID] = domain.RisesInWindow(compEvents, since)
		// events vem ordenado ASC por occurred_at (ListLevelEventsForGoal) —
		// o primeiro da competência é quando o baseline foi congelado.
		if len(compEvents) > 0 {
			baselineSetAt[c.ID] = compEvents[0].OccurredAt
		}
	}

	return &DeltaResult{
		Goal:           g,
		Competencies:   comps,
		RisesLast90d:   rises,
		BaselineSetAt:  baselineSetAt,
		EvidenceCount:  evidenceCount,
		MilestonesDone: milestonesDone,
		SessionMinutes: sessionMin,
	}, nil
}

// GetProjection é §7.2: ritmo real dos últimos 30 dias, nunca uma data de
// chegada inventada — a query é a de 02-modelo-de-dados.md §14.4.
func (s *Service) GetProjection(ctx context.Context, userID, goalID string) (domain.Projection, error) {
	if _, err := postgres.GetGoal(ctx, s.Pool, userID, goalID); err != nil {
		return domain.Projection{}, err
	}
	pace, err := postgres.SessionPaceLast30d(ctx, s.Pool, goalID)
	if err != nil {
		return domain.Projection{}, err
	}
	return domain.ComputeProjection(pace.TotalMinutes, pace.ActiveDays, pace.SpanDays), nil
}

func (s *Service) GetConsistency(ctx context.Context, userID, goalID string) (domain.ConsistencyWindow, error) {
	if _, err := postgres.GetGoal(ctx, s.Pool, userID, goalID); err != nil {
		return domain.ConsistencyWindow{}, err
	}
	dates, err := postgres.ActiveLocalDates(ctx, s.Pool, goalID, domain.DefaultConsistencyWindowDays)
	if err != nil {
		return domain.ConsistencyWindow{}, err
	}
	return domain.ComputeConsistency(dates, time.Now(), domain.DefaultConsistencyWindowDays), nil
}
