// Package referee provides consensus-based AI response synthesis.
// It sends the same request to multiple providers and uses a referee model
// to synthesize the best response from all answers.
package referee

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/mindbalancer/mindbalancer/api/openai"
	"github.com/mindbalancer/mindbalancer/internal/balancer"
	"github.com/mindbalancer/mindbalancer/internal/config"
	"github.com/mindbalancer/mindbalancer/internal/provider"
	"github.com/mindbalancer/mindbalancer/internal/storage"
)

// ProviderResponse holds a response from a single provider.
type ProviderResponse struct {
	ProviderName string
	ProviderType string
	Model        string
	Response     *openai.ChatCompletionResponse
	Error        error
	Latency      time.Duration
}

// Engine handles referee mode execution.
type Engine struct {
	config   *config.Config
	storage  *storage.Storage
	balancer *balancer.Balancer
}

// NewEngine creates a new referee engine.
func NewEngine(cfg *config.Config, store *storage.Storage, bal *balancer.Balancer) *Engine {
	return &Engine{
		config:   cfg,
		storage:  store,
		balancer: bal,
	}
}

// Execute runs referee mode for a chat completion request.
func (e *Engine) Execute(ctx context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	if req.RefereeMode == nil || !req.RefereeMode.Enabled {
		return nil, fmt.Errorf("referee mode not enabled in request")
	}

	startTime := time.Now()

	// Get available servers based on provider filter
	servers, err := e.getServersForReferee(ctx, req.RefereeMode.Providers)
	if err != nil {
		return nil, fmt.Errorf("failed to get servers: %w", err)
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("no servers available for referee mode")
	}

	// Limit to max providers
	maxProviders := e.config.RefereeMaxProviders
	if len(servers) > maxProviders {
		servers = servers[:maxProviders]
	}

	// Determine timeout
	timeout := e.config.RefereeTimeout()
	if req.RefereeMode.TimeoutMS > 0 {
		timeout = time.Duration(req.RefereeMode.TimeoutMS) * time.Millisecond
	}

	// Determine minimum responses
	minResponses := e.config.RefereeMinResponses
	if req.RefereeMode.MinResponses > 0 {
		minResponses = req.RefereeMode.MinResponses
	}

	// Execute parallel requests
	responses := e.executeParallel(ctx, req, servers, timeout)

	// Filter successful responses
	var successfulResponses []ProviderResponse
	var failedProviders []string
	for _, resp := range responses {
		if resp.Error == nil && resp.Response != nil {
			successfulResponses = append(successfulResponses, resp)
		} else {
			failedProviders = append(failedProviders, resp.ProviderName)
			if resp.Error != nil {
				log.Printf("[REFEREE] Provider %s failed: %v", resp.ProviderName, resp.Error)
			}
		}
	}

	// Check minimum responses
	if len(successfulResponses) < minResponses {
		return nil, fmt.Errorf("insufficient responses: got %d, need %d (failed: %v)",
			len(successfulResponses), minResponses, failedProviders)
	}

	// Determine referee model
	refereeModel := req.RefereeMode.RefereeModel
	if refereeModel == "" {
		refereeModel = e.config.RefereeDefaultModel
	}

	// Synthesize responses
	synthesisStart := time.Now()
	synthesized, err := e.synthesize(ctx, req, successfulResponses, refereeModel, timeout)
	if err != nil {
		return nil, fmt.Errorf("synthesis failed: %w", err)
	}
	synthesisLatency := time.Since(synthesisStart)

	// Add referee metadata to response
	synthesized.RefereeInfo = &openai.RefereeResponse{
		ProvidersQueried:    len(servers),
		SuccessfulResponses: len(successfulResponses),
		FailedProviders:     failedProviders,
		RefereeModel:        refereeModel,
		SynthesisLatencyMS:  synthesisLatency.Milliseconds(),
	}

	log.Printf("[REFEREE] Completed in %v: queried=%d, successful=%d, failed=%d, synthesis=%v",
		time.Since(startTime), len(servers), len(successfulResponses), len(failedProviders), synthesisLatency)

	return synthesized, nil
}

