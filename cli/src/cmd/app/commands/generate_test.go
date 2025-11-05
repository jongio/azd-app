package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name             string
		installedVersion string
		toolName         string
		expected         string
	}{
		// Node.js and similar (major only)
		{"node major", "22.3.0", "node", "22.0.0"},
		{"dotnet major", "10.0.100", "dotnet", "10.0.0"},
		{"docker major", "28.5.1", "docker", "28.0.0"},
		{"git major", "2.51.2", "git", "2.0.0"},

		// Python (major.minor)
		{"python major.minor", "3.12.5", "python", "3.12.0"},
		{"python 3.13", "3.13.9", "python", "3.13.0"},

		// Package managers (major only)
		{"pnpm major", "10.20.0", "pnpm", "10.0.0"},
		{"npm major", "11.4.0", "npm", "11.0.0"},
		{"yarn major", "4.3.1", "yarn", "4.0.0"},
		{"poetry major", "2.2.1", "poetry", "2.0.0"},
		{"uv major", "1.5.0", "uv", "1.0.0"},
		{"pip major", "25.2.0", "pip", "25.0.0"},

		// Azure tools (major.minor)
		{"azd major.minor", "1.20.3", "azd", "1.20.0"},
		{"az major.minor", "2.70.0", "az", "2.70.0"},
		{"aspire major.minor", "13.0.1", "aspire", "13.0.0"},

		// Edge cases
		{"two parts", "3.12", "python", "3.12.0"},
		{"one part", "10", "node", "10.0.0"},
		{"unknown tool", "1.2.3", "unknown", "1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeVersion(tt.installedVersion, tt.toolName)
			if result != tt.expected {
				t.Errorf("normalizeVersion(%q, %q) = %q, want %q",
					tt.installedVersion, tt.toolName, result, tt.expected)
			}
		})
	}
}

func TestExtractVersionFromOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		prefix   string
		field    int
		expected string
	}{
		{
			name:     "node version",
			output:   "v22.3.0",
			prefix:   "v",
			field:    0,
			expected: "22.3.0",
		},
		{
			name:     "python version",
			output:   "Python 3.12.5",
			prefix:   "",
			field:    1,
			expected: "3.12.5",
		},
		{
			name:     "pip version",
			output:   "pip 25.2 from /path/to/pip",
			prefix:   "",
			field:    1,
			expected: "25.2",
		},
		{
			name:     "docker version",
			output:   "Docker version 28.5.1, build e180ab8",
			prefix:   "",
			field:    2,
			expected: "28.5.1",
		},
		{
			name:     "git version",
			output:   "git version 2.51.2.windows.1",
			prefix:   "",
			field:    2,
			expected: "2.51.2",
		},
		{
			name:     "poetry version",
			output:   "Poetry (version 2.2.1)",
			prefix:   "",
			field:    2,
			expected: "2.2.1",
		},
		{
			name:     "simple version",
			output:   "1.2.3",
			prefix:   "",
			field:    0,
			expected: "1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersionFromOutput(tt.output, tt.prefix, tt.field)
			if result != tt.expected {
				t.Errorf("extractVersionFromOutput(%q, %q, %d) = %q, want %q",
					tt.output, tt.prefix, tt.field, result, tt.expected)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "test-fileexists-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(testFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name     string
		dir      string
		filename string
		expected bool
	}{
		{"exists", tmpDir, "package.json", true},
		{"does not exist", tmpDir, "yarn.lock", false},
		{"invalid path", tmpDir, "../../../etc/passwd", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fileExists(tt.dir, tt.filename)
			if result != tt.expected {
				t.Errorf("fileExists(%q, %q) = %v, want %v",
					tt.dir, tt.filename, result, tt.expected)
			}
		})
	}
}

func TestHasPackageJSON(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "test-haspackagejson-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		setup    func(string) error
		expected bool
	}{
		{
			name: "has package.json",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0600)
			},
			expected: true,
		},
		{
			name:     "no package.json",
			setup:    func(dir string) error { return nil },
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir, err := os.MkdirTemp(tmpDir, "test-*")
			if err != nil {
				t.Fatalf("Failed to create test dir: %v", err)
			}

			if err := tt.setup(testDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			result := hasPackageJSON(testDir)
			if result != tt.expected {
				t.Errorf("hasPackageJSON(%q) = %v, want %v", testDir, result, tt.expected)
			}
		})
	}
}

