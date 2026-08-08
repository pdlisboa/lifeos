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
