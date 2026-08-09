// cmd/seed popula uma meta de Go com ~10 semanas de histórico realista —
// sessões espalhadas, evidências e eventos de nível progressivos em
// competências diferentes. Sem isso não dá para olhar o Painel de Delta nas
// três fases (05-ux.md §5.2), a projeção (§7.2) ou o gráfico temporal (§5.3)
// sem esperar dez semanas de verdade.
//
// Só roda com ENV=development (CLAUDE.md, "Ambiente"): nunca em produção, e
// nunca silenciosamente — falha ruidosa se a variável não bater.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/app"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/modules/goals/packs"
	"github.com/phablo/lifeos/internal/platform/config"
	"github.com/phablo/lifeos/internal/platform/db"
	"github.com/phablo/lifeos/internal/platform/idgen"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}
	if cfg.Env != "development" {
		fatal(fmt.Errorf("seed só roda com ENV=development (atual: %q) — não é pra encostar em produção", cfg.Env))
	}

	ctx := context.Background()
	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fatal(err)
	}
	defer pool.Close()

	userID, err := targetUserID(ctx, pool)
	if err != nil {
		fatal(fmt.Errorf("nenhum usuário encontrado — rode `make create-user` antes do seed: %w", err))
	}

	reg, err := packs.Load()
	if err != nil {
		fatal(err)
	}
	svc := app.NewService(pool, reg)

	goalID, err := seedGoGoal(ctx, pool, svc, userID)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("seed ok: meta de Go (%s) com ~10 semanas de histórico\n", goalID)
}

// targetUserID usa SEED_EMAIL quando definido — útil pra semear um usuário
// de teste sem mexer nos dados da conta principal — e cai no usuário mais
// antigo (o único, no caso comum de uso pessoal) quando não definido.
func targetUserID(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	var id string
	if email := os.Getenv("SEED_EMAIL"); email != "" {
		err := pool.QueryRow(ctx, `SELECT id FROM app_user WHERE email = $1`, email).Scan(&id)
		return id, err
	}
	err := pool.QueryRow(ctx, `SELECT id FROM app_user ORDER BY created_at LIMIT 1`).Scan(&id)
	return id, err
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "seed: "+err.Error())
	os.Exit(1)
}

// weeksOfHistory é o "~10 semanas" pedido — dá margem confortável acima do
// limiar de 3 semanas da projeção honesta (§7.2) e mostra barras de delta
// com janelas de dezenas de dias, como no exemplo de 05-ux.md §5.
const weeksOfHistory = 10
const daysOfHistory = weeksOfHistory * 7

// seedGoGoal cria (ou reaproveita) a meta "Go" do usuário e, se ela ainda
// não tem histórico, povoa sessões/evidências/eventos de nível dos últimos
// ~70 dias. Rodar `make seed` de novo não duplica.
func seedGoGoal(ctx context.Context, pool *pgxpool.Pool, svc *app.Service, userID string) (string, error) {
	existing, err := findGoalByTitle(ctx, pool, userID, seedGoalTitle)
	if err != nil {
		return "", err
	}
	if existing != "" {
		fmt.Println("seed: meta já existe, pulando criação (apague-a antes pra gerar um histórico novo)")
		return existing, nil
	}

	dod := "Escrever e testar um serviço HTTP concorrente, com tratamento de erro e cancelamento, sem seguir tutorial."
	created, err := svc.CreateGoal(ctx, app.CreateGoalInput{
		UserID: userID, Title: seedGoalTitle, Archetype: domain.ArchetypeSkill, PackID: "golang",
	})
	if err != nil {
		return "", fmt.Errorf("criar meta: %w", err)
	}
	if _, err := svc.PatchGoal(ctx, userID, created.Goal.ID, app.PatchGoalInput{DefinitionOfDone: &dod}); err != nil {
		return "", fmt.Errorf("definir DoD: %w", err)
	}
	if _, err := svc.ActivateGoal(ctx, app.ActivateGoalInput{UserID: userID, GoalID: created.Goal.ID}); err != nil {
		return "", fmt.Errorf("ativar meta: %w", err)
	}

	now := time.Now().UTC()
	activatedAt := now.AddDate(0, 0, -daysOfHistory)
	if err := backdateGoalActivation(ctx, pool, userID, created.Goal.ID, activatedAt, now); err != nil {
		return "", fmt.Errorf("retroceder data de ativação: %w", err)
	}

	comps, err := postgres.ListCompetenciesByGoal(ctx, pool, created.Goal.ID)
	if err != nil {
		return "", fmt.Errorf("listar competências: %w", err)
	}
	byKey := make(map[string]domain.Competency, len(comps))
	for _, c := range comps {
		byKey[c.PackKey] = c
	}

	if err := seedSessions(ctx, pool, created.Goal.ID, userID, now); err != nil {
		return "", fmt.Errorf("semear sessões: %w", err)
	}
	if err := seedCompetencyProgress(ctx, pool, created.Goal.ID, userID, byKey, now); err != nil {
		return "", fmt.Errorf("semear evidências e níveis: %w", err)
	}

	return created.Goal.ID, nil
}

