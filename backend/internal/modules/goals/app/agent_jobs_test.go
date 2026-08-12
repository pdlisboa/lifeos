package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/platform/agents"
	"github.com/phablo/lifeos/internal/platform/jobs"
)

var jobTestModels = map[agents.Tier]string{
	agents.TierFast:     "fast-model",
	agents.TierBalanced: "balanced-model",
	agents.TierStrong:   "strong-model",
}

var jobTestPrices = map[agents.Tier]agents.Price{
	agents.TierFast:     {InPerMillion: 1, OutPerMillion: 2},
	agents.TierBalanced: {InPerMillion: 5, OutPerMillion: 10},
	agents.TierStrong:   {InPerMillion: 20, OutPerMillion: 40},
}

// wireGateway pluga um Gateway de verdade (contra o Postgres do pgtest) com
// um FakeProvider — o mesmo desenho de internal/platform/agents/gateway_test.go
// — pra exercitar HandleGenerateNextAction/HandlePlanTrack de ponta a ponta
// sem rede.
func wireGateway(svc *Service, responses ...agents.FakeResponse) *agents.FakeProvider {
	provider := agents.NewFakeProvider(responses...)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc.Gateway = agents.NewGateway(svc.Pool, provider, logger, jobTestModels, jobTestPrices, 10.0)
	svc.Logger = logger
	return provider
}

func mustJobPayload(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return b
}

const validPracticeOutput = `{"title":"Escreva um worker pool que lê 3 URLs em paralelo",` +
	`"detail":"Use goroutines e um WaitGroup para buscar 3 URLs e retornar os status codes.",` +
	`"practiceFormat":"exercise","estimatedMin":22,` +
	`"minimalVariant":"Versão de 5 min: escreva só a assinatura da função e o WaitGroup.",` +
	`"competencyKey":"concurrency",` +
	`"expectedEvidence":{"kind":"code_snippet","description":"o código da função com os 3 status codes"},` +
	`"milestoneId":null,"rationale":"Sinal de concorrência aplicada, não só teoria."}`

func TestHandleGenerateNextAction_ReplacesFallbackWithAgentOutput(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")
	wireGateway(svc, agents.FakeResponse{Content: validPracticeOutput})

	job := jobs.Job{Payload: mustJobPayload(t, generateNextActionPayload{
		GoalID: active.Goal.ID, UserID: userID, FallbackActionID: active.Action.ID,
	})}
	if err := svc.HandleGenerateNextAction(context.Background(), job); err != nil {
		t.Fatalf("HandleGenerateNextAction: %v", err)
	}

	fetched, err := postgres.GetAction(context.Background(), svc.Pool, userID, active.Action.ID)
	if err != nil {
		t.Fatalf("GetAction: %v", err)
	}
	if fetched.GeneratedBy != domain.GeneratedByAgent {
		t.Fatalf("generatedBy = %s, want agent", fetched.GeneratedBy)
	}
	if fetched.Title != "Escreva um worker pool que lê 3 URLs em paralelo" {
		t.Fatalf("título não foi trocado pelo do agente: %q", fetched.Title)
	}
	if fetched.EstimatedMin != 22 {
		t.Fatalf("estimatedMin = %d, want 22", fetched.EstimatedMin)
	}
	if fetched.Status != domain.ActionPending {
		t.Fatalf("status = %s, want pending (RN-03 não deveria ter sido afetado)", fetched.Status)
	}
}

func TestHandleGenerateNextAction_NoGatewayIsNoop(t *testing.T) {
	svc, userID := newFixture(t) // Gateway nil por padrão
	active := readyGoal(t, svc, userID, "Meta ativa")

	job := jobs.Job{Payload: mustJobPayload(t, generateNextActionPayload{
		GoalID: active.Goal.ID, UserID: userID, FallbackActionID: active.Action.ID,
	})}
	if err := svc.HandleGenerateNextAction(context.Background(), job); err != nil {
		t.Fatalf("HandleGenerateNextAction: %v", err)
	}

	fetched, err := postgres.GetAction(context.Background(), svc.Pool, userID, active.Action.ID)
	if err != nil {
		t.Fatalf("GetAction: %v", err)
	}
	if fetched.GeneratedBy != domain.GeneratedByFallback {
		t.Fatalf("generatedBy = %s, want fallback (sem gateway, nada deveria mudar)", fetched.GeneratedBy)
	}
}

