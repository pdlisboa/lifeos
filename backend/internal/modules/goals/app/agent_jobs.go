package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/agents"
	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/platform/db"
	"github.com/phablo/lifeos/internal/platform/idgen"
	"github.com/phablo/lifeos/internal/platform/jobs"
)

const (
	jobKindPlanTrack          = "plan_track"
	jobKindGenerateNextAction = "generate_next_action"
)

type planTrackPayload struct {
	GoalID         string `json:"goalId"`
	UserID         string `json:"userId"`
	RevisionReason string `json:"revisionReason"`
}

type generateNextActionPayload struct {
	GoalID           string `json:"goalId"`
	UserID           string `json:"userId"`
	FallbackActionID string `json:"fallbackActionId"`
}

// enqueuePlanTrack pede ao A1 uma trilha (04-agentes.md §4.1: roda no fim da
// sondagem e sob demanda). unique_key por meta colapsa pedidos repetidos —
// uma revisão em voo já cobre a próxima.
func enqueuePlanTrack(ctx context.Context, tx db.TX, goalID, userID, reason string) error {
	q := jobs.NewQueue(tx)
	payload := planTrackPayload{GoalID: goalID, UserID: userID, RevisionReason: reason}
	if err := q.Enqueue(ctx, jobKindPlanTrack, payload, jobs.WithUniqueKey(goalID)); err != nil {
		return fmt.Errorf("enfileirar %s: %w", jobKindPlanTrack, err)
	}
	return nil
}

// enqueueGenerateNextAction pede ao A2 uma ação melhor que o fallback recém
// criado (04-agentes.md §4.2). unique_key por ação — não por meta — porque
// cada ação de fallback merece sua própria tentativa, mesmo que a anterior
// ainda esteja em voo (§9 do plano: evita colidir com um job de uma ação já
// resolvida).
func enqueueGenerateNextAction(ctx context.Context, tx db.TX, goalID, userID, fallbackActionID string) error {
	q := jobs.NewQueue(tx)
	payload := generateNextActionPayload{GoalID: goalID, UserID: userID, FallbackActionID: fallbackActionID}
	if err := q.Enqueue(ctx, jobKindGenerateNextAction, payload, jobs.WithUniqueKey(fallbackActionID)); err != nil {
		return fmt.Errorf("enfileirar %s: %w", jobKindGenerateNextAction, err)
	}
	return nil
}

// HandleGenerateNextAction é o job de A2 (04-agentes.md §4.2, tier fast):
// tenta trocar o conteúdo da ação de fallback já criada por uma gerada de
// verdade. Sem Gateway configurado, ou em qualquer erro do agente, o
// fallback simplesmente continua sendo a ação — não há retry: o mesmo erro
// tende a se repetir, e o fallback já resolve a experiência (04-agentes.md
// §5), então gastar orçamento tentando de novo não compensa.
func (s *Service) HandleGenerateNextAction(ctx context.Context, job jobs.Job) error {
	var payload generateNextActionPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("decodificar payload de %s: %w", jobKindGenerateNextAction, err)
	}
	if s.Gateway == nil {
		return nil
	}

	action, err := postgres.GetAction(ctx, s.Pool, payload.UserID, payload.FallbackActionID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("buscar ação de fallback: %w", err)
	}
	if action.Status != domain.ActionPending || action.GeneratedBy != domain.GeneratedByFallback {
		// já foi consumida ou já foi substituída — nada a fazer.
		return nil
	}

	g, err := postgres.GetGoal(ctx, s.Pool, payload.UserID, payload.GoalID)
	if err != nil {
		return fmt.Errorf("buscar meta: %w", err)
	}
	pack, err := s.pack(g.PackID)
	if err != nil {
		return err
	}
	track, err := postgres.GetCurrentTrack(ctx, s.Pool, g.ID)
	if err != nil && !errors.Is(err, postgres.ErrNotFound) {
		return fmt.Errorf("buscar trilha: %w", err)
	}
	comps, err := postgres.ListCompetenciesByGoal(ctx, s.Pool, g.ID)
	if err != nil {
		return err
	}
	var milestone *domain.Milestone
	if track != nil {
		milestone = domain.FirstOpen(track.Milestones)
	}
	recent, err := postgres.ListRecentActionsByGoal(ctx, s.Pool, g.ID, 5)
	if err != nil {
		return err
	}
	recentViews := make([]agents.RecentActionView, len(recent))
	for i, r := range recent {
		v := agents.RecentActionView{Title: r.Title, Status: string(r.Status)}
		if r.SkipReason != nil {
			v.SkipReason = string(*r.SkipReason)
		}
		recentViews[i] = v
	}

	difficultyHint := "same"
	if action.DifficultyHint != nil {
		difficultyHint = *action.DifficultyHint
	}
	originKind := string(domain.OriginPractice)
	if action.OriginKind != nil {
		originKind = string(*action.OriginKind)
	}

	promptCtx := agents.BuildPracticeContext(g, pack, milestone, comps, recentViews, difficultyHint, originKind)
	out, result, err := agents.GeneratePractice(ctx, s.Gateway, payload.UserID, promptCtx)
	if err != nil {
		s.Logger.Warn("agent_jobs: A2 falhou, fallback continua valendo", "goal_id", g.ID, "action_id", action.ID, "err", err)
		return nil
	}

	updated, err := domain.NewNextAction(action.ID, g.ID, payload.UserID, out.Title, out.EstimatedMin, out.MinimalVariant, domain.GeneratedByAgent, action.CreatedAt)
	if err != nil {
		s.Logger.Warn("agent_jobs: saída do A2 não passou nos invariantes do domínio, fallback continua valendo", "goal_id", g.ID, "action_id", action.ID, "err", err)
		return nil
	}
	if out.Detail != "" {
		updated.Detail = &out.Detail
	}
	if out.PracticeFormat != "" {
		updated.PracticeFormat = &out.PracticeFormat
	}
	updated.DifficultyHint = action.DifficultyHint
	updated.OriginKind = action.OriginKind
	updated.MilestoneID = action.MilestoneID
	if milestone != nil {
		updated.MilestoneID = &milestone.ID
	}
	if comp, ok := competencyByKey(comps, out.CompetencyKey); ok {
		updated.CompetencyID = &comp.ID
	} else {
		updated.CompetencyID = action.CompetencyID
	}

	replaced, err := postgres.UpdatePendingActionContent(ctx, s.Pool, updated)
	if err != nil {
		return fmt.Errorf("gravar ação gerada pelo A2: %w", err)
	}
	if !replaced {
		s.Logger.Info("agent_jobs: ação já não estava mais pending quando o A2 terminou, resultado descartado", "goal_id", g.ID, "action_id", action.ID)
	} else {
		s.Logger.Info("agent_jobs: A2 substituiu o fallback", "goal_id", g.ID, "action_id", action.ID, "model", result.Model, "cost_usd", result.CostUSD)
	}
	return nil
}

