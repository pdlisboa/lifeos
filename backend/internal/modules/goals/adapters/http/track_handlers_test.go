package http

import "testing"

func TestGetTrackHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")

	rec := ts.do(t, "GET", "/goals/"+goalID+"/track", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	track := decodeJSON[trackDTO](t, rec)
	if len(track.Milestones) == 0 {
		t.Fatal("esperava marcos na trilha")
	}
	if track.Milestones[0].Status != "current" {
		t.Fatalf("primeiro marco status = %s, want current", track.Milestones[0].Status)
	}
}

func TestGetTrackNotFoundHandler(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "GET", "/goals/00000000-0000-0000-0000-000000000000/track", nil)
	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
