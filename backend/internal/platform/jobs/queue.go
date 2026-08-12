package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/phablo/lifeos/internal/platform/db"
)

const defaultMaxAttempts int16 = 5

// Queue enfileira jobs. dbtx pode ser *pgxpool.Pool ou pgx.Tx — enfileirar
// dentro da mesma transação que gerou o trabalho (ex.: junto com o INSERT de
// evidence, num handler de evento) é o caso comum.
type Queue struct {
	dbtx db.TX
}

func NewQueue(dbtx db.TX) *Queue {
	return &Queue{dbtx: dbtx}
}

type enqueueOpts struct {
	uniqueKey   *string
	runAt       time.Time
	maxAttempts int16
}

type EnqueueOption func(*enqueueOpts)

// WithUniqueKey evita duplicar trabalho: se já existe um job do mesmo Kind
// com essa chave ainda queued/running, o novo Enqueue é um no-op (usa o
// índice único parcial da migration).
func WithUniqueKey(key string) EnqueueOption {
	return func(o *enqueueOpts) { o.uniqueKey = &key }
}

func WithRunAt(t time.Time) EnqueueOption {
	return func(o *enqueueOpts) { o.runAt = t }
}

func WithMaxAttempts(n int16) EnqueueOption {
	return func(o *enqueueOpts) { o.maxAttempts = n }
}

// Enqueue insere um job pronto pra ser reivindicado pelo Worker. payload é
// serializado como jsonb.
func (q *Queue) Enqueue(ctx context.Context, kind string, payload any, opts ...EnqueueOption) error {
	o := enqueueOpts{runAt: time.Now(), maxAttempts: defaultMaxAttempts}
	for _, opt := range opts {
		opt(&o)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("jobs: serializar payload de %s: %w", kind, err)
	}

	_, err = q.dbtx.Exec(ctx, `
		INSERT INTO job (kind, payload, unique_key, run_at, max_attempts)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (kind, unique_key) WHERE unique_key IS NOT NULL AND status IN ('queued','running')
		DO NOTHING`,
		kind, body, o.uniqueKey, o.runAt, o.maxAttempts,
	)
	if err != nil {
		return fmt.Errorf("jobs: enfileirar %s: %w", kind, err)
	}
	return nil
}
