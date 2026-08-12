package agents

import (
	"context"
	"fmt"
	"sync"
)

// FakeProvider devolve respostas gravadas, em ordem, sem rede — pedido
// explícito para o gateway inteiro (cache, orçamento, validação, retry) ser
// testável sem depender de um LLM de verdade.
type FakeProvider struct {
	mu        sync.Mutex
	responses []FakeResponse
	calls     []ProviderRequest
}

type FakeResponse struct {
	Content      string
	InputTokens  int
	OutputTokens int
	Err          error
}

// NewFakeProvider recebe as respostas na ordem em que serão consumidas, uma
// por chamada a Complete.
func NewFakeProvider(responses ...FakeResponse) *FakeProvider {
	return &FakeProvider{responses: responses}
}

func (f *FakeProvider) Complete(ctx context.Context, req ProviderRequest) (ProviderResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, req)
	if len(f.responses) == 0 {
		return ProviderResponse{}, fmt.Errorf("fake provider: sem respostas gravadas restantes")
	}
	next := f.responses[0]
	f.responses = f.responses[1:]
	if next.Err != nil {
		return ProviderResponse{}, next.Err
	}
	return ProviderResponse{
		Content:      next.Content,
		InputTokens:  next.InputTokens,
		OutputTokens: next.OutputTokens,
	}, nil
}

// Calls devolve uma cópia de todas as requisições recebidas — usado nos
// testes do Gateway pra provar que o cache/orçamento se comportam sem
// precisar inspecionar rede nenhuma.
func (f *FakeProvider) Calls() []ProviderRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]ProviderRequest, len(f.calls))
	copy(out, f.calls)
	return out
}
