package fileutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
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

func TestAtomicWriteJSON(t *testing.T) {
	tmpDir := t.TempDir()

	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name    string
		path    string
		data    interface{}
		wantErr bool
	}{
		{
			name:    "valid json write",
			path:    filepath.Join(tmpDir, "test.json"),
			data:    TestData{Name: "test", Value: 42},
			wantErr: false,
		},
		{
			name:    "overwrite existing file",
			path:    filepath.Join(tmpDir, "overwrite.json"),
			data:    TestData{Name: "updated", Value: 100},
			wantErr: false,
		},
		{
			name:    "nested directory",
			path:    filepath.Join(tmpDir, "nested", "test.json"),
			data:    TestData{Name: "nested", Value: 1},
			wantErr: true, // Should fail because directory doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AtomicWriteJSON(tt.path, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("AtomicWriteJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify file exists and contains expected data
				var result TestData
				data, err := os.ReadFile(tt.path)
				if err != nil {
					t.Errorf("Failed to read written file: %v", err)
					return
				}
				if err := json.Unmarshal(data, &result); err != nil {
					t.Errorf("Failed to unmarshal written JSON: %v", err)
					return
				}
				expected := tt.data.(TestData)
				if result.Name != expected.Name || result.Value != expected.Value {
					t.Errorf("AtomicWriteJSON() wrote %+v, want %+v", result, expected)
				}

				// Verify temp file was cleaned up
				tmpPath := tt.path + ".tmp"
				if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
					t.Errorf("Temp file still exists: %s", tmpPath)
				}
			}
		})
	}
}

func TestAtomicWriteJSON_UnmarshalableData(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	// Create data that can't be marshaled (channel type)
	type BadData struct {
		Ch chan int
	}

	err := AtomicWriteJSON(path, BadData{Ch: make(chan int)})
	if err == nil {
		t.Error("AtomicWriteJSON() expected error for unmarshalable data, got nil")
	}
}

func TestAtomicWriteFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		data    []byte
		perm    os.FileMode
		wantErr bool
	}{
		{
			name:    "simple write",
			path:    filepath.Join(tmpDir, "simple.txt"),
			data:    []byte("Hello, World!"),
			perm:    0644,
			wantErr: false,
		},
		{
			name:    "binary data",
			path:    filepath.Join(tmpDir, "binary.dat"),
			data:    []byte{0x00, 0xFF, 0xAB, 0xCD},
			perm:    0600,
			wantErr: false,
		},
		{
			name:    "empty file",
			path:    filepath.Join(tmpDir, "empty.txt"),
			data:    []byte{},
			perm:    0644,
			wantErr: false,
		},
		{
			name:    "overwrite existing",
			path:    filepath.Join(tmpDir, "overwrite.txt"),
			data:    []byte("new content"),
			perm:    0644,
			wantErr: false,
		},
		{
			name:    "invalid directory",
			path:    filepath.Join(tmpDir, "nonexistent", "file.txt"),
			data:    []byte("data"),
			perm:    0644,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AtomicWriteFile(tt.path, tt.data, tt.perm)
			if (err != nil) != tt.wantErr {
				t.Errorf("AtomicWriteFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify file exists and contains expected data
				data, err := os.ReadFile(tt.path)
				if err != nil {
					t.Errorf("Failed to read written file: %v", err)
					return
				}
				if string(data) != string(tt.data) {
					t.Errorf("AtomicWriteFile() wrote %q, want %q", data, tt.data)
				}

				// Verify temp file was cleaned up
				tmpPath := tt.path + ".tmp"
				if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
					t.Errorf("Temp file still exists: %s", tmpPath)
				}
			}
		})
	}
}

func TestReadJSON(t *testing.T) {
	tmpDir := t.TempDir()

	type TestConfig struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
		Count   int    `json:"count"`
	}

	validJSON := `{"name": "test-config", "enabled": true, "count": 5}`
	invalidJSON := `{"name": "incomplete"`

	validFile := filepath.Join(tmpDir, "valid.json")
	if err := os.WriteFile(validFile, []byte(validJSON), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	invalidFile := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidFile, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		want    *TestConfig
		wantErr bool
	}{
		{
			name:    "valid json file",
			path:    validFile,
			want:    &TestConfig{Name: "test-config", Enabled: true, Count: 5},
			wantErr: false,
		},
		{
			name:    "invalid json file",
			path:    invalidFile,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "non-existent file",
			path:    filepath.Join(tmpDir, "missing.json"),
			want:    nil,
			wantErr: false, // ReadJSON returns nil for missing files
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result TestConfig
			err := ReadJSON(tt.path, &result)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want != nil {
				if result.Name != tt.want.Name || result.Enabled != tt.want.Enabled || result.Count != tt.want.Count {
					t.Errorf("ReadJSON() got %+v, want %+v", result, tt.want)
				}
			}
		})
	}
}

func TestReadJSON_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.json")
	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var result map[string]interface{}
	err := ReadJSON(emptyFile, &result)
	if err == nil {
		t.Error("ReadJSON() expected error for empty file, got nil")
	}
}

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "create new directory",
			path:    filepath.Join(tmpDir, "newdir"),
			wantErr: false,
		},
		{
			name:    "create nested directories",
			path:    filepath.Join(tmpDir, "nested", "deep", "path"),
			wantErr: false,
		},
		{
			name:    "existing directory",
			path:    tmpDir, // Already exists
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnsureDir(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify directory exists
				info, err := os.Stat(tt.path)
				if err != nil {
					t.Errorf("Directory doesn't exist after EnsureDir(): %v", err)
					return
				}
				if !info.IsDir() {
					t.Errorf("Path exists but is not a directory: %s", tt.path)
				}
			}
		})
	}
}

func TestAtomicWrite_Concurrency(t *testing.T) {
	// Skip on Windows due to file locking constraints that make this test flaky
	// Windows doesn't allow renaming files that are locked by another process
	if runtime.GOOS == "windows" {
		t.Skip("Skipping concurrent atomic write test on Windows due to file locking behavior")
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "concurrent.txt")

	// Test concurrent writes don't corrupt the file
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			data := []byte("data-" + string(rune('0'+n)))
			err := AtomicWriteFile(path, data, 0644)
			if err != nil {
				t.Logf("AtomicWriteFile error for goroutine %d: %v", n, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Give a brief moment for any final filesystem operations to settle
	// (especially important on Windows where file operations can be delayed)
	time.Sleep(100 * time.Millisecond)

	// Verify file exists and is not corrupted
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Failed to read file after concurrent writes: %v", err)
		return
	}
	if len(data) == 0 {
		t.Error("File is empty after concurrent writes")
	}
}

func TestContainsText_InvalidPath(t *testing.T) {
	// Test with path traversal attempt
	result := ContainsText("../../../etc/passwd", "root")
	if result {
		t.Error("ContainsText() should return false for invalid paths")
	}
}

func TestFileExists_EdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		setup    func() string
		filename string
		want     bool
	}{
		{
			name: "directory not file",
			setup: func() string {
				subdir := filepath.Join(tmpDir, "subdir")
				_ = os.Mkdir(subdir, 0755)
				return tmpDir
			},
			filename: "subdir",
			want:     true, // Directories also return true for stat
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup()
			if got := FileExists(dir, tt.filename); got != tt.want {
				t.Errorf("FileExists() = %v, want %v", got, tt.want)
			}
		})
	}
}
