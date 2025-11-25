// Package notifications provides the event pipeline for routing state changes to notifications
package notifications

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/config"
	"github.com/jongio/azd-app/cli/src/internal/monitor"
	"github.com/jongio/azd-app/cli/src/internal/notify"
	"github.com/jongio/azd-app/cli/src/internal/registry"
)

// NotificationManager integrates the state monitor with the notification pipeline.
// It watches for service state changes and sends OS notifications when appropriate.
type NotificationManager struct {
	stateMonitor *monitor.StateMonitor
	pipeline     *Pipeline
	osHandler    *OSNotificationHandler
	notifier     notify.Notifier
	registry     *registry.ServiceRegistry
	projectDir   string
	started      bool
}

// NotificationManagerConfig contains configuration for the notification manager.
type NotificationManagerConfig struct {
	ProjectDir      string
	MonitorInterval time.Duration
	BufferSize      int
}

// DefaultNotificationManagerConfig returns default configuration.
func DefaultNotificationManagerConfig(projectDir string) NotificationManagerConfig {
	return NotificationManagerConfig{
		ProjectDir:      projectDir,
		MonitorInterval: 5 * time.Second,
		BufferSize:      100,
	}
}

// NewNotificationManager creates a new notification manager that integrates
// service monitoring with OS notifications.
func NewNotificationManager(cfg NotificationManagerConfig) (*NotificationManager, error) {
	// Get the service registry for this project
	reg := registry.GetRegistry(cfg.ProjectDir)

	// Create the notification pipeline
	pipeline := NewPipeline(cfg.BufferSize)

	// Create OS notifier
	notifyConfig := notify.DefaultConfig()
	notifier, err := notify.New(notifyConfig)
	if err != nil {
		// Non-fatal: notifications will just be unavailable
		log.Printf("OS notifications unavailable: %v", err)
		notifier = nil
	}

	// Load notification preferences
	prefs := config.GetGlobalNotificationPreferences()

	// Create OS notification handler if notifier is available and enabled
	var osHandler *OSNotificationHandler
	if notifier != nil && prefs.OSNotifications && notifier.IsAvailable() {
		osHandler = NewOSNotificationHandler(notifier, prefs)
		pipeline.RegisterHandler(osHandler)
	}

	// Create state monitor
	monitorConfig := monitor.MonitorConfig{
		Interval:        cfg.MonitorInterval,
		MaxHistory:      1000,
		RateLimitWindow: prefs.GetRateLimitDuration(),
	}
	stateMonitor := monitor.NewStateMonitor(reg, monitorConfig)

	nm := &NotificationManager{
		stateMonitor: stateMonitor,
		pipeline:     pipeline,
		osHandler:    osHandler,
		notifier:     notifier,
		registry:     reg,
		projectDir:   cfg.ProjectDir,
	}

	// Connect state monitor to pipeline
	stateMonitor.AddListener(nm.handleStateTransition)

	return nm, nil
}

// Start begins monitoring and notification processing.
func (nm *NotificationManager) Start() {
	if nm.started {
		return
	}
	nm.started = true

	// Start the notification pipeline
	nm.pipeline.Start()

	// Start the state monitor
	nm.stateMonitor.Start()

	// Log notification status
	if nm.notifier != nil && nm.notifier.IsAvailable() {
		log.Printf("OS notifications enabled")
	} else {
		log.Printf("OS notifications disabled or unavailable")
	}
}

// Stop stops monitoring and notification processing.
func (nm *NotificationManager) Stop() error {
	if !nm.started {
		return nil
	}
	nm.started = false

	// Stop state monitor first
	nm.stateMonitor.Stop()

	// Stop pipeline
	if err := nm.pipeline.Stop(); err != nil {
		log.Printf("Error stopping pipeline: %v", err)
	}

	// Close OS handler
	if nm.osHandler != nil {
		if err := nm.osHandler.Close(); err != nil {
			log.Printf("Error closing OS handler: %v", err)
		}
	}

	// Close notifier
	if nm.notifier != nil {
		if err := nm.notifier.Close(); err != nil {
			log.Printf("Error closing notifier: %v", err)
		}
	}

	return nil
}

// handleStateTransition converts monitor transitions to notification events.
func (nm *NotificationManager) handleStateTransition(transition monitor.StateTransition) {
	// Convert monitor severity to notification severity string
	severity := "info"
	switch transition.Severity {
	case monitor.SeverityCritical:
		severity = "critical"
	case monitor.SeverityWarning:
		severity = "warning"
	case monitor.SeverityInfo:
		severity = "info"
	}

	// Create notification event
	event := Event{
		Type:        EventServiceStateChange,
		ServiceName: transition.ServiceName,
		OldState:    transition.FromState,
		NewState:    transition.ToState,
		Message:     transition.Description,
		Severity:    severity,
		Timestamp:   transition.Timestamp,
		Metadata:    make(map[string]interface{}),
	}

	// Add metadata
	if transition.ToState != nil {
		event.Metadata["status"] = transition.ToState.Status
		event.Metadata["health"] = transition.ToState.Health
		event.Metadata["pid"] = transition.ToState.PID
		event.Metadata["port"] = transition.ToState.Port
	}

	// Publish to pipeline
	if err := nm.pipeline.Publish(event); err != nil {
		log.Printf("Failed to publish notification event: %v", err)
	}
}

// SendTestNotification sends a test notification to verify OS notifications work.
func (nm *NotificationManager) SendTestNotification() error {
	if nm.notifier == nil {
		return fmt.Errorf("notifier not available")
	}

	if !nm.notifier.IsAvailable() {
		return fmt.Errorf("OS notifications not available on this system")
	}

	notification := notify.Notification{
		Title:     "Azure Dev Test",
		Message:   "Notifications are working!",
		Severity:  "info",
		Timestamp: time.Now(),
	}

	return nm.notifier.Send(context.Background(), notification)
}

// IsNotificationsEnabled returns true if OS notifications are enabled and available.
func (nm *NotificationManager) IsNotificationsEnabled() bool {
	return nm.notifier != nil && nm.notifier.IsAvailable() && nm.osHandler != nil
}

// SetDashboardURL sets the dashboard URL for clickable notifications.
// Call this after the dashboard server starts.
func (nm *NotificationManager) SetDashboardURL(url string) {
	if nm.osHandler != nil {
		nm.osHandler.SetDashboardURL(url)
	}
}

// GetHistory returns the state transition history.
func (nm *NotificationManager) GetHistory() []monitor.StateTransition {
	return nm.stateMonitor.GetHistory()
}
