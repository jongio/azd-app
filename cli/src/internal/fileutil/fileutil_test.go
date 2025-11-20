package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	testFile := "test.txt"
	if err := os.WriteFile(filepath.Join(tmpDir, testFile), []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"existing file", testFile, true},
		{"non-existing file", "missing.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileExists(tmpDir, tt.filename); got != tt.want {
				t.Errorf("FileExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasFileWithExt(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := []string{"test.csproj", "another.txt"}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, file), []byte("test"), 0600); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	tests := []struct {
		name string
		ext  string
		want bool
	}{
		{"existing extension", ".csproj", true},
		{"non-existing extension", ".fsproj", false},
		{"existing txt", ".txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasFileWithExt(tmpDir, tt.ext); got != tt.want {
				t.Errorf("HasFileWithExt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsText(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "This is a test file with specific content"
	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name     string
		filePath string
		text     string
		want     bool
	}{
		{"contains text", testFile, "specific content", true},
		{"does not contain", testFile, "missing text", false},
		{"non-existing file", filepath.Join(tmpDir, "missing.txt"), "any", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsText(tt.filePath, tt.text); got != tt.want {
				t.Errorf("ContainsText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileExistsAny(t *testing.T) {
	tmpDir := t.TempDir()

	// Create one file
	if err := os.WriteFile(filepath.Join(tmpDir, "exists.txt"), []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name      string
		filenames []string
		want      bool
	}{
		{"one exists", []string{"exists.txt", "missing.txt"}, true},
		{"none exist", []string{"missing1.txt", "missing2.txt"}, false},
		{"all exist", []string{"exists.txt"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileExistsAny(tmpDir, tt.filenames...); got != tt.want {
				t.Errorf("FileExistsAny() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilesExistAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files
	files := []string{"file1.txt", "file2.txt"}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, file), []byte("test"), 0600); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	tests := []struct {
		name      string
		filenames []string
		want      bool
	}{
		{"all exist", []string{"file1.txt", "file2.txt"}, true},
		{"one missing", []string{"file1.txt", "missing.txt"}, false},
		{"none exist", []string{"missing1.txt", "missing2.txt"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FilesExistAll(tmpDir, tt.filenames...); got != tt.want {
				t.Errorf("FilesExistAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsTextInFile(t *testing.T) {
	tmpDir := t.TempDir()

	filename := "config.json"
	content := `{"name": "test-package"}`
	if err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name     string
		filename string
		text     string
		want     bool
	}{
		{"contains json key", filename, "test-package", true},
		{"does not contain", filename, "missing-value", false},
		{"missing file", "missing.json", "anything", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsTextInFile(tmpDir, tt.filename, tt.text); got != tt.want {
				t.Errorf("ContainsTextInFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasAnyFileWithExts(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	if err := os.WriteFile(filepath.Join(tmpDir, "test.csproj"), []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name string
		exts []string
		want bool
	}{
		{"one extension matches", []string{".csproj", ".fsproj"}, true},
		{"no extensions match", []string{".fsproj", ".vbproj"}, false},
		{"exact match", []string{".csproj"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasAnyFileWithExts(tmpDir, tt.exts...); got != tt.want {
				t.Errorf("HasAnyFileWithExts() = %v, want %v", got, tt.want)
			}
		})
	}
}
