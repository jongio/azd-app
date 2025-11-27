package dashboard

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/healthcheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthStreamManagerCreation tests creating a new health stream manager.
func TestHealthStreamManagerCreation(t *testing.T) {
	// Create a temp directory for the test
	tmpDir := t.TempDir()

	// Create a minimal azure.yaml
	azureYaml := `name: test-project
services:
  api:
    language: node
    project: ./api
`
	err := os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte(azureYaml), 0644)
	require.NoError(t, err)

	manager, err := NewHealthStreamManager(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, tmpDir, manager.projectDir)
	assert.NotNil(t, manager.monitor)
	assert.NotNil(t, manager.previousStates)
}

// TestHealthStreamManagerDetectChanges tests change detection.
func TestHealthStreamManagerDetectChanges(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal azure.yaml
	azureYaml := `name: test-project
services:
  api:
    language: node
    project: ./api
`
	err := os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte(azureYaml), 0644)
	require.NoError(t, err)

	manager, err := NewHealthStreamManager(tmpDir)
	require.NoError(t, err)

	// First check - no previous state, no changes expected
	results := []healthcheck.HealthCheckResult{
		{
			ServiceName: "api",
			Status:      healthcheck.HealthStatusHealthy,
			CheckType:   healthcheck.HealthCheckTypeHTTP,
			Timestamp:   time.Now(),
		},
	}
	changes := manager.DetectChanges(results)
	assert.Empty(t, changes, "First check should not report changes")

	// Second check - same status, no changes
	changes = manager.DetectChanges(results)
	assert.Empty(t, changes, "Same status should not report changes")

	// Third check - status changed
	results[0].Status = healthcheck.HealthStatusUnhealthy
	results[0].Error = "connection refused"
	changes = manager.DetectChanges(results)
	require.Len(t, changes, 1, "Status change should be detected")
	assert.Equal(t, "api", changes[0].Service)
	assert.Equal(t, "healthy", changes[0].OldStatus)
	assert.Equal(t, "unhealthy", changes[0].NewStatus)
	assert.Equal(t, "connection refused", changes[0].Reason)
}

// TestHealthStreamManagerDetectChangesMultipleServices tests change detection with multiple services.
func TestHealthStreamManagerDetectChangesMultipleServices(t *testing.T) {
	tmpDir := t.TempDir()

	azureYaml := `name: test-project
services:
  api:
    language: node
  web:
    language: node
`
	err := os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte(azureYaml), 0644)
	require.NoError(t, err)

	manager, err := NewHealthStreamManager(tmpDir)
	require.NoError(t, err)

	// Initial check
	results := []healthcheck.HealthCheckResult{
		{ServiceName: "api", Status: healthcheck.HealthStatusHealthy},
		{ServiceName: "web", Status: healthcheck.HealthStatusHealthy},
	}
	changes := manager.DetectChanges(results)
	assert.Empty(t, changes)

	// Change one service
	results[0].Status = healthcheck.HealthStatusUnhealthy
	changes = manager.DetectChanges(results)
	require.Len(t, changes, 1)
	assert.Equal(t, "api", changes[0].Service)

	// Change both services
	results[0].Status = healthcheck.HealthStatusHealthy
	results[1].Status = healthcheck.HealthStatusDegraded
	changes = manager.DetectChanges(results)
	require.Len(t, changes, 2)
}

// TestParseHealthStreamParams tests parsing of query parameters.
func TestParseHealthStreamParams(t *testing.T) {
	tests := []struct {
		name           string
		queryString    string
		expectInterval time.Duration
		expectServices []string
		expectError    bool
	}{
		{
			name:           "default values",
			queryString:    "",
			expectInterval: defaultHealthInterval,
			expectServices: nil,
			expectError:    false,
		},
		{
			name:           "custom interval",
			queryString:    "interval=10s",
			expectInterval: 10 * time.Second,
			expectServices: nil,
			expectError:    false,
		},
		{
			name:           "single service",
			queryString:    "service=api",
			expectInterval: defaultHealthInterval,
			expectServices: []string{"api"},
			expectError:    false,
		},
		{
			name:           "multiple services",
			queryString:    "service=api,web,worker",
			expectInterval: defaultHealthInterval,
			expectServices: []string{"api", "web", "worker"},
			expectError:    false,
		},
		{
			name:           "interval below minimum",
			queryString:    "interval=500ms",
			expectInterval: minHealthInterval,
			expectServices: nil,
			expectError:    false,
		},
		{
			name:           "interval above maximum",
			queryString:    "interval=120s",
			expectInterval: maxHealthInterval,
			expectServices: nil,
			expectError:    false,
		},
		{
			name:           "services with spaces",
			queryString:    "service=api%2C%20web%20%2C%20worker",
			expectInterval: defaultHealthInterval,
			expectServices: []string{"api", "web", "worker"},
			expectError:    false,
		},
		{
			name:        "invalid interval format",
			queryString: "interval=invalid",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/health/stream?"+tc.queryString, nil)
			interval, services, err := parseHealthStreamParams(req)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectInterval, interval)
			assert.Equal(t, tc.expectServices, services)
		})
	}
}

