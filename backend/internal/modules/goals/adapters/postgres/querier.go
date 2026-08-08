// Package postgres implementa a persistência do módulo Metas com pgx puro —
// sem sqlc, no mesmo estilo já usado em internal/platform/auth. Não há ORM
// (02-modelo-de-dados.md §2): SQL é a parte interessante em Go.
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier é satisfeito tanto por *pgxpool.Pool quanto por pgx.Tx — os stores
// recebem um Querier em vez de guardar o pool, para que o app/ decida se a
// operação roda solta ou dentro de uma transação (RN-03, RN-02 etc. exigem
// várias tabelas consistentes na mesma transação).
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
