package app

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/platform/idgen"
)

type CreateEvidenceInput struct {
	UserID        string
	GoalID        string
	ActionID      *string
	Kind          domain.EvidenceKind
	Title         *string
	Body          *string
	BlobKey       *string
	ExternalURL   *string
	Language      *string
	SupersedesID  *string
	CompetencyIDs []string
	Timezone      *time.Location
}

// CreateEvidence é P1: a moeda real do sistema. Imutável a partir daqui — o
// banco recusa UPDATE/DELETE (RN-06); correção é sempre uma evidência nova
// com SupersedesID apontando para a anterior.
func (s *Service) CreateEvidence(ctx context.Context, in CreateEvidenceInput) (*domain.Evidence, error) {
	var created *domain.Evidence
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		g, err := postgres.GetGoal(ctx, tx, in.UserID, in.GoalID)
		if err != nil {
			return err
		}
		if len(in.CompetencyIDs) > 0 {
			comps, err := postgres.ListCompetenciesByGoal(ctx, tx, g.ID)
			if err != nil {
				return err
			}
			valid := make(map[string]bool, len(comps))
			for _, c := range comps {
				valid[c.ID] = true
			}
			for _, cid := range in.CompetencyIDs {
				if !valid[cid] {
					return domain.NewRuleError("", "competência não pertence a esta meta")
				}
			}
		}
		loc := in.Timezone
		if loc == nil {
			loc = time.UTC
		}
		id, err := idgen.NewUUIDv7()
		if err != nil {
			return err
		}
		now := time.Now()
		ev, err := domain.NewEvidence(id, g.ID, in.UserID, in.ActionID, in.Kind, in.Title, in.Body, in.BlobKey, in.ExternalURL, in.Language, in.SupersedesID, loc, now)
		if err != nil {
			return err
		}
		if err := postgres.InsertEvidence(ctx, tx, ev); err != nil {
			return err
		}
		// "Competências tocadas" (05-ux.md §7) — sem cerimônia, é a mesma
		// evidência marcando o que ela cobre; some quem não passou nesta
		// evidência específica, nunca duplicada.
		seen := make(map[string]bool, len(in.CompetencyIDs))
		for _, cid := range in.CompetencyIDs {
			if seen[cid] {
				continue
			}
			seen[cid] = true
			if err := postgres.InsertEvidenceCompetency(ctx, tx, ev.ID, cid); err != nil {
				return err
			}
		}
		g.TouchActivity(now)
		if err := postgres.UpdateGoal(ctx, tx, g); err != nil {
			return err
		}
		created = ev
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Service) GetEvidence(ctx context.Context, userID, evidenceID string) (*domain.Evidence, error) {
	return postgres.GetEvidence(ctx, s.Pool, userID, evidenceID)
}

// EvidenceCard é o cartão do museu (§7.4): a evidência mais o que dá pra
// reconstruir sem agente — quais competências ela tocou (tag manual, §7 da
// UX) e o nível de cada competência da meta naquele instante (point-in-time
// a partir de competency_level_event, 02-modelo-de-dados.md §14.2).
type EvidenceCard struct {
	Evidence      domain.Evidence
	CompetencyIDs []string
	LevelsAtTime  map[string]int
}

// ListEvidence serve o museu, opcionalmente filtrado por competência tocada
// (§7.4, `?competencyId=`). LevelsAtTime é reconstruído de verdade — nunca
// os placeholders vazios da Fatia 1 (fatia-1-implementacao.md §... dizia que
// dependia do avaliador da Fatia 4; o nível point-in-time não depende dele,
// só do histórico de eventos que já existe desde a Fatia 1 manual).
func (s *Service) ListEvidence(ctx context.Context, userID, goalID string, competencyID *string, ascending bool, limit int) ([]EvidenceCard, error) {
	if _, err := postgres.GetGoal(ctx, s.Pool, userID, goalID); err != nil {
		return nil, err
	}

	var items []domain.Evidence
	var err error
	if competencyID != nil {
		items, err = postgres.ListEvidenceByGoalAndCompetency(ctx, s.Pool, goalID, *competencyID, ascending, limit)
	} else {
		items, err = postgres.ListEvidenceByGoal(ctx, s.Pool, goalID, ascending, limit)
	}
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(items))
	for i, e := range items {
		ids[i] = e.ID
	}
	linksByEvidence, err := postgres.ListCompetencyIDsForEvidences(ctx, s.Pool, ids)
	if err != nil {
		return nil, err
	}

	events, err := postgres.ListLevelEventsForGoal(ctx, s.Pool, goalID)
	if err != nil {
		return nil, err
	}
	eventsByCompetency := make(map[string][]domain.LevelEvent)
	for _, e := range events {
		eventsByCompetency[e.CompetencyID] = append(eventsByCompetency[e.CompetencyID], e)
	}

	cards := make([]EvidenceCard, len(items))
	for i, e := range items {
		levels := make(map[string]int)
		for compID, compEvents := range eventsByCompetency {
			if lvl := domain.LevelAtTime(compEvents, e.CreatedAt); lvl != nil {
				levels[compID] = *lvl
			}
		}
		compIDs := linksByEvidence[e.ID]
		if compIDs == nil {
			compIDs = []string{}
		}
		cards[i] = EvidenceCard{Evidence: e, CompetencyIDs: compIDs, LevelsAtTime: levels}
	}
	return cards, nil
}
