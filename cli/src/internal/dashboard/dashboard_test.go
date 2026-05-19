package dashboard

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
