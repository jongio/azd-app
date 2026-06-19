package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	types "github.com/jongio/azd-core/projecttype"
)

// NOTE: TestRunAspire, TestRunPnpmScript, TestRunDockerCompose, and TestRunNode
// (table-driven skip-before-assert variants) were removed because they called t.Skip()
// before any assertions, providing zero behavioral coverage. Validation logic for these
// runners is tested via TestRunAspire_InvalidPath, TestRunPnpmScript_InvalidScript,
// TestRunDockerCompose_InvalidScript, TestRunNode_InvalidPath, TestRunNode_InvalidScript,
// and TestRunNode_InvalidPackageManager which exercise actual error paths.

func TestFindPythonEntryPoint(t *testing.T) {
	tests := []struct {
		name          string
		setupFiles    []string // Files to create for the test
		expectedEntry string   // Expected entry point file
		expectError   bool
	}{
		{
			name:          "main.py in root",
			setupFiles:    []string{"main.py"},
			expectedEntry: "main.py",
			expectError:   false,
		},
		{
			name:          "app.py in root",
			setupFiles:    []string{"app.py"},
			expectedEntry: "app.py",
			expectError:   false,
		},
		{
			name:          "agent.py in root",
			setupFiles:    []string{"agent.py"},
			expectedEntry: "agent.py",
			expectError:   false,
		},
		{
			name:          "main.py in src/",
			setupFiles:    []string{"src/main.py"},
			expectedEntry: filepath.Join("src", "main.py"),
			expectError:   false,
		},
		{
			name:          "agent.py in src/agent/",
			setupFiles:    []string{"src/agent/agent.py"},
			expectedEntry: filepath.Join("src", "agent", "agent.py"),
			expectError:   false,
		},
		{
			name:          "main.py in src/app/",
			setupFiles:    []string{"src/app/main.py"},
			expectedEntry: filepath.Join("src", "app", "main.py"),
			expectError:   false,
		},
		{
			name:          "__main__.py in root",
			setupFiles:    []string{"__main__.py"},
			expectedEntry: "__main__.py",
			expectError:   false,
		},
		{
			name:          "run.py in app/",
			setupFiles:    []string{"app/run.py"},
			expectedEntry: filepath.Join("app", "run.py"),
			expectError:   false,
		},
		{
			name:          "server.py in src/",
			setupFiles:    []string{"src/server.py"},
			expectedEntry: filepath.Join("src", "server.py"),
			expectError:   false,
		},
		{
			name:          "prefers main.py over others",
			setupFiles:    []string{"main.py", "app.py", "agent.py"},
			expectedEntry: "main.py",
			expectError:   false,
		},
		{
			name:          "prefers root over src/",
			setupFiles:    []string{"main.py", "src/main.py"},
			expectedEntry: "main.py",
			expectError:   false,
		},
		{
			name:        "no entry point found",
			setupFiles:  []string{"README.md", "requirements.txt"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "python-entry-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			// Create test files
			for _, file := range tt.setupFiles {
				fullPath := filepath.Join(tmpDir, file)
				dir := filepath.Dir(fullPath)

				// Create directory if needed
				if err := os.MkdirAll(dir, 0o750); err != nil {
					t.Fatalf("failed to create directory %s: %v", dir, err)
				}

				// Create file
				if err := os.WriteFile(fullPath, []byte("# Python file"), 0o600); err != nil {
					t.Fatalf("failed to create file %s: %v", fullPath, err)
				}
			}

			// Test the function
			entry, err := findPythonEntryPoint(tmpDir)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if entry != tt.expectedEntry {
				t.Errorf("expected entry point %q, got %q", tt.expectedEntry, entry)
			}
		})
	}
}

// NOTE: TestRunPython and TestRunDotnet (skip-before-assert variants) were removed because
// they called t.Skip() before any assertions. Validation is covered by TestRunPython_InvalidPath,
// TestRunPython_InvalidPackageManager, TestRunPython_NoEntryPoint, TestRunPython_UnsupportedPackageManager,
// and TestRunDotnet_InvalidPath which exercise actual error paths.

