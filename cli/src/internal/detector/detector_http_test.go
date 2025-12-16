package detector

import (
	"os"
	"path/filepath"
	"testing"
)

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
