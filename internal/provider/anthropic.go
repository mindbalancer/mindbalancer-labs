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

// Anthropic implements the Provider interface for Anthropic Claude.
type Anthropic struct {
	*BaseProvider
}

// AnthropicRequest represents an Anthropic API request.
type AnthropicRequest struct {
	Model         string             `json:"model"`
	Messages      []AnthropicMessage `json:"messages"`
	MaxTokens     int                `json:"max_tokens"`
	Temperature   *float64           `json:"temperature,omitempty"`
	TopP          *float64           `json:"top_p,omitempty"`
	TopK          *int               `json:"top_k,omitempty"`
	System        any                `json:"system,omitempty"` // string or []SystemBlock for caching
	StopSequences []string           `json:"stop_sequences,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
}

// AnthropicMessage represents an Anthropic message.
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []ContentBlock
}

// CacheControl represents cache control for prompt caching.
type CacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

// SystemBlock represents a system message block with optional cache control.
type SystemBlock struct {
	Type         string        `json:"type"` // "text"
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// ContentBlock represents a content block with optional cache control.
type ContentBlock struct {
	Type         string        `json:"type"`
	Text         string        `json:"text,omitempty"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// AnthropicResponse represents an Anthropic API response.
type AnthropicResponse struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Content      []AnthropicContent `json:"content"`
	Model        string             `json:"model"`
	StopReason   string             `json:"stop_reason"`
	StopSequence string             `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage     `json:"usage"`
}

// AnthropicContent represents content in an Anthropic response.
type AnthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// AnthropicUsage represents usage in an Anthropic response.
type AnthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// NewAnthropic creates a new Anthropic provider.
func NewAnthropic(server storage.Server, timeout time.Duration) *Anthropic {
	return &Anthropic{
		BaseProvider: NewBaseProvider(server, timeout),
	}
}

func (p *Anthropic) Name() string {
	return "anthropic"
}

func (p *Anthropic) authHeaders() map[string]string {
	headers := map[string]string{
		"anthropic-version": "2023-06-01",
		"anthropic-beta":    "prompt-caching-2024-07-31", // Enable prompt caching
	}
	if p.Server.APIKeyEncrypted != "" {
		headers["x-api-key"] = p.Server.APIKeyEncrypted
	}
	return headers
}

// Minimum tokens for prompt caching (Anthropic requires at least 1024 tokens for caching)
const minTokensForCaching = 1024

// estimateTokens roughly estimates token count (avg 4 chars per token)
func estimateTokens(text string) int {
	return len(text) / 4
}

// convertToAnthropic converts OpenAI request to Anthropic format with prompt caching support.
func (p *Anthropic) convertToAnthropic(req *openai.ChatCompletionRequest) *AnthropicRequest {
	ar := &AnthropicRequest{
		Model:       req.Model,
		Messages:    make([]AnthropicMessage, 0),
		MaxTokens:   4096,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
	}

	if req.MaxTokens != nil {
		ar.MaxTokens = *req.MaxTokens
	}

	var systemContent string
	for _, msg := range req.Messages {
		// Extract system message
		if msg.Role == "system" {
			if content, ok := msg.Content.(string); ok {
				systemContent = content
			}
			continue
		}

		am := AnthropicMessage{Role: msg.Role}

		// Convert content
		switch c := msg.Content.(type) {
		case string:
			am.Content = c
		default:
			am.Content = msg.Content
		}

		// Map assistant role
		if msg.Role == "assistant" {
			am.Role = "assistant"
		} else if msg.Role == "user" {
			am.Role = "user"
		}

		ar.Messages = append(ar.Messages, am)
	}

	// Apply prompt caching to system message if it's large enough
	if systemContent != "" {
		if estimateTokens(systemContent) >= minTokensForCaching {
			// Use structured system with cache_control for large system prompts
			ar.System = []SystemBlock{
				{
					Type: "text",
					Text: systemContent,
					CacheControl: &CacheControl{
						Type: "ephemeral",
					},
				},
			}
		} else {
			ar.System = systemContent
		}
	}

	// Mark the last long user message for caching (useful for few-shot examples)
	if len(ar.Messages) > 0 {
		lastIdx := len(ar.Messages) - 1
		// Find the last user message with substantial content
		for i := lastIdx; i >= 0; i-- {
			if ar.Messages[i].Role == "user" {
				if content, ok := ar.Messages[i].Content.(string); ok {
					if estimateTokens(content) >= minTokensForCaching {
						// Convert to content block with cache control
						ar.Messages[i].Content = []ContentBlock{
							{
								Type: "text",
								Text: content,
								CacheControl: &CacheControl{
									Type: "ephemeral",
								},
							},
						}
					}
				}
				break
			}
		}
	}

	if len(req.Stop) > 0 {
		ar.StopSequences = req.Stop
	}

	return ar
}

// convertToOpenAI converts Anthropic response to OpenAI format.
func (p *Anthropic) convertToOpenAI(resp *AnthropicResponse) *openai.ChatCompletionResponse {
	var content string
	for _, c := range resp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	finishReason := "stop"
	switch resp.StopReason {
	case "end_turn":
		finishReason = "stop"
	case "max_tokens":
		finishReason = "length"
	case "stop_sequence":
		finishReason = "stop"
	}

	usage := &openai.Usage{
		PromptTokens:     resp.Usage.InputTokens,
		CompletionTokens: resp.Usage.OutputTokens,
		TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
	}

	// Include cache information if available
	if resp.Usage.CacheReadInputTokens > 0 || resp.Usage.CacheCreationInputTokens > 0 {
		usage.PromptTokensDetails = &openai.PromptTokensDetails{
			CachedTokens: resp.Usage.CacheReadInputTokens,
		}
	}

	return &openai.ChatCompletionResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   resp.Model,
		Choices: []openai.Choice{
			{
				Index: 0,
				Message: &openai.Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: finishReason,
			},
		},
		Usage: usage,
	}
}

func (p *Anthropic) ChatCompletion(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	anthropicReq := p.convertToAnthropic(req)

	resp, err := p.doRequest(ctx, "POST", "/v1/messages", anthropicReq, p.authHeaders())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var result AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.convertToOpenAI(&result), nil
}

func (p *Anthropic) ChatCompletionStream(ctx context.Context, req *openai.ChatCompletionRequest) (<-chan StreamEvent, error) {
	anthropicReq := p.convertToAnthropic(req)
	anthropicReq.Stream = true

	headers := p.authHeaders()
	headers["Accept"] = "text/event-stream"

	resp, err := p.doRequest(ctx, "POST", "/v1/messages", anthropicReq, headers)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, p.parseError(resp)
	}

	ch := make(chan StreamEvent, 100)
	go p.readAnthropicSSE(resp.Body, ch, req.Model)
	return ch, nil
}

func (p *Anthropic) readAnthropicSSE(body io.ReadCloser, ch chan<- StreamEvent, model string) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	var eventType string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			switch eventType {
			case "content_block_delta":
				var delta struct {
					Delta struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"delta"`
				}
				if err := json.Unmarshal([]byte(data), &delta); err == nil {
					// Convert to OpenAI format
					chunk := openai.StreamChunk{
						ID:      "chatcmpl-anthropic",
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Model:   model,
						Choices: []openai.Choice{
							{
								Index: 0,
								Delta: &openai.Message{
									Content: delta.Delta.Text,
								},
							},
						},
					}
					chunkData, _ := json.Marshal(chunk)
					ch <- StreamEvent{Data: chunkData}
				}

			case "message_stop":
				ch <- StreamEvent{Done: true}
				return

			case "error":
				ch <- StreamEvent{Error: fmt.Errorf("anthropic stream error: %s", data)}
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- StreamEvent{Error: err}
	}
}

