package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
)

// ==================== Mock implementations for testing ====================

// mockDashboardClient implements DashboardClient for testing.
type mockDashboardClient struct {
	pingErr        error
	services       []*serviceinfo.ServiceInfo
	getServicesErr error
	streamLogsErr  error
	logEntries     []service.LogEntry
}

func (m *mockDashboardClient) Ping(ctx context.Context) error {
	return m.pingErr
}

func (m *mockDashboardClient) GetServices(ctx context.Context) ([]*serviceinfo.ServiceInfo, error) {
	return m.services, m.getServicesErr
}

func (m *mockDashboardClient) StreamLogs(ctx context.Context, serviceName string, logs chan<- service.LogEntry) error {
	for _, entry := range m.logEntries {
		select {
		case logs <- entry:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return m.streamLogsErr
}

// mockLogManager implements LogManagerInterface for testing.
type mockLogManager struct {
	buffers map[string]*service.LogBuffer
}

func newMockLogManager() *mockLogManager {
	return &mockLogManager{
		buffers: make(map[string]*service.LogBuffer),
	}
}

func (m *mockLogManager) GetBuffer(serviceName string) (*service.LogBuffer, bool) {
	buf, exists := m.buffers[serviceName]
	return buf, exists
}

func (m *mockLogManager) GetAllBuffers() map[string]*service.LogBuffer {
	return m.buffers
}

// ==================== Executor unit tests ====================

func TestLogsExecutor_ParseServiceFilter(t *testing.T) {
	opts := &logsOptions{}
	executor := &logsExecutor{opts: opts}

	t.Run("from positional args", func(t *testing.T) {
		result := executor.parseServiceFilter([]string{"api"})
		if len(result) != 1 || result[0] != "api" {
			t.Errorf("Expected [api], got %v", result)
		}
	})

	t.Run("from service flag", func(t *testing.T) {
		opts.service = "api, web, worker"
		result := executor.parseServiceFilter([]string{})
		if len(result) != 3 {
			t.Errorf("Expected 3 services, got %d", len(result))
		}
		if result[0] != "api" || result[1] != "web" || result[2] != "worker" {
			t.Errorf("Unexpected services: %v", result)
		}
		opts.service = ""
	})

	t.Run("empty", func(t *testing.T) {
		opts.service = ""
		result := executor.parseServiceFilter([]string{})
		if len(result) != 0 {
			t.Errorf("Expected empty, got %v", result)
		}
	})
}

func TestLogsExecutor_ValidateServiceFilter(t *testing.T) {
	opts := &logsOptions{}
	executor := &logsExecutor{opts: opts}

	t.Run("valid service", func(t *testing.T) {
		err := executor.validateServiceFilter([]string{"api"}, []string{"api", "web"})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("invalid service", func(t *testing.T) {
		err := executor.validateServiceFilter([]string{"unknown"}, []string{"api", "web"})
		if err == nil {
			t.Error("Expected error for unknown service")
		}
		if !strings.Contains(err.Error(), "unknown") {
			t.Errorf("Error should mention unknown service: %v", err)
		}
	})

	t.Run("empty filter", func(t *testing.T) {
		err := executor.validateServiceFilter([]string{}, []string{"api", "web"})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func TestLogsExecutor_ParseSinceTime(t *testing.T) {
	opts := &logsOptions{}
	executor := &logsExecutor{opts: opts}

	t.Run("valid duration", func(t *testing.T) {
		opts.since = "5m"
		result, err := executor.parseSinceTime()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.IsZero() {
			t.Error("Expected non-zero time")
		}
		diff := time.Since(result)
		if diff < 4*time.Minute || diff > 6*time.Minute {
			t.Errorf("Time should be ~5m ago, got %v", diff)
		}
	})

	t.Run("empty duration", func(t *testing.T) {
		opts.since = ""
		result, err := executor.parseSinceTime()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !result.IsZero() {
			t.Error("Expected zero time for empty since")
		}
	})

	t.Run("invalid duration returns error", func(t *testing.T) {
		opts.since = "invalid"
		result, err := executor.parseSinceTime()
		if err == nil {
			t.Error("Expected error for invalid duration")
		}
		if !result.IsZero() {
			t.Error("Expected zero time for invalid duration")
		}
	})
}

func TestLogsExecutor_SetupOutputWriter(t *testing.T) {
	t.Run("default writer", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &logsOptions{}
		executor := &logsExecutor{outputWriter: &buf, opts: opts}
		writer, cleanup, err := executor.setupOutputWriter()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if cleanup != nil {
			t.Error("Expected nil cleanup for default writer")
		}
		if writer != &buf {
			t.Error("Expected default output writer")
		}
	})

	t.Run("file writer", func(t *testing.T) {
		tmpDir, _ := os.MkdirTemp("", "logs_test_*")
		defer os.RemoveAll(tmpDir)

		outputFile := filepath.Join(tmpDir, "output.log")
		opts := &logsOptions{file: outputFile}
		executor := &logsExecutor{opts: opts}

		writer, cleanup, err := executor.setupOutputWriter()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if cleanup == nil {
			t.Error("Expected non-nil cleanup for file writer")
		}
		if writer == nil {
			t.Error("Expected non-nil writer")
		}
		cleanup()
	})

	t.Run("nested directory creation", func(t *testing.T) {
		tmpDir, _ := os.MkdirTemp("", "logs_test_*")
		defer os.RemoveAll(tmpDir)

		outputFile := filepath.Join(tmpDir, "nested", "dir", "output.log")
		opts := &logsOptions{file: outputFile}
		executor := &logsExecutor{opts: opts}

		writer, cleanup, err := executor.setupOutputWriter()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if writer == nil {
			t.Error("Expected non-nil writer")
		}
		cleanup()

		if _, err := os.Stat(filepath.Dir(outputFile)); os.IsNotExist(err) {
			t.Error("Directory should have been created")
		}
	})
}

func TestLogsExecutor_CollectLogs(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "logs_test_*")
	defer os.RemoveAll(tmpDir)

	logsDir := filepath.Join(tmpDir, ".azure", "logs")
	_ = os.MkdirAll(logsDir, 0755)

	logContent := `[2024-01-15 10:30:45.100] [INFO] [OUT] Message 1
[2024-01-15 10:30:45.200] [INFO] [OUT] Message 2
`
	_ = os.WriteFile(filepath.Join(logsDir, "api.log"), []byte(logContent), 0644)

	t.Run("from log files", func(t *testing.T) {
		opts := &logsOptions{tail: 100}
		executor := &logsExecutor{opts: opts}
		mockLM := newMockLogManager()

		logs, err := executor.collectLogs(context.Background(), tmpDir, []string{"api"}, mockLM, time.Time{})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(logs) != 2 {
			t.Errorf("Expected 2 logs, got %d", len(logs))
		}
	})

	t.Run("respects since filter", func(t *testing.T) {
		opts := &logsOptions{tail: 100, since: "1h"}
		executor := &logsExecutor{opts: opts}
		mockLM := newMockLogManager()

		since := time.Date(2024, 1, 15, 10, 30, 45, 150000000, time.UTC)
		logs, err := executor.collectLogs(context.Background(), tmpDir, []string{"api"}, mockLM, since)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(logs) != 1 {
			t.Errorf("Expected 1 log after since filter, got %d", len(logs))
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		opts := &logsOptions{tail: 100}
		executor := &logsExecutor{opts: opts}
		mockLM := newMockLogManager()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := executor.collectLogs(ctx, tmpDir, []string{"api"}, mockLM, time.Time{})
		if err == nil {
			t.Error("Expected context cancellation error")
		}
	})
}

func TestLogsExecutor_BuildLogFilterInternal(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "logs_test_*")
	defer os.RemoveAll(tmpDir)

	t.Run("with exclude patterns", func(t *testing.T) {
		opts := &logsOptions{
			exclude:    "pattern1,pattern2",
			noBuiltins: true,
		}
		executor := &logsExecutor{opts: opts}

		filter, err := executor.buildLogFilterInternal(tmpDir)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if filter.PatternCount() != 2 {
			t.Errorf("Expected 2 patterns, got %d", filter.PatternCount())
		}
	})

	t.Run("with builtins", func(t *testing.T) {
		opts := &logsOptions{
			noBuiltins: false,
		}
		executor := &logsExecutor{opts: opts}

		filter, err := executor.buildLogFilterInternal(tmpDir)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if filter.PatternCount() == 0 {
			t.Error("Expected built-in patterns")
		}
	})
}

// ==================== Execute tests ====================

func TestLogsExecutor_Execute(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "logs_test_*")
	defer os.RemoveAll(tmpDir)

	logsDir := filepath.Join(tmpDir, ".azure", "logs")
	_ = os.MkdirAll(logsDir, 0755)

	logContent := `[2024-01-15 10:30:45.100] [INFO] [OUT] Test message
`
	_ = os.WriteFile(filepath.Join(logsDir, "api.log"), []byte(logContent), 0644)

	t.Run("dashboard not running shows info message", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &logsOptions{tail: 100, level: "all", format: "text"}
		executor := newLogsExecutorForTest(
			func(ctx context.Context, projectDir string) (DashboardClient, error) {
				return nil, errors.New("dashboard not running")
			},
			func(projectDir string) LogManagerInterface {
				return newMockLogManager()
			},
			func() (string, error) { return tmpDir, nil },
			&buf,
			opts,
		)

		err := executor.execute(context.Background(), []string{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("dashboard ping fails shows info message", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &logsOptions{tail: 100, level: "all", format: "text"}
		executor := newLogsExecutorForTest(
			func(ctx context.Context, projectDir string) (DashboardClient, error) {
				return &mockDashboardClient{pingErr: errors.New("ping failed")}, nil
			},
			func(projectDir string) LogManagerInterface {
				return newMockLogManager()
			},
			func() (string, error) { return tmpDir, nil },
			&buf,
			opts,
		)

		err := executor.execute(context.Background(), []string{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("no services shows info message", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &logsOptions{tail: 100, level: "all", format: "text"}
		executor := newLogsExecutorForTest(
			func(ctx context.Context, projectDir string) (DashboardClient, error) {
				return &mockDashboardClient{services: []*serviceinfo.ServiceInfo{}}, nil
			},
			func(projectDir string) LogManagerInterface {
				return newMockLogManager()
			},
			func() (string, error) { return tmpDir, nil },
			&buf,
			opts,
		)

		err := executor.execute(context.Background(), []string{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("displays logs from file", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &logsOptions{tail: 100, level: "all", format: "text", timestamps: true, noColor: true}
		executor := newLogsExecutorForTest(
			func(ctx context.Context, projectDir string) (DashboardClient, error) {
				return &mockDashboardClient{
					services: []*serviceinfo.ServiceInfo{{Name: "api"}},
				}, nil
			},
			func(projectDir string) LogManagerInterface {
				return newMockLogManager()
			},
			func() (string, error) { return tmpDir, nil },
			&buf,
			opts,
		)

		err := executor.execute(context.Background(), []string{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "Test message") {
			t.Errorf("Output should contain log message, got: %s", buf.String())
		}
	})

	t.Run("JSON format output", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &logsOptions{tail: 100, level: "all", format: "json"}
		executor := newLogsExecutorForTest(
			func(ctx context.Context, projectDir string) (DashboardClient, error) {
				return &mockDashboardClient{
					services: []*serviceinfo.ServiceInfo{{Name: "api"}},
				}, nil
			},
			func(projectDir string) LogManagerInterface {
				return newMockLogManager()
			},
			func() (string, error) { return tmpDir, nil },
			&buf,
			opts,
		)

		err := executor.execute(context.Background(), []string{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		var entry map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Errorf("Output should be valid JSON: %v", err)
		}
	})

	t.Run("service filter validation error", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &logsOptions{tail: 100, level: "all", format: "text"}
		executor := newLogsExecutorForTest(
			func(ctx context.Context, projectDir string) (DashboardClient, error) {
				return &mockDashboardClient{
					services: []*serviceinfo.ServiceInfo{{Name: "api"}},
				}, nil
			},
			func(projectDir string) LogManagerInterface {
				return newMockLogManager()
			},
			func() (string, error) { return tmpDir, nil },
			&buf,
			opts,
		)

		err := executor.execute(context.Background(), []string{"nonexistent"})
		if err == nil {
			t.Error("Expected error for non-existent service")
		}
	})

	t.Run("get services error", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &logsOptions{tail: 100, level: "all", format: "text"}
		executor := newLogsExecutorForTest(
			func(ctx context.Context, projectDir string) (DashboardClient, error) {
				return &mockDashboardClient{
					getServicesErr: errors.New("failed to get services"),
				}, nil
			},
			func(projectDir string) LogManagerInterface {
				return newMockLogManager()
			},
			func() (string, error) { return tmpDir, nil },
			&buf,
			opts,
		)

		err := executor.execute(context.Background(), []string{})
		if err == nil {
			t.Error("Expected error when GetServices fails")
		}
	})

	t.Run("getwd error", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &logsOptions{}
		executor := newLogsExecutorForTest(
			func(ctx context.Context, projectDir string) (DashboardClient, error) {
				return &mockDashboardClient{}, nil
			},
			func(projectDir string) LogManagerInterface {
				return newMockLogManager()
			},
			func() (string, error) { return "", errors.New("getwd failed") },
			&buf,
			opts,
		)

		err := executor.execute(context.Background(), []string{})
		if err == nil {
			t.Error("Expected error when getwd fails")
		}
	})

	t.Run("level filtering", func(t *testing.T) {
		levelContent := `[2024-01-15 10:30:45.100] [INFO] [OUT] Info message
[2024-01-15 10:30:45.200] [ERROR] [ERR] Error message
[2024-01-15 10:30:45.300] [WARN] [OUT] Warn message
`
		_ = os.WriteFile(filepath.Join(logsDir, "api.log"), []byte(levelContent), 0644)

		var buf bytes.Buffer
		opts := &logsOptions{tail: 100, level: "error", format: "text", timestamps: true, noColor: true}
		executor := newLogsExecutorForTest(
			func(ctx context.Context, projectDir string) (DashboardClient, error) {
				return &mockDashboardClient{
					services: []*serviceinfo.ServiceInfo{{Name: "api"}},
				}, nil
			},
			func(projectDir string) LogManagerInterface {
				return newMockLogManager()
			},
			func() (string, error) { return tmpDir, nil },
			&buf,
			opts,
		)

		err := executor.execute(context.Background(), []string{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "Error message") {
			t.Error("Should contain error message")
		}
		if strings.Contains(output, "Info message") {
			t.Error("Should NOT contain info message when filtering by error")
		}
	})

	t.Run("tail limit", func(t *testing.T) {
		var manyLogs strings.Builder
		for i := 0; i < 50; i++ {
			manyLogs.WriteString(fmt.Sprintf("[2024-01-15 10:30:%02d.000] [INFO] [OUT] Message %d\n", i, i))
		}
		_ = os.WriteFile(filepath.Join(logsDir, "api.log"), []byte(manyLogs.String()), 0644)

		var buf bytes.Buffer
		opts := &logsOptions{tail: 5, level: "all", format: "text", noColor: true}
		executor := newLogsExecutorForTest(
			func(ctx context.Context, projectDir string) (DashboardClient, error) {
				return &mockDashboardClient{
					services: []*serviceinfo.ServiceInfo{{Name: "api"}},
				}, nil
			},
			func(projectDir string) LogManagerInterface {
				return newMockLogManager()
			},
			func() (string, error) { return tmpDir, nil },
			&buf,
			opts,
		)

		err := executor.execute(context.Background(), []string{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		if len(lines) != 5 {
			t.Errorf("Expected 5 lines with tail=5, got %d", len(lines))
		}
	})
}
