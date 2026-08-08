// Package migrate aplica os arquivos SQL de db/migrations em ordem, sempre
// para frente (01-arquitetura.md §10.2, 02-modelo-de-dados.md §16: sem down).
package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Runner struct {
	pool *pgxpool.Pool
	dir  string
}

func NewRunner(pool *pgxpool.Pool, dir string) *Runner {
	return &Runner{pool: pool, dir: dir}
}

func (r *Runner) ensureTable(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`)
	return err
}

func (r *Runner) files() ([]string, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return nil, fmt.Errorf("ler %s: %w", r.dir, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func (r *Runner) applied(ctx context.Context) (map[string]bool, error) {
	rows, err := r.pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	done := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		done[v] = true
	}
	return done, rows.Err()
}

func (r *Runner) Status(ctx context.Context) error {
	if err := r.ensureTable(ctx); err != nil {
		return err
	}
	names, err := r.files()
	if err != nil {
		return err
	}
	done, err := r.applied(ctx)
	if err != nil {
		return err
	}
	for _, n := range names {
		state := "pendente"
		if done[n] {
			state = "aplicada"
		}
		fmt.Printf("%-30s %s\n", n, state)
	}
	return nil
}

func (r *Runner) Up(ctx context.Context) error {
	if err := r.ensureTable(ctx); err != nil {
		return fmt.Errorf("preparar schema_migrations: %w", err)
	}
	names, err := r.files()
	if err != nil {
		return err
	}
	done, err := r.applied(ctx)
	if err != nil {
		return err
	}

	for _, name := range names {
		if done[name] {
			continue
		}
		content, err := os.ReadFile(filepath.Join(r.dir, name))
		if err != nil {
			return fmt.Errorf("ler %s: %w", name, err)
		}

		tx, err := r.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("iniciar transação para %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, string(content)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("aplicar %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("registrar %s: %w", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit %s: %w", name, err)
		}
		fmt.Printf("aplicada: %s\n", name)
	}
	return nil
}
