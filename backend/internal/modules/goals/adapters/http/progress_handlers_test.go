package http

import (
	"context"
	"testing"
	"time"

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

func TestGetProjectionUnavailableHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")

	rec := ts.do(t, "GET", "/goals/"+goalID+"/projection", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	p := decodeJSON[projectionDTO](t, rec)
	if p.Available {
		t.Fatal("Available deveria ser false sem sessão nenhuma")
	}
	if p.Reason == nil || *p.Reason == "" {
		t.Fatal("Reason deveria explicar a ausência de dado")
	}
	if p.MinutesPerWeek != nil || p.WeeksToNextMin != nil || p.WeeksToNextMax != nil {
		t.Fatal("nunca inventa previsão: campos de ritmo/chegada deveriam ser nil quando indisponível")
	}
}

// TestGetProjectionAvailableHandler cobre a regra de >= 3 semanas de
// histórico (§7.2) registrando sessões de verdade via API, espaçadas o
// suficiente pra passar do limiar.
func TestGetProjectionAvailableHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")

	now := time.Now().UTC()
	for _, daysAgo := range []int{25, 15, 5, 1} {
		startedAt := now.AddDate(0, 0, -daysAgo).Format(time.RFC3339)
		rec := ts.do(t, "POST", "/goals/"+goalID+"/sessions", map[string]any{
			"startedAt": startedAt, "durationMin": 30,
		})
		if rec.Code != 201 {
			t.Fatalf("criar sessão (%d dias atrás): status = %d, body=%s", daysAgo, rec.Code, rec.Body.String())
		}
	}

	rec := ts.do(t, "GET", "/goals/"+goalID+"/projection", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	p := decodeJSON[projectionDTO](t, rec)
	if !p.Available {
		t.Fatalf("Available deveria ser true com >= 3 semanas de histórico, reason=%v", p.Reason)
	}
	if p.MinutesPerWeek == nil || *p.MinutesPerWeek <= 0 {
		t.Fatalf("MinutesPerWeek = %v, want > 0", p.MinutesPerWeek)
	}
	if p.NextMilestone != nil || p.WeeksToNextMin != nil || p.WeeksToNextMax != nil || p.IfYouDouble != nil {
		t.Fatal("sem estimativa de esforço no modelo, esses campos devem continuar nil (nunca inventar previsão)")
	}
}

func TestGetProjectionNotFoundHandler(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "GET", "/goals/00000000-0000-0000-0000-000000000000/projection", nil)
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
