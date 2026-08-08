package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/phablo/lifeos/internal/modules/goals/app"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/platform/auth"
	"github.com/phablo/lifeos/internal/platform/httpx"
)

func userID(r *http.Request) (string, bool) {
	return auth.UserIDFromContext(r.Context())
}

func requireUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := userID(r)
	if !ok {
		httpx.WriteProblem(w, http.StatusUnauthorized, "Não autenticado", "", "")
		return "", false
	}
	return id, true
}

type createGoalRequest struct {
	Title     string  `json:"title"`
	Archetype string  `json:"archetype"`
	PackID    string  `json:"packId"`
	Why       *string `json:"why"`
}

func (h *Handler) CreateGoal(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	var req createGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, "Corpo inválido", err.Error(), "")
		return
	}

	result, err := h.Service.CreateGoal(r.Context(), app.CreateGoalInput{
		UserID:    uid,
		Title:     req.Title,
		Archetype: domain.GoalArchetype(req.Archetype),
		PackID:    req.PackID,
		Why:       req.Why,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"goal":  toGoalDTO(result.Goal, result.Competencies, h.Service.Packs),
		"probe": toProbeDTO(result.Probe),
	})
}

func (h *Handler) ListGoals(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	var statuses []string
	if raw := r.URL.Query().Get("status"); raw != "" {
		statuses = strings.Split(raw, ",")
	}
	goals, err := h.Service.ListGoals(r.Context(), uid, statuses)
	if err != nil {
		writeAppError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toGoalSummaryDTOs(goals))
}

func (h *Handler) GetGoal(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	goalID := chi.URLParam(r, "goalId")
	detail, err := h.Service.GetGoal(r.Context(), uid, goalID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toGoalDTO(detail.Goal, detail.Competencies, h.Service.Packs))
}

type patchGoalRequest struct {
	Title            *string `json:"title"`
	Why              *string `json:"why"`
	DefinitionOfDone *string `json:"definitionOfDone"`
	HorizonOn        *string `json:"horizonOn"`
}

func (h *Handler) PatchGoal(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	goalID := chi.URLParam(r, "goalId")

	var req patchGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, "Corpo inválido", err.Error(), "")
		return
	}

	in := app.PatchGoalInput{Title: req.Title, Why: req.Why, DefinitionOfDone: req.DefinitionOfDone}
	if req.HorizonOn != nil {
		t, err := time.Parse("2006-01-02", *req.HorizonOn)
		if err != nil {
			httpx.WriteProblem(w, http.StatusBadRequest, "horizonOn inválido, use YYYY-MM-DD", err.Error(), "")
			return
		}
		in.HorizonOn = &t
	}

	if _, err := h.Service.PatchGoal(r.Context(), uid, goalID, in); err != nil {
		writeAppError(w, err)
		return
	}
	detail, err := h.Service.GetGoal(r.Context(), uid, goalID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toGoalDTO(detail.Goal, detail.Competencies, h.Service.Packs))
}

type activateGoalRequest struct {
	PauseGoalID *string `json:"pauseGoalId"`
}

func (h *Handler) ActivateGoal(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	goalID := chi.URLParam(r, "goalId")

	var req activateGoalRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req) // corpo é opcional
	}

	result, err := h.Service.ActivateGoal(r.Context(), app.ActivateGoalInput{
		UserID:      uid,
		GoalID:      goalID,
		PauseGoalID: req.PauseGoalID,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	detail, err := h.Service.GetGoal(r.Context(), uid, goalID)
	if err != nil {
		writeAppError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"goal":   toGoalDTO(detail.Goal, detail.Competencies, h.Service.Packs),
		"action": toNextActionDTO(result.Action),
	})
}
