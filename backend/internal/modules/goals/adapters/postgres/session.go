package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres/sqlcgen"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func InsertSession(ctx context.Context, q Querier, s *domain.Session) error {
	err := sqlcgen.New(q).InsertSession(ctx, sqlcgen.InsertSessionParams{
		ID:          s.ID,
		GoalID:      s.GoalID,
		UserID:      s.UserID,
		ActionID:    s.ActionID,
		StartedAt:   s.StartedAt,
		DurationMin: int16(s.DurationMin),
		Note:        s.Note,
		Energy:      i16ptr(s.Energy),
		LocalOn:     s.LocalOn,
		CreatedAt:   s.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("inserir session: %w", err)
	}
	return nil
}

func ListSessionsByGoal(ctx context.Context, q Querier, goalID string, limit int) ([]domain.Session, error) {
	rows, err := sqlcgen.New(q).ListSessionsByGoal(ctx, sqlcgen.ListSessionsByGoalParams{GoalID: goalID, Limit: int32(limit)})
	if err != nil {
		return nil, fmt.Errorf("listar sessions: %w", err)
	}
	out := make([]domain.Session, len(rows))
	for i, r := range rows {
		out[i] = domain.Session{
			ID:          r.ID,
			GoalID:      r.GoalID,
			UserID:      r.UserID,
			ActionID:    r.ActionID,
			StartedAt:   r.StartedAt,
			DurationMin: int(r.DurationMin),
			Note:        r.Note,
			Energy:      intptr(r.Energy),
			LocalOn:     r.LocalOn,
			CreatedAt:   r.CreatedAt,
		}
	}
	return out, nil
}

// SumSessionMinutes é usado no total de minutos do Painel de Delta (§7.1).
func SumSessionMinutes(ctx context.Context, q Querier, goalID string) (int, error) {
	total, err := sqlcgen.New(q).SumSessionMinutes(ctx, goalID)
	if err != nil {
		return 0, fmt.Errorf("somar minutos de sessão: %w", err)
	}
	return int(total), nil
}

// SessionPace é o ritmo real dos últimos 30 dias (02-modelo-de-dados.md
// §14.4). SpanDays só tem sentido quando ActiveDays > 0 — sem sessão
// nenhuma na janela, span_days sai zerado pelo coalesce do SQL, e não pode
// ser confundido com "sessão hoje" (ver comentário na query).
type SessionPace struct {
	TotalMinutes int
	ActiveDays   int
	SpanDays     int
}

func SessionPaceLast30d(ctx context.Context, q Querier, goalID string) (SessionPace, error) {
	r, err := sqlcgen.New(q).SessionPaceLast30d(ctx, goalID)
	if err != nil {
		return SessionPace{}, fmt.Errorf("buscar ritmo de sessão: %w", err)
	}
	return SessionPace{TotalMinutes: int(r.TotalMin), ActiveDays: int(r.DiasAtivos), SpanDays: int(r.SpanDays)}, nil
}

// ActiveLocalDates junta local_on de sessões e evidências de UMA meta nos
// últimos `days` dias — fonte de ComputeConsistency para GET
// /goals/{id}/consistency (RN-11). 02-modelo-de-dados.md §14.3 mostra a
// versão global (todas as metas do usuário); aqui filtramos por goal_id
// porque é isso que o endpoint expõe.
func ActiveLocalDates(ctx context.Context, q Querier, goalID string, days int) ([]time.Time, error) {
	rows, err := sqlcgen.New(q).ActiveLocalDates(ctx, sqlcgen.ActiveLocalDatesParams{GoalID: goalID, Column2: int32(days)})
	if err != nil {
		return nil, fmt.Errorf("buscar dias ativos: %w", err)
	}
	return rows, nil
}