// getServersForReferee returns servers filtered by provider types.
func (e *Engine) getServersForReferee(ctx context.Context, providerTypes []string) ([]storage.Server, error) {
	// Decrypt keys: referee fans out real provider calls using these credentials.
	allServers, err := e.storage.GetServersWithDecryptedKeys(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Filter by status
	var activeServers []storage.Server
	for _, srv := range allServers {
		if srv.Status == storage.ServerStatusOnline {
			activeServers = append(activeServers, srv)
		}
	}

	// If no provider filter, return all active servers
	if len(providerTypes) == 0 {
		return activeServers, nil
	}

	// Filter by provider type
	providerSet := make(map[string]bool)
	for _, pt := range providerTypes {
		providerSet[strings.ToLower(pt)] = true
	}

	var filtered []storage.Server
	for _, srv := range activeServers {
		if providerSet[strings.ToLower(srv.ProviderType)] {
			filtered = append(filtered, srv)
		}
	}

	return filtered, nil
}

// executeParallel sends the request to all servers in parallel.
func (e *Engine) executeParallel(ctx context.Context, req *openai.ChatCompletionRequest, servers []storage.Server, timeout time.Duration) []ProviderResponse {
	var wg sync.WaitGroup
	responses := make([]ProviderResponse, len(servers))

	// Create a clean request without referee mode to avoid infinite loops
	cleanReq := *req
	cleanReq.RefereeMode = nil
	cleanReq.Stream = false // Referee mode doesn't support streaming

	for i, srv := range servers {
		wg.Add(1)
		go func(idx int, server storage.Server) {
			defer wg.Done()

			provCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			start := time.Now()

			// Create provider
			prov := provider.New(server, timeout)

			// Execute request
			resp, err := prov.ChatCompletion(provCtx, &cleanReq)

			responses[idx] = ProviderResponse{
				ProviderName: server.Name,
				ProviderType: server.ProviderType,
				Model:        cleanReq.Model,
				Response:     resp,
				Error:        err,
				Latency:      time.Since(start),
			}

			if err == nil {
				log.Printf("[REFEREE] Provider %s (%s) responded in %v", server.Name, server.ProviderType, responses[idx].Latency)
			}
		}(i, srv)
	}

	wg.Wait()
	return responses
}

// synthesize uses the referee model to combine all responses into one.
func (e *Engine) synthesize(ctx context.Context, originalReq *openai.ChatCompletionRequest, responses []ProviderResponse, refereeModel string, timeout time.Duration) (*openai.ChatCompletionResponse, error) {
	// Build synthesis prompt
	prompt := e.buildSynthesisPrompt(originalReq, responses)

	// Find a server that can run the referee model
	refereeServer, err := e.findServerForModel(ctx, refereeModel)
	if err != nil {
		return nil, fmt.Errorf("no server available for referee model %s: %w", refereeModel, err)
	}

	// Create synthesis request
	synthReq := &openai.ChatCompletionRequest{
		Model: refereeModel,
		Messages: []openai.Message{
			{
				Role:    "system",
				Content: "You are an expert AI response synthesizer. Your task is to analyze multiple AI responses to the same question and produce the best possible answer by combining accurate information, resolving contradictions, and eliminating errors.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: originalReq.Temperature,
		MaxTokens:   originalReq.MaxTokens,
	}

	// Execute synthesis
	prov := provider.New(*refereeServer, timeout)
	return prov.ChatCompletion(ctx, synthReq)
}

// buildSynthesisPrompt creates the prompt for the referee model.
func (e *Engine) buildSynthesisPrompt(originalReq *openai.ChatCompletionRequest, responses []ProviderResponse) string {
	var sb strings.Builder

	sb.WriteString("## Original Question\n\n")

	// Extract original user message
	for _, msg := range originalReq.Messages {
		if msg.Role == "user" {
			if content, ok := msg.Content.(string); ok {
				sb.WriteString(content)
				sb.WriteString("\n\n")
				break
			}
		}
	}

	sb.WriteString("## Responses from Different AI Models\n\n")

	for i, resp := range responses {
		sb.WriteString(fmt.Sprintf("### Response %d (from %s via %s)\n\n", i+1, resp.Model, resp.ProviderType))

		if resp.Response != nil && len(resp.Response.Choices) > 0 {
			choice := resp.Response.Choices[0]
			if choice.Message != nil {
				if content, ok := choice.Message.Content.(string); ok {
					sb.WriteString(content)
				}
			}
		}
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Your Task\n\n")
	sb.WriteString("Please analyze all the responses above and create a synthesized answer that:\n")
	sb.WriteString("1. Identifies and includes the correct and accurate information from each response\n")
	sb.WriteString("2. Resolves any contradictions by determining which response is more accurate\n")
	sb.WriteString("3. Combines the best elements from all responses\n")
	sb.WriteString("4. Provides a comprehensive and accurate final answer\n\n")
	sb.WriteString("Do NOT mention that you are synthesizing from multiple sources. Simply provide the best answer as if you were answering the original question directly.\n\n")
	sb.WriteString("## Synthesized Answer\n\n")

	return sb.String()
}

// findServerForModel finds a server that can run the specified model.
func (e *Engine) findServerForModel(ctx context.Context, model string) (*storage.Server, error) {
	// Decrypt keys: the returned server is used to make an authenticated provider call.
	servers, err := e.storage.GetServersWithDecryptedKeys(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Determine provider type from model name
	providerType := guessProviderFromModel(model)

	// First, try to find a server of the matching provider type
	for _, srv := range servers {
		if srv.Status == storage.ServerStatusOnline && strings.EqualFold(srv.ProviderType, providerType) {
			return &srv, nil
		}
	}

	// Fallback: return any online server
	for _, srv := range servers {
		if srv.Status == storage.ServerStatusOnline {
			return &srv, nil
		}
	}

	return nil, fmt.Errorf("no online server found")
}

// guessProviderFromModel guesses the provider type from a model name.
func guessProviderFromModel(model string) string {
	model = strings.ToLower(model)

	switch {
	case strings.HasPrefix(model, "gpt") || strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3"):
		return "openai"
	case strings.HasPrefix(model, "claude"):
		return "anthropic"
	case strings.HasPrefix(model, "gemini"):
		return "google"
	case strings.HasPrefix(model, "llama") || strings.HasPrefix(model, "mixtral"):
		return "groq"
	default:
		return "openai" // Default fallback
	}
}

// Result holds the final referee mode result.
type Result struct {
	Response         *openai.ChatCompletionResponse
	ProviderResults  []ProviderResponse
	TotalLatency     time.Duration
	SynthesisLatency time.Duration
}

// GetProviderSummary returns a JSON-serializable summary of provider results.
func (r *Result) GetProviderSummary() []map[string]interface{} {
	var summary []map[string]interface{}
	for _, pr := range r.ProviderResults {
		entry := map[string]interface{}{
			"provider":   pr.ProviderName,
			"type":       pr.ProviderType,
			"model":      pr.Model,
			"latency_ms": pr.Latency.Milliseconds(),
			"success":    pr.Error == nil,
		}
		if pr.Error != nil {
			entry["error"] = pr.Error.Error()
		}
		summary = append(summary, entry)
	}
	return summary
}

// MarshalJSON implements custom JSON marshaling for Result.
func (r *Result) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"response":             r.Response,
		"provider_summary":     r.GetProviderSummary(),
		"total_latency_ms":     r.TotalLatency.Milliseconds(),
		"synthesis_latency_ms": r.SynthesisLatency.Milliseconds(),
	})
}
