package agents

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/phablo/lifeos/internal/platform/idgen"
)

const budgetDegradeThreshold = 0.8 // 80%: strong -> balanced (01-arquitetura.md §6.3)

// Gateway implementa os 8 passos de 01-arquitetura.md §6.1: resolve tier,
// monta prompt (já pronto no Request), consulta cache, checa orçamento,
// chama o provider, valida a saída, registra a chamada, devolve tipado.
type Gateway struct {
	pool     *pgxpool.Pool
	provider Provider
	logger   *slog.Logger

	models           map[Tier]string
	prices           map[Tier]Price
	monthlyBudgetUSD float64

	now func() time.Time // testável
}

func NewGateway(pool *pgxpool.Pool, provider Provider, logger *slog.Logger, models map[Tier]string, prices map[Tier]Price, monthlyBudgetUSD float64) *Gateway {
	return &Gateway{
		pool:             pool,
		provider:         provider,
		logger:           logger,
		models:           models,
		prices:           prices,
		monthlyBudgetUSD: monthlyBudgetUSD,
		now:              time.Now,
	}
}

// Run executa uma chamada de agente de ponta a ponta. Nunca devolve saída
// não validada — em caso de falha, quem chama aplica o fallback sem LLM
// (04-agentes.md §5).
func (g *Gateway) Run(ctx context.Context, req Request) (Result, error) {
	tier, err := g.resolveTier(ctx, req.UserID, req.Tier)
	if err != nil {
		return Result{}, err
	}
	model := g.models[tier]

	callID, err := idgen.NewUUIDv7()
	if err != nil {
		return Result{}, fmt.Errorf("agents: gerar id de agent_call: %w", err)
	}

	inputHash := hashInput(req)

	if cached, ok, err := g.lookupCache(ctx, req.UserID, inputHash, req.PromptVersion); err != nil {
		g.logger.Error("agents: consultar cache", "err", err)
	} else if ok {
		g.logCall(ctx, callID, req, tier, model, 0, 0, 0, 0, "ok", "", true, inputHash, cached)
		return Result{
			Output:        cached,
			Model:         model,
			Tier:          tier,
			CacheHit:      true,
			PromptVersion: req.PromptVersion,
			CallID:        callID,
		}, nil
	}

	start := g.now()
	resp, err := g.provider.Complete(ctx, providerRequest(model, req))
	latency := g.now().Sub(start)
	if err != nil {
		g.logCall(ctx, callID, req, tier, model, 0, 0, 0, latency, "provider_error", err.Error(), false, inputHash, nil)
		return Result{}, fmt.Errorf("%w: %v", ErrProviderFailed, err)
	}

	validationErrs := validate(req.Schema, []byte(resp.Content))
	if len(validationErrs) > 0 {
		retryReq := req
		retryReq.UserPrompt = req.UserPrompt + "\n\nA resposta anterior falhou validação de schema:\n- " +
			strings.Join(validationErrs, "\n- ") + "\n\nCorrija e responda de novo seguindo o schema à risca."

		retryStart := g.now()
		resp2, err2 := g.provider.Complete(ctx, providerRequest(model, retryReq))
		latency += g.now().Sub(retryStart)

		if err2 == nil {
			if errs2 := validate(req.Schema, []byte(resp2.Content)); len(errs2) == 0 {
				resp = resp2
				validationErrs = nil
			} else {
				validationErrs = errs2
			}
		}

		if len(validationErrs) > 0 {
			detail := strings.Join(validationErrs, "; ")
			g.logCall(ctx, callID, req, tier, model, resp.InputTokens, resp.OutputTokens, 0, latency, "invalid_output", detail, false, inputHash, nil)
			return Result{}, fmt.Errorf("%w: %s", ErrInvalidOutput, detail)
		}
	}

	cost := g.cost(tier, resp.InputTokens, resp.OutputTokens)
	g.logCall(ctx, callID, req, tier, model, resp.InputTokens, resp.OutputTokens, cost, latency, "ok", "", false, inputHash, json.RawMessage(resp.Content))
	if err := g.addSpend(ctx, req.UserID, cost); err != nil {
		g.logger.Error("agents: atualizar agent_budget", "err", err)
	}

	return Result{
		Output:        json.RawMessage(resp.Content),
		Model:         model,
		Tier:          tier,
		InputTokens:   resp.InputTokens,
		OutputTokens:  resp.OutputTokens,
		CostUSD:       cost,
		LatencyMs:     latency.Milliseconds(),
		PromptVersion: req.PromptVersion,
		CallID:        callID,
	}, nil
}

