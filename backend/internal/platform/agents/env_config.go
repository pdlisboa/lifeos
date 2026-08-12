package agents

import (
	"os"
	"strconv"
	"time"
)

// Price é o custo por milhão de tokens de um tier, usado pra calcular
// cost_usd (01-arquitetura.md §6.3). Fica por tier — não por modelo exato —
// porque com só 3 tiers e troca de modelo já sendo por env var, uma tabela
// fina por modelo seria complexidade sem uso real ainda.
type Price struct {
	InPerMillion  float64
	OutPerMillion float64
}

// EnvConfig é a configuração do gateway lida de variáveis de ambiente
// (01-arquitetura.md §6.1.1). Fica dentro do próprio pacote agents — não em
// platform/config — porque nenhum cmd/ desta fatia instancia o Gateway
// ainda: não há chamador real, então não faz sentido inchar a config
// compartilhada com campos que nada lê.
type EnvConfig struct {
	BaseURL          string
	APIKey           string
	Models           map[Tier]string
	Prices           map[Tier]Price
	MonthlyBudgetUSD float64
	Timeout          time.Duration
}

// LoadEnvConfig lê LLM_BASE_URL, LLM_API_KEY, LLM_MODEL_FAST/BALANCED/STRONG,
// LLM_MONTHLY_BUDGET_USD e LLM_PRICE_<TIER>_IN_PER_M / _OUT_PER_M. Nunca
// falha — variáveis ausentes viram valores default/zero; quem for
// efetivamente ligar o Gateway decide se isso é suficiente pra subir.
func LoadEnvConfig() EnvConfig {
	return EnvConfig{
		BaseURL: getEnv("LLM_BASE_URL", "https://openrouter.ai/api/v1"),
		APIKey:  os.Getenv("LLM_API_KEY"),
		Models: map[Tier]string{
			TierFast:     os.Getenv("LLM_MODEL_FAST"),
			TierBalanced: os.Getenv("LLM_MODEL_BALANCED"),
			TierStrong:   os.Getenv("LLM_MODEL_STRONG"),
		},
		Prices: map[Tier]Price{
			TierFast:     {InPerMillion: getEnvFloat("LLM_PRICE_FAST_IN_PER_M", 0), OutPerMillion: getEnvFloat("LLM_PRICE_FAST_OUT_PER_M", 0)},
			TierBalanced: {InPerMillion: getEnvFloat("LLM_PRICE_BALANCED_IN_PER_M", 0), OutPerMillion: getEnvFloat("LLM_PRICE_BALANCED_OUT_PER_M", 0)},
			TierStrong:   {InPerMillion: getEnvFloat("LLM_PRICE_STRONG_IN_PER_M", 0), OutPerMillion: getEnvFloat("LLM_PRICE_STRONG_OUT_PER_M", 0)},
		},
		MonthlyBudgetUSD: getEnvFloat("LLM_MONTHLY_BUDGET_USD", 10.0),
		Timeout:          getEnvSeconds("LLM_TIMEOUT_SECONDS", 30),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getEnvSeconds(key string, fallbackSeconds int) time.Duration {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return time.Duration(fallbackSeconds) * time.Second
}
