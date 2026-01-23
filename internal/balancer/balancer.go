// Package balancer provides load balancing functionality for AI servers.
package balancer

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mindbalancer/mindbalancer/internal/circuit"
	"github.com/mindbalancer/mindbalancer/internal/health"
	"github.com/mindbalancer/mindbalancer/internal/storage"
)

var (
	ErrNoServersAvailable = errors.New("no servers available")
	ErrCircuitOpen        = errors.New("circuit breaker is open")
	ErrServerNotFound     = errors.New("server not found")
)

// Algorithm represents a load balancing algorithm.
type Algorithm string

const (
	AlgorithmWeightedRoundRobin Algorithm = "weighted_round_robin"
	AlgorithmLeastConnections   Algorithm = "least_connections"
	AlgorithmRandom             Algorithm = "random"
	AlgorithmLatencyBased       Algorithm = "latency_based"
)

// ServerState tracks the state of a server for load balancing.
type ServerState struct {
	Server      storage.Server
	Connections int64
	TotalReqs   int64
	Errors      int64
	AvgLatency  time.Duration
	LastUsed    time.Time
}

// Balancer implements load balancing across AI servers.
type Balancer struct {
	mu          sync.RWMutex
	storage     *storage.Storage
	health      *health.Checker
	circuits    *circuit.Manager
	algorithm   Algorithm
	servers     map[string]*ServerState
	hostgroups  map[int][]*ServerState
	rrCounters  map[int]*uint64 // Round-robin counters per hostgroup
}

// NewBalancer creates a new load balancer.
func NewBalancer(store *storage.Storage, healthChecker *health.Checker, circuitMgr *circuit.Manager) *Balancer {
	b := &Balancer{
		storage:    store,
		health:     healthChecker,
		circuits:   circuitMgr,
		algorithm:  AlgorithmWeightedRoundRobin,
		servers:    make(map[string]*ServerState),
		hostgroups: make(map[int][]*ServerState),
		rrCounters: make(map[int]*uint64),
	}

	// Register health callback
	if healthChecker != nil {
		healthChecker.SetUpdateCallback(b.onHealthChange)
	}

	return b
}

// LoadServers loads servers from storage into the balancer.
func (b *Balancer) LoadServers(ctx context.Context) error {
	servers, err := b.storage.GetServers(ctx, nil)
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Reset state
	b.servers = make(map[string]*ServerState)
	b.hostgroups = make(map[int][]*ServerState)

	for _, srv := range servers {
		if srv.Status == storage.ServerStatusOffline {
			continue
		}

		state := &ServerState{
			Server: srv,
		}
		b.servers[srv.Name] = state

		// Add to hostgroup
		b.hostgroups[srv.Hostgroup] = append(b.hostgroups[srv.Hostgroup], state)

		// Initialize round-robin counter if needed
		if _, ok := b.rrCounters[srv.Hostgroup]; !ok {
			counter := uint64(0)
			b.rrCounters[srv.Hostgroup] = &counter
		}
	}

	return nil
}

// SetAlgorithm sets the load balancing algorithm.
func (b *Balancer) SetAlgorithm(alg Algorithm) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.algorithm = alg
}

// SelectServer selects a server for a request.
func (b *Balancer) SelectServer(ctx context.Context, hostgroup int) (*storage.Server, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	servers := b.hostgroups[hostgroup]
	if len(servers) == 0 {
		// Fall back to default hostgroup
		servers = b.hostgroups[0]
	}

	if len(servers) == 0 {
		return nil, ErrNoServersAvailable
	}

	// Filter healthy servers with open circuits
	var available []*ServerState
	for _, s := range servers {
		// Check health
		if b.health != nil && !b.health.IsHealthy(s.Server.Name) {
			continue
		}

		// Check circuit breaker
		if b.circuits != nil {
			cb := b.circuits.Get(s.Server.Name)
			if !cb.Allow() {
				continue
			}
		}

		// Check if server is online
		if s.Server.Status != storage.ServerStatusOnline {
			continue
		}

		available = append(available, s)
	}

	if len(available) == 0 {
		return nil, ErrNoServersAvailable
	}

	// Select based on algorithm
	var selected *ServerState
	switch b.algorithm {
	case AlgorithmWeightedRoundRobin:
		selected = b.selectWeightedRoundRobin(available, hostgroup)
	case AlgorithmLeastConnections:
		selected = b.selectLeastConnections(available)
	case AlgorithmRandom:
		selected = b.selectRandom(available)
	case AlgorithmLatencyBased:
		selected = b.selectLatencyBased(available)
	default:
		selected = b.selectWeightedRoundRobin(available, hostgroup)
	}

	if selected == nil {
		return nil, ErrNoServersAvailable
	}

	// Increment connection count
	atomic.AddInt64(&selected.Connections, 1)
	atomic.AddInt64(&selected.TotalReqs, 1)

	return &selected.Server, nil
}

func (b *Balancer) selectWeightedRoundRobin(servers []*ServerState, hostgroup int) *ServerState {
	if len(servers) == 0 {
		return nil
	}

	// Calculate total weight
	totalWeight := 0
	for _, s := range servers {
		totalWeight += s.Server.Weight
	}

	if totalWeight == 0 {
		// Equal weight fallback
		counter := b.rrCounters[hostgroup]
		idx := atomic.AddUint64(counter, 1) % uint64(len(servers))
		return servers[idx]
	}

	// Get counter and select
	counter := b.rrCounters[hostgroup]
	pos := int(atomic.AddUint64(counter, 1) % uint64(totalWeight))

	cumulative := 0
	for _, s := range servers {
		cumulative += s.Server.Weight
		if pos < cumulative {
			return s
		}
	}

	return servers[0]
}