const seedGoalTitle = "Aprender Go"

func findGoalByTitle(ctx context.Context, pool *pgxpool.Pool, userID, title string) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `SELECT id FROM goal WHERE user_id = $1 AND title = $2 LIMIT 1`, userID, title).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return id, nil
}

// backdateGoalActivation empurra ActivatedAt/LastActivityAt pro passado —
// ActivateGoal usa time.Now() de propósito (é o caminho real do produto),
// então retroceder é responsabilidade deste script, direto no banco.
func backdateGoalActivation(ctx context.Context, pool *pgxpool.Pool, userID, goalID string, activatedAt, lastActivityAt time.Time) error {
	g, err := postgres.GetGoal(ctx, pool, userID, goalID)
	if err != nil {
		return err
	}
	g.ActivatedAt = &activatedAt
	g.LastActivityAt = &lastActivityAt
	return postgres.UpdateGoal(ctx, pool, g)
}

// seedSessions espalha 3 sessões por semana ao longo de weeksOfHistory,
// terminando ontem — dá ritmo real suficiente pra projeção (>= 3 semanas)
// e pra consistência (§7.5) sem cravar todo santo dia (isso pareceria
// robótico, e ninguém pratica todo santo dia igual).
func seedSessions(ctx context.Context, pool *pgxpool.Pool, goalID, userID string, now time.Time) error {
	durations := []int{30, 45, 60}
	offsetsInWeek := []int{1, 3, 5}

	for week := 0; week < weeksOfHistory; week++ {
		for i, dayInWeek := range offsetsInWeek {
			daysAgo := week*7 + dayInWeek
			startedAt := now.AddDate(0, 0, -daysAgo)
			duration := durations[(week+i)%len(durations)]

			id, err := idgen.NewUUIDv7()
			if err != nil {
				return err
			}
			s, err := domain.NewSession(id, goalID, userID, nil, startedAt, duration, nil, nil, time.UTC, startedAt)
			if err != nil {
				return err
			}
			if err := postgres.InsertSession(ctx, pool, s); err != nil {
				return err
			}
		}
	}
	return nil
}

// progressEvent é um passo na evolução de uma competência: um evento de
// nível, opcionalmente sustentado por uma evidência (que também pode conter
// mais evidências "soltas", sem mudar o nível — é assim que o museu ganha
// mais de uma entrada por competência pra comparação lado a lado, §7.4).
type progressEvent struct {
	daysAgo   int
	toLevel   int
	rationale string
	evidence  *evidenceSeed // nil quando o nível nasce sem uma evidência específica
}

type evidenceSeed struct {
	daysAgo int
	kind    domain.EvidenceKind
	title   string
	body    string
}

