package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	platformagents "github.com/phablo/lifeos/internal/platform/agents"
)

var (
	practicePrompt = mustLoadPrompt("practice.v1.md")
	practiceSchema = mustLoadSchema("practice.schema.json")
)

type PracticeGoalView struct {
	Title            string
	DefinitionOfDone string
	ScopeMode        string
}

type PracticeMilestoneView struct {
	Title              string
	CompletionCriteria string
	CompetencyKeys     string
}

type PracticeFormatView struct {
	Key     string
	Label   string
	Minutes [2]int
	GoodFor string
	Shape   string
}

type PracticePackView struct {
	Label           string
	PracticeFormats []PracticeFormatView
}

// RecentActionView é usado duas vezes no template (histórico com status, e
// "já feito" só com título) — BuildPracticeContext monta as duas listas a
// partir da mesma consulta.
type RecentActionView struct {
	Title      string
	Status     string
	SkipReason string
}

// PracticeContext alimenta practice.v1.md. Milestone nil == "sem marco
// definido ainda" (04-agentes.md §5, fallback também trata esse caso).
type PracticeContext struct {
	Goal           PracticeGoalView
	Milestone      *PracticeMilestoneView
	Competencies   []CompetencyView
	Pack           PracticePackView
	RecentActions  []RecentActionView
	DifficultyHint string
	OriginKind     string
	RecentTitles   []string
}

type PracticeExpectedEvidence struct {
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

// PracticeOutput espelha practice.schema.json — minimalVariant e
// expectedEvidence são obrigatórios lá por motivo de produto (04-agentes.md
// §4.2), refletido aqui como campos não-ponteiro.
type PracticeOutput struct {
	Title            string                   `json:"title"`
	Detail           string                   `json:"detail"`
	PracticeFormat   string                   `json:"practiceFormat"`
	EstimatedMin     int                      `json:"estimatedMin"`
	MinimalVariant   string                   `json:"minimalVariant"`
	CompetencyKey    string                   `json:"competencyKey"`
	ExpectedEvidence PracticeExpectedEvidence `json:"expectedEvidence"`
	MilestoneID      *string                  `json:"milestoneId"`
	Rationale        *string                  `json:"rationale"`
}

// GeneratePractice chama A2 (04-agentes.md §4.2): tier fast, roda toda vez
// que uma ação é consumida.
func GeneratePractice(ctx context.Context, gw *platformagents.Gateway, userID string, in PracticeContext) (*PracticeOutput, platformagents.Result, error) {
	var buf bytes.Buffer
	if err := practicePrompt.UserTmpl.Execute(&buf, in); err != nil {
		return nil, platformagents.Result{}, fmt.Errorf("agents: montar prompt de prática: %w", err)
	}

	result, err := gw.Run(ctx, platformagents.Request{
		Task:          "generate_action",
		Tier:          platformagents.TierFast,
		UserID:        userID,
		PromptVersion: "practice.v1",
		SystemPrompt:  practicePrompt.System,
		UserPrompt:    buf.String(),
		SchemaName:    "practice",
		Schema:        practiceSchema,
	})
	if err != nil {
		return nil, result, err
	}

	var out PracticeOutput
	if err := json.Unmarshal(result.Output, &out); err != nil {
		return nil, result, fmt.Errorf("agents: parsear saída da prática: %w", err)
	}
	return &out, result, nil
}
