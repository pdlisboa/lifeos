package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	platformagents "github.com/phablo/lifeos/internal/platform/agents"
)

var (
	plannerPrompt = mustLoadPrompt("planner.v1.md")
	plannerSchema = mustLoadSchema("planner.schema.json")
)

// CompetencyView é a visão de competência que os prompts de A1 e A2
// compartilham. Level é ponteiro de propósito: `{{if .Level}}` no template
// só olha se o ponteiro é nil (text/template.IsTrue não desreferencia
// ponteiro pra checar o valor apontado), então nível 0 conhecido nunca vira
// "desconhecido" — a mesma regra de `level ?? 0` (CLAUDE.md, armadilhas).
type CompetencyView struct {
	PackKey         string
	Label           string
	Weight          float64
	Level           *int
	LevelDescriptor *string
	Confidence      string
}

type PlannerGoalView struct {
	Title            string
	Why              string
	DefinitionOfDone string
}

type PlannerMilestoneLibraryItem struct {
	Title        string
	TypicalLevel int
	Competencies string
	Criteria     string
}

type PlannerPackView struct {
	Label            string
	MilestoneLibrary []PlannerMilestoneLibraryItem
}

type PlannerExistingMilestone struct {
	Ordinal            int
	Status             string
	Title              string
	CompletionCriteria string
}

type PlannerExistingTrack struct {
	Milestones []PlannerExistingMilestone
}

// PlannerContext alimenta planner.v1.md. ExistingTrack nil = criação;
// não-nil = revisão (04-agentes.md §4.1, "sobre revisão de trilha").
type PlannerContext struct {
	Goal           PlannerGoalView
	Pack           PlannerPackView
	Competencies   []CompetencyView
	UnknownCount   int
	ExistingTrack  *PlannerExistingTrack
	RevisionReason string
}

type PlannerMilestoneOutput struct {
	Ordinal            int      `json:"ordinal"`
	Title              string   `json:"title"`
	CompletionCriteria string   `json:"completionCriteria"`
	CompetencyKeys     []string `json:"competencyKeys"`
	CarriedOver        bool     `json:"carriedOver"`
	SourceLibraryTitle *string  `json:"sourceLibraryTitle"`
}

// PlannerOutput espelha planner.schema.json. NoChangeNeeded true é resposta
// válida numa revisão (04-agentes.md §4.1) — quem chama não deve criar
// proposta nesse caso.
type PlannerOutput struct {
	Milestones     []PlannerMilestoneOutput `json:"milestones"`
	Rationale      string                   `json:"rationale"`
	NoChangeNeeded bool                     `json:"noChangeNeeded"`
}

// GeneratePlan chama A1 (04-agentes.md §4.1): tier strong, roda pouco e
// produz sempre uma proposta — nunca escreve trilha direto (RN-07).
func GeneratePlan(ctx context.Context, gw *platformagents.Gateway, userID string, in PlannerContext) (*PlannerOutput, platformagents.Result, error) {
	var buf bytes.Buffer
	if err := plannerPrompt.UserTmpl.Execute(&buf, in); err != nil {
		return nil, platformagents.Result{}, fmt.Errorf("agents: montar prompt do planejador: %w", err)
	}

	result, err := gw.Run(ctx, platformagents.Request{
		Task:          "plan_track",
		Tier:          platformagents.TierStrong,
		UserID:        userID,
		PromptVersion: "planner.v1",
		SystemPrompt:  plannerPrompt.System,
		UserPrompt:    buf.String(),
		SchemaName:    "planner",
		Schema:        plannerSchema,
	})
	if err != nil {
		return nil, result, err
	}

	var out PlannerOutput
	if err := json.Unmarshal(result.Output, &out); err != nil {
		return nil, result, fmt.Errorf("agents: parsear saída do planejador: %w", err)
	}
	return &out, result, nil
}
