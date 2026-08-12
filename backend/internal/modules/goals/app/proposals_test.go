package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/agents"
	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/platform/idgen"
)

func insertProposal(t *testing.T, svc *Service, userID, goalID string, payload []byte) *domain.Proposal {
	t.Helper()
	id, err := idgen.NewUUIDv7()
	if err != nil {
		t.Fatalf("gerar id: %v", err)
	}
	p := &domain.Proposal{
		ID:        id,
		UserID:    userID,
		GoalID:    &goalID,
		Kind:      domain.ProposalTrack,
		Payload:   payload,
		Rationale: "porque a evidência mostrou nível maior do que o registrado",
		Status:    domain.ProposalPending,
		CreatedAt: time.Now(),
	}
	if err := postgres.InsertProposal(context.Background(), svc.Pool, p); err != nil {
		t.Fatalf("InsertProposal: %v", err)
	}
	return p
}

const twoMilestonesTrackPayload = `{"milestones":[
	{"ordinal":1,"title":"Escreve um worker pool básico","completionCriteria":"worker pool sem vazar goroutine","competencyKeys":["concurrency"],"carriedOver":false,"sourceLibraryTitle":null},
	{"ordinal":2,"title":"Cobre com testes de tabela","completionCriteria":"table-driven test cobrindo erro e sucesso","competencyKeys":["testing"],"carriedOver":false,"sourceLibraryTitle":null}
]}`

func TestListProposals_FiltersByStatusAndGoal(t *testing.T) {
	svc, userID := newFixture(t)
	g1 := readyGoal(t, svc, userID, "Meta 1")
	g2 := readyGoal(t, svc, userID, "Meta 2")
	insertProposal(t, svc, userID, g1.Goal.ID, []byte(twoMilestonesTrackPayload))
	insertProposal(t, svc, userID, g2.Goal.ID, []byte(twoMilestonesTrackPayload))

	all, err := svc.ListProposals(context.Background(), userID, string(domain.ProposalPending), nil)
	if err != nil {
		t.Fatalf("ListProposals: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("esperava 2 propostas, teve %d", len(all))
	}

	onlyG1, err := svc.ListProposals(context.Background(), userID, string(domain.ProposalPending), &g1.Goal.ID)
	if err != nil {
		t.Fatalf("ListProposals com goalID: %v", err)
	}
	if len(onlyG1) != 1 || *onlyG1[0].GoalID != g1.Goal.ID {
		t.Fatalf("esperava só a proposta da meta 1, teve %+v", onlyG1)
	}
}

func TestAcceptProposal_AppliesTrackAsNewVersion(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")
	p := insertProposal(t, svc, userID, active.Goal.ID, []byte(twoMilestonesTrackPayload))

	accepted, err := svc.AcceptProposal(context.Background(), AcceptProposalInput{UserID: userID, ProposalID: p.ID})
	if err != nil {
		t.Fatalf("AcceptProposal: %v", err)
	}
	if accepted.Status != domain.ProposalAccepted {
		t.Fatalf("status = %s, want accepted", accepted.Status)
	}

	track, err := svc.GetTrack(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
	}
	if track.Version != 2 {
		t.Fatalf("version = %d, esperava 2 (supersedeu o fallback)", track.Version)
	}
	if len(track.Milestones) != 2 {
		t.Fatalf("esperava 2 marcos, teve %d", len(track.Milestones))
	}
	if track.Milestones[0].Status != domain.MilestoneCurrent {
		t.Fatalf("primeiro marco status = %s, want current", track.Milestones[0].Status)
	}
	if len(track.Milestones[0].CompetencyIDs) != 1 {
		t.Fatalf("esperava competencyKeys mapeados para IDs reais, teve %+v", track.Milestones[0].CompetencyIDs)
	}
}

func TestAcceptProposal_WithEditsReplacesPayload(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")
	p := insertProposal(t, svc, userID, active.Goal.ID, []byte(twoMilestonesTrackPayload))

	edited := []agents.PlannerMilestoneOutput{
		{Ordinal: 1, Title: "Título editado", CompletionCriteria: "critério editado", CompetencyKeys: []string{"concurrency"}},
	}
	if _, err := svc.AcceptProposal(context.Background(), AcceptProposalInput{UserID: userID, ProposalID: p.ID, Milestones: edited}); err != nil {
		t.Fatalf("AcceptProposal: %v", err)
	}

	track, err := svc.GetTrack(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
	}
	if len(track.Milestones) != 1 || track.Milestones[0].Title != "Título editado" {
		t.Fatalf("edição não foi aplicada: %+v", track.Milestones)
	}
}

func TestAcceptProposal_EmptyEditsIsRejected(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")
	p := insertProposal(t, svc, userID, active.Goal.ID, []byte(twoMilestonesTrackPayload))

	_, err := svc.AcceptProposal(context.Background(), AcceptProposalInput{
		UserID: userID, ProposalID: p.ID, Milestones: []agents.PlannerMilestoneOutput{},
	})
	if err == nil {
		t.Fatal("esperava erro ao aceitar com lista de marcos editada vazia")
	}
}

func TestAcceptProposal_CarriesOverCompletedMilestoneStatus(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")

	track, err := svc.GetTrack(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
	}
	firstTitle := track.Milestones[0].Title
	if err := track.Milestones[0].Complete(time.Now()); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if err := postgres.UpdateMilestone(context.Background(), svc.Pool, &track.Milestones[0]); err != nil {
		t.Fatalf("UpdateMilestone: %v", err)
	}

	edited := []agents.PlannerMilestoneOutput{
		{Ordinal: 1, Title: firstTitle, CompletionCriteria: "critério original", CarriedOver: true},
		{Ordinal: 2, Title: "Marco novo", CompletionCriteria: "critério novo"},
	}
	p := insertProposal(t, svc, userID, active.Goal.ID, []byte(`{"milestones":[]}`))
	if _, err := svc.AcceptProposal(context.Background(), AcceptProposalInput{UserID: userID, ProposalID: p.ID, Milestones: edited}); err != nil {
		t.Fatalf("AcceptProposal: %v", err)
	}

	newTrack, err := svc.GetTrack(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
	}
	if newTrack.Milestones[0].Status != domain.MilestoneCompleted || newTrack.Milestones[0].CompletedAt == nil {
		t.Fatalf("marco carriedOver deveria continuar completed: %+v", newTrack.Milestones[0])
	}
	if newTrack.Milestones[1].Status != domain.MilestoneCurrent {
		t.Fatalf("segundo marco deveria virar current: %+v", newTrack.Milestones[1])
	}
}

func TestAcceptProposal_RejectsNonTrackKind(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")

	id, err := idgen.NewUUIDv7()
	if err != nil {
		t.Fatalf("gerar id: %v", err)
	}
	p := &domain.Proposal{
		ID: id, UserID: userID, GoalID: &active.Goal.ID, Kind: domain.ProposalLevelChange,
		Payload: []byte(`{}`), Rationale: "razão qualquer", Status: domain.ProposalPending, CreatedAt: time.Now(),
	}
	if err := postgres.InsertProposal(context.Background(), svc.Pool, p); err != nil {
		t.Fatalf("InsertProposal: %v", err)
	}

	if _, err := svc.AcceptProposal(context.Background(), AcceptProposalInput{UserID: userID, ProposalID: p.ID}); err == nil {
		t.Fatal("esperava erro — level_change ainda não tem aceite automático implementado")
	}
}

func TestAcceptProposal_AlreadyResolvedIsRN07Conflict(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")
	p := insertProposal(t, svc, userID, active.Goal.ID, []byte(twoMilestonesTrackPayload))

	if _, err := svc.AcceptProposal(context.Background(), AcceptProposalInput{UserID: userID, ProposalID: p.ID}); err != nil {
		t.Fatalf("primeiro aceite: %v", err)
	}
	_, err := svc.AcceptProposal(context.Background(), AcceptProposalInput{UserID: userID, ProposalID: p.ID})
	var ruleErr *domain.RuleError
	if !errors.As(err, &ruleErr) || ruleErr.Rule != "RN-07" {
		t.Fatalf("esperava RuleError RN-07, teve: %v", err)
	}
}

func TestRejectProposal_DoesNotApplyAnything(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")
	p := insertProposal(t, svc, userID, active.Goal.ID, []byte(twoMilestonesTrackPayload))

	reason := "prefiro a trilha atual"
	rejected, err := svc.RejectProposal(context.Background(), userID, p.ID, &reason)
	if err != nil {
		t.Fatalf("RejectProposal: %v", err)
	}
	if rejected.Status != domain.ProposalRejected || rejected.RejectReason == nil || *rejected.RejectReason != reason {
		t.Fatalf("proposta rejeitada não bate: %+v", rejected)
	}

	track, err := svc.GetTrack(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
	}
	if track.Version != 1 {
		t.Fatalf("version = %d, esperava 1 (nada deveria ter sido aplicado)", track.Version)
	}
}

func TestRejectProposal_NotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.RejectProposal(context.Background(), userID, "00000000-0000-0000-0000-000000000000", nil)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("erro = %v, want ErrNotFound", err)
	}
}

