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

// Google implements the Provider interface for Google AI (Gemini).
type Google struct {
	*BaseProvider
}

// GeminiRequest represents a Gemini API request.
type GeminiRequest struct {
	Contents         []GeminiContent   `json:"contents"`
	GenerationConfig *GeminiGenConfig  `json:"generationConfig,omitempty"`
	SafetySettings   []GeminiSafety    `json:"safetySettings,omitempty"`
}

// GeminiContent represents Gemini content.
type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart represents a part of Gemini content.
type GeminiPart struct {
	Text string `json:"text,omitempty"`
}

// GeminiGenConfig represents Gemini generation config.
type GeminiGenConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	TopK            *int     `json:"topK,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

// GeminiSafety represents Gemini safety settings.
type GeminiSafety struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// GeminiResponse represents a Gemini API response.
type GeminiResponse struct {
	Candidates    []GeminiCandidate `json:"candidates"`
	UsageMetadata *GeminiUsage      `json:"usageMetadata,omitempty"`
}

// GeminiCandidate represents a Gemini candidate.
type GeminiCandidate struct {
	Content       GeminiContent `json:"content"`
	FinishReason  string        `json:"finishReason"`
	SafetyRatings []any         `json:"safetyRatings,omitempty"`
}

// GeminiUsage represents Gemini usage metadata.
type GeminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// NewGoogle creates a new Google provider.
func NewGoogle(server storage.Server, timeout time.Duration) *Google {
	return &Google{
		BaseProvider: NewBaseProvider(server, timeout),
	}
}

func (p *Google) Name() string {
	return "google"
}

// buildURL builds Google AI URL.
func (p *Google) buildURL(model string, stream bool) string {
	endpoint := strings.TrimSuffix(p.Server.Endpoint, "/")
	if endpoint == "" {
		endpoint = "https://generativelanguage.googleapis.com"
	}

	action := "generateContent"
	if stream {
		action = "streamGenerateContent"
	}

	return fmt.Sprintf("%s/v1beta/models/%s:%s?key=%s",
		endpoint, model, action, p.Server.APIKeyEncrypted)
}

func (p *Google) convertToGemini(req *openai.ChatCompletionRequest) *GeminiRequest {
	gr := &GeminiRequest{
		Contents: make([]GeminiContent, 0),
	}

	for _, msg := range req.Messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		if role == "system" {
			// Prepend to first user message
			continue
		}

		var text string
		switch c := msg.Content.(type) {
		case string:
			text = c
		default:
			data, _ := json.Marshal(c)
			text = string(data)
		}

		gr.Contents = append(gr.Contents, GeminiContent{
			Role:  role,
			Parts: []GeminiPart{{Text: text}},
		})
	}

	if req.Temperature != nil || req.TopP != nil || req.MaxTokens != nil || len(req.Stop) > 0 {
		gr.GenerationConfig = &GeminiGenConfig{
			Temperature:     req.Temperature,
			TopP:            req.TopP,
			MaxOutputTokens: req.MaxTokens,
			StopSequences:   req.Stop,
		}
	}

	return gr
}

func (p *Google) convertToOpenAI(resp *GeminiResponse, model string) *openai.ChatCompletionResponse {
	var content string
	var finishReason string

	if len(resp.Candidates) > 0 {
		cand := resp.Candidates[0]
		if len(cand.Content.Parts) > 0 {
			content = cand.Content.Parts[0].Text
		}
		switch cand.FinishReason {
		case "STOP":
			finishReason = "stop"
		case "MAX_TOKENS":
			finishReason = "length"
		case "SAFETY":
			finishReason = "content_filter"
		default:
			finishReason = "stop"
		}
	}

	result := &openai.ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-google-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
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
	}

	if resp.UsageMetadata != nil {
		result.Usage = &openai.Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	return result
}

func (p *Google) ChatCompletion(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	geminiReq := p.convertToGemini(req)
	url := p.buildURL(req.Model, false)

	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var result GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.convertToOpenAI(&result, req.Model), nil
}

func (p *Google) ChatCompletionStream(ctx context.Context, req *openai.ChatCompletionRequest) (<-chan StreamEvent, error) {
	geminiReq := p.convertToGemini(req)
	url := p.buildURL(req.Model, true)

	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, p.parseError(resp)
	}

	ch := make(chan StreamEvent, 100)
	go p.readGeminiStream(resp.Body, ch, req.Model)
	return ch, nil
}

func (p *Google) readGeminiStream(body io.ReadCloser, ch chan<- StreamEvent, model string) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line == "[" || line == "]" || line == "," {
			continue
		}

		line = strings.TrimPrefix(line, ",")

		var resp GeminiResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue
		}

		if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
			content := resp.Candidates[0].Content.Parts[0].Text

			chunk := openai.StreamChunk{
				ID:      fmt.Sprintf("chatcmpl-google-%d", time.Now().UnixNano()),
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   model,
				Choices: []openai.Choice{
					{
						Index: 0,
						Delta: &openai.Message{
							Content: content,
						},
					},
				},
			}
			chunkData, _ := json.Marshal(chunk)
			ch <- StreamEvent{Data: chunkData}
		}
	}

	ch <- StreamEvent{Done: true}
}

func (p *Google) Completion(ctx context.Context, req *openai.CompletionRequest) (*openai.CompletionResponse, error) {
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

func (p *Google) Embedding(ctx context.Context, req *openai.EmbeddingRequest) (*openai.EmbeddingResponse, error) {
	// Google has embedding models but different API format
	// TODO: Implement Google embeddings
	return nil, fmt.Errorf("google embeddings not yet implemented")
}

func (p *Google) ListModels(ctx context.Context) (*openai.ModelList, error) {
	models := []openai.Model{
		{ID: "gemini-1.5-pro", Object: "model", OwnedBy: "google"},
		{ID: "gemini-1.5-flash", Object: "model", OwnedBy: "google"},
		{ID: "gemini-1.0-pro", Object: "model", OwnedBy: "google"},
	}

	return &openai.ModelList{
		Object: "list",
		Data:   models,
	}, nil
}

func (p *Google) SupportsStreaming() bool {
	return true
}

func (p *Google) SupportsEmbeddings() bool {
	return true
}
