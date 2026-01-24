// Package openai provides OpenAI API compatible types and interfaces.
package openai

import "time"

// ChatCompletionRequest represents a chat completion request.
type ChatCompletionRequest struct {
	Model            string          `json:"model"`
	Messages         []Message       `json:"messages"`
	Temperature      *float64        `json:"temperature,omitempty"`
	TopP             *float64        `json:"top_p,omitempty"`
	N                *int            `json:"n,omitempty"`
	Stream           bool            `json:"stream,omitempty"`
	Stop             []string        `json:"stop,omitempty"`
	MaxTokens        *int            `json:"max_tokens,omitempty"`
	PresencePenalty  *float64        `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64        `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]int  `json:"logit_bias,omitempty"`
	User             string          `json:"user,omitempty"`
	Functions        []Function      `json:"functions,omitempty"`
	FunctionCall     any             `json:"function_call,omitempty"`
	Tools            []Tool          `json:"tools,omitempty"`
	ToolChoice       any             `json:"tool_choice,omitempty"`
	ResponseFormat   *ResponseFormat `json:"response_format,omitempty"`
	Seed             *int            `json:"seed,omitempty"`
}

// Message represents a chat message.
type Message struct {
	Role         string     `json:"role"`
	Content      any        `json:"content"` // string or []ContentPart
	Name         string     `json:"name,omitempty"`
	FunctionCall *FuncCall  `json:"function_call,omitempty"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID   string     `json:"tool_call_id,omitempty"`
}

// ContentPart represents a part of multi-modal content.
type ContentPart struct {
	Type     string    `json:"type"` // "text" or "image_url"
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL represents an image URL in content.
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "auto", "low", "high"
}

// Function represents a function definition for function calling.
type Function struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

// FuncCall represents a function call in a message.
type FuncCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool represents a tool for tool use.
type Tool struct {
	Type     string   `json:"type"` // "function"
	Function Function `json:"function"`
}

// ToolCall represents a tool call in a message.
type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"` // "function"
	Function FuncCall `json:"function"`
}

// ResponseFormat specifies the format of the response.
type ResponseFormat struct {
	Type string `json:"type"` // "text" or "json_object"
}

// ChatCompletionResponse represents a chat completion response.
type ChatCompletionResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             *Usage   `json:"usage,omitempty"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}

// Choice represents a completion choice.
type Choice struct {
	Index        int      `json:"index"`
	Message      *Message `json:"message,omitempty"`
	Delta        *Message `json:"delta,omitempty"` // For streaming
	FinishReason string   `json:"finish_reason,omitempty"`
	Logprobs     any      `json:"logprobs,omitempty"`
}

// Usage represents token usage information.
type Usage struct {
	PromptTokens     int           `json:"prompt_tokens"`
	CompletionTokens int           `json:"completion_tokens"`
	TotalTokens      int           `json:"total_tokens"`
	PromptTokensDetails *PromptTokensDetails `json:"prompt_tokens_details,omitempty"`
}

// PromptTokensDetails provides detailed breakdown of prompt tokens.
type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
	AudioTokens  int `json:"audio_tokens,omitempty"`
}

// CompletionRequest represents a legacy completion request.
type CompletionRequest struct {
	Model            string         `json:"model"`
	Prompt           any            `json:"prompt"` // string or []string
	Suffix           string         `json:"suffix,omitempty"`
	MaxTokens        *int           `json:"max_tokens,omitempty"`
	Temperature      *float64       `json:"temperature,omitempty"`
	TopP             *float64       `json:"top_p,omitempty"`
	N                *int           `json:"n,omitempty"`
	Stream           bool           `json:"stream,omitempty"`
	Logprobs         *int           `json:"logprobs,omitempty"`
	Echo             bool           `json:"echo,omitempty"`
	Stop             []string       `json:"stop,omitempty"`
	PresencePenalty  *float64       `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64       `json:"frequency_penalty,omitempty"`
	BestOf           *int           `json:"best_of,omitempty"`
	LogitBias        map[string]int `json:"logit_bias,omitempty"`
	User             string         `json:"user,omitempty"`
}

// CompletionResponse represents a legacy completion response.
type CompletionResponse struct {
	ID                string             `json:"id"`
	Object            string             `json:"object"`
	Created           int64              `json:"created"`
	Model             string             `json:"model"`
	Choices           []CompletionChoice `json:"choices"`
	Usage             *Usage             `json:"usage,omitempty"`
	SystemFingerprint string             `json:"system_fingerprint,omitempty"`
}

// CompletionChoice represents a completion choice for legacy completions.
type CompletionChoice struct {
	Text         string `json:"text"`
	Index        int    `json:"index"`
	Logprobs     any    `json:"logprobs,omitempty"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// EmbeddingRequest represents an embedding request.
type EmbeddingRequest struct {
	Model          string `json:"model"`
	Input          any    `json:"input"` // string or []string
	User           string `json:"user,omitempty"`
	EncodingFormat string `json:"encoding_format,omitempty"` // "float" or "base64"
	Dimensions     *int   `json:"dimensions,omitempty"`
}

// EmbeddingResponse represents an embedding response.
type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  *EmbeddingUsage `json:"usage,omitempty"`
}

// EmbeddingData represents a single embedding.
type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

// EmbeddingUsage represents token usage for embeddings.
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Model represents an AI model.
type Model struct {
	ID         string `json:"id"`
	Object     string `json:"object"`
	Created    int64  `json:"created"`
	OwnedBy    string `json:"owned_by"`
	Permission []any  `json:"permission,omitempty"`
	Root       string `json:"root,omitempty"`
	Parent     string `json:"parent,omitempty"`
}

// ModelList represents a list of models.
type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error *APIError `json:"error"`
}

// APIError represents an API error.
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}

// StreamChunk represents a streaming response chunk.
type StreamChunk struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}

// RequestMetadata holds metadata about a request for logging/metrics.
type RequestMetadata struct {
	RequestID     string
	UserID        string
	Model         string
	Provider      string
	ServerName    string
	StartTime     time.Time
	EndTime       time.Time
	PromptTokens  int
	OutputTokens  int
	TotalTokens   int
	Streaming     bool
	StatusCode    int
	Error         string
	Latency       time.Duration
	FirstByteTime time.Duration
}
