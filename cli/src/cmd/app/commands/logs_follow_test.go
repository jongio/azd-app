package commands

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// newTestExecutor creates a logsExecutor for testing with the given options.
func newTestExecutor(buf *bytes.Buffer, sigChan chan os.Signal, opts *logsOptions) *logsExecutor {
	if opts == nil {
		opts = &logsOptions{format: "text"}
	}
	return &logsExecutor{
		outputWriter: buf,
		signalChan:   sigChan,
		opts:         opts,
	}
}

func TestLogsExecutor_FollowLogsViaDashboard(t *testing.T) {
	t.Run("ping error", func(t *testing.T) {
		var buf bytes.Buffer
		executor := newTestExecutor(&buf, make(chan os.Signal, 1), &logsOptions{format: "text"})

		mockClient := &mockDashboardClient{pingErr: context.DeadlineExceeded}

		err := executor.followLogsViaDashboard(context.Background(), mockClient, nil, LogLevelAll, nil, &buf)
		if err == nil {
			t.Error("Expected error when ping fails")
		}
	})

	t.Run("streams logs until context cancel", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{
			format:     "text",
			timestamps: true,
			noColor:    true,
		})

		now := time.Now()
		mockClient := &mockDashboardClient{
			logEntries: []service.LogEntry{
				{Service: "api", Level: service.LogLevelInfo, Message: "Log 1", Timestamp: now},
				{Service: "api", Level: service.LogLevelInfo, Message: "Log 2", Timestamp: now},
			},
		}

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error)
		go func() {
			done <- executor.followLogsViaDashboard(ctx, mockClient, nil, LogLevelAll, nil, &buf)
		}()

		time.Sleep(100 * time.Millisecond)
		cancel()

		err := <-done
		if err != nil && err != context.Canceled {
			t.Errorf("Unexpected error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "Log") {
			t.Errorf("Should contain streamed logs, got: %s", output)
		}
	})

	t.Run("filters by level", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{
			format:     "text",
			timestamps: true,
			noColor:    true,
		})

		now := time.Now()
		mockClient := &mockDashboardClient{
			logEntries: []service.LogEntry{
				{Service: "api", Level: service.LogLevelInfo, Message: "Info log", Timestamp: now},
				{Service: "api", Level: service.LogLevelError, Message: "Error log", Timestamp: now},
			},
		}

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error)
		go func() {
			done <- executor.followLogsViaDashboard(ctx, mockClient, nil, service.LogLevelError, nil, &buf)
		}()

		time.Sleep(100 * time.Millisecond)
		cancel()

		<-done

		output := buf.String()
		if strings.Contains(output, "Info log") {
			t.Error("Should NOT contain info log when filtering by error")
		}
	})

	t.Run("filters by service", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{
			format:     "text",
			timestamps: true,
			noColor:    true,
		})

		now := time.Now()
		mockClient := &mockDashboardClient{
			logEntries: []service.LogEntry{
				{Service: "api", Level: service.LogLevelInfo, Message: "API log", Timestamp: now},
				{Service: "web", Level: service.LogLevelInfo, Message: "Web log", Timestamp: now},
			},
		}

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error)
		go func() {
			done <- executor.followLogsViaDashboard(ctx, mockClient, []string{"api", "worker"}, LogLevelAll, nil, &buf)
		}()

		time.Sleep(100 * time.Millisecond)
		cancel()

		<-done

		output := buf.String()
		if strings.Contains(output, "Web log") {
			t.Error("Should NOT contain Web log when filtering by [api, worker]")
		}
	})

	t.Run("signal interrupts streaming", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{format: "text"})

		mockClient := &mockDashboardClient{}

		ctx := context.Background()

		done := make(chan error)
		go func() {
			done <- executor.followLogsViaDashboard(ctx, mockClient, nil, LogLevelAll, nil, &buf)
		}()

		time.Sleep(10 * time.Millisecond)
		sigChan <- os.Interrupt

		err := <-done
		if err != nil {
			t.Errorf("Expected nil error on signal, got: %v", err)
		}
	})
}

