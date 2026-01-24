// Package proxy provides the OpenAI-compatible API proxy.
package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mindbalancer/mindbalancer/api/openai"
	"github.com/mindbalancer/mindbalancer/internal/balancer"
	"github.com/mindbalancer/mindbalancer/internal/cache"
	"github.com/mindbalancer/mindbalancer/internal/config"
	"github.com/mindbalancer/mindbalancer/internal/metrics"
	"github.com/mindbalancer/mindbalancer/internal/provider"
	"github.com/mindbalancer/mindbalancer/internal/ratelimit"
	"github.com/mindbalancer/mindbalancer/internal/referee"
	"github.com/mindbalancer/mindbalancer/internal/retry"
	"github.com/mindbalancer/mindbalancer/internal/router"
	"github.com/mindbalancer/mindbalancer/internal/storage"
)

// Proxy handles OpenAI-compatible API requests.
type Proxy struct {
	config    *config.Config
	storage   *storage.Storage
	balancer  *balancer.Balancer
	router    *router.Router
	metrics   *metrics.Collector
	ratelimit *ratelimit.Limiter
	cache     *cache.Cache
	referee   *referee.Engine
}

// NewProxy creates a new proxy.
func NewProxy(cfg *config.Config, store *storage.Storage, bal *balancer.Balancer, rtr *router.Router, met *metrics.Collector, rl *ratelimit.Limiter) *Proxy {
	// Initialize cache from config
	cacheCfg := cfg.GetCacheConfig()
	cacheInstance := cache.NewCache(cache.Config{
		Enabled:            cacheCfg.Enabled,
		MaxSize:            cacheCfg.MaxSize,
		MaxMemoryMB:        cacheCfg.MaxMemoryMB,
		TTL:                cacheCfg.TTL,
		MaxItemSize:        cacheCfg.MaxItemSize,
		NumShards:          16,
		CompressionEnabled: cacheCfg.CompressionEnabled,
		CompressionMinSize: 1024, // 1KB
		CleanupInterval:    time.Minute,
		EmbeddingsTTL:      cacheCfg.EmbeddingsTTL,
	})

	// Initialize referee engine
	refereeEngine := referee.NewEngine(cfg, store, bal)

	return &Proxy{
		config:    cfg,
		storage:   store,
		balancer:  bal,
		router:    rtr,
		metrics:   met,
		ratelimit: rl,
		cache:     cacheInstance,
		referee:   refereeEngine,
	}
}

// GetCache returns the cache instance for external management.
func (p *Proxy) GetCache() *cache.Cache {
	return p.cache
}

// Handler returns the HTTP handler for the proxy.
func (p *Proxy) Handler() http.Handler {
	mux := http.NewServeMux()

	// OpenAI-compatible endpoints
	mux.HandleFunc("/v1/chat/completions", p.handleChatCompletions)
	mux.HandleFunc("/v1/completions", p.handleCompletions)
	mux.HandleFunc("/v1/embeddings", p.handleEmbeddings)
	mux.HandleFunc("/v1/models", p.handleModels)

	// Health endpoint
	mux.HandleFunc("/health", p.handleHealth)
	mux.HandleFunc("/healthz", p.handleHealth)

	// Root
	mux.HandleFunc("/", p.handleRoot)

	return mux
}

func (p *Proxy) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		p.writeError(w, http.StatusNotFound, "not_found", "Endpoint not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"name":    "MindBalancer",
		"version": "1.0.0",
		"status":  "running",
	})
}

func (p *Proxy) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

