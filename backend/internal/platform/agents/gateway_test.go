package agents

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/testsupport/pgtest"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

const testSchema = `{
	"type": "object",
	"required": ["title", "estimatedMin"],
	"additionalProperties": false,
	"properties": {
		"title": {"type": "string", "minLength": 5},
		"estimatedMin": {"type": "integer", "minimum": 5, "maximum": 30}
	}
}`

var testModels = map[Tier]string{
	TierFast:     "fast-model",
	TierBalanced: "balanced-model",
	TierStrong:   "strong-model",
}

var testPrices = map[Tier]Price{
	TierFast:     {InPerMillion: 1, OutPerMillion: 2},
	TierBalanced: {InPerMillion: 5, OutPerMillion: 10},
	TierStrong:   {InPerMillion: 20, OutPerMillion: 40},
}

func newTestRequest(userID string) Request {
	return Request{
		Task:          "generate_action",
		Tier:          TierFast,
		UserID:        userID,
		PromptVersion: "practice.v1",
		SystemPrompt:  "você é o Gerador de Prática",
		UserPrompt:    "gere uma ação para worker pool",
		SchemaName:    "practice",
		Schema:        []byte(testSchema),
	}
}

func TestGateway_ValidOutputPassesAndLogsCall(t *testing.T) {
	pgtest.Reset(t)
	user := pgtest.NewUser(t)

	provider := NewFakeProvider(FakeResponse{
		Content:      `{"title":"Worker pool com channels","estimatedMin":20}`,
		InputTokens:  100,
		OutputTokens: 50,
	})
	gw := NewGateway(pgtest.Pool(), provider, testLogger(), testModels, testPrices, 10.0)

	res, err := gw.Run(context.Background(), newTestRequest(user))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.CacheHit {
		t.Fatal("primeira chamada não deveria ser cache hit")
	}
	if res.Model != "fast-model" {
		t.Fatalf("model = %s, esperava fast-model", res.Model)
	}
	if string(res.Output) != `{"title":"Worker pool com channels","estimatedMin":20}` {
		t.Fatalf("output inesperado: %s", res.Output)
	}
	wantCost := (100.0/1_000_000)*1 + (50.0/1_000_000)*2
	if res.CostUSD < wantCost {
		t.Fatalf("cost_usd = %f, esperava pelo menos %f (arredonda pra cima)", res.CostUSD, wantCost)
	}

	var status string
	var cacheHit bool
	err = pgtest.Pool().QueryRow(context.Background(),
		`SELECT status, cache_hit FROM agent_call WHERE user_id = $1`, user,
	).Scan(&status, &cacheHit)
	if err != nil {
		t.Fatalf("consultar agent_call: %v", err)
	}
	if status != "ok" || cacheHit {
		t.Fatalf("agent_call inesperado: status=%s cache_hit=%v", status, cacheHit)
	}
}

func TestGateway_InvalidOutputRetriesThenReturnsTypedError(t *testing.T) {
	pgtest.Reset(t)
	user := pgtest.NewUser(t)

	provider := NewFakeProvider(
		FakeResponse{Content: `{"title":"oi"}`},          // curto demais + falta estimatedMin
		FakeResponse{Content: `{"title":"ainda curto"}`}, // 2ª tentativa também inválida
	)
	gw := NewGateway(pgtest.Pool(), provider, testLogger(), testModels, testPrices, 10.0)

	_, err := gw.Run(context.Background(), newTestRequest(user))
	if !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("esperava ErrInvalidOutput, teve: %v", err)
	}
	if calls := len(provider.Calls()); calls != 2 {
		t.Fatalf("esperava 2 chamadas ao provider (1 + retry), teve %d", calls)
	}

	var status string
	err = pgtest.Pool().QueryRow(context.Background(),
		`SELECT status FROM agent_call WHERE user_id = $1`, user,
	).Scan(&status)
	if err != nil {
		t.Fatalf("consultar agent_call: %v", err)
	}
	if status != "invalid_output" {
		t.Fatalf("status = %s, esperava invalid_output", status)
	}
}

func TestGateway_InvalidOutputRecoversOnRetry(t *testing.T) {
	pgtest.Reset(t)
	user := pgtest.NewUser(t)

	provider := NewFakeProvider(
		FakeResponse{Content: `{"title":"oi"}`},                                         // inválido
		FakeResponse{Content: `{"title":"Worker pool com channels","estimatedMin":15}`}, // corrige na retentativa
	)
	gw := NewGateway(pgtest.Pool(), provider, testLogger(), testModels, testPrices, 10.0)

	res, err := gw.Run(context.Background(), newTestRequest(user))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if string(res.Output) != `{"title":"Worker pool com channels","estimatedMin":15}` {
		t.Fatalf("output inesperado: %s", res.Output)
	}
	if calls := len(provider.Calls()); calls != 2 {
		t.Fatalf("esperava 2 chamadas ao provider, teve %d", calls)
	}
}

