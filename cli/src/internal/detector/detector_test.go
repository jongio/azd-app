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

func TestFindFunctionApps(t *testing.T) {
	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "detector-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test structure for various Azure Functions projects
	projects := map[string]string{
		// Logic Apps Standard
		"logicapp1/host.json": `{
			"version": "2.0",
			"extensionBundle": {
				"id": "Microsoft.Azure.Functions.ExtensionBundle.Workflows",
				"version": "[1.*, 2.0.0)"
			}
		}`,
		"logicapp1/workflows/workflow1/workflow.json": `{"definition": {}}`,

		// Node.js Functions v4 (TypeScript)
		"nodejs-ts/host.json":     `{"version": "2.0"}`,
		"nodejs-ts/package.json":  `{"dependencies": {"@azure/functions": "^4.0.0"}}`,
		"nodejs-ts/tsconfig.json": `{"compilerOptions": {}}`,
		"nodejs-ts/src/index.ts":  `import { app } from "@azure/functions";`,

		// Node.js Functions v3 (JavaScript)
		"nodejs-js/host.json":                 `{"version": "2.0"}`,
		"nodejs-js/package.json":              `{"dependencies": {"express": "^4.18.0"}}`,
		"nodejs-js/HttpTrigger/function.json": `{"bindings": []}`,

		// Python Functions v2
		"python-v2/host.json":       `{"version": "2.0"}`,
		"python-v2/function_app.py": `import azure.functions as func`,

		// Python Functions v1
		"python-v1/host.json":                 `{"version": "2.0"}`,
		"python-v1/requirements.txt":          `azure-functions==1.11.0`,
		"python-v1/HttpTrigger/function.json": `{"bindings": []}`,

		// .NET Isolated Worker
		"dotnet-isolated/host.json":          `{"version": "2.0"}`,
		"dotnet-isolated/FunctionApp.csproj": `<Project Sdk="Microsoft.NET.Sdk"><ItemGroup><PackageReference Include="Microsoft.Azure.Functions.Worker" /></ItemGroup></Project>`,

		// .NET In-Process
		"dotnet-inprocess/host.json":          `{"version": "2.0"}`,
		"dotnet-inprocess/FunctionApp.csproj": `<Project Sdk="Microsoft.NET.Sdk.Functions"></Project>`,

		// Java Maven
		"java-maven/host.json": `{"version": "2.0"}`,
		"java-maven/pom.xml":   `<project><build><plugins><plugin><groupId>com.microsoft.azure</groupId><artifactId>azure-functions-maven-plugin</artifactId></plugin></plugins></build></project>`,

		// Java Gradle
		"java-gradle/host.json":    `{"version": "2.0"}`,
		"java-gradle/build.gradle": `plugins { id 'com.microsoft.azure.azurefunctions' version '1.7.0' }`,

		// Invalid: host.json but no function definitions
		"invalid/host.json": `{"version": "2.0"}`,

		// Should be ignored (in node_modules)
		"node_modules/fake-func/host.json": `{"version": "2.0"}`,
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
	results, err := FindFunctionApps(tmpDir)
	if err != nil {
		t.Fatalf("FindFunctionApps() error = %v", err)
	}

	// Verify results (should find 9 valid projects, excluding invalid and node_modules)
	if len(results) != 9 {
		t.Errorf("FindFunctionApps() found %d projects, want 9", len(results))
		for _, r := range results {
			t.Logf("Found: %s (variant: %s, language: %s)", r.Dir, r.Variant, r.Language)
		}
	}

	// Count variants
	variants := make(map[string]int)
	languages := make(map[string]int)
	for _, proj := range results {
		variants[proj.Variant]++
		languages[proj.Language]++
	}

	// Verify variant counts
	expectedVariants := map[string]int{
		"logicapps": 1,
		"nodejs":    2,
		"python":    2,
		"dotnet":    2,
		"java":      2,
	}

	for variant, expectedCount := range expectedVariants {
		if variants[variant] != expectedCount {
			t.Errorf("Expected %d %s projects, got %d", expectedCount, variant, variants[variant])
		}
	}

	// Verify language detection
	expectedLanguages := map[string]int{
		"Logic Apps": 1,
		"TypeScript": 1,
		"JavaScript": 1,
		"Python":     2,
		"C#":         2,
		"Java":       2,
	}

	for language, expectedCount := range expectedLanguages {
		if languages[language] != expectedCount {
			t.Errorf("Expected %d %s projects, got %d", expectedCount, language, languages[language])
		}
	}
}

func TestDetectFunctionsVariantForDiscovery(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected string
	}{
		{
			name: "Logic Apps with workflows",
			files: map[string]string{
				"host.json":                         `{"version": "2.0"}`,
				"workflows/workflow1/workflow.json": `{"definition": {}}`,
			},
			expected: "logicapps",
		},
		{
			name: "Logic Apps with extension bundle",
			files: map[string]string{
				"host.json": `{
					"version": "2.0",
					"extensionBundle": {
						"id": "Microsoft.Azure.Functions.ExtensionBundle.Workflows",
						"version": "[1.*, 2.0.0)"
					}
				}`,
			},
			expected: "logicapps",
		},
		{
			name: "Node.js v4 (with @azure/functions)",
			files: map[string]string{
				"host.json":    `{"version": "2.0"}`,
				"package.json": `{"dependencies": {"@azure/functions": "^4.0.0"}}`,
			},
			expected: "nodejs",
		},
		{
			name: "Node.js v3 (with function.json)",
			files: map[string]string{
				"host.json":                 `{"version": "2.0"}`,
				"package.json":              `{"name": "test"}`,
				"HttpTrigger/function.json": `{"bindings": []}`,
			},
			expected: "nodejs",
		},
		{
			name: "Python v2 (function_app.py)",
			files: map[string]string{
				"host.json":       `{"version": "2.0"}`,
				"function_app.py": `import azure.functions`,
			},
			expected: "python",
		},
		{
			name: "Python v1 (requirements.txt + function.json)",
			files: map[string]string{
				"host.json":                 `{"version": "2.0"}`,
				"requirements.txt":          `azure-functions`,
				"HttpTrigger/function.json": `{"bindings": []}`,
			},
			expected: "python",
		},
		{
			name: ".NET Isolated Worker",
			files: map[string]string{
				"host.json":       `{"version": "2.0"}`,
				"Function.csproj": `<Project><ItemGroup><PackageReference Include="Microsoft.Azure.Functions.Worker" /></ItemGroup></Project>`,
			},
			expected: "dotnet",
		},
		{
			name: ".NET In-Process",
			files: map[string]string{
				"host.json":       `{"version": "2.0"}`,
				"Function.csproj": `<Project Sdk="Microsoft.NET.Sdk.Functions"></Project>`,
			},
			expected: "dotnet",
		},
		{
			name: "Java Maven",
			files: map[string]string{
				"host.json": `{"version": "2.0"}`,
				"pom.xml":   `<project><build><plugins><plugin><artifactId>azure-functions-maven-plugin</artifactId></plugin></plugins></build></project>`,
			},
			expected: "java",
		},
		{
			name: "Java Gradle",
			files: map[string]string{
				"host.json":    `{"version": "2.0"}`,
				"build.gradle": `plugins { id 'com.microsoft.azure.azurefunctions' }`,
			},
			expected: "java",
		},
		{
			name: "Unknown - host.json only",
			files: map[string]string{
				"host.json": `{"version": "2.0"}`,
			},
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

			// Create test files
			for path, content := range tt.files {
				fullPath := filepath.Join(tmpDir, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0750); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
					t.Fatalf("failed to create file %s: %v", path, err)
				}
			}

			// Test detection
			result := detectFunctionsVariantForDiscovery(tmpDir)
			if result != tt.expected {
				t.Errorf("detectFunctionsVariantForDiscovery() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDetectFunctionsLanguageForDiscovery(t *testing.T) {
	tests := []struct {
		name     string
		variant  string
		files    map[string]string
		expected string
	}{
		{
			name:     "Logic Apps",
			variant:  "logicapps",
			files:    map[string]string{},
			expected: "Logic Apps",
		},
		{
			name:    "Node.js TypeScript",
			variant: "nodejs",
			files: map[string]string{
				"tsconfig.json": `{}`,
			},
			expected: "TypeScript",
		},
		{
			name:     "Node.js JavaScript",
			variant:  "nodejs",
			files:    map[string]string{},
			expected: "JavaScript",
		},
		{
			name:     "Python",
			variant:  "python",
			files:    map[string]string{},
			expected: "Python",
		},
		{
			name:     ".NET",
			variant:  "dotnet",
			files:    map[string]string{},
			expected: "C#",
		},
		{
			name:     "Java",
			variant:  "java",
			files:    map[string]string{},
			expected: "Java",
		},
		{
			name:     "Unknown",
			variant:  "unknown",
			files:    map[string]string{},
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

			// Create test files
			for path, content := range tt.files {
				fullPath := filepath.Join(tmpDir, path)
				if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
					t.Fatalf("failed to create file %s: %v", path, err)
				}
			}

			// Test detection
			result := detectFunctionsLanguageForDiscovery(tt.variant, tmpDir)
			if result != tt.expected {
				t.Errorf("detectFunctionsLanguageForDiscovery(%q) = %q, want %q", tt.variant, result, tt.expected)
			}
		})
	}
}