func TestRejectProposal_AlreadyResolvedIsRN07Conflict(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")
	p := insertProposal(t, svc, userID, active.Goal.ID, []byte(twoMilestonesTrackPayload))

	if _, err := svc.RejectProposal(context.Background(), userID, p.ID, nil); err != nil {
		t.Fatalf("primeira rejeição: %v", err)
	}
	_, err := svc.RejectProposal(context.Background(), userID, p.ID, nil)
	var ruleErr *domain.RuleError
	if !errors.As(err, &ruleErr) || ruleErr.Rule != "RN-07" {
		t.Fatalf("esperava RuleError RN-07, teve: %v", err)
	}
}

func TestAcceptProposal_NotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.AcceptProposal(context.Background(), AcceptProposalInput{UserID: userID, ProposalID: "00000000-0000-0000-0000-000000000000"})
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("erro = %v, want ErrNotFound", err)
	}
}

func TestRequestTrackRevision_EnqueuesJob(t *testing.T) {
	svc, userID := newFixture(t)
	active := readyGoal(t, svc, userID, "Meta ativa")

	jobID, err := svc.RequestTrackRevision(context.Background(), userID, active.Goal.ID)
	if err != nil {
		t.Fatalf("RequestTrackRevision: %v", err)
	}
	if jobID == "" {
		t.Fatal("esperava um jobID não vazio")
	}

	var count int
	err = svc.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM job WHERE kind = 'plan_track' AND status = 'queued'`,
	).Scan(&count)
	if err != nil {
		t.Fatalf("consultar job: %v", err)
	}
	if count != 1 {
		t.Fatalf("esperava 1 job plan_track, teve %d", count)
	}
}

func TestRequestTrackRevision_UnknownGoal(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.RequestTrackRevision(context.Background(), userID, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("erro = %v, want ErrNotFound", err)
	}
}
