// cmd/evalexport lê os casos marcados como eval (evidence_eval_case,
// 04-agentes.md §6.1) e escreve um JSON por caso em evals/assessor/ — o
// conjunto que vai medir o A3 quando ele existir (Fatia 4). Puro dado: não
// chama agente nenhum, não decide nível de ninguém.
//
// Roda via `make eval-export`, que monta o diretório de saída como volume —
// a imagem não carrega o código-fonte, só os binários (backend/Dockerfile).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/modules/goals/packs"
	"github.com/phablo/lifeos/internal/platform/config"
	"github.com/phablo/lifeos/internal/platform/db"
)

// outputDir é o caminho *dentro do container* — bind-montado pelo alvo
// `eval-export` do Makefile em backend/internal/modules/goals/adapters/agents/evals.
// Fora do container (go run direto), pode ser sobrescrito por EVAL_EXPORT_DIR.
const defaultOutputDir = "/app/evals/assessor"

func main() {
	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}

	ctx := context.Background()
	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fatal(err)
	}
	defer pool.Close()

	reg, err := packs.Load()
	if err != nil {
		fatal(err)
	}

	rows, err := postgres.ListEvalCasesForExport(ctx, pool)
	if err != nil {
		fatal(err)
	}

	outDir := os.Getenv("EVAL_EXPORT_DIR")
	if outDir == "" {
		outDir = defaultOutputDir
	}

	cases := make([]evalCaseFile, 0, len(rows))
	for _, row := range rows {
		c, err := buildCase(ctx, pool, reg, row)
		if err != nil {
			fatal(fmt.Errorf("montar caso %s: %w", row.EvidenceID, err))
		}
		cases = append(cases, c)
	}

	if err := writeCases(outDir, cases); err != nil {
		fatal(err)
	}
	fmt.Printf("eval-export: %d caso(s) escrito(s) em %s\n", len(cases), outDir)
}

type evalCaseFile struct {
	ID       string           `json:"id"`
	Reason   string           `json:"reason"`
	MarkedAt string           `json:"markedAt"`
	Input    evalCaseInput    `json:"input"`
	Expected evalCaseExpected `json:"expected"`
}

type evalCaseInput struct {
	Pack         evalCasePack         `json:"pack"`
	Competencies []evalCaseCompetency `json:"competencies"`
	Action       *evalCaseAction      `json:"action"`
	Evidence     evalCaseEvidence     `json:"evidence"`
}

type evalCasePack struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
	Label   string `json:"label"`
}

type evalCaseCompetency struct {
	PackKey    string  `json:"packKey"`
	Label      string  `json:"label"`
	Weight     float64 `json:"weight"`
	Level      *int    `json:"level"`
	Confidence string  `json:"confidence"`
}

type evalCaseAction struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type evalCaseEvidence struct {
	Kind         string `json:"kind"`
	Title        string `json:"title,omitempty"`
	Body         string `json:"body,omitempty"`
	ExternalURL  string `json:"externalUrl,omitempty"`
	IsTranscript bool   `json:"isTranscript"`
}

type evalCaseExpected struct {
	Scores []evalCaseExpectedScore `json:"scores"`
}

type evalCaseExpectedScore struct {
	CompetencyKey string `json:"competencyKey"`
	Level         int    `json:"level"`
}

