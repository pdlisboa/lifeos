package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/phablo/lifeos/internal/modules/goals/app"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/platform/httpx"
)

func (h *Handler) GetPendingAction(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	a, err := h.Service.GetPendingAction(r.Context(), uid, chi.URLParam(r, "goalId"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toNextActionDTO(a))
}

type createActionRequest struct {
	Title        string  `json:"title"`
	Detail       *string `json:"detail"`
	EstimatedMin *int    `json:"estimatedMin"`
	CompetencyID *string `json:"competencyId"`
}

func (h *Handler) CreateUserAction(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	goalID := chi.URLParam(r, "goalId")

	var req createActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, "Corpo inválido", err.Error(), "")
		return
	}

	a, err := h.Service.CreateUserAction(r.Context(), app.CreateUserActionInput{
		UserID:       uid,
		GoalID:       goalID,
		Title:        req.Title,
		Detail:       req.Detail,
		EstimatedMin: req.EstimatedMin,
		CompetencyID: req.CompetencyID,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toNextActionDTO(a))
}

type completeActionRequest struct {
	UsedMinimalVariant bool    `json:"usedMinimalVariant"`
	DurationMin        *int    `json:"durationMin"`
	Note               *string `json:"note"`
}

func (h *Handler) CompleteAction(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	actionID := chi.URLParam(r, "actionId")

	var req completeActionRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req) // corpo é opcional
	}

	result, err := h.Service.CompleteAction(r.Context(), app.CompleteActionInput{
		UserID:             uid,
		ActionID:           actionID,
		UsedMinimalVariant: req.UsedMinimalVariant,
		DurationMin:        req.DurationMin,
		Note:               req.Note,
		Timezone:           h.userLocation(r.Context(), uid),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}

	var session *sessionDTO
	if result.Session != nil {
		d := toSessionDTO(*result.Session)
		session = &d
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"completed": toNextActionDTO(result.Completed),
		"next":      toNextActionDTO(result.Next),
		"session":   session,
	})
}

type skipActionRequest struct {
	Reason string  `json:"reason"`
	Note   *string `json:"note"`
}

func (h *Handler) SkipAction(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	actionID := chi.URLParam(r, "actionId")

	var req skipActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, "Corpo inválido", err.Error(), "")
		return
	}

	result, err := h.Service.SkipAction(r.Context(), app.SkipActionInput{
		UserID:   uid,
		ActionID: actionID,
		Reason:   domain.SkipReason(req.Reason),
		Note:     req.Note,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"next": toNextActionDTO(result.Next)})
}
