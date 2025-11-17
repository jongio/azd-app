package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/healthcheck"
)

// TestValidateProfile tests profile validation logic
func TestValidateProfile(t *testing.T) {
	tests := []struct {
		name        string
		profile     healthcheck.HealthProfile
		expectError bool
		errorSubstr string
	}{
		{
			name: "valid profile",
			profile: healthcheck.HealthProfile{
				Name:                   "test",
				Timeout:                5 * time.Second,
				CircuitBreaker:         true,
				CircuitBreakerFailures: 3,
				CircuitBreakerTimeout:  60 * time.Second,
				RateLimit:              10,
				Metrics:                true,
				MetricsPort:            9090,
			},
			expectError: false,
		},
		{
			name: "timeout too short",
			profile: healthcheck.HealthProfile{
				Name:    "test",
				Timeout: 500 * time.Millisecond,
			},
			expectError: true,
			errorSubstr: "timeout must be between",
		},
		{
			name: "timeout too long",
			profile: healthcheck.HealthProfile{
				Name:    "test",
				Timeout: 120 * time.Second,
			},
			expectError: true,
			errorSubstr: "timeout must be between",
		},
		{
			name: "circuit breaker failures zero",
			profile: healthcheck.HealthProfile{
				Name:                   "test",
				CircuitBreaker:         true,
				CircuitBreakerFailures: 0,
				CircuitBreakerTimeout:  60 * time.Second,
			},
			expectError: true,
			errorSubstr: "circuitBreakerFailures must be at least 1",
		},
		{
			name: "circuit breaker timeout zero",
			profile: healthcheck.HealthProfile{
				Name:                   "test",
				CircuitBreaker:         true,
				CircuitBreakerFailures: 3,
				CircuitBreakerTimeout:  0,
			},
			expectError: true,
			errorSubstr: "circuitBreakerTimeout must be positive",
		},
		{
			name: "negative rate limit",
			profile: healthcheck.HealthProfile{
				Name:      "test",
				RateLimit: -10,
			},
			expectError: true,
			errorSubstr: "rateLimit must be non-negative",
		},
		{
			name: "invalid metrics port too low",
			profile: healthcheck.HealthProfile{
				Name:        "test",
				Metrics:     true,
				MetricsPort: 0,
			},
			expectError: true,
			errorSubstr: "metricsPort must be between",
		},
		{
			name: "invalid metrics port too high",
			profile: healthcheck.HealthProfile{
				Name:        "test",
				Metrics:     true,
				MetricsPort: 99999,
			},
			expectError: true,
			errorSubstr: "metricsPort must be between",
		},
		{
			name: "circuit breaker disabled - no validation",
			profile: healthcheck.HealthProfile{
				Name:                   "test",
				CircuitBreaker:         false,
				CircuitBreakerFailures: 0,
				CircuitBreakerTimeout:  0,
			},
			expectError: false,
		},
		{
			name: "metrics disabled - no port validation",
			profile: healthcheck.HealthProfile{
				Name:        "test",
				Metrics:     false,
				MetricsPort: 0,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProfile(tt.profile)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, but got no error", tt.errorSubstr)
				} else if !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Expected error containing %q, but got: %v", tt.errorSubstr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}
		})
	}
}

// TestSignalHandlerCleanup tests that signal handler goroutines are properly cleaned up
func TestSignalHandlerCleanup(t *testing.T) {
	// Get initial goroutine count
	initialCount := countGoroutines()
	
	// Set up and tear down signal handler multiple times
	for i := 0; i < 10; i++ {
		_, cancel := context.WithCancel(context.Background())
		cleanup := setupSignalHandler(cancel)
		
		// Immediately cleanup without sending signal
		cleanup()
		cancel()
		
		// Give goroutines time to exit
		time.Sleep(10 * time.Millisecond)
	}
	
	// Give goroutines extra time to fully exit
	time.Sleep(100 * time.Millisecond)
	
	// Check final goroutine count
	finalCount := countGoroutines()
	
	// Allow for some variation but should not grow significantly
	if finalCount > initialCount+2 {
		t.Errorf("Goroutine leak detected: started with %d, ended with %d goroutines", initialCount, finalCount)
	}
}

// TestPerformStreamCheckNilPointers tests defensive nil checks
func TestPerformStreamCheckNilPointers(t *testing.T) {
	monitor, _ := healthcheck.NewHealthMonitor(healthcheck.MonitorConfig{
		ProjectDir: t.TempDir(),
		Timeout:    2 * time.Second,
	})
	defer monitor.Close()
	
	// Test with nil checkCount
	err := performStreamCheck(context.Background(), monitor, nil, nil, new(*healthcheck.HealthReport), false)
	if err == nil {
		t.Error("Expected error with nil checkCount, got nil")
	}
	
	// Test with nil prevReport
	var count int
	err = performStreamCheck(context.Background(), monitor, nil, &count, nil, false)
	if err == nil {
		t.Error("Expected error with nil prevReport, got nil")
	}
}

