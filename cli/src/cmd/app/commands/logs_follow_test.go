package commands

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// safeBuf wraps bytes.Buffer with a mutex for concurrent read/write safety.
type safeBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// newTestExecutor creates a logsExecutor for testing with the given options.
func newTestExecutor(w io.Writer, sigChan chan os.Signal, opts *logsOptions) *logsExecutor {
	if opts == nil {
		opts = &logsOptions{output: "text"}
	}
	return &logsExecutor{
		outputWriter: w,
		signalChan:   sigChan,
		opts:         opts,
	}
}

// waitForOutput polls the buffer until it contains the expected string or times out.
func waitForOutput(buf *safeBuf, contains string, timeout time.Duration) bool {
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			return false
		default:
			if strings.Contains(buf.String(), contains) {
				return true
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func TestLogsExecutor_FollowLogsViaDashboard(t *testing.T) {
	t.Run("ping error", func(t *testing.T) {
		var buf bytes.Buffer
		executor := newTestExecutor(&buf, make(chan os.Signal, 1), &logsOptions{output: "text"})

		mockClient := &mockDashboardClient{pingErr: context.DeadlineExceeded}

		err := executor.followLogsViaDashboard(context.Background(), mockClient, nil, LogLevelAll, nil, &buf)
		if err == nil {
			t.Error("Expected error when ping fails")
		}
	})

	t.Run("streams logs until context cancel", func(t *testing.T) {
		var buf safeBuf
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{
			output: "text",
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

		// Poll for output instead of fixed sleep
		if !waitForOutput(&buf, "Log", 2*time.Second) {
			t.Fatal("timed out waiting for log output")
		}
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
		var buf safeBuf
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{
			output: "text",
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

		// Wait for the error log to appear (it passes the filter)
		if !waitForOutput(&buf, "Error log", 2*time.Second) {
			t.Log("Warning: no filtered output appeared within timeout")
		}
		cancel()

		<-done

		output := buf.String()
		if strings.Contains(output, "Info log") {
			t.Error("Should NOT contain info log when filtering by error")
		}
	})

	t.Run("filters by service", func(t *testing.T) {
		var buf safeBuf
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{
			output: "text",
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

		// Wait for the API log to appear (it passes the service filter)
		if !waitForOutput(&buf, "API log", 2*time.Second) {
			t.Log("Warning: no filtered output appeared within timeout")
		}
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
		executor := newTestExecutor(&buf, sigChan, &logsOptions{output: "text"})

		mockClient := &mockDashboardClient{}

		ctx := context.Background()

		done := make(chan error, 1)
		go func() {
			done <- executor.followLogsViaDashboard(ctx, mockClient, nil, LogLevelAll, nil, &buf)
		}()

		// Buffered channel - signal will be picked up when goroutine reaches select
		sigChan <- os.Interrupt

		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Expected nil error on signal, got: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for signal to interrupt streaming")
		}
	})
}

func TestLogsExecutor_FollowLogsInMemory(t *testing.T) {
	t.Run("signal interrupts streaming", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{output: "text"})

		subscriptions := make(map[string]chan service.LogEntry)
		mockLM := newMockLogManager()

		done := make(chan error, 1)
		go func() {
			done <- executor.followLogsInMemory(subscriptions, mockLM, LogLevelAll, nil, &buf)
		}()

		sigChan <- os.Interrupt

		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Expected nil error on signal, got: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for signal to interrupt streaming")
		}
	})

	t.Run("processes logs from subscription", func(t *testing.T) {
		var buf safeBuf
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{
			output: "text",
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

		// Poll for output instead of fixed sleep
		if !waitForOutput(&buf, "Test message", 2*time.Second) {
			t.Fatal("timed out waiting for log output")
		}
		sigChan <- os.Interrupt

		<-done

		output := buf.String()
		if !strings.Contains(output, "Test message") {
			t.Errorf("Should contain log message, got: %s", output)
		}
	})

	t.Run("filters by level", func(t *testing.T) {
		var buf safeBuf
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{
			output: "text",
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

		// Poll for the error message to appear (it passes the filter)
		if !waitForOutput(&buf, "Error msg", 2*time.Second) {
			t.Log("Warning: no filtered output appeared within timeout")
		}
		sigChan <- os.Interrupt

		<-done

		output := buf.String()
		if strings.Contains(output, "Info msg") {
			t.Error("Should NOT contain info message when filtering by error")
		}
	})

	t.Run("closes when all subscriptions close", func(t *testing.T) {
		var buf bytes.Buffer
		executor := newTestExecutor(&buf, make(chan os.Signal, 1), &logsOptions{output: "text"})

		logChan := make(chan service.LogEntry, 10)
		subscriptions := map[string]chan service.LogEntry{
			"api": logChan,
		}
		mockLM := newMockLogManager()
		buf3, _ := service.NewLogBuffer("api", 100, false, "")
		mockLM.buffers["api"] = buf3

		done := make(chan error, 1)
		go func() {
			done <- executor.followLogsInMemory(subscriptions, mockLM, LogLevelAll, nil, &buf)
		}()

		// Close channel - buffered done chan ensures we don't block
		close(logChan)

		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Expected nil error when channels close, got: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for close to propagate")
		}
	})

	t.Run("JSON format output", func(t *testing.T) {
		var buf safeBuf
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{output: "json"})

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

		if !waitForOutput(&buf, "JSON test", 2*time.Second) {
			t.Fatal("timed out waiting for JSON output")
		}
		sigChan <- os.Interrupt

		<-done

		output := buf.String()
		if !strings.Contains(output, "JSON test") {
			t.Errorf("Should contain log message, got: %s", output)
		}
	})
}

func TestLogsExecutor_FollowLogs(t *testing.T) {
	t.Run("dashboard streaming completes without error", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{output: "text"})

		mockClient := &mockDashboardClient{}
		done := make(chan error, 1)
		go func() {
			done <- executor.followLogsViaDashboard(context.Background(), mockClient, nil, LogLevelAll, nil, &buf)
		}()

		sigChan <- os.Interrupt
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Expected nil error, got: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for completion")
		}
	})

	t.Run("in-memory streaming completes without error", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{output: "text"})

		logChan := make(chan service.LogEntry, 10)
		subscriptions := map[string]chan service.LogEntry{
			"api": logChan,
		}
		mockLM := newMockLogManager()
		buf6, _ := service.NewLogBuffer("api", 100, false, "")
		mockLM.buffers["api"] = buf6

		done := make(chan error, 1)
		go func() {
			done <- executor.followLogsInMemory(subscriptions, mockLM, LogLevelAll, nil, &buf)
		}()

		sigChan <- os.Interrupt
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Expected nil error, got: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for completion")
		}
	})
}