func (p *Proxy) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		p.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed")
		return
	}

	ctx := r.Context()
	requestID := uuid.New().String()
	startTime := time.Now()

	// Get username from auth header
	username := p.extractUsername(r)

	// Check rate limit
	if p.ratelimit != nil {
		result, err := p.ratelimit.Allow(ctx, username)
		if err != nil {
			p.writeError(w, http.StatusInternalServerError, "rate_limit_error", "Failed to check rate limit")
			return
		}
		if !result.Allowed {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))
			w.Header().Set("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
			p.writeError(w, http.StatusTooManyRequests, "rate_limit_exceeded", 
				fmt.Sprintf("Rate limit exceeded. Retry after %v", result.RetryAfter.Round(time.Second)))
			return
		}
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.RemainingRequests))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))
	}

	// Parse request
	var req openai.ChatCompletionRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, p.config.MaxRequestBodySize)).Decode(&req); err != nil {
		p.writeError(w, http.StatusBadRequest, "invalid_request", "Failed to parse request: "+err.Error())
		return
	}

	// Check for referee mode
	if req.RefereeMode != nil && req.RefereeMode.Enabled {
		if !p.config.RefereeEnabled {
			p.writeError(w, http.StatusBadRequest, "referee_disabled", "Referee mode is disabled on this server")
			return
		}
		p.handleRefereeModeChat(ctx, w, &req, requestID, username, startTime)
		return
	}

	// Extract prompt for routing
	var promptText string
	if len(req.Messages) > 0 {
		if content, ok := req.Messages[len(req.Messages)-1].Content.(string); ok {
			promptText = content
		}
	}

	// Route the request
	hostgroup, _ := p.router.RouteRequest(req.Model, promptText, username, 0)

	// Select a server
	server, err := p.balancer.SelectServer(ctx, hostgroup)
	if err != nil {
		p.writeError(w, http.StatusServiceUnavailable, "no_servers", "No servers available: "+err.Error())
		return
	}

	// Create provider with model-specific timeout
	timeout := p.config.GetModelTimeout(req.Model)
	prov := provider.New(*server, timeout)

	// Handle streaming vs non-streaming
	if req.Stream {
		p.handleStreamingChat(ctx, w, &req, prov, server, requestID, username, startTime)
	} else {
		p.handleNonStreamingChat(ctx, w, &req, prov, server, requestID, username, startTime)
	}
}

