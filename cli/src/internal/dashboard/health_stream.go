// Package dashboard provides health streaming capabilities for the dashboard.
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/healthcheck"
)

const (
	// defaultHealthInterval is the default interval between health checks
	defaultHealthInterval = 5 * time.Second

	// minHealthInterval is the minimum allowed interval
	minHealthInterval = 1 * time.Second

	// maxHealthInterval is the maximum allowed interval
	maxHealthInterval = 60 * time.Second

	// heartbeatInterval is how often to send heartbeat events
	heartbeatInterval = 30 * time.Second

	// healthCheckTimeout is the timeout for individual health checks
	healthCheckTimeout = 5 * time.Second
)

// HealthEventType represents the type of health event sent via SSE.
type HealthEventType string

const (
	HealthEventTypeHealth    HealthEventType = "health"
	HealthEventTypeChange    HealthEventType = "health-change"
	HealthEventTypeHeartbeat HealthEventType = "heartbeat"
)

// HealthEvent is the base event structure for SSE.
type HealthEvent struct {
	Type      HealthEventType `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
}

// HealthReportEvent contains the full health report.
type HealthReportEvent struct {
	HealthEvent
	Services []healthcheck.HealthCheckResult `json:"services"`
	Summary  healthcheck.HealthSummary       `json:"summary"`
}

// HealthChangeEvent indicates a health status change for a service.
type HealthChangeEvent struct {
	HealthEvent
	Service   string `json:"service"`
	OldStatus string `json:"oldStatus"`
	NewStatus string `json:"newStatus"`
	Reason    string `json:"reason,omitempty"`
}

// HeartbeatEvent is a keep-alive signal.
type HeartbeatEvent struct {
	HealthEvent
}

// HealthStreamManager manages health check streaming for the dashboard.
type HealthStreamManager struct {
	projectDir     string
	monitor        *healthcheck.HealthMonitor
	previousStates map[string]healthcheck.HealthStatus
	mu             sync.RWMutex
}

// NewHealthStreamManager creates a new health stream manager.
func NewHealthStreamManager(projectDir string) (*HealthStreamManager, error) {
	config := healthcheck.MonitorConfig{
		ProjectDir:      projectDir,
		DefaultEndpoint: "/health",
		Timeout:         healthCheckTimeout,
		Verbose:         false,
		LogLevel:        "warn",
		LogFormat:       "text",
	}

	monitor, err := healthcheck.NewHealthMonitor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create health monitor: %w", err)
	}

	return &HealthStreamManager{
		projectDir:     projectDir,
		monitor:        monitor,
		previousStates: make(map[string]healthcheck.HealthStatus),
	}, nil
}

// PerformHealthCheck performs a single health check and returns the report.
func (m *HealthStreamManager) PerformHealthCheck(ctx context.Context, serviceFilter []string) (*healthcheck.HealthReport, error) {
	return m.monitor.Check(ctx, serviceFilter)
}

// DetectChanges compares current results with previous states and returns changes.
func (m *HealthStreamManager) DetectChanges(results []healthcheck.HealthCheckResult) []HealthChangeEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	var changes []HealthChangeEvent
	now := time.Now()

	for _, result := range results {
		prevStatus, exists := m.previousStates[result.ServiceName]
		if exists && prevStatus != result.Status {
			change := HealthChangeEvent{
				HealthEvent: HealthEvent{
					Type:      HealthEventTypeChange,
					Timestamp: now,
				},
				Service:   result.ServiceName,
				OldStatus: string(prevStatus),
				NewStatus: string(result.Status),
			}
			if result.Error != "" {
				change.Reason = result.Error
			}
			changes = append(changes, change)
		}
		m.previousStates[result.ServiceName] = result.Status
	}

	return changes
}

// parseHealthStreamParams parses query parameters for health streaming.
func parseHealthStreamParams(r *http.Request) (interval time.Duration, serviceFilter []string, err error) {
	// Parse interval
	intervalStr := r.URL.Query().Get("interval")
	if intervalStr != "" {
		interval, err = time.ParseDuration(intervalStr)
		if err != nil {
			return 0, nil, fmt.Errorf("invalid interval format: %w", err)
		}
		if interval < minHealthInterval {
			interval = minHealthInterval
		}
		if interval > maxHealthInterval {
			interval = maxHealthInterval
		}
	} else {
		interval = defaultHealthInterval
	}

	// Parse service filter
	serviceStr := r.URL.Query().Get("service")
	if serviceStr != "" {
		services := strings.Split(serviceStr, ",")
		for _, s := range services {
			trimmed := strings.TrimSpace(s)
			if trimmed != "" {
				serviceFilter = append(serviceFilter, trimmed)
			}
		}
	}

	return interval, serviceFilter, nil
}

// writeSSEEvent writes a Server-Sent Event to the response.
func writeSSEEvent(w http.ResponseWriter, eventType HealthEventType, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	// Write event type if not the default "message" type
	if eventType != HealthEventTypeHealth {
		if _, err := fmt.Fprintf(w, "event: %s\n", eventType); err != nil {
			return fmt.Errorf("failed to write event type: %w", err)
		}
	}

	// Write data
	if _, err := fmt.Fprintf(w, "data: %s\n\n", jsonData); err != nil {
		return fmt.Errorf("failed to write event data: %w", err)
	}

	// Flush immediately
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

// handleHealthCheck handles GET /api/health for one-shot health checks.
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse service filter
	serviceStr := r.URL.Query().Get("service")
	var serviceFilter []string
	if serviceStr != "" {
		services := strings.Split(serviceStr, ",")
		for _, svc := range services {
			trimmed := strings.TrimSpace(svc)
			if trimmed != "" {
				serviceFilter = append(serviceFilter, trimmed)
			}
		}
	}

	// Parse timeout
	timeout := healthCheckTimeout
	timeoutStr := r.URL.Query().Get("timeout")
	if timeoutStr != "" {
		if parsed, err := strconv.Atoi(timeoutStr); err == nil && parsed > 0 && parsed <= 60 {
			timeout = time.Duration(parsed) * time.Second
		}
	}

	// Create health monitor for this request
	config := healthcheck.MonitorConfig{
		ProjectDir:      s.projectDir,
		DefaultEndpoint: "/health",
		Timeout:         timeout,
		Verbose:         false,
		LogLevel:        "warn",
		LogFormat:       "text",
	}

	monitor, err := healthcheck.NewHealthMonitor(config)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to create health monitor", err)
		return
	}

	// Perform health check with request context
	ctx, cancel := context.WithTimeout(r.Context(), timeout+time.Second)
	defer cancel()

	report, err := monitor.Check(ctx, serviceFilter)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Health check failed", err)
		return
	}

	if err := writeJSON(w, report); err != nil {
		log.Printf("Failed to write health report: %v", err)
	}
}

// handleHealthStream handles GET /api/health/stream for SSE health updates.
func (s *Server) handleHealthStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if client supports SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Parse parameters
	interval, serviceFilter, err := parseHealthStreamParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create health stream manager
	manager, err := NewHealthStreamManager(s.projectDir)
	if err != nil {
		http.Error(w, "Failed to initialize health monitor", http.StatusInternalServerError)
		return
	}

	// Create context that cancels when client disconnects
	ctx := r.Context()

	// Create tickers
	healthTicker := time.NewTicker(interval)
	defer healthTicker.Stop()

	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	// Send initial health check immediately
	if err := s.sendHealthUpdate(ctx, w, manager, serviceFilter); err != nil {
		log.Printf("Failed to send initial health update: %v", err)
		return
	}
	flusher.Flush()

	// Stream loop
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return

		case <-s.stopChan:
			// Server stopping
			return

		case <-healthTicker.C:
			if err := s.sendHealthUpdate(ctx, w, manager, serviceFilter); err != nil {
				log.Printf("Failed to send health update: %v", err)
				return
			}
			flusher.Flush()

		case <-heartbeatTicker.C:
			heartbeat := HeartbeatEvent{
				HealthEvent: HealthEvent{
					Type:      HealthEventTypeHeartbeat,
					Timestamp: time.Now(),
				},
			}
			if err := writeSSEEvent(w, HealthEventTypeHeartbeat, heartbeat); err != nil {
				log.Printf("Failed to send heartbeat: %v", err)
				return
			}
			flusher.Flush()
		}
	}
}

// sendHealthUpdate performs a health check and sends the results via SSE.
func (s *Server) sendHealthUpdate(ctx context.Context, w http.ResponseWriter, manager *HealthStreamManager, serviceFilter []string) error {
	// Create timeout context for health check
	checkCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	report, err := manager.PerformHealthCheck(checkCtx, serviceFilter)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	// Detect and send changes first
	changes := manager.DetectChanges(report.Services)
	for _, change := range changes {
		if err := writeSSEEvent(w, HealthEventTypeChange, change); err != nil {
			return fmt.Errorf("failed to send change event: %w", err)
		}
	}

	// Send full health report
	event := HealthReportEvent{
		HealthEvent: HealthEvent{
			Type:      HealthEventTypeHealth,
			Timestamp: report.Timestamp,
		},
		Services: report.Services,
		Summary:  report.Summary,
	}

	if err := writeSSEEvent(w, HealthEventTypeHealth, event); err != nil {
		return fmt.Errorf("failed to send health event: %w", err)
	}

	return nil
}
