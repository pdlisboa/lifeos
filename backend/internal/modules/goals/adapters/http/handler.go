package http

import (
	"context"
	"log/slog"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/app"
	"github.com/phablo/lifeos/internal/platform/auth"
)

// Handler implementa os endpoints do módulo Metas usados na Fatia 1
// (backend/api/openapi.yaml). Depende de platform/auth só para resolver o
// timezone do usuário — módulos importam platform, nunca o contrário
// (CLAUDE.md, fronteira entre módulos).
type Handler struct {
	Service *app.Service
	Users   *auth.UserStore
	Logger  *slog.Logger
}

// userLocation resolve o timezone do usuário para calcular local_on no
// servidor (Session/Evidence) — nunca no navegador (CLAUDE.md, armadilhas).
func (h *Handler) userLocation(ctx context.Context, userID string) *time.Location {
	u, err := h.Users.ByID(ctx, userID)
	if err != nil {
		return time.UTC
	}
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}
