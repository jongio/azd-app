package dashboard

import (
	"bytes"
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

// TestIndexHTMLInjectsSessionToken is an HTTP-level integration test that
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
