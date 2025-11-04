package authserver

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Port != 8080 {
		t.Errorf("expected port 8080, got %d", config.Port)
	}

	if config.EnableTLS {
		t.Error("expected TLS to be disabled by default")
	}

	if config.TokenExpiry != 15*time.Minute {
		t.Errorf("expected token expiry 15m, got %v", config.TokenExpiry)
	}

	if config.BindAddress != "0.0.0.0" {
		t.Errorf("expected bind address 0.0.0.0, got %s", config.BindAddress)
	}

	if config.RateLimitRequests != 10 {
		t.Errorf("expected rate limit 10, got %d", config.RateLimitRequests)
	}

	if config.CacheExpiry != 50*time.Minute {
		t.Errorf("expected cache expiry 50m, got %v", config.CacheExpiry)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &Config{
				Port:              8080,
				SharedSecret:      "test-secret",
				TokenExpiry:       15 * time.Minute,
				RateLimitRequests: 10,
			},
			expectError: false,
		},
		{
			name: "invalid port - zero",
			config: &Config{
				Port:              0,
				SharedSecret:      "test-secret",
				TokenExpiry:       15 * time.Minute,
				RateLimitRequests: 10,
			},
			expectError: true,
		},
		{
			name: "invalid port - negative",
			config: &Config{
				Port:              -1,
				SharedSecret:      "test-secret",
				TokenExpiry:       15 * time.Minute,
				RateLimitRequests: 10,
			},
			expectError: true,
		},
		{
			name: "invalid port - too high",
			config: &Config{
				Port:              65536,
				SharedSecret:      "test-secret",
				TokenExpiry:       15 * time.Minute,
				RateLimitRequests: 10,
			},
			expectError: true,
		},
		{
			name: "missing secret",
			config: &Config{
				Port:              8080,
				SharedSecret:      "",
				TokenExpiry:       15 * time.Minute,
				RateLimitRequests: 10,
			},
			expectError: true,
		},
		{
			name: "TLS enabled without cert",
			config: &Config{
				Port:              8080,
				EnableTLS:         true,
				SharedSecret:      "test-secret",
				TokenExpiry:       15 * time.Minute,
				RateLimitRequests: 10,
			},
			expectError: true,
		},
		{
			name: "TLS enabled with cert but no key",
			config: &Config{
				Port:              8080,
				EnableTLS:         true,
				CertFile:          "cert.pem",
				SharedSecret:      "test-secret",
				TokenExpiry:       15 * time.Minute,
				RateLimitRequests: 10,
			},
			expectError: true,
		},
		{
			name: "TLS enabled with both cert and key",
			config: &Config{
				Port:              8080,
				EnableTLS:         true,
				CertFile:          "cert.pem",
				KeyFile:           "key.pem",
				SharedSecret:      "test-secret",
				TokenExpiry:       15 * time.Minute,
				RateLimitRequests: 10,
			},
			expectError: false,
		},
		{
			name: "invalid token expiry",
			config: &Config{
				Port:              8080,
				SharedSecret:      "test-secret",
				TokenExpiry:       0,
				RateLimitRequests: 10,
			},
			expectError: true,
		},
		{
			name: "invalid rate limit",
			config: &Config{
				Port:              8080,
				SharedSecret:      "test-secret",
				TokenExpiry:       15 * time.Minute,
				RateLimitRequests: 0,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfig_Addr(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name: "default address",
			config: &Config{
				BindAddress: "0.0.0.0",
				Port:        8080,
			},
			expected: "0.0.0.0:8080",
		},
		{
			name: "localhost",
			config: &Config{
				BindAddress: "localhost",
				Port:        9000,
			},
			expected: "localhost:9000",
		},
		{
			name: "specific IP",
			config: &Config{
				BindAddress: "192.168.1.1",
				Port:        3000,
			},
			expected: "192.168.1.1:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := tt.config.Addr()
			if addr != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, addr)
			}
		})
	}
}

func TestConfig_TLSConfig(t *testing.T) {
	t.Run("TLS disabled", func(t *testing.T) {
		config := &Config{EnableTLS: false}
		tlsConfig, err := config.TLSConfig()
		
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		
		if tlsConfig != nil {
			t.Error("expected nil TLS config when TLS is disabled")
		}
	})

	t.Run("TLS enabled", func(t *testing.T) {
		config := &Config{EnableTLS: true}
		tlsConfig, err := config.TLSConfig()
		
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		
		if tlsConfig == nil {
			t.Error("expected non-nil TLS config when TLS is enabled")
			return
		}

		// Verify secure defaults
		if tlsConfig.MinVersion < 771 { // TLS 1.2
			t.Error("expected minimum TLS version 1.2")
		}

		if len(tlsConfig.CipherSuites) == 0 {
			t.Error("expected cipher suites to be configured")
		}
	})
}
