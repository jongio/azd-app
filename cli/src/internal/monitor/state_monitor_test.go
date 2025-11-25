package monitor

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
)

func TestNewStateMonitor(t *testing.T) {
	reg := registry.GetRegistry(t.TempDir())
	config := DefaultMonitorConfig()

	monitor := NewStateMonitor(reg, config)

	if monitor == nil {
		t.Fatal("NewStateMonitor() returned nil")
	}
	if monitor.interval != config.Interval {
		t.Errorf("interval = %v, want %v", monitor.interval, config.Interval)
	}
	if monitor.maxHistory != config.MaxHistory {
		t.Errorf("maxHistory = %d, want %d", monitor.maxHistory, config.MaxHistory)
	}
	if monitor.rateLimitWindow != config.RateLimitWindow {
		t.Errorf("rateLimitWindow = %v, want %v", monitor.rateLimitWindow, config.RateLimitWindow)
	}
}

func TestDefaultMonitorConfig(t *testing.T) {
	config := DefaultMonitorConfig()

	if config.Interval != 5*time.Second {
		t.Errorf("Interval = %v, want 5s", config.Interval)
	}
	if config.MaxHistory != 1000 {
		t.Errorf("MaxHistory = %d, want 1000", config.MaxHistory)
	}
	if config.RateLimitWindow != 5*time.Minute {
		t.Errorf("RateLimitWindow = %v, want 5m", config.RateLimitWindow)
	}
}

func TestStateMonitor_StartStop(t *testing.T) {
	reg := registry.GetRegistry(t.TempDir())
	config := DefaultMonitorConfig()
	config.Interval = 100 * time.Millisecond

	monitor := NewStateMonitor(reg, config)
	monitor.Start()

	// Let it run for a bit
	time.Sleep(250 * time.Millisecond)

	// Stop should not hang
	done := make(chan bool)
	go func() {
		monitor.Stop()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() timed out")
	}
}

func TestStateMonitor_DetectProcessCrash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping process crash test in short mode")
	}

	tempDir := t.TempDir()
	reg := registry.GetRegistry(tempDir)

	// Start a real process
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("timeout", "10")
	} else {
		cmd = exec.Command("sleep", "10")
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start test process: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	// Register the service
	entry := &registry.ServiceRegistryEntry{
		Name:      "test-service",
		PID:       cmd.Process.Pid,
		Port:      8080,
		Status:    "running",
		Health:    "healthy",
		StartTime: time.Now(),
	}
	if err := reg.Register(entry); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Create monitor with fast polling
	config := DefaultMonitorConfig()
	config.Interval = 100 * time.Millisecond
	config.RateLimitWindow = 1 * time.Second

	monitor := NewStateMonitor(reg, config)

	// Add listener to capture transitions
	var transitionMu sync.Mutex
	var transitions []StateTransition
	monitor.AddListener(func(t StateTransition) {
		transitionMu.Lock()
		defer transitionMu.Unlock()
		transitions = append(transitions, t)
	})

	monitor.Start()
	defer monitor.Stop()

	// Wait for initial state capture
	time.Sleep(200 * time.Millisecond)

	// Kill the process
	if err := cmd.Process.Kill(); err != nil {
		// On Windows, timeout.exe might already have exited or be protected
		// Skip this specific test on Windows if we can't kill it
		if runtime.GOOS == "windows" {
			t.Skip("Cannot kill timeout.exe on Windows due to permissions")
		}
		t.Fatalf("Failed to kill test process: %v", err)
	}

	// Wait for monitor to detect the crash
	time.Sleep(500 * time.Millisecond)

	// Check if crash was detected
	transitionMu.Lock()
	defer transitionMu.Unlock()

	found := false
	for _, trans := range transitions {
		if trans.Severity == SeverityCritical && trans.ServiceName == "test-service" {
			found = true
			if trans.Description == "" {
				t.Error("Transition description is empty")
			}
			t.Logf("Detected transition: %s", trans.Description)
		}
	}

	if !found {
		t.Error("Process crash was not detected")
	}
}

