package logging

import (
	"bytes"
	"os"
	"testing"
)

func TestSetupLogger(t *testing.T) {
	tests := []struct {
		name       string
		debug      bool
		structured bool
	}{
		{"default logger", false, false},
		{"debug logger", true, false},
		{"structured logger", false, true},
		{"debug structured logger", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetupLogger(tt.debug, tt.structured)
			if globalLogger == nil {
				t.Error("Expected logger to be initialized")
			}
		})
	}
}

func TestSetLevel(t *testing.T) {
	tests := []struct {
		name  string
		level Level
	}{
		{"debug level", LevelDebug},
		{"info level", LevelInfo},
		{"warn level", LevelWarn},
		{"error level", LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLevel(tt.level)
			if currentLevel != tt.level {
				t.Errorf("Expected level %v, got %v", tt.level, currentLevel)
			}
		})
	}
}

func TestIsDebugEnabled(t *testing.T) {
	// Test with level
	SetLevel(LevelDebug)
	if !IsDebugEnabled() {
		t.Error("Expected debug to be enabled when level is Debug")
	}

	SetLevel(LevelInfo)
	if IsDebugEnabled() {
		t.Error("Expected debug to be disabled when level is Info")
	}

	// Test with environment variable
	os.Setenv("AZD_APP_DEBUG", "true")
	if !IsDebugEnabled() {
		t.Error("Expected debug to be enabled when AZD_APP_DEBUG=true")
	}
	os.Unsetenv("AZD_APP_DEBUG")
}

func TestLoggingFunctions(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	SetLevel(LevelDebug)

	tests := []struct {
		name    string
		logFunc func(string, ...any)
		message string
	}{
		{"debug", Debug, "debug message"},
		{"info", Info, "info message"},
		{"warn", Warn, "warning message"},
		{"error", Error, "error message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the function doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s() panicked: %v", tt.name, r)
				}
			}()
			
			buf.Reset()
			tt.logFunc(tt.message)
			// Just verify that the function executed without panicking
			// The actual output format is controlled by slog
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"warning", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"invalid", LevelInfo}, // Default to Info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseLevel(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestWith(t *testing.T) {
	logger := With("key", "value")
	if logger == nil {
		t.Error("Expected With to return a logger")
	}
}