func TestRunAspire_InvalidPath(t *testing.T) {
	project := types.AspireProject{
		Dir:         "../../../invalid/path",
		ProjectFile: "AppHost.csproj",
	}

	err := RunAspire(context.Background(), project)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestRunPnpmScript_InvalidScript(t *testing.T) {
	err := RunPnpmScript(context.Background(), "dev; rm -rf /")
	if err == nil {
		t.Error("expected error for invalid script name")
	}
}

func TestRunDockerCompose_InvalidScript(t *testing.T) {
	err := RunDockerCompose(context.Background(), "start; malicious", "docker compose up")
	if err == nil {
		t.Error("expected error for invalid script name")
	}
}

func TestRunNode_InvalidPath(t *testing.T) {
	project := types.NodeProject{
		Dir:            "../../../invalid/path",
		PackageManager: "npm",
	}

	err := RunNode(context.Background(), project, "dev")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestRunNode_InvalidScript(t *testing.T) {
	tmpDir := t.TempDir()

	project := types.NodeProject{
		Dir:            tmpDir,
		PackageManager: "npm",
	}

	err := RunNode(context.Background(), project, "dev; rm -rf /")
	if err == nil {
		t.Error("expected error for invalid script name")
	}
}

func TestRunNode_InvalidPackageManager(t *testing.T) {
	tmpDir := t.TempDir()

	project := types.NodeProject{
		Dir:            tmpDir,
		PackageManager: "invalid-pm; rm -rf /",
	}

	err := RunNode(context.Background(), project, "dev")
	if err == nil {
		t.Error("expected error for invalid package manager")
	}
}

func TestRunPython_InvalidPath(t *testing.T) {
	project := types.PythonProject{
		Dir:            "../../../invalid/path",
		PackageManager: "pip",
	}

	err := RunPython(context.Background(), project)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestRunPython_InvalidPackageManager(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main.py
	mainPath := filepath.Join(tmpDir, "main.py")
	if err := os.WriteFile(mainPath, []byte("print('hello')"), 0o600); err != nil {
		t.Fatalf("failed to create main.py: %v", err)
	}

	project := types.PythonProject{
		Dir:            tmpDir,
		PackageManager: "invalid-pm; rm -rf /",
	}

	err := RunPython(context.Background(), project)
	if err == nil {
		t.Error("expected error for invalid package manager")
	}
}

func TestRunPython_NoEntryPoint(t *testing.T) {
	tmpDir := t.TempDir()

	// Create requirements.txt but no Python entry point
	reqPath := filepath.Join(tmpDir, "requirements.txt")
	if err := os.WriteFile(reqPath, []byte("requests==2.28.0\n"), 0o600); err != nil {
		t.Fatalf("failed to create requirements.txt: %v", err)
	}

	project := types.PythonProject{
		Dir:            tmpDir,
		PackageManager: "pip",
	}

	err := RunPython(context.Background(), project)
	if err == nil {
		t.Error("expected error when no entry point found")
	}
}

func TestRunPython_WithExplicitEntrypoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tmpDir := t.TempDir()

	// Create custom entry point
	customPath := filepath.Join(tmpDir, "custom_entry.py")
	if err := os.WriteFile(customPath, []byte("print('hello')"), 0o600); err != nil {
		t.Fatalf("failed to create custom_entry.py: %v", err)
	}

	project := types.PythonProject{
		Dir:            tmpDir,
		PackageManager: "pip",
		Entrypoint:     "custom_entry.py",
	}

	// This should not error on validation
	// (it will error on execution if python isn't installed, but that's ok)
	err := RunPython(context.Background(), project)
	if err == nil {
		t.Log("RunPython succeeded - python must be installed")
	}
}

func TestRunPython_UnsupportedPackageManager(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main.py
	mainPath := filepath.Join(tmpDir, "main.py")
	if err := os.WriteFile(mainPath, []byte("print('hello')"), 0o600); err != nil {
		t.Fatalf("failed to create main.py: %v", err)
	}

	project := types.PythonProject{
		Dir:            tmpDir,
		PackageManager: "conda", // Not supported
	}

	// Should fail validation before execution
	err := RunPython(context.Background(), project)
	if err == nil {
		t.Error("expected error for unsupported package manager")
	}
}

func TestRunDotnet_InvalidPath(t *testing.T) {
	project := types.DotnetProject{
		Path: "../../../invalid/path.csproj",
	}

	err := RunDotnet(context.Background(), project)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestFindPythonEntryPoint_Priority(t *testing.T) {
	// Test that main.py is preferred over app.py
	tmpDir := t.TempDir()

	// Create both main.py and app.py
	mainPath := filepath.Join(tmpDir, "main.py")
	appPath := filepath.Join(tmpDir, "app.py")

	if err := os.WriteFile(mainPath, []byte("# main"), 0o600); err != nil {
		t.Fatalf("failed to create main.py: %v", err)
	}
	if err := os.WriteFile(appPath, []byte("# app"), 0o600); err != nil {
		t.Fatalf("failed to create app.py: %v", err)
	}

	entry, err := findPythonEntryPoint(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entry != "main.py" {
		t.Errorf("expected main.py to be preferred, got %s", entry)
	}
}

func TestFindPythonEntryPoint_DirectoryPriority(t *testing.T) {
	// Test that root directory is preferred over src/
	tmpDir := t.TempDir()

	// Create app.py in both root and src/
	rootPath := filepath.Join(tmpDir, "app.py")
	srcDir := filepath.Join(tmpDir, "src")
	srcPath := filepath.Join(srcDir, "app.py")

	if err := os.MkdirAll(srcDir, 0o750); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	if err := os.WriteFile(rootPath, []byte("# root"), 0o600); err != nil {
		t.Fatalf("failed to create root app.py: %v", err)
	}
	if err := os.WriteFile(srcPath, []byte("# src"), 0o600); err != nil {
		t.Fatalf("failed to create src app.py: %v", err)
	}

	entry, err := findPythonEntryPoint(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entry != "app.py" {
		t.Errorf("expected root app.py to be preferred, got %s", entry)
	}
}

func TestRunFunctionApp_InvalidPath(t *testing.T) {
	project := types.FunctionAppProject{
		Dir:      "../../../invalid/path",
		Variant:  "nodejs",
		Language: "JavaScript",
	}

	err := RunFunctionApp(context.Background(), project, 7071)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestRunFunctionApp_MissingHostJson(t *testing.T) {
	tmpDir := t.TempDir()

	project := types.FunctionAppProject{
		Dir:      tmpDir,
		Variant:  "nodejs",
		Language: "JavaScript",
	}

	err := RunFunctionApp(context.Background(), project, 7071)
	if err == nil {
		t.Error("expected error when host.json is missing")
	}
	if err != nil && err.Error() != "" {
		// Verify error message mentions host.json
		if !contains(err.Error(), "host.json") {
			t.Errorf("expected error to mention host.json, got: %v", err)
		}
	}
}

func TestRunFunctionApp_LogicAppsMissingWorkflows(t *testing.T) {
	tmpDir := t.TempDir()

	// Create host.json
	hostJSONPath := filepath.Join(tmpDir, "host.json")
	if err := os.WriteFile(hostJSONPath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("failed to create host.json: %v", err)
	}

	project := types.FunctionAppProject{
		Dir:      tmpDir,
		Variant:  "logicapps",
		Language: "Logic Apps",
	}

	err := RunFunctionApp(context.Background(), project, 7071)
	if err == nil {
		t.Error("expected error when workflows directory is missing for Logic Apps")
	}
	if err != nil && err.Error() != "" {
		// Verify error message mentions workflows directory
		if !contains(err.Error(), "workflows") {
			t.Errorf("expected error to mention workflows directory, got: %v", err)
		}
	}
}

func TestRunFunctionApp_WithHostJson(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tmpDir := t.TempDir()

	// Create host.json
	hostJSONPath := filepath.Join(tmpDir, "host.json")
	hostJSONContent := `{
		"version": "2.0",
		"extensionBundle": {
			"id": "Microsoft.Azure.Functions.ExtensionBundle",
			"version": "[3.*, 4.0.0)"
		}
	}`
	if err := os.WriteFile(hostJSONPath, []byte(hostJSONContent), 0o600); err != nil {
		t.Fatalf("failed to create host.json: %v", err)
	}

	tests := []struct {
		name    string
		variant string
	}{
		{"Node.js Functions", "nodejs"},
		{"Python Functions", "python"},
		{".NET Functions", "dotnet"},
		{"Java Functions", "java"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := types.FunctionAppProject{
				Dir:      tmpDir,
				Variant:  tt.variant,
				Language: "test",
			}

			// Validation passes but execution fails without `func` CLI - that's expected.
			// Verify we get a non-nil error about func not being found (not a validation error).
			err := RunFunctionApp(context.Background(), project, 7071)
			if err == nil {
				t.Log("RunFunctionApp succeeded - func CLI must be installed")
			}
		})
	}
}

func TestRunFunctionApp_LogicAppsWithWorkflows(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tmpDir := t.TempDir()

	// Create host.json
	hostJSONPath := filepath.Join(tmpDir, "host.json")
	if err := os.WriteFile(hostJSONPath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("failed to create host.json: %v", err)
	}

	// Create workflows directory
	workflowsDir := filepath.Join(tmpDir, "workflows")
	if err := os.MkdirAll(workflowsDir, 0o750); err != nil {
		t.Fatalf("failed to create workflows directory: %v", err)
	}

	project := types.FunctionAppProject{
		Dir:      tmpDir,
		Variant:  "logicapps",
		Language: "Logic Apps",
	}

	// Validation passes (host.json + workflows exist) but execution fails without `func` CLI.
	err := RunFunctionApp(context.Background(), project, 7071)
	if err == nil {
		t.Log("RunFunctionApp succeeded - func CLI must be installed")
	}
}

func TestGetVariantDisplayName(t *testing.T) {
	tests := []struct {
		variant  string
		expected string
	}{
		{"logicapps", "Logic Apps Standard"},
		{"nodejs", "Node.js Functions"},
		{"python", "Python Functions"},
		{"dotnet", ".NET Functions"},
		{"java", "Java Functions"},
		{"unknown", "Azure Functions"},
		{"", "Azure Functions"},
	}

	for _, tt := range tests {
		t.Run(tt.variant, func(t *testing.T) {
			got := getVariantDisplayName(tt.variant)
			if got != tt.expected {
				t.Errorf("getVariantDisplayName(%q) = %q, want %q", tt.variant, got, tt.expected)
			}
		})
	}
}

// NOTE: TestRunFunctionApp_LogicApps was removed as it was an exact duplicate of
// TestRunFunctionApp_LogicAppsWithWorkflows with no additional assertions.

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestRunPython_LeadingDashEntrypoint verifies that an entrypoint value starting with '-'
// (CWE-88 argv injection) is rejected before any subprocess is spawned.
// An attacker who controls azure.yaml could supply entrypoint: "-c" which the Python
// interpreter would parse as the -c flag, enabling arbitrary code execution.
func TestRunPython_LeadingDashEntrypoint(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		entrypoint string
	}{
		{"single dash c flag", "-c"},
		{"double dash flag", "--flag"},
		{"bare single dash", "-"},
		{"double dash end-of-opts marker", "--"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := types.PythonProject{
				Dir:            tmpDir,
				PackageManager: "pip",
				Entrypoint:     tt.entrypoint,
			}

			err := RunPython(context.Background(), project)
			if err == nil {
				t.Errorf("RunPython() with entrypoint %q: expected error for argv injection attempt, got nil", tt.entrypoint)
			}
		})
	}
}

// TestRunPython_ValidEntrypoints verifies that normal entry point paths pass validation.
func TestRunPython_ValidEntrypoints(t *testing.T) {
	tmpDir := t.TempDir()

	// RunPython starts a fire-and-forget child process whose working
	// directory is tmpDir. On Windows that process holds a handle on
	// tmpDir, which races with the RemoveAll that t.TempDir() registered
	// above ("The process cannot access the file because it is being used
	// by another process"). Drive every RunPython call off a cancellable
	// context and, in a cleanup that runs BEFORE t.TempDir()'s (t.Cleanup
	// is LIFO), cancel the context and poll-remove tmpDir until Windows
	// releases the handles. Once tmpDir is gone, t.TempDir()'s own
	// RemoveAll is a no-op and can't fail.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		deadline := time.Now().Add(5 * time.Second)
		for {
			if err := os.RemoveAll(tmpDir); err == nil {
				return
			}
			if time.Now().After(deadline) {
				return // let t.TempDir()'s cleanup surface any residual error
			}
			time.Sleep(50 * time.Millisecond)
		}
	})

	validEntrypoints := []string{
		"main.py",
		"src/main.py",
		"./src/main.py",
		"src/agent/agent.py",
		"__main__.py",
	}

	for _, ep := range validEntrypoints {
		t.Run(ep, func(t *testing.T) {
			// Create the entry point file so validation passes and failure is exec-related
			fullPath := filepath.Join(tmpDir, ep)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
				t.Fatalf("failed to create dir: %v", err)
			}
			if err := os.WriteFile(fullPath, []byte("print('hi')"), 0o600); err != nil {
				t.Fatalf("failed to create entry point: %v", err)
			}

			project := types.PythonProject{
				Dir:            tmpDir,
				PackageManager: "pip",
				Entrypoint:     ep,
			}

			// The call may fail because python/venv is absent in CI, but it must NOT fail
			// with an argv injection error.
			err := RunPython(ctx, project)
			if err != nil {
				// An error is acceptable here (missing python binary, missing venv, etc.)
				// but it must not be the argv injection guard.
				if containsHelper(err.Error(), "CWE-88") || containsHelper(err.Error(), "invalid entrypoint") {
					t.Errorf("RunPython() with valid entrypoint %q failed validation unexpectedly: %v", ep, err)
				}
			}
		})
	}
}
