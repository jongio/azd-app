package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		rl := NewRateLimiter(10, 1) // 10 max, 1 per second

		// Should allow first request
		if !rl.Allow() {
			t.Error("Expected first request to be allowed")
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		rl := NewRateLimiter(2, 0.1) // 2 max, very slow refill

		// Use up tokens
		rl.Allow()
		rl.Allow()

		// Third request should be blocked
		if rl.Allow() {
			t.Error("Expected third request to be blocked")
		}
	})

	t.Run("refills tokens over time", func(t *testing.T) {
		rl := NewRateLimiter(10, 10) // 10 max, 10 per second

		// Use all tokens
		for i := 0; i < 10; i++ {
			rl.Allow()
		}

		// Should be blocked now
		if rl.Allow() {
			t.Error("Expected request to be blocked after using all tokens")
		}

		// Wait for refill (need at least 1 second for 10 tokens/sec)
		time.Sleep(150 * time.Millisecond)

		// Should allow now (at least 1 token refilled)
		if !rl.Allow() {
			t.Error("Expected request to be allowed after refill")
		}
	})

	t.Run("respects maximum tokens", func(t *testing.T) {
		rl := NewRateLimiter(5, 100) // 5 max, fast refill

		// Wait to let it try to refill beyond max
		time.Sleep(100 * time.Millisecond)

		// Should still only have max tokens (5)
		allowed := 0
		for i := 0; i < 10; i++ {
			if rl.Allow() {
				allowed++
			}
		}

		if allowed > 5 {
			t.Errorf("Expected at most 5 allowed requests, got %d", allowed)
		}
	})
}

func TestEndpointRateLimits(t *testing.T) {
	t.Run("returns correct limiter for save endpoint", func(t *testing.T) {
		erl := NewEndpointRateLimits()
		req := httptest.NewRequest(http.MethodPost, "/api/editor/config", nil)

		limiter := erl.GetLimiter(req)
		if limiter == nil {
			t.Error("Expected limiter to be returned")
		}
	})

	t.Run("returns correct limiter for validate endpoint", func(t *testing.T) {
		erl := NewEndpointRateLimits()
		req := httptest.NewRequest(http.MethodPost, "/api/editor/validate", nil)

		limiter := erl.GetLimiter(req)
		if limiter == nil {
			t.Error("Expected limiter to be returned")
		}
	})

	t.Run("returns default limiter for other endpoints", func(t *testing.T) {
		erl := NewEndpointRateLimits()
		req := httptest.NewRequest(http.MethodGet, "/api/editor/backups", nil)

		limiter := erl.GetLimiter(req)
		if limiter == nil {
			t.Error("Expected default limiter to be returned")
		}
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		erl := NewEndpointRateLimits()
		called := false

		handler := RateLimitMiddleware(erl, func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/api/editor/config", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if !called {
			t.Error("Expected handler to be called")
		}
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		erl := NewEndpointRateLimits()

		// Create a limiter with very low limits for testing
		erl.limiters["default"] = NewRateLimiter(1, 0.01)

		handler := RateLimitMiddleware(erl, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/api/editor/config", nil)

		// First request should work
		w1 := httptest.NewRecorder()
		handler(w1, req)
		if w1.Code != http.StatusOK {
			t.Errorf("Expected first request to succeed, got %d", w1.Code)
		}

		// Second request should be rate limited
		w2 := httptest.NewRecorder()
		handler(w2, req)
		if w2.Code != http.StatusTooManyRequests {
			t.Errorf("Expected 429 Too Many Requests, got %d", w2.Code)
		}

		// Check for Retry-After header
		if w2.Header().Get("Retry-After") == "" {
			t.Error("Expected Retry-After header")
		}
	})
}

func TestValidateRequestSize(t *testing.T) {
	t.Run("accepts requests within size limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/editor/config", nil)
		req.ContentLength = 1024 // 1KB

		err := ValidateRequestSize(req, MaxRequestBody)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("rejects requests over size limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/editor/config", nil)
		req.ContentLength = MaxRequestBody + 1

		err := ValidateRequestSize(req, MaxRequestBody)
		if err == nil {
			t.Error("Expected error for oversized request")
		}
	})
}

func TestValidateBackupsTotalSize(t *testing.T) {
	t.Run("accepts backups within total size limit", func(t *testing.T) {
		backups := []BackupInfo{
			{Size: 1024 * 1024},     // 1MB
			{Size: 2 * 1024 * 1024}, // 2MB
			{Size: 3 * 1024 * 1024}, // 3MB
		}

		err := ValidateBackupsTotalSize(backups)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("rejects backups over total size limit", func(t *testing.T) {
		backups := []BackupInfo{
			{Size: 30 * 1024 * 1024}, // 30MB
			{Size: 25 * 1024 * 1024}, // 25MB (total 55MB > 50MB limit)
		}

		err := ValidateBackupsTotalSize(backups)
		if err == nil {
			t.Error("Expected error for oversized backups")
		}
	})

	t.Run("handles empty backup list", func(t *testing.T) {
		backups := []BackupInfo{}

		err := ValidateBackupsTotalSize(backups)
		if err != nil {
			t.Errorf("Expected no error for empty list, got %v", err)
		}
	})
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{5 * 1024 * 1024, "5.0 MB"},
		{10 * 1024 * 1024, "10.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
	}

	for _, tt := range tests {
		result := FormatBytes(tt.input)
		if result != tt.expected {
			t.Errorf("FormatBytes(%d) = %s; want %s", tt.input, result, tt.expected)
		}
	}
}
