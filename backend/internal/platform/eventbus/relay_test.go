package eventbus

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/platform/idgen"
	"github.com/phablo/lifeos/internal/testsupport/pgtest"
)

func TestRelay_TickPublishesUnpublishedRowsAndMarksThem(t *testing.T) {
	pgtest.Reset(t)
	bus := NewBus(pgtest.Pool(), testLogger())

	var got []string
	var mu sync.Mutex
	bus.Subscribe("goal.created", "recorder", func(ctx context.Context, ev Event) error {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, ev.AggregateID)
		return nil
	})

	relay := NewRelay(pgtest.Pool(), bus, testLogger())

	ctx := context.Background()
	mustAppend(t, "goal", "goal.created")
	mustAppend(t, "goal", "goal.created")

	if err := relay.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}

	mu.Lock()
	n := len(got)
	mu.Unlock()
	if n != 2 {
		t.Fatalf("esperava 2 eventos publicados, teve %d", n)
	}

	var unpublished int
	err := pgtest.Pool().QueryRow(ctx, `SELECT count(*) FROM outbox WHERE published_at IS NULL`).Scan(&unpublished)
	if err != nil {
		t.Fatalf("contar outbox pendente: %v", err)
	}
	if unpublished != 0 {
		t.Fatalf("esperava outbox vazio de pendentes, sobrou %d", unpublished)
	}
}

func TestRelay_LeavesRowUnpublishedWhenHandlerFails(t *testing.T) {
	pgtest.Reset(t)
	bus := NewBus(pgtest.Pool(), testLogger())

	var attempts int32
	bus.Subscribe("goal.created", "instavel", func(ctx context.Context, ev Event) error {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			return errors.New("falha simulada")
		}
		return nil
	})

	relay := NewRelay(pgtest.Pool(), bus, testLogger())
	ctx := context.Background()
	mustAppend(t, "goal", "goal.created")

	if err := relay.Tick(ctx); err == nil {
		t.Fatal("esperava erro na 1ª rodada")
	}
	var unpublished int
	_ = pgtest.Pool().QueryRow(ctx, `SELECT count(*) FROM outbox WHERE published_at IS NULL`).Scan(&unpublished)
	if unpublished != 1 {
		t.Fatalf("linha deveria continuar pendente após falha, unpublished=%d", unpublished)
	}

	if err := relay.Tick(ctx); err != nil {
		t.Fatalf("2ª rodada deveria ter sucesso: %v", err)
	}
	_ = pgtest.Pool().QueryRow(ctx, `SELECT count(*) FROM outbox WHERE published_at IS NULL`).Scan(&unpublished)
	if unpublished != 0 {
		t.Fatalf("linha deveria estar publicada após retry bem-sucedido, unpublished=%d", unpublished)
	}
	if attempts != 2 {
		t.Fatalf("handler deveria ter sido chamado 2 vezes, foi %d", attempts)
	}
}

func TestRelay_RunPublishesOnTicksUntilContextCancelled(t *testing.T) {
	pgtest.Reset(t)
	bus := NewBus(pgtest.Pool(), testLogger())

	done := make(chan struct{}, 1)
	bus.Subscribe("goal.created", "recorder", func(ctx context.Context, ev Event) error {
		select {
		case done <- struct{}{}:
		default:
		}
		return nil
	})

	relay := NewRelay(pgtest.Pool(), bus, testLogger(), WithRelayInterval(20*time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())

	go relay.Run(ctx)
	mustAppend(t, "goal", "goal.created")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("relay não publicou o evento a tempo")
	}
	cancel()
}

func mustAppend(t *testing.T, aggregate, eventType string) {
	t.Helper()
	aggregateID, err := idgen.NewUUIDv7()
	if err != nil {
		t.Fatalf("gerar aggregate id: %v", err)
	}
	if err := Append(context.Background(), pgtest.Pool(), aggregate, aggregateID, eventType, rawJSON(`{}`)); err != nil {
		t.Fatalf("Append: %v", err)
	}
}