func (p *Anthropic) Completion(ctx context.Context, req *openai.CompletionRequest) (*openai.CompletionResponse, error) {
	// Convert to chat format
	chatReq := &openai.ChatCompletionRequest{
		Model: req.Model,
		Messages: []openai.Message{
			{Role: "user", Content: req.Prompt},
		},
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
		Stop:        req.Stop,
	}

	chatResp, err := p.ChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, err
	}

	var text string
	if len(chatResp.Choices) > 0 && chatResp.Choices[0].Message != nil {
		if content, ok := chatResp.Choices[0].Message.Content.(string); ok {
			text = content
		}
	}

	return &openai.CompletionResponse{
		ID:      chatResp.ID,
		Object:  "text_completion",
		Created: chatResp.Created,
		Model:   chatResp.Model,
		Choices: []openai.CompletionChoice{
			{
				Text:         text,
				Index:        0,
				FinishReason: chatResp.Choices[0].FinishReason,
			},
		},
		Usage: chatResp.Usage,
	}, nil
}

func (p *Anthropic) Embedding(ctx context.Context, req *openai.EmbeddingRequest) (*openai.EmbeddingResponse, error) {
	return nil, fmt.Errorf("anthropic does not support embeddings")
}

func (p *Anthropic) ListModels(ctx context.Context) (*openai.ModelList, error) {
	// Anthropic doesn't have a models endpoint, return known models
	models := []openai.Model{
		{ID: "claude-3-opus-20240229", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-3-sonnet-20240229", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-3-haiku-20240307", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-2.1", Object: "model", OwnedBy: "anthropic"},
		{ID: "claude-2.0", Object: "model", OwnedBy: "anthropic"},
	}

	return &openai.ModelList{
		Object: "list",
		Data:   models,
	}, nil
}

func (p *Anthropic) SupportsStreaming() bool {
	return true
}

func (p *Anthropic) SupportsEmbeddings() bool {
	return false
}
