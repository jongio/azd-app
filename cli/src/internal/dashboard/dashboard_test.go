package dashboard

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
)

func TestSecurityHeaders(t *testing.T) {
	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for key, want := range expectedHeaders {
		got := rec.Header().Get(key)
		if got != want {
			t.Errorf("header %q = %q, want %q", key, got, want)
		}
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Content-Security-Policy header should be set")
	}
}

func TestSecurityHeaders_RetryPath(t *testing.T) {
	// buildHandler() is the shared construction point used by both the primary
	// start path (server_core.go) and the retry-port path (server_port_mgmt.go).
	// This test verifies that it applies securityHeaders so both paths return
	// X-Frame-Options: DENY, guarding against CWE-693.
	//
	// The Host header must be a valid loopback value so that the outermost
	// hostAllow middleware (CWE-346) allows the request to reach securityHeaders.
	s := &Server{mux: http.NewServeMux()}
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := s.buildHandler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "localhost:8080"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	want := "DENY"
	if got := rec.Header().Get("X-Frame-Options"); got != want {
		t.Errorf("retry-path handler X-Frame-Options = %q, want %q", got, want)
	}

	// Verify the full set of security headers is present, not just X-Frame-Options.
	checks := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for key, wantVal := range checks {
		if got := rec.Header().Get(key); got != wantVal {
			t.Errorf("retry-path handler %q = %q, want %q", key, got, wantVal)
		}
	}
	if csp := rec.Header().Get("Content-Security-Policy"); csp == "" {
		t.Error("retry-path handler Content-Security-Policy should be set")
	}
}

func TestSecurityHeaders_PassesThrough(t *testing.T) {
	called := false
	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler should be called")
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

// TestSecurityHeaders_CSPTokens validates the exact Content-Security-Policy
// directive set required by SEC-022 (CWE-693):
//   - script-src is 'self' with no 'unsafe-eval' or 'unsafe-inline'
//   - legacy WebSocket origins (ws://localhost:*, wss://localhost:*) removed
//   - defence-in-depth directives: object-src 'none', base-uri 'none', form-action 'none'
func TestSecurityHeaders_CSPTokens(t *testing.T) {
	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("Content-Security-Policy header must be set")
	}

	// script-src must be 'self' only — no eval, no inline scripts.
	if !strings.Contains(csp, "script-src 'self'") {
		t.Errorf("CSP must contain \"script-src 'self'\"; got: %s", csp)
	}
	if strings.Contains(csp, "'unsafe-eval'") {
		t.Errorf("CSP script-src must not contain 'unsafe-eval'; got: %s", csp)
	}

	// Legacy WebSocket origins were removed when the WS endpoint was dropped.
	// connect-src must be 'self' only; ws:// and wss:// must be absent.
	if strings.Contains(csp, "ws://localhost") {
		t.Errorf("CSP connect-src must not contain dead ws://localhost:* origin; got: %s", csp)
	}
	if strings.Contains(csp, "wss://localhost") {
		t.Errorf("CSP connect-src must not contain dead wss://localhost:* origin; got: %s", csp)
	}

	// Defence-in-depth directives added by SEC-022.
	for _, want := range []string{
		"connect-src 'self'",
		"object-src 'none'",
		"base-uri 'none'",
		"form-action 'none'",
		"frame-ancestors 'none'",
	} {
		if !strings.Contains(csp, want) {
			t.Errorf("CSP must contain %q; got: %s", want, csp)
		}
	}
}

func TestServiceInfoPayload(t *testing.T) {
	t.Run("nil services produces empty slice", func(t *testing.T) {
		payload := serviceInfoPayload(nil)
		services, ok := payload["services"]
		if !ok {
			t.Fatal("payload should have 'services' key")
		}
		slice, ok := services.([]*serviceinfo.ServiceInfo)
		if !ok {
			t.Fatal("services should be []*serviceinfo.ServiceInfo")
		}
		if len(slice) != 0 {
			t.Errorf("expected empty slice, got %d items", len(slice))
		}
	})

	t.Run("non-nil services passes through", func(t *testing.T) {
		input := []*serviceinfo.ServiceInfo{
			{Name: "api"},
			{Name: "web"},
		}
		payload := serviceInfoPayload(input)
		services := payload["services"].([]*serviceinfo.ServiceInfo)
		if len(services) != 2 {
			t.Errorf("expected 2 services, got %d", len(services))
		}
	})
}

func TestLoadAzureYaml(t *testing.T) {
	t.Run("missing file returns error", func(t *testing.T) {
		_, err := loadAzureYaml(t.TempDir())
		if err == nil {
			t.Error("expected error for missing azure.yaml")
		}
	})

	t.Run("valid yaml parses successfully", func(t *testing.T) {
		dir := t.TempDir()
		content := `name: test-app
services:
  api:
    host: appservice
    language: python
  web:
    host: appservice
    language: typescript
`
		if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		result, err := loadAzureYaml(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Name != "test-app" {
			t.Errorf("name = %q, want %q", result.Name, "test-app")
		}
		if len(result.Services) != 2 {
			t.Errorf("got %d services, want 2", len(result.Services))
		}
	})

	t.Run("invalid yaml returns error", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte("{{invalid"), 0o644); err != nil {
			t.Fatal(err)
		}

		_, err := loadAzureYaml(dir)
		if err == nil {
			t.Error("expected error for invalid yaml")
		}
	})
}