// TestWriteSSEEvent tests SSE event writing.
func TestWriteSSEEvent(t *testing.T) {
	tests := []struct {
		name        string
		eventType   HealthEventType
		data        interface{}
		expectEvent bool
	}{
		{
			name:      "health event (default type)",
			eventType: HealthEventTypeHealth,
			data: HealthReportEvent{
				HealthEvent: HealthEvent{
					Type:      HealthEventTypeHealth,
					Timestamp: time.Now(),
				},
				Services: []healthcheck.HealthCheckResult{},
				Summary:  healthcheck.HealthSummary{Total: 0},
			},
			expectEvent: false, // default type doesn't include event: line
		},
		{
			name:      "heartbeat event",
			eventType: HealthEventTypeHeartbeat,
			data: HeartbeatEvent{
				HealthEvent: HealthEvent{
					Type:      HealthEventTypeHeartbeat,
					Timestamp: time.Now(),
				},
			},
			expectEvent: true,
		},
		{
			name:      "change event",
			eventType: HealthEventTypeChange,
			data: HealthChangeEvent{
				HealthEvent: HealthEvent{
					Type:      HealthEventTypeChange,
					Timestamp: time.Now(),
				},
				Service:   "api",
				OldStatus: "healthy",
				NewStatus: "unhealthy",
			},
			expectEvent: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			err := writeSSEEvent(rec, tc.eventType, tc.data)
			require.NoError(t, err)

			body := rec.Body.String()
			if tc.expectEvent {
				assert.Contains(t, body, "event: "+string(tc.eventType))
			} else {
				assert.NotContains(t, body, "event:")
			}
			assert.Contains(t, body, "data:")
			assert.True(t, strings.HasSuffix(body, "\n\n"))
		})
	}
}