func (b *Balancer) selectLeastConnections(servers []*ServerState) *ServerState {
	if len(servers) == 0 {
		return nil
	}

	var selected *ServerState
	minConns := int64(^uint64(0) >> 1) // Max int64

	for _, s := range servers {
		conns := atomic.LoadInt64(&s.Connections)
		// Weight-adjusted: divide by weight to favor higher-weight servers
		adjusted := conns
		if s.Server.Weight > 0 {
			adjusted = conns / int64(s.Server.Weight)
		}
		if adjusted < minConns {
			minConns = adjusted
			selected = s
		}
	}

	return selected
}

func (b *Balancer) selectRandom(servers []*ServerState) *ServerState {
	if len(servers) == 0 {
		return nil
	}

	// Weighted random
	totalWeight := 0
	for _, s := range servers {
		totalWeight += s.Server.Weight
	}

	if totalWeight == 0 {
		return servers[rand.Intn(len(servers))]
	}

	r := rand.Intn(totalWeight)
	cumulative := 0
	for _, s := range servers {
		cumulative += s.Server.Weight
		if r < cumulative {
			return s
		}
	}

	return servers[0]
}

func (b *Balancer) selectLatencyBased(servers []*ServerState) *ServerState {
	if len(servers) == 0 {
		return nil
	}

	var selected *ServerState
	minLatency := time.Duration(^uint64(0) >> 1)

	for _, s := range servers {
		latency := s.AvgLatency
		if latency == 0 {
			// Unknown latency, give benefit of doubt
			latency = time.Millisecond * 100
		}
		// Weight-adjusted
		if s.Server.Weight > 0 {
			latency = latency / time.Duration(s.Server.Weight)
		}
		if latency < minLatency {
			minLatency = latency
			selected = s
		}
	}

	return selected
}

// ReleaseServer releases a server after a request completes.
func (b *Balancer) ReleaseServer(name string, latency time.Duration, success bool) {
	b.mu.RLock()
	state, ok := b.servers[name]
	b.mu.RUnlock()

	if !ok {
		return
	}

	// Decrement connection count
	atomic.AddInt64(&state.Connections, -1)

	if !success {
		atomic.AddInt64(&state.Errors, 1)
		if b.circuits != nil {
			b.circuits.Get(name).RecordFailure()
		}
	} else {
		if b.circuits != nil {
			b.circuits.Get(name).RecordSuccess()
		}
	}

	// Update latency (exponential moving average)
	b.mu.Lock()
	if state.AvgLatency == 0 {
		state.AvgLatency = latency
	} else {
		// EMA with alpha = 0.3
		state.AvgLatency = time.Duration(float64(state.AvgLatency)*0.7 + float64(latency)*0.3)
	}
	state.LastUsed = time.Now()
	b.mu.Unlock()
}

// GetServer returns a server by name.
func (b *Balancer) GetServer(name string) (*storage.Server, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if state, ok := b.servers[name]; ok {
		return &state.Server, nil
	}
	return nil, ErrServerNotFound
}

// GetServerState returns the state of a server.
func (b *Balancer) GetServerState(name string) (*ServerState, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if state, ok := b.servers[name]; ok {
		// Return a copy
		copy := *state
		return &copy, nil
	}
	return nil, ErrServerNotFound
}

// GetAllServerStates returns all server states.
func (b *Balancer) GetAllServerStates() []ServerState {
	b.mu.RLock()
	defer b.mu.RUnlock()

	states := make([]ServerState, 0, len(b.servers))
	for _, s := range b.servers {
		states = append(states, *s)
	}
	return states
}

// GetHostgroupServers returns all servers in a hostgroup.
func (b *Balancer) GetHostgroupServers(hostgroup int) []storage.Server {
	b.mu.RLock()
	defer b.mu.RUnlock()

	states := b.hostgroups[hostgroup]
	servers := make([]storage.Server, len(states))
	for i, s := range states {
		servers[i] = s.Server
	}
	return servers
}

// onHealthChange handles health status changes.
func (b *Balancer) onHealthChange(name string, healthy bool) {
	// This could trigger rebalancing or logging
	// For now, the health check is done in SelectServer
}

// Stats returns balancer statistics.
func (b *Balancer) Stats() BalancerStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	stats := BalancerStats{
		Algorithm:     string(b.algorithm),
		TotalServers:  len(b.servers),
		ServerStats:   make([]ServerStats, 0, len(b.servers)),
		HostgroupSizes: make(map[int]int),
	}

	var healthyCount int
	for _, s := range b.servers {
		if b.health == nil || b.health.IsHealthy(s.Server.Name) {
			healthyCount++
		}

		ss := ServerStats{
			Name:        s.Server.Name,
			Hostgroup:   s.Server.Hostgroup,
			Weight:      s.Server.Weight,
			Connections: atomic.LoadInt64(&s.Connections),
			TotalReqs:   atomic.LoadInt64(&s.TotalReqs),
			Errors:      atomic.LoadInt64(&s.Errors),
			AvgLatency:  s.AvgLatency,
		}
		stats.ServerStats = append(stats.ServerStats, ss)
	}
	stats.HealthyServers = healthyCount

	for hg, servers := range b.hostgroups {
		stats.HostgroupSizes[hg] = len(servers)
	}

	return stats
}

// BalancerStats holds balancer statistics.
type BalancerStats struct {
	Algorithm      string
	TotalServers   int
	HealthyServers int
	ServerStats    []ServerStats
	HostgroupSizes map[int]int
}

// ServerStats holds per-server statistics.
type ServerStats struct {
	Name        string
	Hostgroup   int
	Weight      int
	Connections int64
	TotalReqs   int64
	Errors      int64
	AvgLatency  time.Duration
}