func (p *Proxy) handleNonStreamingChat(ctx context.Context, w http.ResponseWriter, req *openai.ChatCompletionRequest, prov provider.Provider, server *storage.Server, requestID, username string, startTime time.Time) {
	// Check if request is cacheable (temperature = 0 or nil means deterministic output)
	cacheable := p.cache != nil && (req.Temperature == nil || *req.Temperature == 0)
	var cacheKey string

	if cacheable {
		// Generate cache key from request
		cacheKey = cache.GenerateChatKey(req.Model, req.Messages, req.Temperature, req.MaxTokens, "")

		// Check cache first
		if cachedData, found := p.cache.Get(cacheKey); found {
			// Cache HIT!
			latency := time.Since(startTime)

			// Record cache hit in metrics
			if p.metrics != nil {
				p.metrics.RecordCacheHit()
			}

			// Log cache hit
			p.logRequest(requestID, username, "cache", req.Model, "/v1/chat/completions", 0, 0, latency, 0, http.StatusOK, "cache_hit", false)

			// Release server since we didn't use it
			p.balancer.ReleaseServer(server.Name, 0, true)

			// Return cached response
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Request-ID", requestID)
			w.Header().Set("X-Cache", "HIT")
			w.Write(cachedData)
			return
		}

		// Record cache miss
		if p.metrics != nil {
			p.metrics.RecordCacheMiss()
		}
	}

	// Use request deduplication for identical concurrent requests
	dedupeKey := cacheKey
	if dedupeKey == "" {
		// Generate key even for non-cacheable requests for deduplication
		dedupeKey = cache.GenerateChatKey(req.Model, req.Messages, req.Temperature, req.MaxTokens, "")
	}

	respBytes, err, wasDeduplicated := p.cache.DeduplicatedCall(dedupeKey, func() ([]byte, error) {
		return p.executeChatCompletion(ctx, req, prov, server, username, startTime)
	})

	latency := time.Since(startTime)

	if wasDeduplicated {
		// This request was deduplicated - another identical request got the result
		if p.metrics != nil {
			p.metrics.RecordDeduplication()
		}
		p.logRequest(requestID, username, "dedupe", req.Model, "/v1/chat/completions", 0, 0, latency, 0, http.StatusOK, "deduplicated", false)

		// Release server since we didn't use it
		p.balancer.ReleaseServer(server.Name, 0, true)
	}

	if err != nil {
		statusCode := http.StatusInternalServerError
		if provErr, ok := err.(*provider.ProviderError); ok {
			statusCode = provErr.StatusCode
		}
		p.writeError(w, statusCode, "provider_error", err.Error())
		return
	}

	// Store in cache if cacheable and not deduplicated (avoid double-caching)
	if cacheable && cacheKey != "" && !wasDeduplicated {
		p.cache.Set(cacheKey, respBytes, req.Model, "/v1/chat/completions")
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	if wasDeduplicated {
		w.Header().Set("X-Cache", "DEDUPE")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}
	w.Write(respBytes)
}

// executeChatCompletion performs the actual LLM call with retry logic.
func (p *Proxy) executeChatCompletion(ctx context.Context, req *openai.ChatCompletionRequest, prov provider.Provider, server *storage.Server, username string, startTime time.Time) ([]byte, error) {
	// Get model-specific timeout
	timeout := p.config.GetModelTimeout(req.Model)

	// Configure retry with exponential backoff
	retryCfg := retry.Config{
		MaxRetries:   p.config.MaxRetries,
		InitialDelay: p.config.RetryDelay(),
		MaxDelay:     p.config.RetryMaxDelay(),
		Multiplier:   p.config.RetryMultiplier,
		Jitter:       0.1,
	}

	var usedServer *storage.Server = server
	var usedProv provider.Provider = prov
	attemptCount := 0

	// Retry logic with failover to different servers
	result := retry.Do(ctx, retryCfg, func(ctx context.Context, attempt int) (*openai.ChatCompletionResponse, error) {
		attemptCount = attempt + 1

		// On retry, try to get a different server
		if attempt > 0 {
			p.balancer.ReleaseServer(usedServer.Name, time.Since(startTime), false)

			// Extract prompt for routing
			var promptText string
			if len(req.Messages) > 0 {
				if content, ok := req.Messages[len(req.Messages)-1].Content.(string); ok {
					promptText = content
				}
			}
			hostgroup, _ := p.router.RouteRequest(req.Model, promptText, username, 0)

			newServer, err := p.balancer.SelectServer(ctx, hostgroup)
			if err != nil {
				return nil, retry.NewRetryableError(err, false) // No more servers, don't retry
			}
			usedServer = newServer
			usedProv = provider.New(*newServer, timeout)
			log.Printf("[RETRY] Attempt %d: switched to server %s", attempt+1, newServer.Name)
		}

		response, err := usedProv.ChatCompletion(ctx, req)
		if err != nil {
			// Check if error is retryable
			if provErr, ok := err.(*provider.ProviderError); ok {
				if provErr.IsRetryable() {
					return nil, retry.NewRetryableError(err, true)
				}
			}
			return nil, retry.NewRetryableError(err, false)
		}
		return response, nil
	})

	resp := result.Value
	lastErr := result.Err
	latency := time.Since(startTime)
	success := lastErr == nil

	// Release final server back to pool
	p.balancer.ReleaseServer(usedServer.Name, latency, success)

	if lastErr != nil {
		p.logRequest("", username, usedServer.Name, req.Model, "/v1/chat/completions", 0, 0, latency, 0, http.StatusInternalServerError, fmt.Sprintf("after %d attempts: %s", attemptCount, lastErr.Error()), false)
		return nil, lastErr
	}

	// Record metrics
	var promptTokens, outputTokens int
	if resp.Usage != nil {
		promptTokens = resp.Usage.PromptTokens
		outputTokens = resp.Usage.CompletionTokens
	}

	// Record token usage for rate limiting
	if p.ratelimit != nil {
		p.ratelimit.RecordTokens(username, promptTokens+outputTokens)
	}

	p.logRequest("", username, usedServer.Name, req.Model, "/v1/chat/completions", promptTokens, outputTokens, latency, 0, http.StatusOK, "", false)

	if p.metrics != nil {
		p.metrics.RecordRequest(usedServer.Name, req.Model, true, latency, promptTokens, outputTokens)
		// Record cost
		p.metrics.RecordCost(usedServer.Name, req.Model, usedServer.ProviderType, promptTokens, outputTokens)
	}

	// Serialize response
	respBytes, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize response: %w", err)
	}

	return respBytes, nil
}

