package http

import (
	"net/http/httptest"
	"testing"
)

func TestRequireAuthRejectsWithoutToken(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.doNoAuth(t, "GET", "/today")
	if rec.Code != 401 {
		t.Fatalf("status = %d, want 401 sem token", rec.Code)
	}
}

func TestCreateAndGetGoal(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "POST", "/goals/", map[string]any{
		"title": "Aprender Go de verdade", "archetype": "skill", "packId": "golang",
	})
	if rec.Code != 201 {
		t.Fatalf("POST /goals status = %d, body=%s", rec.Code, rec.Body.String())
	}
	created := decodeJSON[map[string]any](t, rec)
	goalObj := created["goal"].(map[string]any)
	goalID := goalObj["id"].(string)
	if len(goalObj["competencies"].([]any)) != 6 {
		t.Fatalf("esperava 6 competências na resposta, got %+v", goalObj["competencies"])
	}

	getRec := ts.do(t, "GET", "/goals/"+goalID+"/", nil)
	if getRec.Code != 200 {
		t.Fatalf("GET /goals/{id} status = %d, body=%s", getRec.Code, getRec.Body.String())
	}
	fetched := decodeJSON[goalDTO](t, getRec)
	if fetched.ID != goalID || fetched.Title != "Aprender Go de verdade" {
		t.Fatalf("goal buscada não bate: %+v", fetched)
	}
	if fetched.ReadyToActivate {
		t.Fatal("meta sem definição de pronto não deveria estar pronta para ativar")
	}
}

func TestCreateGoalInvalidBody(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "POST", "/goals/", map[string]any{"title": "Go", "archetype": "skill", "packId": "golang"})
	if rec.Code != 400 {
		t.Fatalf("título curto demais deveria virar 400 (RuleError sem Rule), got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListGoals(t *testing.T) {
	ts := newTestServer(t)
	ts.do(t, "POST", "/goals/", map[string]any{"title": "Meta 1", "archetype": "skill", "packId": "golang"})
	ts.do(t, "POST", "/goals/", map[string]any{"title": "Meta 2", "archetype": "skill", "packId": "golang"})

	rec := ts.do(t, "GET", "/goals/", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	list := decodeJSON[[]goalSummaryDTO](t, rec)
	if len(list) != 2 {
		t.Fatalf("esperava 2 metas, got %d", len(list))
	}
}

// TestPatchGoalAllFields cobre os quatro campos independentes de PatchGoal
// (title, why, definitionOfDone, horizonOn), incluindo o parse de data.
func TestPatchGoalAllFields(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "POST", "/goals/", map[string]any{"title": "Meta original", "archetype": "skill", "packId": "golang"})
	created := decodeJSON[map[string]any](t, rec)
	goalID := created["goal"].(map[string]any)["id"].(string)

	patchRec := ts.do(t, "PATCH", "/goals/"+goalID+"/", map[string]any{
		"title": "Meta com novo título", "why": "porque decidi", "horizonOn": "2026-12-31",
	})
	if patchRec.Code != 200 {
		t.Fatalf("status = %d, body=%s", patchRec.Code, patchRec.Body.String())
	}
	fetched := decodeJSON[goalDTO](t, patchRec)
	if fetched.Title != "Meta com novo título" {
		t.Fatalf("title = %q", fetched.Title)
	}
	if fetched.Why == nil || *fetched.Why != "porque decidi" {
		t.Fatalf("why = %v", fetched.Why)
	}
	if fetched.HorizonOn == nil || *fetched.HorizonOn != "2026-12-31" {
		t.Fatalf("horizonOn = %v, want 2026-12-31", fetched.HorizonOn)
	}
}

func TestPatchGoalInvalidHorizonOn(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "POST", "/goals/", map[string]any{"title": "Meta", "archetype": "skill", "packId": "golang"})
	created := decodeJSON[map[string]any](t, rec)
	goalID := created["goal"].(map[string]any)["id"].(string)

	patchRec := ts.do(t, "PATCH", "/goals/"+goalID+"/", map[string]any{"horizonOn": "não é uma data"})
	if patchRec.Code != 400 {
		t.Fatalf("status = %d, want 400 (horizonOn inválido)", patchRec.Code)
	}
}

