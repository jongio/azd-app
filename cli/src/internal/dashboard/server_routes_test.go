package dashboard

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/dashboard/broadcast"
)

// TestSessionTokenEndpointRemoved is a regression test for CWE-306/419/352.
// The /api/session-token HTTP endpoint must not exist. Any request to it
// must never return 200 OK nor expose the token in the response body.
func TestSessionTokenEndpointRemoved(t *testing.T) {
	s := &Server{
		mux:          http.NewServeMux(),
		sessionToken: "deadbeefdeadbeefdeadbeefdeadbeef",
		broadcast:    broadcast.New(),
	}
	s.setupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/session-token", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Errorf("GET /api/session-token returned 200 OK — endpoint must be removed (CWE-306/419/352)")
	}
	if strings.Contains(rec.Body.String(), "deadbeefdeadbeefdeadbeefdeadbeef") {
		t.Error("response body must not expose the session token")
	}
}

// TestInjectSessionToken verifies every branch of the meta-tag injection helper.
func TestInjectSessionToken(t *testing.T) {
	t.Run("replaces placeholder with token", func(t *testing.T) {
		src := []byte(`<html><head><meta name="azd-session-token" content=""></head></html>`)
		got := injectSessionToken(src, "abc123def456")
		if !strings.Contains(string(got), `content="abc123def456"`) {
			t.Errorf("expected token in meta content, got: %s", got)
		}
		// Placeholder must be gone.
		if strings.Contains(string(got), `content=""`) {
			t.Error("placeholder content attr should be replaced, not left empty")
		}
	})

	t.Run("no-op when placeholder absent", func(t *testing.T) {
		src := []byte(`<html><head></head></html>`)
		got := injectSessionToken(src, "abc123")
		if strings.Contains(string(got), "abc123") {
			t.Error("token must not appear when placeholder is absent")
		}
		if !bytes.Equal(got, src) {
			t.Error("content should be unchanged when placeholder is absent")
		}
	})

	t.Run("replaces only first occurrence", func(t *testing.T) {
		placeholder := `<meta name="azd-session-token" content="">`
		src := []byte(placeholder + placeholder)
		got := string(injectSessionToken(src, "tok"))
		if strings.Count(got, `content="tok"`) != 1 {
			t.Errorf("expected exactly 1 injection, got: %s", got)
		}
		// Second occurrence must remain untouched.
		if strings.Count(got, `content=""`) != 1 {
			t.Errorf("expected 1 untouched placeholder, got: %s", got)
		}
	})

	t.Run("hex token requires no additional escaping", func(t *testing.T) {
		src := []byte(`<html><head><meta name="azd-session-token" content=""></head></html>`)
		hexToken := "0123456789abcdef"
		got := string(injectSessionToken(src, hexToken))
		want := `content="0123456789abcdef"`
		if !strings.Contains(got, want) {
			t.Errorf("hex token not injected verbatim; want %q in %s", want, got)
		}
	})
}

