package http

import "testing"

func TestGetTodayHandler(t *testing.T) {
	ts := newTestServer(t)
	activateReadyGoal(t, ts, "Meta 1")
	activateReadyGoal(t, ts, "Meta 2")

	rec := ts.do(t, "GET", "/today", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	today := decodeJSON[todayDTO](t, rec)
	if len(today.Goals) != 2 {
		t.Fatalf("esperava 2 metas em Hoje, got %d", len(today.Goals))
	}
	for _, g := range today.Goals {
		if g.Action.ID == "" {
			t.Fatalf("meta %+v sem ação em Hoje (RN-03 quebrada)", g.Goal)
		}
	}
	if today.Nudge != nil {
		t.Fatal("nudge deveria ser nil na Fatia 1 (sem Coach)")
	}
}

func TestGetTodayEmptyHandler(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "GET", "/today", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	today := decodeJSON[todayDTO](t, rec)
	if len(today.Goals) != 0 {
		t.Fatalf("esperava 0 metas, got %d", len(today.Goals))
	}
}
