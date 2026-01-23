// Package circuit provides circuit breaker functionality.
package circuit

import (
	"sync"
	"time"
)

// State represents the state of a circuit breaker.
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Breaker implements the circuit breaker pattern.
type Breaker struct {
	mu sync.RWMutex

	name             string
	state            State
	failures         int
	successes        int
	threshold        int           // failures to open
	successThreshold int           // successes to close from half-open
	timeout          time.Duration // time before half-open
	lastFailure      time.Time
	lastStateChange  time.Time
}

// NewBreaker creates a new circuit breaker.
func NewBreaker(name string, threshold, successThreshold int, timeout time.Duration) *Breaker {
	return &Breaker{
		name:             name,
		state:            StateClosed,
		threshold:        threshold,
		successThreshold: successThreshold,
		timeout:          timeout,
		lastStateChange:  time.Now(),
	}
}

// Allow checks if a request should be allowed through.
func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(b.lastFailure) >= b.timeout {
			b.state = StateHalfOpen
			b.lastStateChange = time.Now()
			b.successes = 0
			return true
		}
		return false

	case StateHalfOpen:
		return true

	default:
		return false
	}
}

// RecordSuccess records a successful request.
func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		b.failures = 0

	case StateHalfOpen:
		b.successes++
		if b.successes >= b.successThreshold {
			b.state = StateClosed
			b.lastStateChange = time.Now()
			b.failures = 0
			b.successes = 0
		}
	}
}

// RecordFailure records a failed request.
func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.failures++
	b.lastFailure = time.Now()

	switch b.state {
	case StateClosed:
		if b.failures >= b.threshold {
			b.state = StateOpen
			b.lastStateChange = time.Now()
		}

	case StateHalfOpen:
		b.state = StateOpen
		b.lastStateChange = time.Now()
	}
}

// State returns the current state.
func (b *Breaker) State() State {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// Stats returns circuit breaker statistics.
func (b *Breaker) Stats() BreakerStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return BreakerStats{
		Name:            b.name,
		State:           b.state.String(),
		Failures:        b.failures,
		Successes:       b.successes,
		LastFailure:     b.lastFailure,
		LastStateChange: b.lastStateChange,
	}
}

// Reset resets the circuit breaker to closed state.
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.state = StateClosed
	b.failures = 0
	b.successes = 0
	b.lastStateChange = time.Now()
}

// BreakerStats holds circuit breaker statistics.
type BreakerStats struct {
	Name            string
	State           string
	Failures        int
	Successes       int
	LastFailure     time.Time
	LastStateChange time.Time
}

// Manager manages multiple circuit breakers.
type Manager struct {
	mu               sync.RWMutex
	breakers         map[string]*Breaker
	threshold        int
	successThreshold int
	timeout          time.Duration
}

// NewManager creates a new circuit breaker manager.
func NewManager(threshold, successThreshold int, timeout time.Duration) *Manager {
	return &Manager{
		breakers:         make(map[string]*Breaker),
		threshold:        threshold,
		successThreshold: successThreshold,
		timeout:          timeout,
	}
}

// Get returns or creates a circuit breaker for a server.
func (m *Manager) Get(name string) *Breaker {
	m.mu.Lock()
	defer m.mu.Unlock()

	if b, ok := m.breakers[name]; ok {
		return b
	}

	b := NewBreaker(name, m.threshold, m.successThreshold, m.timeout)
	m.breakers[name] = b
	return b
}

// AllStats returns stats for all circuit breakers.
func (m *Manager) AllStats() []BreakerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make([]BreakerStats, 0, len(m.breakers))
	for _, b := range m.breakers {
		stats = append(stats, b.Stats())
	}
	return stats
}

// Reset resets all circuit breakers.
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, b := range m.breakers {
		b.Reset()
	}
}

// Remove removes a circuit breaker.
func (m *Manager) Remove(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.breakers, name)
}
