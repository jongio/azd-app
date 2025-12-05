package commands

import (
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestParseLogLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		serviceName string
		wantErr     bool
		wantLevel   service.LogLevel
		wantStderr  bool
		wantMsg     string
	}{
		{
			name:        "valid info log",
			line:        "[2024-01-15 10:30:45.123] [INFO] [OUT] Server started on port 3000",
			serviceName: "api",
			wantErr:     false,
			wantLevel:   service.LogLevelInfo,
			wantStderr:  false,
			wantMsg:     "Server started on port 3000",
		},
		{
			name:        "valid error log with stderr",
			line:        "[2024-01-15 10:30:45.123] [ERROR] [ERR] Connection failed",
			serviceName: "db",
			wantErr:     false,
			wantLevel:   service.LogLevelError,
			wantStderr:  true,
			wantMsg:     "Connection failed",
		},
		{
			name:        "valid warn log",
			line:        "[2024-01-15 10:30:45.123] [WARN] [OUT] Deprecated function called",
			serviceName: "web",
			wantErr:     false,
			wantLevel:   service.LogLevelWarn,
			wantStderr:  false,
			wantMsg:     "Deprecated function called",
		},
		{
			name:        "valid debug log",
			line:        "[2024-01-15 10:30:45.123] [DEBUG] [OUT] Processing request id=123",
			serviceName: "api",
			wantErr:     false,
			wantLevel:   service.LogLevelDebug,
			wantStderr:  false,
			wantMsg:     "Processing request id=123",
		},
		{
			name:        "invalid - no timestamp bracket",
			line:        "2024-01-15 10:30:45.123] [INFO] [OUT] Message",
			serviceName: "api",
			wantErr:     true,
		},
		{
			name:        "invalid - too short",
			line:        "[2024-01-15]",
			serviceName: "api",
			wantErr:     true,
		},
		{
			name:        "invalid - empty line",
			line:        "",
			serviceName: "api",
			wantErr:     true,
		},
		{
			name:        "partial format - only timestamp",
			line:        "[2024-01-15 10:30:45.123] Some message without level",
			serviceName: "api",
			wantErr:     false,
			wantLevel:   service.LogLevelInfo,
			wantMsg:     "Some message without level",
		},
		{
			name:        "message with special characters",
			line:        "[2024-01-15 10:30:45.123] [INFO] [OUT] JSON: {\"key\": \"value\", \"count\": 42}",
			serviceName: "api",
			wantErr:     false,
			wantLevel:   service.LogLevelInfo,
			wantMsg:     "JSON: {\"key\": \"value\", \"count\": 42}",
		},
		{
			name:        "message with brackets",
			line:        "[2024-01-15 10:30:45.123] [INFO] [OUT] Array: [1, 2, 3]",
			serviceName: "api",
			wantErr:     false,
			wantLevel:   service.LogLevelInfo,
			wantMsg:     "Array: [1, 2, 3]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := parseLogLine(tt.line, tt.serviceName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseLogLine() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseLogLine() unexpected error: %v", err)
				return
			}

			if entry.Service != tt.serviceName {
				t.Errorf("Service = %q, want %q", entry.Service, tt.serviceName)
			}

			if entry.Level != tt.wantLevel {
				t.Errorf("Level = %v, want %v", entry.Level, tt.wantLevel)
			}

			if entry.IsStderr != tt.wantStderr {
				t.Errorf("IsStderr = %v, want %v", entry.IsStderr, tt.wantStderr)
			}

			if entry.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", entry.Message, tt.wantMsg)
			}
		})
	}
}

func TestParseLogLineEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		serviceName string
		wantErr     bool
		wantLevel   service.LogLevel
		wantMsg     string
	}{
		{
			name:        "missing stream marker",
			line:        "[2024-01-15 10:30:45.123] [INFO] Message without stream",
			serviceName: "api",
			wantErr:     false,
			wantLevel:   service.LogLevelInfo,
			wantMsg:     "Message without stream",
		},
		{
			name:        "unknown level defaults to info",
			line:        "[2024-01-15 10:30:45.123] [TRACE] [OUT] Trace message",
			serviceName: "api",
			wantErr:     false,
			wantLevel:   service.LogLevelInfo,
			wantMsg:     "Trace message",
		},
		{
			name:        "very short valid line",
			line:        "[2024-01-15 10:30:45.123] X",
			serviceName: "api",
			wantErr:     false,
			wantLevel:   service.LogLevelInfo,
			wantMsg:     "X",
		},
		{
			name:        "line with only timestamp and space",
			line:        "[2024-01-15 10:30:45.123] ",
			serviceName: "api",
			wantErr:     false,
			wantLevel:   service.LogLevelInfo,
			wantMsg:     "",
		},
		{
			name:        "unicode in message",
			line:        "[2024-01-15 10:30:45.123] [INFO] [OUT] 你好世界 🎉 emoji test",
			serviceName: "api",
			wantErr:     false,
			wantLevel:   service.LogLevelInfo,
			wantMsg:     "你好世界 🎉 emoji test",
		},
		{
			name:        "invalid timestamp format",
			line:        "[not-a-valid-timestamp] [INFO] [OUT] Message",
			serviceName: "api",
			wantErr:     true,
		},
		{
			name:        "missing closing bracket on timestamp",
			line:        "[2024-01-15 10:30:45.123 no closing bracket",
			serviceName: "api",
			wantErr:     true,
		},
		{
			name:        "level without closing bracket",
			line:        "[2024-01-15 10:30:45.123] [INFO no closing level bracket",
			serviceName: "api",
			wantErr:     false,
			wantLevel:   service.LogLevelInfo,
			wantMsg:     "[INFO no closing level bracket",
		},
		{
			name:        "remaining text shorter than 3 chars",
			line:        "[2024-01-15 10:30:45.123] ab",
			serviceName: "api",
			wantErr:     false,
			wantLevel:   service.LogLevelInfo,
			wantMsg:     "ab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := parseLogLine(tt.line, tt.serviceName)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if entry.Level != tt.wantLevel {
				t.Errorf("Level = %v, want %v", entry.Level, tt.wantLevel)
			}
			if entry.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", entry.Message, tt.wantMsg)
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  service.LogLevel
	}{
		{"info", service.LogLevelInfo},
		{"INFO", service.LogLevelInfo},
		{"Info", service.LogLevelInfo},
		{"warn", service.LogLevelWarn},
		{"warning", service.LogLevelWarn},
		{"WARN", service.LogLevelWarn},
		{"WARNING", service.LogLevelWarn},
		{"error", service.LogLevelError},
		{"ERROR", service.LogLevelError},
		{"debug", service.LogLevelDebug},
		{"DEBUG", service.LogLevelDebug},
		{"all", LogLevelAll},
		{"ALL", LogLevelAll},
		{"", LogLevelAll},
		{"unknown", LogLevelAll},
		{"trace", LogLevelAll},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLogLevel(tt.input)
			if got != tt.want {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseLogLevelFromString(t *testing.T) {
	tests := []struct {
		input string
		want  service.LogLevel
	}{
		{"INFO", service.LogLevelInfo},
		{"info", service.LogLevelInfo},
		{"WARN", service.LogLevelWarn},
		{"WARNING", service.LogLevelWarn},
		{"ERROR", service.LogLevelError},
		{"DEBUG", service.LogLevelDebug},
		{"", service.LogLevelInfo},
		{"unknown", service.LogLevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLogLevelFromString(tt.input)
			if got != tt.want {
				t.Errorf("parseLogLevelFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func BenchmarkParseLogLine(b *testing.B) {
	line := "[2024-01-15 10:30:45.123] [INFO] [OUT] Server started on port 3000 with configuration loaded"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseLogLine(line, "api")
	}
}
