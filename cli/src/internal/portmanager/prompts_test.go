package portmanager

import (
	"bufio"
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortConflictAction_Constants(t *testing.T) {
	// Verify the constants are defined
	actions := []PortConflictAction{
		ActionKill,
		ActionReassign,
		ActionCancel,
		ActionAlwaysKill,
	}

	// Just verify they have different values
	seen := make(map[PortConflictAction]bool)
	for _, action := range actions {
		if seen[action] {
			t.Errorf("Duplicate PortConflictAction value: %d", action)
		}
		seen[action] = true
	}

	if len(seen) != 4 {
		t.Errorf("Expected 4 unique PortConflictAction values, got %d", len(seen))
	}
}

func TestHandlePortConflict_ForceMode(t *testing.T) {
	pm := &PortManager{forceMode: true}

	action, err := handlePortConflict(pm, 8080, "web", " by node (PID 1234)", false)
	if err != nil {
		t.Fatalf("handlePortConflict() error = %v", err)
	}
	if action != ActionKill {
		t.Errorf("handlePortConflict() = %d, want ActionKill (%d)", action, ActionKill)
	}
}

func TestHandlePortConflict_ForceMode_Explicit(t *testing.T) {
	pm := &PortManager{forceMode: true}

	action, err := handlePortConflict(pm, 3000, "api", " by python (PID 5678)", true)
	if err != nil {
		t.Fatalf("handlePortConflict() error = %v", err)
	}
	if action != ActionKill {
		t.Errorf("handlePortConflict() = %d, want ActionKill (%d)", action, ActionKill)
	}
}

func TestHandlePortConflict_ForceTakesPriorityOverAlwaysKill(t *testing.T) {
	pm := &PortManager{
		forceMode:         true,
		sessionAlwaysKill: true,
	}

	// Force mode should be checked first
	action, err := handlePortConflict(pm, 8080, "web", "", false)
	if err != nil {
		t.Fatalf("handlePortConflict() error = %v", err)
	}
	if action != ActionKill {
		t.Errorf("handlePortConflict() = %d, want ActionKill (%d)", action, ActionKill)
	}
}

func TestSetForceMode(t *testing.T) {
	pm := &PortManager{}

	if pm.forceMode {
		t.Error("forceMode should be false by default")
	}

	pm.SetForceMode(true)
	if !pm.forceMode {
		t.Error("forceMode should be true after SetForceMode(true)")
	}

	pm.SetForceMode(false)
	if pm.forceMode {
		t.Error("forceMode should be false after SetForceMode(false)")
	}
}

func TestGetProcessInfoString(t *testing.T) {
	// This function calls getProcessInfoOnPort which is a method, not mockable easily
	// We'll test with a real PortManager instance
	pm := &PortManager{}

	// Test with a port that likely has no process
	result := getProcessInfoString(pm, 65500)

	// Result should be either empty or contain PID info
	// We can't predict the exact output, just verify it doesn't panic
	_ = result
}

func TestPrintFunctions(t *testing.T) {
	// These tests verify the print functions don't panic
	// They write to stderr so we can't easily capture output
	// but we can verify they execute without errors

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "printAutoKillMessage explicit",
			fn: func() {
				printAutoKillMessage("test-service", 8080, " by node (PID 1234)", true)
			},
		},
		{
			name: "printAutoKillMessage non-explicit",
			fn: func() {
				printAutoKillMessage("test-service", 8080, " by node (PID 1234)", false)
			},
		},
		{
			name: "printConflictMessage explicit",
			fn: func() {
				printConflictMessage("test-service", 8080, " by node (PID 1234)", true)
			},
		},
		{
			name: "printConflictMessage non-explicit",
			fn: func() {
				printConflictMessage("test-service", 8080, " by node (PID 1234)", false)
			},
		},
		{
			name: "printPortFreedMessage",
			fn: func() {
				printPortFreedMessage("test-service", 8080)
			},
		},
		{
			name: "printPortAssignedMessage",
			fn: func() {
				printPortAssignedMessage("test-service", 8080)
			},
		},
		{
			name: "printPortAvailableMessage",
			fn: func() {
				printPortAvailableMessage("test-service", 8080)
			},
		},
		{
			name: "printFindingPortMessage",
			fn: func() {
				printFindingPortMessage("test-service")
			},
		},
		{
			name: "printPreferenceSavedMessage",
			fn: func() {
				printPreferenceSavedMessage()
			},
		},
		{
			name: "printKillFailedTip",
			fn: func() {
				printKillFailedTip()
			},
		},
		{
			name: "printPortStillInUseMessage",
			fn: func() {
				printPortStillInUseMessage(8080)
			},
		},
		{
			name: "printAutoAssignedMessage",
			fn: func() {
				printAutoAssignedMessage("test-service", 8080)
			},
		},
		{
			name: "printForceKillMessage explicit",
			fn: func() {
				printForceKillMessage("test-service", 8080, " by node (PID 1234)", true)
			},
		},
		{
			name: "printForceKillMessage non-explicit",
			fn: func() {
				printForceKillMessage("test-service", 8080, " by node (PID 1234)", false)
			},
		},
		{
			name: "printNonInteractiveKillMessage explicit",
			fn: func() {
				printNonInteractiveKillMessage("test-service", 8080, " by node (PID 1234)", true)
			},
		},
		{
			name: "printNonInteractiveKillMessage non-explicit",
			fn: func() {
				printNonInteractiveKillMessage("test-service", 8080, " by node (PID 1234)", false)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Defer recover to catch any panics
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Function panicked: %v", r)
				}
			}()

			tt.fn()
		})
	}
}

