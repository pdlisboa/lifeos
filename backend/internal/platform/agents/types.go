// Package agents é o Agent Gateway (01-arquitetura.md §6): fala com um
// provedor compatível com OpenAI, valida a saída contra o JSON Schema
// completo (a garantia real — §4.8 de 04-agentes.md diz que `strict` do
// provedor é otimização, não contrato), cacheia por hash de input, controla
// orçamento mensal com degradação de tier, e registra tudo em `agent_call`.
//
// Nenhum `cmd/` desta fatia instancia o Gateway — não há agente plugado
// ainda (pedido explícito). O pacote nasce pronto e testado (com
// FakeProvider, sem rede) para quando A1/A2 forem plugados.
package agents

import (
	"encoding/json"
	"errors"
)

type Tier string

const (
	TierFast     Tier = "fast"
	TierBalanced Tier = "balanced"
	TierStrong   Tier = "strong"
)

// Request é o que um módulo de domínio pede ao gateway. Nunca nomeia um
// modelo — só um Tier; o mapeamento pra modelo concreto vem de fora
// (01-arquitetura.md §6.2).
type Request struct {
	Task          string // 'assess','plan_track','generate_action','curate','coach','probe'
	Tier          Tier
	UserID        string
	PromptVersion string // 'assessor.v1'
	SystemPrompt  string
	UserPrompt    string
	SchemaName    string
	Schema        json.RawMessage // JSON Schema cru, validado por completo (§4.8)
}

type Result struct {
	Output        json.RawMessage
	Model         string
	Tier          Tier // tier efetivamente usado — pode ter sido rebaixado por orçamento
	InputTokens   int
	OutputTokens  int
	CostUSD       float64
	LatencyMs     int64
	CacheHit      bool
	PromptVersion string
	// CallID é o id da linha de agent_call gravada por esta execução — o que
	// permite quem chama (ex.: proposal.agent_call_id) rastrear até a
	// chamada real (04-agentes.md §9, migration 0010).
	CallID string
}

var (
	// ErrBudgetExhausted: teto mensal batido e o tier pedido não é fast
	// (01-arquitetura.md §6.3).
	ErrBudgetExhausted = errors.New("agents: orçamento mensal esgotado, apenas tier fast disponível")
	// ErrInvalidOutput: saída não passou no schema nem depois da
	// retentativa com o erro embutido no prompt (04-agentes.md §4.8).
	ErrInvalidOutput = errors.New("agents: saída do agente não passou na validação de schema após retentativa")
	// ErrProviderFailed: erro de rede/HTTP do provedor, mesmo após retry.
	ErrProviderFailed = errors.New("agents: provedor de LLM falhou")
)