func TestHandleGenerateNextAction_ProviderErrorKeepsFallback(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")
	wireGateway(svc, agents.FakeResponse{Err: errors.New("timeout simulado")})

	job := jobs.Job{Payload: mustJobPayload(t, generateNextActionPayload{
		GoalID: active.Goal.ID, UserID: userID, FallbackActionID: active.Action.ID,
	})}
	if err := svc.HandleGenerateNextAction(context.Background(), job); err != nil {
		t.Fatalf("HandleGenerateNextAction não deveria propagar erro do provider (fallback já resolve): %v", err)
	}

	fetched, err := postgres.GetAction(context.Background(), svc.Pool, userID, active.Action.ID)
	if err != nil {
		t.Fatalf("GetAction: %v", err)
	}
	if fetched.GeneratedBy != domain.GeneratedByFallback {
		t.Fatalf("generatedBy = %s, want fallback (falha do provider não deveria sobrescrever)", fetched.GeneratedBy)
	}
}

func TestHandleGenerateNextAction_SkipsWhenActionAlreadyResolved(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")
	wireGateway(svc, agents.FakeResponse{Content: validPracticeOutput})

	// simula a corrida: a pessoa já concluiu a ação antes do job do A2 rodar.
	completed, err := svc.CompleteAction(context.Background(), CompleteActionInput{UserID: userID, ActionID: active.Action.ID})
	if err != nil {
		t.Fatalf("CompleteAction: %v", err)
	}

	job := jobs.Job{Payload: mustJobPayload(t, generateNextActionPayload{
		GoalID: active.Goal.ID, UserID: userID, FallbackActionID: active.Action.ID,
	})}
	if err := svc.HandleGenerateNextAction(context.Background(), job); err != nil {
		t.Fatalf("HandleGenerateNextAction: %v", err)
	}

	fetched, err := postgres.GetAction(context.Background(), svc.Pool, userID, active.Action.ID)
	if err != nil {
		t.Fatalf("GetAction: %v", err)
	}
	if fetched.Status != domain.ActionCompleted {
		t.Fatalf("status = %s, want completed (não deveria ter sido tocado)", fetched.Status)
	}
	_ = completed
}

func TestHandleGenerateNextAction_UnknownActionIsNoop(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")
	wireGateway(svc, agents.FakeResponse{Content: validPracticeOutput})

	job := jobs.Job{Payload: mustJobPayload(t, generateNextActionPayload{
		GoalID: active.Goal.ID, UserID: userID, FallbackActionID: "00000000-0000-0000-0000-000000000000",
	})}
	if err := svc.HandleGenerateNextAction(context.Background(), job); err != nil {
		t.Fatalf("HandleGenerateNextAction não deveria propagar erro para ação inexistente: %v", err)
	}
}

func TestHandleGenerateNextAction_InvalidPayloadReturnsError(t *testing.T) {
	svc, _ := newFixture(t)
	job := jobs.Job{Payload: json.RawMessage(`{"goalId": não é json}`)}
	if err := svc.HandleGenerateNextAction(context.Background(), job); err == nil {
		t.Fatal("esperava erro para payload malformado")
	}
}

const validPlannerOutput = `{"milestones":[` +
	`{"ordinal":1,"title":"Escreve um worker pool básico com channels","completionCriteria":"worker pool sem vazar goroutine, com WaitGroup","competencyKeys":["concurrency"],"carriedOver":false,"sourceLibraryTitle":null},` +
	`{"ordinal":2,"title":"Cobre o worker pool com testes de tabela","completionCriteria":"table-driven test cobrindo erro e sucesso","competencyKeys":["testing"],"carriedOver":false,"sourceLibraryTitle":null},` +
	`{"ordinal":3,"title":"Usa context para cancelar o worker pool","completionCriteria":"context.Context propagado e cancelamento testado","competencyKeys":["concurrency"],"carriedOver":false,"sourceLibraryTitle":null},` +
	`{"ordinal":4,"title":"Abre uma PR real usando o que praticou","completionCriteria":"PR aberta num repositório real, com testes passando","competencyKeys":["concurrency","testing"],"carriedOver":false,"sourceLibraryTitle":null}` +
	`],"rationale":"Você já conhece o básico de goroutines, então comecei pelo worker pool.","noChangeNeeded":false}`

func TestHandlePlanTrack_CreatesProposal(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")
	wireGateway(svc, agents.FakeResponse{Content: validPlannerOutput})

	job := jobs.Job{Payload: mustJobPayload(t, planTrackPayload{GoalID: active.Goal.ID, UserID: userID})}
	if err := svc.HandlePlanTrack(context.Background(), job); err != nil {
		t.Fatalf("HandlePlanTrack: %v", err)
	}

	proposals, err := postgres.ListProposals(context.Background(), svc.Pool, userID, string(domain.ProposalPending), &active.Goal.ID)
	if err != nil {
		t.Fatalf("ListProposals: %v", err)
	}
	if len(proposals) != 1 {
		t.Fatalf("esperava 1 proposta de trilha, teve %d", len(proposals))
	}
	if proposals[0].Kind != domain.ProposalTrack {
		t.Fatalf("kind = %s, want track", proposals[0].Kind)
	}
	if proposals[0].AgentCallID == nil {
		t.Error("agentCallID vazio — proposta deveria rastrear a chamada real (04-agentes.md §9)")
	}
}