func TestLogsExecutor_FollowLogsInMemory(t *testing.T) {
	t.Run("signal interrupts streaming", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{format: "text"})

		subscriptions := make(map[string]chan service.LogEntry)
		mockLM := newMockLogManager()

		done := make(chan error)
		go func() {
			done <- executor.followLogsInMemory(subscriptions, mockLM, LogLevelAll, nil, &buf)
		}()

		time.Sleep(10 * time.Millisecond)
		sigChan <- os.Interrupt

		err := <-done
		if err != nil {
			t.Errorf("Expected nil error on signal, got: %v", err)
		}
	})

	t.Run("processes logs from subscription", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{
			format:     "text",
			timestamps: true,
			noColor:    true,
		})

		logChan := make(chan service.LogEntry, 10)
		subscriptions := map[string]chan service.LogEntry{
			"api": logChan,
		}

		mockLM := newMockLogManager()
		buf1, _ := service.NewLogBuffer("api", 100, false, "")
		mockLM.buffers["api"] = buf1

		done := make(chan error)
		go func() {
			done <- executor.followLogsInMemory(subscriptions, mockLM, LogLevelAll, nil, &buf)
		}()

		now := time.Now()
		logChan <- service.LogEntry{Service: "api", Level: service.LogLevelInfo, Message: "Test message", Timestamp: now}

		time.Sleep(50 * time.Millisecond)
		sigChan <- os.Interrupt

		<-done

		output := buf.String()
		if !strings.Contains(output, "Test message") {
			t.Errorf("Should contain log message, got: %s", output)
		}
	})

	t.Run("filters by level", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{
			format:  "text",
			noColor: true,
		})

		logChan := make(chan service.LogEntry, 10)
		subscriptions := map[string]chan service.LogEntry{
			"api": logChan,
		}
		mockLM := newMockLogManager()
		buf2, _ := service.NewLogBuffer("api", 100, false, "")
		mockLM.buffers["api"] = buf2

		done := make(chan error)
		go func() {
			done <- executor.followLogsInMemory(subscriptions, mockLM, service.LogLevelError, nil, &buf)
		}()

		now := time.Now()
		logChan <- service.LogEntry{Service: "api", Level: service.LogLevelInfo, Message: "Info msg", Timestamp: now}
		logChan <- service.LogEntry{Service: "api", Level: service.LogLevelError, Message: "Error msg", Timestamp: now}

		time.Sleep(50 * time.Millisecond)
		sigChan <- os.Interrupt

		<-done

		output := buf.String()
		if strings.Contains(output, "Info msg") {
			t.Error("Should NOT contain info message when filtering by error")
		}
	})

	t.Run("closes when all subscriptions close", func(t *testing.T) {
		var buf bytes.Buffer
		executor := newTestExecutor(&buf, make(chan os.Signal, 1), &logsOptions{format: "text"})

		logChan := make(chan service.LogEntry, 10)
		subscriptions := map[string]chan service.LogEntry{
			"api": logChan,
		}
		mockLM := newMockLogManager()
		buf3, _ := service.NewLogBuffer("api", 100, false, "")
		mockLM.buffers["api"] = buf3

		done := make(chan error)
		go func() {
			done <- executor.followLogsInMemory(subscriptions, mockLM, LogLevelAll, nil, &buf)
		}()

		time.Sleep(10 * time.Millisecond)
		close(logChan)

		err := <-done
		if err != nil {
			t.Errorf("Expected nil error when channels close, got: %v", err)
		}
	})

	t.Run("JSON format output", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{format: "json"})

		logChan := make(chan service.LogEntry, 10)
		subscriptions := map[string]chan service.LogEntry{
			"api": logChan,
		}
		mockLM := newMockLogManager()
		buf4, _ := service.NewLogBuffer("api", 100, false, "")
		mockLM.buffers["api"] = buf4

		done := make(chan error)
		go func() {
			done <- executor.followLogsInMemory(subscriptions, mockLM, LogLevelAll, nil, &buf)
		}()

		now := time.Now()
		logChan <- service.LogEntry{Service: "api", Level: service.LogLevelInfo, Message: "JSON test", Timestamp: now}

		time.Sleep(50 * time.Millisecond)
		sigChan <- os.Interrupt

		<-done

		output := buf.String()
		if !strings.Contains(output, "JSON test") {
			t.Errorf("Should contain log message, got: %s", output)
		}
	})
}