func TestStateMonitor_DetectHealthChange(t *testing.T) {
	tempDir := t.TempDir()
	reg := registry.GetRegistry(tempDir)

	// Register a healthy service
	entry := &registry.ServiceRegistryEntry{
		Name:      "test-service",
		PID:       os.Getpid(), // Use our own PID
		Port:      8080,
		Status:    "running",
		Health:    "healthy",
		StartTime: time.Now(),
	}
	if err := reg.Register(entry); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Create monitor
	config := DefaultMonitorConfig()
	config.Interval = 100 * time.Millisecond
	config.RateLimitWindow = 1 * time.Second

	monitor := NewStateMonitor(reg, config)

	var transitionMu sync.Mutex
	var transitions []StateTransition
	monitor.AddListener(func(t StateTransition) {
		transitionMu.Lock()
		defer transitionMu.Unlock()
		transitions = append(transitions, t)
	})

	monitor.Start()
	defer monitor.Stop()

	// Wait for initial state
	time.Sleep(200 * time.Millisecond)

	// Change health to unhealthy
	if err := reg.UpdateStatus("test-service", "running", "unhealthy"); err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	// Wait for detection
	time.Sleep(300 * time.Millisecond)

	// Check transitions
	transitionMu.Lock()
	defer transitionMu.Unlock()

	found := false
	for _, trans := range transitions {
		if trans.Severity == SeverityCritical &&
			trans.ServiceName == "test-service" &&
			trans.ToState.Health == "unhealthy" {
			found = true
			t.Logf("Detected health change: %s", trans.Description)
		}
	}

	if !found {
		t.Error("Health change was not detected")
	}
}

func TestStateMonitor_DetectStatusChange(t *testing.T) {
	tempDir := t.TempDir()
	reg := registry.GetRegistry(tempDir)

	// Register a running service
	entry := &registry.ServiceRegistryEntry{
		Name:      "test-service",
		PID:       os.Getpid(),
		Port:      8080,
		Status:    "running",
		Health:    "healthy",
		StartTime: time.Now(),
	}
	if err := reg.Register(entry); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	config := DefaultMonitorConfig()
	config.Interval = 100 * time.Millisecond

	monitor := NewStateMonitor(reg, config)

	var transitionMu sync.Mutex
	var transitions []StateTransition
	monitor.AddListener(func(t StateTransition) {
		transitionMu.Lock()
		defer transitionMu.Unlock()
		transitions = append(transitions, t)
	})

	monitor.Start()
	defer monitor.Stop()

	time.Sleep(200 * time.Millisecond)

	// Change status to error
	if err := reg.UpdateStatus("test-service", "error", "unhealthy"); err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	transitionMu.Lock()
	defer transitionMu.Unlock()

	found := false
	for _, trans := range transitions {
		if trans.Severity == SeverityCritical &&
			trans.ToState.Status == "error" {
			found = true
			t.Logf("Detected status change: %s", trans.Description)
		}
	}

	if !found {
		t.Error("Status change to error was not detected")
	}
}

