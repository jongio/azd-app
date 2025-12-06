package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func TestFilterLogsByLevel(t *testing.T) {
	now := time.Now()
	logs := []service.LogEntry{
		{Service: "api", Level: service.LogLevelInfo, Message: "info msg", Timestamp: now},
		{Service: "api", Level: service.LogLevelWarn, Message: "warn msg", Timestamp: now},
		{Service: "api", Level: service.LogLevelError, Message: "error msg", Timestamp: now},
		{Service: "api", Level: service.LogLevelDebug, Message: "debug msg", Timestamp: now},
		{Service: "web", Level: service.LogLevelInfo, Message: "web info", Timestamp: now},
	}

	tests := []struct {
		name      string
		level     service.LogLevel
		wantCount int
	}{
		{"filter info", service.LogLevelInfo, 2},
		{"filter warn", service.LogLevelWarn, 1},
		{"filter error", service.LogLevelError, 1},
		{"filter debug", service.LogLevelDebug, 1},
		{"filter all (LogLevelAll)", LogLevelAll, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterLogsByLevel(logs, tt.level)
			if len(filtered) != tt.wantCount {
				t.Errorf("filterLogsByLevel() returned %d entries, want %d", len(filtered), tt.wantCount)
			}
		})
	}
}

func TestFilterLogsByLevelEdgeCases(t *testing.T) {
	now := time.Now()

	t.Run("empty slice", func(t *testing.T) {
		filtered := filterLogsByLevel([]service.LogEntry{}, service.LogLevelInfo)
		if len(filtered) != 0 {
			t.Errorf("Expected 0 entries, got %d", len(filtered))
		}
	})

	t.Run("all entries same level", func(t *testing.T) {
		logs := make([]service.LogEntry, 100)
		for i := range logs {
			logs[i] = service.LogEntry{Level: service.LogLevelInfo, Timestamp: now}
		}
		filtered := filterLogsByLevel(logs, service.LogLevelInfo)
		if len(filtered) != 100 {
			t.Errorf("Expected 100 entries, got %d", len(filtered))
		}
	})

	t.Run("no matching level", func(t *testing.T) {
		logs := []service.LogEntry{
			{Level: service.LogLevelInfo, Timestamp: now},
			{Level: service.LogLevelWarn, Timestamp: now},
		}
		filtered := filterLogsByLevel(logs, service.LogLevelDebug)
		if len(filtered) != 0 {
			t.Errorf("Expected 0 entries, got %d", len(filtered))
		}
	})
}

func TestBuildLogFilter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logs_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("no azure.yaml uses builtins", func(t *testing.T) {
		filter, err := buildLogFilter(tmpDir, "", false)
		if err != nil {
			t.Fatalf("buildLogFilter() error: %v", err)
		}
		if filter == nil {
			t.Fatal("Expected non-nil filter")
		}
		if filter.PatternCount() == 0 {
			t.Error("Expected built-in patterns to be loaded")
		}
	})

	t.Run("custom exclude patterns", func(t *testing.T) {
		filter, err := buildLogFilter(tmpDir, "pattern1,pattern2", true)
		if err != nil {
			t.Fatalf("buildLogFilter() error: %v", err)
		}
		if filter == nil {
			t.Fatal("Expected non-nil filter")
		}
		if filter.PatternCount() != 2 {
			t.Errorf("Expected 2 custom patterns, got %d", filter.PatternCount())
		}
	})

	t.Run("both flags set", func(t *testing.T) {
		filter, err := buildLogFilter(tmpDir, "pattern1, pattern2 , pattern3", true)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if filter.PatternCount() != 3 {
			t.Errorf("Expected 3 patterns, got %d", filter.PatternCount())
		}
	})

	t.Run("empty exclude with builtins", func(t *testing.T) {
		filter, err := buildLogFilter(tmpDir, "", false)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if filter.PatternCount() == 0 {
			t.Error("Expected built-in patterns")
		}
	})
}

