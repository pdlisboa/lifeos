package pgtest

import (
	"context"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(Main(m))
}

func TestContainerBootAndMigrations(t *testing.T) {
	Reset(t)
	uid := NewUser(t)

	var email string
	if err := Pool().QueryRow(context.Background(), "SELECT email FROM app_user WHERE id = $1", uid).Scan(&email); err != nil {
		t.Fatalf("usuário criado não encontrado: %v", err)
	}

	var count int
	if err := Pool().QueryRow(context.Background(), "SELECT count(*) FROM goal").Scan(&count); err != nil {
		t.Fatalf("tabela goal não existe (migrations não rodaram?): %v", err)
	}
	if count != 0 {
		t.Fatalf("esperava tabela goal vazia após Reset, got %d", count)
	}
}
