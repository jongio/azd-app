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

func TestFindNodeProjects(t *testing.T) {
	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "detector-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test structure
	projects := map[string]string{
		"project1/package.json":          `{"name": "test1"}`,
		"project1/pnpm-lock.yaml":        "",
		"project2/package.json":          `{"name": "test2"}`,
		"project2/yarn.lock":             "",
		"project3/package.json":          `{"name": "test3"}`,
		"node_modules/fake/package.json": `{"name": "should-be-ignored"}`,
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
	results, err := FindNodeProjects(tmpDir)
	if err != nil {
		t.Fatalf("FindNodeProjects() error = %v", err)
	}

	// Verify results (should find 3, excluding node_modules)
	if len(results) != 3 {
		t.Errorf("FindNodeProjects() found %d projects, want 3", len(results))
	}

	// Check package managers
	pkgMgrs := make(map[string]int)
	for _, proj := range results {
		pkgMgrs[proj.PackageManager]++
	}

	if pkgMgrs["pnpm"] != 1 || pkgMgrs["yarn"] != 1 || pkgMgrs["npm"] != 1 {
		t.Errorf("Expected 1 pnpm, 1 yarn, 1 npm project, got: %+v", pkgMgrs)
	}
}

func TestHasPackageJson(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "detector-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test without package.json
	if HasPackageJson(tmpDir) {
		t.Error("HasPackageJson() = true, want false when no package.json exists")
	}

	// Create package.json
	packageJsonPath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(packageJsonPath, []byte(`{"name":"test"}`), 0600); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}

	// Test with package.json
	if !HasPackageJson(tmpDir) {
		t.Error("HasPackageJson() = false, want true when package.json exists")
	}
}

func TestDetectPnpmScript(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "has dev script",
			content:  `{"scripts": {"dev": "vite", "build": "vite build"}}`,
			expected: "dev",
		},
		{
			name:     "has start script",
			content:  `{"scripts": {"start": "node index.js", "build": "webpack"}}`,
			expected: "start",
		},
		{
			name:     "has both dev and start - dev wins",
			content:  `{"scripts": {"dev": "vite", "start": "serve"}}`,
			expected: "dev",
		},
		{
			name:     "no dev or start scripts",
			content:  `{"scripts": {"build": "webpack", "test": "jest"}}`,
			expected: "",
		},
		{
			name:     "invalid json",
			content:  `{invalid json}`,
			expected: "",
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

			// Create package.json
			packageJsonPath := filepath.Join(tmpDir, "package.json")
			if err := os.WriteFile(packageJsonPath, []byte(tt.content), 0600); err != nil {
				t.Fatalf("failed to create package.json: %v", err)
			}

			// Test detection
			result := DetectPnpmScript(tmpDir)
			if result != tt.expected {
				t.Errorf("DetectPnpmScript() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHasDockerComposeScript(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "has docker compose up",
			content:  `{"scripts": {"start": "docker compose up"}}`,
			expected: true,
		},
		{
			name:     "has docker-compose up",
			content:  `{"scripts": {"dev": "docker-compose up -d"}}`,
			expected: true,
		},
		{
			name:     "no docker compose",
			content:  `{"scripts": {"start": "node index.js"}}`,
			expected: false,
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

			// Create package.json
			packageJsonPath := filepath.Join(tmpDir, "package.json")
			if err := os.WriteFile(packageJsonPath, []byte(tt.content), 0600); err != nil {
				t.Fatalf("failed to create package.json: %v", err)
			}

			// Test detection
			result := HasDockerComposeScript(tmpDir)
			if result != tt.expected {
				t.Errorf("HasDockerComposeScript() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFindDotnetProjects(t *testing.T) {
	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "detector-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test structure
	files := []string{
		"project1/app.csproj",
		"project2/library.csproj",
		"solution.sln",
		"bin/ignored.csproj", // should be ignored
	}

	for _, file := range files {
		fullPath := filepath.Join(tmpDir, file)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0750); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("<Project></Project>"), 0600); err != nil {
			t.Fatalf("failed to create file %s: %v", file, err)
		}
	}

	// Test detection
	results, err := FindDotnetProjects(tmpDir)
	if err != nil {
		t.Fatalf("FindDotnetProjects() error = %v", err)
	}

	// Verify results (2 csproj + 1 sln, bin excluded)
	if len(results) != 3 {
		t.Errorf("FindDotnetProjects() found %d projects, want 3", len(results))
	}
}

func TestFindAppHost(t *testing.T) {
	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "detector-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test structure
	files := map[string]string{
		"AppHost/AppHost.cs":      "// Aspire AppHost",
		"AppHost/AppHost.csproj":  "<Project></Project>",
		"OtherProject/Program.cs": "// Not Aspire",
		"bin/AppHost.cs":          "// should be ignored",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0750); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
			t.Fatalf("failed to create file %s: %v", path, err)
		}
	}

	// Test detection
	result, err := FindAppHost(tmpDir)
	if err != nil {
		t.Fatalf("FindAppHost() error = %v", err)
	}

	if result == nil {
		t.Fatal("FindAppHost() returned nil, expected Aspire project")
	}

	expectedDir := filepath.Join(tmpDir, "AppHost")
	if result.Dir != expectedDir {
		t.Errorf("FindAppHost() Dir = %q, want %q", result.Dir, expectedDir)
	}

	if result.ProjectFile == "" {
		t.Error("FindAppHost() ProjectFile is empty, expected .csproj path")
	}
}

