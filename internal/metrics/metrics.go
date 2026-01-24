// Package metrics provides Prometheus metrics collection.
package metrics

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector collects and exposes metrics.
type Collector struct {
	mu sync.RWMutex

	// Request metrics
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight *prometheus.GaugeVec

	// Token metrics
	tokensTotal *prometheus.CounterVec

	// Cost metrics
	costTotal *prometheus.CounterVec

	// Server metrics
	serverStatus        *prometheus.GaugeVec
	circuitBreakerState *prometheus.GaugeVec
	serverLatency       *prometheus.GaugeVec

	// Connection metrics
	connectionsActive *prometheus.GaugeVec
	connectionsIdle   *prometheus.GaugeVec

	// Error metrics
	errorsTotal *prometheus.CounterVec

	// Cache metrics
	cacheHits   prometheus.Counter
	cacheMisses prometheus.Counter

	registry *prometheus.Registry

	// Model pricing (USD per 1K tokens)
	pricing map[string]ModelPricing
}

// ModelPricing holds pricing information for a model.
type ModelPricing struct {
	InputPer1K  float64
	OutputPer1K float64
}

// NewCollector creates a new metrics collector.
func NewCollector() *Collector {
	c := &Collector{
		registry: prometheus.NewRegistry(),
	}

	// Request metrics
	c.requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mindbalancer_requests_total",
			Help: "Total number of requests processed",
		},
		[]string{"server", "model", "status"},
	)

	c.requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mindbalancer_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120},
		},
		[]string{"server", "model"},
	)

	c.requestsInFlight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mindbalancer_requests_in_flight",
			Help: "Number of requests currently being processed",
		},
		[]string{"server"},
	)

	// Token metrics
	c.tokensTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mindbalancer_tokens_total",
			Help: "Total number of tokens processed",
		},
		[]string{"server", "direction"}, // direction: input, output
	)

	// Cost metrics
	c.costTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mindbalancer_cost_usd_total",
			Help: "Total estimated cost in USD",
		},
		[]string{"server", "model", "provider"},
	)

	// Cache metrics
	c.cacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "mindbalancer_cache_hits_total",
			Help: "Total number of cache hits",
		},
	)
	c.cacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "mindbalancer_cache_misses_total",
			Help: "Total number of cache misses",
		},
	)

	// Initialize pricing (USD per 1K tokens) - Updated Jan 2025
	c.pricing = map[string]ModelPricing{
		// OpenAI
		"gpt-4":               {InputPer1K: 0.03, OutputPer1K: 0.06},
		"gpt-4-turbo":         {InputPer1K: 0.01, OutputPer1K: 0.03},
		"gpt-4o":              {InputPer1K: 0.005, OutputPer1K: 0.015},
		"gpt-4o-mini":         {InputPer1K: 0.00015, OutputPer1K: 0.0006},
		"gpt-3.5-turbo":       {InputPer1K: 0.0005, OutputPer1K: 0.0015},
		"o1":                  {InputPer1K: 0.015, OutputPer1K: 0.06},
		"o1-mini":             {InputPer1K: 0.003, OutputPer1K: 0.012},
		"o1-preview":          {InputPer1K: 0.015, OutputPer1K: 0.06},
		// Anthropic
		"claude-3-5-sonnet-20241022": {InputPer1K: 0.003, OutputPer1K: 0.015},
		"claude-3-5-haiku-20241022":  {InputPer1K: 0.0008, OutputPer1K: 0.004},
		"claude-3-opus-20240229":     {InputPer1K: 0.015, OutputPer1K: 0.075},
		"claude-3-sonnet-20240229":   {InputPer1K: 0.003, OutputPer1K: 0.015},
		"claude-3-haiku-20240307":    {InputPer1K: 0.00025, OutputPer1K: 0.00125},
		// Groq
		"llama-3.1-70b-versatile": {InputPer1K: 0.00059, OutputPer1K: 0.00079},
		"llama-3.1-8b-instant":    {InputPer1K: 0.00005, OutputPer1K: 0.00008},
		"mixtral-8x7b-32768":      {InputPer1K: 0.00024, OutputPer1K: 0.00024},
		// Google
		"gemini-1.5-pro":   {InputPer1K: 0.00125, OutputPer1K: 0.005},
		"gemini-1.5-flash": {InputPer1K: 0.000075, OutputPer1K: 0.0003},
		"gemini-2.0-flash": {InputPer1K: 0.0001, OutputPer1K: 0.0004},
	}

	// Server metrics
	c.serverStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mindbalancer_server_status",
			Help: "Server status (1=healthy, 0=unhealthy)",
		},
		[]string{"server", "provider"},
	)

	c.circuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mindbalancer_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"server"},
	)

	c.serverLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mindbalancer_server_latency_seconds",
			Help: "Average server latency in seconds",
		},
		[]string{"server"},
	)

	// Connection metrics
	c.connectionsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mindbalancer_connections_active",
			Help: "Number of active connections",
		},
		[]string{"server"},
	)

	c.connectionsIdle = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mindbalancer_connections_idle",
			Help: "Number of idle connections",
		},
		[]string{"server"},
	)

	// Error metrics
	c.errorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mindbalancer_errors_total",
			Help: "Total number of errors",
		},
		[]string{"server", "type"},
	)

	// Register all metrics
	c.registry.MustRegister(
		c.requestsTotal,
		c.requestDuration,
		c.requestsInFlight,
		c.tokensTotal,
		c.costTotal,
		c.cacheHits,
		c.cacheMisses,
		c.serverStatus,
		c.circuitBreakerState,
		c.serverLatency,
		c.connectionsActive,
		c.connectionsIdle,
		c.errorsTotal,
	)

	// Register default Go metrics
	c.registry.MustRegister(prometheus.NewGoCollector())
	c.registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	return c
}

