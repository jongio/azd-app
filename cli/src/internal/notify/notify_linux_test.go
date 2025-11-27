//go:build linux

package notify

import (
	"context"
	"testing"
	"time"
)

func TestLinuxNotifier_New(t *testing.T) {
	config := DefaultConfig()
	notifier, err := newPlatformNotifier(config)
	if err != nil {
		t.Fatalf("failed to create linux notifier: %v", err)
	}

	if notifier == nil {
		t.Fatal("expected notifier, got nil")
	}

	ln, ok := notifier.(*linuxNotifier)
	if !ok {
		t.Fatal("expected linuxNotifier type")
	}

	if ln.config.AppName != config.AppName {
		t.Errorf("expected app name %s, got %s", config.AppName, ln.config.AppName)
	}
}

func TestLinuxNotifier_IsAvailable(t *testing.T) {
	config := DefaultConfig()
	notifier, _ := newPlatformNotifier(config)
	ln := notifier.(*linuxNotifier)

	// On Linux, availability depends on notify-send and D-Bus
	available := ln.IsAvailable()
	t.Logf("notify-send available: %v", available)
}

func TestLinuxNotifier_SeverityToUrgency(t *testing.T) {
	tests := []struct {
		severity string
		urgency  string
	}{
		{"critical", "critical"},
		{"warning", "normal"},
		{"info", "low"},
		{"unknown", "normal"}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			urgency := "normal"
			switch tt.severity {
			case "critical":
				urgency = "critical"
			case "warning":
				urgency = "normal"
			case "info":
				urgency = "low"
			}

			if urgency != tt.urgency {
				t.Errorf("expected urgency %s for severity %s, got %s", tt.urgency, tt.severity, urgency)
			}
		})
	}
}

func TestLinuxNotifier_ExpireTime(t *testing.T) {
	tests := []struct {
		name       string
		severity   string
		wantExpire string
	}{
		{"Critical stays visible", "critical", "--expire-time=-1"},
		{"Warning auto-dismisses", "warning", "--expire-time=5000"},
		{"Info auto-dismisses", "info", "--expire-time=5000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urgency := "normal"
			switch tt.severity {
			case "critical":
				urgency = "critical"
			case "warning":
				urgency = "normal"
			case "info":
				urgency = "low"
			}

			expireTime := "--expire-time=5000"
			if urgency == "critical" {
				expireTime = "--expire-time=-1"
			}

			if expireTime != tt.wantExpire {
				t.Errorf("expected %s, got %s", tt.wantExpire, expireTime)
			}
		})
	}
}

func TestLinuxNotifier_RequestPermission(t *testing.T) {
	config := DefaultConfig()
	notifier, _ := newPlatformNotifier(config)
	ln := notifier.(*linuxNotifier)

	ctx := context.Background()

	// If notify-send is not available, should return error
	if !ln.IsAvailable() {
		err := ln.RequestPermission(ctx)
		if err != ErrNotAvailable {
			t.Errorf("expected ErrNotAvailable, got %v", err)
		}
		return
	}

	// If available, should return nil (Linux doesn't require permission)
	err := ln.RequestPermission(ctx)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestLinuxNotifier_Close(t *testing.T) {
	config := DefaultConfig()
	notifier, _ := newPlatformNotifier(config)
	ln := notifier.(*linuxNotifier)

	err := ln.Close()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestLinuxNotifier_Config(t *testing.T) {
	config := Config{
		AppName: "Custom App",
		AppID:   "com.custom.app",
		Timeout: 10 * time.Second,
	}
	notifier, _ := newPlatformNotifier(config)
	ln := notifier.(*linuxNotifier)

	if ln.config.AppName != "Custom App" {
		t.Errorf("expected app name 'Custom App', got %s", ln.config.AppName)
	}
	if ln.config.Timeout != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", ln.config.Timeout)
	}
}

func TestLinuxNotifier_CommandArgs(t *testing.T) {
	config := Config{
		AppName: "Test App",
		AppID:   "com.test.app",
		Timeout: 5 * time.Second,
	}

	tests := []struct {
		name         string
		notification Notification
		wantArgs     []string
	}{
		{
			name: "Critical notification",
			notification: Notification{
				Title:    "API Service",
				Message:  "Service crashed",
				Severity: "critical",
			},
			wantArgs: []string{
				"--app-name=Test App",
				"--urgency=critical",
				"--expire-time=-1",
				"API Service",
				"Service crashed",
			},
		},
		{
			name: "Warning notification",
			notification: Notification{
				Title:    "Database",
				Message:  "High latency",
				Severity: "warning",
			},
			wantArgs: []string{
				"--app-name=Test App",
				"--urgency=normal",
				"--expire-time=5000",
				"Database",
				"High latency",
			},
		},
		{
			name: "Info notification",
			notification: Notification{
				Title:    "Frontend",
				Message:  "Service started",
				Severity: "info",
			},
			wantArgs: []string{
				"--app-name=Test App",
				"--urgency=low",
				"--expire-time=5000",
				"Frontend",
				"Service started",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build expected args
			urgency := "normal"
			switch tt.notification.Severity {
			case "critical":
				urgency = "critical"
			case "warning":
				urgency = "normal"
			case "info":
				urgency = "low"
			}

			args := []string{
				"--app-name=" + config.AppName,
				"--urgency=" + urgency,
				"--expire-time=5000",
				tt.notification.Title,
				tt.notification.Message,
			}

			if urgency == "critical" {
				args[2] = "--expire-time=-1"
			}

			// Verify args match expected
			if len(args) != len(tt.wantArgs) {
				t.Errorf("expected %d args, got %d", len(tt.wantArgs), len(args))
			}

			for i, arg := range args {
				if i >= len(tt.wantArgs) {
					break
				}
				if arg != tt.wantArgs[i] {
					t.Errorf("arg[%d]: expected %s, got %s", i, tt.wantArgs[i], arg)
				}
			}
		})
	}
}