func TestConstants(t *testing.T) {
	// Verify constants are defined with expected types and reasonable values
	if PortRangeStart < 1 || PortRangeStart > 65535 {
		t.Errorf("PortRangeStart = %d, should be in valid port range", PortRangeStart)
	}

	if PortRangeEnd < 1 || PortRangeEnd > 65535 {
		t.Errorf("PortRangeEnd = %d, should be in valid port range", PortRangeEnd)
	}

	if PortRangeStart >= PortRangeEnd {
		t.Errorf("PortRangeStart (%d) should be less than PortRangeEnd (%d)", PortRangeStart, PortRangeEnd)
	}

	if ProcessKillGracePeriod <= 0 {
		t.Errorf("ProcessKillGracePeriod = %v, should be positive", ProcessKillGracePeriod)
	}

	if ProcessKillMaxRetries < 1 {
		t.Errorf("ProcessKillMaxRetries = %d, should be at least 1", ProcessKillMaxRetries)
	}

	if ProcessKillTimeout <= 0 {
		t.Errorf("ProcessKillTimeout = %v, should be positive", ProcessKillTimeout)
	}

	if StalePortCleanupAge <= 0 {
		t.Errorf("StalePortCleanupAge = %v, should be positive", StalePortCleanupAge)
	}
}

// errReadFailure is an arbitrary non-EOF read failure used to prove that
// partial data from a broken stream is never treated as an answer.
var errReadFailure = errors.New("read failure")

// truncatedReader returns data followed by a non-EOF error, mimicking a stream
// that breaks midway through a line.
type truncatedReader struct {
	data []byte
	pos  int
}

func (r *truncatedReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, errReadFailure
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func TestReadPromptLine(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "answer with newline", input: "2\n", want: "2"},
		{name: "answer with carriage return", input: "3\r\n", want: "3"},
		{name: "surrounding whitespace trimmed", input: "  1  \n", want: "1"},
		{name: "answer without trailing newline is honoured", input: "4", want: "4"},
		{name: "empty stream reports no input", input: "", wantErr: errNoInput},
		{name: "whitespace only stream reports no input", input: "   ", wantErr: errNoInput},
		{name: "blank line is a valid empty answer", input: "\n", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readPromptLine(bufio.NewReader(strings.NewReader(tt.input)))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("readPromptLine() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("readPromptLine() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("readPromptLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

// A truncated read from a broken stream is not consent to pick a destructive
// menu entry, so the partial data must be discarded and the error surfaced.
func TestReadPromptLineDiscardsPartialDataOnNonEOFError(t *testing.T) {
	got, err := readPromptLine(bufio.NewReader(&truncatedReader{data: []byte("2")}))

	if !errors.Is(err, errReadFailure) {
		t.Fatalf("readPromptLine() error = %v, want %v", err, errReadFailure)
	}
	if errors.Is(err, errNoInput) {
		t.Error("readPromptLine() reported errNoInput, which would trigger the non-interactive fallback")
	}
	if got != "" {
		t.Errorf("readPromptLine() = %q, want empty so no menu choice is inferred", got)
	}
}

func TestParsePortConflictChoice(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     PortConflictAction
	}{
		{name: "always kill", response: "1", want: ActionAlwaysKill},
		{name: "kill", response: "2", want: ActionKill},
		{name: "reassign", response: "3", want: ActionReassign},
		{name: "cancel", response: "4", want: ActionCancel},
		{name: "empty cancels", response: "", want: ActionCancel},
		{name: "unrecognised cancels", response: "9", want: ActionCancel},
		{name: "word cancels", response: "kill", want: ActionCancel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parsePortConflictChoice(tt.response); got != tt.want {
				t.Errorf("parsePortConflictChoice(%q) = %d, want %d", tt.response, got, tt.want)
			}
		})
	}
}

// nullDevicePath returns the platform's null device, which is a character
// device that yields EOF immediately.
func nullDevicePath() string {
	if runtime.GOOS == "windows" {
		return "NUL"
	}
	return "/dev/null"
}

// useStdin swaps os.Stdin for the duration of a test.
func useStdin(t *testing.T, f *os.File) {
	t.Helper()
	original := os.Stdin
	os.Stdin = f
	t.Cleanup(func() { os.Stdin = original })
}

// Regression test for #556. The azd host hands down a console-like stdin that
// nothing ever writes to, so the prompt looks interactive but reads EOF. That
// used to abort the whole run with "failed to read user input: EOF". It must now
// degrade to the documented non-interactive behaviour instead.
func TestHandlePortConflict_InteractiveStdinYieldingEOF_DoesNotError(t *testing.T) {
	nullDev, err := os.Open(nullDevicePath())
	require.NoError(t, err)
	t.Cleanup(func() { _ = nullDev.Close() })
	useStdin(t, nullDev)

	// The scenario only matters when stdin looks interactive; otherwise the
	// earlier non-interactive guard would handle it and prove nothing.
	require.True(t, isStdinInteractive(), "null device should report as a character device")

	pm := &PortManager{}
	action, err := handlePortConflict(pm, 3000, "api", " by node (PID 1)", true)

	require.NoError(t, err, "EOF at the prompt must not fail the run")
	assert.Equal(t, ActionKill, action)
}

// The non-interactive guard still short-circuits before the prompt is drawn.
func TestHandlePortConflict_NonInteractiveStdin_AutoKills(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() { _ = r.Close() })
	require.NoError(t, w.Close())
	useStdin(t, r)

	require.False(t, isStdinInteractive(), "a pipe must not look interactive")

	action, err := handlePortConflict(&PortManager{}, 3000, "api", "", false)

	require.NoError(t, err)
	assert.Equal(t, ActionKill, action)
}
