// Package provider provides adapters for different AI providers.
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mindbalancer/mindbalancer/api/openai"
	"github.com/mindbalancer/mindbalancer/internal/pool"
	"github.com/mindbalancer/mindbalancer/internal/storage"
)

// Provider interface for AI providers.
type Provider interface {
	// Name returns the provider name
	Name() string

	// ChatCompletion sends a chat completion request
	ChatCompletion(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error)

	// ChatCompletionStream sends a streaming chat completion request
	ChatCompletionStream(ctx context.Context, req *openai.ChatCompletionRequest) (<-chan StreamEvent, error)

	// Completion sends a legacy completion request
	Completion(ctx context.Context, req *openai.CompletionRequest) (*openai.CompletionResponse, error)

	// Embedding sends an embedding request
	Embedding(ctx context.Context, req *openai.EmbeddingRequest) (*openai.EmbeddingResponse, error)

	// ListModels returns available models
	ListModels(ctx context.Context) (*openai.ModelList, error)

	// SupportsStreaming returns true if provider supports streaming
	SupportsStreaming() bool

	// SupportsEmbeddings returns true if provider supports embeddings
	SupportsEmbeddings() bool
}

// StreamEvent represents a streaming event.
type StreamEvent struct {
	Data  []byte
	Error error
	Done  bool
}

// BaseProvider provides common functionality for providers.
type BaseProvider struct {
	Server  storage.Server
	Client  *http.Client
	Timeout time.Duration
}

// NewBaseProvider creates a new base provider.
// Uses the global connection pool for efficient connection reuse.
func NewBaseProvider(server storage.Server, timeout time.Duration) *BaseProvider {
	return &BaseProvider{
		Server:  server,
		Client:  pool.GetGlobalClientWithTimeout(timeout),
		Timeout: timeout,
	}
}

// doRequest performs an HTTP request.
func (p *BaseProvider) doRequest(ctx context.Context, method, path string, body any, headers map[string]string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := strings.TrimSuffix(p.Server.Endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return p.Client.Do(req)
}

// parseError parses an error response.
func (p *BaseProvider) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	var errResp openai.ErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != nil {
		return &ProviderError{
			StatusCode: resp.StatusCode,
			Type:       errResp.Error.Type,
			Message:    errResp.Error.Message,
			Code:       errResp.Error.Code,
		}
	}

	return &ProviderError{
		StatusCode: resp.StatusCode,
		Message:    string(body),
	}
}

// ProviderError represents an error from a provider.
type ProviderError struct {
	StatusCode int
	Type       string
	Message    string
	Code       string
}

func (e *ProviderError) Error() string {
	if e.Type != "" {
		return fmt.Sprintf("%s: %s (status %d)", e.Type, e.Message, e.StatusCode)
	}
	return fmt.Sprintf("provider error (status %d): %s", e.StatusCode, e.Message)
}

// IsRetryable returns true if the error is retryable.
func (e *ProviderError) IsRetryable() bool {
	switch e.StatusCode {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

// New creates a provider for a server.
func New(server storage.Server, timeout time.Duration) Provider {
	switch strings.ToLower(server.ProviderType) {
	case "openai":
		return NewOpenAI(server, timeout)
	case "anthropic":
		return NewAnthropic(server, timeout)
	case "ollama":
		return NewOllama(server, timeout)
	case "azure":
		return NewAzure(server, timeout)
	case "groq":
		return NewGroq(server, timeout)
	case "google":
		return NewGoogle(server, timeout)
	case "custom":
		return NewOpenAI(server, timeout) // Custom assumes OpenAI-compatible
	default:
		return NewOpenAI(server, timeout)
	}
}
