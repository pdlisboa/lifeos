package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres/sqlcgen"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

var ErrNotFound = errors.New("não encontrado")

func InsertGoal(ctx context.Context, q Querier, g *domain.Goal) error {
	err := sqlcgen.New(q).InsertGoal(ctx, sqlcgen.InsertGoalParams{
		ID:               g.ID,
		UserID:           g.UserID,
		Title:            g.Title,
		Archetype:        string(g.Archetype),
		PackID:           g.PackID,
		Why:              g.Why,
		DefinitionOfDone: g.DefinitionOfDone,
		HorizonOn:        g.HorizonOn,
		Status:           string(g.Status),
		ScopeMode:        g.ScopeMode,
		PausedUntil:      g.PausedUntil,
		ActivatedAt:      g.ActivatedAt,
		LastActivityAt:   g.LastActivityAt,
		ClosedAt:         g.ClosedAt,
		CreatedAt:        g.CreatedAt,
		UpdatedAt:        g.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("inserir goal: %w", err)
	}
	return nil
}

// UpdateGoal grava o estado inteiro da meta. Chamado depois de qualquer
// transição de domínio (Activate, Pause, Close) ou PATCH.
func UpdateGoal(ctx context.Context, q Querier, g *domain.Goal) error {
	rows, err := sqlcgen.New(q).UpdateGoal(ctx, sqlcgen.UpdateGoalParams{
		ID:               g.ID,
		Title:            g.Title,
		Why:              g.Why,
		DefinitionOfDone: g.DefinitionOfDone,
		HorizonOn:        g.HorizonOn,
		Status:           string(g.Status),
		ScopeMode:        g.ScopeMode,
		PausedUntil:      g.PausedUntil,
		ActivatedAt:      g.ActivatedAt,
		LastActivityAt:   g.LastActivityAt,
		ClosedAt:         g.ClosedAt,
		UpdatedAt:        g.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("atualizar goal: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func GetGoal(ctx context.Context, q Querier, userID, goalID string) (*domain.Goal, error) {
	row, err := sqlcgen.New(q).GetGoal(ctx, sqlcgen.GetGoalParams{ID: goalID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("buscar goal: %w", err)
	}
	return toDomainGoal(row), nil
}

func ListGoals(ctx context.Context, q Querier, userID string, statuses []string) ([]domain.Goal, error) {
	qs := sqlcgen.New(q)
	var rows []sqlcgen.Goal
	var err error
	if len(statuses) > 0 {
		rows, err = qs.ListGoalsByStatus(ctx, sqlcgen.ListGoalsByStatusParams{UserID: userID, Column2: statuses})
	} else {
		rows, err = qs.ListGoals(ctx, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("listar goals: %w", err)
	}
	out := make([]domain.Goal, len(rows))
	for i, r := range rows {
		out[i] = *toDomainGoal(r)
	}
	return out, nil
}

// CountActiveGoals é a metade "amigável" de RN-02: o caso de uso consulta
// isto antes de tentar ativar, para devolver a lista de "qual pausar" em vez
// de deixar o trigger do banco estourar uma exceção crua.
func CountActiveGoals(ctx context.Context, q Querier, userID string) (int, error) {
	n, err := sqlcgen.New(q).CountActiveGoals(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("contar goals ativas: %w", err)
	}
	return int(n), nil
}

// LockUser serializa ativações concorrentes do mesmo usuário — RN-02 precisa
// que "contar ativas" e "ativar mais uma" sejam atômicos (02-modelo-de-dados.md §4.1).
func LockUser(ctx context.Context, q Querier, userID string) error {
	if _, err := sqlcgen.New(q).LockUser(ctx, userID); err != nil {
		return fmt.Errorf("travar usuário: %w", err)
	}
	return nil
}

func toDomainGoal(r sqlcgen.Goal) *domain.Goal {
	return &domain.Goal{
		ID:               r.ID,
		UserID:           r.UserID,
		Title:            r.Title,
		Archetype:        domain.GoalArchetype(r.Archetype),
		PackID:           r.PackID,
		Why:              r.Why,
		DefinitionOfDone: r.DefinitionOfDone,
		HorizonOn:        r.HorizonOn,
		Status:           domain.GoalStatus(r.Status),
		ScopeMode:        r.ScopeMode,
		PausedUntil:      r.PausedUntil,
		ActivatedAt:      r.ActivatedAt,
		LastActivityAt:   r.LastActivityAt,
		ClosedAt:         r.ClosedAt,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}
