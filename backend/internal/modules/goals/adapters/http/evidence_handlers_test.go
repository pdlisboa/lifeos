package http

import "testing"

func TestCreateAndGetEvidenceHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")

	rec := ts.do(t, "POST", "/goals/"+goalID+"/evidence", map[string]any{
		"kind": "code_snippet", "body": "func main() {}",
	})
	if rec.Code != 202 {
		t.Fatalf("status = %d, want 202, body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Evidence evidenceDTO `json:"evidence"`
		Job      any         `json:"job"`
	}
	decodeInto(t, rec, &body)
	if body.Evidence.ID == "" {
		t.Fatal("evidência sem ID na resposta")
	}
	if body.Job != nil {
		t.Fatalf("job deveria ser nil na Fatia 1 (sem avaliador), got %v", body.Job)
	}

	getRec := ts.do(t, "GET", "/evidence/"+body.Evidence.ID, nil)
	if getRec.Code != 200 {
		t.Fatalf("GET /evidence/{id} status = %d, body=%s", getRec.Code, getRec.Body.String())
	}
	fetched := decodeJSON[evidenceDTO](t, getRec)
	if fetched.Body == nil || *fetched.Body != "func main() {}" {
		t.Fatalf("body não bate: %+v", fetched)
	}
}

func TestCreateEvidenceEmptyBodyHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")

	rec := ts.do(t, "POST", "/goals/"+goalID+"/evidence", map[string]any{"kind": "code_snippet"})
	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400 (evidência vazia)", rec.Code)
	}
}

func TestGetEvidenceNotFoundHandler(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "GET", "/evidence/00000000-0000-0000-0000-000000000000", nil)
	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestListEvidenceHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")
	for i := 0; i < 2; i++ {
		ts.do(t, "POST", "/goals/"+goalID+"/evidence", map[string]any{"kind": "code_snippet", "body": "conteúdo"})
	}

	rec := ts.do(t, "GET", "/goals/"+goalID+"/evidence", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	page := decodeJSON[pageDTO[evidenceCardDTO]](t, rec)
	if len(page.Items) != 2 {
		t.Fatalf("esperava 2 itens, got %d", len(page.Items))
	}
	if page.HasMore {
		t.Fatal("hasMore deveria ser false (Fatia 1 não pagina de verdade)")
	}
}

func TestListEvidenceOrderAndLimitParams(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")
	ts.do(t, "POST", "/goals/"+goalID+"/evidence", map[string]any{"kind": "code_snippet", "body": "primeira"})
	ts.do(t, "POST", "/goals/"+goalID+"/evidence", map[string]any{"kind": "code_snippet", "body": "segunda"})

	rec := ts.do(t, "GET", "/goals/"+goalID+"/evidence?order=desc&limit=1", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	page := decodeJSON[pageDTO[evidenceCardDTO]](t, rec)
	if len(page.Items) != 1 {
		t.Fatalf("limit=1 deveria devolver 1 item, got %d", len(page.Items))
	}
	if page.Items[0].Body == nil || *page.Items[0].Body != "segunda" {
		t.Fatalf("order=desc deveria trazer a mais recente primeiro, got %+v", page.Items[0])
	}

	// limit inválido cai no fallback (parseLimit).
	fallbackRec := ts.do(t, "GET", "/goals/"+goalID+"/evidence?limit=abc", nil)
	if fallbackRec.Code != 200 {
		t.Fatalf("status = %d, body=%s", fallbackRec.Code, fallbackRec.Body.String())
	}
	fallbackPage := decodeJSON[pageDTO[evidenceCardDTO]](t, fallbackRec)
	if len(fallbackPage.Items) != 2 {
		t.Fatalf("limit inválido deveria cair no fallback (30) e trazer as 2 evidências, got %d", len(fallbackPage.Items))
	}
}
