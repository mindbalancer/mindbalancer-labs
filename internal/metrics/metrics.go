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

	// Server metrics
	serverStatus        *prometheus.GaugeVec
	circuitBreakerState *prometheus.GaugeVec
	serverLatency       *prometheus.GaugeVec

	// Connection metrics
	connectionsActive *prometheus.GaugeVec
	connectionsIdle   *prometheus.GaugeVec

	// Error metrics
	errorsTotal *prometheus.CounterVec

	registry *prometheus.Registry
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