// seedCompetencyProgress é o coração do seed: replica, com dados de
// verdade, o exemplo de 05-ux.md §5 — duas competências que sobem bem
// (concurrency, error_handling), uma que sobe pouco (testing), uma medida
// uma vez só e depois esquecida (interfaces_design, pro ⟳ de "sem
// evidência recente"), e duas nunca medidas (idioms, tooling_modules, pro
// "ainda não medimos" — null nunca vira 0).
func seedCompetencyProgress(ctx context.Context, pool *pgxpool.Pool, goalID, userID string, byKey map[string]domain.Competency, now time.Time) error {
	plan := map[string][]progressEvent{
		"concurrency": {
			{daysAgo: 70, toLevel: 2, rationale: "ponto de partida da sondagem inicial"},
			{
				daysAgo: 45, toLevel: 3, rationale: "escrevi um worker pool com channels e select, sem consultar",
				evidence: &evidenceSeed{daysAgo: 45, kind: domain.EvidenceCodeSnippet, title: "worker pool com channels",
					body: "func workerPool(jobs <-chan int, results chan<- int) {\n\tfor i := 0; i < 4; i++ {\n\t\tgo func() {\n\t\t\tfor j := range jobs {\n\t\t\t\tresults <- j * j\n\t\t\t}\n\t\t}()\n\t}\n}"},
			},
			{
				daysAgo: 20, toLevel: 4, rationale: "usei errgroup com cancelamento correto e tratei o erro do contexto",
				evidence: &evidenceSeed{daysAgo: 20, kind: domain.EvidenceCodeSnippet, title: "fetchAll com errgroup",
					body: "func fetchAll(ctx context.Context, urls []string) error {\n\tg, ctx := errgroup.WithContext(ctx)\n\tfor _, u := range urls {\n\t\tu := u\n\t\tg.Go(func() error { return fetch(ctx, u) })\n\t}\n\treturn g.Wait()\n}"},
			},
			{daysAgo: 30, toLevel: 3, rationale: "", evidence: &evidenceSeed{daysAgo: 30, kind: domain.EvidenceCodeSnippet, title: "select com cancelamento por context", body: "select {\ncase <-ctx.Done():\n\treturn ctx.Err()\ncase v := <-ch:\n\treturn process(v)\n}"}},
		},
		"error_handling": {
			{daysAgo: 70, toLevel: 2, rationale: "ponto de partida da sondagem inicial"},
			{
				daysAgo: 35, toLevel: 3, rationale: "passei a envolver erro com %w e checar com errors.Is",
				evidence: &evidenceSeed{daysAgo: 35, kind: domain.EvidenceCodeSnippet, title: "wrap de erro com contexto",
					body: "if err != nil {\n\treturn fmt.Errorf(\"buscar usuário %s: %w\", id, err)\n}"},
			},
			{
				daysAgo: 12, toLevel: 4, rationale: "defini um erro sentinela pro caso de not-found",
				evidence: &evidenceSeed{daysAgo: 12, kind: domain.EvidenceCodeSnippet, title: "erro sentinela",
					body: "var ErrNotFound = errors.New(\"não encontrado\")\n\nif errors.Is(err, ErrNotFound) {\n\treturn nil\n}"},
			},
			{daysAgo: 66, toLevel: 2, rationale: "", evidence: &evidenceSeed{daysAgo: 66, kind: domain.EvidenceWrittenText, title: "primeiro contato com error handling", body: "Ainda checo err != nil e só dou log — não sei o que fazer além disso na maior parte dos casos."}},
		},
		"testing": {
			{daysAgo: 70, toLevel: 1, rationale: "ponto de partida da sondagem inicial"},
			{
				daysAgo: 25, toLevel: 2, rationale: "escrevi um table-driven test cobrindo os casos de erro",
				evidence: &evidenceSeed{daysAgo: 25, kind: domain.EvidenceCodeSnippet, title: "table-driven test",
					body: "func TestParse(t *testing.T) {\n\tcases := []struct{ in string; want int; wantErr bool }{\n\t\t{\"3\", 3, false},\n\t\t{\"x\", 0, true},\n\t}\n\tfor _, tc := range cases {\n\t\tt.Run(tc.in, func(t *testing.T) { /* ... */ })\n\t}\n}"},
			},
			{daysAgo: 8, toLevel: 2, rationale: "", evidence: &evidenceSeed{daysAgo: 8, kind: domain.EvidenceCodeSnippet, title: "teste de concorrência com -race", body: "go test -race ./...\n// passou limpo depois de trocar o mutex de lugar"}},
		},
		"interfaces_design": {
			{daysAgo: 68, toLevel: 3, rationale: "já uso interfaces pequenas do lado de quem consome, de outra experiência"},
		},
	}

	for packKey, events := range plan {
		comp, ok := byKey[packKey]
		if !ok {
			return fmt.Errorf("competência %q não existe no pack golang", packKey)
		}
		if err := applyProgressEvents(ctx, pool, goalID, userID, &comp, events, now); err != nil {
			return fmt.Errorf("competência %s: %w", packKey, err)
		}
	}
	return nil
}