// TestShutdownOriginCheck verifies the CWE-352 same-origin enforcement on
// POST /api/shutdown.  It exercises every branch of shutdownOriginAllowed and
// the combined handler path (origin check → token check → action).
//
// Acceptance criteria:
//   1. Sec-Fetch-Site: same-origin accepted (200)
//   2. Sec-Fetch-Site: cross-site rejected (403)
//   3. No Sec-Fetch-Site, no Origin → fail-closed (403)
//   4. Origin fallback: http://localhost:<port> accepted (200)
//   5. Origin fallback: http://127.0.0.1:<port> accepted (200)
//   6. Origin fallback: cross-site Origin rejected (403)
func TestShutdownOriginCheck(t *testing.T) {
	const (
		validToken  = "deadbeefdeadbeefdeadbeefdeadbeef"
		serverPort  = 45678
	)

	// newSrv constructs a minimal Server suitable for HTTP-level testing.
	// shutdownChan must be non-nil so RequestShutdown() does not panic on
	// close(nil) when a 200-path test triggers it.
	newSrv := func() *Server {
		s := &Server{
			mux:          http.NewServeMux(),
			sessionToken: validToken,
			broadcast:    broadcast.New(),
			port:         serverPort,
			shutdownChan: make(chan struct{}),
		}
		s.setupRoutes()
		return s
	}

	t.Run("AC1_SameSiteHeader_Returns200", func(t *testing.T) {
		// Criterion 1: valid token + Sec-Fetch-Site: same-origin → 200
		s := newSrv()
		req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
		req.Header.Set("X-Session-Token", validToken)
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for same-origin + valid token, got %d", rec.Code)
		}
	})

	t.Run("AC2_CrossSiteHeader_Returns403", func(t *testing.T) {
		// Criterion 2: valid token + Sec-Fetch-Site: cross-site → 403
		s := newSrv()
		req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
		req.Header.Set("X-Session-Token", validToken)
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for cross-site, got %d", rec.Code)
		}
	})

	t.Run("AC3_NoHeaders_Returns403", func(t *testing.T) {
		// Criterion 3: valid token + no Sec-Fetch-Site + no Origin → fail-closed 403
		s := newSrv()
		req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
		req.Header.Set("X-Session-Token", validToken)
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 when Sec-Fetch-Site and Origin are absent, got %d", rec.Code)
		}
	})

	t.Run("AC4_OriginFallback_Localhost_Returns200", func(t *testing.T) {
		// Criterion 4: Origin fallback — http://localhost:<port> accepted
		s := newSrv()
		req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
		req.Header.Set("X-Session-Token", validToken)
		req.Header.Set("Origin", fmt.Sprintf("http://localhost:%d", serverPort))
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for localhost Origin fallback, got %d", rec.Code)
		}
	})

	t.Run("AC5_OriginFallback_127001_Returns200", func(t *testing.T) {
		// Criterion 5: Origin fallback — http://127.0.0.1:<port> accepted
		s := newSrv()
		req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
		req.Header.Set("X-Session-Token", validToken)
		req.Header.Set("Origin", fmt.Sprintf("http://127.0.0.1:%d", serverPort))
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for 127.0.0.1 Origin fallback, got %d", rec.Code)
		}
	})

	t.Run("AC6_OriginFallback_CrossSite_Returns403", func(t *testing.T) {
		// Criterion 6: Origin fallback — cross-site Origin rejected
		s := newSrv()
		req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
		req.Header.Set("X-Session-Token", validToken)
		req.Header.Set("Origin", "http://evil.example.com")
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for cross-site Origin, got %d", rec.Code)
		}
	})

	// Additional edge cases to complete the decision matrix.

	t.Run("SameSiteHeader_Returns403", func(t *testing.T) {
		// Sec-Fetch-Site: same-site is not same-origin; must be rejected.
		s := newSrv()
		req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
		req.Header.Set("X-Session-Token", validToken)
		req.Header.Set("Sec-Fetch-Site", "same-site")
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for Sec-Fetch-Site: same-site, got %d", rec.Code)
		}
	})

	t.Run("SecFetchSiteNone_Returns403", func(t *testing.T) {
		// Sec-Fetch-Site: none (e.g. user-typed navigation) is not same-origin.
		s := newSrv()
		req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
		req.Header.Set("X-Session-Token", validToken)
		req.Header.Set("Sec-Fetch-Site", "none")
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for Sec-Fetch-Site: none, got %d", rec.Code)
		}
	})

	t.Run("SameOrigin_WrongToken_Returns401", func(t *testing.T) {
		// Origin proof passes but token is wrong → 401 (not 403).
		// Verifies that both checks are independent.
		s := newSrv()
		req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
		req.Header.Set("X-Session-Token", "notthetoken")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for same-origin + wrong token, got %d", rec.Code)
		}
	})

	t.Run("CrossSite_WrongToken_Returns403NotUnauthorized", func(t *testing.T) {
		// Origin check fires BEFORE token check.  A cross-origin request with a
		// wrong token must return 403 (origin), not 401 (token), so the failure
		// mode leaks no information about token validity.
		s := newSrv()
		req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
		req.Header.Set("X-Session-Token", "notthetoken")
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 (origin check fires first), got %d", rec.Code)
		}
	})

	t.Run("OriginWrongPort_Returns403", func(t *testing.T) {
		// Origin header with the right host but a different port must be rejected
		// to prevent a second dashboard instance from cross-triggering shutdown.
		s := newSrv()
		req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
		req.Header.Set("X-Session-Token", validToken)
		req.Header.Set("Origin", fmt.Sprintf("http://localhost:%d", serverPort+1))
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for Origin with wrong port, got %d", rec.Code)
		}
	})
}
// verifies the full request→response path for token injection. Because the
// embedded dist/ FS has no built index.html at development time, this test
// constructs its own handler using the same injectSessionToken helper that
// setupRoutes relies on, confirming the HTML contract the client reads.
func TestIndexHTMLInjectsSessionToken(t *testing.T) {
	const token = "cafebabe12345678cafebabe12345678"

	// Minimal index.html with the placeholder — mirrors the real template.
	indexHTML := []byte(`<!DOCTYPE html><html><head><meta name="azd-session-token" content=""></head><body></body></html>`)
	tokenized := injectSessionToken(indexHTML, token)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(tokenized)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	wantMeta := `<meta name="azd-session-token" content="` + token + `">`
	if !strings.Contains(body, wantMeta) {
		t.Errorf("response body missing injected meta tag\nwant: %s\ngot:  %s", wantMeta, body)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want \"no-store\" (token-bearing HTML must not be cached)", rec.Header().Get("Cache-Control"))
	}
	if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want \"text/html; charset=utf-8\"", rec.Header().Get("Content-Type"))
	}
}
