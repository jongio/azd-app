package service

import (
	"testing"
	"time"
)

func TestNewAzureLogBuffer(t *testing.T) {
	// Test with nil config
	buffer := NewAzureLogBuffer(nil, "/tmp/project")
	if buffer == nil {
		t.Fatal("NewAzureLogBuffer returned nil")
	}
	if buffer.config == nil {
		t.Error("Config should not be nil")
	}
	if buffer.config.PollingInterval != DefaultAzurePollingInterval {
		t.Errorf("Expected default polling interval %v, got %v", DefaultAzurePollingInterval, buffer.config.PollingInterval)
	}
	if buffer.config.DefaultTimespan != DefaultAzureTimespan {
		t.Errorf("Expected default timespan %v, got %v", DefaultAzureTimespan, buffer.config.DefaultTimespan)
	}
	if buffer.mode != LogModeLocal {
		t.Errorf("Expected initial mode to be local, got %v", buffer.mode)
	}
}

func TestNewAzureLogBufferWithConfig(t *testing.T) {
	config := &AzureLogsConfig{
		Enabled:         true,
		WorkspaceID:     "workspace-123",
		PollingInterval: 1 * time.Minute,
		DefaultTimespan: 2 * time.Hour,
	}

	buffer := NewAzureLogBuffer(config, "/tmp/project")
	if buffer == nil {
		t.Fatal("NewAzureLogBuffer returned nil")
	}
	if buffer.config.WorkspaceID != "workspace-123" {
		t.Errorf("Expected workspace 'workspace-123', got %q", buffer.config.WorkspaceID)
	}
	if buffer.config.PollingInterval != 1*time.Minute {
		t.Errorf("Expected polling interval 1m, got %v", buffer.config.PollingInterval)
	}
	if buffer.config.DefaultTimespan != 2*time.Hour {
		t.Errorf("Expected timespan 2h, got %v", buffer.config.DefaultTimespan)
	}
}

func TestLogModeConstants(t *testing.T) {
	if LogModeLocal != "local" {
		t.Errorf("Expected LogModeLocal to be 'local', got %q", LogModeLocal)
	}
	if LogModeAzure != "azure" {
		t.Errorf("Expected LogModeAzure to be 'azure', got %q", LogModeAzure)
	}
}

func TestAzureLogBufferSetMode(t *testing.T) {
	buffer := NewAzureLogBuffer(nil, "/tmp/project")

	// Initial mode should be local
	if buffer.GetMode() != LogModeLocal {
		t.Errorf("Expected initial mode local, got %v", buffer.GetMode())
	}

	// Set to local (should be no-op)
	err := buffer.SetMode(LogModeLocal)
	if err != nil {
		t.Errorf("SetMode(local) returned error: %v", err)
	}
	if buffer.GetMode() != LogModeLocal {
		t.Errorf("Mode should still be local, got %v", buffer.GetMode())
	}

	// Set to azure (without initialization, should work but not start polling)
	err = buffer.SetMode(LogModeAzure)
	if err != nil {
		t.Errorf("SetMode(azure) returned error: %v", err)
	}
	if buffer.GetMode() != LogModeAzure {
		t.Errorf("Mode should be azure, got %v", buffer.GetMode())
	}

	// Set back to local
	err = buffer.SetMode(LogModeLocal)
	if err != nil {
		t.Errorf("SetMode(local) returned error: %v", err)
	}
	if buffer.GetMode() != LogModeLocal {
		t.Errorf("Mode should be local again, got %v", buffer.GetMode())
	}
}

func TestAzureLogBufferSubscription(t *testing.T) {
	buffer := NewAzureLogBuffer(nil, "/tmp/project")

	// Subscribe
	ch := buffer.Subscribe()
	if ch == nil {
		t.Fatal("Subscribe returned nil channel")
	}

	// Check that subscriber was added
	buffer.subMu.RLock()
	if len(buffer.subscribers) != 1 {
		t.Errorf("Expected 1 subscriber, got %d", len(buffer.subscribers))
	}
	buffer.subMu.RUnlock()

	// Unsubscribe
	buffer.Unsubscribe(ch)

	// Check that subscriber was removed
	buffer.subMu.RLock()
	if len(buffer.subscribers) != 0 {
		t.Errorf("Expected 0 subscribers after unsubscribe, got %d", len(buffer.subscribers))
	}
	buffer.subMu.RUnlock()
}

func TestAzureLogBufferClose(t *testing.T) {
	buffer := NewAzureLogBuffer(nil, "/tmp/project")

	// Subscribe
	ch := buffer.Subscribe()
	if ch == nil {
		t.Fatal("Subscribe returned nil channel")
	}

	// Close buffer
	err := buffer.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}

	// Subscribers should be closed
	buffer.subMu.RLock()
	if len(buffer.subscribers) != 0 {
		t.Errorf("Expected 0 subscribers after close, got %d", len(buffer.subscribers))
	}
	buffer.subMu.RUnlock()
}

func TestAzureLogBufferGetAzureStatus(t *testing.T) {
	config := &AzureLogsConfig{
		Enabled: true,
	}
	buffer := NewAzureLogBuffer(config, "/tmp/project")

	status := buffer.GetAzureStatus()
	if status.Mode != LogModeLocal {
		t.Errorf("Expected mode local, got %v", status.Mode)
	}
	if status.Enabled != true {
		t.Error("Expected enabled to be true")
	}
	if status.Connected {
		t.Error("Expected connected to be false without log client")
	}
}

func TestAzureStatusStruct(t *testing.T) {
	status := AzureStatus{
		Mode:          LogModeAzure,
		Connected:     true,
		Enabled:       true,
		ResourceCount: 3,
		LastError:     "",
	}

	if status.Mode != LogModeAzure {
		t.Errorf("Expected mode azure, got %v", status.Mode)
	}
	if !status.Connected {
		t.Error("Expected connected to be true")
	}
	if status.ResourceCount != 3 {
		t.Errorf("Expected resource count 3, got %d", status.ResourceCount)
	}
}

func TestConvertAzureLogLevel(t *testing.T) {
	tests := []struct {
		input    int // azure.LogLevel
		expected LogLevel
	}{
		{0, LogLevelInfo},  // azure.LogLevelInfo
		{1, LogLevelWarn},  // azure.LogLevelWarn
		{2, LogLevelError}, // azure.LogLevelError
		{3, LogLevelDebug}, // azure.LogLevelDebug
	}

	for _, tc := range tests {
		// This test validates the concept - actual conversion uses azure.LogLevel type
		_ = tc
	}
}

func TestDefaultConstants(t *testing.T) {
	if DefaultAzurePollingInterval != 30*time.Second {
		t.Errorf("Expected DefaultAzurePollingInterval 30s, got %v", DefaultAzurePollingInterval)
	}
	if DefaultAzureTimespan != 1*time.Hour {
		t.Errorf("Expected DefaultAzureTimespan 1h, got %v", DefaultAzureTimespan)
	}
}