func (p *Proxy) handleStreamingChat(ctx context.Context, w http.ResponseWriter, req *openai.ChatCompletionRequest, prov provider.Provider, server *storage.Server, requestID, username string, startTime time.Time) {
	stream, err := prov.ChatCompletionStream(ctx, req)
	if err != nil {
		p.balancer.ReleaseServer(server.Name, time.Since(startTime), false)
		statusCode := http.StatusInternalServerError
		if provErr, ok := err.(*provider.ProviderError); ok {
			statusCode = provErr.StatusCode
		}
		p.writeError(w, statusCode, "provider_error", err.Error())
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Request-ID", requestID)

	flusher, ok := w.(http.Flusher)
	if !ok {
		p.balancer.ReleaseServer(server.Name, time.Since(startTime), false)
		p.writeError(w, http.StatusInternalServerError, "streaming_error", "Streaming not supported")
		return
	}

	var firstByteTime time.Duration
	firstByte := true

	for event := range stream {
		if event.Error != nil {
			latency := time.Since(startTime)
			p.balancer.ReleaseServer(server.Name, latency, false)
			p.logRequest(requestID, username, server.Name, req.Model, "/v1/chat/completions", 0, 0, latency, firstByteTime, http.StatusInternalServerError, event.Error.Error(), true)
			return
		}

		if event.Done {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
			break
		}

		if firstByte {
			firstByteTime = time.Since(startTime)
			firstByte = false
		}

		fmt.Fprintf(w, "data: %s\n\n", string(event.Data))
		flusher.Flush()
	}

	latency := time.Since(startTime)
	p.balancer.ReleaseServer(server.Name, latency, true)
	p.logRequest(requestID, username, server.Name, req.Model, "/v1/chat/completions", 0, 0, latency, firstByteTime, http.StatusOK, "", true)

	if p.metrics != nil {
		p.metrics.RecordRequest(server.Name, req.Model, true, latency, 0, 0)
	}
}

func (p *Proxy) handleRefereeModeChat(ctx context.Context, w http.ResponseWriter, req *openai.ChatCompletionRequest, requestID, username string, startTime time.Time) {
	// Referee mode doesn't support streaming
	if req.Stream {
		p.writeError(w, http.StatusBadRequest, "invalid_request", "Referee mode does not support streaming")
		return
	}

	// Execute referee mode
	resp, err := p.referee.Execute(ctx, req)

	latency := time.Since(startTime)

	if err != nil {
		p.writeError(w, http.StatusInternalServerError, "referee_error", err.Error())
		p.logRequest(requestID, username, "referee", req.Model, "/v1/chat/completions", 0, 0, latency, 0, http.StatusInternalServerError, err.Error(), false)

		if p.metrics != nil {
			p.metrics.RecordRefereeRequest(false, latency, 0, 0)
		}
		return
	}

	// Calculate tokens for metrics
	var promptTokens, outputTokens int
	if resp.Usage != nil {
		promptTokens = resp.Usage.PromptTokens
		outputTokens = resp.Usage.CompletionTokens
	}

	// Record token usage for rate limiting
	if p.ratelimit != nil {
		p.ratelimit.RecordTokens(username, promptTokens+outputTokens)
	}

	// Log the request
	refereeModel := req.RefereeMode.RefereeModel
	if refereeModel == "" {
		refereeModel = p.config.RefereeDefaultModel
	}
	p.logRequest(requestID, username, "referee:"+refereeModel, req.Model, "/v1/chat/completions", promptTokens, outputTokens, latency, 0, http.StatusOK, "", false)

	// Record metrics
	if p.metrics != nil {
		providersQueried := 0
		successfulResponses := 0
		if resp.RefereeInfo != nil {
			providersQueried = resp.RefereeInfo.ProvidersQueried
			successfulResponses = resp.RefereeInfo.SuccessfulResponses
		}
		p.metrics.RecordRefereeRequest(true, latency, providersQueried, successfulResponses)
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.Header().Set("X-Referee-Mode", "true")
	if resp.RefereeInfo != nil {
		w.Header().Set("X-Referee-Providers-Queried", strconv.Itoa(resp.RefereeInfo.ProvidersQueried))
		w.Header().Set("X-Referee-Successful-Responses", strconv.Itoa(resp.RefereeInfo.SuccessfulResponses))
	}
	json.NewEncoder(w).Encode(resp)
}

func (p *Proxy) handleCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		p.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed")
		return
	}

	ctx := r.Context()
	requestID := uuid.New().String()
	startTime := time.Now()

	var req openai.CompletionRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, p.config.MaxRequestBodySize)).Decode(&req); err != nil {
		p.writeError(w, http.StatusBadRequest, "invalid_request", "Failed to parse request: "+err.Error())
		return
	}

	username := p.extractUsername(r)

	var promptText string
	switch v := req.Prompt.(type) {
	case string:
		promptText = v
	case []string:
		if len(v) > 0 {
			promptText = v[0]
		}
	}

	hostgroup, _ := p.router.RouteRequest(req.Model, promptText, username, 0)
	server, err := p.balancer.SelectServer(ctx, hostgroup)
	if err != nil {
		p.writeError(w, http.StatusServiceUnavailable, "no_servers", "No servers available: "+err.Error())
		return
	}

	prov := provider.New(*server, p.config.RequestTimeout())
	resp, err := prov.Completion(ctx, &req)

	latency := time.Since(startTime)
	success := err == nil
	p.balancer.ReleaseServer(server.Name, latency, success)

	if err != nil {
		statusCode := http.StatusInternalServerError
		if provErr, ok := err.(*provider.ProviderError); ok {
			statusCode = provErr.StatusCode
		}
		p.writeError(w, statusCode, "provider_error", err.Error())
		p.logRequest(requestID, username, server.Name, req.Model, "/v1/completions", 0, 0, latency, 0, statusCode, err.Error(), false)
		return
	}

	var promptTokens, outputTokens int
	if resp.Usage != nil {
		promptTokens = resp.Usage.PromptTokens
		outputTokens = resp.Usage.CompletionTokens
	}

	p.logRequest(requestID, username, server.Name, req.Model, "/v1/completions", promptTokens, outputTokens, latency, 0, http.StatusOK, "", false)

	if p.metrics != nil {
		p.metrics.RecordRequest(server.Name, req.Model, true, latency, promptTokens, outputTokens)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	json.NewEncoder(w).Encode(resp)
}

