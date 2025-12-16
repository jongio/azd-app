package detector

import (
	"os"
	"path/filepath"
	"testing"
)

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
	packageJSONPath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(packageJSONPath, []byte(`{"name":"test"}`), 0600); err != nil {
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

func TestGetPackageManagerFromPackageJSON(t *testing.T) {
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
			result := GetPackageManagerFromPackageJSON(tmpDir)
			if result != tt.expected {
				t.Errorf("GetPackageManagerFromPackageJSON() = %q, want %q", result, tt.expected)
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
			packageJSONPath := filepath.Join(tmpDir, "package.json")
			if err := os.WriteFile(packageJSONPath, []byte(tt.packageJson), 0600); err != nil {
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