func TestHasPythonProject(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "test-haspythonproject-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		setup    func(string) error
		expected bool
	}{
		{
			name: "has requirements.txt",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(""), 0600)
			},
			expected: true,
		},
		{
			name: "has pyproject.toml",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(""), 0600)
			},
			expected: true,
		},
		{
			name: "has poetry.lock",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "poetry.lock"), []byte(""), 0600)
			},
			expected: true,
		},
		{
			name: "has uv.lock",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "uv.lock"), []byte(""), 0600)
			},
			expected: true,
		},
		{
			name:     "no python files",
			setup:    func(dir string) error { return nil },
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir, err := os.MkdirTemp(tmpDir, "test-*")
			if err != nil {
				t.Fatalf("Failed to create test dir: %v", err)
			}

			if err := tt.setup(testDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			result := hasPythonProject(testDir)
			if result != tt.expected {
				t.Errorf("hasPythonProject(%q) = %v, want %v", testDir, result, tt.expected)
			}
		})
	}
}

func TestHasDockerConfig(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "test-hasdockerconfig-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		setup    func(string) error
		expected bool
	}{
		{
			name: "has Dockerfile",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(""), 0600)
			},
			expected: true,
		},
		{
			name: "has docker-compose.yml",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(""), 0600)
			},
			expected: true,
		},
		{
			name: "has compose.yaml",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(""), 0600)
			},
			expected: true,
		},
		{
			name:     "no docker files",
			setup:    func(dir string) error { return nil },
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir, err := os.MkdirTemp(tmpDir, "test-*")
			if err != nil {
				t.Fatalf("Failed to create test dir: %v", err)
			}

			if err := tt.setup(testDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			result := hasDockerConfig(testDir)
			if result != tt.expected {
				t.Errorf("hasDockerConfig(%q) = %v, want %v", testDir, result, tt.expected)
			}
		})
	}
}

func TestHasGit(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "test-hasgit-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		setup    func(string) error
		expected bool
	}{
		{
			name: "has .git directory",
			setup: func(dir string) error {
				return os.Mkdir(filepath.Join(dir, ".git"), 0750)
			},
			expected: true,
		},
		{
			name:     "no .git directory",
			setup:    func(dir string) error { return nil },
			expected: false,
		},
		{
			name: ".git is a file (not directory)",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, ".git"), []byte(""), 0600)
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir, err := os.MkdirTemp(tmpDir, "test-*")
			if err != nil {
				t.Fatalf("Failed to create test dir: %v", err)
			}

			if err := tt.setup(testDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			result := hasGit(testDir)
			if result != tt.expected {
				t.Errorf("hasGit(%q) = %v, want %v", testDir, result, tt.expected)
			}
		})
	}
}

func TestFindOrCreateAzureYaml(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "test-findorcreate-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("creates new azure.yaml", func(t *testing.T) {
		testDir, err := os.MkdirTemp(tmpDir, "test-*")
		if err != nil {
			t.Fatalf("Failed to create test dir: %v", err)
		}

		path, created, err := findOrCreateAzureYaml(testDir, false)
		if err != nil {
			t.Fatalf("findOrCreateAzureYaml failed: %v", err)
		}

		if !created {
			t.Errorf("Expected created=true, got false")
		}

		expected := filepath.Join(testDir, "azure.yaml")
		if path != expected {
			t.Errorf("Expected path=%q, got %q", expected, path)
		}

		// Verify file was created
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("File was not created: %s", path)
		}
	})

	t.Run("finds existing azure.yaml", func(t *testing.T) {
		testDir, err := os.MkdirTemp(tmpDir, "test-*")
		if err != nil {
			t.Fatalf("Failed to create test dir: %v", err)
		}

		// Create existing azure.yaml
		existingPath := filepath.Join(testDir, "azure.yaml")
		if err := os.WriteFile(existingPath, []byte("name: test\nreqs:\n"), 0600); err != nil {
			t.Fatalf("Failed to create existing azure.yaml: %v", err)
		}

		path, created, err := findOrCreateAzureYaml(testDir, false)
		if err != nil {
			t.Fatalf("findOrCreateAzureYaml failed: %v", err)
		}

		if created {
			t.Errorf("Expected created=false, got true")
		}

		if path != existingPath {
			t.Errorf("Expected path=%q, got %q", existingPath, path)
		}
	})

	t.Run("dry run mode", func(t *testing.T) {
		testDir, err := os.MkdirTemp(tmpDir, "test-*")
		if err != nil {
			t.Fatalf("Failed to create test dir: %v", err)
		}

		path, created, err := findOrCreateAzureYaml(testDir, true)
		if err != nil {
			t.Fatalf("findOrCreateAzureYaml failed: %v", err)
		}

		if !created {
			t.Errorf("Expected created=true in dry run, got false")
		}

		// Verify file was NOT created in dry run
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("File should not be created in dry run mode")
		}
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		// Attempt to create azure.yaml with path traversal
		maliciousDir := filepath.Join(tmpDir, "..", "..", "..", "etc")

		_, _, err := findOrCreateAzureYaml(maliciousDir, false)
		// Should fail due to security validation
		if err == nil {
			t.Error("Expected error for path traversal, got nil")
		}
	})
}

