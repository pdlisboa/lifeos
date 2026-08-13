package domain

import (
	"fmt"
	"strings"
)

// EvalCaseScore é o gabarito humano para uma competência: o nível que você
// daria, não o que um agente propôs (04-agentes.md §6.1 — o A3 ainda não
// existe; ele é quem esse conjunto vai avaliar).
type EvalCaseScore struct {
	CompetencyID string
	Level        int
}

// EvalCase marca uma evidência como caso de eval (04-agentes.md §6.1): nota
// livre explicando por que o caso importa, mais o gabarito por competência.
// Não é imutável como Evidence — é curadoria sua sobre a evidência, não a
// evidência em si, então pode ser corrigida com uma nova chamada.
type EvalCase struct {
	EvidenceID string
	Note       string
	Scores     []EvalCaseScore
}

func NewEvalCase(evidenceID, note string, scores []EvalCaseScore) (*EvalCase, error) {
	note = strings.TrimSpace(note)
	if note == "" {
		return nil, newRuleError("", "conte por que esse caso importa antes de marcar")
	}
	if len(scores) == 0 {
		return nil, newRuleError("", "marque o nível de pelo menos uma competência")
	}
	seen := make(map[string]bool, len(scores))
	for _, s := range scores {
		if s.Level < MinLevel || s.Level > MaxLevel {
			return nil, newRuleError("", fmt.Sprintf("nível deve estar entre %d e %d", MinLevel, MaxLevel))
		}
		if seen[s.CompetencyID] {
			return nil, newRuleError("", "competência duplicada no gabarito")
		}
		seen[s.CompetencyID] = true
	}
	return &EvalCase{EvidenceID: evidenceID, Note: note, Scores: scores}, nil
}
