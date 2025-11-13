package output

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// TaskStatus represents the status of a single task.
type TaskStatus string

const (
	TaskStatusPending TaskStatus = "pending"
	TaskStatusRunning TaskStatus = "running"
	TaskStatusSuccess TaskStatus = "success"
	TaskStatusFailed  TaskStatus = "failed"
	TaskStatusSkipped TaskStatus = "skipped"
)

// Progress display constants
const (
	defaultTermWidth        = 80
	refreshInterval         = 250 * time.Millisecond
	estimatedTotalBytes     = 10 * 1024 * 1024 // 10MB typical npm install
	estimatedCompletionTime = 30.0             // seconds
	bytesPerIncrement       = 1024             // 1KB per write
)

// Progress bar layout constants
const (
	iconWidth     = 2  // icon + space
	percentWidth  = 6  // " 100% "
	timeWidth     = 7  // " 999.9s"
	layoutPadding = 4  // spaces
	maxDescWidth  = 25 // Maximum description width
	minBarWidth   = 20
	maxBarWidth   = 40
)

// Progress calculation constants
// These caps prevent showing 100% before tasks actually complete,
// avoiding user confusion when progress reaches 100% but task is still running.
const (
	// Cap running progress to 95% - only Complete() sets 100%
	progressCapRunning = 95.0
	// Cap time-based estimates lower since they're less accurate
	progressCapTimeEstimate = 90.0
)

// ProgressSpinner represents a progress tracker for a single task.
type ProgressSpinner struct {
	description   string
	status        TaskStatus
	bytesWritten  int64
	finalProgress float64
	startTime     time.Time
	endTime       time.Time
	mu            sync.Mutex
	stopChan      chan struct{}
	errorMsg      string
}

// MultiProgress manages multiple concurrent progress bars.
type MultiProgress struct {
	bars          map[string]*ProgressSpinner
	barOrder      []string // Maintain insertion order
	mu            sync.RWMutex
	stopChan      chan struct{}
	stopped       bool
	lastLineCount int
	termWidth     int
}

// NewMultiProgress creates a new multi-progress manager.
func NewMultiProgress() *MultiProgress {
	return &MultiProgress{
		bars:      make(map[string]*ProgressSpinner),
		barOrder:  []string{},
		stopChan:  make(chan struct{}),
		termWidth: defaultTermWidth,
	}
}

// AddBar adds a new progress bar with the given description.
func (mp *MultiProgress) AddBar(id, description string) *ProgressSpinner {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	bar := &ProgressSpinner{
		description:   description,
		status:        TaskStatusPending,
		finalProgress: 0,
		startTime:     time.Now(),
		stopChan:      make(chan struct{}),
	}
	mp.bars[id] = bar
	mp.barOrder = append(mp.barOrder, id)
	return bar
}

// GetBar returns the progress bar with the given ID.
func (mp *MultiProgress) GetBar(id string) *ProgressSpinner {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return mp.bars[id]
}

// Start starts the multi-progress display (renders all bars periodically).
func (mp *MultiProgress) Start() {
	// Hide cursor during progress display
	fmt.Print("\033[?25l")

	// Set initial line count based on number of bars
	mp.mu.Lock()
	mp.lastLineCount = len(mp.bars)
	mp.mu.Unlock()

	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-mp.stopChan:
				return
			case <-ticker.C:
				mp.render()
			}
		}
	}()
}

// Stop stops all progress bars and displays the final state.
func (mp *MultiProgress) Stop() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.stopped {
		return
	}
	mp.stopped = true
	close(mp.stopChan)

	// Stop all individual bars
	for _, bar := range mp.bars {
		bar.Stop()
	}

	// Render one final time to show completed states with frozen timers
	mp.renderFinal()

	// Show cursor again
	fmt.Print("\033[?25h")
}

