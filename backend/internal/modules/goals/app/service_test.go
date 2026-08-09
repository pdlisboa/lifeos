package app

import (
	"os"
	"testing"

	"github.com/phablo/lifeos/internal/modules/goals/packs"
	"github.com/phablo/lifeos/internal/testsupport/pgtest"
)

func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}

// newFixture reseta o banco e devolve um Service pronto para uso, com o
// registro de packs real (golang + english embutidos) e um usuário dono das
// linhas criadas no teste.
func newFixture(t *testing.T) (*Service, string) {
	t.Helper()
	pgtest.Reset(t)
	reg, err := packs.Load()
	if err != nil {
		t.Fatalf("carregar packs: %v", err)
	}
	svc := NewService(pgtest.Pool(), reg)
	userID := pgtest.NewUser(t)
	return svc, userID
}
