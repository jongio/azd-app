// progress.go re-exports progress types and functions from azd-core/progress.
// New code should import github.com/jongio/azd-core/progress directly.
package output

import (
coreprogress "github.com/jongio/azd-core/progress"
)

// Re-export all types from azd-core/progress
type TaskStatus = coreprogress.TaskStatus
type ProgressSpinner = coreprogress.ProgressSpinner
type MultiProgress = coreprogress.MultiProgress
type SpinnerWriter = coreprogress.SpinnerWriter
type ProgressResult = coreprogress.ProgressResult
type StatusLine = coreprogress.StatusLine

// Re-export constants
const (
TaskStatusPending = coreprogress.TaskStatusPending
TaskStatusRunning = coreprogress.TaskStatusRunning
TaskStatusSuccess = coreprogress.TaskStatusSuccess
TaskStatusFailed  = coreprogress.TaskStatusFailed
TaskStatusSkipped = coreprogress.TaskStatusSkipped
)

// Re-export functions
var (
NewMultiProgress   = coreprogress.NewMultiProgress
NewSpinnerWriter   = coreprogress.NewSpinnerWriter
PrintStatus        = coreprogress.PrintStatus
PrintSummary       = coreprogress.PrintSummary
EnsureInitialLines = coreprogress.EnsureInitialLines
ClearLine          = coreprogress.ClearLine
MoveCursorUp       = coreprogress.MoveCursorUp
MoveCursorDown     = coreprogress.MoveCursorDown
ClearLines         = coreprogress.ClearLines
OverwriteLines     = coreprogress.OverwriteLines
FormatStatusLines  = coreprogress.FormatStatusLines
)