func TestStateMonitor_RateLimiting(t *testing.T) {
	tempDir := t.TempDir()
	reg := registry.GetRegistry(tempDir)

	entry := &registry.ServiceRegistryEntry{
		Name:      "test-service",
		PID:       os.Getpid(),
		Port:      8080,
		Status:    "starting",
		Health:    "unknown",
		StartTime: time.Now(),
	}
	if err := reg.Register(entry); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	config := DefaultMonitorConfig()
	config.Interval = 50 * time.Millisecond
	config.RateLimitWindow = 500 * time.Millisecond

	monitor := NewStateMonitor(reg, config)

	var transitionMu sync.Mutex
	var transitions []StateTransition
	monitor.AddListener(func(t StateTransition) {
		transitionMu.Lock()
		defer transitionMu.Unlock()
		transitions = append(transitions, t)
	})

	monitor.Start()
	defer monitor.Stop()

	time.Sleep(100 * time.Millisecond)

	// Trigger multiple warning transitions rapidly
	// These should be rate limited
	for i := 0; i < 5; i++ {
		if err := reg.UpdateStatus("test-service", "starting", "unknown"); err != nil {
			t.Fatalf("Failed to update status: %v", err)
		}
		time.Sleep(60 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)

	transitionMu.Lock()
	defer transitionMu.Unlock()

	// Should have fewer transitions than updates due to rate limiting
	warningCount := 0
	for _, trans := range transitions {
		if trans.Severity == SeverityWarning {
			warningCount++
		}
	}

	// Should be rate limited, so expect significantly fewer than 5
	if warningCount >= 4 {
		t.Errorf("Rate limiting not working: got %d warning transitions, expected fewer due to rate limiting", warningCount)
	}
	t.Logf("Rate limiting working: %d warnings from 5 rapid updates", warningCount)
}

func TestStateMonitor_GetHistory(t *testing.T) {
	tempDir := t.TempDir()
	reg := registry.GetRegistry(tempDir)

	config := DefaultMonitorConfig()
	config.MaxHistory = 10

	monitor := NewStateMonitor(reg, config)

	// Add some mock transitions
	monitor.mu.Lock()
	for i := 0; i < 5; i++ {
		transition := StateTransition{
			ServiceName: "test-service",
			Severity:    SeverityInfo,
			Description: "Test transition",
			Timestamp:   time.Now(),
		}
		monitor.stateHistory = append(monitor.stateHistory, transition)
	}
	monitor.mu.Unlock()

	history := monitor.GetHistory()

	if len(history) != 5 {
		t.Errorf("GetHistory() returned %d items, want 5", len(history))
	}

	// Verify it's a copy (modifying shouldn't affect internal state)
	history[0].Description = "Modified"

	monitor.mu.RLock()
	if monitor.stateHistory[0].Description == "Modified" {
		t.Error("GetHistory() did not return a copy")
	}
	monitor.mu.RUnlock()
}

func TestStateMonitor_GetCurrentState(t *testing.T) {
	tempDir := t.TempDir()
	reg := registry.GetRegistry(tempDir)

	monitor := NewStateMonitor(reg, DefaultMonitorConfig())

	// Add a mock state
	testState := &ServiceState{
		Name:   "test-service",
		Status: "running",
		Health: "healthy",
		PID:    1234,
	}

	monitor.mu.Lock()
	monitor.previousStates["test-service"] = testState
	monitor.mu.Unlock()

	state, exists := monitor.GetCurrentState("test-service")

	if !exists {
		t.Fatal("GetCurrentState() returned exists=false")
	}
	if state.Name != "test-service" {
		t.Errorf("Name = %s, want test-service", state.Name)
	}
	if state.Status != "running" {
		t.Errorf("Status = %s, want running", state.Status)
	}

	// Test non-existent service
	_, exists = monitor.GetCurrentState("non-existent")
	if exists {
		t.Error("GetCurrentState() returned exists=true for non-existent service")
	}
}

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		severity Severity
		want     string
	}{
		{SeverityInfo, "info"},
		{SeverityWarning, "warning"},
		{SeverityCritical, "critical"},
		{Severity(999), "unknown"},
	}

	for _, tt := range tests {
		got := tt.severity.String()
		if got != tt.want {
			t.Errorf("Severity(%d).String() = %s, want %s", tt.severity, got, tt.want)
		}
	}
}

func TestIsProcessRunning(t *testing.T) {
	// Test with our own PID (should be running)
	if !isProcessRunning(os.Getpid()) {
		t.Error("isProcessRunning(current PID) = false, want true")
	}

	// Test with invalid PID
	if isProcessRunning(0) {
		t.Error("isProcessRunning(0) = true, want false")
	}
	if isProcessRunning(-1) {
		t.Error("isProcessRunning(-1) = true, want false")
	}

	// Test with likely non-existent PID
	if isProcessRunning(999999) {
		t.Error("isProcessRunning(999999) = true, want false")
	}
}