func TestLogsExecutor_FollowLogs(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "logs_test_*")
	defer os.RemoveAll(tmpDir)

	t.Run("dashboard streaming completes without error", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{format: "text"})

		mockClient := &mockDashboardClient{}
		done := make(chan error)
		go func() {
			done <- executor.followLogsViaDashboard(context.Background(), mockClient, nil, LogLevelAll, nil, &buf)
		}()

		time.Sleep(10 * time.Millisecond)
		sigChan <- os.Interrupt
		err := <-done

		if err != nil {
			t.Errorf("Expected nil error, got: %v", err)
		}
	})

	t.Run("in-memory streaming completes without error", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{format: "text"})

		logChan := make(chan service.LogEntry, 10)
		subscriptions := map[string]chan service.LogEntry{
			"api": logChan,
		}
		mockLM := newMockLogManager()
		buf6, _ := service.NewLogBuffer("api", 100, false, "")
		mockLM.buffers["api"] = buf6

		done := make(chan error)
		go func() {
			done <- executor.followLogsInMemory(subscriptions, mockLM, LogLevelAll, nil, &buf)
		}()

		time.Sleep(10 * time.Millisecond)
		sigChan <- os.Interrupt
		err := <-done

		if err != nil {
			t.Errorf("Expected nil error, got: %v", err)
		}
	})
}

func TestLogsExecutor_FollowLogsOrchestration(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "logs_test_*")
	defer os.RemoveAll(tmpDir)

	t.Run("uses in-memory when buffers available", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{
			format:     "text",
			timestamps: true,
			noColor:    true,
		})

		mockLM := newMockLogManager()
		logBuf, _ := service.NewLogBuffer("api", 100, false, "")
		mockLM.buffers["api"] = logBuf

		mockClient := &mockDashboardClient{
			pingErr: context.DeadlineExceeded,
		}

		done := make(chan error)
		go func() {
			done <- executor.followLogs(context.Background(), tmpDir, mockLM, mockClient, nil, LogLevelAll, nil, &buf)
		}()

		time.Sleep(10 * time.Millisecond)
		sigChan <- os.Interrupt
		err := <-done

		if err != nil {
			t.Errorf("Expected nil error, got: %v", err)
		}
	})

	t.Run("uses in-memory for specific service filter", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{format: "text"})

		mockLM := newMockLogManager()
		logBuf, _ := service.NewLogBuffer("api", 100, false, "")
		mockLM.buffers["api"] = logBuf

		mockClient := &mockDashboardClient{}

		done := make(chan error)
		go func() {
			done <- executor.followLogs(context.Background(), tmpDir, mockLM, mockClient, []string{"api"}, LogLevelAll, nil, &buf)
		}()

		time.Sleep(10 * time.Millisecond)
		sigChan <- os.Interrupt
		err := <-done

		if err != nil {
			t.Errorf("Expected nil error, got: %v", err)
		}
	})

	t.Run("falls back to dashboard when no buffers", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{format: "text"})

		mockLM := newMockLogManager()
		mockClient := &mockDashboardClient{}

		done := make(chan error)
		go func() {
			done <- executor.followLogs(context.Background(), tmpDir, mockLM, mockClient, nil, LogLevelAll, nil, &buf)
		}()

		time.Sleep(10 * time.Millisecond)
		sigChan <- os.Interrupt
		err := <-done

		if err != nil {
			t.Errorf("Expected nil error, got: %v", err)
		}
	})

	t.Run("falls back to dashboard for non-existent service", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{format: "text"})

		mockLM := newMockLogManager()
		logBuf, _ := service.NewLogBuffer("other", 100, false, "")
		mockLM.buffers["other"] = logBuf

		mockClient := &mockDashboardClient{}

		done := make(chan error)
		go func() {
			done <- executor.followLogs(context.Background(), tmpDir, mockLM, mockClient, []string{"nonexistent"}, LogLevelAll, nil, &buf)
		}()

		time.Sleep(10 * time.Millisecond)
		sigChan <- os.Interrupt
		err := <-done

		if err != nil {
			t.Errorf("Expected nil error, got: %v", err)
		}
	})
}
