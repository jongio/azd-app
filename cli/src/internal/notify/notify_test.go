package notify

import (
	"context"
	"testing"
	"time"
)

func TestNotificationStruct(t *testing.T) {
	now := time.Now()
	n := Notification{
		Title:     "Test Service",
		Message:   "Service crashed",
		Severity:  "critical",
		Timestamp: now,
		Actions: []Action{
			{ID: "view", Label: "View Dashboard"},
			{ID: "dismiss", Label: "Dismiss"},
		},
		Data: map[string]string{
			"service": "test-service",
			"pid":     "1234",
		},
	}

	if n.Title != "Test Service" {
		t.Errorf("expected title 'Test Service', got %s", n.Title)
	}
	if n.Message != "Service crashed" {
		t.Errorf("expected message 'Service crashed', got %s", n.Message)
	}
	if n.Severity != "critical" {
		t.Errorf("expected severity 'critical', got %s", n.Severity)
	}
	if len(n.Actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(n.Actions))
	}
	if n.Data["service"] != "test-service" {
		t.Errorf("expected service 'test-service', got %s", n.Data["service"])
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.AppName != "Azure Developer CLI" {
		t.Errorf("expected app name 'Azure Developer CLI', got %s", config.AppName)
	}
	if config.AppID != "Microsoft.AzureDeveloperCLI" {
		t.Errorf("expected app ID 'Microsoft.AzureDeveloperCLI', got %s", config.AppID)
	}
	if config.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", config.Timeout)
	}
}

func TestCustomConfig(t *testing.T) {
	config := Config{
		AppName: "Custom App",
		AppID:   "com.custom.app",
		Timeout: 10 * time.Second,
	}

	if config.AppName != "Custom App" {
		t.Errorf("expected app name 'Custom App', got %s", config.AppName)
	}
	if config.Timeout != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", config.Timeout)
	}
}

func TestNotificationWithoutActions(t *testing.T) {
	n := Notification{
		Title:    "Test",
		Message:  "Message",
		Severity: "info",
	}

	if len(n.Actions) != 0 {
		t.Errorf("expected 0 actions, got %d", len(n.Actions))
	}
}

func TestNotificationWithoutData(t *testing.T) {
	n := Notification{
		Title:    "Test",
		Message:  "Message",
		Severity: "info",
	}

	if n.Data != nil {
		t.Errorf("expected nil data, got %v", n.Data)
	}
}

func TestAction(t *testing.T) {
	action := Action{
		ID:    "view",
		Label: "View Dashboard",
	}

	if action.ID != "view" {
		t.Errorf("expected ID 'view', got %s", action.ID)
	}
	if action.Label != "View Dashboard" {
		t.Errorf("expected label 'View Dashboard', got %s", action.Label)
	}
}

// TestNewNotifier verifies that New creates a platform-specific notifier
func TestNewNotifier(t *testing.T) {
	config := DefaultConfig()
	notifier, err := New(config)
	if err != nil {
		t.Fatalf("failed to create notifier: %v", err)
	}

	if notifier == nil {
		t.Fatal("expected notifier, got nil")
	}

	// Clean up
	if err := notifier.Close(); err != nil {
		t.Errorf("failed to close notifier: %v", err)
	}
}

// TestNotifierInterface verifies the notifier implements the interface correctly
func TestNotifierInterface(t *testing.T) {
	config := DefaultConfig()
	var _ Notifier = &mockNotifier{config: config}
}

// mockNotifier is a test implementation of Notifier
type mockNotifier struct {
	config        Config
	sendCalled    bool
	lastNotif     Notification
	available     bool
	permRequested bool
	closed        bool
}

func (m *mockNotifier) Send(ctx context.Context, notification Notification) error {
	m.sendCalled = true
	m.lastNotif = notification
	if !m.available {
		return ErrNotAvailable
	}
	return nil
}

func (m *mockNotifier) IsAvailable() bool {
	return m.available
}

func (m *mockNotifier) RequestPermission(ctx context.Context) error {
	m.permRequested = true
	if !m.available {
		return ErrNotAvailable
	}
	return nil
}

func (m *mockNotifier) Close() error {
	m.closed = true
	return nil
}

func TestMockNotifier(t *testing.T) {
	config := DefaultConfig()
	mock := &mockNotifier{config: config, available: true}

	ctx := context.Background()

	// Test Send
	notif := Notification{
		Title:    "Test",
		Message:  "Test message",
		Severity: "info",
	}
	if err := mock.Send(ctx, notif); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !mock.sendCalled {
		t.Error("expected Send to be called")
	}
	if mock.lastNotif.Title != "Test" {
		t.Errorf("expected title 'Test', got %s", mock.lastNotif.Title)
	}

	// Test IsAvailable
	if !mock.IsAvailable() {
		t.Error("expected available to be true")
	}

	// Test RequestPermission
	if err := mock.RequestPermission(ctx); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !mock.permRequested {
		t.Error("expected RequestPermission to be called")
	}

	// Test Close
	if err := mock.Close(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !mock.closed {
		t.Error("expected Close to be called")
	}
}

func TestMockNotifierUnavailable(t *testing.T) {
	config := DefaultConfig()
	mock := &mockNotifier{config: config, available: false}

	ctx := context.Background()

	// Test Send when unavailable
	notif := Notification{
		Title:    "Test",
		Message:  "Test message",
		Severity: "info",
	}
	err := mock.Send(ctx, notif)
	if err != ErrNotAvailable {
		t.Errorf("expected ErrNotAvailable, got %v", err)
	}

	// Test RequestPermission when unavailable
	err = mock.RequestPermission(ctx)
	if err != ErrNotAvailable {
		t.Errorf("expected ErrNotAvailable, got %v", err)
	}
}

func TestNotificationSeverityLevels(t *testing.T) {
	tests := []struct {
		name     string
		severity string
	}{
		{"Critical", "critical"},
		{"Warning", "warning"},
		{"Info", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := Notification{
				Title:    "Test",
				Message:  "Test message",
				Severity: tt.severity,
			}
			if n.Severity != tt.severity {
				t.Errorf("expected severity %s, got %s", tt.severity, n.Severity)
			}
		})
	}
}
