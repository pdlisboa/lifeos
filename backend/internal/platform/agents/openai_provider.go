package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIProvider fala POST {baseURL}/chat/completions no formato OpenAI
// (01-arquitetura.md §6.1.1): response_format json_schema com strict:true.
// `strict` é enviado como otimização — quando o provedor honra, economiza
// uma retentativa — mas a validação real é sempre do Gateway (§4.8).
type OpenAIProvider struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewOpenAIProvider(baseURL, apiKey string, timeout time.Duration) *OpenAIProvider {
	return &OpenAIProvider{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: timeout},
	}
}

type chatCompletionRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	ResponseFormat responseFormat `json:"response_format"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type       string         `json:"type"`
	JSONSchema jsonSchemaSpec `json:"json_schema"`
}

type jsonSchemaSpec struct {
	Name   string          `json:"name"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (p *OpenAIProvider) Complete(ctx context.Context, req ProviderRequest) (ProviderResponse, error) {
	body := chatCompletionRequest{
		Model: req.Model,
		Messages: []chatMessage{
			{Role: "system", Content: req.SystemPrompt},
			{Role: "user", Content: req.UserPrompt},
		},
		ResponseFormat: responseFormat{
			Type: "json_schema",
			JSONSchema: jsonSchemaSpec{
				Name:   req.SchemaName,
				Strict: true,
				Schema: req.Schema,
			},
		},
	}

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		resp, err := p.doRequest(ctx, body)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return ProviderResponse{}, fmt.Errorf("agents: chamar provedor após retry: %w", lastErr)
}

func (p *OpenAIProvider) doRequest(ctx context.Context, body chatCompletionRequest) (ProviderResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return ProviderResponse{}, fmt.Errorf("serializar requisição: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return ProviderResponse{}, fmt.Errorf("montar requisição: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return ProviderResponse{}, fmt.Errorf("requisição HTTP: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return ProviderResponse{}, fmt.Errorf("ler corpo da resposta: %w", err)
	}

	if httpResp.StatusCode >= 300 {
		return ProviderResponse{}, fmt.Errorf("provedor respondeu %d: %s", httpResp.StatusCode, string(respBody))
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return ProviderResponse{}, fmt.Errorf("parsear resposta: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return ProviderResponse{}, fmt.Errorf("resposta sem choices")
	}

	return ProviderResponse{
		Content:      parsed.Choices[0].Message.Content,
		InputTokens:  parsed.Usage.PromptTokens,
		OutputTokens: parsed.Usage.CompletionTokens,
	}, nil
}
