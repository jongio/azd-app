package dashboard

import (
	"net"
	"net/http"
	"strings"
)

// securityHeaders is middleware that adds security headers to all HTTP responses.
// These headers provide defense-in-depth against common web attacks (CWE-693).
//
// CSP notes:
//   - script-src: 'self' only — no eval, no inline scripts.
//   - style-src:  'unsafe-inline' is retained because multiple React components
//     use style={} props (e.g. progress bars, panel widths, animation hints).
//     Those props render as HTML style= attributes, which the browser blocks
//     without this token. Removing it requires refactoring ~10 components;
//     that work is tracked separately and out of scope for this patch.
//   - connect-src: 'self' only — the legacy WebSocket endpoint
//     (ws://localhost:* / wss://localhost:*) was removed; Connect-RPC uses HTTP.
//   - object-src, base-uri, form-action: 'none' for defence-in-depth.
//
// Cache-Control notes:
//
//	API and Connect-RPC responses receive Cache-Control: no-store (CWE-525) to
//	prevent browsers and shared proxies from persisting potentially sensitive
//	data (service state, auth tokens, log content) across requests.
//	Static assets (/app.js, /app.css, fonts, images) are intentionally excluded
//	so they benefit from normal browser caching; those paths never carry
//	per-user data.  index.html is excluded here because serveIndex (server_routes.go)
//	already sets no-store on that response to prevent the session-token-bearing
//	HTML from being cached.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:; "+
				"connect-src 'self'; "+
				"object-src 'none'; "+
				"base-uri 'none'; "+
				"form-action 'none'; "+
				"frame-ancestors 'none'",
		)
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// CWE-525: prevent caching of API responses that may carry sensitive data.
		// /api/* covers the REST shutdown endpoint; /azdapp.v1. covers all
		// Connect-RPC service paths (e.g. /azdapp.v1.LifecycleService/Ping).
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/azdapp.v1.") {
			w.Header().Set("Cache-Control", "no-store")
		}
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