func TestHostAllow(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := hostAllow(inner)

	tests := []struct {
		name       string
		host       string
		wantStatus int
	}{
		// Allowed: loopback hosts with explicit port.
		{"localhost with port", "localhost:8080", http.StatusOK},
		{"127.0.0.1 with port", "127.0.0.1:8080", http.StatusOK},
		{"[::1] with port", "[::1]:8080", http.StatusOK},
		// Allowed: loopback hosts without port (SplitHostPort fails → raw Host used).
		{"localhost without port", "localhost", http.StatusOK},
		{"127.0.0.1 without port", "127.0.0.1", http.StatusOK},
		{"[::1] without port", "[::1]", http.StatusOK},
		// Rejected: attacker-controlled or unexpected hosts.
		{"external domain with port", "attacker.com:8080", http.StatusForbidden},
		{"external domain without port", "attacker.com", http.StatusForbidden},
		{"loopback look-alike", "127.0.0.2:8080", http.StatusForbidden},
		{"empty host", "", http.StatusForbidden},
		{"subdomain of localhost", "evil.localhost:8080", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Host = tt.host
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Errorf("Host %q: status = %d, want %d", tt.host, rec.Code, tt.wantStatus)
			}
		})
	}
}

// TestHostAllow_PassesThrough verifies that an allowed host reaches the inner handler
// and that the inner handler's status code is preserved.
func TestHostAllow_PassesThrough(t *testing.T) {
	called := false
	handler := hostAllow(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "127.0.0.1:9000"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler should be called for an allowed host")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

// TestHostAllow_ViaBuiltHandler verifies that buildHandler() composes hostAllow
// outermost so that DNS rebinding protection (CWE-346) is enforced on both
// the primary-start path and the retry-port path.
func TestHostAllow_ViaBuiltHandler(t *testing.T) {
	s := &Server{mux: http.NewServeMux()}
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := s.buildHandler()

	t.Run("attacker.com:8080 returns 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "attacker.com:8080"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("attacker host: status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("localhost:8080 succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "localhost:8080"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("localhost host: status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("127.0.0.1:8080 succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "127.0.0.1:8080"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("127.0.0.1 host: status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("[::1]:8080 succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "[::1]:8080"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("[::1] host: status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("no port in Host header works correctly", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "localhost" // no port — SplitHostPort fails, raw host used
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("localhost (no port): status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

func TestTimeoutContext(t *testing.T) {
	ctx, cancel := timeoutContext(5 * time.Second)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context to have a deadline")
	}

	remaining := time.Until(deadline)
	if remaining < 4*time.Second || remaining > 6*time.Second {
		t.Errorf("deadline should be ~5s from now, got %v", remaining)
	}
}

// TestSecurityHeaders_CacheControl verifies SEC-028 (CWE-525): the
// securityHeaders middleware must set Cache-Control: no-store on API and
// Connect-RPC paths so browsers and shared proxies cannot cache sensitive
// response data.  Static asset paths must NOT receive that header so they
// continue to benefit from normal browser caching.
//
// Acceptance criteria:
//
//	AC1 — /api/shutdown receives Cache-Control: no-store
//	AC2 — Connect-RPC path (/azdapp.v1.*) receives Cache-Control: no-store
//	AC3 — root path (/) does NOT receive Cache-Control: no-store
//	AC4 — static asset paths do NOT receive Cache-Control: no-store
func TestSecurityHeaders_CacheControl(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantSet bool // true → header must equal "no-store"; false → header must be absent
	}{
		// AC1: REST API endpoint.
		{"api_shutdown_sets_no_store", "/api/shutdown", true},
		// Any other /api/ path must also be covered.
		{"api_prefix_sets_no_store", "/api/anything", true},
		// AC2: Connect-RPC paths (/azdapp.v1.<Service>/<Method>).
		{"connect_rpc_lifecycle_sets_no_store", "/azdapp.v1.LifecycleService/Ping", true},
		{"connect_rpc_health_sets_no_store", "/azdapp.v1.HealthService/GetHealth", true},
		// AC3: SPA root — must NOT get no-store (serveIndex handles it separately).
		{"root_does_not_set_no_store", "/", false},
		// AC4: static assets — must NOT get no-store so they remain cacheable.
		{"app_js_does_not_set_no_store", "/app.js", false},
		{"favicon_does_not_set_no_store", "/favicon.ico", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			got := rec.Header().Get("Cache-Control")
			if tt.wantSet && got != "no-store" {
				t.Errorf("path %q: Cache-Control = %q, want \"no-store\" (CWE-525)", tt.path, got)
			}
			if !tt.wantSet && got == "no-store" {
				t.Errorf("path %q: Cache-Control = \"no-store\", want not set (static assets must remain cacheable)", tt.path)
			}
		})
	}
}

// TestSecurityHeaders_CacheControl_OtherHeadersUnchanged confirms that adding
// the conditional Cache-Control header does not disturb any of the other
// security headers on API paths.
func TestSecurityHeaders_CacheControl_OtherHeadersUnchanged(t *testing.T) {
	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Cache-Control must be set.
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want \"no-store\"", got)
	}
	// All other security headers must still be present.
	checks := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for key, want := range checks {
		if got := rec.Header().Get(key); got != want {
			t.Errorf("header %q = %q, want %q", key, got, want)
		}
	}
	if csp := rec.Header().Get("Content-Security-Policy"); csp == "" {
		t.Error("Content-Security-Policy must still be set on API paths")
	}
}
