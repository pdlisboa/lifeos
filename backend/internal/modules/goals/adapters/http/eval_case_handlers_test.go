package http

import (
	"context"
	"testing"
)

func createEvidenceForEval(t *testing.T, ts *testServer, goalID string) string {
	t.Helper()
	rec := ts.do(t, "POST", "/goals/"+goalID+"/evidence", map[string]any{
		"kind": "code_snippet", "body": "func worker() {}",
	})
	if rec.Code != 202 {
		t.Fatalf("criar evidência: status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var created struct {
		Evidence evidenceDTO `json:"evidence"`
	}
	decodeInto(t, rec, &created)
	return created.Evidence.ID
}

func TestMarkAndGetEvidenceEvalCaseHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")
	detail, err := ts.service.GetGoal(context.Background(), ts.userID, goalID)
	if err != nil {
		t.Fatalf("GetGoal: %v", err)
	}
	comp := detail.Competencies[0].ID
	evidenceID := createEvidenceForEval(t, ts, goalID)

	markRec := ts.do(t, "POST", "/evidence/"+evidenceID+"/eval-case", map[string]any{
		"note":   "gabarito claro de nível 2",
		"scores": []map[string]any{{"competencyId": comp, "level": 2}},
	})
	if markRec.Code != 200 {
		t.Fatalf("POST eval-case: status = %d, body=%s", markRec.Code, markRec.Body.String())
	}
	marked := decodeJSON[evidenceDTO](t, markRec)
	if marked.EvalCase == nil || marked.EvalCase.Note != "gabarito claro de nível 2" {
		t.Fatalf("evalCase da resposta = %+v, want nota preenchida", marked.EvalCase)
	}
	if len(marked.EvalCase.Scores) != 1 || marked.EvalCase.Scores[0].CompetencyID != comp || marked.EvalCase.Scores[0].Level != 2 {
		t.Fatalf("scores = %+v", marked.EvalCase.Scores)
	}

	getRec := ts.do(t, "GET", "/evidence/"+evidenceID, nil)
	fetched := decodeJSON[evidenceDTO](t, getRec)
	if fetched.EvalCase == nil || fetched.EvalCase.Note != "gabarito claro de nível 2" {
		t.Fatalf("GET /evidence/{id} deveria trazer evalCase, got %+v", fetched.EvalCase)
	}
}

func TestGetEvidenceWithoutEvalCaseHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")
	evidenceID := createEvidenceForEval(t, ts, goalID)

	getRec := ts.do(t, "GET", "/evidence/"+evidenceID, nil)
	fetched := decodeJSON[evidenceDTO](t, getRec)
	if fetched.EvalCase != nil {
		t.Fatalf("evalCase = %+v, want nil (evidência não marcada)", fetched.EvalCase)
	}
}

func TestMarkEvidenceEvalCaseEmptyBodyHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")
	evidenceID := createEvidenceForEval(t, ts, goalID)

	rec := ts.do(t, "POST", "/evidence/"+evidenceID+"/eval-case", map[string]any{"note": "", "scores": []map[string]any{}})
	if rec.Code < 300 {
		t.Fatalf("nota vazia e sem gabarito deveria ser rejeitado, status = %d", rec.Code)
	}
}

func TestMarkEvidenceEvalCaseRejectsUnknownCompetencyHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")
	evidenceID := createEvidenceForEval(t, ts, goalID)

	rec := ts.do(t, "POST", "/evidence/"+evidenceID+"/eval-case", map[string]any{
		"note":   "nota",
		"scores": []map[string]any{{"competencyId": "00000000-0000-0000-0000-000000000000", "level": 2}},
	})
	if rec.Code < 300 {
		t.Fatalf("competência inexistente deveria ser rejeitada, status = %d", rec.Code)
	}
}

func TestUnmarkEvidenceEvalCaseHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")
	detail, err := ts.service.GetGoal(context.Background(), ts.userID, goalID)
	if err != nil {
		t.Fatalf("GetGoal: %v", err)
	}
	comp := detail.Competencies[0].ID
	evidenceID := createEvidenceForEval(t, ts, goalID)

	ts.do(t, "POST", "/evidence/"+evidenceID+"/eval-case", map[string]any{
		"note":   "nota",
		"scores": []map[string]any{{"competencyId": comp, "level": 3}},
	})

	delRec := ts.do(t, "DELETE", "/evidence/"+evidenceID+"/eval-case", nil)
	if delRec.Code != 204 {
		t.Fatalf("DELETE eval-case: status = %d, body=%s", delRec.Code, delRec.Body.String())
	}

	getRec := ts.do(t, "GET", "/evidence/"+evidenceID, nil)
	fetched := decodeJSON[evidenceDTO](t, getRec)
	if fetched.EvalCase != nil {
		t.Fatalf("evalCase depois de desmarcar = %+v, want nil", fetched.EvalCase)
	}
}

func TestMarkEvidenceEvalCaseNotFoundHandler(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "POST", "/evidence/00000000-0000-0000-0000-000000000000/eval-case", map[string]any{
		"note":   "nota",
		"scores": []map[string]any{},
	})
	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