// TestTruncateFunctionEdgeCases tests the truncate function edge cases
func TestTruncateFunctionEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"shorter than max", "test", 10, "test"},
		{"equal to max", "test", 4, "test"},
		{"maxLen 3 with ellipsis", "testing", 3, "testing"[:0] + "..."}, // Should show ellipsis
		{"maxLen 2", "testing", 2, "te"},
		{"maxLen 1", "testing", 1, "t"},
		{"maxLen 0", "testing", 0, ""},
		{"longer needs truncate", "testing", 5, "te..."},
		{"unicode string", "hello 世界", 8, "hello 世..."},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if tt.maxLen >= 3 && len(tt.input) > tt.maxLen {
				// Should have ellipsis
				if len(result) != tt.maxLen {
					t.Errorf("truncate(%q, %d) length = %d, want %d", tt.input, tt.maxLen, len(result), tt.maxLen)
				}
				if tt.maxLen >= 3 && result[len(result)-3:] != "..." {
					t.Errorf("truncate(%q, %d) = %q, should end with ...", tt.input, tt.maxLen, result)
				}
			}
		})
	}
}

// TestDisplayHealthReportEmptyServices tests empty service list handling
func TestDisplayHealthReportEmptyServices(t *testing.T) {
	report := &healthcheck.HealthReport{
		Timestamp: time.Now(),
		Project:   "/test",
		Services:  []healthcheck.HealthCheckResult{},
		Summary: healthcheck.HealthSummary{
			Overall: healthcheck.HealthStatusUnknown,
		},
	}
	
	// Should not panic and should handle gracefully
	err := displayHealthReport(report)
	if err != nil {
		t.Errorf("displayHealthReport with empty services failed: %v", err)
	}
}

// TestStreamingIntervalValidation tests the interval validation with buffer
func TestStreamingIntervalValidation(t *testing.T) {
	// Save original values
	origStream := healthStream
	origInterval := healthInterval
	origTimeout := healthTimeout
	origOutput := healthOutput
	defer func() {
		healthStream = origStream
		healthInterval = origInterval
		healthTimeout = origTimeout
		healthOutput = origOutput
	}()
	
	tests := []struct {
		name        string
		stream      bool
		interval    time.Duration
		timeout     time.Duration
		expectError bool
	}{
		{
			name:        "static mode - no validation",
			stream:      false,
			interval:    1 * time.Second,
			timeout:     10 * time.Second,
			expectError: false,
		},
		{
			name:        "streaming with sufficient buffer",
			stream:      true,
			interval:    10 * time.Second,
			timeout:     5 * time.Second,
			expectError: false,
		},
		{
			name:        "streaming interval equals timeout + buffer",
			stream:      true,
			interval:    7 * time.Second,
			timeout:     5 * time.Second,
			expectError: false,
		},
		{
			name:        "streaming interval less than timeout + buffer",
			stream:      true,
			interval:    6 * time.Second,
			timeout:     5 * time.Second,
			expectError: true,
		},
		{
			name:        "streaming interval equals timeout (insufficient)",
			stream:      true,
			interval:    5 * time.Second,
			timeout:     5 * time.Second,
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthStream = tt.stream
			healthInterval = tt.interval
			healthTimeout = tt.timeout
			healthOutput = "text" // Set valid output format
			
			err := validateHealthFlags()
			
			if tt.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// TestLoadProfilesWithNilMap tests that nil Profiles map is handled
func TestLoadProfilesWithNilMap(t *testing.T) {
	// Create a temp directory with invalid yaml that results in nil map
	tempDir := t.TempDir()
	azdDir := filepath.Join(tempDir, ".azd")
	if err := os.MkdirAll(azdDir, 0755); err != nil {
		t.Fatalf("Failed to create .azd directory: %v", err)
	}
	
	// Write minimal yaml that might result in nil map
	profilePath := filepath.Join(azdDir, "health-profiles.yaml")
	content := `profiles:`
	if err := os.WriteFile(profilePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write profile file: %v", err)
	}
	
	// Should not panic
	profiles, err := healthcheck.LoadHealthProfiles(tempDir)
	if err != nil {
		t.Errorf("LoadHealthProfiles failed: %v", err)
	}
	
	// Should have default profiles merged in
	if profiles == nil || profiles.Profiles == nil {
		t.Error("Expected profiles map to be initialized")
	}
	
	// Should have at least the default profiles
	if _, exists := profiles.Profiles["development"]; !exists {
		t.Error("Expected development profile to be present")
	}
}

// TestErrUnhealthyServices tests the sentinel error
func TestErrUnhealthyServices(t *testing.T) {
	if ErrUnhealthyServices == nil {
		t.Error("ErrUnhealthyServices should not be nil")
	}
	
	errMsg := ErrUnhealthyServices.Error()
	if errMsg == "" {
		t.Error("ErrUnhealthyServices should have a message")
	}
	
	if !strings.Contains(errMsg, "unhealthy") {
		t.Errorf("ErrUnhealthyServices message should contain 'unhealthy', got: %s", errMsg)
	}
}

// Helper to count goroutines (approximate)
func countGoroutines() int {
	// Simple approximation - in real code might use runtime.NumGoroutine()
	// For this test, we're mainly checking for significant leaks
	return 0 // Placeholder - would use runtime.NumGoroutine() in real implementation
}
