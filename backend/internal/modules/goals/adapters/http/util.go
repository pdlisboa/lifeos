package http

import (
	"net/http"
	"strconv"
)

func parseLimit(r *http.Request, fallback int) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > 100 {
		return fallback
	}
	return n
}