// render renders all active progress bars.
func (mp *MultiProgress) render() {
	// Use read lock to get bars snapshot, avoiding deadlock with individual bar locks
	mp.mu.RLock()
	if mp.stopped {
		mp.mu.RUnlock()
		return
	}

	// Copy bar references while holding read lock
	bars := make([]*ProgressSpinner, 0, len(mp.barOrder))
	for _, id := range mp.barOrder {
		if bar, exists := mp.bars[id]; exists {
			bars = append(bars, bar)
		}
	}
	mp.mu.RUnlock()

	// Move cursor and process bars without holding mp.mu
	mp.moveCursorToStart()

	lineCount := 0
	now := time.Now()

	for _, bar := range bars {
		bar.mu.Lock()

		elapsed := mp.calculateElapsed(bar, now)
		progressPct := mp.calculateProgress(bar, elapsed)

		// Build the progress bar line
		statusLine := mp.buildProgressLine(bar, progressPct, elapsed)
		// Clear entire line and print
		fmt.Print("\r\033[2K" + statusLine + "\n")
		lineCount++

		// Add error line if failed
		if bar.status == TaskStatusFailed && bar.errorMsg != "" {
			errorLine := mp.formatErrorLine(bar.errorMsg)
			fmt.Print("\r\033[2K" + errorLine + "\n")
			lineCount++
		}

		bar.mu.Unlock()
	}

	mp.clearExtraLines(lineCount)

	// Update lastLineCount with mutex since we released it earlier
	mp.mu.Lock()
	mp.lastLineCount = lineCount
	mp.mu.Unlock()
}

// renderFinal renders the final state of all progress bars (called once on Stop).
func (mp *MultiProgress) renderFinal() {
	mp.moveCursorToStart()

	lineCount := 0
	for _, id := range mp.barOrder {
		bar, exists := mp.bars[id]
		if !exists {
			continue
		}

		bar.mu.Lock()
		elapsed := mp.calculateElapsed(bar, time.Now())
		statusLine := mp.buildProgressLine(bar, bar.finalProgress, elapsed)
		fmt.Print("\r\033[2K" + statusLine + "\n")
		lineCount++

		if bar.status == TaskStatusFailed && bar.errorMsg != "" {
			errorLine := mp.formatErrorLine(bar.errorMsg)
			fmt.Print("\r\033[2K" + errorLine + "\n")
			lineCount++
		}
		bar.mu.Unlock()
	}

	mp.clearExtraLines(lineCount)
	mp.lastLineCount = 0
}

// buildProgressLine constructs a single progress bar line
func (mp *MultiProgress) buildProgressLine(bar *ProgressSpinner, progressPct, elapsed float64) string {
	barWidth := mp.calculateBarWidth()
	icon, color := mp.getStatusIconAndColor(bar.status, time.Now())
	barContent := mp.formatBarContent(bar.status, barWidth, progressPct)
	desc := truncateString(bar.description, maxDescWidth)
	timeStr := mp.formatElapsedTime(bar.status, elapsed)

	return mp.assembleProgressLine(icon, color, desc, barContent, progressPct, timeStr, bar.status)
}

// calculateBarWidth determines the width of the progress bar
func (mp *MultiProgress) calculateBarWidth() int {
	barWidth := mp.termWidth - iconWidth - maxDescWidth - percentWidth - timeWidth - layoutPadding
	if barWidth < minBarWidth {
		return minBarWidth
	}
	if barWidth > maxBarWidth {
		return maxBarWidth
	}
	return barWidth
}

// getStatusIconAndColor returns the appropriate icon and color for a task status
func (mp *MultiProgress) getStatusIconAndColor(status TaskStatus, t time.Time) (string, string) {

	switch status {
	case TaskStatusPending:
		return "○", Dim
	case TaskStatusRunning:
		return getSpinnerFrame(t), Cyan
	case TaskStatusSuccess:
		return "✓", Green
	case TaskStatusFailed:
		return "✗", Red
	case TaskStatusSkipped:
		return "-", Gray
	default:
		return "○", Dim
	}
}

// formatBarContent creates the visual bar content based on status and progress
func (mp *MultiProgress) formatBarContent(status TaskStatus, barWidth int, progressPct float64) string {
	filled := int(float64(barWidth) * progressPct / 100.0)
	if filled > barWidth {
		filled = barWidth
	}

	switch status {
	case TaskStatusSuccess:
		return strings.Repeat("━", barWidth)
	case TaskStatusFailed:
		return strings.Repeat("╍", filled) + strings.Repeat("╌", barWidth-filled)
	case TaskStatusRunning:
		if filled > 0 {
			return strings.Repeat("━", filled-1) + "▶" + strings.Repeat("─", barWidth-filled)
		}
		return strings.Repeat("─", barWidth)
	default:
		return strings.Repeat("─", barWidth)
	}
}

// formatElapsedTime formats the elapsed time for display
func (mp *MultiProgress) formatElapsedTime(status TaskStatus, elapsed float64) string {
	if status == TaskStatusSuccess || status == TaskStatusFailed || status == TaskStatusRunning {
		return fmt.Sprintf("%.1fs", elapsed)
	}
	return ""
}

