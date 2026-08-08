package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/phablo/lifeos/internal/modules/goals/app"
	"github.com/phablo/lifeos/internal/platform/httpx"
)

func (h *Handler) GetProbe(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	probe, err := h.Service.GetProbe(r.Context(), uid, chi.URLParam(r, "goalId"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toProbeDTO(probe))
}

type answerProbeRequest struct {
	TurnID string `json:"turnId"`
	Answer string `json:"answer"`
}

func (h *Handler) AnswerProbe(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	goalID := chi.URLParam(r, "goalId")

	var req answerProbeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, "Corpo inválido", err.Error(), "")
		return
	}

	result, err := h.Service.AnswerProbe(r.Context(), app.AnswerProbeInput{
		UserID: uid,
		GoalID: goalID,
		TurnID: req.TurnID,
		Answer: req.Answer,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}

	var next *probeTurnDTO
	if result.NextQuestion != nil {
		d := toProbeTurnDTO(*result.NextQuestion)
		next = &d
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"probe":        toProbeDTO(result.Probe),
		"nextQuestion": next,
	})
}

func (h *Handler) SkipProbe(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	probe, err := h.Service.SkipProbe(r.Context(), uid, chi.URLParam(r, "goalId"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toProbeDTO(probe))
}
