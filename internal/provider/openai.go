package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mindbalancer/mindbalancer/api/openai"
	"github.com/mindbalancer/mindbalancer/internal/storage"
)

// OpenAI implements the Provider interface for OpenAI.
type OpenAI struct {
	*BaseProvider
}

// NewOpenAI creates a new OpenAI provider.
func NewOpenAI(server storage.Server, timeout time.Duration) *OpenAI {
	return &OpenAI{
		BaseProvider: NewBaseProvider(server, timeout),
	}
}

func (p *OpenAI) Name() string {
	return "openai"
}

func (p *OpenAI) authHeaders() map[string]string {
	headers := make(map[string]string)
	if p.Server.APIKeyEncrypted != "" {
		headers["Authorization"] = "Bearer " + p.Server.APIKeyEncrypted
	}
	return headers
}

func (p *OpenAI) ChatCompletion(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	resp, err := p.doRequest(ctx, "POST", "/v1/chat/completions", req, p.authHeaders())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var result openai.ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (p *OpenAI) ChatCompletionStream(ctx context.Context, req *openai.ChatCompletionRequest) (<-chan StreamEvent, error) {
	req.Stream = true

	headers := p.authHeaders()
	headers["Accept"] = "text/event-stream"

	resp, err := p.doRequest(ctx, "POST", "/v1/chat/completions", req, headers)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, p.parseError(resp)
	}

	ch := make(chan StreamEvent, 100)
	go p.readSSE(resp.Body, ch)
	return ch, nil
}

func (p *OpenAI) readSSE(body io.ReadCloser, ch chan<- StreamEvent) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			if data == "[DONE]" {
				ch <- StreamEvent{Done: true}
				return
			}

			ch <- StreamEvent{Data: []byte(data)}
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- StreamEvent{Error: err}
	}
}

func (p *OpenAI) Completion(ctx context.Context, req *openai.CompletionRequest) (*openai.CompletionResponse, error) {
	resp, err := p.doRequest(ctx, "POST", "/v1/completions", req, p.authHeaders())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var result openai.CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (p *OpenAI) Embedding(ctx context.Context, req *openai.EmbeddingRequest) (*openai.EmbeddingResponse, error) {
	resp, err := p.doRequest(ctx, "POST", "/v1/embeddings", req, p.authHeaders())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var result openai.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (p *OpenAI) ListModels(ctx context.Context) (*openai.ModelList, error) {
	resp, err := p.doRequest(ctx, "GET", "/v1/models", nil, p.authHeaders())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var result openai.ModelList
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (p *OpenAI) SupportsStreaming() bool {
	return true
}

func (p *OpenAI) SupportsEmbeddings() bool {
	return true
}