// TestHandleHealthCheckEndpoint tests the /api/health REST endpoint.
func TestHandleHealthCheckEndpoint(t *testing.T) {
	tmpDir := t.TempDir()

	// Create azure.yaml
	azureYaml := `name: test-project
services:
  api:
    language: node
    project: ./api
`
	err := os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte(azureYaml), 0644)
	require.NoError(t, err)

	// Create server
	server := &Server{
		projectDir: tmpDir,
		mux:        http.NewServeMux(),
		stopChan:   make(chan struct{}),
	}

	t.Run("GET returns health report", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		rec := httptest.NewRecorder()

		server.handleHealthCheck(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var report healthcheck.HealthReport
		err := json.Unmarshal(rec.Body.Bytes(), &report)
		require.NoError(t, err)
		assert.NotNil(t, report.Summary)
	})

	t.Run("POST not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/health", nil)
		rec := httptest.NewRecorder()

		server.handleHealthCheck(rec, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})

	t.Run("service filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/health?service=api", nil)
		rec := httptest.NewRecorder()

		server.handleHealthCheck(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("custom timeout", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/health?timeout=3", nil)
		rec := httptest.NewRecorder()

		server.handleHealthCheck(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

// TestHandleHealthStreamEndpoint tests the /api/health/stream SSE endpoint.
func TestHandleHealthStreamEndpoint(t *testing.T) {
	tmpDir := t.TempDir()

	// Create azure.yaml
	azureYaml := `name: test-project
services:
  api:
    language: node
    project: ./api
`
	err := os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte(azureYaml), 0644)
	require.NoError(t, err)

	server := &Server{
		projectDir: tmpDir,
		mux:        http.NewServeMux(),
		stopChan:   make(chan struct{}),
	}

	t.Run("POST not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/health/stream", nil)
		rec := httptest.NewRecorder()

		server.handleHealthStream(rec, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})

	t.Run("invalid interval rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/health/stream?interval=invalid", nil)
		rec := httptest.NewRecorder()

		server.handleHealthStream(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("SSE headers set correctly", func(t *testing.T) {
		// Use a short-lived context to test initial connection
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		req := httptest.NewRequest(http.MethodGet, "/api/health/stream?interval=1s", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		// Run in goroutine since it blocks
		done := make(chan struct{})
		go func() {
			server.handleHealthStream(rec, req)
			close(done)
		}()

		// Wait for handler to complete or timeout
		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
		}

		assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
		assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
	})

	t.Run("receives initial health event", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		req := httptest.NewRequest(http.MethodGet, "/api/health/stream?interval=1s", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		done := make(chan struct{})
		go func() {
			server.handleHealthStream(rec, req)
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
		}

		body := rec.Body.String()
		// Should have received at least initial health event
		assert.Contains(t, body, "data:")
	})
}

// TestHealthEventTypes tests event type constants.
func TestHealthEventTypes(t *testing.T) {
	assert.Equal(t, HealthEventType("health"), HealthEventTypeHealth)
	assert.Equal(t, HealthEventType("health-change"), HealthEventTypeChange)
	assert.Equal(t, HealthEventType("heartbeat"), HealthEventTypeHeartbeat)
}

// TestHealthReportEventSerialization tests JSON serialization of health report event.
func TestHealthReportEventSerialization(t *testing.T) {
	event := HealthReportEvent{
		HealthEvent: HealthEvent{
			Type:      HealthEventTypeHealth,
			Timestamp: time.Date(2024, 11, 27, 10, 30, 0, 0, time.UTC),
		},
		Services: []healthcheck.HealthCheckResult{
			{
				ServiceName:  "api",
				Status:       healthcheck.HealthStatusHealthy,
				CheckType:    healthcheck.HealthCheckTypeHTTP,
				Endpoint:     "http://localhost:8080/health",
				ResponseTime: 45 * time.Millisecond,
				StatusCode:   200,
				Timestamp:    time.Date(2024, 11, 27, 10, 30, 0, 0, time.UTC),
			},
		},
		Summary: healthcheck.HealthSummary{
			Total:   1,
			Healthy: 1,
			Overall: healthcheck.HealthStatusHealthy,
		},
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var parsed HealthReportEvent
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, HealthEventTypeHealth, parsed.Type)
	assert.Len(t, parsed.Services, 1)
	assert.Equal(t, "api", parsed.Services[0].ServiceName)
	assert.Equal(t, 1, parsed.Summary.Total)
}

// TestHealthChangeEventSerialization tests JSON serialization of health change event.
func TestHealthChangeEventSerialization(t *testing.T) {
	event := HealthChangeEvent{
		HealthEvent: HealthEvent{
			Type:      HealthEventTypeChange,
			Timestamp: time.Date(2024, 11, 27, 10, 30, 0, 0, time.UTC),
		},
		Service:   "api",
		OldStatus: "healthy",
		NewStatus: "unhealthy",
		Reason:    "connection refused",
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var parsed HealthChangeEvent
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, HealthEventTypeChange, parsed.Type)
	assert.Equal(t, "api", parsed.Service)
	assert.Equal(t, "healthy", parsed.OldStatus)
	assert.Equal(t, "unhealthy", parsed.NewStatus)
	assert.Equal(t, "connection refused", parsed.Reason)
}

// TestHeartbeatEventSerialization tests JSON serialization of heartbeat event.
func TestHeartbeatEventSerialization(t *testing.T) {
	event := HeartbeatEvent{
		HealthEvent: HealthEvent{
			Type:      HealthEventTypeHeartbeat,
			Timestamp: time.Date(2024, 11, 27, 10, 30, 0, 0, time.UTC),
		},
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var parsed HeartbeatEvent
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, HealthEventTypeHeartbeat, parsed.Type)
}

// TestSSEEventFormat tests that SSE events are formatted correctly.
func TestSSEEventFormat(t *testing.T) {
	rec := httptest.NewRecorder()

	event := HeartbeatEvent{
		HealthEvent: HealthEvent{
			Type:      HealthEventTypeHeartbeat,
			Timestamp: time.Now(),
		},
	}

	err := writeSSEEvent(rec, HealthEventTypeHeartbeat, event)
	require.NoError(t, err)

	body := rec.Body.String()

	// Parse SSE format
	scanner := bufio.NewScanner(strings.NewReader(body))
	var eventLine, dataLine string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event:") {
			eventLine = line
		} else if strings.HasPrefix(line, "data:") {
			dataLine = line
		}
	}

	assert.Equal(t, "event: heartbeat", eventLine)
	assert.True(t, strings.HasPrefix(dataLine, "data: "))

	// Verify data is valid JSON
	jsonPart := strings.TrimPrefix(dataLine, "data: ")
	var parsed HeartbeatEvent
	err = json.Unmarshal([]byte(jsonPart), &parsed)
	require.NoError(t, err)
}

// TestHealthStreamConcurrency tests concurrent access to health stream manager.
func TestHealthStreamConcurrency(t *testing.T) {
	tmpDir := t.TempDir()

	azureYaml := `name: test-project
services:
  api:
    language: node
`
	err := os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte(azureYaml), 0644)
	require.NoError(t, err)

	manager, err := NewHealthStreamManager(tmpDir)
	require.NoError(t, err)

	// Simulate concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			results := []healthcheck.HealthCheckResult{
				{
					ServiceName: "api",
					Status:      healthcheck.HealthStatus("status-" + string(rune('0'+id))),
				},
			}
			manager.DetectChanges(results)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestHealthCheckConstants tests that constants have expected values.
func TestHealthCheckConstants(t *testing.T) {
	assert.Equal(t, 5*time.Second, defaultHealthInterval)
	assert.Equal(t, 1*time.Second, minHealthInterval)
	assert.Equal(t, 60*time.Second, maxHealthInterval)
	assert.Equal(t, 30*time.Second, heartbeatInterval)
	assert.Equal(t, 5*time.Second, healthCheckTimeout)
}