func TestStateMonitor_MultipleListeners(t *testing.T) {
	tempDir := t.TempDir()
	reg := registry.GetRegistry(tempDir)

	config := DefaultMonitorConfig()
	config.Interval = 100 * time.Millisecond

	monitor := NewStateMonitor(reg, config)

	// Add multiple listeners
	var listener1Called, listener2Called bool
	var mu sync.Mutex

	monitor.AddListener(func(t StateTransition) {
		mu.Lock()
		defer mu.Unlock()
		listener1Called = true
	})

	monitor.AddListener(func(t StateTransition) {
		mu.Lock()
		defer mu.Unlock()
		listener2Called = true
	})

	// Register and update service to trigger transition
	entry := &registry.ServiceRegistryEntry{
		Name:      "test-service",
		PID:       os.Getpid(),
		Status:    "running",
		Health:    "healthy",
		StartTime: time.Now(),
	}
	if err := reg.Register(entry); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	monitor.Start()
	defer monitor.Stop()

	time.Sleep(200 * time.Millisecond)

	// Trigger transition
	if err := reg.UpdateStatus("test-service", "error", "unhealthy"); err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if !listener1Called {
		t.Error("Listener 1 was not called")
	}
	if !listener2Called {
		t.Error("Listener 2 was not called")
	}
}

func TestStateMonitor_HistoryLimit(t *testing.T) {
	tempDir := t.TempDir()
	reg := registry.GetRegistry(tempDir)

	config := DefaultMonitorConfig()
	config.MaxHistory = 5

	monitor := NewStateMonitor(reg, config)

	// Add more transitions than max history
	monitor.mu.Lock()
	for i := 0; i < 10; i++ {
		transition := StateTransition{
			ServiceName: "test-service",
			Severity:    SeverityInfo,
			Description: "Test transition",
			Timestamp:   time.Now().Add(time.Duration(i) * time.Second),
		}
		monitor.addTransitionLocked(&transition)
	}
	monitor.mu.Unlock()

	history := monitor.GetHistory()

	if len(history) != 5 {
		t.Errorf("History length = %d, want %d (maxHistory)", len(history), config.MaxHistory)
	}

	// Verify we kept the most recent ones
	// The last transition should be the most recent timestamp
	if len(history) > 0 {
		last := history[len(history)-1]
		// Should be close to the 10th second
		expectedTime := time.Now().Add(9 * time.Second)
		if last.Timestamp.Before(expectedTime.Add(-2 * time.Second)) {
			t.Error("History did not keep most recent transitions")
		}
	}
}

func TestStateMonitor_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	reg := registry.GetRegistry(tempDir)

	config := DefaultMonitorConfig()
	config.Interval = 10 * time.Millisecond

	monitor := NewStateMonitor(reg, config)
	monitor.Start()
	defer monitor.Stop()

	// Concurrent operations
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// Add listeners concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			monitor.AddListener(func(t StateTransition) {})
		}()
	}

	// Get history concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					_ = monitor.GetHistory()
					time.Sleep(5 * time.Millisecond)
				}
			}
		}()
	}

	// Get current state concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					_, _ = monitor.GetCurrentState("test")
					time.Sleep(5 * time.Millisecond)
				}
			}
		}()
	}

	// Register services concurrently
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			entry := &registry.ServiceRegistryEntry{
				Name:      "service-" + string(rune('A'+idx)),
				PID:       os.Getpid(),
				Status:    "running",
				Health:    "healthy",
				StartTime: time.Now(),
			}
			_ = reg.Register(entry)
		}(i)
	}

	// Wait for context timeout
	<-ctx.Done()

	// No test should panic - if we get here, concurrent access is safe
	t.Log("Concurrent access test completed without panic")
}
