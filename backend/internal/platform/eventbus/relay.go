package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultRelayInterval  = 2 * time.Second
	defaultRelayBatchSize = 100
)

// Relay lê o outbox e publica no Bus (01-arquitetura.md §5.2). É a peça que
// dá durabilidade ao barramento in-process: se o processo cair entre gravar
// o outbox e publicar, o próximo Relay que subir pega de onde parou.
type Relay struct {
	pool      *pgxpool.Pool
	bus       *Bus
	logger    *slog.Logger
	interval  time.Duration
	batchSize int
}

type RelayOption func(*Relay)

func WithRelayInterval(d time.Duration) RelayOption { return func(r *Relay) { r.interval = d } }
func WithRelayBatchSize(n int) RelayOption          { return func(r *Relay) { r.batchSize = n } }

func NewRelay(pool *pgxpool.Pool, bus *Bus, logger *slog.Logger, opts ...RelayOption) *Relay {
	r := &Relay{
		pool:      pool,
		bus:       bus,
		logger:    logger,
		interval:  defaultRelayInterval,
		batchSize: defaultRelayBatchSize,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Run apura o outbox a cada intervalo até ctx ser cancelado.
func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.Tick(ctx); err != nil {
				r.logger.Error("eventbus: relay tick", "err", err)
			}
		}
	}
}

// Tick roda uma rodada de publicação — exportado para os testes dispararem
// sem esperar o ticker, e para um cmd/ que queira drenar o outbox sob
// demanda.
func (r *Relay) Tick(ctx context.Context) error {
	rows, err := r.pool.Query(ctx, `
		SELECT id, aggregate, aggregate_id, event_type, payload, occurred_at
		FROM outbox
		WHERE published_at IS NULL
		ORDER BY id
		LIMIT $1`, r.batchSize,
	)
	if err != nil {
		return fmt.Errorf("eventbus: consultar outbox pendente: %w", err)
	}

	var events []Event
	for rows.Next() {
		var ev Event
		if err := rows.Scan(&ev.ID, &ev.Aggregate, &ev.AggregateID, &ev.Type, &ev.Payload, &ev.OccurredAt); err != nil {
			rows.Close()
			return fmt.Errorf("eventbus: ler linha do outbox: %w", err)
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("eventbus: iterar outbox: %w", err)
	}

	var firstErr error
	for _, ev := range events {
		if err := r.bus.Publish(ctx, ev); err != nil {
			// não marca published_at: a próxima rodada tenta de novo. Os
			// assinantes que já tiveram sucesso não repetem o efeito
			// (processed_event no Bus.Publish).
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if _, err := r.pool.Exec(ctx, `UPDATE outbox SET published_at = now() WHERE id = $1`, ev.ID); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("eventbus: marcar outbox %d como publicado: %w", ev.ID, err)
			}
		}
	}
	return firstErr
}
