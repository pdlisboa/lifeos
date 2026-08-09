package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

func newEvidence(t *testing.T, q Querier, g *domain.Goal, body string) *domain.Evidence {
	t.Helper()
	e, err := domain.NewEvidence(mustID(t), g.ID, g.UserID, nil, domain.EvidenceCodeSnippet, nil, &body, nil, nil, nil, nil, time.UTC, time.Now())
	if err != nil {
		t.Fatalf("domain.NewEvidence: %v", err)
	}
	if err := InsertEvidence(context.Background(), q, e); err != nil {
		t.Fatalf("InsertEvidence: %v", err)
	}
	return e
}

func TestInsertAndGetEvidenceRoundTrip(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	e := newEvidence(t, q, g, "func main() {}")

	fetched, err := GetEvidence(context.Background(), q, userID, e.ID)
	if err != nil {
		t.Fatalf("GetEvidence: %v", err)
	}
	if fetched.Body == nil || *fetched.Body != "func main() {}" {
		t.Fatalf("round-trip não bate: %+v", fetched)
	}
}

func TestGetEvidenceNotFound(t *testing.T) {
	q, userID := setup(t)
	if _, err := GetEvidence(context.Background(), q, userID, mustID(t)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("erro = %v, want ErrNotFound", err)
	}
}

func TestListEvidenceByGoalOrderingAndLimit(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	first := newEvidence(t, q, g, "primeira")
	time.Sleep(2 * time.Millisecond)
	second := newEvidence(t, q, g, "segunda")
	time.Sleep(2 * time.Millisecond)
	third := newEvidence(t, q, g, "terceira")

	asc, err := ListEvidenceByGoal(context.Background(), q, g.ID, true, 10)
	if err != nil {
		t.Fatalf("ListEvidenceByGoal asc: %v", err)
	}
	if len(asc) != 3 || asc[0].ID != first.ID || asc[2].ID != third.ID {
		t.Fatalf("ordem ascendente inesperada: %+v", asc)
	}

	desc, err := ListEvidenceByGoal(context.Background(), q, g.ID, false, 10)
	if err != nil {
		t.Fatalf("ListEvidenceByGoal desc: %v", err)
	}
	if len(desc) != 3 || desc[0].ID != third.ID {
		t.Fatalf("ordem descendente inesperada: %+v", desc)
	}

	limited, err := ListEvidenceByGoal(context.Background(), q, g.ID, true, 2)
	if err != nil {
		t.Fatalf("ListEvidenceByGoal com limit: %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("esperava 2 itens com limit=2, got %d", len(limited))
	}
	_ = second

	n, err := CountEvidenceByGoal(context.Background(), q, g.ID)
	if err != nil {
		t.Fatalf("CountEvidenceByGoal: %v", err)
	}
	if n != 3 {
		t.Fatalf("count = %d, want 3", n)
	}
}

// TestEvidenceIsImmutableRN06 confirma a regra no banco (RULE ... DO INSTEAD
// NOTHING, 02-modelo-de-dados.md): UPDATE e DELETE em evidence são no-ops
// silenciosos — a linha continua exatamente como foi gravada.
func TestEvidenceIsImmutableRN06(t *testing.T) {
	q, userID := setup(t)
	g := newGoal(t, q, userID, "Meta")
	e := newEvidence(t, q, g, "conteúdo original")

	if _, err := q.Exec(context.Background(), `UPDATE evidence SET body = 'alterado' WHERE id = $1`, e.ID); err != nil {
		t.Fatalf("UPDATE em evidence não deveria retornar erro (é um no-op), got %v", err)
	}
	if _, err := q.Exec(context.Background(), `DELETE FROM evidence WHERE id = $1`, e.ID); err != nil {
		t.Fatalf("DELETE em evidence não deveria retornar erro (é um no-op), got %v", err)
	}

	fetched, err := GetEvidence(context.Background(), q, userID, e.ID)
	if err != nil {
		t.Fatalf("evidência deveria continuar existindo após tentativa de UPDATE/DELETE: %v", err)
	}
	if fetched.Body == nil || *fetched.Body != "conteúdo original" {
		t.Fatalf("body deveria continuar 'conteúdo original', got %v", fetched.Body)
	}
}
