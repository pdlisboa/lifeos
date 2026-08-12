package http

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/platform/idgen"
)

// insertTrackProposal grava uma proposta de trilha pronta pra ser aceita —
// mesmo formato que HandlePlanTrack grava de verdade (dois marcos, um deles
// carriedOver, pra exercitar a cópia de status do marco concluído).
func insertTrackProposal(t *testing.T, ts *testServer, goalID string, milestonesJSON string) *domain.Proposal {
	t.Helper()
	id, err := idgen.NewUUIDv7()
	if err != nil {
		t.Fatalf("gerar id: %v", err)
	}
	p := &domain.Proposal{
		ID:        id,
		UserID:    ts.userID,
		GoalID:    &goalID,
		Kind:      domain.ProposalTrack,
		Payload:   []byte(milestonesJSON),
		Rationale: "você já demonstrou o básico, pulei o marco introdutório",
		Status:    domain.ProposalPending,
		CreatedAt: time.Now(),
	}
	if err := postgres.InsertProposal(context.Background(), ts.service.Pool, p); err != nil {
		t.Fatalf("InsertProposal: %v", err)
	}
	return p
}

const twoMilestonesPayload = `{"milestones":[
	{"ordinal":1,"title":"Escreve um worker pool básico","completionCriteria":"worker pool sem vazar goroutine","competencyKeys":["concurrency"],"carriedOver":false,"sourceLibraryTitle":null},
	{"ordinal":2,"title":"Cobre com testes de tabela","completionCriteria":"table-driven test cobrindo erro e sucesso","competencyKeys":["testing"],"carriedOver":false,"sourceLibraryTitle":null}
]}`

func TestListProposalsHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")
	insertTrackProposal(t, ts, goalID, twoMilestonesPayload)

	rec := ts.do(t, "GET", "/proposals", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	items := decodeJSON[[]proposalDTO](t, rec)
	if len(items) != 1 {
		t.Fatalf("esperava 1 proposta pendente, teve %d", len(items))
	}
	if items[0].Status != "pending" || items[0].Kind != "track" {
		t.Fatalf("proposta inesperada: %+v", items[0])
	}
	payload, ok := items[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("payload não decodificou como objeto: %v", items[0].Payload)
	}
	if _, ok := payload["milestones"]; !ok {
		t.Fatalf("payload sem 'milestones': %v", payload)
	}
}

func TestListProposalsFiltersByGoalIDHandler(t *testing.T) {
	ts := newTestServer(t)
	goal1, _ := activateReadyGoal(t, ts, "Meta 1")
	goal2, _ := activateReadyGoal(t, ts, "Meta 2")
	insertTrackProposal(t, ts, goal1, twoMilestonesPayload)
	insertTrackProposal(t, ts, goal2, twoMilestonesPayload)

	rec := ts.do(t, "GET", "/proposals?goalId="+goal1, nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	items := decodeJSON[[]proposalDTO](t, rec)
	if len(items) != 1 || *items[0].GoalID != goal1 {
		t.Fatalf("esperava só a proposta da meta 1, teve %+v", items)
	}
}

func TestAcceptProposalHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")
	p := insertTrackProposal(t, ts, goalID, twoMilestonesPayload)

	rec := ts.do(t, "POST", "/proposals/"+p.ID+"/accept", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[proposalDTO](t, rec)
	if got.Status != "accepted" {
		t.Fatalf("status = %s, want accepted", got.Status)
	}

	track, err := ts.service.GetTrack(context.Background(), ts.userID, goalID)
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
	}
	if track.Version != 2 {
		t.Fatalf("version = %d, esperava 2 (supersedeu a trilha do fallback)", track.Version)
	}
	if len(track.Milestones) != 2 || track.Milestones[0].Title != "Escreve um worker pool básico" {
		t.Fatalf("marcos inesperados: %+v", track.Milestones)
	}
	if track.Milestones[0].Status != domain.MilestoneCurrent {
		t.Fatalf("primeiro marco status = %s, want current", track.Milestones[0].Status)
	}
}

func TestAcceptProposalWithEditsHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")
	p := insertTrackProposal(t, ts, goalID, twoMilestonesPayload)

	body := map[string]any{
		"edits": map[string]any{
			"milestones": []map[string]any{
				{
					"ordinal": 1, "title": "Título editado por mim",
					"completionCriteria": "critério que eu preferi", "competencyKeys": []string{"concurrency"},
					"carriedOver": false, "sourceLibraryTitle": nil,
				},
			},
		},
	}
	rec := ts.do(t, "POST", "/proposals/"+p.ID+"/accept", body)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	track, err := ts.service.GetTrack(context.Background(), ts.userID, goalID)
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
	}
	if len(track.Milestones) != 1 || track.Milestones[0].Title != "Título editado por mim" {
		t.Fatalf("edição não foi aplicada: %+v", track.Milestones)
	}
}

func TestAcceptProposalCarriesOverCompletedMilestone(t *testing.T) {
	ts := newTestServer(t)
	goalID, actionID := activateReadyGoal(t, ts, "Meta ativa")
	_ = actionID

	fallbackTrack, err := ts.service.GetTrack(context.Background(), ts.userID, goalID)
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
	}
	firstTitle := fallbackTrack.Milestones[0].Title
	if err := fallbackTrack.Milestones[0].Complete(time.Now()); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if err := postgres.UpdateMilestone(context.Background(), ts.service.Pool, &fallbackTrack.Milestones[0]); err != nil {
		t.Fatalf("UpdateMilestone: %v", err)
	}

	payload, err := json.Marshal(map[string]any{
		"milestones": []map[string]any{
			{
				"ordinal": 1, "title": firstTitle, "completionCriteria": "critério original",
				"competencyKeys": []string{}, "carriedOver": true, "sourceLibraryTitle": nil,
			},
			{
				"ordinal": 2, "title": "Marco novo", "completionCriteria": "critério novo",
				"competencyKeys": []string{}, "carriedOver": false, "sourceLibraryTitle": nil,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	p := insertTrackProposal(t, ts, goalID, string(payload))

	rec := ts.do(t, "POST", "/proposals/"+p.ID+"/accept", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	track, err := ts.service.GetTrack(context.Background(), ts.userID, goalID)
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
	}
	if track.Milestones[0].Status != domain.MilestoneCompleted {
		t.Fatalf("marco carriedOver deveria continuar completed, ficou %s", track.Milestones[0].Status)
	}
	if track.Milestones[0].CompletedAt == nil {
		t.Fatal("completedAt do marco carriedOver deveria ter sido copiado")
	}
	if track.Milestones[1].Status != domain.MilestoneCurrent {
		t.Fatalf("segundo marco deveria virar current, ficou %s", track.Milestones[1].Status)
	}
}

func TestAcceptProposalNotFoundHandler(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "POST", "/proposals/00000000-0000-0000-0000-000000000000/accept", nil)
	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404, body=%s", rec.Code, rec.Body.String())
	}
}

func TestAcceptProposalAlreadyResolvedHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")
	p := insertTrackProposal(t, ts, goalID, twoMilestonesPayload)

	first := ts.do(t, "POST", "/proposals/"+p.ID+"/accept", nil)
	if first.Code != 200 {
		t.Fatalf("primeiro aceite falhou: status = %d, body=%s", first.Code, first.Body.String())
	}

	second := ts.do(t, "POST", "/proposals/"+p.ID+"/accept", nil)
	if second.Code != 409 {
		t.Fatalf("status = %d, want 409 (RN-07), body=%s", second.Code, second.Body.String())
	}
}

func TestRejectProposalHandler(t *testing.T) {
	ts := newTestServer(t)
	goalID, _ := activateReadyGoal(t, ts, "Meta ativa")
	p := insertTrackProposal(t, ts, goalID, twoMilestonesPayload)

	rec := ts.do(t, "POST", "/proposals/"+p.ID+"/reject", map[string]any{"reason": "prefiro a trilha atual"})
	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[proposalDTO](t, rec)
	if got.Status != "rejected" {
		t.Fatalf("status = %s, want rejected", got.Status)
	}

	// a trilha original continua valendo — rejeitar não aplica nada.
	track, err := ts.service.GetTrack(context.Background(), ts.userID, goalID)
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
	}
	if track.Version != 1 {
		t.Fatalf("version = %d, esperava 1 (proposta rejeitada não deveria ter sido aplicada)", track.Version)
	}
}
