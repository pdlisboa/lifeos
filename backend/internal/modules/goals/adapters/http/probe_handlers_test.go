package http

import "testing"

func TestProbeFlowHandlers(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := createGoalWithCompetencies(t, ts, "Meta com sondagem")

	rec := ts.do(t, "GET", "/goals/"+goalID+"/probe", nil)
	if rec.Code != 200 {
		t.Fatalf("GET probe status = %d, body=%s", rec.Code, rec.Body.String())
	}
	probe := decodeJSON[probeDTO](t, rec)
	if probe.CurrentQuestion == nil {
		t.Fatal("esperava pergunta atual na sondagem recém-criada")
	}

	answerRec := ts.do(t, "POST", "/goals/"+goalID+"/probe/answer", map[string]any{
		"turnId": probe.CurrentQuestion.ID, "answer": "sim",
	})
	if answerRec.Code != 200 {
		t.Fatalf("POST probe/answer status = %d, body=%s", answerRec.Code, answerRec.Body.String())
	}
	var body struct {
		Probe        probeDTO      `json:"probe"`
		NextQuestion *probeTurnDTO `json:"nextQuestion"`
	}
	decodeInto(t, answerRec, &body)
	if body.NextQuestion == nil {
		t.Fatal("esperava follow-up após responder sim")
	}
}

func TestSkipProbeHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := createGoalWithCompetencies(t, ts, "Meta com sondagem")

	rec := ts.do(t, "POST", "/goals/"+goalID+"/probe/skip", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	probe := decodeJSON[probeDTO](t, rec)
	if probe.Status != "skipped" {
		t.Fatalf("status = %s, want skipped", probe.Status)
	}
}

func TestAnswerProbeGoalNotFoundHandler(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "POST", "/goals/00000000-0000-0000-0000-000000000000/probe/answer", map[string]any{
		"turnId": "x", "answer": "sim",
	})
	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
