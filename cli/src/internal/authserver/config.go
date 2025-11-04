package authserver

import (
	"crypto/tls"
	"fmt"
	"time"
)

// Config holds the authentication server configuration.
type Config struct {
	// Port is the HTTP/HTTPS port to listen on (default: 8080)
	Port int

	// EnableTLS enables HTTPS instead of HTTP
	EnableTLS bool

	// CertFile is the path to the TLS certificate file
	CertFile string

	// KeyFile is the path to the TLS key file
	KeyFile string

	// SharedSecret is the authentication secret for clients
	SharedSecret string

	// TokenExpiry is the token expiration duration (default: 15 minutes)
	TokenExpiry time.Duration

	// BindAddress is the network interface to bind to (default: "0.0.0.0")
	BindAddress string

	// RateLimitRequests is the max requests per minute per client (default: 10)
	RateLimitRequests int

	// CacheExpiry is the duration to cache Azure tokens (default: 50 minutes)
	CacheExpiry time.Duration
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Port:              8080,
		EnableTLS:         false, // Default to HTTP for local dev; TLS recommended for production
		TokenExpiry:       15 * time.Minute,
		BindAddress:       "0.0.0.0",
		RateLimitRequests: 10,
		CacheExpiry:       50 * time.Minute,
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be between 1-65535)", c.Port)
	}

	if c.SharedSecret == "" {
		return fmt.Errorf("shared secret is required (set via --secret flag or AZD_AUTH_SECRET env var)")
	}

	if c.EnableTLS {
		if c.CertFile == "" {
			return fmt.Errorf("TLS certificate file required when TLS is enabled")
		}
		if c.KeyFile == "" {
			return fmt.Errorf("TLS key file required when TLS is enabled")
		}
	}

	if c.TokenExpiry <= 0 {
		return fmt.Errorf("token expiry must be positive")
	}

	if c.RateLimitRequests <= 0 {
		return fmt.Errorf("rate limit must be positive")
	}

	return nil
}

// TLSConfig returns the TLS configuration if TLS is enabled.
func (c *Config) TLSConfig() (*tls.Config, error) {
	if !c.EnableTLS {
		return nil, nil
	}

	// Use secure TLS configuration
	return &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
	}, nil
}

// Addr returns the full listen address.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.BindAddress, c.Port)
}
