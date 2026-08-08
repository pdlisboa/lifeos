// cmd/worker é o consumidor de jobs (01-arquitetura.md §3 e §7). Na Fatia 0
// não existe fila ainda — o processo só prova que o entrypoint sobe e mantém
// conexão com o banco, para o compose já nascer no formato final.
package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/phablo/lifeos/internal/platform/config"
	"github.com/phablo/lifeos/internal/platform/db"
	"github.com/phablo/lifeos/internal/platform/obs"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger := obs.NewLogger(cfg.Env)

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("conectar ao banco", "err", err)
		return
	}
	defer pool.Close()

	logger.Info("worker iniciado — nenhuma fila de jobs implementada ainda (roadmap Fatia 3+)")
	<-ctx.Done()
	logger.Info("worker encerrado")
}
