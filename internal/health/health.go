// Package health provides health checking functionality for AI servers.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/mindbalancer/mindbalancer/internal/storage"
)

// Status represents the health status of a server.
type Status struct {
	Healthy       bool
	LastCheck     time.Time
	LastSuccess   time.Time
	LastError     string
	Latency       time.Duration
	ConsecFails   int
	ConsecSuccess int
}

// Checker performs health checks on AI servers.
type Checker struct {
	mu       sync.RWMutex
	client   *http.Client
	interval time.Duration
	timeout  time.Duration
	status   map[string]*Status
	stopCh   chan struct{}
	storage  *storage.Storage
	onUpdate func(name string, healthy bool)
}

// NewChecker creates a new health checker.
func NewChecker(store *storage.Storage, interval, timeout time.Duration) *Checker {
	return &Checker{
		client: &http.Client{
			Timeout: timeout,
		},
		interval: interval,
		timeout:  timeout,
		status:   make(map[string]*Status),
		stopCh:   make(chan struct{}),
		storage:  store,
	}
}

// SetUpdateCallback sets a callback for when a server's health status changes.
func (c *Checker) SetUpdateCallback(fn func(name string, healthy bool)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onUpdate = fn
}

// Start begins the health check loop.
func (c *Checker) Start(ctx context.Context) {
	go c.run(ctx)
}

// Stop stops the health check loop.
func (c *Checker) Stop() {
	close(c.stopCh)
}

func (c *Checker) run(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	// Run immediately
	c.checkAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.checkAll(ctx)
		}
	}
}

func (c *Checker) checkAll(ctx context.Context) {
	servers, err := c.storage.GetServers(ctx, nil)
	if err != nil {
		return
	}

	var wg sync.WaitGroup
	for _, srv := range servers {
		if srv.Status == storage.ServerStatusOffline {
			continue
		}

		wg.Add(1)
		go func(s storage.Server) {
			defer wg.Done()
			c.checkServer(ctx, s)
		}(srv)
	}
	wg.Wait()
}

func (c *Checker) checkServer(ctx context.Context, srv storage.Server) {
	start := time.Now()
	healthy := false
	var errMsg string

	// Perform the health check based on provider type
	switch srv.ProviderType {
	case "openai", "azure":
		healthy, errMsg = c.checkOpenAI(ctx, srv)
	case "anthropic":
		healthy, errMsg = c.checkAnthropic(ctx, srv)
	case "ollama":
		healthy, errMsg = c.checkOllama(ctx, srv)
	default:
		healthy, errMsg = c.checkGeneric(ctx, srv)
	}

	latency := time.Since(start)

	c.mu.Lock()
	status, ok := c.status[srv.Name]
	if !ok {
		status = &Status{}
		c.status[srv.Name] = status
	}

	prevHealthy := status.Healthy
	status.Healthy = healthy
	status.LastCheck = time.Now()
	status.Latency = latency

	if healthy {
		status.LastSuccess = time.Now()
		status.LastError = ""
		status.ConsecFails = 0
		status.ConsecSuccess++
	} else {
		status.LastError = errMsg
		status.ConsecFails++
		status.ConsecSuccess = 0
	}

	callback := c.onUpdate
	c.mu.Unlock()

	// Notify if status changed
	if callback != nil && prevHealthy != healthy {
		callback(srv.Name, healthy)
	}
}

func (c *Checker) checkOpenAI(ctx context.Context, srv storage.Server) (bool, string) {
	// Try to list models as health check
	req, err := http.NewRequestWithContext(ctx, "GET", srv.Endpoint+"/v1/models", nil)
	if err != nil {
		return false, err.Error()
	}

	if srv.APIKeyEncrypted != "" {
		req.Header.Set("Authorization", "Bearer "+srv.APIKeyEncrypted)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, ""
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return false, fmt.Sprintf("status %d: %s", resp.StatusCode, string(body))
}

func (c *Checker) checkAnthropic(ctx context.Context, srv storage.Server) (bool, string) {
	// Anthropic doesn't have a models endpoint, so we use a lightweight ping
	req, err := http.NewRequestWithContext(ctx, "GET", srv.Endpoint, nil)
	if err != nil {
		return false, err.Error()
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	// Any response is considered healthy for basic connectivity
	return resp.StatusCode < 500, ""
}

func (c *Checker) checkOllama(ctx context.Context, srv storage.Server) (bool, string) {
	// Ollama has a /api/tags endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", srv.Endpoint+"/api/tags", nil)
	if err != nil {
		return false, err.Error()
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, ""
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return false, fmt.Sprintf("status %d: %s", resp.StatusCode, string(body))
}

func (c *Checker) checkGeneric(ctx context.Context, srv storage.Server) (bool, string) {
	// Generic check: try /v1/models or just the endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", srv.Endpoint+"/v1/models", nil)
	if err != nil {
		return false, err.Error()
	}

	if srv.APIKeyEncrypted != "" {
		req.Header.Set("Authorization", "Bearer "+srv.APIKeyEncrypted)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		// Try just the base endpoint
		req2, _ := http.NewRequestWithContext(ctx, "GET", srv.Endpoint, nil)
		resp2, err2 := c.client.Do(req2)
		if err2 != nil {
			return false, err2.Error()
		}
		defer resp2.Body.Close()
		return resp2.StatusCode < 500, ""
	}
	defer resp.Body.Close()

	return resp.StatusCode < 500, ""
}

// GetStatus returns the health status for a server.
func (c *Checker) GetStatus(name string) *Status {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if s, ok := c.status[name]; ok {
		// Return a copy
		copy := *s
		return &copy
	}
	return nil
}

// GetAllStatus returns health status for all servers.
func (c *Checker) GetAllStatus() map[string]Status {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]Status, len(c.status))
	for k, v := range c.status {
		result[k] = *v
	}
	return result
}

// IsHealthy checks if a server is healthy.
func (c *Checker) IsHealthy(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if s, ok := c.status[name]; ok {
		return s.Healthy
	}
	// Unknown servers are considered healthy (new servers)
	return true
}

// MarkUnhealthy manually marks a server as unhealthy.
func (c *Checker) MarkUnhealthy(name string, reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if s, ok := c.status[name]; ok {
		s.Healthy = false
		s.LastError = reason
		s.ConsecFails++
		s.ConsecSuccess = 0
	} else {
		c.status[name] = &Status{
			Healthy:     false,
			LastCheck:   time.Now(),
			LastError:   reason,
			ConsecFails: 1,
		}
	}
}

// MarkHealthy manually marks a server as healthy.
func (c *Checker) MarkHealthy(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if s, ok := c.status[name]; ok {
		s.Healthy = true
		s.LastSuccess = time.Now()
		s.LastError = ""
		s.ConsecFails = 0
		s.ConsecSuccess++
	} else {
		c.status[name] = &Status{
			Healthy:       true,
			LastCheck:     time.Now(),
			LastSuccess:   time.Now(),
			ConsecSuccess: 1,
		}
	}
}

// ToJSON returns health status as JSON.
func (c *Checker) ToJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return json.Marshal(c.status)
}
