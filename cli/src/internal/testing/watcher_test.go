package testing

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewFileWatcher(t *testing.T) {
	tmpDir := t.TempDir()
	paths := []string{tmpDir}

	watcher := NewFileWatcher(paths)
	if watcher == nil {
		t.Fatal("Expected watcher to be created")
	}
	if len(watcher.paths) != 1 {
		t.Errorf("Expected 1 path, got %d", len(watcher.paths))
	}
	if watcher.pollInterval != 500*time.Millisecond {
		t.Errorf("Expected interval 500ms, got %v", watcher.pollInterval)
	}
	if len(watcher.ignorePatterns) == 0 {
		t.Error("Expected ignore patterns to be set")
	}
}

func TestFileWatcherScanFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some test files
	testFile1 := filepath.Join(tmpDir, "test1.js")
	testFile2 := filepath.Join(tmpDir, "test2.py")
	if err := os.WriteFile(testFile1, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	watcher := NewFileWatcher([]string{tmpDir})
	err := watcher.scanFiles()
	if err != nil {
		t.Fatalf("scanFiles failed: %v", err)
	}

	if len(watcher.lastCheck) < 2 {
		t.Errorf("Expected at least 2 files tracked, got %d", len(watcher.lastCheck))
	}

	// Check that both files are tracked
	_, foundFile1 := watcher.lastCheck[testFile1]
	_, foundFile2 := watcher.lastCheck[testFile2]

	if !foundFile1 || !foundFile2 {
		t.Error("Expected both test files to be tracked")
	}
}

func TestFileWatcherShouldIgnore(t *testing.T) {
	watcher := NewFileWatcher([]string{})

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Should ignore node_modules",
			path:     "/path/to/node_modules/file.js",
			expected: true,
		},
		{
			name:     "Should ignore .git",
			path:     "/path/to/.git/config",
			expected: true,
		},
		{
			name:     "Should ignore coverage",
			path:     "/path/to/coverage/lcov.info",
			expected: true,
		},
		{
			name:     "Should ignore test-results",
			path:     "/path/to/test-results/report.xml",
			expected: true,
		},
		{
			name:     "Should ignore __pycache__",
			path:     "/path/to/__pycache__/file.pyc",
			expected: true,
		},
		{
			name:     "Should not ignore regular files",
			path:     "/path/to/src/index.js",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := watcher.shouldIgnore(tt.path)
			if result != tt.expected {
				t.Errorf("Expected shouldIgnore(%s) to be %v, got %v", tt.path, tt.expected, result)
			}
		})
	}
}

func TestFileWatcherIsRelevantFile(t *testing.T) {
	watcher := NewFileWatcher([]string{})

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "JavaScript file",
			filename: "index.js",
			expected: true,
		},
		{
			name:     "TypeScript file",
			filename: "component.ts",
			expected: true,
		},
		{
			name:     "Python file",
			filename: "main.py",
			expected: true,
		},
		{
			name:     "C# file",
			filename: "Program.cs",
			expected: true,
		},
		{
			name:     "Go file",
			filename: "main.go",
			expected: true,
		},
		{
			name:     "JSX file",
			filename: "Component.jsx",
			expected: true,
		},
		{
			name:     "Binary file",
			filename: "file.exe",
			expected: false,
		},
		{
			name:     "Image file",
			filename: "logo.png",
			expected: false,
		},
		{
			name:     "JSON file",
			filename: "package.json",
			expected: false,
		},
		{
			name:     "YAML file",
			filename: "config.yaml",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := watcher.isRelevantFile(tt.filename)
			if result != tt.expected {
				t.Errorf("Expected isRelevantFile(%s) to be %v, got %v", tt.filename, tt.expected, result)
			}
		})
	}
}

func TestFileWatcherCheckForChanges_NoChanges(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.js")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	watcher := NewFileWatcher([]string{tmpDir})

	// First scan
	if err := watcher.scanFiles(); err != nil {
		t.Fatalf("scanFiles failed: %v", err)
	}

	// Check for changes (should be none)
	hasChanges, err := watcher.checkForChanges()
	if err != nil {
		t.Fatalf("checkForChanges failed: %v", err)
	}
	if hasChanges {
		t.Error("Expected no changes on first check")
	}
}

func TestFileWatcherCheckForChanges_WithChanges(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.js")
	if err := os.WriteFile(testFile, []byte("initial content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	watcher := NewFileWatcher([]string{tmpDir})

	// First scan
	if err := watcher.scanFiles(); err != nil {
		t.Fatalf("scanFiles failed: %v", err)
	}

	// Modify the file
	time.Sleep(10 * time.Millisecond) // Ensure different timestamp
	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Check for changes
	hasChanges, err := watcher.checkForChanges()
	if err != nil {
		t.Fatalf("checkForChanges failed: %v", err)
	}
	if !hasChanges {
		t.Error("Expected changes to be detected")
	}
}

func TestFileWatcherCheckForChanges_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile1 := filepath.Join(tmpDir, "test1.js")
	if err := os.WriteFile(testFile1, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	watcher := NewFileWatcher([]string{tmpDir})

	// First scan
	if err := watcher.scanFiles(); err != nil {
		t.Fatalf("scanFiles failed: %v", err)
	}

	// Add a new file
	testFile2 := filepath.Join(tmpDir, "test2.js")
	if err := os.WriteFile(testFile2, []byte("new file"), 0644); err != nil {
		t.Fatalf("Failed to create new test file: %v", err)
	}

	// Check for changes
	hasChanges, err := watcher.checkForChanges()
	if err != nil {
		t.Fatalf("checkForChanges failed: %v", err)
	}
	if !hasChanges {
		t.Error("Expected new file to be detected as change")
	}
}

func TestFileWatcherCheckForChanges_IgnoresIrrelevantFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testFile1 := filepath.Join(tmpDir, "test1.js")
	if err := os.WriteFile(testFile1, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	watcher := NewFileWatcher([]string{tmpDir})

	// First scan
	if err := watcher.scanFiles(); err != nil {
		t.Fatalf("scanFiles failed: %v", err)
	}

	// Add an irrelevant file (PNG image)
	imgFile := filepath.Join(tmpDir, "logo.png")
	if err := os.WriteFile(imgFile, []byte("fake image"), 0644); err != nil {
		t.Fatalf("Failed to create image file: %v", err)
	}

	// Check for changes - should not detect irrelevant file
	hasChanges, err := watcher.checkForChanges()
	if err != nil {
		t.Fatalf("checkForChanges failed: %v", err)
	}
	if hasChanges {
		t.Error("Expected irrelevant file to be ignored")
	}
}
