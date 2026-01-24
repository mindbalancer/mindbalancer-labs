// Package pool provides HTTP client connection pooling.
package pool

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"
)

// Config holds pool configuration.
type Config struct {
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int
	IdleConnTimeout     time.Duration
	TLSHandshakeTimeout time.Duration
	ResponseTimeout     time.Duration
	KeepAlive           time.Duration
}

// DefaultConfig returns sensible defaults for HTTP client pool.
func DefaultConfig() Config {
	return Config{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 20,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ResponseTimeout:     120 * time.Second,
		KeepAlive:           30 * time.Second,
	}
}

// Pool manages a pool of HTTP clients.
type Pool struct {
	mu      sync.RWMutex
	cfg     Config
	client  *http.Client
	clients map[string]*http.Client // Per-host clients if needed
}

// NewPool creates a new HTTP client pool.
func NewPool(cfg Config) *Pool {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: cfg.KeepAlive,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.ResponseTimeout,
	}

	return &Pool{
		cfg:     cfg,
		client:  client,
		clients: make(map[string]*http.Client),
	}
}

// GetClient returns an HTTP client for making requests.
func (p *Pool) GetClient() *http.Client {
	return p.client
}

// GetClientWithTimeout returns an HTTP client with a specific timeout.
func (p *Pool) GetClientWithTimeout(timeout time.Duration) *http.Client {
	p.mu.RLock()
	key := timeout.String()
	if client, ok := p.clients[key]; ok {
		p.mu.RUnlock()
		return client
	}
	p.mu.RUnlock()

	// Create a new client with this timeout
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if client, ok := p.clients[key]; ok {
		return client
	}

	// Share the transport from the main client
	client := &http.Client{
		Transport: p.client.Transport,
		Timeout:   timeout,
	}
	p.clients[key] = client

	return client
}

// CloseIdleConnections closes idle connections.
func (p *Pool) CloseIdleConnections() {
	if transport, ok := p.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}

// Stats returns pool statistics.
type Stats struct {
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int
	IdleConnTimeout     time.Duration
}

// Stats returns the pool configuration as stats.
func (p *Pool) Stats() Stats {
	return Stats{
		MaxIdleConns:        p.cfg.MaxIdleConns,
		MaxIdleConnsPerHost: p.cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:     p.cfg.MaxConnsPerHost,
		IdleConnTimeout:     p.cfg.IdleConnTimeout,
	}
}

// Global pool instance
var (
	globalPool     *Pool
	globalPoolOnce sync.Once
)

// InitGlobalPool initializes the global pool.
func InitGlobalPool(cfg Config) {
	globalPoolOnce = sync.Once{} // Reset for testing
	globalPoolOnce.Do(func() {
		globalPool = NewPool(cfg)
	})
}

// GlobalPool returns the global pool instance.
func GlobalPool() *Pool {
	globalPoolOnce.Do(func() {
		globalPool = NewPool(DefaultConfig())
	})
	return globalPool
}

// GetGlobalClient returns a client from the global pool.
func GetGlobalClient() *http.Client {
	return GlobalPool().GetClient()
}

// GetGlobalClientWithTimeout returns a client with specific timeout from the global pool.
func GetGlobalClientWithTimeout(timeout time.Duration) *http.Client {
	return GlobalPool().GetClientWithTimeout(timeout)
}
