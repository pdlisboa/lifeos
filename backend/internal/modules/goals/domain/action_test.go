package domain

import (
	"testing"
	"time"
)

func TestNewNextActionEnforcesP4(t *testing.T) {
	now := time.Now()
	minimal := "Versão mínima: releia o enunciado por 5 minutos."

	cases := []struct {
		name         string
		estimatedMin int
		minimal      string
		wantErr      bool
		wantRule     string
	}{
		{"válido", 20, minimal, false, ""},
		{"curto demais", 4, minimal, true, "P4"},
		{"longo demais", 31, minimal, true, "P4"},
		{"minimalVariant curto demais", 20, "muito curto", true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, err := NewNextAction("a1", "g1", "u1", "Fazer algo", tc.estimatedMin, tc.minimal, GeneratedByFallback, now)
			if (err != nil) != tc.wantErr {
				t.Fatalf("erro = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr && tc.wantRule != "" {
				re, ok := err.(*RuleError)
				if !ok || re.Rule != tc.wantRule {
					t.Fatalf("esperava RuleError Rule=%s, got %v", tc.wantRule, err)
				}
			}
			if !tc.wantErr && a.Status != ActionPending {
				t.Fatalf("ação nova deveria nascer pending, got %s", a.Status)
			}
		})
	}
}

func TestNextActionCompleteAndSkipAreTerminal(t *testing.T) {
	now := time.Now()
	a, err := NewNextAction("a1", "g1", "u1", "Fazer algo", 20, "Versão mínima de 5 minutos aqui.", GeneratedByFallback, now)
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}

	if err := a.Complete(now); err != nil {
		t.Fatalf("Complete falhou: %v", err)
	}
	if a.Status != ActionCompleted || a.ResolvedAt == nil {
		t.Fatal("ação deveria estar completed com ResolvedAt preenchido")
	}
	if err := a.Complete(now); err == nil {
		t.Fatal("completar uma ação já resolvida deveria falhar")
	}
}

func TestNextActionSkipRequiresValidReason(t *testing.T) {
	now := time.Now()
	a, _ := NewNextAction("a1", "g1", "u1", "Fazer algo", 20, "Versão mínima de 5 minutos aqui.", GeneratedByFallback, now)

	if err := a.Skip(SkipReason("bogus"), now); err == nil {
		t.Fatal("motivo inválido deveria falhar")
	}
	if err := a.Skip(SkipTooHard, now); err != nil {
		t.Fatalf("skip válido falhou: %v", err)
	}
	if a.Status != ActionSkipped || a.SkipReason == nil || *a.SkipReason != SkipTooHard {
		t.Fatal("ação deveria estar skipped com o motivo gravado")
	}
}

func TestNextDifficultyHint(t *testing.T) {
	if got := NextDifficultyHint(SkipTooHard); got != "easier" {
		t.Fatalf("got %s, want easier", got)
	}
	if got := NextDifficultyHint(SkipTooEasy); got != "harder" {
		t.Fatalf("got %s, want harder", got)
	}
	if got := NextDifficultyHint(SkipNoTime); got != "same" {
		t.Fatalf("got %s, want same", got)
	}
}
