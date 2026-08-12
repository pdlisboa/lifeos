package jobs

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/phablo/lifeos/internal/testsupport/pgtest"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestEnqueue_CreatesQueuedJob(t *testing.T) {
	pgtest.Reset(t)
	q := NewQueue(pgtest.Pool())

	if err := q.Enqueue(context.Background(), "generate_next_action", map[string]string{"goalId": "g1"}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	var status string
	err := pgtest.Pool().QueryRow(context.Background(),
		`SELECT status FROM job WHERE kind = 'generate_next_action'`,
	).Scan(&status)
	if err != nil {
		t.Fatalf("consultar job: %v", err)
	}
	if status != string(StatusQueued) {
		t.Fatalf("status = %s, esperava queued", status)
	}
}

func TestEnqueue_UniqueKeyDeduplicatesWhileQueued(t *testing.T) {
	pgtest.Reset(t)
	q := NewQueue(pgtest.Pool())
	ctx := context.Background()

	if err := q.Enqueue(ctx, "assess_evidence", map[string]string{}, WithUniqueKey("evidence-1")); err != nil {
		t.Fatalf("1º Enqueue: %v", err)
	}
	if err := q.Enqueue(ctx, "assess_evidence", map[string]string{}, WithUniqueKey("evidence-1")); err != nil {
		t.Fatalf("2º Enqueue: %v", err)
	}

	var count int
	if err := pgtest.Pool().QueryRow(ctx, `SELECT count(*) FROM job WHERE kind = 'assess_evidence'`).Scan(&count); err != nil {
		t.Fatalf("contar jobs: %v", err)
	}
	if count != 1 {
		t.Fatalf("esperava 1 job (dedup por unique_key), teve %d", count)
	}
}

func TestEnqueue_UniqueKeyReusableAfterDone(t *testing.T) {
	pgtest.Reset(t)
	q := NewQueue(pgtest.Pool())
	ctx := context.Background()

	if err := q.Enqueue(ctx, "assess_evidence", map[string]string{}, WithUniqueKey("evidence-1")); err != nil {
		t.Fatalf("1º Enqueue: %v", err)
	}
	if _, err := pgtest.Pool().Exec(ctx, `UPDATE job SET status = 'done' WHERE kind = 'assess_evidence'`); err != nil {
		t.Fatalf("marcar done: %v", err)
	}
	if err := q.Enqueue(ctx, "assess_evidence", map[string]string{}, WithUniqueKey("evidence-1")); err != nil {
		t.Fatalf("2º Enqueue (após done): %v", err)
	}

	var count int
	if err := pgtest.Pool().QueryRow(ctx, `SELECT count(*) FROM job WHERE kind = 'assess_evidence'`).Scan(&count); err != nil {
		t.Fatalf("contar jobs: %v", err)
	}
	if count != 2 {
		t.Fatalf("esperava 2 jobs (unique_key liberado após done), teve %d", count)
	}
}

func TestEnqueue_WithoutUniqueKeyAllowsDuplicates(t *testing.T) {
	pgtest.Reset(t)
	q := NewQueue(pgtest.Pool())
	ctx := context.Background()

	if err := q.Enqueue(ctx, "detect_stagnation", map[string]string{}); err != nil {
		t.Fatalf("1º Enqueue: %v", err)
	}
	if err := q.Enqueue(ctx, "detect_stagnation", map[string]string{}); err != nil {
		t.Fatalf("2º Enqueue: %v", err)
	}

	var count int
	if err := pgtest.Pool().QueryRow(ctx, `SELECT count(*) FROM job WHERE kind = 'detect_stagnation'`).Scan(&count); err != nil {
		t.Fatalf("contar jobs: %v", err)
	}
	if count != 2 {
		t.Fatalf("sem unique_key, esperava 2 jobs, teve %d", count)
	}
}
