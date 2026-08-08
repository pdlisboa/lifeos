// Package app orquestra os casos de uso do módulo Metas: transação, domínio
// e persistência (01-arquitetura.md §3.1). Conhece domain/ e os adapters de
// postgres/packs — nunca o contrário.
package app

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/modules/goals/packs"
)

type Service struct {
	Pool  *pgxpool.Pool
	Packs *packs.Registry
}

func NewService(pool *pgxpool.Pool, reg *packs.Registry) *Service {
	return &Service{Pool: pool, Packs: reg}
}

func (s *Service) withTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) pack(packID string) (packs.Pack, error) {
	p, ok := s.Packs.Get(packID)
	if !ok {
		return packs.Pack{}, domain.NewRuleError("", fmt.Sprintf("pack desconhecido: %s", packID))
	}
	return p, nil
}
