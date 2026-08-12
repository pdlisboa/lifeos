package eventbus

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/phablo/lifeos/internal/platform/idgen"
	"github.com/phablo/lifeos/internal/testsupport/pgtest"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestBus_PublishDispatchesToSubscriber(t *testing.T) {
	pgtest.Reset(t)
	bus := NewBus(pgtest.Pool(), testLogger())

	var got Event
	var calls int
	bus.Subscribe("goal.created", "c1", func(ctx context.Context, ev Event) error {
		calls++
		got = ev
		return nil
	})

	ev := insertOutboxEvent(t, "goal", "goal.created", `{"title":"aprender go"}`)
	if err := bus.Publish(context.Background(), ev); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if calls != 1 {
		t.Fatalf("esperava 1 chamada, teve %d", calls)
	}
	if got.AggregateID != ev.AggregateID {
		t.Fatalf("aggregate_id inesperado: %s", got.AggregateID)
	}
}

func TestBus_IgnoresUnsubscribedTypes(t *testing.T) {
	pgtest.Reset(t)
	bus := NewBus(pgtest.Pool(), testLogger())
	bus.Subscribe("goal.created", "c1", func(ctx context.Context, ev Event) error {
		t.Fatal("não deveria ser chamado para outro tipo de evento")
		return nil
	})

	ev := insertOutboxEvent(t, "goal", "goal.activated", `{}`)
	if err := bus.Publish(context.Background(), ev); err != nil {
		t.Fatalf("Publish: %v", err)
	}
}

func TestBus_HandlerNotCalledTwiceOnceProcessed(t *testing.T) {
	pgtest.Reset(t)
	bus := NewBus(pgtest.Pool(), testLogger())

	var calls int32
	bus.Subscribe("evidence.created", "assessor", func(ctx context.Context, ev Event) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})

	ev := insertOutboxEvent(t, "evidence", "evidence.created", `{}`)
	if err := bus.Publish(context.Background(), ev); err != nil {
		t.Fatalf("1ª Publish: %v", err)
	}
	// Simula o relay relendo a mesma linha (ex.: processo caiu entre
	// publicar e marcar published_at).
	if err := bus.Publish(context.Background(), ev); err != nil {
		t.Fatalf("2ª Publish: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("esperava 1 chamada no total (idempotência), teve %d", got)
	}
}

func TestBus_MultipleConsumersHaveIndependentIdempotency(t *testing.T) {
	pgtest.Reset(t)
	bus := NewBus(pgtest.Pool(), testLogger())

	var callsA, callsB int32
	bus.Subscribe("evidence.created", "consumer-a", func(ctx context.Context, ev Event) error {
		atomic.AddInt32(&callsA, 1)
		return nil
	})
	bus.Subscribe("evidence.created", "consumer-b", func(ctx context.Context, ev Event) error {
		atomic.AddInt32(&callsB, 1)
		return nil
	})

	ev := insertOutboxEvent(t, "evidence", "evidence.created", `{}`)
	_ = bus.Publish(context.Background(), ev)
	_ = bus.Publish(context.Background(), ev)

	if callsA != 1 || callsB != 1 {
		t.Fatalf("esperava 1 chamada por consumidor, teve a=%d b=%d", callsA, callsB)
	}
}

func TestBus_FailedHandlerIsRetriedWithoutDuplicatingSucceededOnes(t *testing.T) {
	pgtest.Reset(t)
	bus := NewBus(pgtest.Pool(), testLogger())

	var okCalls int32
	bus.Subscribe("evidence.created", "sempre-ok", func(ctx context.Context, ev Event) error {
		atomic.AddInt32(&okCalls, 1)
		return nil
	})

	var flakyCalls int32
	var mu sync.Mutex
	shouldFail := true
	bus.Subscribe("evidence.created", "instavel", func(ctx context.Context, ev Event) error {
		atomic.AddInt32(&flakyCalls, 1)
		mu.Lock()
		defer mu.Unlock()
		if shouldFail {
			shouldFail = false
			return context.DeadlineExceeded
		}
		return nil
	})

	ev := insertOutboxEvent(t, "evidence", "evidence.created", `{}`)

	if err := bus.Publish(context.Background(), ev); err == nil {
		t.Fatal("esperava erro na 1ª rodada (handler instável falhou)")
	}
	if err := bus.Publish(context.Background(), ev); err != nil {
		t.Fatalf("2ª rodada deveria ter sucesso: %v", err)
	}

	if okCalls != 1 {
		t.Fatalf("consumidor que já teve sucesso não deveria rodar de novo, rodou %d vezes", okCalls)
	}
	if flakyCalls != 2 {
		t.Fatalf("consumidor instável deveria ter sido tentado 2 vezes, foi %d", flakyCalls)
	}
}

// insertOutboxEvent grava uma linha real no outbox (via Append) e devolve o
// Event correspondente, para os testes de Bus não precisarem do Relay.
// aggregate_id é uuid no banco, então gera um UUIDv7 de verdade em vez de um
// literal tipo "g1".
func insertOutboxEvent(t *testing.T, aggregate, eventType, payloadJSON string) Event {
	t.Helper()
	ctx := context.Background()
	aggregateID, err := idgen.NewUUIDv7()
	if err != nil {
		t.Fatalf("gerar aggregate id: %v", err)
	}
	if err := Append(ctx, pgtest.Pool(), aggregate, aggregateID, eventType, rawJSON(payloadJSON)); err != nil {
		t.Fatalf("Append: %v", err)
	}
	var ev Event
	err = pgtest.Pool().QueryRow(ctx, `
		SELECT id, aggregate, aggregate_id, event_type, payload, occurred_at
		FROM outbox WHERE aggregate_id = $1 AND event_type = $2
		ORDER BY id DESC LIMIT 1`, aggregateID, eventType,
	).Scan(&ev.ID, &ev.Aggregate, &ev.AggregateID, &ev.Type, &ev.Payload, &ev.OccurredAt)
	if err != nil {
		t.Fatalf("ler outbox recém-inserido: %v", err)
	}
	return ev
}

type rawJSON string

func (r rawJSON) MarshalJSON() ([]byte, error) { return []byte(r), nil }
