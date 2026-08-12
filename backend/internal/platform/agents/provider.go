package agents

import (
	"context"
	"encoding/json"
)

// ProviderRequest é o que o Gateway manda pro provider — já com o modelo
// concreto resolvido (o provider nunca decide tier).
type ProviderRequest struct {
	Model        string
	SystemPrompt string
	UserPrompt   string
	SchemaName   string
	Schema       json.RawMessage
}

type ProviderResponse struct {
	Content      string // JSON cru devolvido pelo modelo — o Gateway valida
	InputTokens  int
	OutputTokens int
}

// Provider fala o protocolo POST /chat/completions compatível com OpenAI
// (01-arquitetura.md §6.1.1). A interface nasce no consumidor (Gateway),
// como manda a convenção do projeto — hoje tem duas implementações
// plausíveis (OpenAIProvider e FakeProvider), que é exatamente o critério
// pra existir.
type Provider interface {
	Complete(ctx context.Context, req ProviderRequest) (ProviderResponse, error)
}
