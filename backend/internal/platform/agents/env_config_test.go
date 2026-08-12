package agents

import (
	"os"
	"testing"
)

func TestLoadEnvConfig_Defaults(t *testing.T) {
	clearLLMEnv(t)
	cfg := LoadEnvConfig()
	if cfg.BaseURL != "https://openrouter.ai/api/v1" {
		t.Fatalf("base url default inesperado: %s", cfg.BaseURL)
	}
	if cfg.MonthlyBudgetUSD != 10.0 {
		t.Fatalf("orçamento default = %f, esperava 10.0", cfg.MonthlyBudgetUSD)
	}
	if cfg.Timeout.Seconds() != 30 {
		t.Fatalf("timeout default = %s, esperava 30s", cfg.Timeout)
	}
}

func TestLoadEnvConfig_ReadsOverrides(t *testing.T) {
	clearLLMEnv(t)
	t.Setenv("LLM_BASE_URL", "https://example.com/v1")
	t.Setenv("LLM_API_KEY", "sk-teste")
	t.Setenv("LLM_MODEL_FAST", "modelo-rapido")
	t.Setenv("LLM_MONTHLY_BUDGET_USD", "25.5")
	t.Setenv("LLM_PRICE_FAST_IN_PER_M", "1.5")

	cfg := LoadEnvConfig()
	if cfg.BaseURL != "https://example.com/v1" {
		t.Fatalf("base url = %s", cfg.BaseURL)
	}
	if cfg.APIKey != "sk-teste" {
		t.Fatalf("api key = %s", cfg.APIKey)
	}
	if cfg.Models[TierFast] != "modelo-rapido" {
		t.Fatalf("model fast = %s", cfg.Models[TierFast])
	}
	if cfg.MonthlyBudgetUSD != 25.5 {
		t.Fatalf("orçamento = %f", cfg.MonthlyBudgetUSD)
	}
	if cfg.Prices[TierFast].InPerMillion != 1.5 {
		t.Fatalf("preço fast in = %f", cfg.Prices[TierFast].InPerMillion)
	}
}

func clearLLMEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"LLM_BASE_URL", "LLM_API_KEY", "LLM_MODEL_FAST", "LLM_MODEL_BALANCED", "LLM_MODEL_STRONG",
		"LLM_MONTHLY_BUDGET_USD", "LLM_TIMEOUT_SECONDS",
		"LLM_PRICE_FAST_IN_PER_M", "LLM_PRICE_FAST_OUT_PER_M",
		"LLM_PRICE_BALANCED_IN_PER_M", "LLM_PRICE_BALANCED_OUT_PER_M",
		"LLM_PRICE_STRONG_IN_PER_M", "LLM_PRICE_STRONG_OUT_PER_M",
	} {
		v, had := os.LookupEnv(k)
		if had {
			t.Cleanup(func(k, v string) func() { return func() { os.Setenv(k, v) } }(k, v))
		} else {
			t.Cleanup(func(k string) func() { return func() { os.Unsetenv(k) } }(k))
		}
		os.Unsetenv(k)
	}
}
