package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPythonPackageManager(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected string
	}{
		{
			name: "uv lock file",
			files: map[string]string{
				"uv.lock": "",
			},
			expected: "uv",
		},
		{
			name: "poetry lock file",
			files: map[string]string{
				"poetry.lock": "",
			},
			expected: "poetry",
		},
		{
			name: "pyproject.toml with poetry",
			files: map[string]string{
				"pyproject.toml": "[tool.poetry]\nname = \"test\"",
			},
			expected: "poetry",
		},
		{
			name: "pyproject.toml with uv",
			files: map[string]string{
				"pyproject.toml": "[tool.uv]\nname = \"test\"",
			},
			expected: "uv",
		},
		{
			name: "requirements.txt only",
			files: map[string]string{
				"requirements.txt": "requests==2.28.0",
			},
			expected: "pip",
		},
		{
			name:     "no files",
			files:    map[string]string{},
			expected: "pip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "detector-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			// Create test files
			for filename, content := range tt.files {
				path := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(path, []byte(content), 0600); err != nil {
					t.Fatalf("failed to create test file %s: %v", filename, err)
				}
			}

			// Test detection
			result := DetectPythonPackageManager(tmpDir)
			if result != tt.expected {
				t.Errorf("DetectPythonPackageManager() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFindPythonProjects(t *testing.T) {
	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "detector-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test structure
	projects := map[string]string{
		"project1/requirements.txt":  "requests==2.28.0",
		"project2/pyproject.toml":    "[tool.poetry]\nname = \"test\"",
		"project2/poetry.lock":       "",
		"project3/pyproject.toml":    "[tool.uv]\nname = \"test\"",
		"project3/uv.lock":           "",
		"node_modules/fake/setup.py": "# should be ignored",
	}

	for path, content := range projects {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0750); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
			t.Fatalf("failed to create file %s: %v", path, err)
		}
	}

	// Test detection
	results, err := FindPythonProjects(tmpDir)
	if err != nil {
		t.Fatalf("FindPythonProjects() error = %v", err)
	}

	// Verify results
	if len(results) != 3 {
		t.Errorf("FindPythonProjects() found %d projects, want 3", len(results))
	}

	// Check package managers
	pkgMgrs := make(map[string]bool)
	for _, proj := range results {
		pkgMgrs[proj.PackageManager] = true
	}

	if !pkgMgrs["pip"] || !pkgMgrs["poetry"] || !pkgMgrs["uv"] {
		t.Errorf("Expected to find pip, poetry, and uv projects, got: %+v", results)
	}
}
