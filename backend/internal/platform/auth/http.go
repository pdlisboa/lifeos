package auth

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/phablo/lifeos/internal/platform/httpx"
)

// Handler implementa /auth/login, /auth/logout e /me conforme backend/api/openapi.yaml.
type Handler struct {
	Users        *UserStore
	Sessions     *SessionStore
	Logger       *slog.Logger
	Limiter      *RateLimiter
	WebTTL       time.Duration
	MobileTTL    time.Duration
	CookieSecure bool
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Client   string `json:"client"`
}

type userResponse struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	Timezone       string `json:"timezone"`
	QuietHoursFrom *int32 `json:"quietHoursFrom"`
	QuietHoursTo   *int32 `json:"quietHoursTo"`
}

func toUserResponse(u User) userResponse {
	return userResponse{
		ID:             u.ID,
		Email:          u.Email,
		Timezone:       u.Timezone,
		QuietHoursFrom: u.QuietHoursFrom,
		QuietHoursTo:   u.QuietHoursTo,
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if ok, retryAfter := h.Limiter.Allow(ClientIP(r)); !ok {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
		httpx.WriteProblem(w, http.StatusTooManyRequests, "Muitas tentativas de login", "aguarde antes de tentar novamente", "")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteProblem(w, http.StatusBadRequest, "Corpo inválido", err.Error(), "")
		return
	}
	if req.Client != "web" && req.Client != "mobile" {
		httpx.WriteProblem(w, http.StatusBadRequest, "client deve ser 'web' ou 'mobile'", "", "")
		return
	}

	user, err := h.Users.ByEmail(r.Context(), req.Email)
	if err != nil {
		httpx.WriteProblem(w, http.StatusUnauthorized, "Credenciais inválidas", "", "")
		return
	}
	if ok, err := VerifyPassword(req.Password, user.PasswordHash); err != nil || !ok {
		httpx.WriteProblem(w, http.StatusUnauthorized, "Credenciais inválidas", "", "")
		return
	}

	ttl := h.WebTTL
	if req.Client == "mobile" {
		ttl = h.MobileTTL
	}
	token, expiresAt, err := h.Sessions.Create(r.Context(), user.ID, req.Client, r.UserAgent(), ttl)
	if err != nil {
		h.Logger.Error("criar sessão", "err", err)
		httpx.WriteProblem(w, http.StatusInternalServerError, "Erro ao criar sessão", "", "")
		return
	}

	// web recebe cookie HttpOnly e token nulo; mobile recebe o token no corpo (03-api.md §8).
	resp := map[string]any{
		"user":      toUserResponse(user),
		"token":     nil,
		"expiresAt": expiresAt.Format(time.RFC3339),
	}
	if req.Client == "web" {
		http.SetCookie(w, &http.Cookie{
			Name:     SessionCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   h.CookieSecure,
			SameSite: http.SameSiteLaxMode,
			Expires:  expiresAt,
		})
	} else {
		resp["token"] = token
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if token := TokenFromRequest(r); token != "" {
		if err := h.Sessions.Revoke(r.Context(), token); err != nil {
			h.Logger.Error("revogar sessão", "err", err)
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteProblem(w, http.StatusUnauthorized, "Não autenticado", "", "")
		return
	}
	user, err := h.Users.ByID(r.Context(), userID)
	if err != nil {
		httpx.WriteProblem(w, http.StatusNotFound, "Usuário não encontrado", "", "")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toUserResponse(user))
}