func TestGetPackageManagerFromPackageJson(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "packageManager field with npm",
			content:  `{"name": "test", "packageManager": "npm@10.5.0"}`,
			expected: "npm",
		},
		{
			name:     "packageManager field with yarn",
			content:  `{"name": "test", "packageManager": "yarn@4.1.0"}`,
			expected: "yarn",
		},
		{
			name:     "packageManager field with pnpm",
			content:  `{"name": "test", "packageManager": "pnpm@8.15.0"}`,
			expected: "pnpm",
		},
		{
			name:     "no packageManager field",
			content:  `{"name": "test", "version": "1.0.0"}`,
			expected: "",
		},
		{
			name:     "empty packageManager field",
			content:  `{"name": "test", "packageManager": ""}`,
			expected: "",
		},
		{
			name:     "unsupported package manager",
			content:  `{"name": "test", "packageManager": "bun@1.0.0"}`,
			expected: "",
		},
		{
			name:     "invalid JSON",
			content:  `{invalid json}`,
			expected: "",
		},
		{
			name:     "packageManager without version",
			content:  `{"name": "test", "packageManager": "pnpm"}`,
			expected: "pnpm",
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

			// Create package.json
			packageJsonPath := filepath.Join(tmpDir, "package.json")
			if err := os.WriteFile(packageJsonPath, []byte(tt.content), 0600); err != nil {
				t.Fatalf("failed to create package.json: %v", err)
			}

			// Test detection
			result := getPackageManagerFromPackageJson(tmpDir)
			if result != tt.expected {
				t.Errorf("getPackageManagerFromPackageJson() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDetectNodePackageManagerWithPackageManagerField(t *testing.T) {
	tests := []struct {
		name        string
		packageJson string
		lockFiles   []string
		expected    string
	}{
		{
			name:        "packageManager field takes priority over lock files",
			packageJson: `{"name": "test", "packageManager": "yarn@4.1.0"}`,
			lockFiles:   []string{"pnpm-lock.yaml", "package-lock.json"},
			expected:    "yarn",
		},
		{
			name:        "fallback to pnpm lock file when no packageManager field",
			packageJson: `{"name": "test"}`,
			lockFiles:   []string{"pnpm-lock.yaml"},
			expected:    "pnpm",
		},
		{
			name:        "fallback to yarn lock file when no packageManager field",
			packageJson: `{"name": "test"}`,
			lockFiles:   []string{"yarn.lock"},
			expected:    "yarn",
		},
		{
			name:        "fallback to npm lock file when no packageManager field",
			packageJson: `{"name": "test"}`,
			lockFiles:   []string{"package-lock.json"},
			expected:    "npm",
		},
		{
			name:        "default to npm when no packageManager field and no lock files",
			packageJson: `{"name": "test"}`,
			lockFiles:   []string{},
			expected:    "npm",
		},
		{
			name:        "packageManager field with pnpm overrides yarn lock",
			packageJson: `{"name": "test", "packageManager": "pnpm@8.15.0"}`,
			lockFiles:   []string{"yarn.lock"},
			expected:    "pnpm",
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

			// Create package.json
			packageJsonPath := filepath.Join(tmpDir, "package.json")
			if err := os.WriteFile(packageJsonPath, []byte(tt.packageJson), 0600); err != nil {
				t.Fatalf("failed to create package.json: %v", err)
			}

			// Create lock files
			for _, lockFile := range tt.lockFiles {
				lockPath := filepath.Join(tmpDir, lockFile)
				if err := os.WriteFile(lockPath, []byte(""), 0600); err != nil {
					t.Fatalf("failed to create lock file %s: %v", lockFile, err)
				}
			}

			// Test detection
			result := DetectNodePackageManagerWithBoundary(tmpDir, "")
			if result != tt.expected {
				t.Errorf("DetectNodePackageManagerWithBoundary() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestDetectNodePackageManager tests the convenience wrapper function
func TestDetectNodePackageManager(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json with packageManager field
	packageJSON := `{"name": "test", "packageManager": "pnpm@8.0.0"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0600); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}

	result := DetectNodePackageManager(tmpDir)
	if result != "pnpm" {
		t.Errorf("DetectNodePackageManager() = %q, want %q", result, "pnpm")
	}
}

// TestFindDockerComposeScript tests finding docker compose scripts in package.json
func TestFindDockerComposeScript(t *testing.T) {
	tests := []struct {
		name        string
		packageJSON string
		expected    string
	}{
		{
			name:        "docker compose up in scripts",
			packageJSON: `{"scripts": {"start": "docker compose up"}}`,
			expected:    "start",
		},
		{
			name:        "docker-compose up in scripts",
			packageJSON: `{"scripts": {"dev": "docker-compose up -d"}}`,
			expected:    "dev",
		},
		{
			name:        "no docker compose script",
			packageJSON: `{"scripts": {"start": "node server.js"}}`,
			expected:    "",
		},
		{
			name:        "no package.json",
			packageJSON: "",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if tt.packageJSON != "" {
				if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(tt.packageJSON), 0600); err != nil {
					t.Fatalf("failed to create package.json: %v", err)
				}
			}

			result := FindDockerComposeScript(tmpDir)
			if result != tt.expected {
				t.Errorf("FindDockerComposeScript() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestFindAzureYaml tests finding azure.yaml in directory tree
func TestFindAzureYaml(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("azure.yaml in same directory", func(t *testing.T) {
		// Create azure.yaml in tmp dir
		azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
		if err := os.WriteFile(azureYamlPath, []byte("name: test"), 0600); err != nil {
			t.Fatalf("failed to create azure.yaml: %v", err)
		}
		defer os.Remove(azureYamlPath)

		result, err := FindAzureYaml(tmpDir)
		if err != nil {
			t.Errorf("FindAzureYaml() error = %v", err)
		}
		if result != azureYamlPath {
			t.Errorf("FindAzureYaml() = %q, want %q", result, azureYamlPath)
		}
	})

	t.Run("azure.yaml in parent directory", func(t *testing.T) {
		// Create subdirectory
		subDir := filepath.Join(tmpDir, "subdir")
		if err := os.MkdirAll(subDir, 0750); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}

		// Create azure.yaml in parent
		azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
		if err := os.WriteFile(azureYamlPath, []byte("name: test"), 0600); err != nil {
			t.Fatalf("failed to create azure.yaml: %v", err)
		}
		defer os.Remove(azureYamlPath)

		result, err := FindAzureYaml(subDir)
		if err != nil {
			t.Errorf("FindAzureYaml() error = %v", err)
		}
		if result != azureYamlPath {
			t.Errorf("FindAzureYaml() = %q, want %q", result, azureYamlPath)
		}
	})
	
	t.Run("no azure.yaml found", func(t *testing.T) {
		// Use a directory without azure.yaml
		noYamlDir := t.TempDir()
		
		result, err := FindAzureYaml(noYamlDir)
		if err != nil {
			t.Errorf("FindAzureYaml() unexpected error: %v", err)
		}
		if result != "" {
			t.Errorf("FindAzureYaml() = %q, want empty string", result)
		}
	})
}
