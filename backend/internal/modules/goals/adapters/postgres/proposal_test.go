package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func newProposal(t *testing.T, q Querier, userID string, goalID *string, status domain.ProposalStatus) *domain.Proposal {
	t.Helper()
	p := &domain.Proposal{
		ID:        mustID(t),
		UserID:    userID,
		GoalID:    goalID,
		Kind:      domain.ProposalTrack,
		Payload:   []byte(`{"milestones":[]}`),
		Rationale: "porque a evidência mostrou nível maior do que o registrado",
		Status:    status,
		CreatedAt: time.Now(),
	}
	if err := InsertProposal(context.Background(), q, p); err != nil {
		t.Fatalf("InsertProposal: %v", err)
	}
	return p
}

func TestInsertAndGetProposalRoundTrip(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	p := newProposal(t, q, userID, &g.ID, domain.ProposalPending)

	fetched, err := GetProposal(context.Background(), q, userID, p.ID)
	if err != nil {
		t.Fatalf("GetProposal: %v", err)
	}
	if fetched.Kind != domain.ProposalTrack || fetched.Rationale != p.Rationale {
		t.Fatalf("round-trip não bate: %+v", fetched)
	}
	if fetched.GoalID == nil || *fetched.GoalID != g.ID {
		t.Fatalf("goalID não persistiu: %+v", fetched.GoalID)
	}
	// jsonb normaliza a formatação (ex.: espaço depois de ':') — compara o
	// valor decodificado, não a string crua.
	var payload map[string]any
	if err := json.Unmarshal(fetched.Payload, &payload); err != nil {
		t.Fatalf("payload não é JSON válido: %s (%v)", fetched.Payload, err)
	}
	if _, ok := payload["milestones"]; !ok {
		t.Fatalf("payload não tem a chave 'milestones': %s", fetched.Payload)
	}
}

func TestGetProposalNotFound(t *testing.T) {
	q, userID := setup(t)
	if _, err := GetProposal(context.Background(), q, userID, mustID(t)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("erro = %v, want ErrNotFound", err)
	}
}

func TestListProposalsFiltersByStatusAndGoal(t *testing.T) {
	q, userID := setup(t)
	g1 := newGoal(t, q, userID, "Meta 1")
	g2 := newGoal(t, q, userID, "Meta 2")

	pending1 := newProposal(t, q, userID, &g1.ID, domain.ProposalPending)
	_ = newProposal(t, q, userID, &g2.ID, domain.ProposalPending)
	_ = newProposal(t, q, userID, &g1.ID, domain.ProposalAccepted)

	allPending, err := ListProposals(context.Background(), q, userID, string(domain.ProposalPending), nil)
	if err != nil {
		t.Fatalf("ListProposals: %v", err)
	}
	if len(allPending) != 2 {
		t.Fatalf("esperava 2 propostas pendentes (das duas metas), teve %d", len(allPending))
	}

	onlyGoal1, err := ListProposals(context.Background(), q, userID, string(domain.ProposalPending), &g1.ID)
	if err != nil {
		t.Fatalf("ListProposals com goalID: %v", err)
	}
	if len(onlyGoal1) != 1 || onlyGoal1[0].ID != pending1.ID {
		t.Fatalf("esperava só a proposta pendente da meta 1, teve %+v", onlyGoal1)
	}

	accepted, err := ListProposals(context.Background(), q, userID, string(domain.ProposalAccepted), nil)
	if err != nil {
		t.Fatalf("ListProposals accepted: %v", err)
	}
	if len(accepted) != 1 {
		t.Fatalf("esperava 1 proposta aceita, teve %d", len(accepted))
	}
}

func TestResolveProposalPersistsAcceptance(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	p := newProposal(t, q, userID, &g.ID, domain.ProposalPending)

	now := time.Now()
	if err := p.Accept(now); err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if err := ResolveProposal(context.Background(), q, p); err != nil {
		t.Fatalf("ResolveProposal: %v", err)
	}

	fetched, err := GetProposal(context.Background(), q, userID, p.ID)
	if err != nil {
		t.Fatalf("GetProposal: %v", err)
	}
	if fetched.Status != domain.ProposalAccepted || fetched.ResolvedAt == nil {
		t.Fatalf("aceite não persistiu: %+v", fetched)
	}
}

func TestResolveProposalPersistsRejectionReason(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	p := newProposal(t, q, userID, &g.ID, domain.ProposalPending)

	reason := "prefiro manter o marco atual"
	if err := p.Reject(&reason, time.Now()); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if err := ResolveProposal(context.Background(), q, p); err != nil {
		t.Fatalf("ResolveProposal: %v", err)
	}

	fetched, err := GetProposal(context.Background(), q, userID, p.ID)
	if err != nil {
		t.Fatalf("GetProposal: %v", err)
	}
	if fetched.Status != domain.ProposalRejected || fetched.RejectReason == nil || *fetched.RejectReason != reason {
		t.Fatalf("rejeição não persistiu: %+v", fetched)
	}
}
