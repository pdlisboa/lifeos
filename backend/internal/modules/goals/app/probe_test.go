package app

import (
	"context"
	"errors"
	"testing"

	"github.com/phablo/lifeos/internal/modules/goals/adapters/postgres"
	"github.com/phablo/lifeos/internal/modules/goals/domain"
)

// TestAnswerProbeWalksStaticSeeds cobre o percorredor estático de
// 04-agentes.md §4.7: sem LLM, segue os probe_seeds do pack golang na ordem,
// usando o follow-up de sim/não quando a heurística léxica reconhece a
// resposta, e fecha a sondagem quando as sementes acabam.
func TestAnswerProbeWalksStaticSeeds(t *testing.T) {
	svc, userID := newFixture(t)
	created := createDraftGoal(t, svc, userID, "Meta com sondagem")

	probe, err := svc.GetProbe(context.Background(), userID, created.Goal.ID)
	if err != nil {
		t.Fatalf("GetProbe falhou: %v", err)
	}
	if len(probe.Turns) != 1 {
		t.Fatalf("esperava 1 turno inicial (seed de concurrency), got %d", len(probe.Turns))
	}
	firstTurn := probe.Turns[0]

	// responde "sim" -> dispara o follow-up de concurrency.
	step1, err := svc.AnswerProbe(context.Background(), AnswerProbeInput{
		UserID: userID, GoalID: created.Goal.ID, TurnID: firstTurn.ID, Answer: "sim",
	})
	if err != nil {
		t.Fatalf("AnswerProbe (1) falhou: %v", err)
	}
	if step1.NextQuestion == nil {
		t.Fatal("esperava follow-up de concurrency após responder sim")
	}
	if step1.Probe.Status != domain.ProbeOpen {
		t.Fatalf("status = %s, want open", step1.Probe.Status)
	}

	// responde o follow-up -> avança para a seed de testing (sem follow-up).
	step2, err := svc.AnswerProbe(context.Background(), AnswerProbeInput{
		UserID: userID, GoalID: created.Goal.ID, TurnID: step1.NextQuestion.ID, Answer: "consigo sim",
	})
	if err != nil {
		t.Fatalf("AnswerProbe (2) falhou: %v", err)
	}
	if step2.NextQuestion == nil {
		t.Fatal("esperava a próxima seed (testing)")
	}

	// responde testing -> avança para interfaces_design.
	step3, err := svc.AnswerProbe(context.Background(), AnswerProbeInput{
		UserID: userID, GoalID: created.Goal.ID, TurnID: step2.NextQuestion.ID, Answer: "às vezes",
	})
	if err != nil {
		t.Fatalf("AnswerProbe (3) falhou: %v", err)
	}
	if step3.NextQuestion == nil {
		t.Fatal("esperava a última seed (interfaces_design)")
	}

	// responde a última seed -> sementes esgotadas, sondagem fecha sozinha.
	step4, err := svc.AnswerProbe(context.Background(), AnswerProbeInput{
		UserID: userID, GoalID: created.Goal.ID, TurnID: step3.NextQuestion.ID, Answer: "junto da implementação",
	})
	if err != nil {
		t.Fatalf("AnswerProbe (4) falhou: %v", err)
	}
	if step4.NextQuestion != nil {
		t.Fatal("sementes esgotadas: não deveria haver mais pergunta")
	}
	if step4.Probe.Status != domain.ProbeCompleted {
		t.Fatalf("status final = %s, want completed", step4.Probe.Status)
	}
}

func TestGetProbeGoalNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.GetProbe(context.Background(), userID, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}

func TestSkipProbeGoalNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.SkipProbe(context.Background(), userID, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}

func TestAnswerProbeGoalNotFound(t *testing.T) {
	svc, userID := newFixture(t)
	_, err := svc.AnswerProbe(context.Background(), AnswerProbeInput{
		UserID: userID, GoalID: "00000000-0000-0000-0000-000000000000", TurnID: "x", Answer: "sim",
	})
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, got %v", err)
	}
}

// TestAnswerProbeAfterSkipFails cobre o guard de estado em domain.Probe.Answer:
// uma sondagem já fechada (skip ou completed) não aceita mais respostas.
func TestAnswerProbeAfterSkipFails(t *testing.T) {
	svc, userID := newFixture(t)
	created := createDraftGoal(t, svc, userID, "Meta com sondagem")
	if _, err := svc.SkipProbe(context.Background(), userID, created.Goal.ID); err != nil {
		t.Fatalf("SkipProbe falhou: %v", err)
	}

	_, err := svc.AnswerProbe(context.Background(), AnswerProbeInput{
		UserID: userID, GoalID: created.Goal.ID, TurnID: "qualquer", Answer: "sim",
	})
	if err == nil {
		t.Fatal("responder uma sondagem já pulada deveria falhar")
	}
}

func TestAnswerProbeUnknownTurn(t *testing.T) {
	svc, userID := newFixture(t)
	created := createDraftGoal(t, svc, userID, "Meta com sondagem")

	_, err := svc.AnswerProbe(context.Background(), AnswerProbeInput{
		UserID: userID, GoalID: created.Goal.ID, TurnID: "00000000-0000-0000-0000-000000000000", Answer: "sim",
	})
	if err == nil {
		t.Fatal("esperava erro para turno inexistente")
	}
}

func TestSkipProbe(t *testing.T) {
	svc, userID := newFixture(t)
	created := createDraftGoal(t, svc, userID, "Meta com sondagem")

	probe, err := svc.SkipProbe(context.Background(), userID, created.Goal.ID)
	if err != nil {
		t.Fatalf("SkipProbe falhou: %v", err)
	}
	if probe.Status != domain.ProbeSkipped {
		t.Fatalf("status = %s, want skipped", probe.Status)
	}

	fetched, err := svc.GetProbe(context.Background(), userID, created.Goal.ID)
	if err != nil {
		t.Fatalf("GetProbe falhou: %v", err)
	}
	if fetched.Status != domain.ProbeSkipped {
		t.Fatalf("status persistido = %s, want skipped", fetched.Status)
	}
}
