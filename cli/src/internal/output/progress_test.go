package output

import (
"testing"
"time"

coreprogress "github.com/jongio/azd-core/progress"
)

// TestReExportedTypes verifies that the re-exported types are compatible with azd-core/progress.
func TestReExportedTypes(t *testing.T) {
// Verify type aliases work correctly
var ts TaskStatus = TaskStatusPending
var coreTs coreprogress.TaskStatus = ts
if coreTs != coreprogress.TaskStatusPending {
t.Errorf("TaskStatus alias mismatch: got %q, want %q", coreTs, coreprogress.TaskStatusPending)
}

// Verify all status constants match
if TaskStatusPending != coreprogress.TaskStatusPending {
t.Error("TaskStatusPending mismatch")
}
if TaskStatusRunning != coreprogress.TaskStatusRunning {
t.Error("TaskStatusRunning mismatch")
}
if TaskStatusSuccess != coreprogress.TaskStatusSuccess {
t.Error("TaskStatusSuccess mismatch")
}
if TaskStatusFailed != coreprogress.TaskStatusFailed {
t.Error("TaskStatusFailed mismatch")
}
if TaskStatusSkipped != coreprogress.TaskStatusSkipped {
t.Error("TaskStatusSkipped mismatch")
}
}

// TestReExportedNewMultiProgress verifies NewMultiProgress returns a usable instance.
func TestReExportedNewMultiProgress(t *testing.T) {
mp := NewMultiProgress()
if mp == nil {
t.Fatal("NewMultiProgress() returned nil")
}

bar := mp.AddBar("test", "Test task")
if bar == nil {
t.Fatal("AddBar() returned nil")
}

bar.Start()
bar.Complete()
}

// TestReExportedSpinnerWriter verifies NewSpinnerWriter works through re-export.
func TestReExportedSpinnerWriter(t *testing.T) {
mp := NewMultiProgress()
bar := mp.AddBar("test", "Test task")

writer := NewSpinnerWriter(bar)
if writer == nil {
t.Fatal("NewSpinnerWriter() returned nil")
}

data := []byte("test data")
n, err := writer.Write(data)
if err != nil {
t.Errorf("Write() error = %v", err)
}
if n != len(data) {
t.Errorf("Write() returned n = %d, want %d", n, len(data))
}
}

// TestReExportedProgressLifecycle tests the full lifecycle through re-exports.
func TestReExportedProgressLifecycle(t *testing.T) {
mp := NewMultiProgress()

bar1 := mp.AddBar("bar1", "Task 1")
bar2 := mp.AddBar("bar2", "Task 2")
bar3 := mp.AddBar("bar3", "Task 3")

bar1.Start()
bar1.Complete()

bar2.Start()
bar2.Fail("error occurred")

bar3.Skip()

// Verify GetBar works
retrieved := mp.GetBar("bar1")
if retrieved == nil {
t.Fatal("GetBar() returned nil for existing bar")
}

// Test stop doesn't panic
done := make(chan bool, 1)
go func() {
mp.Stop()
done <- true
}()

select {
case <-done:
case <-time.After(1 * time.Second):
t.Fatal("Stop() blocked for too long")
}
}

// TestReExportedStatusLine verifies StatusLine type works.
func TestReExportedStatusLine(t *testing.T) {
lines := []StatusLine{
{Description: "Task 1", Success: true},
{Description: "Task 2", Success: false, Error: "failed"},
}

result := FormatStatusLines(lines)
if result == "" {
t.Error("FormatStatusLines() returned empty string")
}
}