package agents

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOpenAIProvider_CompleteParsesSuccessResponse(t *testing.T) {
	var gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{"message": {"content": "{\"title\":\"ok\",\"estimatedMin\":10}"}}],
			"usage": {"prompt_tokens": 42, "completion_tokens": 7}
		}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "sk-teste", 5*time.Second)
	resp, err := p.Complete(context.Background(), ProviderRequest{
		Model:        "algum-modelo",
		SystemPrompt: "sistema",
		UserPrompt:   "usuário",
		SchemaName:   "practice",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Content != `{"title":"ok","estimatedMin":10}` {
		t.Fatalf("content inesperado: %s", resp.Content)
	}
	if resp.InputTokens != 42 || resp.OutputTokens != 7 {
		t.Fatalf("tokens inesperados: in=%d out=%d", resp.InputTokens, resp.OutputTokens)
	}
	if gotAuth != "Bearer sk-teste" {
		t.Fatalf("Authorization header inesperado: %s", gotAuth)
	}
	if !strings.Contains(gotBody, `"strict":true`) {
		t.Fatalf("corpo da requisição deveria mandar strict:true, teve: %s", gotBody)
	}
	if !strings.Contains(gotBody, `"type":"json_schema"`) {
		t.Fatalf("corpo da requisição deveria mandar response_format json_schema, teve: %s", gotBody)
	}
}

func TestOpenAIProvider_RetriesOnceOnServerError(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"boom"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{"message": {"content": "{\"ok\":true}"}}],
			"usage": {"prompt_tokens": 1, "completion_tokens": 1}
		}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "sk-teste", 5*time.Second)
	resp, err := p.Complete(context.Background(), ProviderRequest{Model: "m", Schema: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatalf("Complete deveria se recuperar no retry: %v", err)
	}
	if resp.Content != `{"ok":true}` {
		t.Fatalf("content inesperado: %s", resp.Content)
	}
	if attempts != 2 {
		t.Fatalf("esperava 2 tentativas, teve %d", attempts)
	}
}

func TestOpenAIProvider_FailsAfterTwoServerErrors(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "sk-teste", 5*time.Second)
	_, err := p.Complete(context.Background(), ProviderRequest{Model: "m", Schema: json.RawMessage(`{}`)})
	if err == nil {
		t.Fatal("esperava erro após esgotar as retentativas")
	}
	if attempts != 2 {
		t.Fatalf("esperava exatamente 2 tentativas, teve %d", attempts)
	}
}

func TestOpenAIProvider_FailsOnEmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices": [], "usage": {}}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "sk-teste", 5*time.Second)
	_, err := p.Complete(context.Background(), ProviderRequest{Model: "m", Schema: json.RawMessage(`{}`)})
	if err == nil {
		t.Fatal("esperava erro para resposta sem choices")
	}
}
