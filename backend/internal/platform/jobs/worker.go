package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultPollInterval = 500 * time.Millisecond
	defaultBatchSize    = 10
	defaultConcurrency  = 4
)

// Worker reivindica jobs com SELECT ... FOR UPDATE SKIP LOCKED e os processa
// concorrentemente até `concurrency` de cada vez (01-arquitetura.md §7).
type Worker struct {
	pool     *pgxpool.Pool
	logger   *slog.Logger
	handlers map[string]Handler

	id           string
	pollInterval time.Duration
	batchSize    int
	concurrency  int
}

type WorkerOption func(*Worker)

func WithWorkerID(id string) WorkerOption           { return func(w *Worker) { w.id = id } }
func WithPollInterval(d time.Duration) WorkerOption { return func(w *Worker) { w.pollInterval = d } }
func WithBatchSize(n int) WorkerOption              { return func(w *Worker) { w.batchSize = n } }
func WithConcurrency(n int) WorkerOption            { return func(w *Worker) { w.concurrency = n } }

func NewWorker(pool *pgxpool.Pool, logger *slog.Logger, opts ...WorkerOption) *Worker {
	w := &Worker{
		pool:         pool,
		logger:       logger,
		handlers:     make(map[string]Handler),
		id:           fmt.Sprintf("worker-%d", time.Now().UnixNano()),
		pollInterval: defaultPollInterval,
		batchSize:    defaultBatchSize,
		concurrency:  defaultConcurrency,
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Register associa um Handler a um Kind de job. O worker só compete pelos
// kinds registrados — sem handler nenhum, ele fica ocioso sem tocar a fila.
func (w *Worker) Register(kind string, h Handler) {
	w.handlers[kind] = h
}

// Run reivindica e processa jobs até ctx ser cancelado. No shutdown, espera
// todo job em voo terminar antes de retornar (cancelamento por context é
// responsabilidade do Handler, que recebe o mesmo ctx).
func (w *Worker) Run(ctx context.Context) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, w.concurrency)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case <-ticker.C:
		}

		claimed, err := w.fetch(ctx)
		if err != nil {
			w.logger.Error("jobs: fetch falhou", "err", err)
			continue
		}
		for _, j := range claimed {
			j := j
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				w.process(ctx, j)
			}()
		}
	}
}

func (w *Worker) fetch(ctx context.Context) ([]Job, error) {
	if len(w.handlers) == 0 {
		return nil, nil
	}
	kinds := make([]string, 0, len(w.handlers))
	for k := range w.handlers {
		kinds = append(kinds, k)
	}

	rows, err := w.pool.Query(ctx, `
		WITH next AS (
			SELECT id FROM job
			WHERE status = 'queued' AND run_at <= now() AND kind = ANY($1)
			ORDER BY run_at
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		UPDATE job SET status = 'running', locked_at = now(), locked_by = $3, attempts = attempts + 1
		FROM next WHERE job.id = next.id
		RETURNING job.id, job.kind, job.payload, job.attempts, job.max_attempts`,
		kinds, w.batchSize, w.id,
	)
	if err != nil {
		return nil, fmt.Errorf("jobs: reivindicar jobs: %w", err)
	}
	defer rows.Close()

	var out []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.Kind, &j.Payload, &j.Attempts, &j.MaxAttempts); err != nil {
			return nil, fmt.Errorf("jobs: ler job reivindicado: %w", err)
		}
		out = append(out, j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("jobs: iterar jobs reivindicados: %w", err)
	}
	return out, nil
}

func (w *Worker) process(ctx context.Context, j Job) {
	h, ok := w.handlers[j.Kind]
	if !ok {
		w.fail(j, fmt.Errorf("jobs: nenhum handler registrado para kind %q", j.Kind))
		return
	}

	if err := h(ctx, j); err != nil {
		w.fail(j, err)
		return
	}

	// context.Background(): o job já rodou com sucesso — gravar o resultado
	// não deve ser abortado só porque o worker está desligando.
	if _, err := w.pool.Exec(context.Background(),
		`UPDATE job SET status = 'done', locked_at = NULL, locked_by = NULL WHERE id = $1`, j.ID,
	); err != nil {
		w.logger.Error("jobs: marcar job como done", "job_id", j.ID, "err", err)
	}
}

func (w *Worker) fail(j Job, cause error) {
	next := StatusQueued
	if j.Attempts >= j.MaxAttempts {
		next = StatusDead
	}
	runAt := nextRunAt(time.Now(), j.Attempts)

	if _, err := w.pool.Exec(context.Background(), `
		UPDATE job SET status = $1, run_at = $2, last_error = $3, locked_at = NULL, locked_by = NULL
		WHERE id = $4`,
		next, runAt, cause.Error(), j.ID,
	); err != nil {
		w.logger.Error("jobs: marcar job como falho", "job_id", j.ID, "err", err)
		return
	}
	w.logger.Warn("jobs: job falhou", "job_id", j.ID, "kind", j.Kind, "attempts", j.Attempts, "next_status", next, "cause", cause)
}
