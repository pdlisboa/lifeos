package http

import (
	"context"
	"testing"

	"github.com/phablo/lifeos/internal/modules/goals/app"
)

func TestGetDeltaHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")

	rec := ts.do(t, "GET", "/goals/"+goalID+"/delta", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	d := decodeJSON[deltaPanelDTO](t, rec)
	if d.GoalID != goalID {
		t.Fatalf("goalId = %s, want %s", d.GoalID, goalID)
	}
	if d.Headline != nil {
		t.Fatal("headline deveria ser nil na Fatia 1 (sem Coach)")
	}
}

// TestGetDeltaWithLevelSetIncludesWindow cobre o branch de deltaWindowDays
// (baseline já congelado) em toDeltaPanelDTO.
func TestGetDeltaWithLevelSetIncludesWindow(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")

	detail, err := ts.service.GetGoal(context.Background(), ts.userID, goalID)
	if err != nil {
		t.Fatalf("GetGoal: %v", err)
	}
	compID := detail.Competencies[0].ID
	if _, err := ts.service.SetCompetencyLevel(context.Background(), app.SetCompetencyLevelInput{
		UserID: ts.userID, CompetencyID: compID, Level: 2, Rationale: "baseline",
	}); err != nil {
		t.Fatalf("SetCompetencyLevel: %v", err)
	}

	rec := ts.do(t, "GET", "/goals/"+goalID+"/delta", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	d := decodeJSON[deltaPanelDTO](t, rec)
	var found bool
	for _, c := range d.Competencies {
		if c.ID == compID {
			found = true
			if c.DeltaWindowDays == nil {
				t.Fatal("esperava deltaWindowDays preenchido após baseline congelado")
			}
			if c.Delta == nil || *c.Delta != 0 {
				t.Fatalf("delta = %v, want 0 (nível atual == baseline)", c.Delta)
			}
		}
	}
	if !found {
		t.Fatal("competência avaliada não apareceu no painel de delta")
	}
}

func TestGetDeltaNotFoundHandler(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "GET", "/goals/00000000-0000-0000-0000-000000000000/delta", nil)
	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGetConsistencyHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")

	rec := ts.do(t, "GET", "/goals/"+goalID+"/consistency", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	c := decodeJSON[consistencyDTO](t, rec)
	if c.WindowDays != 30 {
		t.Fatalf("windowDays = %d, want 30", c.WindowDays)
	}
	if c.Label == "" {
		t.Fatal("label não deveria ser vazio")
	}
}
