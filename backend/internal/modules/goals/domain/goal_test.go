package domain

import (
	"strings"
	"testing"
	"time"
)

func mustGoal(t *testing.T) *Goal {
	t.Helper()
	g, err := NewGoal("g1", "u1", "Aprender Go", ArchetypeSkill, "golang", nil, time.Now())
	if err != nil {
		t.Fatalf("NewGoal falhou: %v", err)
	}
	return g
}

func TestNewGoalValidation(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name      string
		title     string
		archetype GoalArchetype
		packID    string
		wantErr   bool
	}{
		{"válido", "Aprender Go", ArchetypeSkill, "golang", false},
		{"título curto demais", "Go", ArchetypeSkill, "golang", true},
		{"arquétipo inválido", "Aprender Go", GoalArchetype("nonsense"), "golang", true},
		{"pack vazio", "Aprender Go", ArchetypeSkill, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewGoal("g1", "u1", tc.title, tc.archetype, tc.packID, nil, now)
			if (err != nil) != tc.wantErr {
				t.Fatalf("erro = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestGoalBlockersRN01(t *testing.T) {
	g := mustGoal(t)

	blockers := g.Blockers(0)
	if len(blockers) != 2 {
		t.Fatalf("esperava 2 blockers (dod + competências), got %v", blockers)
	}

	dod := "Escrever e testar um serviço HTTP concorrente sem tutorial"
	g.DefinitionOfDone = &dod
	blockers = g.Blockers(2)
	if len(blockers) != 1 || !strings.Contains(blockers[0], "competências") {
		t.Fatalf("esperava só o blocker de competências, got %v", blockers)
	}

	if !g.ReadyToActivate(3) {
		t.Fatal("meta com dod e 3 competências deveria estar pronta para ativar")
	}
}

func TestGoalActivateRequiresRN01(t *testing.T) {
	g := mustGoal(t)
	now := time.Now()

	err := g.Activate(3, now)
	if err == nil {
		t.Fatal("ativar sem definição de pronto deveria falhar (RN-01)")
	}
	ruleErr, ok := err.(*RuleError)
	if !ok || ruleErr.Rule != "RN-01" {
		t.Fatalf("esperava RuleError com Rule=RN-01, got %v", err)
	}

	dod := "critério observável"
	g.DefinitionOfDone = &dod
	if err := g.Activate(3, now); err != nil {
		t.Fatalf("ativar com RN-01 satisfeita deveria funcionar: %v", err)
	}
	if g.Status != GoalActive {
		t.Fatalf("status = %s, want active", g.Status)
	}
	if g.ActivatedAt == nil || g.LastActivityAt == nil {
		t.Fatal("ActivatedAt e LastActivityAt deveriam estar preenchidos")
	}
}

func TestGoalLifecycleTransitions(t *testing.T) {
	now := time.Now()
	cases := []struct {
		from GoalStatus
		to   GoalStatus
		ok   bool
	}{
		{GoalDraft, GoalActive, true},
		{GoalDraft, GoalPaused, false},
		{GoalActive, GoalAtRisk, true},
		{GoalActive, GoalCompleted, true},
		{GoalAtRisk, GoalStagnant, true},
		{GoalStagnant, GoalActive, true},
		{GoalPaused, GoalActive, true},
		{GoalPaused, GoalCompleted, false},
		{GoalCompleted, GoalActive, false},
		{GoalAbandoned, GoalActive, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			g := &Goal{Status: tc.from}
			err := g.Transition(tc.to, now)
			if (err == nil) != tc.ok {
				t.Fatalf("Transition(%s->%s) err=%v, want ok=%v", tc.from, tc.to, err, tc.ok)
			}
			if tc.ok && g.Status != tc.to {
				t.Fatalf("status final = %s, want %s", g.Status, tc.to)
			}
		})
	}
}

func TestGoalCloseRequiresDebriefRN10(t *testing.T) {
	g := &Goal{Status: GoalActive}
	now := time.Now()

	if err := g.Close(GoalCompleted, false, now); err == nil {
		t.Fatal("fechar sem debrief deveria falhar (RN-10)")
	}
	if err := g.Close(GoalCompleted, true, now); err != nil {
		t.Fatalf("fechar com debrief deveria funcionar: %v", err)
	}
	if g.Status != GoalCompleted || g.ClosedAt == nil {
		t.Fatal("meta deveria estar completed com ClosedAt preenchido")
	}
}
