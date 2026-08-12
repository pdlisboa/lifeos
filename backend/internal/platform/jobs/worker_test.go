package jobs

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/testsupport/pgtest"
)

func TestWorker_ProcessesJobAndMarksDone(t *testing.T) {
	pgtest.Reset(t)
	q := NewQueue(pgtest.Pool())
	ctx := context.Background()
	if err := q.Enqueue(ctx, "ping", map[string]string{}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	var handled int32
	w := NewWorker(pgtest.Pool(), testLogger(), WithPollInterval(20*time.Millisecond))
	w.Register("ping", func(ctx context.Context, j Job) error {
		atomic.AddInt32(&handled, 1)
		return nil
	})

	runCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	go w.Run(runCtx)

	waitForStatus(t, "ping", StatusDone, 2*time.Second)
	if atomic.LoadInt32(&handled) != 1 {
		t.Fatalf("handler deveria ter rodado 1 vez, rodou %d", handled)
	}
}

func TestWorker_RetriesThenGoesDeadAfterMaxAttempts(t *testing.T) {
	pgtest.Reset(t)
	q := NewQueue(pgtest.Pool())
	ctx := context.Background()
	if err := q.Enqueue(ctx, "flaky", map[string]string{}, WithMaxAttempts(2)); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	var attempts int32
	w := NewWorker(pgtest.Pool(), testLogger(), WithPollInterval(20*time.Millisecond))
	w.Register("flaky", func(ctx context.Context, j Job) error {
		atomic.AddInt32(&attempts, 1)
		return fmt.Errorf("falha proposital")
	})

	runCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	go w.Run(runCtx)

	waitForStatus(t, "flaky", StatusDead, 3*time.Second)
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("esperava 2 tentativas (max_attempts=2), teve %d", got)
	}
}

func TestWorker_GracefulShutdownWaitsInFlightJob(t *testing.T) {
	pgtest.Reset(t)
	q := NewQueue(pgtest.Pool())
	ctx := context.Background()
	if err := q.Enqueue(ctx, "slow", map[string]string{}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	w := NewWorker(pgtest.Pool(), testLogger(), WithPollInterval(20*time.Millisecond))
	w.Register("slow", func(ctx context.Context, j Job) error {
		close(started)
		<-release
		return nil
	})

	runCtx, cancel := context.WithCancel(ctx)
	runDone := make(chan struct{})
	go func() {
		w.Run(runCtx)
		close(runDone)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("job nunca começou a rodar")
	}

	// Cancela o worker enquanto o job ainda está em voo — Run não deve
	// retornar até o handler terminar.
	cancel()
	select {
	case <-runDone:
		t.Fatal("Run retornou antes do job em voo terminar")
	case <-time.After(200 * time.Millisecond):
	}

	close(release)
	select {
	case <-runDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Run não retornou depois que o job em voo terminou")
	}

	waitForStatus(t, "slow", StatusDone, 2*time.Second)
}

// TestWorker_ConcurrencyNoJobProcessedTwice é o teste pedido explicitamente:
// N workers concorrentes competindo pela mesma fila, M jobs, nenhum
// processado duas vezes. O handler registra o job.ID numa tabela auxiliar
// com PK — um double-processing viraria erro de constraint, que o teste
// detecta.
func TestWorker_ConcurrencyNoJobProcessedTwice(t *testing.T) {
	pgtest.Reset(t)
	ctx := context.Background()

	if _, err := pgtest.Pool().Exec(ctx, `CREATE TABLE IF NOT EXISTS test_job_seen (job_id bigint PRIMARY KEY)`); err != nil {
		t.Fatalf("criar tabela auxiliar: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pgtest.Pool().Exec(context.Background(), `DROP TABLE IF EXISTS test_job_seen`)
	})

	const numWorkers = 4
	const numJobs = 50

	q := NewQueue(pgtest.Pool())
	for i := 0; i < numJobs; i++ {
		if err := q.Enqueue(ctx, "count_me", map[string]int{"n": i}); err != nil {
			t.Fatalf("Enqueue job %d: %v", i, err)
		}
	}

	var processed int32
	var dupErrMu sync.Mutex
	var dupErr error

	handler := func(ctx context.Context, j Job) error {
		// Simula algum trabalho real, o suficiente pra dar chance de
		// corrida entre workers se o SKIP LOCKED não estivesse certo.
		time.Sleep(2 * time.Millisecond)
		_, err := pgtest.Pool().Exec(ctx, `INSERT INTO test_job_seen (job_id) VALUES ($1)`, j.ID)
		if err != nil {
			dupErrMu.Lock()
			if dupErr == nil {
				dupErr = fmt.Errorf("job %d processado mais de uma vez: %w", j.ID, err)
			}
			dupErrMu.Unlock()
			return err
		}
		atomic.AddInt32(&processed, 1)
		return nil
	}

	runCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		w := NewWorker(pgtest.Pool(), testLogger(),
			WithWorkerID(fmt.Sprintf("test-worker-%d", i)),
			WithPollInterval(10*time.Millisecond),
			WithConcurrency(3),
		)
		w.Register("count_me", handler)
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.Run(runCtx)
		}()
	}

	waitForAllDone(t, "count_me", numJobs, 8*time.Second)
	cancel()
	wg.Wait()

	dupErrMu.Lock()
	if dupErr != nil {
		t.Fatal(dupErr)
	}
	dupErrMu.Unlock()

	if got := atomic.LoadInt32(&processed); got != numJobs {
		t.Fatalf("esperava %d jobs processados, teve %d", numJobs, got)
	}

	var seenCount int
	if err := pgtest.Pool().QueryRow(context.Background(), `SELECT count(*) FROM test_job_seen`).Scan(&seenCount); err != nil {
		t.Fatalf("contar test_job_seen: %v", err)
	}
	if seenCount != numJobs {
		t.Fatalf("test_job_seen tem %d linhas, esperava %d (nenhum job duplicado)", seenCount, numJobs)
	}
}

func waitForStatus(t *testing.T, kind string, want Status, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var status string
		err := pgtest.Pool().QueryRow(context.Background(),
			`SELECT status FROM job WHERE kind = $1 ORDER BY id DESC LIMIT 1`, kind,
		).Scan(&status)
		if err == nil && status == string(want) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("job kind=%s não chegou em status=%s a tempo", kind, want)
}

func waitForAllDone(t *testing.T, kind string, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var count int
		err := pgtest.Pool().QueryRow(context.Background(),
			`SELECT count(*) FROM job WHERE kind = $1 AND status = 'done'`, kind,
		).Scan(&count)
		if err == nil && count == want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("nem todos os %d jobs kind=%s chegaram a done a tempo", want, kind)
}
