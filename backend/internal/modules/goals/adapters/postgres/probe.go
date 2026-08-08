package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres/sqlcgen"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func InsertProbe(ctx context.Context, q Querier, p *domain.Probe) error {
	err := sqlcgen.New(q).InsertProbe(ctx, sqlcgen.InsertProbeParams{
		ID:         p.ID,
		GoalID:     p.GoalID,
		UserID:     p.UserID,
		Status:     string(p.Status),
		AskedCount: int16(p.AskedCount),
		ClosedAt:   p.ClosedAt,
	})
	if err != nil {
		return fmt.Errorf("inserir probe: %w", err)
	}
	return nil
}

func UpdateProbe(ctx context.Context, q Querier, p *domain.Probe) error {
	err := sqlcgen.New(q).UpdateProbe(ctx, sqlcgen.UpdateProbeParams{
		ID:         p.ID,
		Status:     string(p.Status),
		AskedCount: int16(p.AskedCount),
		ClosedAt:   p.ClosedAt,
	})
	if err != nil {
		return fmt.Errorf("atualizar probe: %w", err)
	}
	return nil
}

func InsertProbeTurn(ctx context.Context, q Querier, t *domain.ProbeTurn) error {
	err := sqlcgen.New(q).InsertProbeTurn(ctx, sqlcgen.InsertProbeTurnParams{
		ID:            t.ID,
		ProbeID:       t.ProbeID,
		CompetencyID:  t.CompetencyID,
		Ordinal:       int16(t.Ordinal),
		Question:      t.Question,
		Answer:        t.Answer,
		InferredLevel: i16ptr(t.InferredLevel),
		AnsweredAt:    t.AnsweredAt,
	})
	if err != nil {
		return fmt.Errorf("inserir probe_turn: %w", err)
	}
	return nil
}

func UpdateProbeTurnAnswer(ctx context.Context, q Querier, t *domain.ProbeTurn) error {
	err := sqlcgen.New(q).UpdateProbeTurnAnswer(ctx, sqlcgen.UpdateProbeTurnAnswerParams{
		ID:         t.ID,
		Answer:     t.Answer,
		AnsweredAt: t.AnsweredAt,
	})
	if err != nil {
		return fmt.Errorf("atualizar resposta do probe_turn: %w", err)
	}
	return nil
}

// GetProbeByGoal devolve a sondagem de uma meta com todos os turnos já
// feitos, na ordem em que foram perguntados.
func GetProbeByGoal(ctx context.Context, q Querier, goalID string) (*domain.Probe, error) {
	qs := sqlcgen.New(q)

	r, err := qs.GetProbeByGoal(ctx, goalID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("buscar probe: %w", err)
	}
	p := &domain.Probe{
		ID:         r.ID,
		GoalID:     r.GoalID,
		UserID:     r.UserID,
		Status:     domain.ProbeStatus(r.Status),
		AskedCount: int(r.AskedCount),
		ClosedAt:   r.ClosedAt,
	}

	turns, err := qs.ListProbeTurnsByProbe(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("listar probe_turn: %w", err)
	}
	for _, t := range turns {
		p.Turns = append(p.Turns, domain.ProbeTurn{
			ID:            t.ID,
			ProbeID:       t.ProbeID,
			CompetencyID:  t.CompetencyID,
			Ordinal:       int(t.Ordinal),
			Question:      t.Question,
			Answer:        t.Answer,
			InferredLevel: intptr(t.InferredLevel),
			AnsweredAt:    t.AnsweredAt,
		})
	}
	return p, nil
}
