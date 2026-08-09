// Package pgtest sobe um Postgres efêmero via testcontainers-go para os
// testes de integração do módulo Metas (app/, adapters/postgres,
// adapters/http). Decisão registrada em
// docs/modulos/metas/fatia-1-implementacao.md §7.2: testcontainers-go em vez
// de exigir um Postgres já rodando, porque Docker já é dependência do
// projeto e o container fica isolado por rodada de teste.
package pgtest

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/phablo/lifeos/internal/platform/idgen"
)

// migrationsDir resolve db/migrations a partir do caminho deste arquivo-fonte
// (não do diretório de trabalho do `go test`, que é o pacote sob teste).
func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "db", "migrations")
}

var pool *pgxpool.Pool

// Main é o TestMain compartilhado pelos pacotes que precisam de Postgres
// real: sobe um container uma vez para todo o binário de teste, roda as
// migrations via goose, executa m.Run() e derruba o container no final.
//
//	func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }
func Main(m *testing.M) int {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("lifeos_test"),
		tcpostgres.WithUsername("lifeos"),
		tcpostgres.WithPassword("lifeos"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Println("pgtest: subir postgres de teste:", err)
		return 1
	}
	defer func() { _ = container.Terminate(ctx) }()

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Println("pgtest: connection string:", err)
		return 1
	}
	if err := applyMigrations(connStr); err != nil {
		fmt.Println("pgtest: aplicar migrations:", err)
		return 1
	}

	p, err := pgxpool.New(ctx, connStr)
	if err != nil {
		fmt.Println("pgtest: abrir pool:", err)
		return 1
	}
	defer p.Close()
	pool = p

	return m.Run()
}

func applyMigrations(connStr string) error {
	sqlDB, err := sql.Open("pgx", connStr)
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(sqlDB, migrationsDir())
}

// Pool devolve o pool compartilhado. Só funciona dentro de um TestMain que já
// chamou pgtest.Main.
func Pool() *pgxpool.Pool {
	if pool == nil {
		panic("pgtest: Pool() chamado sem pgtest.Main no TestMain do pacote")
	}
	return pool
}

// Reset limpa todo o estado do módulo Metas (TRUNCATE em cascata a partir de
// app_user, que é a raiz de todas as FKs) para isolar um teste do outro sem
// recriar o container a cada função.
func Reset(t *testing.T) {
	t.Helper()
	if _, err := Pool().Exec(context.Background(), "TRUNCATE TABLE app_user CASCADE"); err != nil {
		t.Fatalf("pgtest: truncate: %v", err)
	}
}

// NewUser cria um app_user mínimo e devolve o ID — quase todo teste do
// módulo Metas precisa de um dono de linha por causa das FKs (goal, session,
// evidence etc. referenciam app_user).
func NewUser(t *testing.T) string {
	t.Helper()
	id, err := idgen.NewUUIDv7()
	if err != nil {
		t.Fatalf("pgtest: gerar id de usuário: %v", err)
	}
	_, err = Pool().Exec(context.Background(),
		`INSERT INTO app_user (id, email, password_hash) VALUES ($1, $2, $3)`,
		id, id+"@example.com", "hash-de-teste",
	)
	if err != nil {
		t.Fatalf("pgtest: criar usuário de teste: %v", err)
	}
	return id
}
