package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

// recentWinWindowDays casa com o "nas últimas 6 semanas" do mockup da tela
// Hoje (05-ux.md §3).
const recentWinWindowDays = 42

type TodayGoalResult struct {
	Goal      domain.Goal
	Action    *domain.NextAction
	RecentWin *string
}

type TodayResult struct {
	Goals       []TodayGoalResult
	Consistency domain.ConsistencyWindow
}

// GetToday monta a tela inicial numa chamada só (03-api.md, openapi.yaml
// /today): até 3 metas ativas com sua ação pendente, mais consistência
// agregada das metas mostradas. Sem nudge — depende do Coach (Fatia 6).
func (s *Service) GetToday(ctx context.Context, userID string) (*TodayResult, error) {
	goals, err := postgres.ListGoals(ctx, s.Pool, userID, []string{"active", "at_risk", "stagnant"})
	if err != nil {
		return nil, err
	}
	if len(goals) > domain.MaxActiveGoals {
		goals = goals[:domain.MaxActiveGoals]
	}

	var allDates []time.Time
	items := make([]TodayGoalResult, 0, len(goals))
	for _, g := range goals {
		action, err := postgres.GetPendingAction(ctx, s.Pool, g.ID)
		if err != nil {
			if errors.Is(err, postgres.ErrNotFound) {
				// RN-03 garante que isso não acontece; se acontecer, é bug em
				// outro lugar — não derruba a tela inteira por causa dele.
				continue
			}
			return nil, err
		}

		dates, err := postgres.ActiveLocalDates(ctx, s.Pool, g.ID, domain.DefaultConsistencyWindowDays)
		if err != nil {
			return nil, err
		}
		allDates = append(allDates, dates...)

		win, err := s.recentWin(ctx, g)
		if err != nil {
			return nil, err
		}

		items = append(items, TodayGoalResult{Goal: g, Action: action, RecentWin: win})
	}

	consistency := domain.ComputeConsistency(allDates, time.Now(), domain.DefaultConsistencyWindowDays)
	return &TodayResult{Goals: items, Consistency: consistency}, nil
}

// recentWin procura a subida de nível mais recente da meta dentro da janela
// e formata a frase de ganho (05-ux.md §3) — vem antes da ação de propósito,
// pra você ler o que subiu antes do que fazer. Sem subida honesta, nil: "melhor
// vazio que inventado".
func (s *Service) recentWin(ctx context.Context, g domain.Goal) (*string, error) {
	events, err := postgres.ListLevelEventsForGoal(ctx, s.Pool, g.ID)
	if err != nil {
		return nil, err
	}
	comps, err := postgres.ListCompetenciesByGoal(ctx, s.Pool, g.ID)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]domain.Competency, len(comps))
	for _, c := range comps {
		byID[c.ID] = c
	}

	since := time.Now().AddDate(0, 0, -recentWinWindowDays)
	var best *domain.LevelEvent
	for _, e := range events {
		if e.FromLevel == nil || e.ToLevel <= *e.FromLevel || e.OccurredAt.Before(since) {
			continue
		}
		ev := e
		if best == nil || ev.OccurredAt.After(best.OccurredAt) {
			best = &ev
		}
	}
	if best == nil {
		return nil, nil
	}
	comp, ok := byID[best.CompetencyID]
	if !ok {
		return nil, nil
	}

	weeks := int(time.Since(best.OccurredAt).Hours() / 24 / 7)
	if weeks < 1 {
		weeks = 1
	}
	msg := fmt.Sprintf("%s subiu de %d para %d nas últimas %d semanas.", comp.Label, *best.FromLevel, best.ToLevel, weeks)
	return &msg, nil
}
