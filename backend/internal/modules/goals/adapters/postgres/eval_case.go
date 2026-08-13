package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres/sqlcgen"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

// UpsertEvalCase grava a marcação (nota + gabarito), substituindo o que
// houver antes — marcar de novo é uma correção legítima, diferente de
// evidence (RN-06), que nunca é editada.
func UpsertEvalCase(ctx context.Context, q Querier, ec *domain.EvalCase, now time.Time) error {
	qs := sqlcgen.New(q)
	if err := qs.UpsertEvalCase(ctx, sqlcgen.UpsertEvalCaseParams{
		EvidenceID: ec.EvidenceID,
		Note:       ec.Note,
		CreatedAt:  now,
	}); err != nil {
		return fmt.Errorf("gravar eval case: %w", err)
	}
	if err := qs.ReplaceEvalCaseScores(ctx, ec.EvidenceID); err != nil {
		return fmt.Errorf("limpar gabarito anterior: %w", err)
	}
	for _, s := range ec.Scores {
		if err := qs.InsertEvalCaseScore(ctx, sqlcgen.InsertEvalCaseScoreParams{
			EvidenceID:   ec.EvidenceID,
			CompetencyID: s.CompetencyID,
			Level:        int32(s.Level),
		}); err != nil {
			return fmt.Errorf("gravar nota do gabarito: %w", err)
		}
	}
	return nil
}

// GetEvalCase devolve nil (sem erro) quando a evidência não foi marcada —
// é um estado normal, não uma exceção. Usado para compor a leitura de
// evidence (evalCase é opcional no DTO).
func GetEvalCase(ctx context.Context, q Querier, evidenceID string) (*domain.EvalCase, error) {
	qs := sqlcgen.New(q)
	row, err := qs.GetEvalCase(ctx, evidenceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("buscar eval case: %w", err)
	}
	scores, err := qs.ListEvalCaseScores(ctx, evidenceID)
	if err != nil {
		return nil, fmt.Errorf("listar gabarito: %w", err)
	}
	return &domain.EvalCase{
		EvidenceID: row.EvidenceID,
		Note:       row.Note,
		Scores:     toDomainEvalCaseScores(scores),
	}, nil
}

// ListEvalCasesForEvidences agrupa por evidência de uma vez só — mesmo
// padrão de ListCompetencyIDsForEvidences, evita N+1 no museu.
func ListEvalCasesForEvidences(ctx context.Context, q Querier, evidenceIDs []string) (map[string]domain.EvalCase, error) {
	out := make(map[string]domain.EvalCase, len(evidenceIDs))
	if len(evidenceIDs) == 0 {
		return out, nil
	}
	qs := sqlcgen.New(q)
	cases, err := qs.ListEvalCasesByEvidences(ctx, evidenceIDs)
	if err != nil {
		return nil, fmt.Errorf("listar eval cases: %w", err)
	}
	scores, err := qs.ListEvalCaseScoresForEvidences(ctx, evidenceIDs)
	if err != nil {
		return nil, fmt.Errorf("listar gabaritos: %w", err)
	}
	scoresByEvidence := make(map[string][]sqlcgen.EvidenceEvalCaseScore, len(evidenceIDs))
	for _, s := range scores {
		scoresByEvidence[s.EvidenceID] = append(scoresByEvidence[s.EvidenceID], s)
	}
	for _, c := range cases {
		out[c.EvidenceID] = domain.EvalCase{
			EvidenceID: c.EvidenceID,
			Note:       c.Note,
			Scores:     toDomainEvalCaseScores(scoresByEvidence[c.EvidenceID]),
		}
	}
	return out, nil
}

func DeleteEvalCase(ctx context.Context, q Querier, evidenceID string) error {
	if err := sqlcgen.New(q).DeleteEvalCase(ctx, evidenceID); err != nil {
		return fmt.Errorf("remover eval case: %w", err)
	}
	return nil
}

// EvalCaseExportRow é a linha crua usada só por `cmd/evalexport` (04-agentes.md
// §6.1) — não é domain porque exportar pra JSON é uma leitura de dados, não
// uma regra de negócio.
type EvalCaseExportRow struct {
	EvidenceID  string
	Note        string
	CreatedAt   time.Time
	GoalID      string
	UserID      string
	ActionID    *string
	Kind        string
	Title       *string
	Body        *string
	ExternalURL *string
	Language    *string
}

// ListEvalCasesForExport devolve todos os casos marcados, mais antigo
// primeiro — é a ordem que numera os arquivos exportados.
func ListEvalCasesForExport(ctx context.Context, q Querier) ([]EvalCaseExportRow, error) {
	rows, err := sqlcgen.New(q).ListEvalCasesForExport(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar casos de eval para export: %w", err)
	}
	out := make([]EvalCaseExportRow, len(rows))
	for i, r := range rows {
		out[i] = EvalCaseExportRow{
			EvidenceID: r.EvidenceID, Note: r.Note, CreatedAt: r.CreatedAt,
			GoalID: r.GoalID, UserID: r.UserID, ActionID: r.ActionID,
			Kind: r.Kind, Title: r.Title, Body: r.Body,
			ExternalURL: r.ExternalUrl, Language: r.Language,
		}
	}
	return out, nil
}

func toDomainEvalCaseScores(rows []sqlcgen.EvidenceEvalCaseScore) []domain.EvalCaseScore {
	out := make([]domain.EvalCaseScore, len(rows))
	for i, r := range rows {
		out[i] = domain.EvalCaseScore{CompetencyID: r.CompetencyID, Level: int(r.Level)}
	}
	return out
}
