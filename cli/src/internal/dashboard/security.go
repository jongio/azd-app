package dashboard

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	mu       sync.Mutex
	tokens   float64
	max      float64
	refill   float64 // tokens per second
	lastTime time.Time
}

// NewRateLimiter creates a new rate limiter
// max: maximum number of tokens (burst size)
// refill: tokens to add per second
func NewRateLimiter(max float64, refill float64) *RateLimiter {
	return &RateLimiter{
		tokens:   max,
		max:      max,
		refill:   refill,
		lastTime: time.Now(),
	}
}

// Allow checks if a request should be allowed
// Returns true if request is allowed, false if rate limited
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastTime).Seconds()

	// Add tokens based on elapsed time
	rl.tokens = min(rl.max, rl.tokens+elapsed*rl.refill)
	rl.lastTime = now

	// Check if we have enough tokens
	if rl.tokens >= 1.0 {
		rl.tokens -= 1.0
		return true
	}

	return false
}

// Reset resets the rate limiter to full capacity
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.tokens = rl.max
	rl.lastTime = time.Now()
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// EndpointRateLimits holds rate limiters for different endpoints
type EndpointRateLimits struct {
	mu       sync.RWMutex
	limiters map[string]*RateLimiter
}

// NewEndpointRateLimits creates endpoint-specific rate limiters
func NewEndpointRateLimits() *EndpointRateLimits {
	return &EndpointRateLimits{
		limiters: map[string]*RateLimiter{
			// Save: 10 requests/minute = 0.167 requests/second
			"/api/editor/config_POST": NewRateLimiter(10, 10.0/60.0),

			// Validate: 60 requests/minute = 1 request/second
			"/api/editor/validate": NewRateLimiter(60, 60.0/60.0),

			// Other endpoints: 100 requests/minute = 1.67 requests/second
			"default": NewRateLimiter(100, 100.0/60.0),
		},
	}
}

// GetLimiter returns the appropriate rate limiter for a request
func (erl *EndpointRateLimits) GetLimiter(r *http.Request) *RateLimiter {
	erl.mu.RLock()
	defer erl.mu.RUnlock()

	// Create key from path and method
	key := r.URL.Path
	if r.Method == http.MethodPost && r.URL.Path == "/api/editor/config" {
		key = "/api/editor/config_POST"
	}

	// Get specific limiter or default
	if limiter, ok := erl.limiters[key]; ok {
		return limiter
	}
	return erl.limiters["default"]
}

// RateLimitMiddleware wraps an http.Handler with rate limiting
func RateLimitMiddleware(limiter *EndpointRateLimits, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rl := limiter.GetLimiter(r)

		if !rl.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			writeJSON(w, map[string]string{
				"error": "Rate limit exceeded. Please try again later.",
			})
			return
		}

		next(w, r)
	}
}

// FileSizeLimits defines maximum file sizes
const (
	MaxAzureYamlSize = 10 * 1024 * 1024 // 10MB
	MaxBackupTotal   = 50 * 1024 * 1024 // 50MB
	MaxRequestBody   = 10 * 1024 * 1024 // 10MB
)

// ValidateRequestSize checks if request body size is within limits
func ValidateRequestSize(r *http.Request, maxSize int64) error {
	if r.ContentLength > maxSize {
		return fmt.Errorf("request body size (%d bytes) exceeds maximum (%d bytes)", r.ContentLength, maxSize)
	}
	return nil
}

// ValidateBackupsTotalSize checks if total backup size is within limits
func ValidateBackupsTotalSize(backups []BackupInfo) error {
	var totalSize int64
	for _, backup := range backups {
		totalSize += backup.Size
	}

	if totalSize > MaxBackupTotal {
		return fmt.Errorf("total backup size (%d bytes) exceeds maximum (%d bytes)", totalSize, MaxBackupTotal)
	}

	return nil
}

// FormatBytes formats bytes to human-readable string
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	sizes := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), sizes[exp])
}
