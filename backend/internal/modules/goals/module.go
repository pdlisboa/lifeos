// Package goals é o módulo de domínio Metas (01-arquitetura.md §3): monta o
// wiring (packs → app → http) que cmd/api usa para registrar as rotas.
package goals

import (
	"fmt"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	goalshttp "github.com/phablo/lifeos/internal/modules/goals/adapters/http"
	"github.com/phablo/lifeos/internal/modules/goals/app"
	"github.com/phablo/lifeos/internal/modules/goals/packs"
	"github.com/phablo/lifeos/internal/platform/auth"
)

type Module struct {
	Service *app.Service
	Handler *goalshttp.Handler
}

// New carrega e valida os domain packs no boot, com falha ruidosa
// (04-agentes.md §4.6): um pack inválido não sobe — é pior descobrir isso
// semanas depois, com dados gravados em cima de uma rubrica quebrada.
func New(pool *pgxpool.Pool, users *auth.UserStore, logger *slog.Logger) (*Module, error) {
	registry, err := packs.Load()
	if err != nil {
		return nil, fmt.Errorf("carregar domain packs: %w", err)
	}
	svc := app.NewService(pool, registry)
	h := &goalshttp.Handler{Service: svc, Users: users, Logger: logger}
	return &Module{Service: svc, Handler: h}, nil
}

func (m *Module) RegisterRoutes(r chi.Router) {
	goalshttp.RegisterRoutes(r, m.Handler)
}