// Handler returns the HTTP handler for metrics endpoint.
func (c *Collector) Handler() http.Handler {
	return promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// RecordRequest records a request.
func (c *Collector) RecordRequest(server, model string, success bool, duration time.Duration, promptTokens, outputTokens int) {
	status := "success"
	if !success {
		status = "error"
	}

	c.requestsTotal.WithLabelValues(server, model, status).Inc()
	c.requestDuration.WithLabelValues(server, model).Observe(duration.Seconds())

	if promptTokens > 0 {
		c.tokensTotal.WithLabelValues(server, "input").Add(float64(promptTokens))
	}
	if outputTokens > 0 {
		c.tokensTotal.WithLabelValues(server, "output").Add(float64(outputTokens))
	}
}

// RecordError records an error.
func (c *Collector) RecordError(server, errorType string) {
	c.errorsTotal.WithLabelValues(server, errorType).Inc()
}

// SetServerStatus sets the server status.
func (c *Collector) SetServerStatus(server, provider string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	c.serverStatus.WithLabelValues(server, provider).Set(value)
}

// SetCircuitBreakerState sets the circuit breaker state.
func (c *Collector) SetCircuitBreakerState(server string, state int) {
	c.circuitBreakerState.WithLabelValues(server).Set(float64(state))
}

// SetServerLatency sets the server latency.
func (c *Collector) SetServerLatency(server string, latency time.Duration) {
	c.serverLatency.WithLabelValues(server).Set(latency.Seconds())
}

// SetConnectionsActive sets the active connections count.
func (c *Collector) SetConnectionsActive(server string, count int) {
	c.connectionsActive.WithLabelValues(server).Set(float64(count))
}

// SetConnectionsIdle sets the idle connections count.
func (c *Collector) SetConnectionsIdle(server string, count int) {
	c.connectionsIdle.WithLabelValues(server).Set(float64(count))
}

// IncrementInFlight increments the in-flight request count.
func (c *Collector) IncrementInFlight(server string) {
	c.requestsInFlight.WithLabelValues(server).Inc()
}

// DecrementInFlight decrements the in-flight request count.
func (c *Collector) DecrementInFlight(server string) {
	c.requestsInFlight.WithLabelValues(server).Dec()
}

// Registry returns the Prometheus registry.
func (c *Collector) Registry() *prometheus.Registry {
	return c.registry
}

// RecordCost records the cost for a request.
func (c *Collector) RecordCost(server, model, provider string, inputTokens, outputTokens int) {
	cost := c.CalculateCost(model, inputTokens, outputTokens)
	if cost > 0 {
		c.costTotal.WithLabelValues(server, model, provider).Add(cost)
	}
}

// CalculateCost calculates the estimated cost for a request.
func (c *Collector) CalculateCost(model string, inputTokens, outputTokens int) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	pricing, ok := c.pricing[model]
	if !ok {
		// Try to match partial model name (e.g., "gpt-4o-2024-08-06" -> "gpt-4o")
		for name, p := range c.pricing {
			if len(model) >= len(name) && model[:len(name)] == name {
				pricing = p
				ok = true
				break
			}
		}
	}

	if !ok {
		return 0 // Unknown model, no pricing available
	}

	inputCost := float64(inputTokens) / 1000.0 * pricing.InputPer1K
	outputCost := float64(outputTokens) / 1000.0 * pricing.OutputPer1K
	return inputCost + outputCost
}

// SetModelPricing updates pricing for a model.
func (c *Collector) SetModelPricing(model string, inputPer1K, outputPer1K float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pricing[model] = ModelPricing{
		InputPer1K:  inputPer1K,
		OutputPer1K: outputPer1K,
	}
}

// GetModelPricing returns pricing for a model.
func (c *Collector) GetModelPricing(model string) (ModelPricing, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	pricing, ok := c.pricing[model]
	return pricing, ok
}

// GetAllPricing returns all model pricing.
func (c *Collector) GetAllPricing() map[string]ModelPricing {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	result := make(map[string]ModelPricing, len(c.pricing))
	for k, v := range c.pricing {
		result[k] = v
	}
	return result
}

// RecordCacheHit records a cache hit.
func (c *Collector) RecordCacheHit() {
	c.cacheHits.Inc()
}

// RecordCacheMiss records a cache miss.
func (c *Collector) RecordCacheMiss() {
	c.cacheMisses.Inc()
}

// CostSummary holds cost summary information.
type CostSummary struct {
	TotalCost     float64
	CostByModel   map[string]float64
	CostByServer  map[string]float64
	TotalTokens   int
	InputTokens   int
	OutputTokens  int
}
