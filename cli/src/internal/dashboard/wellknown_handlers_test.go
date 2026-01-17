package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/wellknown"
)

func TestHandleGetWellKnownServices(t *testing.T) {
	// Create server
	server := &Server{
		projectDir: t.TempDir(),
		mux:        http.NewServeMux(),
	}
	server.registerEditorRoutes()

	t.Run("returns all well-known services", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/editor/wellknown", nil)
		rec := httptest.NewRecorder()

		server.handleGetWellKnownServices(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response WellKnownServicesListResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response.Services) == 0 {
			t.Error("Expected services to be returned")
		}

		// Verify all services from registry are included
		if len(response.Services) != len(wellknown.Registry) {
			t.Errorf("Expected %d services, got %d", len(wellknown.Registry), len(response.Services))
		}
	})

	t.Run("services have required fields", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/editor/wellknown", nil)
		rec := httptest.NewRecorder()

		server.handleGetWellKnownServices(rec, req)

		var response WellKnownServicesListResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		for _, svc := range response.Services {
			if svc.Name == "" {
				t.Error("Service missing name")
			}
			if svc.DisplayName == "" {
				t.Error("Service missing display name")
			}
			if svc.Description == "" {
				t.Error("Service missing description")
			}
			if svc.Category == "" {
				t.Error("Service missing category")
			}
			if svc.Image == "" {
				t.Error("Service missing image")
			}
			if len(svc.Ports) == 0 {
				t.Errorf("Service %s missing ports", svc.Name)
			}
			if len(svc.ConnectionStrings) == 0 {
				t.Errorf("Service %s missing connection strings", svc.Name)
			}
		}
	})

	t.Run("includes icons and docs URLs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/editor/wellknown", nil)
		rec := httptest.NewRecorder()

		server.handleGetWellKnownServices(rec, req)

		var response WellKnownServicesListResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		foundWithIcon := false
		foundWithDocsURL := false

		for _, svc := range response.Services {
			if svc.Icon != "" {
				foundWithIcon = true
			}
			if svc.DocsURL != "" {
				foundWithDocsURL = true
			}
		}

		if !foundWithIcon {
			t.Error("Expected at least one service to have an icon")
		}
		if !foundWithDocsURL {
			t.Error("Expected at least one service to have a docs URL")
		}
	})
}

