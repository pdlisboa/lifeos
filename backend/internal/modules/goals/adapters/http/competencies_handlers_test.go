package http

import (
	"context"
	"testing"

	"github.com/phablo/lifeos/internal/modules/goals/app"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func createGoalWithCompetencies(t *testing.T, ts *testServer, title string) (goalID, compID string) {
	t.Helper()
	created, err := ts.service.CreateGoal(context.Background(), app.CreateGoalInput{
		UserID: ts.userID, Title: title, Archetype: domain.ArchetypeSkill, PackID: "golang",
	})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	return created.Goal.ID, created.Competencies[0].ID
}

func TestSetCompetencyLevelHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, compID := createGoalWithCompetencies(t, ts, "Meta")
	_ = goalID

	rec := ts.do(t, "PUT", "/goals/x/competencies/"+compID+"/level", map[string]any{
		"level": 2, "rationale": "Consigo fazer isso sem consultar",
	})
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	c := decodeJSON[competencyDTO](t, rec)
	if c.Level == nil || *c.Level != 2 {
		t.Fatalf("level = %v, want 2", c.Level)
	}
	if c.LevelDescriptor == nil || *c.LevelDescriptor == "" {
		t.Fatal("esperava o descritor do nível resolvido a partir do pack")
	}
}

func TestSetCompetencyLevelInvalidBody(t *testing.T) {
	ts := newTestServer(t)
	_, compID := createGoalWithCompetencies(t, ts, "Meta")
	rec := ts.doRaw(t, "PUT", "/goals/x/competencies/"+compID+"/level", "{not json")
	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestSetCompetencyLevelNotFoundHandler(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "PUT", "/goals/x/competencies/00000000-0000-0000-0000-000000000000/level", map[string]any{
		"level": 2, "rationale": "qualquer",
	})
	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestCompetencyHistoryHandler(t *testing.T) {
	ts := newTestServer(t)
	_, compID := createGoalWithCompetencies(t, ts, "Meta")
	ts.do(t, "PUT", "/goals/x/competencies/"+compID+"/level", map[string]any{"level": 1, "rationale": "baseline"})

	rec := ts.do(t, "GET", "/goals/x/competencies/"+compID+"/history", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	events := decodeJSON[[]levelEventDTO](t, rec)
	if len(events) != 1 {
		t.Fatalf("esperava 1 evento, got %d", len(events))
	}
}
