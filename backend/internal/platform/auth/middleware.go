package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/phablo/lifeos/internal/platform/httpx"
)

// SessionCookieName precisa bater com securitySchemes.sessionCookie.name em openapi.yaml.
const SessionCookieName = "lifeos_session"

type ctxKey int

const userIDKey ctxKey = iota

// TokenFromRequest lê o cookie de sessão (web) ou o header Authorization (mobile),
// conforme 03-api.md §8: o mesmo endpoint atende os dois.
func TokenFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	if authz := r.Header.Get("Authorization"); strings.HasPrefix(authz, "Bearer ") {
		return strings.TrimPrefix(authz, "Bearer ")
	}
	return ""
}

func RequireAuth(sessions *SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := TokenFromRequest(r)
			if token == "" {
				httpx.WriteProblem(w, http.StatusUnauthorized, "Não autenticado", "informe cookie de sessão ou Bearer token", "")
				return
			}
			userID, err := sessions.Validate(r.Context(), token)
			if err != nil {
				httpx.WriteProblem(w, http.StatusUnauthorized, "Sessão inválida ou expirada", "", "")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey, userID)))
		})
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}