func providerRequest(model string, req Request) ProviderRequest {
	return ProviderRequest{
		Model:        model,
		SystemPrompt: req.SystemPrompt,
		UserPrompt:   req.UserPrompt,
		SchemaName:   req.SchemaName,
		Schema:       req.Schema,
	}
}

// resolveTier aplica a degradação de 01-arquitetura.md §6.3: ≥80% do
// orçamento do mês rebaixa strong→balanced; ≥100% só deixa passar fast.
func (g *Gateway) resolveTier(ctx context.Context, userID string, requested Tier) (Tier, error) {
	spent, limit, err := g.budgetStatus(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("agents: consultar orçamento: %w", err)
	}
	if limit <= 0 {
		return requested, nil
	}

	ratio := spent / limit
	if ratio >= 1.0 && requested != TierFast {
		return "", ErrBudgetExhausted
	}
	if ratio >= budgetDegradeThreshold && requested == TierStrong {
		g.logger.Warn("agents: orçamento acima de 80%, rebaixando strong para balanced", "spent_usd", spent, "limit_usd", limit)
		return TierBalanced, nil
	}
	return requested, nil
}

func (g *Gateway) budgetStatus(ctx context.Context, userID string) (spent, limit float64, err error) {
	period := periodOn(g.now())
	err = g.pool.QueryRow(ctx, `
		SELECT spent_usd, limit_usd FROM agent_budget WHERE user_id = $1 AND period_on = $2`,
		userID, period,
	).Scan(&spent, &limit)
	if err == pgx.ErrNoRows {
		return 0, g.monthlyBudgetUSD, nil
	}
	if err != nil {
		return 0, 0, err
	}
	return spent, limit, nil
}

func (g *Gateway) addSpend(ctx context.Context, userID string, cost float64) error {
	period := periodOn(g.now())
	_, err := g.pool.Exec(ctx, `
		INSERT INTO agent_budget (user_id, period_on, limit_usd, spent_usd)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, period_on)
		DO UPDATE SET spent_usd = agent_budget.spent_usd + EXCLUDED.spent_usd`,
		userID, period, g.monthlyBudgetUSD, cost,
	)
	return err
}

func periodOn(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func (g *Gateway) lookupCache(ctx context.Context, userID, inputHash, promptVersion string) (json.RawMessage, bool, error) {
	var output json.RawMessage
	err := g.pool.QueryRow(ctx, `
		SELECT output FROM agent_call
		WHERE user_id = $1 AND input_hash = $2 AND prompt_version = $3 AND status = 'ok' AND output IS NOT NULL
		ORDER BY created_at DESC LIMIT 1`,
		userID, inputHash, promptVersion,
	).Scan(&output)
	if err == pgx.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return output, true, nil
}

func (g *Gateway) logCall(ctx context.Context, id string, req Request, tier Tier, model string, inputTokens, outputTokens int, cost float64, latency time.Duration, status, errMsg string, cacheHit bool, inputHash string, output json.RawMessage) {
	_, err := g.pool.Exec(ctx, `
		INSERT INTO agent_call (id, user_id, task, tier, model, prompt_version,
			input_tokens, output_tokens, cost_usd, latency_ms, status, error,
			cache_hit, input_hash, output)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		id, req.UserID, req.Task, tier, model, req.PromptVersion,
		nullableInt(inputTokens), nullableInt(outputTokens), cost, latency.Milliseconds(), status, nullableStr(errMsg),
		cacheHit, inputHash, output,
	)
	if err != nil {
		g.logger.Error("agents: gravar agent_call", "err", err)
	}
}

func nullableInt(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}

func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// cost calcula o custo em USD, arredondado pra cima (numeric(10,6)) —
// errar pra menos é o que estoura o teto mensal (01-arquitetura.md §6.3).
func (g *Gateway) cost(tier Tier, inputTokens, outputTokens int) float64 {
	price := g.prices[tier]
	raw := (float64(inputTokens)/1_000_000)*price.InPerMillion + (float64(outputTokens)/1_000_000)*price.OutPerMillion
	return math.Ceil(raw*1_000_000) / 1_000_000
}

func hashInput(req Request) string {
	h := sha256.New()
	h.Write([]byte(req.Task))
	h.Write([]byte{0})
	h.Write([]byte(req.PromptVersion))
	h.Write([]byte{0})
	h.Write([]byte(req.SchemaName))
	h.Write([]byte{0})
	h.Write([]byte(req.SystemPrompt))
	h.Write([]byte{0})
	h.Write([]byte(req.UserPrompt))
	return hex.EncodeToString(h.Sum(nil))
}