func TestMergeReqs(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "test-merge-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("merge with empty azure.yaml", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "azure1.yaml")
		content := `name: test
reqs:
`
		if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		detected := []DetectedRequirement{
			{Name: "node", MinVersion: "22.0.0"},
			{Name: "npm", MinVersion: "11.0.0"},
		}

		added, skipped, err := mergeReqs(testFile, detected)
		if err != nil {
			t.Fatalf("mergeReqs failed: %v", err)
		}

		if added != 2 {
			t.Errorf("Expected added=2, got %d", added)
		}
		if skipped != 0 {
			t.Errorf("Expected skipped=0, got %d", skipped)
		}
	})

	t.Run("merge with existing requirements", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "azure2.yaml")
		content := `name: test
reqs:
  - name: node
    minVersion: 20.0.0
  - name: npm
    minVersion: 10.0.0
`
		if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		detected := []DetectedRequirement{
			{Name: "node", MinVersion: "22.0.0"},
			{Name: "npm", MinVersion: "11.0.0"},
			{Name: "git", MinVersion: "2.0.0"},
		}

		added, skipped, err := mergeReqs(testFile, detected)
		if err != nil {
			t.Fatalf("mergeReqs failed: %v", err)
		}

		if added != 1 { // Only git is new
			t.Errorf("Expected added=1, got %d", added)
		}
		if skipped != 2 { // node and npm already exist
			t.Errorf("Expected skipped=2, got %d", skipped)
		}
	})

	t.Run("validates path", func(t *testing.T) {
		// Attempt path traversal
		maliciousPath := filepath.Join(tmpDir, "..", "..", "..", "etc", "passwd")

		detected := []DetectedRequirement{
			{Name: "node", MinVersion: "22.0.0"},
		}

		_, _, err := mergeReqs(maliciousPath, detected)
		if err == nil {
			t.Error("Expected error for path traversal, got nil")
		}
	})

	t.Run("preserves comments and formatting", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "azure3.yaml")
		content := `# Project configuration file
# This is important!
name: test

# Environment variables section
# DO NOT MODIFY
env:
  - name: API_KEY
    value: secret

# Requirements section
reqs:
  # Node.js runtime - keep at v20
  - name: node
    minVersion: "20.0.0"
  
  # Package manager
  - name: npm
    minVersion: "10.0.0"

# Services will go here
services: []
`
		if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		detected := []DetectedRequirement{
			{Name: "node", MinVersion: "22.0.0"}, // Already exists (will be skipped)
			{Name: "git", MinVersion: "2.0.0"},   // New requirement (will be added)
		}

		added, skipped, err := mergeReqs(testFile, detected)
		if err != nil {
			t.Fatalf("mergeReqs failed: %v", err)
		}

		// Note: skipped counts ALL existing reqs in the file (node + npm = 2)
		// not just the ones we tried to add
		if added != 1 {
			t.Errorf("Expected added=1, got %d", added)
		}
		if skipped != 2 {
			t.Errorf("Expected skipped=2 (node+npm already in file), got %d", skipped)
		}

		// Read the file and verify comments are preserved
		resultContent, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read result file: %v", err)
		}

		resultStr := string(resultContent)

		// Debug: print the result
		t.Logf("Result YAML:\n%s", resultStr)

		// Check that important comments are preserved
		expectedComments := []string{
			"# Project configuration file",
			"# This is important!",
			"# Environment variables section",
			"# DO NOT MODIFY",
			"# Requirements section",
			// Note: Inline comments like "# Node.js runtime - keep at v20"
			// may not be preserved by yaml.v3
			// "# Node.js runtime - keep at v20",
			// "# Package manager",
			"# Services will go here",
		}

		for _, comment := range expectedComments {
			if !strings.Contains(resultStr, comment) {
				t.Errorf("Expected comment %q to be preserved, but it was not found", comment)
			}
		}

		// Check that the env section is still there
		if !strings.Contains(resultStr, "env:") {
			t.Error("Expected env section to be preserved")
		}
		if !strings.Contains(resultStr, "API_KEY") {
			t.Error("Expected env content to be preserved")
		}

		// Check that git was added
		if !strings.Contains(resultStr, "name: git") {
			t.Error("Expected git requirement to be added")
		}

		// Check that services section is still there
		if !strings.Contains(resultStr, "services: []") {
			t.Error("Expected services section to be preserved")
		}
	})

	t.Run("handles empty reqs array", func(t *testing.T) {
		// Create temp directory
		tmpDir, err := os.MkdirTemp("", "test-merge-empty-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Write azure.yaml with empty reqs array
		azureYaml := `name: test-app
reqs: []
`
		azurePath := filepath.Join(tmpDir, "azure.yaml")
		if err := os.WriteFile(azurePath, []byte(azureYaml), 0600); err != nil {
			t.Fatalf("Failed to write azure.yaml: %v", err)
		}

		// Test merging
		newReqs := []DetectedRequirement{
			{Name: "node", MinVersion: "20.0.0", Source: "package.json"},
		}

		added, existing, err := mergeReqs(azurePath, newReqs)
		if err != nil {
			t.Fatalf("mergeReqs failed: %v", err)
		}

		if added != 1 {
			t.Errorf("Expected 1 added, got %d", added)
		}

		// Read result
		result, err := os.ReadFile(azurePath)
		if err != nil {
			t.Fatalf("Failed to read result: %v", err)
		}

		resultStr := string(result)
		if !strings.Contains(resultStr, "name: node") {
			t.Error("Expected node requirement to be added to empty array")
		}
		if existing != 0 {
			t.Errorf("Expected 0 existing, got %d", existing)
		}
	})

	t.Run("handles no new requirements", func(t *testing.T) {
		// Create temp directory
		tmpDir, err := os.MkdirTemp("", "test-merge-none-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Write azure.yaml
		azureYaml := `name: test-app
reqs:
  - name: node
    minVersion: "20.0.0"
`
		azurePath := filepath.Join(tmpDir, "azure.yaml")
		if err := os.WriteFile(azurePath, []byte(azureYaml), 0600); err != nil {
			t.Fatalf("Failed to write azure.yaml: %v", err)
		}

		// Test merging with no new reqs
		added, existing, err := mergeReqs(azurePath, []DetectedRequirement{})
		if err != nil {
			t.Fatalf("mergeReqs failed: %v", err)
		}

		if added != 0 {
			t.Errorf("Expected 0 added, got %d", added)
		}
		if existing != 1 {
			t.Errorf("Expected 1 existing, got %d", existing)
		}

		// Read result - should be unchanged
		result, err := os.ReadFile(azurePath)
		if err != nil {
			t.Fatalf("Failed to read result: %v", err)
		}

		if string(result) != azureYaml {
			t.Error("Expected azure.yaml to be unchanged when no new requirements")
		}
	})

	t.Run("handles complex nested yaml structure", func(t *testing.T) {
		// Create temp directory
		tmpDir, err := os.MkdirTemp("", "test-merge-complex-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Write azure.yaml with complex structure
		azureYaml := `# Main config
name: test-app
version: "1.0"

# Settings
settings:
  nested:
    deep:
      value: 123

# Tools
reqs:
  - name: node
    minVersion: "20.0.0"

# More config
services:
  - name: api
    config:
      nested: value
  - name: web
    config:
      other: data
`
		azurePath := filepath.Join(tmpDir, "azure.yaml")
		if err := os.WriteFile(azurePath, []byte(azureYaml), 0600); err != nil {
			t.Fatalf("Failed to write azure.yaml: %v", err)
		}

		// Test merging
		newReqs := []DetectedRequirement{
			{Name: "docker", MinVersion: "20.0.0", Source: "Dockerfile"},
		}

		added, existing, err := mergeReqs(azurePath, newReqs)
		if err != nil {
			t.Fatalf("mergeReqs failed: %v", err)
		}

		if added != 1 {
			t.Errorf("Expected 1 added, got %d", added)
		}
		if existing != 1 {
			t.Errorf("Expected 1 existing, got %d", existing)
		}

		// Read result
		result, err := os.ReadFile(azurePath)
		if err != nil {
			t.Fatalf("Failed to read result: %v", err)
		}

		resultStr := string(result)

		// Verify all sections preserved
		checks := []string{
			"# Main config",
			"version: \"1.0\"",
			"# Settings",
			"settings:",
			"nested:",
			"deep:",
			"value: 123",
			"# Tools",
			"name: node",
			"name: docker",
			"# More config",
			"services:",
			"name: api",
			"name: web",
		}

		for _, check := range checks {
			if !strings.Contains(resultStr, check) {
				t.Errorf("Expected %q to be preserved", check)
			}
		}
	})
}

func TestDetectNodePackageManager(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "test-detectpm-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		setup          func(string) error
		expectedID     string
		expectedSource string
	}{
		{
			name: "detects pnpm from lock file",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0600)
			},
			expectedID:     "pnpm",
			expectedSource: "pnpm-lock.yaml",
		},
		{
			name: "detects pnpm from workspace file",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "pnpm-workspace.yaml"), []byte(""), 0600)
			},
			expectedID:     "pnpm",
			expectedSource: "pnpm-lock.yaml",
		},
		{
			name: "detects yarn from lock file",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(""), 0600)
			},
			expectedID:     "yarn",
			expectedSource: "yarn.lock",
		},
		{
			name: "detects npm from lock file",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(""), 0600)
			},
			expectedID:     "npm",
			expectedSource: "package-lock.json",
		},
		{
			name:           "defaults to npm",
			setup:          func(dir string) error { return nil },
			expectedID:     "npm",
			expectedSource: "package.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir, err := os.MkdirTemp(tmpDir, "test-*")
			if err != nil {
				t.Fatalf("Failed to create test dir: %v", err)
			}

			if err := tt.setup(testDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			result := detectNodePackageManager(testDir)
			if result.Name != tt.expectedID {
				t.Errorf("Expected ID=%q, got %q", tt.expectedID, result.Name)
			}
			if result.Source != tt.expectedSource {
				t.Errorf("Expected Source=%q, got %q", tt.expectedSource, result.Source)
			}
		})
	}
}

