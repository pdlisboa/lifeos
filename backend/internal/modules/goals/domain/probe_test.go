package domain

import (
	"testing"
	"time"
)

func TestProbeAskTurnRespectsCeiling(t *testing.T) {
	p := NewProbe("p1", "g1", "u1")
	for i := 0; i < MaxProbeQuestions; i++ {
		if err := p.AskTurn(ProbeTurn{ID: "t", Question: "q?"}); err != nil {
			t.Fatalf("pergunta %d deveria ser aceita: %v", i+1, err)
		}
	}
	if p.CanAskMore() {
		t.Fatal("não deveria poder perguntar além do teto de 5 (D2/§7.8)")
	}
	if err := p.AskTurn(ProbeTurn{ID: "extra", Question: "?"}); err == nil {
		t.Fatal("6ª pergunta deveria ser rejeitada")
	}
}

func TestProbeAnswerAndClose(t *testing.T) {
	p := NewProbe("p1", "g1", "u1")
	_ = p.AskTurn(ProbeTurn{ID: "t1", Question: "Já mexeu com goroutines?"})
	now := time.Now()

	turn, err := p.Answer("t1", "sim, bastante", now)
	if err != nil {
		t.Fatalf("Answer falhou: %v", err)
	}
	if turn.Answer == nil || *turn.Answer != "sim, bastante" {
		t.Fatalf("resposta não gravada corretamente: %+v", turn)
	}

	if _, err := p.Answer("t1", "de novo", now); err == nil {
		t.Fatal("responder a mesma pergunta duas vezes deveria falhar")
	}
	if _, err := p.Answer("nao-existe", "x", now); err == nil {
		t.Fatal("responder um turno inexistente deveria falhar")
	}

	if err := p.Close(ProbeSkipped, now); err != nil {
		t.Fatalf("Close falhou: %v", err)
	}
	if p.Status != ProbeSkipped || p.ClosedAt == nil {
		t.Fatal("probe deveria estar skipped com ClosedAt preenchido")
	}
	// pulável a qualquer momento, sem penalidade (§7.8) — Answer depois de
	// fechada deve recusar, não silenciosamente aceitar.
	_ = p.AskTurn(ProbeTurn{ID: "t2", Question: "outra?"})
	if p.CanAskMore() {
		t.Fatal("probe fechada não deveria aceitar mais perguntas")
	}
}

func TestAnsweredYesNoHeuristic(t *testing.T) {
	yes := []string{"sim", "Sim!", "sim, bastante", "yes", "y"}
	for _, a := range yes {
		if !AnsweredYes(a) {
			t.Errorf("AnsweredYes(%q) = false, want true", a)
		}
	}
	no := []string{"não", "nao", "Não, nunca", "no", "n"}
	for _, a := range no {
		if !AnsweredNo(a) {
			t.Errorf("AnsweredNo(%q) = false, want true", a)
		}
	}
	ambiguous := "mais ou menos, sempre esqueço o buffering"
	if AnsweredYes(ambiguous) || AnsweredNo(ambiguous) {
		t.Errorf("resposta ambígua %q não deveria bater com sim nem não", ambiguous)
	}
}
