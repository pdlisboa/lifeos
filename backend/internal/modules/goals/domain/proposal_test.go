package domain

import (
	"testing"
	"time"
)

func TestProposal_Accept(t *testing.T) {
	p := &Proposal{Status: ProposalPending}
	now := time.Now()
	if err := p.Accept(now); err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if p.Status != ProposalAccepted {
		t.Errorf("status = %s, esperava accepted", p.Status)
	}
	if p.ResolvedAt == nil || !p.ResolvedAt.Equal(now) {
		t.Errorf("resolvedAt não foi gravado corretamente")
	}
}

func TestProposal_Accept_RejectsNonPending(t *testing.T) {
	p := &Proposal{Status: ProposalAccepted}
	err := p.Accept(time.Now())
	if err == nil {
		t.Fatal("esperava erro ao aceitar proposta que já não está pendente")
	}
	var re *RuleError
	if !isRuleErrorWithRule(err, &re, "RN-07") {
		t.Errorf("esperava RuleError RN-07, teve: %v", err)
	}
}

func TestProposal_Reject(t *testing.T) {
	p := &Proposal{Status: ProposalPending}
	reason := "não é o que eu esperava"
	now := time.Now()
	if err := p.Reject(&reason, now); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if p.Status != ProposalRejected {
		t.Errorf("status = %s, esperava rejected", p.Status)
	}
	if p.RejectReason == nil || *p.RejectReason != reason {
		t.Errorf("rejectReason não foi gravado corretamente")
	}
}

func TestProposal_Reject_RejectsNonPending(t *testing.T) {
	p := &Proposal{Status: ProposalRejected}
	if err := p.Reject(nil, time.Now()); err == nil {
		t.Fatal("esperava erro ao rejeitar proposta que já não está pendente")
	}
}

func isRuleErrorWithRule(err error, target **RuleError, rule string) bool {
	re, ok := err.(*RuleError)
	if !ok {
		return false
	}
	*target = re
	return re.Rule == rule
}
