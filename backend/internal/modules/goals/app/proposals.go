package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/agents"
	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/platform/idgen"
)

func (s *Service) ListProposals(ctx context.Context, userID, status string, goalID *string) ([]domain.Proposal, error) {
	return postgres.ListProposals(ctx, s.Pool, userID, status, goalID)
}

// trackProposalPayload é o formato gravado em proposal.payload por
// HandlePlanTrack — os mesmos milestones que planner.schema.json descreve.
type trackProposalPayload struct {
	Milestones []agents.PlannerMilestoneOutput `json:"milestones"`
}

type AcceptProposalInput struct {
	UserID     string
	ProposalID string
	// Milestones, quando não-nil, substitui o payload original da proposta
	// — é o "editar antes de aceitar" (P8: o agente propõe, você decide).
	Milestones []agents.PlannerMilestoneOutput
}

// AcceptProposal é o único jeito de uma proposta virar estado (RN-07). Só
// `kind: track` está implementado nesta rodada — os outros tipos ainda não
// têm agente nenhum produzindo-os.
func (s *Service) AcceptProposal(ctx context.Context, in AcceptProposalInput) (*domain.Proposal, error) {
	var result *domain.Proposal
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		p, err := postgres.GetProposal(ctx, tx, in.UserID, in.ProposalID)
		if err != nil {
			return err
		}
		if p.Kind != domain.ProposalTrack {
			return domain.NewRuleError("", fmt.Sprintf("aceite automático ainda não implementado para propostas do tipo %s", p.Kind))
		}
		if p.GoalID == nil {
			return domain.NewRuleError("", "proposta de trilha sem meta associada")
		}

		milestonesOut, err := resolveTrackMilestones(p.Payload, in.Milestones)
		if err != nil {
			return err
		}

		now := time.Now()
		if err := p.Accept(now); err != nil {
			return err
		}
		if err := postgres.ResolveProposal(ctx, tx, p); err != nil {
			return err
		}

		g, err := postgres.GetGoal(ctx, tx, in.UserID, *p.GoalID)
		if err != nil {
			return err
		}
		if err := applyTrackProposal(ctx, tx, g, milestonesOut, now); err != nil {
			return err
		}

		result = p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func resolveTrackMilestones(payload []byte, edited []agents.PlannerMilestoneOutput) ([]agents.PlannerMilestoneOutput, error) {
	if edited != nil {
		if len(edited) == 0 {
			return nil, domain.NewRuleError("", "trilha editada sem marcos")
		}
		return edited, nil
	}
	var decoded trackProposalPayload
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, fmt.Errorf("decodificar payload da proposta: %w", err)
	}
	if len(decoded.Milestones) == 0 {
		return nil, domain.NewRuleError("", "proposta de trilha sem marcos")
	}
	return decoded.Milestones, nil
}

// applyTrackProposal materializa os marcos aceitos como uma nova versão da
// trilha, supersedendo a vigente se houver. Marcos com carriedOver copiam
// status/completedAt do marco de mesmo título na trilha anterior — marcos
// concluídos são intocáveis numa revisão (04-agentes.md §4.1); o resto some
// que ApplyTrackStatuses resolve (primeiro aberto vira current).
func applyTrackProposal(ctx context.Context, tx pgx.Tx, g *domain.Goal, milestonesOut []agents.PlannerMilestoneOutput, now time.Time) error {
	comps, err := postgres.ListCompetenciesByGoal(ctx, tx, g.ID)
	if err != nil {
		return err
	}
	keyToID := make(map[string]string, len(comps))
	for _, c := range comps {
		keyToID[c.PackKey] = c.ID
	}

	newVersion := 1
	existingByTitle := make(map[string]domain.Milestone)
	existing, err := postgres.GetCurrentTrack(ctx, tx, g.ID)
	if err == nil {
		newVersion = existing.Version + 1
		for _, m := range existing.Milestones {
			existingByTitle[normalizeTitle(m.Title)] = m
		}
		if err := postgres.SupersedeTrack(ctx, tx, existing.ID); err != nil {
			return err
		}
	} else if !errors.Is(err, postgres.ErrNotFound) {
		return err
	}

	trackID, err := idgen.NewUUIDv7()
	if err != nil {
		return err
	}
	track := &domain.Track{ID: trackID, GoalID: g.ID, UserID: g.UserID, Version: newVersion, GeneratedBy: "agent", AcceptedAt: &now}
	if err := postgres.InsertTrack(ctx, tx, track); err != nil {
		return err
	}

	milestones := make([]domain.Milestone, 0, len(milestonesOut))
	for i, mo := range milestonesOut {
		status := domain.MilestoneLocked
		var completedAt *time.Time
		if mo.CarriedOver {
			if orig, ok := existingByTitle[normalizeTitle(mo.Title)]; ok {
				status = orig.Status
				completedAt = orig.CompletedAt
			}
		}
		var ids []string
		for _, key := range mo.CompetencyKeys {
			if id, ok := keyToID[key]; ok {
				ids = append(ids, id)
			}
		}
		mid, err := idgen.NewUUIDv7()
		if err != nil {
			return err
		}
		milestones = append(milestones, domain.Milestone{
			ID: mid, TrackID: trackID, GoalID: g.ID, UserID: g.UserID,
			Ordinal: i + 1, Title: mo.Title, CompletionCriteria: mo.CompletionCriteria,
			Status: status, CompletedAt: completedAt, CompetencyIDs: ids,
		})
	}
	domain.ApplyTrackStatuses(milestones)

	for i := range milestones {
		if err := postgres.InsertMilestone(ctx, tx, &milestones[i]); err != nil {
			return err
		}
		for _, cid := range milestones[i].CompetencyIDs {
			if err := postgres.InsertMilestoneCompetency(ctx, tx, milestones[i].ID, cid); err != nil {
				return err
			}
		}
	}
	return nil
}

func normalizeTitle(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// RejectProposal encerra a proposta sem aplicar nada — o payload nunca
// chega perto do núcleo (RN-07).
func (s *Service) RejectProposal(ctx context.Context, userID, proposalID string, reason *string) (*domain.Proposal, error) {
	var result *domain.Proposal
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		p, err := postgres.GetProposal(ctx, tx, userID, proposalID)
		if err != nil {
			return err
		}
		if err := p.Reject(reason, time.Now()); err != nil {
			return err
		}
		if err := postgres.ResolveProposal(ctx, tx, p); err != nil {
			return err
		}
		result = p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// RequestTrackRevision pede ao A1 uma nova trilha sob demanda
// (04-agentes.md §4.1). Só enfileira — quem processa é HandlePlanTrack no
// worker; não há endpoint de status de job neste contrato, então o id
// devolvido é só para preencher JobAccepted, não referencia a linha real.
func (s *Service) RequestTrackRevision(ctx context.Context, userID, goalID string) (string, error) {
	if _, err := postgres.GetGoal(ctx, s.Pool, userID, goalID); err != nil {
		return "", err
	}
	if err := enqueuePlanTrack(ctx, s.Pool, goalID, userID, "revisão pedida por você"); err != nil {
		return "", err
	}
	jobID, err := idgen.NewUUIDv7()
	if err != nil {
		return "", err
	}
	return jobID, nil
}