func buildCase(ctx context.Context, q postgres.Querier, reg *packs.Registry, row postgres.EvalCaseExportRow) (evalCaseFile, error) {
	goal, err := postgres.GetGoal(ctx, q, row.UserID, row.GoalID)
	if err != nil {
		return evalCaseFile{}, fmt.Errorf("buscar goal: %w", err)
	}
	comps, err := postgres.ListCompetenciesByGoal(ctx, q, row.GoalID)
	if err != nil {
		return evalCaseFile{}, fmt.Errorf("listar competências: %w", err)
	}
	byID := make(map[string]domain.Competency, len(comps))
	compViews := make([]evalCaseCompetency, len(comps))
	for i, c := range comps {
		byID[c.ID] = c
		compViews[i] = evalCaseCompetency{
			PackKey: c.PackKey, Label: c.Label, Weight: c.Weight,
			Level: c.CurrentLevel, Confidence: string(c.Confidence),
		}
	}

	pack, ok := reg.Get(goal.PackID)
	if !ok {
		return evalCaseFile{}, fmt.Errorf("pack %q não encontrado no registro", goal.PackID)
	}

	var action *evalCaseAction
	if row.ActionID != nil {
		a, err := postgres.GetAction(ctx, q, row.UserID, *row.ActionID)
		if err != nil {
			return evalCaseFile{}, fmt.Errorf("buscar ação: %w", err)
		}
		detail := ""
		if a.Detail != nil {
			detail = *a.Detail
		}
		action = &evalCaseAction{Title: a.Title, Detail: detail}
	}

	evalCase, err := postgres.GetEvalCase(ctx, q, row.EvidenceID)
	if err != nil {
		return evalCaseFile{}, fmt.Errorf("buscar gabarito: %w", err)
	}
	if evalCase == nil {
		return evalCaseFile{}, fmt.Errorf("caso listado para export sem gabarito gravado (inconsistência)")
	}
	expected := make([]evalCaseExpectedScore, 0, len(evalCase.Scores))
	for _, s := range evalCase.Scores {
		c, ok := byID[s.CompetencyID]
		if !ok {
			continue
		}
		expected = append(expected, evalCaseExpectedScore{CompetencyKey: c.PackKey, Level: s.Level})
	}

	kind := domain.EvidenceKind(row.Kind)
	isTranscript := kind == domain.EvidenceAudioRecording || kind == domain.EvidenceShadowingClip

	return evalCaseFile{
		ID:       row.EvidenceID,
		Reason:   row.Note,
		MarkedAt: row.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		Input: evalCaseInput{
			Pack:         evalCasePack{ID: pack.ID, Version: pack.Version, Label: pack.Label},
			Competencies: compViews,
			Action:       action,
			Evidence: evalCaseEvidence{
				Kind: row.Kind, Title: derefStr(row.Title), Body: derefStr(row.Body),
				ExternalURL: derefStr(row.ExternalURL), IsTranscript: isTranscript,
			},
		},
		Expected: evalCaseExpected{Scores: expected},
	}, nil
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// diacriticsReplacer troca os acentos comuns em português antes do regex de
// slug (que só reconhece a-z0-9) — sem isso, nota em pt-BR vira slug quase
// vazio, já que quase toda palavra tem uma vogal acentuada.
var diacriticsReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "õ", "o", "ô", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

// slugify usa a nota como base — é a coisa mais descritiva que temos sobre
// por que o caso importa.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = diacriticsReplacer.Replace(s)
	words := strings.Fields(s)
	if len(words) > 6 {
		words = words[:6]
	}
	s = strings.Join(words, "-")
	s = nonSlugChars.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "caso"
	}
	return s
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// writeCases substitui o conteúdo do diretório inteiro — export é
// idempotente e reflete sempre o estado atual do que está marcado, sem
// deixar arquivo órfão de um caso que foi desmarcado.
func writeCases(outDir string, cases []evalCaseFile) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("criar diretório de saída: %w", err)
	}
	existing, err := filepath.Glob(filepath.Join(outDir, "*.json"))
	if err != nil {
		return fmt.Errorf("listar arquivos existentes: %w", err)
	}
	for _, f := range existing {
		if err := os.Remove(f); err != nil {
			return fmt.Errorf("remover %s: %w", f, err)
		}
	}

	used := make(map[string]int)
	names := make([]string, len(cases))
	for i, c := range cases {
		base := slugify(c.Reason)
		used[base]++
		slug := base
		if used[base] > 1 {
			slug = fmt.Sprintf("%s-%d", base, used[base])
		}
		names[i] = fmt.Sprintf("%03d-%s.json", i+1, slug)
	}

	for i, c := range cases {
		data, err := json.MarshalIndent(c, "", "  ")
		if err != nil {
			return fmt.Errorf("serializar caso %s: %w", c.ID, err)
		}
		path := filepath.Join(outDir, names[i])
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("escrever %s: %w", path, err)
		}
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "eval-export: "+err.Error())
	os.Exit(1)
}
