// Comando migrate: aplica migrations (up/status) e cria o usuário único do
// sistema (create-user) — necessário para o critério de pronto da Fatia 0.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/phablo/lifeos/internal/platform/auth"
	"github.com/phablo/lifeos/internal/platform/config"
	"github.com/phablo/lifeos/internal/platform/db"
	"github.com/phablo/lifeos/internal/platform/idgen"
	"github.com/phablo/lifeos/internal/platform/migrate"
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

	ctx := context.Background()
	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fatal(err)
	}
	defer pool.Close()

	runner := migrate.NewRunner(pool, cfg.MigrationsDir)

	switch os.Args[1] {
	case "up":
		if err := runner.Up(ctx); err != nil {
			fatal(err)
		}
	case "status":
		if err := runner.Status(ctx); err != nil {
			fatal(err)
		}
	case "create-user":
		if len(os.Args) != 4 {
			fmt.Println("uso: migrate create-user <email> <senha>")
			os.Exit(1)
		}
		if err := createUser(ctx, auth.NewUserStore(pool), os.Args[2], os.Args[3]); err != nil {
			fatal(err)
		}
	default:
		usage()
		os.Exit(1)
	}
}

func createUser(ctx context.Context, users *auth.UserStore, email, password string) error {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("gerar hash: %w", err)
	}
	id, err := idgen.NewUUIDv7()
	if err != nil {
		return fmt.Errorf("gerar id: %w", err)
	}
	if err := users.Create(ctx, id, email, hash); err != nil {
		return err
	}
	fmt.Printf("usuário criado: %s (%s)\n", email, id)
	return nil
}

func usage() {
	fmt.Println("uso: migrate <up|status|create-user>")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "erro:", err)
	os.Exit(1)
}
