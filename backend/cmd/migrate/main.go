// Comando migrate: aplica migrations via goose (up/status) e cria o usuário
// único do sistema (create-user) — necessário para o critério de pronto da
// Fatia 0. goose é a ferramenta travada em 02-modelo-de-dados.md §2.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/phablo/lifeos/internal/platform/auth"
	"github.com/phablo/lifeos/internal/platform/config"
	"github.com/phablo/lifeos/internal/platform/db"
	"github.com/phablo/lifeos/internal/platform/idgen"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}

	switch os.Args[1] {
	case "up", "status":
		runGoose(cfg, os.Args[1])
	case "create-user":
		if len(os.Args) != 4 {
			fmt.Println("uso: migrate create-user <email> <senha>")
			os.Exit(1)
		}
		runCreateUser(cfg, os.Args[2], os.Args[3])
	default:
		usage()
		os.Exit(1)
	}
}

// goose fala database/sql; pgx/v5/stdlib registra o driver "pgx" em cima do
// mesmo driver nativo que o resto do processo usa via pgxpool.
func runGoose(cfg *config.Config, cmd string) {
	sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		fatal(fmt.Errorf("abrir conexão: %w", err))
	}
	defer sqlDB.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		fatal(err)
	}

	ctx := context.Background()
	switch cmd {
	case "up":
		err = goose.UpContext(ctx, sqlDB, cfg.MigrationsDir)
	case "status":
		err = goose.StatusContext(ctx, sqlDB, cfg.MigrationsDir)
	}
	if err != nil {
		fatal(err)
	}
}

func runCreateUser(cfg *config.Config, email, password string) {
	ctx := context.Background()
	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fatal(err)
	}
	defer pool.Close()

	hash, err := auth.HashPassword(password)
	if err != nil {
		fatal(fmt.Errorf("gerar hash: %w", err))
	}
	id, err := idgen.NewUUIDv7()
	if err != nil {
		fatal(fmt.Errorf("gerar id: %w", err))
	}
	if err := auth.NewUserStore(pool).Create(ctx, id, email, hash); err != nil {
		fatal(err)
	}
	fmt.Printf("usuário criado: %s (%s)\n", email, id)
}

func usage() {
	fmt.Println("uso: migrate <up|status|create-user>")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "erro:", err)
	os.Exit(1)
}
