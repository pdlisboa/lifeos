package http

import (
	"context"
	"testing"
)

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

func TestRequestTrackRevisionHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")

	rec := ts.do(t, "POST", "/goals/"+goalID+"/track", nil)
	if rec.Code != 202 {
		t.Fatalf("status = %d, want 202, body=%s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[jobAcceptedDTO](t, rec)
	if got.JobID == "" || got.State != "queued" {
		t.Fatalf("resposta inesperada: %+v", got)
	}

	var count int
	err := ts.service.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM job WHERE kind = 'plan_track' AND status = 'queued'`,
	).Scan(&count)
	if err != nil {
		t.Fatalf("consultar job: %v", err)
	}
	if count != 1 {
		t.Fatalf("esperava 1 job plan_track enfileirado, teve %d", count)
	}
}

func TestRequestTrackRevisionNotFoundHandler(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "POST", "/goals/00000000-0000-0000-0000-000000000000/track", nil)
	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404, body=%s", rec.Code, rec.Body.String())
	}
}
