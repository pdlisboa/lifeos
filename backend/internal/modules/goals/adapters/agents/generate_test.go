package agents

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	platformagents "github.com/phablo/lifeos/internal/platform/agents"
	"github.com/phablo/lifeos/internal/testsupport/pgtest"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

var testModels = map[platformagents.Tier]string{
	platformagents.TierFast:     "fast-model",
	platformagents.TierBalanced: "balanced-model",
	platformagents.TierStrong:   "strong-model",
}

var testPrices = map[platformagents.Tier]platformagents.Price{
	platformagents.TierFast:     {InPerMillion: 1, OutPerMillion: 2},
	platformagents.TierBalanced: {InPerMillion: 5, OutPerMillion: 10},
	platformagents.TierStrong:   {InPerMillion: 20, OutPerMillion: 40},
}

func TestGeneratePlan(t *testing.T) {
	pgtest.Reset(t)
	user := pgtest.NewUser(t)

	output := `{"milestones":[` +
		`{"ordinal":1,"title":"Escreve um worker pool básico com channels","completionCriteria":"worker pool sem vazar goroutine, com WaitGroup","competencyKeys":["concurrency"],"carriedOver":false,"sourceLibraryTitle":null},` +
		`{"ordinal":2,"title":"Cobre o worker pool com testes de tabela","completionCriteria":"table-driven test cobrindo erro e sucesso","competencyKeys":["testing"],"carriedOver":false,"sourceLibraryTitle":null},` +
		`{"ordinal":3,"title":"Usa context para cancelar o worker pool","completionCriteria":"context.Context propagado e cancelamento testado","competencyKeys":["concurrency"],"carriedOver":false,"sourceLibraryTitle":null},` +
		`{"ordinal":4,"title":"Abre uma PR real usando o que praticou","completionCriteria":"PR aberta num repositório real, com testes passando","competencyKeys":["concurrency","testing"],"carriedOver":false,"sourceLibraryTitle":null}` +
		`],"rationale":"Você já conhece o básico de goroutines, então comecei pelo worker pool em vez de reexplicar canais.","noChangeNeeded":false}`

	provider := platformagents.NewFakeProvider(platformagents.FakeResponse{Content: output})
	gw := platformagents.NewGateway(pgtest.Pool(), provider, testLogger(), testModels, testPrices, 10.0)

	pack := loadGolangPack(t)
	ctx := BuildPlannerContext(minimalGoal(t), pack, nil, nil, "")

	out, result, err := GeneratePlan(context.Background(), gw, user, ctx)
	if err != nil {
		t.Fatalf("GeneratePlan: %v", err)
	}
	if len(out.Milestones) != 4 {
		t.Fatalf("esperava 4 marcos, teve %d", len(out.Milestones))
	}
	if out.Rationale == "" {
		t.Error("rationale vazio")
	}
	if result.CallID == "" {
		t.Error("CallID vazio — proposal.agent_call_id ficaria sem rastro")
	}
	if result.Tier != platformagents.TierStrong {
		t.Errorf("tier = %s, esperava strong", result.Tier)
	}
}

func TestGeneratePractice(t *testing.T) {
	pgtest.Reset(t)
	user := pgtest.NewUser(t)

	output := `{"title":"Escreva um worker pool que lê 3 URLs em paralelo",` +
		`"detail":"Use goroutines e um WaitGroup para buscar 3 URLs e retornar os status codes.",` +
		`"practiceFormat":"exercise","estimatedMin":20,` +
		`"minimalVariant":"Versão de 5 min: escreva só a assinatura da função e o WaitGroup.",` +
		`"competencyKey":"concurrency",` +
		`"expectedEvidence":{"kind":"code_snippet","description":"o código da função com os 3 status codes"},` +
		`"milestoneId":null,"rationale":"Sinal de concorrência aplicada, não só teoria."}`

	provider := platformagents.NewFakeProvider(platformagents.FakeResponse{Content: output})
	gw := platformagents.NewGateway(pgtest.Pool(), provider, testLogger(), testModels, testPrices, 10.0)

	pack := loadGolangPack(t)
	ctx := BuildPracticeContext(minimalGoal(t), pack, nil, nil, nil, "same", "practice")

	out, result, err := GeneratePractice(context.Background(), gw, user, ctx)
	if err != nil {
		t.Fatalf("GeneratePractice: %v", err)
	}
	if out.Title == "" {
		t.Error("title vazio")
	}
	if out.EstimatedMin < 5 || out.EstimatedMin > 30 {
		t.Errorf("estimatedMin = %d, fora de 5..30", out.EstimatedMin)
	}
	if result.Tier != platformagents.TierFast {
		t.Errorf("tier = %s, esperava fast", result.Tier)
	}
}