func TestBuildLogFilterWithAzureYaml(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logs_test_yaml_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("azure.yaml with exclude patterns", func(t *testing.T) {
		azureYaml := `name: test-project
logs:
  filters:
    exclude:
      - yaml_pattern1
      - yaml_pattern2
`
		if err := os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte(azureYaml), 0644); err != nil {
			t.Fatal(err)
		}

		filter, err := buildLogFilter(tmpDir, "", true)
		if err != nil {
			t.Fatalf("buildLogFilter() error: %v", err)
		}
		if filter == nil {
			t.Fatal("Expected non-nil filter")
		}
		if filter.PatternCount() != 2 {
			t.Errorf("Expected 2 patterns from azure.yaml, got %d", filter.PatternCount())
		}
	})

	t.Run("azure.yaml patterns combined with CLI patterns", func(t *testing.T) {
		azureYaml := `name: test-project
logs:
  filters:
    exclude:
      - yaml_pattern
`
		if err := os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte(azureYaml), 0644); err != nil {
			t.Fatal(err)
		}

		filter, err := buildLogFilter(tmpDir, "cli_pattern1,cli_pattern2", true)
		if err != nil {
			t.Fatalf("buildLogFilter() error: %v", err)
		}
		if filter.PatternCount() != 3 {
			t.Errorf("Expected 3 combined patterns, got %d", filter.PatternCount())
		}
	})

	t.Run("azure.yaml with includeBuiltins false", func(t *testing.T) {
		azureYaml := `name: test-project
logs:
  filters:
    includeBuiltins: false
    exclude:
      - custom_only
`
		if err := os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte(azureYaml), 0644); err != nil {
			t.Fatal(err)
		}

		filter, err := buildLogFilter(tmpDir, "", false)
		if err != nil {
			t.Fatalf("buildLogFilter() error: %v", err)
		}
		if filter.PatternCount() != 1 {
			t.Errorf("Expected 1 pattern (builtins disabled via includeBuiltins: false), got %d", filter.PatternCount())
		}
	})

	t.Run("CLI --no-builtins overrides azure.yaml includeBuiltins", func(t *testing.T) {
		azureYaml := `name: test-project
logs:
  filters:
    includeBuiltins: true
    exclude:
      - custom_pattern
`
		if err := os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte(azureYaml), 0644); err != nil {
			t.Fatal(err)
		}

		filter, err := buildLogFilter(tmpDir, "", true)
		if err != nil {
			t.Fatalf("buildLogFilter() error: %v", err)
		}
		if filter.PatternCount() != 1 {
			t.Errorf("Expected 1 pattern (CLI --no-builtins overrides), got %d", filter.PatternCount())
		}
	})
}

func TestGetFilterConfig(t *testing.T) {
	t.Run("nil azure yaml", func(t *testing.T) {
		result := getFilterConfig(nil, nil)
		if result != nil {
			t.Error("Expected nil for nil azureYaml")
		}
	})

	t.Run("with error", func(t *testing.T) {
		yaml := &service.AzureYaml{}
		result := getFilterConfig(yaml, fmt.Errorf("some error"))
		if result != nil {
			t.Error("Expected nil when error is provided")
		}
	})

	t.Run("empty logs config", func(t *testing.T) {
		yaml := &service.AzureYaml{}
		result := getFilterConfig(yaml, nil)
		if result != nil {
			t.Error("Expected nil for empty logs config")
		}
	})
}

func BenchmarkFilterLogsByLevel(b *testing.B) {
	now := time.Now()
	logs := make([]service.LogEntry, 1000)
	for i := range logs {
		logs[i] = service.LogEntry{
			Service:   "api",
			Level:     service.LogLevel(i % 4),
			Message:   "Test message",
			Timestamp: now,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filterLogsByLevel(logs, service.LogLevelInfo)
	}
}