// assembleProgressLine assembles the final progress line string
func (mp *MultiProgress) assembleProgressLine(icon, color, desc, barContent string, progressPct float64, timeStr string, status TaskStatus) string {
	if status == TaskStatusPending {
		return fmt.Sprintf("%s%s%s %-25s", color, icon, Reset, desc)
	}

	percentStr := fmt.Sprintf("%3.0f%%", progressPct)
	return fmt.Sprintf("%s%s%s %-25s [%s%s%s] %s %s",
		color, icon, Reset,
		desc,
		color, barContent, Reset,
		percentStr,
		Dim+timeStr+Reset)
}

// truncateString truncates a string to maxLen
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// moveCursorToStart moves cursor to the start of the progress section
func (mp *MultiProgress) moveCursorToStart() {
	if mp.lastLineCount > 0 {
		fmt.Printf("\033[%dA", mp.lastLineCount)
	}
}

// clearExtraLines clears extra lines if line count decreased
func (mp *MultiProgress) clearExtraLines(currentLineCount int) {
	for i := currentLineCount; i < mp.lastLineCount; i++ {
		fmt.Print("\r\033[2K\n")
	}

	if currentLineCount < mp.lastLineCount {
		fmt.Printf("\033[%dA", mp.lastLineCount-currentLineCount)
	}
}

// formatErrorLine formats an error message for display
func (mp *MultiProgress) formatErrorLine(errorMsg string) string {
	return fmt.Sprintf("   %s%s%s", Red, truncateString(errorMsg, mp.termWidth-6), Reset)
}

// calculateElapsed calculates elapsed time for a task
func (mp *MultiProgress) calculateElapsed(bar *ProgressSpinner, now time.Time) float64 {
	if bar.status == TaskStatusSuccess || bar.status == TaskStatusFailed {
		return bar.endTime.Sub(bar.startTime).Seconds()
	}
	return now.Sub(bar.startTime).Seconds()
}

// calculateProgress calculates the current progress percentage
func (mp *MultiProgress) calculateProgress(bar *ProgressSpinner, elapsed float64) float64 {
	// For completed/failed/skipped tasks, use stored final progress
	if bar.status == TaskStatusSuccess || bar.status == TaskStatusFailed || bar.status == TaskStatusSkipped {
		return bar.finalProgress
	}

	if bar.status == TaskStatusPending {
		return 0
	}

	// TaskStatusRunning - calculate based on bytes written or time
	var progressPct float64
	if bar.bytesWritten > 0 {
		progressPct = float64(bar.bytesWritten) / float64(estimatedTotalBytes) * 100
		if progressPct > progressCapRunning {
			progressPct = progressCapRunning
		}
	} else if elapsed > 0 {
		// Use time-based estimation if no bytes yet
		progressPct = (elapsed / estimatedCompletionTime) * 100
		if progressPct > progressCapTimeEstimate {
			progressPct = progressCapTimeEstimate
		}
	}

	// Store current progress to prevent going backwards
	if progressPct > bar.finalProgress {
		bar.finalProgress = progressPct
		return progressPct
	}
	return bar.finalProgress
}

// Increment increments the progress bar count (tracks bytes written).
func (pb *ProgressSpinner) Increment() {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.bytesWritten += bytesPerIncrement
}

// AddBytes adds bytes to the progress tracker.
func (pb *ProgressSpinner) AddBytes(n int64) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.bytesWritten += n
}

// Start marks the task as started and running.
func (pb *ProgressSpinner) Start() {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.status = TaskStatusRunning
	pb.startTime = time.Now()
}

// Complete marks the task as successfully completed.
func (pb *ProgressSpinner) Complete() {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	// Always set to 100 on successful completion
	pb.finalProgress = 100.0
	pb.endTime = time.Now()
	pb.status = TaskStatusSuccess
}

// Fail marks the task as failed with an optional error message.
func (pb *ProgressSpinner) Fail(errMsg string) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	// Capture current progress before failing
	pb.finalProgress = pb.calculateProgressFromBytes()
	if pb.finalProgress > 100 {
		pb.finalProgress = 100
	}
	pb.errorMsg = errMsg
	pb.endTime = time.Now()
	pb.status = TaskStatusFailed
}

// calculateProgressFromBytes calculates progress percentage from bytes written
func (pb *ProgressSpinner) calculateProgressFromBytes() float64 {
	return float64(pb.bytesWritten) / float64(estimatedTotalBytes) * 100
}

