package output

import (
	"strings"
	"testing"
	"time"
)

// TestTaskStatus tests the TaskStatus type and constants
func TestTaskStatus(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected string
	}{
		{TaskStatusPending, "pending"},
		{TaskStatusRunning, "running"},
		{TaskStatusSuccess, "success"},
		{TaskStatusFailed, "failed"},
		{TaskStatusSkipped, "skipped"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("TaskStatus = %q, want %q", tt.status, tt.expected)
			}
		})
	}
}

// TestNewMultiProgress tests the creation of a new MultiProgress instance
func TestNewMultiProgress(t *testing.T) {
	mp := NewMultiProgress()

	if mp == nil {
		t.Fatal("NewMultiProgress() returned nil")
	}

	if mp.bars == nil {
		t.Error("bars map is nil")
	}

	if mp.barOrder == nil {
		t.Error("barOrder slice is nil")
	}

	if mp.termWidth != defaultTermWidth {
		t.Errorf("termWidth = %d, want %d", mp.termWidth, defaultTermWidth)
	}

	if mp.stopped {
		t.Error("stopped should be false initially")
	}
}

// TestAddBar tests adding progress bars
func TestAddBar(t *testing.T) {
	mp := NewMultiProgress()

	bar1 := mp.AddBar("bar1", "Installing package 1")
	if bar1 == nil {
		t.Fatal("AddBar() returned nil")
	}

	if bar1.description != "Installing package 1" {
		t.Errorf("description = %q, want %q", bar1.description, "Installing package 1")
	}

	if bar1.status != TaskStatusPending {
		t.Errorf("initial status = %q, want %q", bar1.status, TaskStatusPending)
	}

	// Add second bar
	bar2 := mp.AddBar("bar2", "Installing package 2")
	if bar2 == nil {
		t.Fatal("AddBar() returned nil for second bar")
	}

	// Check order is maintained
	if len(mp.barOrder) != 2 {
		t.Errorf("barOrder length = %d, want 2", len(mp.barOrder))
	}

	if mp.barOrder[0] != "bar1" || mp.barOrder[1] != "bar2" {
		t.Errorf("barOrder = %v, want [bar1 bar2]", mp.barOrder)
	}
}

// TestGetBar tests retrieving progress bars
func TestGetBar(t *testing.T) {
	mp := NewMultiProgress()
	mp.AddBar("test-bar", "Test description")

	bar := mp.GetBar("test-bar")
	if bar == nil {
		t.Fatal("GetBar() returned nil for existing bar")
	}

	if bar.description != "Test description" {
		t.Errorf("description = %q, want %q", bar.description, "Test description")
	}

	// Test non-existent bar
	nonExistent := mp.GetBar("does-not-exist")
	if nonExistent != nil {
		t.Error("GetBar() should return nil for non-existent bar")
	}
}

// TestProgressSpinnerStart tests starting a progress spinner
func TestProgressSpinnerStart(t *testing.T) {
	mp := NewMultiProgress()
	bar := mp.AddBar("test", "Testing")

	beforeStart := time.Now()
	bar.Start()
	afterStart := time.Now()

	if bar.status != TaskStatusRunning {
		t.Errorf("status after Start() = %q, want %q", bar.status, TaskStatusRunning)
	}

	if bar.startTime.Before(beforeStart) || bar.startTime.After(afterStart) {
		t.Error("startTime not set correctly")
	}
}

// TestProgressSpinnerComplete tests completing a progress spinner
func TestProgressSpinnerComplete(t *testing.T) {
	mp := NewMultiProgress()
	bar := mp.AddBar("test", "Testing")
	bar.Start()

	time.Sleep(10 * time.Millisecond)

	beforeComplete := time.Now()
	bar.Complete()
	afterComplete := time.Now()

	if bar.status != TaskStatusSuccess {
		t.Errorf("status after Complete() = %q, want %q", bar.status, TaskStatusSuccess)
	}

	if bar.finalProgress != 100.0 {
		t.Errorf("finalProgress = %f, want 100.0", bar.finalProgress)
	}

	if bar.endTime.Before(beforeComplete) || bar.endTime.After(afterComplete) {
		t.Error("endTime not set correctly")
	}
}

