package commands

import (
	"log/slog"
	"sync"
	"time"
)

// TokenBucket implements a simple token bucket rate limiter
type TokenBucket struct {
	mu         sync.Mutex
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(maxTokens int, refillRate time.Duration) *TokenBucket {
	return &TokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if the operation is allowed and consumes a token if so
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := int(elapsed / tb.refillRate)

	if tokensToAdd > 0 {
		tb.tokens = minInt(tb.maxTokens, tb.tokens+tokensToAdd)
		tb.lastRefill = now
	}

	// Check if we have tokens available
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// minInt returns the smaller of two integers.
// Named minInt to avoid shadowing the Go 1.21+ builtin min function.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Global rate limiter for MCP tools
// Note: This is a package-level variable for simplicity. For testing, use SetGlobalRateLimiter.
var globalRateLimiter = NewTokenBucket(burstSize, time.Minute/time.Duration(maxToolCallsPerMinute))

// SetGlobalRateLimiter replaces the global rate limiter. Useful for testing.
// Returns the previous rate limiter so it can be restored.
func SetGlobalRateLimiter(limiter *TokenBucket) *TokenBucket {
	old := globalRateLimiter
	globalRateLimiter = limiter
	return old
}

// logRateLimitEvent logs when a rate limit is triggered
func logRateLimitEvent(operation string) {
	slog.Warn("rate limit exceeded for MCP operation",
		"operation", operation,
		"max_per_minute", maxToolCallsPerMinute,
		"burst_size", burstSize)
}
