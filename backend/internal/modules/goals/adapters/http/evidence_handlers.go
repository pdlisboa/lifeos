package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/phablo/lifeos/internal/modules/goals/app"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/platform/httpx"
)

type createEvidenceRequest struct {
	Kind          string   `json:"kind"`
	Title         *string  `json:"title"`
	Body          *string  `json:"body"`
	BlobKey       *string  `json:"blobKey"`
	ExternalURL   *string  `json:"externalUrl"`
	Language      *string  `json:"language"`
	ActionID      *string  `json:"actionId"`
	SupersedesID  *string  `json:"supersedesId"`
	CompetencyIDs []string `json:"competencyIds"`
}

// CreateEvidence responde 202: a evidência é gravada de forma síncrona (você
// nunca perde o que produziu), mas na Fatia 1 não há avaliação para
// enfileirar — o job fica para quando o A3 existir (Fatia 4).
func (h *Handler) CreateEvidence(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	goalID := chi.URLParam(r, "goalId")

	var req createEvidenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, "Corpo inválido", err.Error(), "")
		return
	}

	ev, err := h.Service.CreateEvidence(r.Context(), app.CreateEvidenceInput{
		UserID:        uid,
		GoalID:        goalID,
		ActionID:      req.ActionID,
		Kind:          domain.EvidenceKind(req.Kind),
		Title:         req.Title,
		Body:          req.Body,
		BlobKey:       req.BlobKey,
		ExternalURL:   req.ExternalURL,
		Language:      req.Language,
		SupersedesID:  req.SupersedesID,
		CompetencyIDs: req.CompetencyIDs,
		Timezone:      h.userLocation(r.Context(), uid),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{
		"evidence": toEvidenceDTO(ev),
		"job":      nil,
	})
}

func (h *Handler) GetEvidence(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	evidenceID := chi.URLParam(r, "evidenceId")
	ev, err := h.Service.GetEvidence(r.Context(), uid, evidenceID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	ec, err := h.Service.GetEvalCase(r.Context(), uid, evidenceID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	dto := toEvidenceDTO(ev)
	dto.EvalCase = toEvalCaseDTO(ec)
	httpx.WriteJSON(w, http.StatusOK, dto)
}

type markEvalCaseRequest struct {
	Note   string             `json:"note"`
	Scores []evalCaseScoreDTO `json:"scores"`
}

// MarkEvidenceEvalCase captura o gabarito humano pro conjunto de eval do A3
// (04-agentes.md §6.1) — puro dado, nenhum agente é chamado. Marcar de novo
// substitui nota e gabarito anteriores.
func (h *Handler) MarkEvidenceEvalCase(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	evidenceID := chi.URLParam(r, "evidenceId")

	var req markEvalCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, "Corpo inválido", err.Error(), "")
		return
	}
	scores := make([]domain.EvalCaseScore, len(req.Scores))
	for i, s := range req.Scores {
		scores[i] = domain.EvalCaseScore{CompetencyID: s.CompetencyID, Level: s.Level}
	}

	if _, err := h.Service.MarkEvidenceAsEvalCase(r.Context(), app.MarkEvidenceEvalCaseInput{
		UserID: uid, EvidenceID: evidenceID, Note: req.Note, Scores: scores,
	}); err != nil {
		writeAppError(w, err)
		return
	}

	ev, err := h.Service.GetEvidence(r.Context(), uid, evidenceID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	ec, err := h.Service.GetEvalCase(r.Context(), uid, evidenceID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	dto := toEvidenceDTO(ev)
	dto.EvalCase = toEvalCaseDTO(ec)
	httpx.WriteJSON(w, http.StatusOK, dto)
}

func (h *Handler) UnmarkEvidenceEvalCase(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	if err := h.Service.UnmarkEvidenceEvalCase(r.Context(), uid, chi.URLParam(r, "evidenceId")); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListEvidence(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	ascending := r.URL.Query().Get("order") != "desc"
	limit := parseLimit(r, 30)
	var competencyID *string
	if v := r.URL.Query().Get("competencyId"); v != "" {
		competencyID = &v
	}

	items, err := h.Service.ListEvidence(r.Context(), uid, chi.URLParam(r, "goalId"), competencyID, ascending, limit)
	if err != nil {
		writeAppError(w, err)
		return
	}
	cards := make([]evidenceCardDTO, len(items))
	for i, c := range items {
		cards[i] = toEvidenceCardDTO(c)
	}
	httpx.WriteJSON(w, http.StatusOK, pageDTO[evidenceCardDTO]{Items: cards, HasMore: false})
}
