// Package notify demonstrates integration between state monitor and notification system.
package notify_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/config"
	"github.com/jongio/azd-app/cli/src/internal/monitor"
	"github.com/jongio/azd-app/cli/src/internal/notify"
)

// ExampleNotificationPipeline demonstrates the complete notification pipeline.
func Example_notificationPipeline() {
	// Create notification system
	notifyConfig := notify.DefaultConfig()
	notifier, err := notify.New(notifyConfig)
	if err != nil {
		log.Fatalf("Failed to create notifier: %v", err)
	}
	defer notifier.Close()

	// Load notification preferences
	prefs, err := config.LoadNotificationPreferences()
	if err != nil {
		log.Printf("Failed to load preferences, using defaults: %v", err)
		prefs = config.DefaultNotificationPreferences()
	}

	// Simulate a state transition from the monitor
	transition := monitor.StateTransition{
		ServiceName: "api-service",
		Description: "Process crashed - Exit code 1",
		Severity:    monitor.SeverityCritical,
		Timestamp:   time.Now(),
	}

	// Check if we should notify
	if shouldNotifyForTransition(prefs, transition) {
		// Map severity
		severity := mapSeverity(transition.Severity)

		// Create notification
		notification := notify.Notification{
			Title:     transition.ServiceName,
			Message:   transition.Description,
			Severity:  severity,
			Timestamp: transition.Timestamp,
		}

		// Send OS notification if enabled and available
		if prefs.OSNotifications && notifier.IsAvailable() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := notifier.Send(ctx, notification); err != nil {
				log.Printf("Failed to send OS notification: %v", err)
			} else {
				fmt.Println("OS notification sent successfully")
			}
		}

		// Dashboard notifications would be sent via WebSocket
		if prefs.DashboardNotifications {
			fmt.Printf("Dashboard notification: %s - %s\n", notification.Title, notification.Message)
		}
	}
}

// shouldNotifyForTransition checks if a notification should be sent based on preferences.
func shouldNotifyForTransition(prefs *config.NotificationPreferences, transition monitor.StateTransition) bool {
	// Check if service is enabled
	if !prefs.IsServiceEnabled(transition.ServiceName) {
		return false
	}

	// Check severity filter
	severity := mapSeverity(transition.Severity)
	if !prefs.ShouldNotify(transition.ServiceName, severity) {
		return false
	}

	// Check quiet hours
	if prefs.IsInQuietHours() {
		// Allow critical notifications during quiet hours
		if transition.Severity != monitor.SeverityCritical {
			return false
		}
	}

	return true
}

// mapSeverity maps monitor severity to notification severity.
func mapSeverity(severity monitor.Severity) string {
	switch severity {
	case monitor.SeverityCritical:
		return "critical"
	case monitor.SeverityWarning:
		return "warning"
	case monitor.SeverityInfo:
		return "info"
	default:
		return "info"
	}
}

// ExampleCriticalServiceFailure demonstrates a critical service failure notification.
func Example_criticalServiceFailure() {
	// Create notifier
	notifier, err := notify.New(notify.DefaultConfig())
	if err != nil {
		log.Fatalf("Failed to create notifier: %v", err)
	}
	defer notifier.Close()

	// Simulate critical service failure
	notification := notify.Notification{
		Title:     "api-service",
		Message:   "Process crashed - Exit code 1",
		Severity:  "critical",
		Timestamp: time.Now(),
		Actions: []notify.Action{
			{ID: "view", Label: "View Dashboard"},
			{ID: "restart", Label: "Restart Service"},
		},
		Data: map[string]string{
			"service": "api-service",
			"pid":     "12345",
			"status":  "error",
		},
	}

	ctx := context.Background()
	if notifier.IsAvailable() {
		if err := notifier.Send(ctx, notification); err != nil {
			log.Printf("Failed to send notification: %v", err)
		} else {
			fmt.Println("Critical notification sent")
		}
	}
}

// ExampleWarningServiceDegraded demonstrates a warning notification.
func Example_warningServiceDegraded() {
	notifier, err := notify.New(notify.DefaultConfig())
	if err != nil {
		log.Fatalf("Failed to create notifier: %v", err)
	}
	defer notifier.Close()

	notification := notify.Notification{
		Title:     "database-service",
		Message:   "High latency detected (500ms average)",
		Severity:  "warning",
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	if notifier.IsAvailable() {
		if err := notifier.Send(ctx, notification); err != nil {
			log.Printf("Failed to send notification: %v", err)
		} else {
			fmt.Println("Warning notification sent")
		}
	}
}

// ExampleServiceRecovery demonstrates a recovery notification.
func Example_serviceRecovery() {
	notifier, err := notify.New(notify.DefaultConfig())
	if err != nil {
		log.Fatalf("Failed to create notifier: %v", err)
	}
	defer notifier.Close()

	notification := notify.Notification{
		Title:     "api-service",
		Message:   "Service recovered - Health check passing",
		Severity:  "info",
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	if notifier.IsAvailable() {
		if err := notifier.Send(ctx, notification); err != nil {
			log.Printf("Failed to send notification: %v", err)
		} else {
			fmt.Println("Recovery notification sent")
		}
	}
}

// ExamplePreferencesFiltering demonstrates preference-based filtering.
func Example_preferencesFiltering() {
	// Load preferences
	prefs, err := config.LoadNotificationPreferences()
	if err != nil {
		prefs = config.DefaultNotificationPreferences()
	}

	// Check different severity levels
	scenarios := []struct {
		severity string
		message  string
	}{
		{"critical", "Should always notify (default: critical only)"},
		{"warning", "May not notify if filter is 'critical'"},
		{"info", "May not notify if filter is 'critical' or 'warning'"},
	}

	for _, scenario := range scenarios {
		shouldNotify := prefs.ShouldNotify("test-service", scenario.severity)
		fmt.Printf("%s: %v - %s\n", scenario.severity, shouldNotify, scenario.message)
	}
}