func TestGateway_SecondIdenticalCallIsCacheHitAndSkipsProvider(t *testing.T) {
	pgtest.Reset(t)
	user := pgtest.NewUser(t)

	provider := NewFakeProvider(FakeResponse{
		Content:      `{"title":"Worker pool com channels","estimatedMin":20}`,
		InputTokens:  100,
		OutputTokens: 50,
	})
	gw := NewGateway(pgtest.Pool(), provider, testLogger(), testModels, testPrices, 10.0)

	req := newTestRequest(user)
	res1, err := gw.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("1ª Run: %v", err)
	}
	res2, err := gw.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("2ª Run: %v", err)
	}

	if res2.CacheHit != true {
		t.Fatal("2ª chamada idêntica deveria ser cache hit")
	}
	// jsonb normaliza espaçamento ao voltar do banco, então compara o valor
	// decodificado, não os bytes crus.
	var v1, v2 map[string]any
	if err := json.Unmarshal(res1.Output, &v1); err != nil {
		t.Fatalf("decodificar output 1: %v", err)
	}
	if err := json.Unmarshal(res2.Output, &v2); err != nil {
		t.Fatalf("decodificar output 2: %v", err)
	}
	if !reflect.DeepEqual(v1, v2) {
		t.Fatalf("output do cache diverge: %v != %v", v2, v1)
	}
	if calls := len(provider.Calls()); calls != 1 {
		t.Fatalf("provider deveria ter sido chamado só 1 vez (2ª veio do cache), foi chamado %d vezes", calls)
	}
}

func TestGateway_BudgetDegradesStrongToBalancedAtEightyPercent(t *testing.T) {
	pgtest.Reset(t)
	user := pgtest.NewUser(t)
	seedBudget(t, user, 8.5, 10.0)

	provider := NewFakeProvider(FakeResponse{
		Content:      `{"title":"Trilha revisada com foco em concorrência","estimatedMin":20}`,
		InputTokens:  10,
		OutputTokens: 10,
	})
	gw := NewGateway(pgtest.Pool(), provider, testLogger(), testModels, testPrices, 10.0)

	req := newTestRequest(user)
	req.Tier = TierStrong
	res, err := gw.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Tier != TierBalanced {
		t.Fatalf("tier = %s, esperava rebaixar strong -> balanced a 85%% de orçamento", res.Tier)
	}
	if res.Model != testModels[TierBalanced] {
		t.Fatalf("model = %s, esperava %s", res.Model, testModels[TierBalanced])
	}
}

func TestGateway_BudgetBlocksNonFastTierAtLimit(t *testing.T) {
	pgtest.Reset(t)
	user := pgtest.NewUser(t)
	seedBudget(t, user, 10.0, 10.0)

	provider := NewFakeProvider(FakeResponse{Content: `{"title":"não deveria ser chamado","estimatedMin":10}`})
	gw := NewGateway(pgtest.Pool(), provider, testLogger(), testModels, testPrices, 10.0)

	req := newTestRequest(user)
	req.Tier = TierBalanced
	_, err := gw.Run(context.Background(), req)
	if !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("esperava ErrBudgetExhausted, teve: %v", err)
	}
	if calls := len(provider.Calls()); calls != 0 {
		t.Fatalf("provider não deveria ter sido chamado, foi %d vezes", calls)
	}
}

func TestGateway_BudgetStillAllowsFastTierAtLimit(t *testing.T) {
	pgtest.Reset(t)
	user := pgtest.NewUser(t)
	seedBudget(t, user, 10.0, 10.0)

	provider := NewFakeProvider(FakeResponse{
		Content:      `{"title":"ação rápida de fallback","estimatedMin":10}`,
		InputTokens:  5,
		OutputTokens: 5,
	})
	gw := NewGateway(pgtest.Pool(), provider, testLogger(), testModels, testPrices, 10.0)

	req := newTestRequest(user)
	req.Tier = TierFast
	res, err := gw.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run com tier fast no limite deveria funcionar: %v", err)
	}
	if res.Tier != TierFast {
		t.Fatalf("tier = %s, esperava fast", res.Tier)
	}
}

func TestGateway_ProviderErrorIsWrappedAndLogged(t *testing.T) {
	pgtest.Reset(t)
	user := pgtest.NewUser(t)

	provider := NewFakeProvider(FakeResponse{Err: errors.New("timeout de rede")})
	gw := NewGateway(pgtest.Pool(), provider, testLogger(), testModels, testPrices, 10.0)

	_, err := gw.Run(context.Background(), newTestRequest(user))
	if !errors.Is(err, ErrProviderFailed) {
		t.Fatalf("esperava ErrProviderFailed, teve: %v", err)
	}

	var status string
	err = pgtest.Pool().QueryRow(context.Background(),
		`SELECT status FROM agent_call WHERE user_id = $1`, user,
	).Scan(&status)
	if err != nil {
		t.Fatalf("consultar agent_call: %v", err)
	}
	if status != "provider_error" {
		t.Fatalf("status = %s, esperava provider_error", status)
	}
}

func seedBudget(t *testing.T, userID string, spent, limit float64) {
	t.Helper()
	period := time.Now().UTC()
	period = time.Date(period.Year(), period.Month(), 1, 0, 0, 0, 0, time.UTC)
	_, err := pgtest.Pool().Exec(context.Background(), `
		INSERT INTO agent_budget (user_id, period_on, limit_usd, spent_usd) VALUES ($1, $2, $3, $4)`,
		userID, period, limit, spent,
	)
	if err != nil {
		t.Fatalf("seedBudget: %v", err)
	}
}
