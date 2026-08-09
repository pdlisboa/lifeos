package http

import "testing"

func TestCreateSessionHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")

	rec := ts.do(t, "POST", "/goals/"+goalID+"/sessions", map[string]any{
		"startedAt": "2026-01-15T10:00:00Z", "durationMin": 20,
	})
	if rec.Code != 201 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	s := decodeJSON[sessionDTO](t, rec)
	if s.DurationMin != 20 {
		t.Fatalf("durationMin = %d, want 20", s.DurationMin)
	}
}

func TestCreateSessionInvalidStartedAt(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")

	rec := ts.do(t, "POST", "/goals/"+goalID+"/sessions", map[string]any{
		"startedAt": "não é uma data", "durationMin": 20,
	})
	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400 (startedAt inválido)", rec.Code)
	}
}

func TestCreateSessionInvalidDuration(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")

	rec := ts.do(t, "POST", "/goals/"+goalID+"/sessions", map[string]any{
		"startedAt": "2026-01-15T10:00:00Z", "durationMin": 0,
	})
	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400 (RuleError sem Rule -> 400)", rec.Code)
	}
}

func TestListSessionsHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")
	ts.do(t, "POST", "/goals/"+goalID+"/sessions", map[string]any{"startedAt": "2026-01-15T10:00:00Z", "durationMin": 20})

	rec := ts.do(t, "GET", "/goals/"+goalID+"/sessions", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	page := decodeJSON[pageDTO[sessionDTO]](t, rec)
	if len(page.Items) != 1 {
		t.Fatalf("esperava 1 sessão, got %d", len(page.Items))
	}
}
