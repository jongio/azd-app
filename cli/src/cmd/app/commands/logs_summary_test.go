package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestBuildLogSummaryFromEntries(t *testing.T) {
	start := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	end := start.Add(2 * time.Minute)
	entries := []service.LogEntry{
		{Service: "api", Level: service.LogLevelInfo, Timestamp: start},
		{Service: "api", Level: service.LogLevelError, Timestamp: end},
		{Service: "web", Level: service.LogLevelWarn, Timestamp: start.Add(time.Minute)},
		{Service: "web", Level: service.LogLevelDebug},
	}

	summary := buildLogSummaryFromEntries(entries)

	if summary.Total != 4 {
		t.Fatalf("Total = %d, want 4", summary.Total)
	}
	if summary.StartTime == nil || !summary.StartTime.Equal(start) {
		t.Fatalf("StartTime = %v, want %v", summary.StartTime, start)
	}
	if summary.EndTime == nil || !summary.EndTime.Equal(end) {
		t.Fatalf("EndTime = %v, want %v", summary.EndTime, end)
	}
	if len(summary.Services) != 2 {
		t.Fatalf("Services length = %d, want 2", len(summary.Services))
	}
	if summary.Services[0].Service != "api" {
		t.Fatalf("first service = %q, want api", summary.Services[0].Service)
	}
	if summary.Services[0].Info != 1 || summary.Services[0].Error != 1 || summary.Services[0].Total != 2 {
		t.Fatalf("api counts = %+v", summary.Services[0])
	}
	if summary.Services[1].Warn != 1 || summary.Services[1].Debug != 1 || summary.Services[1].Total != 2 {
		t.Fatalf("web counts = %+v", summary.Services[1])
	}
}

func TestBuildLogSummaryFromContextLogs(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	entries := []LogEntryWithContext{
		{Service: "api", Level: "error", Timestamp: now},
		{Service: "api", Level: "warning", Timestamp: now.Add(time.Second)},
	}

	summary := buildLogSummaryFromContextLogs(entries)

	if summary.Total != 2 {
		t.Fatalf("Total = %d, want 2", summary.Total)
	}
	if len(summary.Services) != 1 {
		t.Fatalf("Services length = %d, want 1", len(summary.Services))
	}
	if summary.Services[0].Error != 1 || summary.Services[0].Warn != 1 {
		t.Fatalf("counts = %+v", summary.Services[0])
	}
}

func TestDisplayLogSummary(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	summary := LogSummaryResult{
		Total:     2,
		StartTime: &now,
		EndTime:   &now,
		Services: []LogServiceSummary{
			{Service: "api", Info: 1, Error: 1, Total: 2},
		},
	}

	t.Run("text", func(t *testing.T) {
		var buf bytes.Buffer
		displayLogSummary(summary, &buf, false)
		out := buf.String()
		for _, want := range []string{"Total log entries: 2", "Time range:", "Service", "api"} {
			if !strings.Contains(out, want) {
				t.Fatalf("output missing %q:\n%s", want, out)
			}
		}
	})

	t.Run("json", func(t *testing.T) {
		var buf bytes.Buffer
		displayLogSummary(summary, &buf, true)

		var parsed LogSummaryResult
		if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
			t.Fatalf("failed to unmarshal summary: %v", err)
		}
		if parsed.Total != 2 || len(parsed.Services) != 1 || parsed.Services[0].Service != "api" {
			t.Fatalf("parsed summary = %+v", parsed)
		}
	})
}
