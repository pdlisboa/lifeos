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
	Kind         string  `json:"kind"`
	Title        *string `json:"title"`
	Body         *string `json:"body"`
	BlobKey      *string `json:"blobKey"`
	ExternalURL  *string `json:"externalUrl"`
	Language     *string `json:"language"`
	ActionID     *string `json:"actionId"`
	SupersedesID *string `json:"supersedesId"`
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
		UserID:       uid,
		GoalID:       goalID,
		ActionID:     req.ActionID,
		Kind:         domain.EvidenceKind(req.Kind),
		Title:        req.Title,
		Body:         req.Body,
		BlobKey:      req.BlobKey,
		ExternalURL:  req.ExternalURL,
		Language:     req.Language,
		SupersedesID: req.SupersedesID,
		Timezone:     h.userLocation(r.Context(), uid),
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
	ev, err := h.Service.GetEvidence(r.Context(), uid, chi.URLParam(r, "evidenceId"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toEvidenceDTO(ev))
}

func (h *Handler) ListEvidence(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	ascending := r.URL.Query().Get("order") != "desc"
	limit := parseLimit(r, 30)

	items, err := h.Service.ListEvidence(r.Context(), uid, chi.URLParam(r, "goalId"), ascending, limit)
	if err != nil {
		writeAppError(w, err)
		return
	}
	cards := make([]evidenceCardDTO, len(items))
	for i, e := range items {
		cards[i] = toEvidenceCardDTO(e)
	}
	httpx.WriteJSON(w, http.StatusOK, pageDTO[evidenceCardDTO]{Items: cards, HasMore: false})
}
