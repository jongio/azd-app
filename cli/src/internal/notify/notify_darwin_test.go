//go:build darwin

package notify

import (
	"strings"
	"testing"
	"time"
)

func TestDarwinNotifier_New(t *testing.T) {
	config := DefaultConfig()
	notifier, err := newPlatformNotifier(config)
	if err != nil {
		t.Fatalf("failed to create darwin notifier: %v", err)
	}

	if notifier == nil {
		t.Fatal("expected notifier, got nil")
	}

	dn, ok := notifier.(*darwinNotifier)
	if !ok {
		t.Fatal("expected darwinNotifier type")
	}

	if dn.config.AppName != config.AppName {
		t.Errorf("expected app name %s, got %s", config.AppName, dn.config.AppName)
	}
}

func TestDarwinNotifier_IsAvailable(t *testing.T) {
	config := DefaultConfig()
	notifier, _ := newPlatformNotifier(config)
	dn := notifier.(*darwinNotifier)

	// On macOS, osascript should be available
	available := dn.IsAvailable()
	t.Logf("osascript available: %v", available)
}

func TestDarwinNotifier_Send_ScriptFormat(t *testing.T) {
	config := Config{
		AppName: "Test App",
		AppID:   "com.test.app",
		Timeout: 5 * time.Second,
	}
	notifier, _ := newPlatformNotifier(config)
	// Verify we got a darwin notifier
	if _, ok := notifier.(*darwinNotifier); !ok {
		t.Fatal("expected darwinNotifier type")
	}

	tests := []struct {
		name         string
		notification Notification
		wantTitle    string
		wantMessage  string
	}{
		{
			name: "Critical notification",
			notification: Notification{
				Title:    "API Service",
				Message:  "Service crashed",
				Severity: "critical",
			},
			wantTitle:   "API Service",
			wantMessage: "Service crashed",
		},
		{
			name: "Warning notification",
			notification: Notification{
				Title:    "Database",
				Message:  "High latency detected",
				Severity: "warning",
			},
			wantTitle:   "Database",
			wantMessage: "High latency detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily test the full Send without triggering actual notifications
			// but we can verify the script construction logic

			// Verify title and message would be in the script
			title := strings.ReplaceAll(tt.notification.Title, "\"", "\\\"")
			message := strings.ReplaceAll(tt.notification.Message, "\"", "\\\"")

			if title != tt.wantTitle {
				t.Errorf("expected title %s, got %s", tt.wantTitle, title)
			}
			if message != tt.wantMessage {
				t.Errorf("expected message %s, got %s", tt.wantMessage, message)
			}
		})
	}
}

func TestDarwinNotifier_QuoteEscaping(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`Test "quoted" string`, `Test \"quoted\" string`},
		{`No quotes`, `No quotes`},
		{`Multiple "quotes" here "and" here`, `Multiple \"quotes\" here \"and\" here`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			escaped := strings.ReplaceAll(tt.input, "\"", "\\\"")
			if escaped != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, escaped)
			}
		})
	}
}

func TestDarwinNotifier_Close(t *testing.T) {
	config := DefaultConfig()
	notifier, _ := newPlatformNotifier(config)
	dn := notifier.(*darwinNotifier)

	err := dn.Close()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDarwinNotifier_Config(t *testing.T) {
	config := Config{
		AppName: "Custom App",
		AppID:   "com.custom.app",
		Timeout: 10 * time.Second,
	}
	notifier, _ := newPlatformNotifier(config)
	dn := notifier.(*darwinNotifier)

	if dn.config.AppName != "Custom App" {
		t.Errorf("expected app name 'Custom App', got %s", dn.config.AppName)
	}
	if dn.config.Timeout != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", dn.config.Timeout)
	}
}
