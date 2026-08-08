package domain

import (
	"strings"
	"time"
)

type ProbeStatus string

const (
	ProbeOpen      ProbeStatus = "open"
	ProbeCompleted ProbeStatus = "completed"
	ProbeSkipped   ProbeStatus = "skipped"
)

// MaxProbeQuestions é o teto de D2/§7.8: um agente entusiasmado — ou, na
// Fatia 1, o percorredor estático — não consegue virar a sondagem num
// questionário, porque o teto está no domínio (e também no banco: CHECK
// probe.asked_count <= 5).
const MaxProbeQuestions = 5

type ProbeTurn struct {
	ID            string
	ProbeID       string
	CompetencyID  *string
	Ordinal       int
	Question      string
	Answer        *string
	InferredLevel *int
	AnsweredAt    *time.Time
}

type Probe struct {
	ID         string
	GoalID     string
	UserID     string
	Status     ProbeStatus
	AskedCount int
	ClosedAt   *time.Time
	Turns      []ProbeTurn
}

func NewProbe(id, goalID, userID string) *Probe {
	return &Probe{ID: id, GoalID: goalID, UserID: userID, Status: ProbeOpen}
}

func (p *Probe) CanAskMore() bool {
	return p.Status == ProbeOpen && p.AskedCount < MaxProbeQuestions
}

// Current devolve o turno ainda sem resposta, se houver.
func (p *Probe) Current() *ProbeTurn {
	for i := range p.Turns {
		if p.Turns[i].Answer == nil {
			return &p.Turns[i]
		}
	}
	return nil
}

func (p *Probe) AskTurn(turn ProbeTurn) error {
	if !p.CanAskMore() {
		return newRuleError("", "sondagem já atingiu o teto de perguntas ou não está aberta")
	}
	turn.Ordinal = p.AskedCount + 1
	p.Turns = append(p.Turns, turn)
	p.AskedCount++
	return nil
}

func (p *Probe) Answer(turnID, answer string, now time.Time) (*ProbeTurn, error) {
	if p.Status != ProbeOpen {
		return nil, newRuleError("", "sondagem não está aberta")
	}
	for i := range p.Turns {
		if p.Turns[i].ID == turnID {
			if p.Turns[i].Answer != nil {
				return nil, newRuleError("", "esta pergunta já foi respondida")
			}
			a := strings.TrimSpace(answer)
			p.Turns[i].Answer = &a
			p.Turns[i].AnsweredAt = &now
			return &p.Turns[i], nil
		}
	}
	return nil, newRuleError("", "pergunta não encontrada nesta sondagem")
}

func (p *Probe) Close(status ProbeStatus, now time.Time) error {
	if status != ProbeCompleted && status != ProbeSkipped {
		return newRuleError("", "status de encerramento inválido")
	}
	p.Status = status
	p.ClosedAt = &now
	return nil
}

// AnsweredYes/AnsweredNo são a heurística léxica que o percorredor estático
// usa para escolher o follow-up de sim/não (04-agentes.md §4.7). É léxica de
// propósito — sem LLM não existe "entender" a resposta, só reconhecer um
// padrão simples; qualquer coisa mais esperta seria reinventar um agente.
func AnsweredYes(answer string) bool {
	switch normalizeYesNo(answer) {
	case "sim", "s", "yes", "y":
		return true
	}
	return false
}

func AnsweredNo(answer string) bool {
	switch normalizeYesNo(answer) {
	case "não", "nao", "n", "no":
		return true
	}
	return false
}

func normalizeYesNo(answer string) string {
	a := strings.ToLower(strings.TrimSpace(answer))
	if i := strings.IndexAny(a, " ,.;!"); i >= 0 {
		a = a[:i]
	}
	return a
}
