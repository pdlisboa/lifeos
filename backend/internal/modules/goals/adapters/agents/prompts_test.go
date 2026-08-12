package agents

import (
	"strings"
	"testing"
)

func TestSplitPromptSections(t *testing.T) {
	raw := "---\nid: x\n---\n\n# Sistema\n\nsystem body\n\n---\n\n# Usuário\n\n{{.Goal.Title}}\n"
	system, user, err := splitPromptSections(raw)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !strings.Contains(system, "system body") {
		t.Errorf("system não contém o corpo esperado: %q", system)
	}
	if !strings.Contains(user, "{{.Goal.Title}}") {
		t.Errorf("user não contém o template esperado: %q", user)
	}
}

func TestSplitPromptSections_WrongDelimiterCount(t *testing.T) {
	_, _, err := splitPromptSections("sem delimitador nenhum")
	if err == nil {
		t.Fatal("esperava erro para arquivo sem os 3 delimitadores")
	}
}

func TestLoadPrompt_RealFiles(t *testing.T) {
	for _, f := range []string{"planner.v1.md", "practice.v1.md"} {
		p, err := loadPrompt(f)
		if err != nil {
			t.Fatalf("loadPrompt(%s): %v", f, err)
		}
		if p.System == "" {
			t.Errorf("%s: system prompt vazio", f)
		}
		if p.UserTmpl == nil {
			t.Errorf("%s: template do usuário não parseado", f)
		}
	}
}

func TestLoadSchema_RealFiles(t *testing.T) {
	for _, f := range []string{"planner.schema.json", "practice.schema.json"} {
		s, err := loadSchema(f)
		if err != nil {
			t.Fatalf("loadSchema(%s): %v", f, err)
		}
		if len(s) == 0 {
			t.Errorf("%s: schema vazio", f)
		}
	}
}
