// Package notify provides examples of using the cross-platform notification system.
package notify_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/notify"
)

// Example demonstrates basic notification usage.
func Example() {
	// Create notifier with default config
	config := notify.DefaultConfig()
	notifier, err := notify.New(config)
	if err != nil {
		log.Fatalf("Failed to create notifier: %v", err)
	}
	defer notifier.Close()

	// Check if notifications are available
	if !notifier.IsAvailable() {
		fmt.Println("OS notifications not available")
		return
	}

	// Send a notification
	ctx := context.Background()
	notification := notify.Notification{
		Title:     "API Service",
		Message:   "Service has crashed",
		Severity:  "critical",
		Timestamp: time.Now(),
	}

	if err := notifier.Send(ctx, notification); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

// ExampleNotifier_Send demonstrates sending different severity notifications.
func ExampleNotifier_Send() {
	config := notify.DefaultConfig()
	notifier, err := notify.New(config)
	if err != nil {
		log.Fatalf("Failed to create notifier: %v", err)
	}
	defer notifier.Close()

	ctx := context.Background()

	// Critical notification
	critical := notify.Notification{
		Title:    "Database Service",
		Message:  "Database connection lost",
		Severity: "critical",
	}
	_ = notifier.Send(ctx, critical)

	// Warning notification
	warning := notify.Notification{
		Title:    "API Service",
		Message:  "High latency detected (500ms)",
		Severity: "warning",
	}
	_ = notifier.Send(ctx, warning)

	// Info notification
	info := notify.Notification{
		Title:    "Frontend Service",
		Message:  "Service started successfully",
		Severity: "info",
	}
	_ = notifier.Send(ctx, info)
}

// ExampleNotifier_RequestPermission demonstrates requesting notification permissions.
func ExampleNotifier_RequestPermission() {
	config := notify.DefaultConfig()
	notifier, err := notify.New(config)
	if err != nil {
		log.Fatalf("Failed to create notifier: %v", err)
	}
	defer notifier.Close()

	ctx := context.Background()

	// Request permission (triggers OS permission prompt on first use)
	if err := notifier.RequestPermission(ctx); err != nil {
		if err == notify.ErrNotAvailable {
			fmt.Println("Notifications not available on this system")
		} else if err == notify.ErrPermissionDenied {
			fmt.Println("User denied notification permissions")
		} else {
			log.Printf("Permission request failed: %v", err)
		}
		return
	}

	fmt.Println("Notification permissions granted")
}

// ExampleConfig demonstrates custom configuration.
func ExampleConfig() {
	config := notify.Config{
		AppName: "My Custom App",
		AppID:   "com.mycompany.myapp",
		Timeout: 10 * time.Second,
	}

	notifier, err := notify.New(config)
	if err != nil {
		log.Fatalf("Failed to create notifier: %v", err)
	}
	defer notifier.Close()

	ctx := context.Background()
	notification := notify.Notification{
		Title:    "Custom App",
		Message:  "This is a custom notification",
		Severity: "info",
	}

	if err := notifier.Send(ctx, notification); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

// ExampleNotification_withActions demonstrates notifications with action buttons.
func ExampleNotification_withActions() {
	config := notify.DefaultConfig()
	notifier, err := notify.New(config)
	if err != nil {
		log.Fatalf("Failed to create notifier: %v", err)
	}
	defer notifier.Close()

	ctx := context.Background()

	notification := notify.Notification{
		Title:    "Service Error",
		Message:  "API service has stopped responding",
		Severity: "critical",
		Actions: []notify.Action{
			{ID: "view", Label: "View Dashboard"},
			{ID: "restart", Label: "Restart Service"},
			{ID: "dismiss", Label: "Dismiss"},
		},
		Data: map[string]string{
			"service": "api-service",
			"pid":     "12345",
		},
	}

	if err := notifier.Send(ctx, notification); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

// ExampleNotifier_IsAvailable demonstrates checking notification availability.
func ExampleNotifier_IsAvailable() {
	config := notify.DefaultConfig()
	notifier, err := notify.New(config)
	if err != nil {
		log.Fatalf("Failed to create notifier: %v", err)
	}
	defer notifier.Close()

	if notifier.IsAvailable() {
		fmt.Println("OS notifications are available")
		// Proceed with sending notifications
	} else {
		fmt.Println("OS notifications not available, falling back to dashboard-only mode")
		// Use alternative notification method
	}
}