func TestLogsExecutor_FollowLogsOrchestration(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("uses in-memory when buffers available", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{
			output: "text",
			timestamps: true,
			noColor:    true,
		})

		mockLM := newMockLogManager()
		logBuf, _ := service.NewLogBuffer("api", 100, false, "")
		mockLM.buffers["api"] = logBuf

		mockClient := &mockDashboardClient{
			pingErr: context.DeadlineExceeded,
		}

		done := make(chan error, 1)
		go func() {
			done <- executor.followLogs(context.Background(), tmpDir, mockLM, mockClient, nil, LogLevelAll, nil, &buf)
		}()

		sigChan <- os.Interrupt
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Expected nil error, got: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for completion")
		}
	})

	t.Run("uses in-memory for specific service filter", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{output: "text"})

		mockLM := newMockLogManager()
		logBuf, _ := service.NewLogBuffer("api", 100, false, "")
		mockLM.buffers["api"] = logBuf

		mockClient := &mockDashboardClient{}

		done := make(chan error, 1)
		go func() {
			done <- executor.followLogs(context.Background(), tmpDir, mockLM, mockClient, []string{"api"}, LogLevelAll, nil, &buf)
		}()

		sigChan <- os.Interrupt
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Expected nil error, got: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for completion")
		}
	})

	t.Run("falls back to dashboard when no buffers", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{output: "text"})

		mockLM := newMockLogManager()
		mockClient := &mockDashboardClient{}

		done := make(chan error, 1)
		go func() {
			done <- executor.followLogs(context.Background(), tmpDir, mockLM, mockClient, nil, LogLevelAll, nil, &buf)
		}()

		sigChan <- os.Interrupt
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Expected nil error, got: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for completion")
		}
	})

	t.Run("falls back to dashboard for non-existent service", func(t *testing.T) {
		var buf bytes.Buffer
		sigChan := make(chan os.Signal, 1)
		executor := newTestExecutor(&buf, sigChan, &logsOptions{output: "text"})

		mockLM := newMockLogManager()
		logBuf, _ := service.NewLogBuffer("other", 100, false, "")
		mockLM.buffers["other"] = logBuf

		mockClient := &mockDashboardClient{}

		done := make(chan error, 1)
		go func() {
			done <- executor.followLogs(context.Background(), tmpDir, mockLM, mockClient, []string{"nonexistent"}, LogLevelAll, nil, &buf)
		}()

		sigChan <- os.Interrupt
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Expected nil error, got: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for completion")
		}
	})
}