// TestProgressSpinnerFail tests failing a progress spinner
func TestProgressSpinnerFail(t *testing.T) {
	mp := NewMultiProgress()
	bar := mp.AddBar("test", "Testing")
	bar.Start()

	errorMsg := "Installation failed: network error"
	bar.Fail(errorMsg)

	if bar.status != TaskStatusFailed {
		t.Errorf("status after Fail() = %q, want %q", bar.status, TaskStatusFailed)
	}

	if bar.errorMsg != errorMsg {
		t.Errorf("errorMsg = %q, want %q", bar.errorMsg, errorMsg)
	}

	if bar.endTime.IsZero() {
		t.Error("endTime not set after Fail()")
	}
}

// TestProgressSpinnerSkip tests skipping a progress spinner
func TestProgressSpinnerSkip(t *testing.T) {
	mp := NewMultiProgress()
	bar := mp.AddBar("test", "Testing")

	bar.Skip()

	if bar.status != TaskStatusSkipped {
		t.Errorf("status after Skip() = %q, want %q", bar.status, TaskStatusSkipped)
	}

	if bar.finalProgress != 0 {
		t.Errorf("finalProgress = %f, want 0", bar.finalProgress)
	}
}

// TestProgressSpinnerIncrement tests incrementing progress
func TestProgressSpinnerIncrement(t *testing.T) {
	mp := NewMultiProgress()
	bar := mp.AddBar("test", "Testing")

	initialBytes := bar.bytesWritten
	bar.Increment()

	if bar.bytesWritten != initialBytes+bytesPerIncrement {
		t.Errorf("bytesWritten after Increment() = %d, want %d", bar.bytesWritten, initialBytes+bytesPerIncrement)
	}
}

// TestProgressSpinnerAddBytes tests adding bytes
func TestProgressSpinnerAddBytes(t *testing.T) {
	mp := NewMultiProgress()
	bar := mp.AddBar("test", "Testing")

	bar.AddBytes(5000)

	if bar.bytesWritten != 5000 {
		t.Errorf("bytesWritten after AddBytes(5000) = %d, want 5000", bar.bytesWritten)
	}

	bar.AddBytes(3000)

	if bar.bytesWritten != 8000 {
		t.Errorf("bytesWritten after second AddBytes() = %d, want 8000", bar.bytesWritten)
	}
}

// TestProgressSpinnerStop tests the deprecated Stop method
func TestProgressSpinnerStop(t *testing.T) {
	mp := NewMultiProgress()
	bar := mp.AddBar("test", "Testing")
	bar.Start()

	bar.Stop()

	if bar.status != TaskStatusSuccess {
		t.Errorf("status after Stop() = %q, want %q", bar.status, TaskStatusSuccess)
	}

	if bar.finalProgress != 100.0 {
		t.Errorf("finalProgress = %f, want 100.0", bar.finalProgress)
	}
}