func competencyByKey(comps []domain.Competency, key string) (domain.Competency, bool) {
	for _, c := range comps {
		if c.PackKey == key {
			return c, true
		}
	}
	return domain.Competency{}, false
}

// HandlePlanTrack é o job de A1 (04-agentes.md §4.1, tier strong): produz
// uma proposal, nunca escreve trilha direto (RN-07). NoChangeNeeded true
// numa revisão é resposta válida — não cria proposta nesse caso.
func (s *Service) HandlePlanTrack(ctx context.Context, job jobs.Job) error {
	var payload planTrackPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("decodificar payload de %s: %w", jobKindPlanTrack, err)
	}
	if s.Gateway == nil {
		return nil
	}

	g, err := postgres.GetGoal(ctx, s.Pool, payload.UserID, payload.GoalID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("buscar meta: %w", err)
	}
	pack, err := s.pack(g.PackID)
	if err != nil {
		return err
	}
	comps, err := postgres.ListCompetenciesByGoal(ctx, s.Pool, g.ID)
	if err != nil {
		return err
	}

	var existingTrack *domain.Track
	track, err := postgres.GetCurrentTrack(ctx, s.Pool, g.ID)
	if err == nil {
		existingTrack = track
	} else if !errors.Is(err, postgres.ErrNotFound) {
		return fmt.Errorf("buscar trilha vigente: %w", err)
	}

	promptCtx := agents.BuildPlannerContext(g, pack, comps, existingTrack, payload.RevisionReason)
	out, result, err := agents.GeneratePlan(ctx, s.Gateway, payload.UserID, promptCtx)
	if err != nil {
		s.Logger.Warn("agent_jobs: A1 falhou, sem proposta desta vez", "goal_id", g.ID, "err", err)
		return nil
	}
	if out.NoChangeNeeded {
		s.Logger.Info("agent_jobs: A1 não propôs mudança", "goal_id", g.ID)
		return nil
	}

	milestonesPayload, err := json.Marshal(trackProposalPayload{Milestones: out.Milestones})
	if err != nil {
		return fmt.Errorf("serializar payload da proposta: %w", err)
	}

	proposalID, err := idgen.NewUUIDv7()
	if err != nil {
		return err
	}
	p := &domain.Proposal{
		ID:        proposalID,
		UserID:    payload.UserID,
		GoalID:    &g.ID,
		Kind:      domain.ProposalTrack,
		Payload:   milestonesPayload,
		Rationale: out.Rationale,
		Status:    domain.ProposalPending,
		CreatedAt: time.Now(),
	}
	if result.CallID != "" {
		p.AgentCallID = &result.CallID
	}
	if err := postgres.InsertProposal(ctx, s.Pool, p); err != nil {
		return fmt.Errorf("gravar proposal: %w", err)
	}
	s.Logger.Info("agent_jobs: A1 gerou proposta de trilha", "goal_id", g.ID, "proposal_id", p.ID, "milestones", len(out.Milestones))
	return nil
}