func (p *Proxy) handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		p.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed")
		return
	}

	ctx := r.Context()
	requestID := uuid.New().String()
	startTime := time.Now()

	var req openai.EmbeddingRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, p.config.MaxRequestBodySize)).Decode(&req); err != nil {
		p.writeError(w, http.StatusBadRequest, "invalid_request", "Failed to parse request: "+err.Error())
		return
	}

	username := p.extractUsername(r)

	// Embeddings are always cacheable (deterministic output)
	cacheKey := cache.GenerateEmbeddingKey(req.Model, req.Input, req.Dimensions)

	// Check cache first
	if p.cache != nil {
		if cachedData, found := p.cache.Get(cacheKey); found {
			latency := time.Since(startTime)

			if p.metrics != nil {
				p.metrics.RecordCacheHit()
			}

			p.logRequest(requestID, username, "cache", req.Model, "/v1/embeddings", 0, 0, latency, 0, http.StatusOK, "cache_hit", false)

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Request-ID", requestID)
			w.Header().Set("X-Cache", "HIT")
			w.Write(cachedData)
			return
		}

		if p.metrics != nil {
			p.metrics.RecordCacheMiss()
		}
	}

	hostgroup, _ := p.router.RouteRequest(req.Model, "", username, 0)
	server, err := p.balancer.SelectServer(ctx, hostgroup)
	if err != nil {
		p.writeError(w, http.StatusServiceUnavailable, "no_servers", "No servers available: "+err.Error())
		return
	}

	prov := provider.New(*server, p.config.RequestTimeout())
	if !prov.SupportsEmbeddings() {
		p.balancer.ReleaseServer(server.Name, 0, false)
		p.writeError(w, http.StatusBadRequest, "unsupported", "Provider does not support embeddings")
		return
	}

	// Use deduplication for identical embedding requests
	respBytes, callErr, wasDeduplicated := p.cache.DeduplicatedCall(cacheKey, func() ([]byte, error) {
		resp, err := prov.Embedding(ctx, &req)
		if err != nil {
			return nil, err
		}
		return json.Marshal(resp)
	})

	latency := time.Since(startTime)

	if wasDeduplicated {
		if p.metrics != nil {
			p.metrics.RecordDeduplication()
		}
		p.logRequest(requestID, username, "dedupe", req.Model, "/v1/embeddings", 0, 0, latency, 0, http.StatusOK, "deduplicated", false)
	}

	success := callErr == nil
	p.balancer.ReleaseServer(server.Name, latency, success)

	if callErr != nil {
		statusCode := http.StatusInternalServerError
		if provErr, ok := callErr.(*provider.ProviderError); ok {
			statusCode = provErr.StatusCode
		}
		p.writeError(w, statusCode, "provider_error", callErr.Error())
		p.logRequest(requestID, username, server.Name, req.Model, "/v1/embeddings", 0, 0, latency, 0, statusCode, callErr.Error(), false)
		return
	}

	// Parse response to get token count for logging
	var resp openai.EmbeddingResponse
	json.Unmarshal(respBytes, &resp)

	var tokens int
	if resp.Usage != nil {
		tokens = resp.Usage.TotalTokens
	}

	if !wasDeduplicated {
		p.logRequest(requestID, username, server.Name, req.Model, "/v1/embeddings", tokens, 0, latency, 0, http.StatusOK, "", false)
	}

	if p.metrics != nil {
		p.metrics.RecordRequest(server.Name, req.Model, true, latency, tokens, 0)
	}

	// Store in cache (embeddings have long TTL)
	if p.cache != nil && !wasDeduplicated {
		p.cache.Set(cacheKey, respBytes, req.Model, "/v1/embeddings")
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	if wasDeduplicated {
		w.Header().Set("X-Cache", "DEDUPE")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}
	w.Write(respBytes)
}

