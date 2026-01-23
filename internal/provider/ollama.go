package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mindbalancer/mindbalancer/api/openai"
	"github.com/mindbalancer/mindbalancer/internal/storage"
)

// Ollama implements the Provider interface for Ollama.
type Ollama struct {
	*BaseProvider
}

// OllamaChatRequest represents an Ollama chat request.
type OllamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []OllamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  *OllamaOptions  `json:"options,omitempty"`
}

// OllamaMessage represents an Ollama message.
type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaOptions represents Ollama generation options.
type OllamaOptions struct {
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	TopK        *int     `json:"top_k,omitempty"`
	NumPredict  *int     `json:"num_predict,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

// OllamaChatResponse represents an Ollama chat response.
type OllamaChatResponse struct {
	Model     string        `json:"model"`
	CreatedAt string        `json:"created_at"`
	Message   OllamaMessage `json:"message"`
	Done      bool          `json:"done"`
	TotalDuration    int64 `json:"total_duration,omitempty"`
	LoadDuration     int64 `json:"load_duration,omitempty"`
	PromptEvalCount  int   `json:"prompt_eval_count,omitempty"`
	EvalCount        int   `json:"eval_count,omitempty"`
}

// OllamaEmbeddingRequest represents an Ollama embedding request.
type OllamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbeddingResponse represents an Ollama embedding response.
type OllamaEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// OllamaTagsResponse represents Ollama tags (models) response.
type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
}

// OllamaModel represents an Ollama model.
type OllamaModel struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
}

// NewOllama creates a new Ollama provider.
func NewOllama(server storage.Server, timeout time.Duration) *Ollama {
	return &Ollama{
		BaseProvider: NewBaseProvider(server, timeout),
	}
}

func (p *Ollama) Name() string {
	return "ollama"
}

func (p *Ollama) ChatCompletion(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	ollamaReq := p.convertToOllama(req)
	ollamaReq.Stream = false

	resp, err := p.doRequest(ctx, "POST", "/api/chat", ollamaReq, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var result OllamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.convertToOpenAI(&result, req.Model), nil
}

func (p *Ollama) convertToOllama(req *openai.ChatCompletionRequest) *OllamaChatRequest {
	or := &OllamaChatRequest{
		Model:    req.Model,
		Messages: make([]OllamaMessage, len(req.Messages)),
		Stream:   req.Stream,
	}

	for i, msg := range req.Messages {
		var content string
		switch c := msg.Content.(type) {
		case string:
			content = c
		default:
			data, _ := json.Marshal(c)
			content = string(data)
		}
		or.Messages[i] = OllamaMessage{
			Role:    msg.Role,
			Content: content,
		}
	}

	if req.Temperature != nil || req.TopP != nil || req.MaxTokens != nil || len(req.Stop) > 0 {
		or.Options = &OllamaOptions{
			Temperature: req.Temperature,
			TopP:        req.TopP,
			NumPredict:  req.MaxTokens,
			Stop:        req.Stop,
		}
	}

	return or
}

func (p *Ollama) convertToOpenAI(resp *OllamaChatResponse, model string) *openai.ChatCompletionResponse {
	return &openai.ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-ollama-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []openai.Choice{
			{
				Index: 0,
				Message: &openai.Message{
					Role:    resp.Message.Role,
					Content: resp.Message.Content,
				},
				FinishReason: "stop",
			},
		},
		Usage: &openai.Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
	}
}

func (p *Ollama) ChatCompletionStream(ctx context.Context, req *openai.ChatCompletionRequest) (<-chan StreamEvent, error) {
	ollamaReq := p.convertToOllama(req)
	ollamaReq.Stream = true

	resp, err := p.doRequest(ctx, "POST", "/api/chat", ollamaReq, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, p.parseError(resp)
	}

	ch := make(chan StreamEvent, 100)
	go p.readOllamaStream(resp.Body, ch, req.Model)
	return ch, nil
}

func (p *Ollama) readOllamaStream(body io.ReadCloser, ch chan<- StreamEvent, model string) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var resp OllamaChatResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			ch <- StreamEvent{Error: err}
			return
		}

		if resp.Done {
			ch <- StreamEvent{Done: true}
			return
		}

		// Convert to OpenAI format
		chunk := openai.StreamChunk{
			ID:      fmt.Sprintf("chatcmpl-ollama-%d", time.Now().UnixNano()),
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   model,
			Choices: []openai.Choice{
				{
					Index: 0,
					Delta: &openai.Message{
						Content: resp.Message.Content,
					},
				},
			},
		}
		chunkData, _ := json.Marshal(chunk)
		ch <- StreamEvent{Data: chunkData}
	}

	if err := scanner.Err(); err != nil {
		ch <- StreamEvent{Error: err}
	}
}

func (p *Ollama) Completion(ctx context.Context, req *openai.CompletionRequest) (*openai.CompletionResponse, error) {
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

func (p *Ollama) Embedding(ctx context.Context, req *openai.EmbeddingRequest) (*openai.EmbeddingResponse, error) {
	var input string
	switch v := req.Input.(type) {
	case string:
		input = v
	case []string:
		if len(v) > 0 {
			input = v[0]
		}
	case []any:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				input = s
			}
		}
	}

	ollamaReq := OllamaEmbeddingRequest{
		Model:  req.Model,
		Prompt: input,
	}

	resp, err := p.doRequest(ctx, "POST", "/api/embeddings", ollamaReq, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var result OllamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &openai.EmbeddingResponse{
		Object: "list",
		Data: []openai.EmbeddingData{
			{
				Object:    "embedding",
				Embedding: result.Embedding,
				Index:     0,
			},
		},
		Model: req.Model,
	}, nil
}

func (p *Ollama) ListModels(ctx context.Context) (*openai.ModelList, error) {
	resp, err := p.doRequest(ctx, "GET", "/api/tags", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var result OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]openai.Model, len(result.Models))
	for i, m := range result.Models {
		models[i] = openai.Model{
			ID:      m.Name,
			Object:  "model",
			OwnedBy: "ollama",
		}
	}

	return &openai.ModelList{
		Object: "list",
		Data:   models,
	}, nil
}

func (p *Ollama) SupportsStreaming() bool {
	return true
}

func (p *Ollama) SupportsEmbeddings() bool {
	return true
}
