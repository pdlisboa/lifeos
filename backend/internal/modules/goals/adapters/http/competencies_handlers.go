package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/phablo/lifeos/internal/modules/goals/app"
	"github.com/phablo/lifeos/internal/platform/httpx"
)

type setLevelRequest struct {
	Level      int     `json:"level"`
	Rationale  string  `json:"rationale"`
	EvidenceID *string `json:"evidenceId"`
}

// SetCompetencyLevel é o caminho manual de RN-04 quando não há agente
// (04-agentes.md §4.7): "você define, source: self".
func (h *Handler) SetCompetencyLevel(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	var req setLevelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, "Corpo inválido", err.Error(), "")
		return
	}

	c, err := h.Service.SetCompetencyLevel(r.Context(), app.SetCompetencyLevelInput{
		UserID:       uid,
		CompetencyID: chi.URLParam(r, "competencyId"),
		Level:        req.Level,
		Rationale:    req.Rationale,
		EvidenceID:   req.EvidenceID,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}

	goal, err := h.Service.GetGoal(r.Context(), uid, c.GoalID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toCompetencyDTO(h.Service.Packs, goal.Goal.PackID, *c))
}

func (h *Handler) CompetencyHistory(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	events, err := h.Service.CompetencyHistory(r.Context(), uid, chi.URLParam(r, "competencyId"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	out := make([]levelEventDTO, len(events))
	for i, e := range events {
		out[i] = toLevelEventDTO(e)
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
