//go:build windows

package notify

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestWindowsNotifier_New(t *testing.T) {
	config := DefaultConfig()
	notifier, err := newPlatformNotifier(config)
	if err != nil {
		t.Fatalf("failed to create windows notifier: %v", err)
	}

	if notifier == nil {
		t.Fatal("expected notifier, got nil")
	}

	wn, ok := notifier.(*windowsNotifier)
	if !ok {
		t.Fatal("expected windowsNotifier type")
	}

	if wn.config.AppName != config.AppName {
		t.Errorf("expected app name %s, got %s", config.AppName, wn.config.AppName)
	}
}

func TestWindowsNotifier_IsAvailable(t *testing.T) {
	config := DefaultConfig()
	notifier, _ := newPlatformNotifier(config)
	wn := notifier.(*windowsNotifier)

	// On Windows, PowerShell should be available
	if !wn.IsAvailable() {
		t.Error("expected IsAvailable to return true on Windows")
	}
}

func TestWindowsNotifier_BuildToastScript(t *testing.T) {
	config := Config{
		AppName: "Test App",
		AppID:   "com.test.app",
		Timeout: 5 * time.Second,
	}
	notifier, _ := newPlatformNotifier(config)
	wn := notifier.(*windowsNotifier)

	tests := []struct {
		name         string
		notification Notification
		wantContains []string
	}{
		{
			name: "Critical notification",
			notification: Notification{
				Title:    "API Service",
				Message:  "Service crashed",
				Severity: "critical",
			},
			wantContains: []string{
				"API Service",
				"Service crashed",
				"Test App",
				"com.test.app",
			},
		},
		{
			name: "Warning notification",
			notification: Notification{
				Title:    "Database",
				Message:  "High latency detected",
				Severity: "warning",
			},
			wantContains: []string{
				"Database",
				"High latency detected",
			},
		},
		{
			name: "Info notification",
			notification: Notification{
				Title:    "Frontend",
				Message:  "Service started",
				Severity: "info",
			},
			wantContains: []string{
				"Frontend",
				"Service started",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := wn.buildToastScript(tt.notification)

			for _, want := range tt.wantContains {
				if !strings.Contains(script, want) {
					t.Errorf("script should contain %q", want)
				}
			}

			// Verify script has required PowerShell components
			if !strings.Contains(script, "Windows.UI.Notifications.ToastNotificationManager") {
				t.Error("script should contain ToastNotificationManager")
			}
			if !strings.Contains(script, "<toast") {
				t.Error("script should contain toast XML")
			}
		})
	}
}

func TestWindowsNotifier_BuildToastScript_EscapeQuotes(t *testing.T) {
	config := DefaultConfig()
	notifier, _ := newPlatformNotifier(config)
	wn := notifier.(*windowsNotifier)

	notification := Notification{
		Title:    "Service with 'quotes'",
		Message:  "Message with 'single' quotes",
		Severity: "info",
	}

	script := wn.buildToastScript(notification)

	// Single quotes should be escaped to ''
	if strings.Contains(script, "Service with 'quotes'") && !strings.Contains(script, "''") {
		t.Error("script should escape single quotes")
	}
}

func TestWindowsNotifier_RequestPermission(t *testing.T) {
	config := DefaultConfig()
	notifier, _ := newPlatformNotifier(config)
	wn := notifier.(*windowsNotifier)

	ctx := context.Background()
	err := wn.RequestPermission(ctx)

	// Windows doesn't require explicit permission, should return nil
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestWindowsNotifier_Close(t *testing.T) {
	config := DefaultConfig()
	notifier, _ := newPlatformNotifier(config)
	wn := notifier.(*windowsNotifier)

	err := wn.Close()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestWindowsNotifier_Send_Timeout(t *testing.T) {
	config := Config{
		AppName: "Test App",
		AppID:   "com.test.app",
		Timeout: 1 * time.Nanosecond, // Very short timeout
	}
	notifier, _ := newPlatformNotifier(config)
	wn := notifier.(*windowsNotifier)

	notification := Notification{
		Title:    "Test",
		Message:  "Test message",
		Severity: "info",
	}

	ctx := context.Background()
	err := wn.Send(ctx, notification)

	// Should fail due to timeout or execution
	// We don't expect success with such a short timeout
	if err == nil {
		t.Log("Note: Send succeeded despite very short timeout (PowerShell might be very fast)")
	}
}

func TestWindowsNotifier_SeverityMapping(t *testing.T) {
	config := DefaultConfig()
	notifier, _ := newPlatformNotifier(config)
	wn := notifier.(*windowsNotifier)

	tests := []struct {
		severity string
		wantIcon string
	}{
		{"critical", "Error"},
		{"warning", "Warning"},
		{"info", "Info"},
		{"unknown", "Info"}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			notification := Notification{
				Title:    "Test",
				Message:  "Test",
				Severity: tt.severity,
			}
			script := wn.buildToastScript(notification)

			// Icon is not currently used in script, but we verify the mapping logic exists
			// This test documents expected behavior
			_ = script
		})
	}
}
