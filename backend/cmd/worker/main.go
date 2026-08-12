// cmd/worker é o consumidor de jobs (01-arquitetura.md §3 e §7): outbox
// relay publicando no barramento in-process, e o worker reivindicando jobs
// com SELECT ... FOR UPDATE SKIP LOCKED. A partir desta rodada, A1
// (planejador) e A2 (prática) estão plugados — RegisterJobs registra
// generate_next_action e plan_track. Sem LLM_API_KEY no ambiente, o worker
// sobe do mesmo jeito: o Gateway fica nil e os handlers viram no-op,
// preservando o fallback determinístico (04-agentes.md §5).
package main

import (
	"context"
	"log/slog"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/phablo/lifeos/internal/modules/goals"
	"github.com/phablo/lifeos/internal/platform/agents"
	"github.com/phablo/lifeos/internal/platform/auth"
	"github.com/phablo/lifeos/internal/platform/config"
	"github.com/phablo/lifeos/internal/platform/db"
	"github.com/phablo/lifeos/internal/platform/eventbus"
	"github.com/phablo/lifeos/internal/platform/jobs"
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

	bus := eventbus.NewBus(pool, logger)
	relay := eventbus.NewRelay(pool, bus, logger)
	worker := jobs.NewWorker(pool, logger)

	gateway := buildGateway(pool, logger)

	users := auth.NewUserStore(pool)
	goalsModule, err := goals.New(pool, users, logger, gateway)
	if err != nil {
		logger.Error("carregar módulo goals", "err", err)
		return
	}
	goalsModule.RegisterJobs(worker)

	logger.Info("worker iniciado",
		"outbox_relay", "ativo",
		"job_queue", "ativa",
		"handlers_registrados", 2,
		"agente_configurado", gateway != nil,
	)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		relay.Run(ctx)
	}()
	go func() {
		defer wg.Done()
		worker.Run(ctx)
	}()

	<-ctx.Done()
	logger.Info("encerrando worker, esperando trabalho em voo terminar")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		logger.Warn("shutdown do worker excedeu o timeout de 30s")
	}

	logger.Info("worker encerrado")
}

// buildGateway lê LLM_* do ambiente (internal/platform/agents/env_config.go)
// e monta o Gateway só se houver API key configurada. Sem ela, devolve nil
// de propósito — RunGenerateNextAction/RunPlanTrack tratam Gateway nil como
// "sem agente disponível" e o fallback determinístico segue valendo
// (04-agentes.md §5): o worker nunca deixa de subir por falta de chave.
func buildGateway(pool *pgxpool.Pool, logger *slog.Logger) *agents.Gateway {
	cfg := agents.LoadEnvConfig()
	if cfg.APIKey == "" {
		logger.Warn("worker: LLM_API_KEY não definida — A1/A2 ficam em modo fallback (sem chamada de agente)")
		return nil
	}
	provider := agents.NewOpenAIProvider(cfg.BaseURL, cfg.APIKey, cfg.Timeout)
	return agents.NewGateway(pool, provider, logger, cfg.Models, cfg.Prices, cfg.MonthlyBudgetUSD)
}
