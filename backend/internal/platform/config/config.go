// Package config carrega a configuração do processo a partir de variáveis de ambiente.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL      string
	ListenAddr       string
	Env              string
	MigrationsDir    string
	WebSessionTTL    time.Duration
	MobileSessionTTL time.Duration
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL não definida")
	}
	return &Config{
		DatabaseURL:      dbURL,
		ListenAddr:       getEnv("LISTEN_ADDR", ":8080"),
		Env:              getEnv("ENV", "production"),
		MigrationsDir:    getEnv("MIGRATIONS_DIR", "db/migrations"),
		WebSessionTTL:    getEnvHours("WEB_SESSION_TTL_HOURS", 720),  // 30 dias
		MobileSessionTTL: getEnvHours("MOBILE_SESSION_TTL_HOURS", 2160), // 90 dias
	}, nil
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
