package testing

import (
	"os"
	"testing"
)

func TestOutputModeString(t *testing.T) {
	tests := []struct {
		mode     OutputMode
		expected string
	}{
		{OutputModeStream, "stream"},
		{OutputModeStreamPrefixed, "stream-prefixed"},
		{OutputModeProgress, "progress"},
		{OutputMode(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.expected {
				t.Errorf("OutputMode.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSelectOutputMode_SingleService(t *testing.T) {
	opts := OutputModeOptions{
		Parallel: false,
	}

	// Single service should always use stream mode (in TTY)
	mode := SelectOutputMode(opts, 1, true)
	if mode != OutputModeStream {
		t.Errorf("Single service should use OutputModeStream, got %v", mode)
	}
}

func TestSelectOutputMode_ForceStream(t *testing.T) {
	opts := OutputModeOptions{
		ForceStream: true,
		Parallel:    true,
	}

	// ForceStream should override parallel mode
	mode := SelectOutputMode(opts, 5, true)
	if mode != OutputModeStream {
		t.Errorf("ForceStream should use OutputModeStream, got %v", mode)
	}
}

func TestSelectOutputMode_ForceProgress(t *testing.T) {
	opts := OutputModeOptions{
		ForceProgress: true,
		Parallel:      false,
	}

	// ForceProgress should use progress mode even with single service
	mode := SelectOutputMode(opts, 1, true)
	if mode != OutputModeProgress {
		t.Errorf("ForceProgress should use OutputModeProgress, got %v", mode)
	}
}

func TestSelectOutputMode_MultipleSequential(t *testing.T) {
	opts := OutputModeOptions{
		Parallel: false,
	}

	// Multiple services running sequentially should use prefixed stream
	mode := SelectOutputMode(opts, 3, true)
	if mode != OutputModeStreamPrefixed {
		t.Errorf("Multiple sequential should use OutputModeStreamPrefixed, got %v", mode)
	}
}

func TestSelectOutputMode_MultipleParallel(t *testing.T) {
	opts := OutputModeOptions{
		Parallel: true,
	}

	// Multiple services running in parallel should use progress
	mode := SelectOutputMode(opts, 3, true)
	if mode != OutputModeProgress {
		t.Errorf("Multiple parallel should use OutputModeProgress, got %v", mode)
	}
}

func TestSelectOutputMode_NonTTY_SingleService(t *testing.T) {
	opts := OutputModeOptions{
		Parallel: true,
	}

	// Non-TTY single service should use stream
	mode := SelectOutputMode(opts, 1, false)
	if mode != OutputModeStream {
		t.Errorf("Non-TTY single service should use OutputModeStream, got %v", mode)
	}
}

func TestSelectOutputMode_NonTTY_MultipleServices(t *testing.T) {
	opts := OutputModeOptions{
		Parallel: true,
	}

	// Non-TTY multiple services should use prefixed stream
	mode := SelectOutputMode(opts, 3, false)
	if mode != OutputModeStreamPrefixed {
		t.Errorf("Non-TTY multiple services should use OutputModeStreamPrefixed, got %v", mode)
	}
}

func TestSelectOutputMode_CI_OverridesForceProgress(t *testing.T) {
	// Set CI environment variable
	oldCI := os.Getenv("CI")
	os.Setenv("CI", "true")
	defer func() {
		if oldCI == "" {
			os.Unsetenv("CI")
		} else {
			os.Setenv("CI", oldCI)
		}
	}()

	opts := OutputModeOptions{
		ForceProgress: true, // Explicit user flag takes precedence over CI
		Parallel:      true,
	}

	// ForceProgress should take precedence over CI detection
	mode := SelectOutputMode(opts, 1, true)
	if mode != OutputModeProgress {
		t.Errorf("ForceProgress should take precedence over CI, got %v", mode)
	}
}

func TestSelectOutputMode_CI_MultipleServices(t *testing.T) {
	// Set CI environment variable
	oldCI := os.Getenv("CI")
	os.Setenv("CI", "true")
	defer func() {
		if oldCI == "" {
			os.Unsetenv("CI")
		} else {
			os.Setenv("CI", oldCI)
		}
	}()

	opts := OutputModeOptions{
		Parallel: true,
	}

	// CI with multiple services should use prefixed stream
	mode := SelectOutputMode(opts, 3, true)
	if mode != OutputModeStreamPrefixed {
		t.Errorf("CI with multiple services should use OutputModeStreamPrefixed, got %v", mode)
	}
}

func TestIsCI(t *testing.T) {
	// Test with no CI variables
	oldVars := make(map[string]string)
	ciVars := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "TRAVIS", "TF_BUILD"}

	// Save and clear CI variables
	for _, v := range ciVars {
		oldVars[v] = os.Getenv(v)
		os.Unsetenv(v)
	}

	// Restore after test
	defer func() {
		for k, v := range oldVars {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Should not detect CI when no variables are set
	if isCI() {
		t.Error("isCI() should return false when no CI variables are set")
	}

	// Test each CI variable
	testCases := []struct {
		envVar string
		value  string
	}{
		{"CI", "true"},
		{"GITHUB_ACTIONS", "true"},
		{"GITLAB_CI", "true"},
		{"TF_BUILD", "True"},
	}

	for _, tc := range testCases {
		t.Run(tc.envVar, func(t *testing.T) {
			os.Setenv(tc.envVar, tc.value)
			defer os.Unsetenv(tc.envVar)

			if !isCI() {
				t.Errorf("isCI() should return true when %s is set", tc.envVar)
			}
		})
	}
}

func TestSelectOutputMode_ZeroServices(t *testing.T) {
	opts := OutputModeOptions{
		Parallel: true,
	}

	// Zero services should still return a valid mode
	mode := SelectOutputMode(opts, 0, true)
	if mode != OutputModeStream {
		t.Errorf("Zero services should use OutputModeStream, got %v", mode)
	}
}

func TestSelectOutputMode_ForceStreamTakesPrecedenceOverForceProgress(t *testing.T) {
	// When both flags are set, ForceStream should take precedence
	// because the logic checks ForceStream first
	opts := OutputModeOptions{
		ForceStream:   true,
		ForceProgress: true,
		Parallel:      true,
	}

	mode := SelectOutputMode(opts, 3, true)
	if mode != OutputModeStream {
		t.Errorf("ForceStream should take precedence over ForceProgress, got %v", mode)
	}
}