func TestHandlePlanTrack_NoChangeNeededSkipsProposal(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")
	noChange := `{"milestones":[` +
		`{"ordinal":1,"title":"Marco A com título válido","completionCriteria":"critério observável A","competencyKeys":["concurrency"],"carriedOver":true,"sourceLibraryTitle":null},` +
		`{"ordinal":2,"title":"Marco B com título válido","completionCriteria":"critério observável B","competencyKeys":["testing"],"carriedOver":true,"sourceLibraryTitle":null},` +
		`{"ordinal":3,"title":"Marco C com título válido","completionCriteria":"critério observável C","competencyKeys":["concurrency"],"carriedOver":true,"sourceLibraryTitle":null},` +
		`{"ordinal":4,"title":"Marco D com título válido","completionCriteria":"critério observável D","competencyKeys":["testing"],"carriedOver":true,"sourceLibraryTitle":null}` +
		`],"rationale":"A trilha atual já reflete bem o seu nível, não mudaria nada.","noChangeNeeded":true}`
	wireGateway(svc, agents.FakeResponse{Content: noChange})

	job := jobs.Job{Payload: mustJobPayload(t, planTrackPayload{GoalID: active.Goal.ID, UserID: userID})}
	if err := svc.HandlePlanTrack(context.Background(), job); err != nil {
		t.Fatalf("HandlePlanTrack: %v", err)
	}

	proposals, err := postgres.ListProposals(context.Background(), svc.Pool, userID, string(domain.ProposalPending), &active.Goal.ID)
	if err != nil {
		t.Fatalf("ListProposals: %v", err)
	}
	if len(proposals) != 0 {
		t.Fatalf("noChangeNeeded não deveria criar proposta, teve %d", len(proposals))
	}
}

func TestHandlePlanTrack_NoGatewayIsNoop(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")

	job := jobs.Job{Payload: mustJobPayload(t, planTrackPayload{GoalID: active.Goal.ID, UserID: userID})}
	if err := svc.HandlePlanTrack(context.Background(), job); err != nil {
		t.Fatalf("HandlePlanTrack: %v", err)
	}

	proposals, err := postgres.ListProposals(context.Background(), svc.Pool, userID, string(domain.ProposalPending), &active.Goal.ID)
	if err != nil {
		t.Fatalf("ListProposals: %v", err)
	}
	if len(proposals) != 0 {
		t.Fatalf("sem gateway não deveria criar proposta, teve %d", len(proposals))
	}
}

func TestHandlePlanTrack_InvalidPayloadReturnsError(t *testing.T) {
	svc, _ := newFixture(t)
	job := jobs.Job{Payload: json.RawMessage(`{"goalId": não é json}`)}
	if err := svc.HandlePlanTrack(context.Background(), job); err == nil {
		t.Fatal("esperava erro para payload malformado")
	}
}

func TestHandlePlanTrack_ProviderErrorSkipsProposalSilently(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")
	wireGateway(svc, agents.FakeResponse{Err: errors.New("timeout simulado")})

	job := jobs.Job{Payload: mustJobPayload(t, planTrackPayload{GoalID: active.Goal.ID, UserID: userID})}
	if err := svc.HandlePlanTrack(context.Background(), job); err != nil {
		t.Fatalf("HandlePlanTrack não deveria propagar erro do provider: %v", err)
	}

	proposals, err := postgres.ListProposals(context.Background(), svc.Pool, userID, string(domain.ProposalPending), &active.Goal.ID)
	if err != nil {
		t.Fatalf("ListProposals: %v", err)
	}
	if len(proposals) != 0 {
		t.Fatalf("falha do provider não deveria gerar proposta, teve %d", len(proposals))
	}
}

func TestHandlePlanTrack_UnknownGoalIsNoop(t *testing.T) {
	svc, userID := newFixture(t)
	wireGateway(svc, agents.FakeResponse{Content: validPlannerOutput})

	job := jobs.Job{Payload: mustJobPayload(t, planTrackPayload{
		GoalID: "00000000-0000-0000-0000-000000000000", UserID: userID,
	})}
	if err := svc.HandlePlanTrack(context.Background(), job); err != nil {
		t.Fatalf("HandlePlanTrack não deveria propagar erro para meta inexistente: %v", err)
	}
}