// TestProgressSpinnerIsIncomplete tests the isIncomplete helper
func TestProgressSpinnerIsIncomplete(t *testing.T) {
	tests := []struct {
		name     string
		status   TaskStatus
		expected bool
	}{
		{"pending is incomplete", TaskStatusPending, true},
		{"running is incomplete", TaskStatusRunning, true},
		{"success is complete", TaskStatusSuccess, false},
		{"failed is complete", TaskStatusFailed, false},
		{"skipped is complete", TaskStatusSkipped, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := &ProgressSpinner{status: tt.status}
			result := bar.isIncomplete()
			if result != tt.expected {
				t.Errorf("isIncomplete() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestCalculateProgressFromBytes tests progress calculation from bytes
func TestCalculateProgressFromBytes(t *testing.T) {
	tests := []struct {
		name         string
		bytesWritten int64
		wantMin      float64
		wantMax      float64
	}{
		{"no bytes", 0, 0, 0},
		{"half of estimated", estimatedTotalBytes / 2, 49, 51},
		{"full estimated", estimatedTotalBytes, 99, 101},
		{"more than estimated", estimatedTotalBytes * 2, 199, 201},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := &ProgressSpinner{bytesWritten: tt.bytesWritten}
			result := bar.calculateProgressFromBytes()

			if result < tt.wantMin || result > tt.wantMax {
				t.Errorf("calculateProgressFromBytes() = %f, want between %f and %f",
					result, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestMultiProgressCalculateElapsed tests elapsed time calculation
func TestMultiProgressCalculateElapsed(t *testing.T) {
	mp := NewMultiProgress()
	now := time.Now()

	// Test running task
	bar1 := &ProgressSpinner{
		status:    TaskStatusRunning,
		startTime: now.Add(-5 * time.Second),
	}

	elapsed := mp.calculateElapsed(bar1, now)
	if elapsed < 4.9 || elapsed > 5.1 {
		t.Errorf("calculateElapsed() for running task = %f, want ~5.0", elapsed)
	}

	// Test completed task
	bar2 := &ProgressSpinner{
		status:    TaskStatusSuccess,
		startTime: now.Add(-10 * time.Second),
		endTime:   now.Add(-3 * time.Second),
	}

	elapsed = mp.calculateElapsed(bar2, now)
	if elapsed < 6.9 || elapsed > 7.1 {
		t.Errorf("calculateElapsed() for completed task = %f, want ~7.0", elapsed)
	}
}

// TestMultiProgressCalculateProgress tests progress calculation
func TestMultiProgressCalculateProgress(t *testing.T) {
	mp := NewMultiProgress()

	tests := []struct {
		name          string
		status        TaskStatus
		bytesWritten  int64
		finalProgress float64
		elapsed       float64
		wantMin       float64
		wantMax       float64
	}{
		{
			name:    "pending task",
			status:  TaskStatusPending,
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:          "success task uses final progress",
			status:        TaskStatusSuccess,
			finalProgress: 100.0,
			wantMin:       100,
			wantMax:       100,
		},
		{
			name:          "failed task uses final progress",
			status:        TaskStatusFailed,
			finalProgress: 45.0,
			wantMin:       45,
			wantMax:       45,
		},
		{
			name:         "running with bytes",
			status:       TaskStatusRunning,
			bytesWritten: estimatedTotalBytes / 2,
			wantMin:      45,
			wantMax:      55,
		},
		{
			name:    "running with time only",
			status:  TaskStatusRunning,
			elapsed: estimatedCompletionTime / 2,
			wantMin: 45,
			wantMax: 55,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := &ProgressSpinner{
				status:        tt.status,
				bytesWritten:  tt.bytesWritten,
				finalProgress: tt.finalProgress,
			}

			result := mp.calculateProgress(bar, tt.elapsed)

			if result < tt.wantMin || result > tt.wantMax {
				t.Errorf("calculateProgress() = %f, want between %f and %f",
					result, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestMultiProgressCalculateBarWidth tests bar width calculation
func TestMultiProgressCalculateBarWidth(t *testing.T) {
	tests := []struct {
		name      string
		termWidth int
		want      int
	}{
		{"very narrow terminal", 40, minBarWidth},
		{"normal terminal", 80, 36},
		{"wide terminal", 200, maxBarWidth},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mp := &MultiProgress{termWidth: tt.termWidth}
			result := mp.calculateBarWidth()

			if result != tt.want {
				t.Errorf("calculateBarWidth() = %d, want %d", result, tt.want)
			}
		})
	}
}

// TestGetStatusIconAndColor tests status icon and color selection
func TestGetStatusIconAndColor(t *testing.T) {
	mp := NewMultiProgress()
	now := time.Now()

	tests := []struct {
		name      string
		status    TaskStatus
		wantColor string
		checkIcon func(string) bool
	}{
		{
			name:      "pending",
			status:    TaskStatusPending,
			wantColor: Dim,
			checkIcon: func(icon string) bool { return icon == "○" },
		},
		{
			name:      "running",
			status:    TaskStatusRunning,
			wantColor: Cyan,
			checkIcon: func(icon string) bool { return len(icon) > 0 }, // spinner char
		},
		{
			name:      "success",
			status:    TaskStatusSuccess,
			wantColor: Green,
			checkIcon: func(icon string) bool { return icon == "✓" },
		},
		{
			name:      "failed",
			status:    TaskStatusFailed,
			wantColor: Red,
			checkIcon: func(icon string) bool { return icon == "✗" },
		},
		{
			name:      "skipped",
			status:    TaskStatusSkipped,
			wantColor: Gray,
			checkIcon: func(icon string) bool { return icon == "-" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon, color := mp.getStatusIconAndColor(tt.status, now)

			if color != tt.wantColor {
				t.Errorf("getStatusIconAndColor() color = %q, want %q", color, tt.wantColor)
			}

			if !tt.checkIcon(icon) {
				t.Errorf("getStatusIconAndColor() icon = %q, unexpected", icon)
			}
		})
	}
}

// TestFormatBarContent tests bar content formatting
func TestFormatBarContent(t *testing.T) {
	mp := NewMultiProgress()

	tests := []struct {
		name        string
		status      TaskStatus
		barWidth    int
		progressPct float64
		checkResult func(string, int) bool
	}{
		{
			name:        "success full bar",
			status:      TaskStatusSuccess,
			barWidth:    20,
			progressPct: 100,
			checkResult: func(result string, width int) bool {
				return strings.Count(result, "━") == width
			},
		},
		{
			name:        "running partial",
			status:      TaskStatusRunning,
			barWidth:    20,
			progressPct: 50,
			checkResult: func(result string, width int) bool {
				// Check it contains the arrow or has right number of characters
				runeCount := len([]rune(result))
				return (strings.Contains(result, "▶") || runeCount == width) && runeCount <= width+1
			},
		},
		{
			name:        "pending empty",
			status:      TaskStatusPending,
			barWidth:    20,
			progressPct: 0,
			checkResult: func(result string, width int) bool {
				return strings.Count(result, "─") == width
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mp.formatBarContent(tt.status, tt.barWidth, tt.progressPct)

			// Use rune count for Unicode characters
			runeCount := len([]rune(result))
			if runeCount != tt.barWidth {
				t.Errorf("formatBarContent() rune count = %d, want %d (result: %q)", runeCount, tt.barWidth, result)
			}

			if !tt.checkResult(result, tt.barWidth) {
				t.Errorf("formatBarContent() = %q, failed validation", result)
			}
		})
	}
}

// TestFormatElapsedTime tests elapsed time formatting
func TestFormatElapsedTime(t *testing.T) {
	mp := NewMultiProgress()

	tests := []struct {
		name    string
		status  TaskStatus
		elapsed float64
		want    string
	}{
		{"success with time", TaskStatusSuccess, 5.3, "5.3s"},
		{"running with time", TaskStatusRunning, 12.7, "12.7s"},
		{"pending no time", TaskStatusPending, 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mp.formatElapsedTime(tt.status, tt.elapsed)

			if result != tt.want {
				t.Errorf("formatElapsedTime() = %q, want %q", result, tt.want)
			}
		})
	}
}

// TestTruncateString tests string truncation
func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncate needed", "hello world", 8, "hello..."},
		{"very short max", "hello", 3, "hel"},
		{"empty string", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)

			if result != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q",
					tt.input, tt.maxLen, result, tt.want)
			}

			if len(result) > tt.maxLen {
				t.Errorf("truncateString() result length %d exceeds maxLen %d",
					len(result), tt.maxLen)
			}
		})
	}
}

// TestGetSpinnerFrame tests spinner frame generation
func TestGetSpinnerFrame(t *testing.T) {
	now := time.Now()
	frame1 := getSpinnerFrame(now)

	if frame1 == "" {
		t.Error("getSpinnerFrame() returned empty string")
	}

	// Test that different times can produce different frames
	later := now.Add(100 * time.Millisecond)
	frame2 := getSpinnerFrame(later)

	if frame2 == "" {
		t.Error("getSpinnerFrame() returned empty string for later time")
	}

	// Both should be valid spinner characters
	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	found1 := false
	found2 := false

	for _, char := range spinnerChars {
		if frame1 == char {
			found1 = true
		}
		if frame2 == char {
			found2 = true
		}
	}

	if !found1 {
		t.Errorf("getSpinnerFrame() = %q, not a valid spinner character", frame1)
	}

	if !found2 {
		t.Errorf("getSpinnerFrame() = %q, not a valid spinner character", frame2)
	}
}

// TestSpinnerWriter tests the SpinnerWriter implementation
func TestSpinnerWriter(t *testing.T) {
	mp := NewMultiProgress()
	bar := mp.AddBar("test", "Testing")

	writer := NewSpinnerWriter(bar)
	if writer == nil {
		t.Fatal("NewSpinnerWriter() returned nil")
	}

	// Write some data
	data := []byte("test data")
	n, err := writer.Write(data)

	if err != nil {
		t.Errorf("Write() error = %v, want nil", err)
	}

	if n != len(data) {
		t.Errorf("Write() returned n = %d, want %d", n, len(data))
	}

	if bar.bytesWritten != int64(len(data)) {
		t.Errorf("bytesWritten = %d, want %d", bar.bytesWritten, len(data))
	}

	// Write more data
	moreData := []byte("more data")
	n2, err := writer.Write(moreData)

	if err != nil {
		t.Errorf("Write() error = %v, want nil", err)
	}

	expectedTotal := int64(len(data) + len(moreData))
	if bar.bytesWritten != expectedTotal {
		t.Errorf("bytesWritten after second write = %d, want %d",
			bar.bytesWritten, expectedTotal)
	}

	if n2 != len(moreData) {
		t.Errorf("second Write() returned n = %d, want %d", n2, len(moreData))
	}
}

// TestSpinnerWriterEmptyWrite tests writing empty data
func TestSpinnerWriterEmptyWrite(t *testing.T) {
	mp := NewMultiProgress()
	bar := mp.AddBar("test", "Testing")
	writer := NewSpinnerWriter(bar)

	n, err := writer.Write([]byte{})

	if err != nil {
		t.Errorf("Write() with empty data error = %v, want nil", err)
	}

	if n != 0 {
		t.Errorf("Write() with empty data returned n = %d, want 0", n)
	}

	if bar.bytesWritten != 0 {
		t.Errorf("bytesWritten after empty write = %d, want 0", bar.bytesWritten)
	}
}

// TestMultiProgressStop tests stopping multi-progress
func TestMultiProgressStop(t *testing.T) {
	mp := NewMultiProgress()
	bar1 := mp.AddBar("bar1", "Task 1")
	bar2 := mp.AddBar("bar2", "Task 2")

	bar1.Start()
	bar2.Start()

	// Ensure we don't block on stop
	done := make(chan bool, 1)
	go func() {
		mp.Stop()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Stop() blocked for too long")
	}

	if !mp.stopped {
		t.Error("stopped flag should be true after Stop()")
	}

	// Multiple stops should be safe
	mp.Stop()
	mp.Stop()
}

// TestBuildProgressLine tests building a complete progress line
func TestBuildProgressLine(t *testing.T) {
	mp := NewMultiProgress()

	tests := []struct {
		name        string
		bar         *ProgressSpinner
		progressPct float64
		elapsed     float64
		checkResult func(string) bool
	}{
		{
			name: "pending task",
			bar: &ProgressSpinner{
				description: "Installing npm",
				status:      TaskStatusPending,
			},
			progressPct: 0,
			elapsed:     0,
			checkResult: func(s string) bool {
				return strings.Contains(s, "Installing npm")
			},
		},
		{
			name: "running task",
			bar: &ProgressSpinner{
				description: "Building project",
				status:      TaskStatusRunning,
			},
			progressPct: 50,
			elapsed:     5.5,
			checkResult: func(s string) bool {
				return strings.Contains(s, "Building project") &&
					strings.Contains(s, "50%") &&
					strings.Contains(s, "5.5s")
			},
		},
		{
			name: "success task",
			bar: &ProgressSpinner{
				description: "Tests passed",
				status:      TaskStatusSuccess,
			},
			progressPct: 100,
			elapsed:     10.2,
			checkResult: func(s string) bool {
				return strings.Contains(s, "Tests passed") &&
					strings.Contains(s, "100%") &&
					strings.Contains(s, "10.2s")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mp.buildProgressLine(tt.bar, tt.progressPct, tt.elapsed)

			if result == "" {
				t.Error("buildProgressLine() returned empty string")
			}

			if !tt.checkResult(result) {
				t.Errorf("buildProgressLine() = %q, failed validation", result)
			}
		})
	}
}

// TestFormatErrorLine tests error line formatting
func TestFormatErrorLine(t *testing.T) {
	mp := NewMultiProgress()
	mp.termWidth = 80

	errorMsg := "Installation failed: network timeout"
	result := mp.formatErrorLine(errorMsg)

	if !strings.Contains(result, errorMsg) {
		t.Errorf("formatErrorLine() = %q, want to contain %q", result, errorMsg)
	}

	if !strings.Contains(result, Red) {
		t.Error("formatErrorLine() should include Red color code")
	}
}

// TestProgressConcurrency tests concurrent access to progress bars
func TestProgressConcurrency(t *testing.T) {
	mp := NewMultiProgress()
	bar := mp.AddBar("test", "Concurrent test")

	// Start multiple goroutines accessing the same bar
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			bar.Increment()
			bar.AddBytes(100)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 10 increments + 10*100 bytes
	expectedBytes := int64(10*bytesPerIncrement + 1000)
	if bar.bytesWritten != expectedBytes {
		t.Errorf("bytesWritten after concurrent access = %d, want %d",
			bar.bytesWritten, expectedBytes)
	}
}

// TestProgressStatusTransitions tests valid status transitions
func TestProgressStatusTransitions(t *testing.T) {
	mp := NewMultiProgress()
	bar := mp.AddBar("test", "Testing transitions")

	// Pending -> Running
	if bar.status != TaskStatusPending {
		t.Errorf("initial status = %q, want %q", bar.status, TaskStatusPending)
	}

	bar.Start()
	if bar.status != TaskStatusRunning {
		t.Errorf("status after Start() = %q, want %q", bar.status, TaskStatusRunning)
	}

	// Running -> Success
	bar.Complete()
	if bar.status != TaskStatusSuccess {
		t.Errorf("status after Complete() = %q, want %q", bar.status, TaskStatusSuccess)
	}

	// Test Pending -> Skipped
	bar2 := mp.AddBar("test2", "Testing skip")
	bar2.Skip()
	if bar2.status != TaskStatusSkipped {
		t.Errorf("status after Skip() = %q, want %q", bar2.status, TaskStatusSkipped)
	}

	// Test Running -> Failed
	bar3 := mp.AddBar("test3", "Testing fail")
	bar3.Start()
	bar3.Fail("error")
	if bar3.status != TaskStatusFailed {
		t.Errorf("status after Fail() = %q, want %q", bar3.status, TaskStatusFailed)
	}
}
