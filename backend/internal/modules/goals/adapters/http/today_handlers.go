package http

import (
	"net/http"

	"github.com/phablo/lifeos/internal/platform/httpx"
)

func (h *Handler) GetToday(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireUser(w, r)
	if !ok {
		return
	}
	result, err := h.Service.GetToday(r.Context(), uid)
	if err != nil {
		writeAppError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toTodayDTO(result))
}
