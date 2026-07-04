package commands

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-core/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildProxyRoutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		entries     []*registry.ServiceRegistryEntry
		expectError bool
		validate    func(t *testing.T, routes map[string]*url.URL)
	}{
		{
			name: "filters to running services with ports",
			entries: []*registry.ServiceRegistryEntry{
				{Name: "api", Status: constants.StatusRunning, Port: 5001},
				{Name: "web", Status: constants.StatusReady, Port: 3000, URL: "http://127.0.0.1:3000"},
				{Name: "worker", Status: constants.StatusStopped, Port: 7000},
				{Name: "queue", Status: constants.StatusRunning, Port: 0},
			},
			validate: func(t *testing.T, routes map[string]*url.URL) {
				require.Len(t, routes, 2)
				require.Contains(t, routes, "api")
				require.Contains(t, routes, "web")
				assert.Equal(t, "http://localhost:5001", routes["api"].String())
				assert.Equal(t, "http://127.0.0.1:3000", routes["web"].String())
			},
		},
		{
			name: "returns error when no eligible services are running",
			entries: []*registry.ServiceRegistryEntry{
				{Name: "api", Status: constants.StatusStopped, Port: 5001},
				{Name: "web", Status: constants.StatusRunning, Port: 0},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			routes, err := buildProxyRoutes(tt.entries)
			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, routes)
			}
		})
	}
}

func TestProxyHandlerStripsServicePrefix(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, r.URL.Path)
	}))
	t.Cleanup(upstream.Close)

	handler := newProxyHandler(map[string]*url.URL{
		"api": mustParseURL(t, upstream.URL),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "/users", recorder.Body.String())
}

func TestProxyHandlerForwardsRequestsToMatchingService(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "check=true", r.URL.RawQuery)
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, "forwarded")
	}))
	t.Cleanup(upstream.Close)

	handler := newProxyHandler(map[string]*url.URL{
		"web": mustParseURL(t, upstream.URL),
	})

	req := httptest.NewRequest(http.MethodGet, "/web/status?check=true", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, "forwarded", recorder.Body.String())
}

func TestProxyHandlerReturnsHelpfulNotFoundForUnknownRoute(t *testing.T) {
	t.Parallel()

	handler := newProxyHandler(map[string]*url.URL{
		"api": mustParseURL(t, "http://localhost:5001"),
		"web": mustParseURL(t, "http://localhost:3000"),
	})

	req := httptest.NewRequest(http.MethodGet, "/unknown/path", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Available routes")
	assert.Contains(t, recorder.Body.String(), "/api/")
	assert.Contains(t, recorder.Body.String(), "/web/")
}

func TestProxyHandlerReturnsBadGatewayWhenUpstreamUnavailable(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	upstreamURL := upstream.URL
	upstream.Close()

	handler := newProxyHandler(map[string]*url.URL{
		"api": mustParseURL(t, upstreamURL),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `service "api"`)
}

func TestNewProxyCommand(t *testing.T) {
	t.Parallel()

	cmd := NewProxyCommand()
	require.NotNil(t, cmd)

	portFlag := cmd.Flags().Lookup("port")
	require.NotNil(t, portFlag)
	assert.Equal(t, "int", portFlag.Value.Type())
	assert.Equal(t, "8080", portFlag.DefValue)
}

func TestProxyHandlerRootListsAvailableRoutes(t *testing.T) {
	t.Parallel()

	handler := newProxyHandler(map[string]*url.URL{
		"api": mustParseURL(t, "http://localhost:5001"),
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Available proxy routes")
	assert.Contains(t, recorder.Body.String(), "/api/ -> http://localhost:5001")
}

func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()

	parsedURL, err := url.Parse(rawURL)
	require.NoError(t, err)
	require.NotEmpty(t, strings.TrimSpace(parsedURL.Host))
	return parsedURL
}