// Skip marks the task as skipped.
func (pb *ProgressSpinner) Skip() {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.finalProgress = 0
	pb.status = TaskStatusSkipped
	pb.endTime = time.Now()
}

// Stop stops the progress bar (deprecated - use Complete/Fail instead).
func (pb *ProgressSpinner) Stop() {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	// If no status was set, mark as success
	if pb.isIncomplete() {
		pb.finalProgress = 100.0
		pb.endTime = time.Now()
		pb.status = TaskStatusSuccess
	}
}

// isIncomplete returns true if the task is not yet complete
func (pb *ProgressSpinner) isIncomplete() bool {
	return pb.status == TaskStatusRunning || pb.status == TaskStatusPending
}

// getSpinnerFrame returns the current spinner character based on time.
func getSpinnerFrame(t time.Time) string {
	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	index := (t.UnixNano() / 80_000_000) % int64(len(spinnerChars))
	return spinnerChars[index]
}

// SpinnerWriter is an io.Writer that increments the progress bar on each write.
type SpinnerWriter struct {
	bar *ProgressSpinner
}

// NewSpinnerWriter creates a new spinner writer that increments the progress bar.
func NewSpinnerWriter(bar *ProgressSpinner) *SpinnerWriter {
	return &SpinnerWriter{bar: bar}
}

// Write implements io.Writer and tracks bytes written for progress.
func (sw *SpinnerWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	if n > 0 {
		sw.bar.AddBytes(int64(n))
	}
	// Discard the actual output (don't show stdout/stderr during progress)
	return n, nil
}

// ProgressResult represents the result of a task with progress tracking.
type ProgressResult struct {
	ID      string
	Success bool
	Error   error
}

// PrintStatus prints the final status for a completed task.
func PrintStatus(description string, success bool, err error) {
	if success {
		fmt.Printf("%s%s%s %s\n", Green, SymbolCheck, Reset, description)
	} else {
		fmt.Printf("%s%s%s %s\n", Red, SymbolCross, Reset, description)
	}
}

// PrintSummary prints a summary of all results.
func PrintSummary(totalCount, successCount int, failedTasks []string) {
	Newline()
	if successCount == totalCount {
		Success("Installed %d project(s)", totalCount)
	} else {
		failureCount := totalCount - successCount
		Error("Failed to install %d project(s)", failureCount)
		for _, task := range failedTasks {
			fmt.Printf("  %s%s%s %s\n", Dim, SymbolDot, Reset, task)
		}
	}
}

// EnsureInitialLines ensures there are enough blank lines for the progress bars to render.
func EnsureInitialLines(count int) {
	for i := 0; i < count; i++ {
		fmt.Println()
	}
}

// ClearLine clears the current line and moves cursor to the beginning.
func ClearLine() {
	fmt.Print("\r\033[2K")
}

// MoveCursorUp moves the cursor up by n lines.
func MoveCursorUp(n int) {
	fmt.Printf("\033[%dA", n)
}

// MoveCursorDown moves the cursor down by n lines.
func MoveCursorDown(n int) {
	fmt.Printf("\033[%dB", n)
}

// ClearLines clears n lines starting from the current cursor position.
func ClearLines(n int) {
	for i := 0; i < n; i++ {
		fmt.Print("\033[2K") // Clear line
		if i < n-1 {
			fmt.Print("\n")
		}
	}
	// Move cursor back up
	if n > 1 {
		MoveCursorUp(n - 1)
	}
	fmt.Print("\r") // Move to beginning of line
}

// OverwriteLines overwrites n lines with new content.
func OverwriteLines(n int, content string) {
	ClearLines(n)
	fmt.Print(content)
}

// StatusLine represents a status line that replaces a progress bar.
type StatusLine struct {
	Description string
	Success     bool
	Error       string
}

// FormatStatusLines formats multiple status lines into a single string.
func FormatStatusLines(lines []StatusLine) string {
	var builder strings.Builder
	for i, line := range lines {
		if line.Success {
			builder.WriteString(fmt.Sprintf("%s%s%s %s", Green, SymbolCheck, Reset, line.Description))
		} else {
			builder.WriteString(fmt.Sprintf("%s%s%s %s", Red, SymbolCross, Reset, line.Description))
		}
		if i < len(lines)-1 {
			builder.WriteString("\n")
		}
	}
	return builder.String()
}