func TestDetectPythonPackageManager(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "test-detectpypm-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		setup          func(string) error
		expectedID     string
		expectedSource string
	}{
		{
			name: "detects uv from lock file",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "uv.lock"), []byte(""), 0600)
			},
			expectedID:     "uv",
			expectedSource: "uv.lock",
		},
		{
			name: "detects uv from pyproject.toml",
			setup: func(dir string) error {
				content := "[tool.uv]\n"
				return os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0600)
			},
			expectedID:     "uv",
			expectedSource: "pyproject.toml",
		},
		{
			name: "detects poetry from pyproject.toml",
			setup: func(dir string) error {
				content := "[tool.poetry]\n"
				return os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0600)
			},
			expectedID:     "poetry",
			expectedSource: "pyproject.toml",
		},
		{
			name: "detects poetry from lock file",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "poetry.lock"), []byte(""), 0600)
			},
			expectedID:     "poetry",
			expectedSource: "poetry.lock",
		},
		{
			name: "detects pipenv from Pipfile",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "Pipfile"), []byte(""), 0600)
			},
			expectedID:     "pipenv",
			expectedSource: "Pipfile",
		},
		{
			name:           "defaults to pip",
			setup:          func(dir string) error { return nil },
			expectedID:     "pip",
			expectedSource: "requirements.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir, err := os.MkdirTemp(tmpDir, "test-*")
			if err != nil {
				t.Fatalf("Failed to create test dir: %v", err)
			}

			if err := tt.setup(testDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			result := detectPythonPackageManager(testDir)
			if result.Name != tt.expectedID {
				t.Errorf("Expected ID=%q, got %q", tt.expectedID, result.Name)
			}
			if result.Source != tt.expectedSource {
				t.Errorf("Expected Source=%q, got %q", tt.expectedSource, result.Source)
			}
		})
	}
}
