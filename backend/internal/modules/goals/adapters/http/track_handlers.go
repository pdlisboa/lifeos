package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/phablo/lifeos/internal/platform/httpx"
)

func (h *Handler) GetTrack(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	track, err := h.Service.GetTrack(r.Context(), uid, chi.URLParam(r, "goalId"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toTrackDTO(track))
}

// RequestTrackRevision pede uma trilha nova ao A1 (04-agentes.md §4.1, "sob
// demanda"). A revisão chega como proposta — RN-07 — então a resposta é só
// o aceite do pedido, não a trilha em si.
func (h *Handler) RequestTrackRevision(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	jobID, err := h.Service.RequestTrackRevision(r.Context(), uid, chi.URLParam(r, "goalId"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, jobAcceptedDTO{JobID: jobID, State: "queued"})
}
