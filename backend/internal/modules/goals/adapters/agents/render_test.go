package agents

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/phablo/lifeos/internal/modules/goals/domain"
	"github.com/phablo/lifeos/internal/modules/goals/packs"
)

func minimalGoal(t *testing.T) *domain.Goal {
	t.Helper()
	dod := "sinto confiança pra abrir uma PR não-trivial num projeto Go real"
	g, err := domain.NewGoal("goal-1", "user-1", "Aprender Go", domain.ArchetypeSkill, "golang", nil, time.Now())
	if err != nil {
		t.Fatalf("NewGoal: %v", err)
	}
	g.DefinitionOfDone = &dod
	return g
}

func loadGolangPack(t *testing.T) packs.Pack {
	t.Helper()
	reg, err := packs.Load()
	if err != nil {
		t.Fatalf("carregar packs: %v", err)
	}
	p, ok := reg.Get("golang")
	if !ok {
		t.Fatal("pack golang não encontrado")
	}
	return p
}

func TestBuildAndRenderPlannerContext(t *testing.T) {
	pack := loadGolangPack(t)
	now := time.Now()
	why := "quero trocar de carreira pra backend"
	dod := "sinto confiança pra abrir uma PR não-trivial num projeto Go real"
	g, err := domain.NewGoal("goal-1", "user-1", "Aprender Go", domain.ArchetypeSkill, "golang", &why, now)
	if err != nil {
		t.Fatalf("NewGoal: %v", err)
	}
	g.DefinitionOfDone = &dod

	level2 := 2
	comps := []domain.Competency{
		{PackKey: "concurrency", Label: "Concorrência", Weight: 0.25, CurrentLevel: &level2, Confidence: domain.ConfidenceLow},
		{PackKey: "testing", Label: "Testes", Weight: 0.15, Confidence: domain.ConfidenceUnknown},
	}

	ctx := BuildPlannerContext(g, pack, comps, nil, "")
	if ctx.UnknownCount != 1 {
		t.Errorf("UnknownCount = %d, esperava 1", ctx.UnknownCount)
	}

	var buf bytes.Buffer
	if err := plannerPrompt.UserTmpl.Execute(&buf, ctx); err != nil {
		t.Fatalf("executar template do planejador: %v", err)
	}
	rendered := buf.String()

	for _, want := range []string{"Aprender Go", dod, why, "desconhecido"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("prompt renderizado não contém %q\n---\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "<no value>") {
		t.Errorf("prompt renderizado tem placeholder de template não resolvido:\n%s", rendered)
	}
}

func TestBuildAndRenderPracticeContext(t *testing.T) {
	pack := loadGolangPack(t)
	now := time.Now()
	dod := "sinto confiança pra abrir uma PR não-trivial num projeto Go real"
	g, err := domain.NewGoal("goal-1", "user-1", "Aprender Go", domain.ArchetypeSkill, "golang", nil, now)
	if err != nil {
		t.Fatalf("NewGoal: %v", err)
	}
	g.DefinitionOfDone = &dod

	level2 := 2
	comps := []domain.Competency{
		{ID: "comp-1", PackKey: "concurrency", Label: "Concorrência", Weight: 0.25, CurrentLevel: &level2, Confidence: domain.ConfidenceLow},
	}
	milestone := &domain.Milestone{
		Title:              "Escreve um worker pool básico",
		CompletionCriteria: "worker pool com channels e WaitGroup, sem vazar goroutine",
		CompetencyIDs:      []string{"comp-1"},
	}
	recent := []RecentActionView{
		{Title: "Ler sobre context.Context", Status: "skipped", SkipReason: "too_hard"},
	}

	ctx := BuildPracticeContext(g, pack, milestone, comps, recent, "easier", "practice")

	var buf bytes.Buffer
	if err := practicePrompt.UserTmpl.Execute(&buf, ctx); err != nil {
		t.Fatalf("executar template de prática: %v", err)
	}
	rendered := buf.String()

	for _, want := range []string{"Aprender Go", dod, milestone.Title, "easier", "practice"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("prompt renderizado não contém %q\n---\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "<no value>") {
		t.Errorf("prompt renderizado tem placeholder de template não resolvido:\n%s", rendered)
	}
}
