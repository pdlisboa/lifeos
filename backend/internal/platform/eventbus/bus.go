package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler reage a um Event. Um erro devolvido significa "tente de novo
// depois" — o Bus não marca processed_event nesse caso.
type Handler func(ctx context.Context, ev Event) error

type subscription struct {
	consumer string
	handler  Handler
}

// Bus é o despachante em memória (01-arquitetura.md §5.2). A durabilidade
// não vem dele — vem do outbox relido pelo Relay — mas a idempotência por
// assinante vem daqui, via processed_event.
type Bus struct {
	pool   *pgxpool.Pool
	logger *slog.Logger

	mu   sync.RWMutex
	subs map[string][]subscription // event type -> assinantes
}

func NewBus(pool *pgxpool.Pool, logger *slog.Logger) *Bus {
	return &Bus{pool: pool, logger: logger, subs: make(map[string][]subscription)}
}

// Subscribe registra um handler para um tipo de evento. consumer identifica
// o assinante de forma estável (ex.: "assess_evidence_enqueuer") — é a
// chave de processed_event, então trocar esse nome faz o consumidor
// reprocessar o histórico inteiro na próxima vez que o evento aparecer.
func (b *Bus) Subscribe(eventType, consumer string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[eventType] = append(b.subs[eventType], subscription{consumer: consumer, handler: h})
}

// Publish despacha ev para cada assinante do seu tipo. Cada assinante é
// protegido individualmente: se já processou este event.ID, pula; se o
// handler falha, não marca como processado (o Relay vai tentar de novo na
// próxima rodada, sem duplicar o que os outros assinantes já fizeram com
// sucesso). Devolve o primeiro erro encontrado, mas sempre tenta todos os
// assinantes antes de retornar.
func (b *Bus) Publish(ctx context.Context, ev Event) error {
	b.mu.RLock()
	subs := make([]subscription, len(b.subs[ev.Type]))
	copy(subs, b.subs[ev.Type])
	b.mu.RUnlock()

	var firstErr error
	for _, sub := range subs {
		already, err := b.alreadyProcessed(ctx, sub.consumer, ev.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if already {
			continue
		}
		if err := sub.handler(ctx, ev); err != nil {
			b.logger.Error("eventbus: handler falhou",
				"consumer", sub.consumer, "event_type", ev.Type, "event_id", ev.ID, "err", err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if err := b.markProcessed(ctx, sub.consumer, ev.ID); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (b *Bus) alreadyProcessed(ctx context.Context, consumer string, eventID int64) (bool, error) {
	var exists bool
	err := b.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM processed_event WHERE consumer = $1 AND event_id = $2)`,
		consumer, eventID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("eventbus: checar processed_event: %w", err)
	}
	return exists, nil
}

func (b *Bus) markProcessed(ctx context.Context, consumer string, eventID int64) error {
	_, err := b.pool.Exec(ctx,
		`INSERT INTO processed_event (consumer, event_id) VALUES ($1, $2)
		 ON CONFLICT (consumer, event_id) DO NOTHING`,
		consumer, eventID,
	)
	if err != nil {
		return fmt.Errorf("eventbus: marcar processed_event: %w", err)
	}
	return nil
}
