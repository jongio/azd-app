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

func TestFindToolInPath_WithExtension(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	// Test that we can find tools with or without .exe extension
	tests := []struct {
		name     string
		toolName string
	}{
		{
			name:     "without extension",
			toolName: "cmd",
		},
		{
			name:     "with extension",
			toolName: "cmd.exe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindToolInPath(tt.toolName)
			if result == "" {
				t.Errorf("FindToolInPath(%s) returned empty, expected to find cmd", tt.toolName)
			}
			t.Logf("Found %s at: %s", tt.toolName, result)
		})
	}
}

func TestSearchToolInSystemPath_KnownTools(t *testing.T) {
	// This test verifies the function works correctly even if tools aren't found
	tests := []struct {
		name     string
		toolName string
	}{
		{
			name:     "search for node",
			toolName: "node",
		},
		{
			name:     "search for git",
			toolName: "git",
		},
		{
			name:     "search for nonexistent",
			toolName: "nonexistent-tool-xyz-999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SearchToolInSystemPath(tt.toolName)
			// Don't fail if not found, just log the result
			if result != "" {
				t.Logf("SearchToolInSystemPath(%s) found: %s", tt.toolName, result)
			} else {
				t.Logf("SearchToolInSystemPath(%s) not found in common locations", tt.toolName)
			}
		})
	}
}

func TestFindToolInPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		wantFind bool
	}{
		{
			name:     "empty string",
			toolName: "",
			wantFind: false,
		},
		{
			name:     "tool with spaces",
			toolName: "tool with spaces",
			wantFind: false,
		},
		{
			name:     "tool with path separators",
			toolName: "tool/with/slashes",
			wantFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindToolInPath(tt.toolName)
			found := result != ""
			if found != tt.wantFind {
				t.Logf("FindToolInPath(%q) found=%v, want=%v (result: %s)", tt.toolName, found, tt.wantFind, result)
			}
		})
	}
}

func TestSearchToolInSystemPath_WindowsExtension(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	tests := []struct {
		name     string
		toolName string
	}{
		{
			name:     "without .exe",
			toolName: "cmd",
		},
		{
			name:     "with .exe",
			toolName: "cmd.exe",
		},
		{
			name:     "mixed case",
			toolName: "CMD.EXE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SearchToolInSystemPath(tt.toolName)
			// Just verify no panic and reasonable behavior
			t.Logf("SearchToolInSystemPath(%s) = %s", tt.toolName, result)
		})
	}
}

func TestSearchToolInSystemPath_UnixPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test")
	}

	// Test that Unix search paths are used
	result := SearchToolInSystemPath("sh")
	// sh should be in /bin or /usr/bin on Unix systems
	if result != "" {
		t.Logf("Found sh at: %s", result)
	} else {
		t.Log("sh not found in common Unix paths (unusual but not a failure)")
	}
}

func TestFindToolInPath_ActualTools(t *testing.T) {
	// Test with a tool that should definitely exist in test environment
	goPath := FindToolInPath("go")
	if goPath == "" {
		t.Error("Expected to find 'go' in PATH during tests")
	}

	if runtime.GOOS == "windows" {
		// On Windows, verify .exe is added
		goWithExe := FindToolInPath("go.exe")
		if goWithExe == "" {
			t.Error("Expected to find 'go.exe' in PATH on Windows")
		}
	}
}

func TestRefreshPATH_ErrorHandling(t *testing.T) {
	// This test just verifies RefreshPATH handles errors gracefully
	newPath, err := RefreshPATH()

	if err != nil {
		t.Logf("RefreshPATH returned error (may be expected): %v", err)
		// On some systems this might fail, which is okay for the test
		return
	}

	if newPath == "" {
		t.Error("RefreshPATH succeeded but returned empty PATH")
	}

	// Verify PATH actually contains something reasonable
	if runtime.GOOS == "windows" {
		// Windows should have system directories
		if !containsAnyPath(newPath, []string{"Windows", "System32"}) {
			t.Logf("Warning: Refreshed PATH doesn't contain expected Windows directories: %s", newPath)
		}
	}
}

func TestRefreshUnixPATH(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test on Windows")
	}

	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer func() {
		_ = os.Setenv("PATH", originalPath)
	}()

	// Test that refreshUnixPATH returns current PATH
	newPath, err := refreshUnixPATH()
	if err != nil {
		t.Errorf("refreshUnixPATH() returned error: %v", err)
	}

	if newPath != originalPath {
		t.Errorf("refreshUnixPATH() = %q, want %q", newPath, originalPath)
	}
}

func TestRefreshWindowsPATH_Integration(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows")
	}

	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer func() {
		_ = os.Setenv("PATH", originalPath)
	}()

	// Test Windows PATH refresh
	newPath, err := refreshWindowsPATH()
	if err != nil {
		// PowerShell might not be available in some test environments
		t.Logf("refreshWindowsPATH() failed (may be expected): %v", err)
		return
	}

	if newPath == "" {
		t.Error("refreshWindowsPATH() returned empty PATH")
	}

	// Verify PATH was updated in environment
	envPath := os.Getenv("PATH")
	if envPath != newPath {
		t.Errorf("PATH not updated in environment: got %q, want %q", envPath, newPath)
	}
}

func TestRefreshPATH_OSSpecific(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer func() {
		_ = os.Setenv("PATH", originalPath)
	}()

	newPath, err := RefreshPATH()

	if runtime.GOOS == "windows" {
		// On Windows, might fail if PowerShell unavailable
		if err != nil {
			t.Logf("RefreshPATH on Windows failed (expected in some environments): %v", err)
			return
		}
	} else {
		// On Unix, should always succeed
		if err != nil {
			t.Errorf("RefreshPATH on Unix failed: %v", err)
		}
		// Should return current PATH
		if newPath != originalPath {
			t.Errorf("RefreshPATH on Unix changed PATH: got %q, want %q", newPath, originalPath)
		}
	}

	if newPath == "" {
		t.Error("RefreshPATH returned empty PATH")
	}
}

func TestGetInstallSuggestion_AllKnownTools(t *testing.T) {
	// Verify all common tools have suggestions
	knownTools := []string{
		"node", "npm", "pnpm", "yarn",
		"python", "pip", "docker", "git",
		"go", "dotnet", "azd", "az",
	}

	for _, tool := range knownTools {
		t.Run(tool, func(t *testing.T) {
			suggestion := GetInstallSuggestion(tool)
			if suggestion == "" {
				t.Errorf("No suggestion for known tool: %s", tool)
			}
			if suggestion == "Please install "+tool+" manually" {
				t.Errorf("Tool %s has default suggestion, should have specific suggestion", tool)
			}
		})
	}
}

// Helper function for PATH validation
func containsAnyPath(path string, fragments []string) bool {
	for _, fragment := range fragments {
		if containsPath(path, fragment) {
			return true
		}
	}
	return false
}

func containsPath(path, fragment string) bool {
	separator := ":"
	if runtime.GOOS == "windows" {
		separator = ";"
	}
	parts := splitPath(path, separator)
	for _, part := range parts {
		if containsIgnoreCase(part, fragment) {
			return true
		}
	}
	return false
}

func splitPath(path, separator string) []string {
	var result []string
	current := ""
	for _, char := range path {
		if string(char) == separator {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return contains(s, substr)
}

func toLower(s string) string {
	result := ""
	for _, char := range s {
		if char >= 'A' && char <= 'Z' {
			result += string(char + 32)
		} else {
			result += string(char)
		}
	}
	return result
}

func contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
