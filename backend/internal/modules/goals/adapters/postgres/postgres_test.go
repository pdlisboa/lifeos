package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/platform/idgen"
	"github.com/phablo/lifeos/internal/testsupport/pgtest"
)

func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}

// setup reseta o banco e devolve o pool (satisfaz Querier) e um usuário dono
// das linhas do teste.
func setup(t *testing.T) (Querier, string) {
	t.Helper()
	pgtest.Reset(t)
	return pgtest.Pool(), pgtest.NewUser(t)
}

func mustID(t *testing.T) string {
	t.Helper()
	id, err := idgen.NewUUIDv7()
	if err != nil {
		t.Fatalf("gerar id: %v", err)
	}
	return id
}

// newGoal monta e insere uma meta em rascunho (status draft não exige DoD
// nem competências — CHECK goal_activation_requires_dod só vale para status
// diferente de draft).
func newGoal(t *testing.T, q Querier, userID, title string) *domain.Goal {
	t.Helper()
	g, err := domain.NewGoal(mustID(t), userID, title, domain.ArchetypeSkill, "golang", nil, time.Now())
	if err != nil {
		t.Fatalf("domain.NewGoal: %v", err)
	}
	if err := InsertGoal(context.Background(), q, g); err != nil {
		t.Fatalf("InsertGoal: %v", err)
	}
	return g
}

func newCompetency(t *testing.T, q Querier, g *domain.Goal, packKey, label string) *domain.Competency {
	t.Helper()
	c, err := domain.NewCompetency(mustID(t), g.ID, g.UserID, packKey, label, 0.5, time.Now())
	if err != nil {
		t.Fatalf("domain.NewCompetency: %v", err)
	}
	if err := InsertCompetency(context.Background(), q, c); err != nil {
		t.Fatalf("InsertCompetency: %v", err)
	}
	return c
}
