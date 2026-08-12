// Package app orquestra os casos de uso do módulo Metas: transação, domínio
// e persistência (01-arquitetura.md §3.1). Conhece domain/ e os adapters de
// postgres/packs — nunca o contrário.
package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/modules/goals/packs"
	"github.com/phablo/lifeos/internal/platform/agents"
)

type Service struct {
	Pool  *pgxpool.Pool
	Packs *packs.Registry
	// Gateway pode ser nil — só o worker constrói um de verdade (cmd/worker,
	// que é o único processo que chama agente). cmd/api enfileira jobs, mas
	// nunca chama o gateway direto (01-arquitetura.md §5.1: LLM não pode
	// estar no caminho crítico de request HTTP).
	Gateway *agents.Gateway
	Logger  *slog.Logger
}

// NewService aceita logger nil (vira io.Discard) — a maioria dos testes não
// se importa com log nenhum, só os handlers de job (agent_jobs.go) logam de
// verdade.
func NewService(pool *pgxpool.Pool, reg *packs.Registry, gateway *agents.Gateway, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Service{Pool: pool, Packs: reg, Gateway: gateway, Logger: logger}
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
