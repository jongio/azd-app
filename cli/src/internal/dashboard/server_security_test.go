package dashboard

import (
	"net/http"
	"testing"
)

// TestWebSocketOriginValidation tests that WebSocket connections only accept localhost origins.
func TestWebSocketOriginValidation(t *testing.T) {
	tests := []struct {
		name        string
		origin      string
		expectAllow bool
	}{
		{
			name:        "empty origin allowed (non-browser clients)",
			origin:      "",
			expectAllow: true,
		},
		{
			name:        "localhost http allowed",
			origin:      "http://localhost:3000",
			expectAllow: true,
		},
		{
			name:        "localhost https allowed",
			origin:      "https://localhost:3000",
			expectAllow: true,
		},
		{
			name:        "127.0.0.1 http allowed",
			origin:      "http://127.0.0.1:8080",
			expectAllow: true,
		},
		{
			name:        "127.0.0.1 https allowed",
			origin:      "https://127.0.0.1:8080",
			expectAllow: true,
		},
		{
			name:        "external domain blocked",
			origin:      "http://example.com",
			expectAllow: false,
		},
		{
			name:        "external https blocked",
			origin:      "https://malicious.com",
			expectAllow: false,
		},
		{
			name:        "localhost subdomain blocked (security)",
			origin:      "http://malicious.localhost.com",
			expectAllow: false,
		},
		{
			name:        "localhost-like domain blocked",
			origin:      "http://localhost.evil.com",
			expectAllow: false,
		},
		{
			name:        "IPv4 non-localhost blocked",
			origin:      "http://192.168.1.1:3000",
			expectAllow: false,
		},
		{
			name:        "localhost without port allowed",
			origin:      "http://localhost",
			expectAllow: false, // Must have port separator
		},
		{
			name:        "localhost with path allowed",
			origin:      "http://localhost:3000/path",
			expectAllow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock request with Origin header
			req := &http.Request{
				Header: http.Header{},
			}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			// Test the CheckOrigin function
			allowed := upgrader.CheckOrigin(req)

			if allowed != tt.expectAllow {
				t.Errorf("Origin validation failed for %q: got allowed=%v, want %v",
					tt.origin, allowed, tt.expectAllow)
			}
		})
	}
}

// TestWebSocketOriginValidation_CSWSH tests protection against Cross-Site WebSocket Hijacking.
func TestWebSocketOriginValidation_CSWSH(t *testing.T) {
	// Simulate CSWSH attack scenarios
	maliciousOrigins := []string{
		"http://attacker.com",
		"https://phishing.net",
		"http://localhost.evil.com",
		"http://127.0.0.1.attacker.com",
		"http://xn--lochst-5wa.com", // IDN homograph attack
	}

	for _, origin := range maliciousOrigins {
		t.Run("CSWSH_protection_"+origin, func(t *testing.T) {
			req := &http.Request{
				Header: http.Header{
					"Origin": []string{origin},
				},
			}

			if upgrader.CheckOrigin(req) {
				t.Errorf("CSWSH vulnerability: malicious origin %q was allowed", origin)
			}
		})
	}
}
