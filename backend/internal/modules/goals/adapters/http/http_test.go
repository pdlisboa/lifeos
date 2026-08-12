package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/phablo/lifeos/internal/modules/goals/app"
	"github.com/phablo/lifeos/internal/modules/goals/packs"
	"github.com/phablo/lifeos/internal/platform/auth"
	"github.com/phablo/lifeos/internal/testsupport/pgtest"
)

func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}

// testServer é o Handler real (mesmo wiring de module.go) atrás do
// middleware de autenticação de verdade, rodando contra o Postgres do
// pgtest — não há mock de Service aqui: Handler.Service é concreto
// (*app.Service), e CLAUDE.md pede para não introduzir interface só para
// facilitar teste.
type testServer struct {
	router  http.Handler
	service *app.Service
	userID  string
	token   string
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	pgtest.Reset(t)

	reg, err := packs.Load()
	if err != nil {
		t.Fatalf("packs.Load: %v", err)
	}
	svc := app.NewService(pgtest.Pool(), reg, nil, nil)
	users := auth.NewUserStore(pgtest.Pool())
	sessions := auth.NewSessionStore(pgtest.Pool())

	userID := pgtest.NewUser(t)
	token, _, err := sessions.Create(context.Background(), userID, "web", "go-test", time.Hour)
	if err != nil {
		t.Fatalf("sessions.Create: %v", err)
	}

	h := &Handler{Service: svc, Users: users, Logger: slog.New(slog.NewTextHandler(os.Stderr, nil))}
	r := chi.NewRouter()
	r.Use(auth.RequireAuth(sessions))
	RegisterRoutes(r, h)

	return &testServer{router: r, service: svc, userID: userID, token: token}
}

func (ts *testServer) do(t *testing.T, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		req = httptest.NewRequest(method, target, bytes.NewReader(b))
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	req.Header.Set("Authorization", "Bearer "+ts.token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ts.router.ServeHTTP(rec, req)
	return rec
}

// doRaw envia um corpo literal (não serializado), útil para testar o
// caminho de "corpo inválido" (400) dos handlers que fazem json.Decode.
func (ts *testServer) doRaw(t *testing.T, method, target, rawBody string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, bytes.NewReader([]byte(rawBody)))
	req.Header.Set("Authorization", "Bearer "+ts.token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ts.router.ServeHTTP(rec, req)
	return rec
}

func (ts *testServer) doNoAuth(t *testing.T, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	ts.router.ServeHTTP(rec, req)
	return rec
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decodificar resposta (%s): %v", rec.Body.String(), err)
	}
	return v
}

func decodeInto(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("decodificar resposta (%s): %v", rec.Body.String(), err)
	}
}
