package dashboard

import (
	"net"
	"net/http"
)

// securityHeaders is middleware that adds security headers to all HTTP responses.
// These headers provide defense-in-depth against common web attacks (CWE-693).
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' ws://localhost:* wss://localhost:*; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// hostAllow is middleware that guards against DNS rebinding attacks (CWE-346).
// It validates the Host header and rejects requests from any host other than
// the known-safe loopback values before they reach any handler or route.
//
// Allowed: 127.0.0.1, localhost, ::1 (parsed from [::1]:port), [::1] (bare).
// Everything else receives 403 Forbidden.
func hostAllow(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			// No port present (or other parse error) — use the raw Host value.
			host = r.Host
		}
		switch host {
		case "127.0.0.1", "localhost", "::1", "[::1]":
			next.ServeHTTP(w, r)
		default:
			http.Error(w, "forbidden host", http.StatusForbidden)
		}
	})
}
