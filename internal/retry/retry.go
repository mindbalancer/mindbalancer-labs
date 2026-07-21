// Package retry provides retry logic with exponential backoff.
package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

// Config holds retry configuration.
type Config struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       float64 // 0.0 to 1.0 - adds randomness to delay
}

// DefaultConfig returns sensible default retry configuration.
func DefaultConfig() Config {
	return Config{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}
}

// RetryableError wraps an error and indicates if it's retryable.
type RetryableError struct {
	Err       error
	Retryable bool
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable checks if an error is retryable.
func IsRetryable(err error) bool {
	var retryable *RetryableError
	if errors.As(err, &retryable) {
		return retryable.Retryable
	}
	// Default: certain error types are retryable
	return false
}

// NewRetryableError creates a new retryable error.
func NewRetryableError(err error, retryable bool) error {
	return &RetryableError{Err: err, Retryable: retryable}
}

// Result holds the result of a retried operation.
type Result[T any] struct {
	Value    T
	Attempts int
	Err      error
}

// Do executes a function with retry logic.
func Do[T any](ctx context.Context, cfg Config, fn func(ctx context.Context, attempt int) (T, error)) Result[T] {
	var result Result[T]
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		result.Attempts = attempt + 1

		// Check context before each attempt
		if ctx.Err() != nil {
			result.Err = ctx.Err()
			return result
		}

		value, err := fn(ctx, attempt)
		if err == nil {
			result.Value = value
			return result
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryable(err) {
			result.Err = err
			return result
		}

		// Don't sleep after the last attempt
		if attempt < cfg.MaxRetries {
			delay := calculateDelay(cfg, attempt)

			select {
			case <-ctx.Done():
				result.Err = ctx.Err()
				return result
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}

	result.Err = lastErr
	return result
}

// DoSimple executes a function with retry logic (simpler signature).
func DoSimple(ctx context.Context, cfg Config, fn func(ctx context.Context) error) error {
	result := Do(ctx, cfg, func(ctx context.Context, attempt int) (struct{}, error) {
		return struct{}{}, fn(ctx)
	})
	return result.Err
}

// calculateDelay calculates the delay for a given attempt with jitter.
func calculateDelay(cfg Config, attempt int) time.Duration {
	delay := float64(cfg.InitialDelay) * math.Pow(cfg.Multiplier, float64(attempt))

	// Apply max delay cap
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}

	// Apply jitter
	if cfg.Jitter > 0 {
		jitterRange := delay * cfg.Jitter
		delay = delay - jitterRange + (rand.Float64() * jitterRange * 2)
	}

	return time.Duration(delay)
}

// Backoff represents a backoff strategy.
type Backoff struct {
	cfg     Config
	attempt int
}

// NewBackoff creates a new backoff instance.
func NewBackoff(cfg Config) *Backoff {
	return &Backoff{cfg: cfg, attempt: 0}
}

// Next returns the next delay and increments the attempt counter.
func (b *Backoff) Next() time.Duration {
	delay := calculateDelay(b.cfg, b.attempt)
	b.attempt++
	return delay
}

// Reset resets the backoff to initial state.
func (b *Backoff) Reset() {
	b.attempt = 0
}

// Attempts returns the current attempt count.
func (b *Backoff) Attempts() int {
	return b.attempt
}

// MaxReached returns true if max retries have been reached.
func (b *Backoff) MaxReached() bool {
	return b.attempt >= b.cfg.MaxRetries
}

// WithRetry wraps a function to add retry capability.
type WithRetry struct {
	cfg Config
}

// NewWithRetry creates a new WithRetry instance.
func NewWithRetry(cfg Config) *WithRetry {
	return &WithRetry{cfg: cfg}
}

// Execute runs the function with retries.
func (w *WithRetry) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return DoSimple(ctx, w.cfg, fn)
}

// ExecuteWithResult runs a function that returns a result with retries.
func (w *WithRetry) ExecuteWithResult(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error) {
	result := Do(ctx, w.cfg, func(ctx context.Context, attempt int) (any, error) {
		return fn(ctx)
	})
	return result.Value, result.Err
}