func TestHandleGetWellKnownService(t *testing.T) {
	server := newTestServer(t)

	t.Run("returns specific service by name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/editor/wellknown/azurite", nil)
		rec := httptest.NewRecorder()

		server.handleGetWellKnownService(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response WellKnownServiceResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Name != "azurite" {
			t.Errorf("Expected service name 'azurite', got %s", response.Name)
		}

		if response.DisplayName == "" {
			t.Error("Expected display name to be set")
		}
	})

	t.Run("returns 404 for nonexistent service", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/editor/wellknown/nonexistent", nil)
		rec := httptest.NewRecorder()

		server.handleGetWellKnownService(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for empty service name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/editor/wellknown/", nil)
		rec := httptest.NewRecorder()

		server.handleGetWellKnownService(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("includes all well-known service fields", func(t *testing.T) {
		// Test with azurite which has all fields populated
		req := httptest.NewRequest(http.MethodGet, "/api/editor/wellknown/azurite", nil)
		rec := httptest.NewRecorder()

		server.handleGetWellKnownService(rec, req)

		var response WellKnownServiceResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Name == "" {
			t.Error("Missing name")
		}
		if response.DisplayName == "" {
			t.Error("Missing display name")
		}
		if response.Description == "" {
			t.Error("Missing description")
		}
		if response.Category == "" {
			t.Error("Missing category")
		}
		if response.Image == "" {
			t.Error("Missing image")
		}
		if len(response.Ports) == 0 {
			t.Error("Missing ports")
		}
		if len(response.ConnectionStrings) == 0 {
			t.Error("Missing connection strings")
		}
		if response.Icon == "" {
			t.Error("Missing icon")
		}
		if response.DocsURL == "" {
			t.Error("Missing docs URL")
		}
	})
}

func TestHandleWellKnownRouter(t *testing.T) {
	server := newTestServer(t)

	t.Run("routes to list handler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/editor/wellknown", nil)
		rec := httptest.NewRecorder()

		server.handleWellKnownRouter(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response WellKnownServicesListResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response.Services) == 0 {
			t.Error("Expected services in response")
		}
	})

	t.Run("routes to specific service handler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/editor/wellknown/redis", nil)
		rec := httptest.NewRecorder()

		server.handleWellKnownRouter(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response WellKnownServiceResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Name != "redis" {
			t.Errorf("Expected service name 'redis', got %s", response.Name)
		}
	})

	t.Run("rejects non-GET methods", func(t *testing.T) {
		methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

		for _, method := range methods {
			req := httptest.NewRequest(method, "/api/editor/wellknown", nil)
			rec := httptest.NewRecorder()

			server.handleWellKnownRouter(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, rec.Code)
			}
		}
	})

	t.Run("returns 404 for invalid paths", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/editor/wellknown/invalid/path", nil)
		rec := httptest.NewRecorder()

		server.handleWellKnownRouter(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}

// newTestServer creates a server instance with editor routes registered for testing
func newTestServer(t *testing.T) *Server {
	t.Helper()
	srv := &Server{
		projectDir: t.TempDir(),
		mux:        http.NewServeMux(),
	}
	srv.registerEditorRoutes()
	return srv
}

func TestToWellKnownServiceResponse(t *testing.T) {
	azuriteDef := wellknown.Get("azurite")
	if azuriteDef == nil {
		t.Fatal("azurite service not found in registry")
	}

	response := toWellKnownServiceResponse(*azuriteDef)

	t.Run("converts basic fields", func(t *testing.T) {
		if response.Name != azuriteDef.Name {
			t.Errorf("Expected name %s, got %s", azuriteDef.Name, response.Name)
		}
		if response.DisplayName != azuriteDef.DisplayName {
			t.Errorf("Expected display name %s, got %s", azuriteDef.DisplayName, response.DisplayName)
		}
		if response.Description != azuriteDef.Description {
			t.Errorf("Expected description %s, got %s", azuriteDef.Description, response.Description)
		}
		if response.Category != azuriteDef.Category {
			t.Errorf("Expected category %s, got %s", azuriteDef.Category, response.Category)
		}
		if response.Image != azuriteDef.Image {
			t.Errorf("Expected image %s, got %s", azuriteDef.Image, response.Image)
		}
	})

	t.Run("adds icon from map", func(t *testing.T) {
		if response.Icon == "" {
			t.Error("Expected icon to be set")
		}
		if expectedIcon, ok := serviceIcons[azuriteDef.Name]; ok {
			if response.Icon != expectedIcon {
				t.Errorf("Expected icon %s, got %s", expectedIcon, response.Icon)
			}
		}
	})

	t.Run("adds docs URL from map", func(t *testing.T) {
		if response.DocsURL == "" {
			t.Error("Expected docs URL to be set")
		}
		if expectedURL, ok := serviceDocsURLs[azuriteDef.Name]; ok {
			if response.DocsURL != expectedURL {
				t.Errorf("Expected docs URL %s, got %s", expectedURL, response.DocsURL)
			}
		}
	})

	t.Run("includes ports and environment", func(t *testing.T) {
		if len(response.Ports) != len(azuriteDef.Ports) {
			t.Errorf("Expected %d ports, got %d", len(azuriteDef.Ports), len(response.Ports))
		}
		if len(response.Environment) != len(azuriteDef.Environment) {
			t.Errorf("Expected %d environment variables, got %d", len(azuriteDef.Environment), len(response.Environment))
		}
	})

	t.Run("includes connection strings", func(t *testing.T) {
		if len(response.ConnectionStrings) == 0 {
			t.Error("Expected connection strings to be set")
		}
		if len(response.ConnectionStrings) != len(azuriteDef.ConnectionStrings) {
			t.Errorf("Expected %d connection strings, got %d", len(azuriteDef.ConnectionStrings), len(response.ConnectionStrings))
		}
	})

	t.Run("includes healthcheck if present", func(t *testing.T) {
		if azuriteDef.Healthcheck != nil && response.Healthcheck == nil {
			t.Error("Expected healthcheck to be included")
		}
	})
}
