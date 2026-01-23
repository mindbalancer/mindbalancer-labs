// Package ratelimit provides rate limiting functionality.
package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/mindbalancer/mindbalancer/internal/storage"
)

// Limiter implements rate limiting per user.
type Limiter struct {
	mu       sync.RWMutex
	storage  *storage.Storage
	enabled  bool
	windows  map[string]*Window
	defaults DefaultLimits
}

// DefaultLimits holds default rate limits.
type DefaultLimits struct {
	RequestsPerMinute int
	TokensPerMinute   int
}

// Window tracks rate limit state for a user.
type Window struct {
	Requests    int
	Tokens      int
	WindowStart time.Time
}

// Result represents the result of a rate limit check.
type Result struct {
	Allowed           bool
	RemainingRequests int
	RemainingTokens   int
	ResetAt           time.Time
	RetryAfter        time.Duration
}

// NewLimiter creates a new rate limiter.
func NewLimiter(store *storage.Storage, enabled bool, defaultReqPerMin, defaultTokensPerMin int) *Limiter {
	return &Limiter{
		storage: store,
		enabled: enabled,
		windows: make(map[string]*Window),
		defaults: DefaultLimits{
			RequestsPerMinute: defaultReqPerMin,
			TokensPerMinute:   defaultTokensPerMin,
		},
	}
}

// Allow checks if a request is allowed for the given user.
func (l *Limiter) Allow(ctx context.Context, username string) (*Result, error) {
	if !l.enabled {
		return &Result{Allowed: true}, nil
	}

	// Get user limits
	limits, err := l.getUserLimits(ctx, username)
	if err != nil {
		return nil, err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	window, ok := l.windows[username]
	
	// Create new window or reset if expired
	if !ok || now.Sub(window.WindowStart) >= time.Minute {
		window = &Window{
			Requests:    0,
			Tokens:      0,
			WindowStart: now,
		}
		l.windows[username] = window
	}

	// Check request limit
	if window.Requests >= limits.RequestsPerMinute {
		resetAt := window.WindowStart.Add(time.Minute)
		return &Result{
			Allowed:           false,
			RemainingRequests: 0,
			RemainingTokens:   limits.TokensPerMinute - window.Tokens,
			ResetAt:           resetAt,
			RetryAfter:        resetAt.Sub(now),
		}, nil
	}

	// Increment request count
	window.Requests++

	return &Result{
		Allowed:           true,
		RemainingRequests: limits.RequestsPerMinute - window.Requests,
		RemainingTokens:   limits.TokensPerMinute - window.Tokens,
		ResetAt:           window.WindowStart.Add(time.Minute),
	}, nil
}

// RecordTokens records token usage for a user.
func (l *Limiter) RecordTokens(username string, tokens int) {
	if !l.enabled {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if window, ok := l.windows[username]; ok {
		window.Tokens += tokens
	}
}

// CheckTokens checks if token usage would exceed limits.
func (l *Limiter) CheckTokens(ctx context.Context, username string, tokens int) (*Result, error) {
	if !l.enabled {
		return &Result{Allowed: true}, nil
	}

	limits, err := l.getUserLimits(ctx, username)
	if err != nil {
		return nil, err
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	window, ok := l.windows[username]
	if !ok {
		return &Result{
			Allowed:         true,
			RemainingTokens: limits.TokensPerMinute,
		}, nil
	}

	now := time.Now()
	if now.Sub(window.WindowStart) >= time.Minute {
		return &Result{
			Allowed:         true,
			RemainingTokens: limits.TokensPerMinute,
		}, nil
	}

	if window.Tokens+tokens > limits.TokensPerMinute {
		resetAt := window.WindowStart.Add(time.Minute)
		return &Result{
			Allowed:         false,
			RemainingTokens: limits.TokensPerMinute - window.Tokens,
			ResetAt:         resetAt,
			RetryAfter:      resetAt.Sub(now),
		}, nil
	}

	return &Result{
		Allowed:         true,
		RemainingTokens: limits.TokensPerMinute - window.Tokens - tokens,
	}, nil
}

// getUserLimits gets rate limits for a user.
func (l *Limiter) getUserLimits(ctx context.Context, username string) (*DefaultLimits, error) {
	// Try to get user-specific limits
	user, err := l.storage.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	if user != nil && user.Active {
		return &DefaultLimits{
			RequestsPerMinute: user.MaxRequestsPerMinute,
			TokensPerMinute:   user.MaxTokensPerMinute,
		}, nil
	}

	// Return defaults
	return &l.defaults, nil
}

// GetStats returns rate limit statistics for a user.
func (l *Limiter) GetStats(username string) *Window {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if window, ok := l.windows[username]; ok {
		// Return a copy
		return &Window{
			Requests:    window.Requests,
			Tokens:      window.Tokens,
			WindowStart: window.WindowStart,
		}
	}
	return nil
}

// Reset resets rate limits for a user.
func (l *Limiter) Reset(username string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.windows, username)
}

// ResetAll resets all rate limits.
func (l *Limiter) ResetAll() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.windows = make(map[string]*Window)
}

// SetEnabled enables or disables rate limiting.
func (l *Limiter) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = enabled
}

// IsEnabled returns whether rate limiting is enabled.
func (l *Limiter) IsEnabled() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.enabled
}

// Cleanup removes expired windows.
func (l *Limiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for username, window := range l.windows {
		if now.Sub(window.WindowStart) >= 5*time.Minute {
			delete(l.windows, username)
		}
	}
}

// StartCleanup starts a background goroutine to clean up expired windows.
func (l *Limiter) StartCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				l.Cleanup()
			}
		}
	}()
}
