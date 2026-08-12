package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres/sqlcgen"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func InsertProposal(ctx context.Context, q Querier, p *domain.Proposal) error {
	err := sqlcgen.New(q).InsertProposal(ctx, sqlcgen.InsertProposalParams{
		ID:          p.ID,
		UserID:      p.UserID,
		GoalID:      p.GoalID,
		Kind:        string(p.Kind),
		Payload:     p.Payload,
		Rationale:   p.Rationale,
		AgentCallID: p.AgentCallID,
		Status:      string(p.Status),
		ExpiresAt:   p.ExpiresAt,
		CreatedAt:   p.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("inserir proposal: %w", err)
	}
	return nil
}

func GetProposal(ctx context.Context, q Querier, userID, proposalID string) (*domain.Proposal, error) {
	r, err := sqlcgen.New(q).GetProposal(ctx, sqlcgen.GetProposalParams{ID: proposalID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("buscar proposal: %w", err)
	}
	return toDomainProposal(r), nil
}

// ListProposals segue o padrão de ListEvidenceByGoal: consultas nomeadas
// separadas por combinação de filtro em vez de SQL dinâmico — goalID nil
// lista todas as propostas do status pedido, não-nil restringe a uma meta.
func ListProposals(ctx context.Context, q Querier, userID, status string, goalID *string) ([]domain.Proposal, error) {
	qs := sqlcgen.New(q)
	var rows []sqlcgen.Proposal
	var err error
	if goalID == nil {
		rows, err = qs.ListProposalsByUserAndStatus(ctx, sqlcgen.ListProposalsByUserAndStatusParams{UserID: userID, Status: status})
	} else {
		rows, err = qs.ListProposalsByUserStatusAndGoal(ctx, sqlcgen.ListProposalsByUserStatusAndGoalParams{UserID: userID, Status: status, GoalID: goalID})
	}
	if err != nil {
		return nil, fmt.Errorf("listar proposals: %w", err)
	}
	out := make([]domain.Proposal, len(rows))
	for i, r := range rows {
		out[i] = *toDomainProposal(r)
	}
	return out, nil
}

// ResolveProposal grava a transição pending → accepted/rejected feita no
// domínio (RN-07) — a struct já validou, aqui só persiste o resultado.
func ResolveProposal(ctx context.Context, q Querier, p *domain.Proposal) error {
	err := sqlcgen.New(q).ResolveProposal(ctx, sqlcgen.ResolveProposalParams{
		ID:           p.ID,
		Status:       string(p.Status),
		ResolvedAt:   p.ResolvedAt,
		RejectReason: p.RejectReason,
	})
	if err != nil {
		return fmt.Errorf("resolver proposal: %w", err)
	}
	return nil
}

func toDomainProposal(r sqlcgen.Proposal) *domain.Proposal {
	return &domain.Proposal{
		ID:           r.ID,
		UserID:       r.UserID,
		GoalID:       r.GoalID,
		Kind:         domain.ProposalKind(r.Kind),
		Payload:      r.Payload,
		Rationale:    r.Rationale,
		AgentCallID:  r.AgentCallID,
		Status:       domain.ProposalStatus(r.Status),
		ResolvedAt:   r.ResolvedAt,
		RejectReason: r.RejectReason,
		ExpiresAt:    r.ExpiresAt,
		CreatedAt:    r.CreatedAt,
	}
}