func TestPatchGoalInvalidBody(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "POST", "/goals/", map[string]any{"title": "Meta", "archetype": "skill", "packId": "golang"})
	created := decodeJSON[map[string]any](t, rec)
	goalID := created["goal"].(map[string]any)["id"].(string)

	patchRec := ts.doRaw(t, "PATCH", "/goals/"+goalID+"/", "{not json")
	if patchRec.Code != 400 {
		t.Fatalf("status = %d, want 400", patchRec.Code)
	}
}

// TestListGoalsFilterByStatusHandler cobre o parse de ?status=a,b na borda
// HTTP (goals_handlers.go: strings.Split).
func TestListGoalsFilterByStatusHandler(t *testing.T) {
	ts := newTestServer(t)
	ts.do(t, "POST", "/goals/", map[string]any{"title": "Rascunho", "archetype": "skill", "packId": "golang"})
	activateReadyGoal(t, ts, "Ativa")

	rec := ts.do(t, "GET", "/goals/?status=active", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	list := decodeJSON[[]goalSummaryDTO](t, rec)
	if len(list) != 1 || list[0].Status != "active" {
		t.Fatalf("esperava só a meta ativa, got %+v", list)
	}
}

func TestGetGoalNotFound(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "GET", "/goals/00000000-0000-0000-0000-000000000000/", nil)
	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// TestActivateGoalFlowAndRN02 percorre: criar -> falha RN-01 (409) -> setar
// DoD (PATCH) -> ativar com sucesso -> quarta ativação estoura RN-02 (409
// com activeGoals).
func TestActivateGoalFlowAndRN02(t *testing.T) {
	ts := newTestServer(t)

	createOne := func(title string) string {
		rec := ts.do(t, "POST", "/goals/", map[string]any{"title": title, "archetype": "skill", "packId": "golang"})
		created := decodeJSON[map[string]any](t, rec)
		return created["goal"].(map[string]any)["id"].(string)
	}
	activate := func(goalID string) *httptest.ResponseRecorder {
		return ts.do(t, "POST", "/goals/"+goalID+"/activate", nil)
	}
	setDoD := func(goalID string) {
		rec := ts.do(t, "PATCH", "/goals/"+goalID+"/", map[string]any{"definitionOfDone": "Critério observável qualquer"})
		if rec.Code != 200 {
			t.Fatalf("PATCH DoD falhou: %d %s", rec.Code, rec.Body.String())
		}
	}

	first := createOne("Meta 1")
	if rec := activate(first); rec.Code != 409 {
		t.Fatalf("ativar sem DoD deveria ser 409 (RN-01), got %d: %s", rec.Code, rec.Body.String())
	}
	setDoD(first)
	if rec := activate(first); rec.Code != 200 {
		t.Fatalf("ativar com DoD deveria funcionar, got %d: %s", rec.Code, rec.Body.String())
	}

	for _, title := range []string{"Meta 2", "Meta 3"} {
		id := createOne(title)
		setDoD(id)
		if rec := activate(id); rec.Code != 200 {
			t.Fatalf("ativar %s falhou: %d %s", title, rec.Code, rec.Body.String())
		}
	}

	fourth := createOne("Meta 4")
	setDoD(fourth)
	rec := activate(fourth)
	if rec.Code != 409 {
		t.Fatalf("quarta ativação deveria ser 409 (RN-02), got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Rule        string `json:"rule"`
		ActiveGoals []any  `json:"activeGoals"`
	}
	decodeInto(t, rec, &body)
	if body.Rule != "RN-02" || len(body.ActiveGoals) != 3 {
		t.Fatalf("corpo do 409 não bate: %+v", body)
	}
}
