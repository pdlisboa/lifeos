// Package config carrega a configuração do processo a partir de variáveis de ambiente.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL      string
	ListenAddr       string
	Env              string
	MigrationsDir    string
	WebSessionTTL    time.Duration
	MobileSessionTTL time.Duration
	// CorsOrigins habilita o front web (Vite em dev, domínio próprio em prod) a
	// chamar a API com cookie de sessão de outra origem. Vazio = CORS desligado.
	CorsOrigins []string
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL não definida")
	}
	env := getEnv("ENV", "production")
	return &Config{
		DatabaseURL:      dbURL,
		ListenAddr:       getEnv("LISTEN_ADDR", ":8080"),
		Env:              env,
		MigrationsDir:    getEnv("MIGRATIONS_DIR", "db/migrations"),
		WebSessionTTL:    getEnvHours("WEB_SESSION_TTL_HOURS", 720),  // 30 dias
		MobileSessionTTL: getEnvHours("MOBILE_SESSION_TTL_HOURS", 2160), // 90 dias
		CorsOrigins:      getEnvOrigins(env),
	}, nil
}

// getEnvOrigins lê CORS_ORIGINS (lista separada por vírgula). Sem variável
// definida, dev ganha um default pro Vite local; produção fica sem CORS —
// o plano é servir o front pelo mesmo domínio via Caddy (06-frontend.md §10.1).
func getEnvOrigins(env string) []string {
	raw := os.Getenv("CORS_ORIGINS")
	if raw == "" {
		if env == "development" {
			return []string{"http://localhost:5173"}
		}
		return nil
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvHours(key string, fallbackHours int) time.Duration {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n) * time.Hour
		}
	}
	return time.Duration(fallbackHours) * time.Hour
}