// applyProgressEvents insere as evidências e eventos de nível de uma
// competência em ordem cronológica — precisa ser em ordem porque
// Competency.ApplyLevelEvent congela o baseline no primeiro evento
// (02-modelo-de-dados.md §4.2) e só o último grava o cache final.
func applyProgressEvents(ctx context.Context, pool *pgxpool.Pool, goalID, userID string, comp *domain.Competency, events []progressEvent, now time.Time) error {
	// mais antigo primeiro — ApplyLevelEvent congela o baseline no primeiro
	// evento que processa, então a ordem cronológica importa de verdade.
	ordered := make([]progressEvent, len(events))
	copy(ordered, events)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].daysAgo > ordered[j].daysAgo })

	for _, e := range ordered {
		occurredAt := now.AddDate(0, 0, -e.daysAgo)

		var evidenceID *string
		if e.evidence != nil {
			id, err := insertEvidenceSeed(ctx, pool, goalID, userID, comp.ID, now, *e.evidence)
			if err != nil {
				return err
			}
			evidenceID = &id
		}

		if e.rationale == "" {
			// evidência "solta": contribui pro museu sem mexer no nível.
			continue
		}

		levelID, err := idgen.NewUUIDv7()
		if err != nil {
			return err
		}
		ev, err := domain.NewLevelEvent(levelID, comp.ID, userID, comp.CurrentLevel, e.toLevel, domain.ConfidenceHigh, domain.SourceSelf, evidenceID, e.rationale, occurredAt)
		if err != nil {
			return err
		}
		if err := comp.ApplyLevelEvent(*ev); err != nil {
			return err
		}
		if err := postgres.InsertLevelEvent(ctx, pool, ev); err != nil {
			return err
		}
	}
	return postgres.UpdateCompetencyState(ctx, pool, comp)
}

func insertEvidenceSeed(ctx context.Context, pool *pgxpool.Pool, goalID, userID, competencyID string, now time.Time, seed evidenceSeed) (string, error) {
	occurredAt := now.AddDate(0, 0, -seed.daysAgo)
	id, err := idgen.NewUUIDv7()
	if err != nil {
		return "", err
	}
	body := seed.body
	title := seed.title
	ev, err := domain.NewEvidence(id, goalID, userID, nil, seed.kind, &title, &body, nil, nil, nil, nil, time.UTC, occurredAt)
	if err != nil {
		return "", err
	}
	if err := postgres.InsertEvidence(ctx, pool, ev); err != nil {
		return "", err
	}
	if err := postgres.InsertEvidenceCompetency(ctx, pool, ev.ID, competencyID); err != nil {
		return "", err
	}
	return ev.ID, nil
}
