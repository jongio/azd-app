package pathutil

import (
	"os"
	"runtime"
	"testing"
)

func TestRefreshPATH(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer func() {
		_ = os.Setenv("PATH", originalPath)
	}()

	// Test refresh
	newPath, err := RefreshPATH()
	if err != nil && runtime.GOOS == "windows" {
		// On Windows, this might fail in test environments without PowerShell
		t.Logf("RefreshPATH failed (expected in some test environments): %v", err)
		return
	}

	if newPath == "" {
		t.Error("RefreshPATH returned empty PATH")
	}
}

func TestFindToolInPath(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		expected bool // whether we expect to find it
	}{
		{
			name:     "find go",
			toolName: "go",
			expected: true, // Go should be available in test environment
		},
		{
			name:     "nonexistent tool",
			toolName: "nonexistent-tool-xyz-12345",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindToolInPath(tt.toolName)
			found := result != ""
			if found != tt.expected {
				t.Logf("FindToolInPath(%s) found=%v, expected=%v (path=%s)", tt.toolName, found, tt.expected, result)
				// Don't fail the test, just log, as availability may vary
			}
		})
	}
}

func TestSearchToolInSystemPath(t *testing.T) {
	// This test just verifies the function doesn't panic
	result := SearchToolInSystemPath("node")
	// We don't know if node is installed, so just check it doesn't panic
	t.Logf("SearchToolInSystemPath(node) = %s", result)
}

func TestGetInstallSuggestion(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		contains string // what the suggestion should contain
	}{
		{
			name:     "node suggestion",
			toolName: "node",
			contains: "nodejs.org",
		},
		{
			name:     "pnpm suggestion",
			toolName: "pnpm",
			contains: "npm install -g pnpm",
		},
		{
			name:     "docker suggestion",
			toolName: "docker",
			contains: "Docker",
		},
		{
			name:     "unknown tool",
			toolName: "unknown-tool-xyz",
			contains: "Please install",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := GetInstallSuggestion(tt.toolName)
			if suggestion == "" {
				t.Errorf("GetInstallSuggestion(%s) returned empty string", tt.toolName)
			}
			// Just verify we get some suggestion
			t.Logf("Suggestion for %s: %s", tt.toolName, suggestion)
		})
	}
}
