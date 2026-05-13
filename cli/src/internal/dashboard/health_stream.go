// Package dashboard provides health streaming capabilities for the dashboard.
package dashboard

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/healthcheck"
)

const (
	// healthCheckTimeout is the timeout for individual health checks
	healthCheckTimeout = 5 * time.Second
)

// HealthEventType represents the type of health event sent via SSE.
type HealthEventType string

// HealthEventTypeHealth and related constants identify the SSE event payloads emitted by health streaming.
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
