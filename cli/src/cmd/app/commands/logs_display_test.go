package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestDisplayLogsText(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 45, 123000000, time.UTC)
	logs := []service.LogEntry{
		{Service: "api", Level: service.LogLevelInfo, Message: "Server started", Timestamp: now},
		{Service: "api", Level: service.LogLevelError, Message: "Error occurred", Timestamp: now, IsStderr: true},
		{Service: "api", Level: service.LogLevelWarn, Message: "Warning message", Timestamp: now},
		{Service: "api", Level: service.LogLevelDebug, Message: "Debug info", Timestamp: now},
	}

	t.Run("with timestamps and colors", func(t *testing.T) {
		var buf bytes.Buffer
		displayLogsText(logs, &buf, true, false)
		output := buf.String()

		if !strings.Contains(output, "10:30:45.123") {
			t.Error("Output should contain timestamp")
		}
		if !strings.Contains(output, "[api]") {
			t.Error("Output should contain service name")
		}
		if !strings.Contains(output, "\033[") {
			t.Error("Output should contain ANSI color codes")
		}
	})

	t.Run("without timestamps", func(t *testing.T) {
		var buf bytes.Buffer
		displayLogsText(logs[:1], &buf, false, true)
		output := buf.String()

		if strings.Contains(output, "10:30:45") {
			t.Error("Output should not contain timestamp when disabled")
		}
		if strings.Contains(output, "\033[") {
			t.Error("Output should not contain ANSI codes in no-color mode")
		}
	})

	t.Run("no-color mode", func(t *testing.T) {
		var buf bytes.Buffer
		displayLogsText(logs, &buf, true, true)
		output := buf.String()

		if strings.Contains(output, "\033[") {
			t.Error("Output should not contain ANSI codes in no-color mode")
		}
	})

	t.Run("empty logs slice", func(t *testing.T) {
		var buf bytes.Buffer
		displayLogsText([]service.LogEntry{}, &buf, true, false)
		if buf.Len() != 0 {
			t.Errorf("Expected empty output, got %d bytes", buf.Len())
		}
	})

	t.Run("all log levels with colors", func(t *testing.T) {
		allLogs := []service.LogEntry{
			{Service: "api", Level: service.LogLevelInfo, Message: "Info", Timestamp: now},
			{Service: "api", Level: service.LogLevelWarn, Message: "Warn", Timestamp: now},
			{Service: "api", Level: service.LogLevelError, Message: "Error", Timestamp: now},
			{Service: "api", Level: service.LogLevelDebug, Message: "Debug", Timestamp: now},
			{Service: "api", Level: service.LogLevelInfo, Message: "Stderr", Timestamp: now, IsStderr: true},
		}

		var buf bytes.Buffer
		displayLogsText(allLogs, &buf, true, false)
		output := buf.String()

		for _, log := range allLogs {
			if !strings.Contains(output, log.Message) {
				t.Errorf("Output should contain %q", log.Message)
			}
		}
	})
}

func TestDisplayLogsJSON(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	logs := []service.LogEntry{
		{Service: "api", Level: service.LogLevelInfo, Message: "Test message", Timestamp: now},
	}

	t.Run("basic json output", func(t *testing.T) {
		var buf bytes.Buffer
		displayLogsJSON(logs, &buf)
		output := buf.String()

		if !strings.Contains(output, `"service":"api"`) {
			t.Error("JSON output should contain service field")
		}
		if !strings.Contains(output, `"message":"Test message"`) {
			t.Error("JSON output should contain message field")
		}
		if !strings.Contains(output, `"level"`) {
			t.Error("JSON output should contain level field")
		}
	})

	t.Run("empty logs slice", func(t *testing.T) {
		var buf bytes.Buffer
		displayLogsJSON([]service.LogEntry{}, &buf)
		if buf.Len() != 0 {
			t.Errorf("Expected empty output, got %d bytes", buf.Len())
		}
	})

	t.Run("multiple entries", func(t *testing.T) {
		multiLogs := []service.LogEntry{
			{Service: "api", Level: service.LogLevelInfo, Message: "First", Timestamp: now},
			{Service: "web", Level: service.LogLevelError, Message: "Second", Timestamp: now},
		}

		var buf bytes.Buffer
		displayLogsJSON(multiLogs, &buf)
		output := buf.String()

		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) != 2 {
			t.Errorf("Expected 2 JSON lines, got %d", len(lines))
		}

		for i, line := range lines {
			var entry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				t.Errorf("Line %d is not valid JSON: %v", i, err)
			}
		}
	})

	t.Run("special characters in message", func(t *testing.T) {
		specialLogs := []service.LogEntry{
			{Service: "api", Level: service.LogLevelInfo, Message: `"quoted" and \backslash\ and 日本語`, Timestamp: now},
		}

		var buf bytes.Buffer
		displayLogsJSON(specialLogs, &buf)

		var entry map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Errorf("Output is not valid JSON: %v", err)
		}
	})
}

func TestColorConstants(t *testing.T) {
	if colorCyan != "\033[36m" {
		t.Errorf("colorCyan = %q, expected %q", colorCyan, "\033[36m")
	}
}

func BenchmarkDisplayLogsText(b *testing.B) {
	now := time.Now()
	logs := make([]service.LogEntry, 100)
	for i := range logs {
		logs[i] = service.LogEntry{
			Service:   "api",
			Level:     service.LogLevelInfo,
			Message:   "Test log message with some content",
			Timestamp: now,
		}
	}

	var buf bytes.Buffer
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		displayLogsText(logs, &buf, true, false)
	}
}
