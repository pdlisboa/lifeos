// cmd/api é o servidor HTTP (01-arquitetura.md §3).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/phablo/lifeos/internal/modules/goals"
	"github.com/phablo/lifeos/internal/platform/auth"
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

	users := auth.NewUserStore(pool)
	sessions := auth.NewSessionStore(pool)
	authHandler := &auth.Handler{
		Users:        users,
		Sessions:     sessions,
		Logger:       logger,
		Limiter:      auth.NewRateLimiter(5, time.Minute),
		WebTTL:       cfg.WebSessionTTL,
		MobileTTL:    cfg.MobileSessionTTL,
		CookieSecure: cfg.Env != "development",
	}

	// Falha ruidosa no boot (04-agentes.md §4.6): um domain pack inválido
	// não pode subir silenciosamente — é pior descobrir isso semanas depois.
	goalsModule, err := goals.New(pool, users, logger, nil)
	if err != nil {
		logger.Error("carregar módulo goals", "err", err)
		return
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, obs.RequestLogger(logger))

	// CORS só entra em cena quando há origem configurada (dev com Vite em
	// outra porta, ou front web em domínio separado). AllowCredentials é
	// obrigatório pro cookie de sessão atravessar a origem cruzada.
	if len(cfg.CorsOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   cfg.CorsOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "Authorization"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthz", healthzHandler(pool))
		r.Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(sessions))
			r.Post("/auth/logout", authHandler.Logout)
			r.Get("/me", authHandler.Me)

			goalsModule.RegisterRoutes(r)
		})
	})

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("api ouvindo", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("servidor encerrou com erro", "err", err)
		}
	}()

	<-ctx.Done()
	logger.Info("encerrando")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("erro no shutdown", "err", err)
	}
}

func healthzHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		dbOK := pool.Ping(ctx) == nil
		status := "ok"
		if !dbOK {
			status = "degraded"
		}

		w.Header().Set("Content-Type", "application/json")
		if !dbOK {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": status,
			"db":     dbOK,
			"queue":  true, // fila de jobs chega na Fatia 3+; nada a checar ainda
		})
	}
}
