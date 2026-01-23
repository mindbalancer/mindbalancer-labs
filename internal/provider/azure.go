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

// Azure implements the Provider interface for Azure OpenAI.
type Azure struct {
	*BaseProvider
	apiVersion string
}

// NewAzure creates a new Azure provider.
func NewAzure(server storage.Server, timeout time.Duration) *Azure {
	return &Azure{
		BaseProvider: NewBaseProvider(server, timeout),
		apiVersion:   "2024-02-15-preview",
	}
}

func (p *Azure) Name() string {
	return "azure"
}

func (p *Azure) authHeaders() map[string]string {
	headers := make(map[string]string)
	if p.Server.APIKeyEncrypted != "" {
		headers["api-key"] = p.Server.APIKeyEncrypted
	}
	return headers
}

// buildURL builds Azure-specific URL with deployment name and API version.
func (p *Azure) buildURL(path, model string) string {
	// Azure URLs look like: {endpoint}/openai/deployments/{deployment-name}/chat/completions?api-version={version}
	// The model name is used as the deployment name
	endpoint := strings.TrimSuffix(p.Server.Endpoint, "/")

	switch {
	case strings.Contains(path, "chat/completions"):
		return fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s", endpoint, model, p.apiVersion)
	case strings.Contains(path, "completions"):
		return fmt.Sprintf("%s/openai/deployments/%s/completions?api-version=%s", endpoint, model, p.apiVersion)
	case strings.Contains(path, "embeddings"):
		return fmt.Sprintf("%s/openai/deployments/%s/embeddings?api-version=%s", endpoint, model, p.apiVersion)
	default:
		return endpoint + path
	}
}

func (p *Azure) ChatCompletion(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	url := p.buildURL("/chat/completions", req.Model)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range p.authHeaders() {
		httpReq.Header.Set(k, v)
	}

	resp, err := p.Client.Do(httpReq)
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

func (p *Azure) ChatCompletionStream(ctx context.Context, req *openai.ChatCompletionRequest) (<-chan StreamEvent, error) {
	req.Stream = true
	url := p.buildURL("/chat/completions", req.Model)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	for k, v := range p.authHeaders() {
		httpReq.Header.Set(k, v)
	}

	resp, err := p.Client.Do(httpReq)
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

func (p *Azure) readSSE(body io.ReadCloser, ch chan<- StreamEvent) {
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

func (p *Azure) Completion(ctx context.Context, req *openai.CompletionRequest) (*openai.CompletionResponse, error) {
	url := p.buildURL("/completions", req.Model)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range p.authHeaders() {
		httpReq.Header.Set(k, v)
	}

	resp, err := p.Client.Do(httpReq)
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

func (p *Azure) Embedding(ctx context.Context, req *openai.EmbeddingRequest) (*openai.EmbeddingResponse, error) {
	url := p.buildURL("/embeddings", req.Model)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range p.authHeaders() {
		httpReq.Header.Set(k, v)
	}

	resp, err := p.Client.Do(httpReq)
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

func (p *Azure) ListModels(ctx context.Context) (*openai.ModelList, error) {
	// Azure doesn't have a standard models endpoint
	// Return an empty list or known deployments
	return &openai.ModelList{
		Object: "list",
		Data:   []openai.Model{},
	}, nil
}

func (p *Azure) SupportsStreaming() bool {
	return true
}

func (p *Azure) SupportsEmbeddings() bool {
	return true
}
