package domain

import "time"

type ProposalKind string

const (
	ProposalTrack             ProposalKind = "track"
	ProposalMilestoneRevision ProposalKind = "milestone_revision"
	ProposalLevelChange       ProposalKind = "level_change"
	ProposalScopeReduction    ProposalKind = "scope_reduction"
	ProposalPackDraft         ProposalKind = "pack_draft"
	ProposalCompetencySet     ProposalKind = "competency_set"
)

type ProposalStatus string

const (
	ProposalPending  ProposalStatus = "pending"
	ProposalAccepted ProposalStatus = "accepted"
	ProposalRejected ProposalStatus = "rejected"
	ProposalExpired  ProposalStatus = "expired"
)

// Proposal é o que um agente produz em vez de escrever direto no núcleo
// (RN-07, 02-modelo-de-dados.md §8): nada dela vira estado até um aceite
// explícito. Payload fica em bytes crus (não json.RawMessage) — quem
// serializa/interpreta o formato específico de cada Kind é app/postgres, não
// o domínio (00-negocio.md, "domain não importa nada do projeto").
type Proposal struct {
	ID     string
	UserID string
	GoalID *string

	Kind      ProposalKind
	Payload   []byte
	Rationale string

	AgentCallID *string

	Status       ProposalStatus
	ResolvedAt   *time.Time
	RejectReason *string

	ExpiresAt *time.Time
	CreatedAt time.Time
}

// Accept é o único jeito de uma proposta virar estado (RN-07) — quem chama
// ainda precisa aplicar o Payload por conta própria; este método só valida
// a transição.
func (p *Proposal) Accept(now time.Time) error {
	if p.Status != ProposalPending {
		return newRuleError("RN-07", "proposta não está pendente")
	}
	p.Status = ProposalAccepted
	p.ResolvedAt = &now
	return nil
}

func (p *Proposal) Reject(reason *string, now time.Time) error {
	if p.Status != ProposalPending {
		return newRuleError("RN-07", "proposta não está pendente")
	}
	p.Status = ProposalRejected
	p.ResolvedAt = &now
	p.RejectReason = reason
	return nil
}
