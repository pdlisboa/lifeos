package http

import (
	"context"
	"testing"

	"github.com/phablo/lifeos/internal/modules/goals/app"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

// activateReadyGoal cria e ativa uma meta de Go via camada app diretamente
// (mais rápido que ir pelos endpoints), devolvendo goalID e a ação pendente
// gerada pelo fallback (RN-03).
func activateReadyGoal(t *testing.T, ts *testServer, title string) (goalID string, actionID string) {
	t.Helper()
	created, err := ts.service.CreateGoal(context.Background(), app.CreateGoalInput{
		UserID: ts.userID, Title: title, Archetype: domain.ArchetypeSkill, PackID: "golang",
	})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	dod := "Critério observável qualquer"
	if _, err := ts.service.PatchGoal(context.Background(), ts.userID, created.Goal.ID, app.PatchGoalInput{DefinitionOfDone: &dod}); err != nil {
		t.Fatalf("PatchGoal: %v", err)
	}
	result, err := ts.service.ActivateGoal(context.Background(), app.ActivateGoalInput{UserID: ts.userID, GoalID: created.Goal.ID})
	if err != nil {
		t.Fatalf("ActivateGoal: %v", err)
	}
	return created.Goal.ID, result.Action.ID
}

func TestGetPendingActionHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, actionID := activateReadyGoal(t, ts, "Meta ativa")

	rec := ts.do(t, "GET", "/goals/"+goalID+"/action", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	a := decodeJSON[nextActionDTO](t, rec)
	if a.ID != actionID {
		t.Fatalf("ID = %s, want %s", a.ID, actionID)
	}
}

func TestCreateUserActionHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")

	rec := ts.do(t, "POST", "/goals/"+goalID+"/action", map[string]any{"title": "Minha ação de 20 min"})
	if rec.Code != 201 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	a := decodeJSON[nextActionDTO](t, rec)
	if a.GeneratedBy != "user" {
		t.Fatalf("generatedBy = %s, want user", a.GeneratedBy)
	}
}

// TestCompleteActionHandlerAlwaysReturnsNext cobre RN-03 na borda HTTP: o
// corpo sempre traz "next" preenchido.
func TestCompleteActionHandlerAlwaysReturnsNext(t *testing.T) {
	ts := newTestServer(t)
	_, actionID := activateReadyGoal(t, ts, "Meta ativa")

	rec := ts.do(t, "POST", "/actions/"+actionID+"/complete", map[string]any{"durationMin": 15})
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Completed nextActionDTO `json:"completed"`
		Next      nextActionDTO `json:"next"`
		Session   *sessionDTO   `json:"session"`
	}
	decodeInto(t, rec, &body)
	if body.Completed.Status != "completed" {
		t.Fatalf("completed.status = %s, want completed", body.Completed.Status)
	}
	if body.Next.ID == "" || body.Next.ID == body.Completed.ID {
		t.Fatalf("next deveria ser uma ação nova: %+v", body.Next)
	}
	if body.Session == nil {
		t.Fatal("informar durationMin deveria criar sessão")
	}
}

func TestCompleteActionNotFoundHandler(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "POST", "/actions/00000000-0000-0000-0000-000000000000/complete", nil)
	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestSkipActionHandler(t *testing.T) {
	ts := newTestServer(t)
	_, actionID := activateReadyGoal(t, ts, "Meta ativa")

	rec := ts.do(t, "POST", "/actions/"+actionID+"/skip", map[string]any{"reason": "too_hard"})
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Next nextActionDTO `json:"next"`
	}
	decodeInto(t, rec, &body)
	if body.Next.ID == "" {
		t.Fatal("esperava next preenchido")
	}
}

func TestSkipActionInvalidBodyHandler(t *testing.T) {
	ts := newTestServer(t)
	_, actionID := activateReadyGoal(t, ts, "Meta ativa")
	rec := ts.doRaw(t, "POST", "/actions/"+actionID+"/skip", "não é json")
	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400 para corpo inválido", rec.Code)
	}
}
