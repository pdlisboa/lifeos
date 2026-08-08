package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/phablo/lifeos/internal/modules/goals/app"
	"github.com/phablo/lifeos/internal/platform/httpx"
)

type createSessionRequest struct {
	StartedAt   string  `json:"startedAt"`
	DurationMin int     `json:"durationMin"`
	Note        *string `json:"note"`
	Energy      *int    `json:"energy"`
	ActionID    *string `json:"actionId"`
}

func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	goalID := chi.URLParam(r, "goalId")

	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, "Corpo inválido", err.Error(), "")
		return
	}
	startedAt, err := time.Parse(time.RFC3339, req.StartedAt)
	if err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, "startedAt inválido, use RFC3339", err.Error(), "")
		return
	}

	sess, err := h.Service.CreateSession(r.Context(), app.CreateSessionInput{
		UserID:      uid,
		GoalID:      goalID,
		StartedAt:   startedAt,
		DurationMin: req.DurationMin,
		Note:        req.Note,
		Energy:      req.Energy,
		ActionID:    req.ActionID,
		Timezone:    h.userLocation(r.Context(), uid),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toSessionDTO(*sess))
}

func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	limit := parseLimit(r, 30)
	sessions, err := h.Service.ListSessions(r.Context(), uid, chi.URLParam(r, "goalId"), limit)
	if err != nil {
		writeAppError(w, err)
		return
	}
	items := make([]sessionDTO, len(sessions))
	for i, s := range sessions {
		items[i] = toSessionDTO(s)
	}
	httpx.WriteJSON(w, http.StatusOK, pageDTO[sessionDTO]{Items: items, HasMore: false})
}