func (p *Proxy) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		p.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed")
		return
	}

	ctx := r.Context()

	// Get all servers and aggregate models
	servers, err := p.storage.GetServers(ctx, nil)
	if err != nil {
		p.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to get servers")
		return
	}

	allModels := make(map[string]openai.Model)

	for _, srv := range servers {
		if srv.Status != storage.ServerStatusOnline {
			continue
		}

		prov := provider.New(srv, p.config.RequestTimeout())
		models, err := prov.ListModels(ctx)
		if err != nil {
			continue
		}

		for _, m := range models.Data {
			allModels[m.ID] = m
		}
	}

	modelList := make([]openai.Model, 0, len(allModels))
	for _, m := range allModels {
		modelList = append(modelList, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(openai.ModelList{
		Object: "list",
		Data:   modelList,
	})
}

func (p *Proxy) extractUsername(r *http.Request) string {
	// Try to extract from Authorization header
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		// Could map API keys to usernames
		return "default"
	}

	// Try X-User header
	if user := r.Header.Get("X-User"); user != "" {
		return user
	}

	return "anonymous"
}

func (p *Proxy) writeError(w http.ResponseWriter, statusCode int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(openai.ErrorResponse{
		Error: &openai.APIError{
			Message: message,
			Type:    errType,
		},
	})
}

func (p *Proxy) logRequest(requestID, username, serverName, model, endpoint string, promptTokens, outputTokens int, latency, firstByte time.Duration, statusCode int, errorMsg string, streaming bool) {
	log := &storage.RequestLog{
		RequestID:    requestID,
		Username:     username,
		ServerName:   serverName,
		Model:        model,
		Endpoint:     endpoint,
		PromptTokens: promptTokens,
		OutputTokens: outputTokens,
		TotalTokens:  promptTokens + outputTokens,
		LatencyMS:    latency.Milliseconds(),
		FirstByteMS:  firstByte.Milliseconds(),
		StatusCode:   statusCode,
		ErrorMessage: errorMsg,
		Streaming:    streaming,
		Timestamp:    time.Now(),
	}

	// Log asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		p.storage.InsertRequestLog(ctx, log)
	}()
}
