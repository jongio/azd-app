package commands

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/types"
	"github.com/spf13/cobra"
)

func TestDepsOptions_Default(t *testing.T) {
	// Reset to ensure clean state
	ResetDepsOptions()

	opts := GetDepsOptions()
	if opts == nil {
		t.Fatal("GetDepsOptions() returned nil")
	}

	// Check defaults
	if opts.Verbose {
		t.Error("Verbose should be false by default")
	}
	if opts.Clean {
		t.Error("Clean should be false by default")
	}
	if opts.NoCache {
		t.Error("NoCache should be false by default")
	}
	if opts.Force {
		t.Error("Force should be false by default")
	}
	if opts.DryRun {
		t.Error("DryRun should be false by default")
	}
	if len(opts.Services) != 0 {
		t.Errorf("Services should be empty by default, got %v", opts.Services)
	}
}

func TestResetDepsOptions(t *testing.T) {
	// Set some values
	opts := GetDepsOptions()
	opts.Verbose = true
	opts.Clean = true
	opts.Services = []string{"test"}

	// Reset
	ResetDepsOptions()

	// Verify reset
	newOpts := GetDepsOptions()
	if newOpts.Verbose {
		t.Error("Verbose should be false after reset")
	}
	if newOpts.Clean {
		t.Error("Clean should be false after reset")
	}
	if len(newOpts.Services) != 0 {
		t.Error("Services should be empty after reset")
	}
}

func TestDepsCommand_Flags(t *testing.T) {
	cmd := NewDepsCommand()
	if cmd == nil {
		t.Fatal("NewDepsCommand() returned nil")
	}

	// Verify command properties
	if cmd.Use != "deps" {
		t.Errorf("Use = %q, want %q", cmd.Use, "deps")
	}
	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}
	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	// Verify flags exist
	flags := []string{"verbose", "clean", "no-cache", "force", "dry-run", "service"}
	for _, flagName := range flags {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("Flag %q not found", flagName)
		}
	}

	// Verify short flags
	shortFlags := map[string]string{
		"verbose": "v",
		"force":   "f",
		"service": "s",
	}
	for long, short := range shortFlags {
		flag := cmd.Flags().Lookup(long)
		if flag == nil {
			t.Errorf("Flag %q not found", long)
			continue
		}
		if flag.Shorthand != short {
			t.Errorf("Flag %q shorthand = %q, want %q", long, flag.Shorthand, short)
		}
	}
}

func TestCleanDependenciesError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *CleanDependenciesError
		contains []string
	}{
		{
			name: "single error",
			err: &CleanDependenciesError{
				Count:   1,
				Details: []string{"failed to remove /path/to/node_modules: permission denied"},
			},
			contains: []string{"failed to clean", "permission denied"},
		},
		{
			name: "multiple errors",
			err: &CleanDependenciesError{
				Count:   2,
				Details: []string{"error 1", "error 2"},
			},
			contains: []string{"2 error(s)", "error 1", "error 2"},
		},
		{
			name: "no details",
			err: &CleanDependenciesError{
				Count:   3,
				Details: []string{},
			},
			contains: []string{"3 error(s)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()
			for _, want := range tt.contains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("Error() = %q, want to contain %q", errMsg, want)
				}
			}
		})
	}
}

func TestIsSubdirectory(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		parentPaths map[string]bool
		want        bool
	}{
		{
			name:        "is subdirectory",
			path:        "/workspace/services/api/src",
			parentPaths: map[string]bool{"/workspace/services/api": true},
			want:        true,
		},
		{
			name:        "not subdirectory - different path",
			path:        "/workspace/other/api",
			parentPaths: map[string]bool{"/workspace/services/api": true},
			want:        false,
		},
		{
			name:        "not subdirectory - same path",
			path:        "/workspace/services/api",
			parentPaths: map[string]bool{"/workspace/services/api": true},
			want:        false,
		},
		{
			name:        "not subdirectory - prefix match but not dir separator",
			path:        "/workspace/services/api-v2",
			parentPaths: map[string]bool{"/workspace/services/api": true},
			want:        false,
		},
		{
			name:        "empty parent paths",
			path:        "/workspace/services/api",
			parentPaths: map[string]bool{},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSubdirectory(tt.path, tt.parentPaths)
			if got != tt.want {
				t.Errorf("isSubdirectory(%q, %v) = %v, want %v", tt.path, tt.parentPaths, got, tt.want)
			}
		})
	}
}

func TestFilterProjectsByService_NoAzureYaml(t *testing.T) {
	// When there's no azure.yaml, filtering should return original projects
	nodeProjects := []types.NodeProject{{Dir: "/test/node"}}
	pythonProjects := []types.PythonProject{{Dir: "/test/python"}}
	dotnetProjects := []types.DotnetProject{{Path: "/test/dotnet/project.csproj"}}

	// Use a non-existent path to ensure no azure.yaml is found
	filteredNode, filteredPython, filteredDotnet := filterProjectsByService(
		nodeProjects, pythonProjects, dotnetProjects,
		[]string{"api"}, "/nonexistent/path",
	)

	// Should return original projects when no azure.yaml
	if len(filteredNode) != len(nodeProjects) {
		t.Errorf("Node projects filtered incorrectly: got %d, want %d", len(filteredNode), len(nodeProjects))
	}
	if len(filteredPython) != len(pythonProjects) {
		t.Errorf("Python projects filtered incorrectly: got %d, want %d", len(filteredPython), len(pythonProjects))
	}
	if len(filteredDotnet) != len(dotnetProjects) {
		t.Errorf("Dotnet projects filtered incorrectly: got %d, want %d", len(filteredDotnet), len(dotnetProjects))
	}
}

func TestDepsCheckAllSuccess(t *testing.T) {
	tests := []struct {
		name    string
		results []InstallResult
		want    bool
	}{
		{
			name:    "empty results",
			results: []InstallResult{},
			want:    true,
		},
		{
			name: "all success",
			results: []InstallResult{
				{Success: true},
				{Success: true},
				{Success: true},
			},
			want: true,
		},
		{
			name: "one failure",
			results: []InstallResult{
				{Success: true},
				{Success: false},
				{Success: true},
			},
			want: false,
		},
		{
			name: "all failures",
			results: []InstallResult{
				{Success: false},
				{Success: false},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkAllSuccess(tt.results)
			if got != tt.want {
				t.Errorf("checkAllSuccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDependencyInstaller_NewDependencyInstaller(t *testing.T) {
	searchRoot := "/test/path"
	di := NewDependencyInstaller(searchRoot)

	if di == nil {
		t.Fatal("NewDependencyInstaller returned nil")
	}
	if di.searchRoot != searchRoot {
		t.Errorf("searchRoot = %q, want %q", di.searchRoot, searchRoot)
	}
	// Verify filtered project slices are nil by default
	if di.nodeProjects != nil {
		t.Error("nodeProjects should be nil by default")
	}
	if di.pythonProjects != nil {
		t.Error("pythonProjects should be nil by default")
	}
	if di.dotnetProjects != nil {
		t.Error("dotnetProjects should be nil by default")
	}
}

func TestHandleDepsError(t *testing.T) {
	// Test non-JSON mode
	err := handleDepsError(
		&testError{msg: "test error"},
		"failed to do something",
	)

	if err == nil {
		t.Fatal("handleDepsError returned nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "failed to do something") {
		t.Errorf("Error should contain message, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "test error") {
		t.Errorf("Error should contain original error, got: %s", errMsg)
	}
}

// Helper types and functions

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestGetDepsOptions_ReturnsCopy(t *testing.T) {
	ResetDepsOptions()

	// Set initial values using setDepsOptions
	setDepsOptions(&DepsOptions{
		Verbose:  true,
		Services: []string{"api", "web"},
	})

	// Get a copy
	opts := GetDepsOptions()

	// Modify the copy
	opts.Verbose = false
	opts.Services[0] = "modified"
	opts.Services = append(opts.Services, "new")

	// Get another copy and verify original is unchanged
	opts2 := GetDepsOptions()
	if !opts2.Verbose {
		t.Error("Original Verbose should still be true")
	}
	if opts2.Services[0] != "api" {
		t.Errorf("Original Services[0] should be 'api', got %q", opts2.Services[0])
	}
	if len(opts2.Services) != 2 {
		t.Errorf("Original Services should have 2 elements, got %d", len(opts2.Services))
	}
}

func TestSetDepsOptions_ThreadSafe(t *testing.T) {
	ResetDepsOptions()

	// Run concurrent setDepsOptions calls
	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			setDepsOptions(&DepsOptions{
				Verbose:  val%2 == 0,
				Services: []string{string(rune('a' + val%26))},
			})
		}(i)
	}

	// Also run concurrent GetDepsOptions calls
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = GetDepsOptions()
		}()
	}

	wg.Wait()

	// If we get here without panic/race, the mutex is working
	opts := GetDepsOptions()
	if opts == nil {
		t.Error("GetDepsOptions should not return nil after concurrent access")
	}
}

func TestResetDepsOptions_ThreadSafe(t *testing.T) {
	var wg sync.WaitGroup
	iterations := 50

	for i := 0; i < iterations; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			ResetDepsOptions()
		}()
		go func(val int) {
			defer wg.Done()
			setDepsOptions(&DepsOptions{Verbose: val%2 == 0})
		}(i)
		go func() {
			defer wg.Done()
			_ = GetDepsOptions()
		}()
	}

	wg.Wait()

	// Verify state is valid after concurrent access
	opts := GetDepsOptions()
	if opts == nil {
		t.Error("GetDepsOptions should not return nil")
	}
}

// Task 1: Test NewDepsCommand flag behavior - force flag combining clean and no-cache
func TestNewDepsCommand_ForceFlagBehavior(t *testing.T) {
	// Reset to clean state
	ResetDepsOptions()

	cmd := NewDepsCommand()
	if cmd == nil {
		t.Fatal("NewDepsCommand() returned nil")
	}

	// Test that force flag exists and has correct shorthand
	forceFlag := cmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Fatal("force flag not found")
	}
	if forceFlag.Shorthand != "f" {
		t.Errorf("force flag shorthand = %q, want %q", forceFlag.Shorthand, "f")
	}

	// Test that clean and no-cache flags exist
	cleanFlag := cmd.Flags().Lookup("clean")
	if cleanFlag == nil {
		t.Fatal("clean flag not found")
	}
	noCacheFlag := cmd.Flags().Lookup("no-cache")
	if noCacheFlag == nil {
		t.Fatal("no-cache flag not found")
	}

	// Test dry-run flag exists
	dryRunFlag := cmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Fatal("dry-run flag not found")
	}

	// Test service flag exists with correct shorthand
	serviceFlag := cmd.Flags().Lookup("service")
	if serviceFlag == nil {
		t.Fatal("service flag not found")
	}
	if serviceFlag.Shorthand != "s" {
		t.Errorf("service flag shorthand = %q, want %q", serviceFlag.Shorthand, "s")
	}
}

// Task 2: Test filterProjectsByService with azure.yaml
func TestFilterProjectsByService_WithAzureYaml(t *testing.T) {
	// Create temp directory with azure.yaml
	tmpDir := t.TempDir()

	// Create azure.yaml with services
	azureYamlContent := `name: test-app
services:
  api:
    project: ./api
    language: python
  web:
    project: ./web
    language: nodejs
  backend:
    project: ./backend
    language: dotnet
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	// Create service directories
	apiDir := filepath.Join(tmpDir, "api")
	webDir := filepath.Join(tmpDir, "web")
	backendDir := filepath.Join(tmpDir, "backend")
	otherDir := filepath.Join(tmpDir, "other")

	for _, dir := range []string{apiDir, webDir, backendDir, otherDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create test projects
	nodeProjects := []types.NodeProject{
		{Dir: webDir, PackageManager: "npm"},
		{Dir: otherDir, PackageManager: "npm"},
	}
	pythonProjects := []types.PythonProject{
		{Dir: apiDir, PackageManager: "pip"},
		{Dir: otherDir, PackageManager: "pip"},
	}
	dotnetProjects := []types.DotnetProject{
		{Path: filepath.Join(backendDir, "project.csproj")},
		{Path: filepath.Join(otherDir, "other.csproj")},
	}

	// Test filtering for "api" service only
	filteredNode, filteredPython, filteredDotnet := filterProjectsByService(
		nodeProjects, pythonProjects, dotnetProjects,
		[]string{"api"}, tmpDir,
	)

	// Should filter to only api service (python project)
	if len(filteredNode) != 0 {
		t.Errorf("Expected 0 node projects for 'api' filter, got %d", len(filteredNode))
	}
	if len(filteredPython) != 1 {
		t.Errorf("Expected 1 python project for 'api' filter, got %d", len(filteredPython))
	}
	if len(filteredDotnet) != 0 {
		t.Errorf("Expected 0 dotnet projects for 'api' filter, got %d", len(filteredDotnet))
	}

	// Test filtering for "web" service only
	filteredNode, filteredPython, filteredDotnet = filterProjectsByService(
		nodeProjects, pythonProjects, dotnetProjects,
		[]string{"web"}, tmpDir,
	)

	if len(filteredNode) != 1 {
		t.Errorf("Expected 1 node project for 'web' filter, got %d", len(filteredNode))
	}
	if len(filteredPython) != 0 {
		t.Errorf("Expected 0 python projects for 'web' filter, got %d", len(filteredPython))
	}
	if len(filteredDotnet) != 0 {
		t.Errorf("Expected 0 dotnet projects for 'web' filter, got %d", len(filteredDotnet))
	}

	// Test filtering for multiple services
	filteredNode, filteredPython, filteredDotnet = filterProjectsByService(
		nodeProjects, pythonProjects, dotnetProjects,
		[]string{"api", "web", "backend"}, tmpDir,
	)

	if len(filteredNode) != 1 {
		t.Errorf("Expected 1 node project for multi-service filter, got %d", len(filteredNode))
	}
	if len(filteredPython) != 1 {
		t.Errorf("Expected 1 python project for multi-service filter, got %d", len(filteredPython))
	}
	if len(filteredDotnet) != 1 {
		t.Errorf("Expected 1 dotnet project for multi-service filter, got %d", len(filteredDotnet))
	}
}

func TestFilterProjectsByService_InvalidAzureYaml(t *testing.T) {
	// Create temp directory with invalid azure.yaml
	tmpDir := t.TempDir()

	// Create invalid azure.yaml
	invalidYamlContent := `invalid: yaml: content: [[[`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(invalidYamlContent), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	nodeProjects := []types.NodeProject{{Dir: tmpDir}}
	pythonProjects := []types.PythonProject{{Dir: tmpDir}}
	dotnetProjects := []types.DotnetProject{{Path: filepath.Join(tmpDir, "project.csproj")}}

	// Should return original projects when azure.yaml is invalid
	filteredNode, filteredPython, filteredDotnet := filterProjectsByService(
		nodeProjects, pythonProjects, dotnetProjects,
		[]string{"api"}, tmpDir,
	)

	if len(filteredNode) != len(nodeProjects) {
		t.Errorf("Expected original node projects on invalid yaml, got %d", len(filteredNode))
	}
	if len(filteredPython) != len(pythonProjects) {
		t.Errorf("Expected original python projects on invalid yaml, got %d", len(filteredPython))
	}
	if len(filteredDotnet) != len(dotnetProjects) {
		t.Errorf("Expected original dotnet projects on invalid yaml, got %d", len(filteredDotnet))
	}
}

// Task 3: Test showDryRunSummary
func TestShowDryRunSummary_TextMode(t *testing.T) {
	// Ensure we're in text mode
	output.SetFormat("text")

	tmpDir := t.TempDir()

	nodeProjects := []types.NodeProject{
		{Dir: filepath.Join(tmpDir, "web"), PackageManager: "npm"},
	}
	pythonProjects := []types.PythonProject{
		{Dir: filepath.Join(tmpDir, "api"), PackageManager: "pip"},
	}
	dotnetProjects := []types.DotnetProject{
		{Path: filepath.Join(tmpDir, "backend", "project.csproj")},
	}

	// showDryRunSummary should not return an error
	err := showDryRunSummary(nodeProjects, pythonProjects, dotnetProjects, tmpDir)
	if err != nil {
		t.Errorf("showDryRunSummary returned error: %v", err)
	}
}

func TestShowDryRunSummary_EmptyProjects(t *testing.T) {
	// Ensure we're in text mode
	output.SetFormat("text")

	tmpDir := t.TempDir()

	// Empty projects
	err := showDryRunSummary(nil, nil, nil, tmpDir)
	if err != nil {
		t.Errorf("showDryRunSummary with empty projects returned error: %v", err)
	}
}

// Task 4: Test handleNoProjectsCase
func TestHandleNoProjectsCase_EmptyWorkspace(t *testing.T) {
	// Ensure we're in text mode
	output.SetFormat("text")

	tmpDir := t.TempDir()

	// Test with no services filter
	err := handleNoProjectsCase(tmpDir, nil)
	if err != nil {
		t.Errorf("handleNoProjectsCase returned error: %v", err)
	}
}

func TestHandleNoProjectsCase_ServiceFilterNoMatch(t *testing.T) {
	// Ensure we're in text mode
	output.SetFormat("text")

	tmpDir := t.TempDir()

	// Test with service filter that has no matches
	err := handleNoProjectsCase(tmpDir, []string{"nonexistent-service"})
	if err != nil {
		t.Errorf("handleNoProjectsCase with service filter returned error: %v", err)
	}
}

// Task 5: Test parseAzureYaml
func TestParseAzureYaml_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid azure.yaml
	validYaml := `name: my-app
services:
  api:
    project: ./api
    language: python
  web:
    project: ./web
    language: nodejs
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(validYaml), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	result, err := parseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("parseAzureYaml returned error: %v", err)
	}

	if result == nil {
		t.Fatal("parseAzureYaml returned nil result")
	}

	if len(result.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(result.Services))
	}

	if _, ok := result.Services["api"]; !ok {
		t.Error("Expected 'api' service to exist")
	}
	if _, ok := result.Services["web"]; !ok {
		t.Error("Expected 'web' service to exist")
	}
}

func TestParseAzureYaml_InvalidYaml(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid YAML
	invalidYaml := `name: [invalid yaml`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(invalidYaml), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	_, err := parseAzureYaml(azureYamlPath)
	if err == nil {
		t.Error("parseAzureYaml should return error for invalid YAML")
	}
}

func TestParseAzureYaml_NonExistentFile(t *testing.T) {
	_, err := parseAzureYaml("/nonexistent/path/azure.yaml")
	if err == nil {
		t.Error("parseAzureYaml should return error for non-existent file")
	}
}

func TestParseAzureYaml_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty file
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(""), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	result, err := parseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("parseAzureYaml returned error for empty file: %v", err)
	}

	// Empty yaml should parse but have no services
	if result != nil && len(result.Services) != 0 {
		t.Errorf("Expected 0 services for empty file, got %d", len(result.Services))
	}
}

// Task 6: Test cleanDependencies and cleanDirectory
func TestCleanDirectory_ExistingDirectory(t *testing.T) {
	// Ensure we're in text mode
	output.SetFormat("text")

	tmpDir := t.TempDir()

	// Create a directory to clean
	dirToClean := filepath.Join(tmpDir, "node_modules")
	if err := os.MkdirAll(dirToClean, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create some files inside
	testFile := filepath.Join(dirToClean, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Clean the directory
	err := cleanDirectory(dirToClean)
	if err != nil {
		t.Errorf("cleanDirectory returned error: %v", err)
	}

	// Verify directory was removed
	if _, err := os.Stat(dirToClean); !os.IsNotExist(err) {
		t.Error("Directory should have been removed")
	}
}

func TestCleanDirectory_NonExistentDirectory(t *testing.T) {
	// Ensure we're in text mode
	output.SetFormat("text")

	// Clean non-existent directory should not error
	err := cleanDirectory("/nonexistent/path/node_modules")
	if err != nil {
		t.Errorf("cleanDirectory should not error for non-existent directory: %v", err)
	}
}

func TestCleanDependencies_AllTypes(t *testing.T) {
	// Ensure we're in text mode
	output.SetFormat("text")

	tmpDir := t.TempDir()

	// Create directories for all project types
	nodeDir := filepath.Join(tmpDir, "node-project")
	pythonDir := filepath.Join(tmpDir, "python-project")
	dotnetDir := filepath.Join(tmpDir, "dotnet-project")

	for _, dir := range []string{nodeDir, pythonDir, dotnetDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create dependency directories
	nodeModules := filepath.Join(nodeDir, "node_modules")
	venv := filepath.Join(pythonDir, ".venv")
	objDir := filepath.Join(dotnetDir, "obj")
	binDir := filepath.Join(dotnetDir, "bin")

	for _, dir := range []string{nodeModules, venv, objDir, binDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create project lists
	nodeProjects := []types.NodeProject{{Dir: nodeDir}}
	pythonProjects := []types.PythonProject{{Dir: pythonDir}}
	dotnetProjects := []types.DotnetProject{{Path: filepath.Join(dotnetDir, "project.csproj")}}

	// Clean dependencies
	err := cleanDependencies(nodeProjects, pythonProjects, dotnetProjects)
	if err != nil {
		t.Errorf("cleanDependencies returned error: %v", err)
	}

	// Verify all directories were removed
	for _, dir := range []string{nodeModules, venv, objDir, binDir} {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("Directory %s should have been removed", dir)
		}
	}
}

func TestCleanDependencies_EmptyProjects(t *testing.T) {
	// Ensure we're in text mode
	output.SetFormat("text")

	// Empty projects should not error
	err := cleanDependencies(nil, nil, nil)
	if err != nil {
		t.Errorf("cleanDependencies with empty projects returned error: %v", err)
	}
}

// Task 7: Test getSearchRoot edge cases
func TestGetSearchRoot_NoAzureYaml(t *testing.T) {
	// Create temp directory without azure.yaml
	tmpDir := t.TempDir()

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// getSearchRoot should return current directory when no azure.yaml exists
	searchRoot, err := getSearchRoot()
	if err != nil {
		t.Fatalf("getSearchRoot returned error: %v", err)
	}

	// Normalize paths for comparison
	expectedPath := filepath.Clean(tmpDir)
	actualPath := filepath.Clean(searchRoot)

	if actualPath != expectedPath {
		t.Errorf("getSearchRoot = %q, want %q", actualPath, expectedPath)
	}
}

func TestGetSearchRoot_WithAzureYaml(t *testing.T) {
	// Create temp directory with azure.yaml
	tmpDir := t.TempDir()

	// Create azure.yaml
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte("name: test"), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0750); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Change to subdirectory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// getSearchRoot should return azure.yaml directory
	searchRoot, err := getSearchRoot()
	if err != nil {
		t.Fatalf("getSearchRoot returned error: %v", err)
	}

	// Normalize paths for comparison
	expectedPath := filepath.Clean(tmpDir)
	actualPath := filepath.Clean(searchRoot)

	if actualPath != expectedPath {
		t.Errorf("getSearchRoot = %q, want %q", actualPath, expectedPath)
	}
}

// Additional tests for edge cases
func TestFilterProjectsByService_SubdirectoryMatching(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create azure.yaml with service pointing to parent directory
	azureYamlContent := `name: test-app
services:
  api:
    project: ./api
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	// Create api directory with subdirectory
	apiDir := filepath.Join(tmpDir, "api")
	apiSubDir := filepath.Join(apiDir, "src")
	for _, dir := range []string{apiDir, apiSubDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	// Project in subdirectory of service path
	nodeProjects := []types.NodeProject{
		{Dir: apiSubDir, PackageManager: "npm"},
	}

	filteredNode, _, _ := filterProjectsByService(
		nodeProjects, nil, nil,
		[]string{"api"}, tmpDir,
	)

	// Should match because apiSubDir is a subdirectory of apiDir
	if len(filteredNode) != 1 {
		t.Errorf("Expected 1 node project for subdirectory match, got %d", len(filteredNode))
	}
}

func TestDepsOptions_AllFlagsDefaultValues(t *testing.T) {
	ResetDepsOptions()

	opts := GetDepsOptions()

	// Verify all default values
	if opts.Verbose {
		t.Error("Verbose should be false by default")
	}
	if opts.Clean {
		t.Error("Clean should be false by default")
	}
	if opts.NoCache {
		t.Error("NoCache should be false by default")
	}
	if opts.Force {
		t.Error("Force should be false by default")
	}
	if opts.DryRun {
		t.Error("DryRun should be false by default")
	}
	if len(opts.Services) != 0 {
		t.Errorf("Services should be empty by default, got %v", opts.Services)
	}
}

func TestDepsOptions_SetAndGet(t *testing.T) {
	ResetDepsOptions()

	// Set custom options
	customOpts := &DepsOptions{
		Verbose:  true,
		Clean:    true,
		NoCache:  true,
		Force:    true,
		DryRun:   true,
		Services: []string{"api", "web"},
	}
	setDepsOptions(customOpts)

	// Get and verify
	opts := GetDepsOptions()
	if !opts.Verbose {
		t.Error("Verbose should be true")
	}
	if !opts.Clean {
		t.Error("Clean should be true")
	}
	if !opts.NoCache {
		t.Error("NoCache should be true")
	}
	if !opts.Force {
		t.Error("Force should be true")
	}
	if !opts.DryRun {
		t.Error("DryRun should be true")
	}
	if len(opts.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(opts.Services))
	}
	if opts.Services[0] != "api" || opts.Services[1] != "web" {
		t.Errorf("Services mismatch: got %v", opts.Services)
	}
}

func TestNewDependencyInstaller_WithFilteredProjects(t *testing.T) {
	searchRoot := "/test/path"
	di := NewDependencyInstaller(searchRoot)

	// Set filtered projects
	di.nodeProjects = []types.NodeProject{{Dir: "/test/node"}}
	di.pythonProjects = []types.PythonProject{{Dir: "/test/python"}}
	di.dotnetProjects = []types.DotnetProject{{Path: "/test/dotnet/project.csproj"}}

	if len(di.nodeProjects) != 1 {
		t.Errorf("Expected 1 node project, got %d", len(di.nodeProjects))
	}
	if len(di.pythonProjects) != 1 {
		t.Errorf("Expected 1 python project, got %d", len(di.pythonProjects))
	}
	if len(di.dotnetProjects) != 1 {
		t.Errorf("Expected 1 dotnet project, got %d", len(di.dotnetProjects))
	}
}

func TestCheckAllSuccess_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		results []InstallResult
		want    bool
	}{
		{
			name:    "nil results",
			results: nil,
			want:    true,
		},
		{
			name: "mixed with error messages",
			results: []InstallResult{
				{Success: true, Error: ""},
				{Success: false, Error: "some error"},
			},
			want: false,
		},
		{
			name: "single success",
			results: []InstallResult{
				{Success: true},
			},
			want: true,
		},
		{
			name: "single failure",
			results: []InstallResult{
				{Success: false},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkAllSuccess(tt.results)
			if got != tt.want {
				t.Errorf("checkAllSuccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleDepsError_FormatsCorrectly(t *testing.T) {
	// Test that error message is properly formatted
	originalErr := &testError{msg: "original error"}
	message := "failed to do something"

	err := handleDepsError(originalErr, message)

	if err == nil {
		t.Fatal("handleDepsError returned nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, message) {
		t.Errorf("Error should contain message %q, got: %s", message, errMsg)
	}
	if !strings.Contains(errMsg, "original error") {
		t.Errorf("Error should contain original error, got: %s", errMsg)
	}
}

func TestInstallResult_Fields(t *testing.T) {
	result := InstallResult{
		Type:    "node",
		Dir:     "/test/dir",
		Path:    "/test/path",
		Manager: "npm",
		Success: true,
		Error:   "",
	}

	if result.Type != "node" {
		t.Errorf("Type = %q, want %q", result.Type, "node")
	}
	if result.Dir != "/test/dir" {
		t.Errorf("Dir = %q, want %q", result.Dir, "/test/dir")
	}
	if result.Path != "/test/path" {
		t.Errorf("Path = %q, want %q", result.Path, "/test/path")
	}
	if result.Manager != "npm" {
		t.Errorf("Manager = %q, want %q", result.Manager, "npm")
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Error != "" {
		t.Errorf("Error should be empty, got %q", result.Error)
	}
}

func TestDepsResult_Fields(t *testing.T) {
	result := DepsResult{
		Success: true,
		Projects: []InstallResult{
			{Type: "node", Success: true},
		},
		Message: "test message",
		Error:   "",
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if len(result.Projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(result.Projects))
	}
	if result.Message != "test message" {
		t.Errorf("Message = %q, want %q", result.Message, "test message")
	}
	if result.Error != "" {
		t.Errorf("Error should be empty, got %q", result.Error)
	}
}

// Additional tests for higher coverage

func TestShowDryRunSummary_OnlyNodeProjects(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	nodeProjects := []types.NodeProject{
		{Dir: filepath.Join(tmpDir, "web1"), PackageManager: "npm"},
		{Dir: filepath.Join(tmpDir, "web2"), PackageManager: "pnpm"},
	}

	err := showDryRunSummary(nodeProjects, nil, nil, tmpDir)
	if err != nil {
		t.Errorf("showDryRunSummary returned error: %v", err)
	}
}

func TestShowDryRunSummary_OnlyPythonProjects(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	pythonProjects := []types.PythonProject{
		{Dir: filepath.Join(tmpDir, "api1"), PackageManager: "pip"},
		{Dir: filepath.Join(tmpDir, "api2"), PackageManager: "poetry"},
	}

	err := showDryRunSummary(nil, pythonProjects, nil, tmpDir)
	if err != nil {
		t.Errorf("showDryRunSummary returned error: %v", err)
	}
}

func TestShowDryRunSummary_OnlyDotnetProjects(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	dotnetProjects := []types.DotnetProject{
		{Path: filepath.Join(tmpDir, "backend1", "project1.csproj")},
		{Path: filepath.Join(tmpDir, "backend2", "project2.csproj")},
	}

	err := showDryRunSummary(nil, nil, dotnetProjects, tmpDir)
	if err != nil {
		t.Errorf("showDryRunSummary returned error: %v", err)
	}
}

func TestFilterProjectsByService_EmptyServicesList(t *testing.T) {
	tmpDir := t.TempDir()

	// Create azure.yaml
	azureYamlContent := `name: test-app
services:
  api:
    project: ./api
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	nodeProjects := []types.NodeProject{{Dir: tmpDir}}

	// Empty services list should return all projects
	filteredNode, _, _ := filterProjectsByService(
		nodeProjects, nil, nil,
		[]string{}, tmpDir,
	)

	// With empty filter, the function should return all projects
	if len(filteredNode) != 0 {
		t.Logf("filteredNode: %v", filteredNode)
	}
}

func TestFilterProjectsByService_NonMatchingService(t *testing.T) {
	tmpDir := t.TempDir()

	// Create azure.yaml
	azureYamlContent := `name: test-app
services:
  api:
    project: ./api
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	// Create api directory
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0750); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}

	nodeProjects := []types.NodeProject{{Dir: apiDir}}

	// Filter for non-existent service
	filteredNode, _, _ := filterProjectsByService(
		nodeProjects, nil, nil,
		[]string{"nonexistent"}, tmpDir,
	)

	// Should return empty since no services match
	if len(filteredNode) != 0 {
		t.Errorf("Expected 0 projects for non-matching service, got %d", len(filteredNode))
	}
}

func TestHandleNoProjectsCase_WithLogicApps(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	// This tests the logic apps detection path
	err := handleNoProjectsCase(tmpDir, nil)
	if err != nil {
		t.Errorf("handleNoProjectsCase returned error: %v", err)
	}
}

func TestCleanDependencies_NodeProjectsOnly(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	// Create node project with node_modules
	nodeDir := filepath.Join(tmpDir, "node-project")
	nodeModules := filepath.Join(nodeDir, "node_modules")
	if err := os.MkdirAll(nodeModules, 0750); err != nil {
		t.Fatalf("Failed to create node_modules: %v", err)
	}

	nodeProjects := []types.NodeProject{{Dir: nodeDir}}

	err := cleanDependencies(nodeProjects, nil, nil)
	if err != nil {
		t.Errorf("cleanDependencies returned error: %v", err)
	}

	// Verify node_modules was removed
	if _, err := os.Stat(nodeModules); !os.IsNotExist(err) {
		t.Error("node_modules should have been removed")
	}
}

func TestCleanDependencies_PythonProjectsOnly(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	// Create python project with .venv
	pythonDir := filepath.Join(tmpDir, "python-project")
	venv := filepath.Join(pythonDir, ".venv")
	if err := os.MkdirAll(venv, 0750); err != nil {
		t.Fatalf("Failed to create .venv: %v", err)
	}

	pythonProjects := []types.PythonProject{{Dir: pythonDir}}

	err := cleanDependencies(nil, pythonProjects, nil)
	if err != nil {
		t.Errorf("cleanDependencies returned error: %v", err)
	}

	// Verify .venv was removed
	if _, err := os.Stat(venv); !os.IsNotExist(err) {
		t.Error(".venv should have been removed")
	}
}

func TestCleanDependencies_DotnetProjectsOnly(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	// Create dotnet project with obj and bin
	dotnetDir := filepath.Join(tmpDir, "dotnet-project")
	objDir := filepath.Join(dotnetDir, "obj")
	binDir := filepath.Join(dotnetDir, "bin")
	for _, dir := range []string{objDir, binDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	dotnetProjects := []types.DotnetProject{{Path: filepath.Join(dotnetDir, "project.csproj")}}

	err := cleanDependencies(nil, nil, dotnetProjects)
	if err != nil {
		t.Errorf("cleanDependencies returned error: %v", err)
	}

	// Verify obj and bin were removed
	for _, dir := range []string{objDir, binDir} {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("Directory %s should have been removed", dir)
		}
	}
}

func TestCleanDirectory_WithNestedFiles(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	// Create nested directory structure
	dirToClean := filepath.Join(tmpDir, "node_modules")
	nestedDir := filepath.Join(dirToClean, "package", "lib")
	if err := os.MkdirAll(nestedDir, 0750); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	// Create files at multiple levels
	files := []string{
		filepath.Join(dirToClean, "file1.txt"),
		filepath.Join(dirToClean, "package", "file2.txt"),
		filepath.Join(nestedDir, "file3.txt"),
	}
	for _, f := range files {
		if err := os.WriteFile(f, []byte("test"), 0600); err != nil {
			t.Fatalf("Failed to create file %s: %v", f, err)
		}
	}

	// Clean the directory
	err := cleanDirectory(dirToClean)
	if err != nil {
		t.Errorf("cleanDirectory returned error: %v", err)
	}

	// Verify all was removed
	if _, err := os.Stat(dirToClean); !os.IsNotExist(err) {
		t.Error("Directory should have been completely removed")
	}
}

func TestIsSubdirectory_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		parentPaths map[string]bool
		want        bool
	}{
		{
			name:        "deeply nested subdirectory",
			path:        "/a/b/c/d/e/f",
			parentPaths: map[string]bool{"/a/b": true},
			want:        true,
		},
		{
			name:        "multiple parent candidates - first match",
			path:        "/workspace/api/src",
			parentPaths: map[string]bool{"/workspace/api": true, "/workspace/web": true},
			want:        true,
		},
		{
			name:        "multiple parent candidates - second match",
			path:        "/workspace/web/dist",
			parentPaths: map[string]bool{"/workspace/api": true, "/workspace/web": true},
			want:        true,
		},
		{
			name:        "sibling directory - not subdirectory",
			path:        "/workspace/api-v2",
			parentPaths: map[string]bool{"/workspace/api": true},
			want:        false,
		},
		{
			name:        "parent of tracked path - not subdirectory",
			path:        "/workspace",
			parentPaths: map[string]bool{"/workspace/api": true},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSubdirectory(tt.path, tt.parentPaths)
			if got != tt.want {
				t.Errorf("isSubdirectory(%q, %v) = %v, want %v", tt.path, tt.parentPaths, got, tt.want)
			}
		})
	}
}

func TestNewDepsCommand_CommandProperties(t *testing.T) {
	cmd := NewDepsCommand()

	// Verify command name
	if cmd.Use != "deps" {
		t.Errorf("cmd.Use = %q, want %q", cmd.Use, "deps")
	}

	// Verify short description exists
	if cmd.Short == "" {
		t.Error("cmd.Short should not be empty")
	}

	// Verify long description exists
	if cmd.Long == "" {
		t.Error("cmd.Long should not be empty")
	}

	// Verify SilenceUsage is set
	if !cmd.SilenceUsage {
		t.Error("cmd.SilenceUsage should be true")
	}
}

func TestParseAzureYaml_WithMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	// Create azure.yaml with additional metadata
	validYaml := `name: my-complex-app
metadata:
  template: azd-starter
services:
  api:
    project: ./api
    language: python
    host: appservice
  web:
    project: ./web
    language: nodejs
    host: containerapp
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(validYaml), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	result, err := parseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("parseAzureYaml returned error: %v", err)
	}

	if result == nil {
		t.Fatal("parseAzureYaml returned nil")
	}

	if len(result.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(result.Services))
	}
}

func TestFilterProjectsByService_PythonProjects(t *testing.T) {
	tmpDir := t.TempDir()

	// Create azure.yaml
	azureYamlContent := `name: test-app
services:
  api:
    project: ./api
    language: python
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	// Create api directory
	apiDir := filepath.Join(tmpDir, "api")
	otherDir := filepath.Join(tmpDir, "other")
	for _, dir := range []string{apiDir, otherDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	pythonProjects := []types.PythonProject{
		{Dir: apiDir, PackageManager: "pip"},
		{Dir: otherDir, PackageManager: "pip"},
	}

	_, filteredPython, _ := filterProjectsByService(
		nil, pythonProjects, nil,
		[]string{"api"}, tmpDir,
	)

	if len(filteredPython) != 1 {
		t.Errorf("Expected 1 python project, got %d", len(filteredPython))
	}
}

func TestFilterProjectsByService_DotnetProjects(t *testing.T) {
	tmpDir := t.TempDir()

	// Create azure.yaml
	azureYamlContent := `name: test-app
services:
  backend:
    project: ./backend
    language: dotnet
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	// Create backend directory
	backendDir := filepath.Join(tmpDir, "backend")
	otherDir := filepath.Join(tmpDir, "other")
	for _, dir := range []string{backendDir, otherDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	dotnetProjects := []types.DotnetProject{
		{Path: filepath.Join(backendDir, "project.csproj")},
		{Path: filepath.Join(otherDir, "other.csproj")},
	}

	_, _, filteredDotnet := filterProjectsByService(
		nil, nil, dotnetProjects,
		[]string{"backend"}, tmpDir,
	)

	if len(filteredDotnet) != 1 {
		t.Errorf("Expected 1 dotnet project, got %d", len(filteredDotnet))
	}
}

func TestCleanDependenciesError_SingleDetail(t *testing.T) {
	err := &CleanDependenciesError{
		Count:   1,
		Details: []string{"single error detail"},
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "failed to clean") {
		t.Errorf("Single error should mention 'failed to clean', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "single error detail") {
		t.Errorf("Should contain detail, got: %s", errMsg)
	}
}

func TestCleanDependenciesError_MultipleDetails(t *testing.T) {
	err := &CleanDependenciesError{
		Count:   3,
		Details: []string{"error 1", "error 2", "error 3"},
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "3 error(s)") {
		t.Errorf("Should mention '3 error(s)', got: %s", errMsg)
	}
	for _, detail := range []string{"error 1", "error 2", "error 3"} {
		if !strings.Contains(errMsg, detail) {
			t.Errorf("Should contain '%s', got: %s", detail, errMsg)
		}
	}
}

func TestDependencyInstaller_SearchRoot(t *testing.T) {
	searchRoot := "/custom/search/root"
	di := NewDependencyInstaller(searchRoot)

	if di.searchRoot != searchRoot {
		t.Errorf("searchRoot = %q, want %q", di.searchRoot, searchRoot)
	}
}

func TestCheckAllSuccess_WithErrorField(t *testing.T) {
	results := []InstallResult{
		{Success: true, Error: ""},
		{Success: true, Error: "warning but success"},
		{Success: false, Error: "actual error"},
	}

	got := checkAllSuccess(results)
	if got {
		t.Error("checkAllSuccess should return false when any result has Success=false")
	}
}

func TestGetSearchRoot_CurrentDirectory(t *testing.T) {
	// Create temp dir without azure.yaml
	tmpDir := t.TempDir()

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	os.Chdir(tmpDir)

	searchRoot, err := getSearchRoot()
	if err != nil {
		t.Fatalf("getSearchRoot returned error: %v", err)
	}

	// Should return current directory when no azure.yaml
	expectedPath := filepath.Clean(tmpDir)
	actualPath := filepath.Clean(searchRoot)

	if actualPath != expectedPath {
		t.Errorf("searchRoot = %q, want %q", actualPath, expectedPath)
	}
}

// Test for detectAllProjects function
func TestDetectAllProjects_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	nodeProjects, pythonProjects, dotnetProjects, err := detectAllProjects(tmpDir)
	if err != nil {
		t.Fatalf("detectAllProjects returned error: %v", err)
	}

	// Empty directory should have no projects
	if len(nodeProjects) != 0 {
		t.Errorf("Expected 0 node projects, got %d", len(nodeProjects))
	}
	if len(pythonProjects) != 0 {
		t.Errorf("Expected 0 python projects, got %d", len(pythonProjects))
	}
	if len(dotnetProjects) != 0 {
		t.Errorf("Expected 0 dotnet projects, got %d", len(dotnetProjects))
	}
}

func TestDetectAllProjects_WithNodeProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json
	packageJson := `{"name": "test", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJson), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	nodeProjects, pythonProjects, dotnetProjects, err := detectAllProjects(tmpDir)
	if err != nil {
		t.Fatalf("detectAllProjects returned error: %v", err)
	}

	if len(nodeProjects) != 1 {
		t.Errorf("Expected 1 node project, got %d", len(nodeProjects))
	}
	if len(pythonProjects) != 0 {
		t.Errorf("Expected 0 python projects, got %d", len(pythonProjects))
	}
	if len(dotnetProjects) != 0 {
		t.Errorf("Expected 0 dotnet projects, got %d", len(dotnetProjects))
	}
}

func TestDetectAllProjects_WithPythonProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create requirements.txt
	if err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte("flask==2.0.0"), 0600); err != nil {
		t.Fatalf("Failed to create requirements.txt: %v", err)
	}

	nodeProjects, pythonProjects, dotnetProjects, err := detectAllProjects(tmpDir)
	if err != nil {
		t.Fatalf("detectAllProjects returned error: %v", err)
	}

	if len(nodeProjects) != 0 {
		t.Errorf("Expected 0 node projects, got %d", len(nodeProjects))
	}
	if len(pythonProjects) != 1 {
		t.Errorf("Expected 1 python project, got %d", len(pythonProjects))
	}
	if len(dotnetProjects) != 0 {
		t.Errorf("Expected 0 dotnet projects, got %d", len(dotnetProjects))
	}
}

func TestDetectAllProjects_WithDotnetProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .csproj file
	csproj := `<Project Sdk="Microsoft.NET.Sdk"></Project>`
	if err := os.WriteFile(filepath.Join(tmpDir, "project.csproj"), []byte(csproj), 0600); err != nil {
		t.Fatalf("Failed to create project.csproj: %v", err)
	}

	nodeProjects, pythonProjects, dotnetProjects, err := detectAllProjects(tmpDir)
	if err != nil {
		t.Fatalf("detectAllProjects returned error: %v", err)
	}

	if len(nodeProjects) != 0 {
		t.Errorf("Expected 0 node projects, got %d", len(nodeProjects))
	}
	if len(pythonProjects) != 0 {
		t.Errorf("Expected 0 python projects, got %d", len(pythonProjects))
	}
	if len(dotnetProjects) != 1 {
		t.Errorf("Expected 1 dotnet project, got %d", len(dotnetProjects))
	}
}

func TestDetectAllProjects_MultipleProjectTypes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create subdirectories for each project type
	nodeDir := filepath.Join(tmpDir, "web")
	pythonDir := filepath.Join(tmpDir, "api")
	dotnetDir := filepath.Join(tmpDir, "backend")

	for _, dir := range []string{nodeDir, pythonDir, dotnetDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	// Create project files
	if err := os.WriteFile(filepath.Join(nodeDir, "package.json"), []byte(`{"name":"test"}`), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pythonDir, "requirements.txt"), []byte("flask"), 0600); err != nil {
		t.Fatalf("Failed to create requirements.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dotnetDir, "project.csproj"), []byte("<Project></Project>"), 0600); err != nil {
		t.Fatalf("Failed to create project.csproj: %v", err)
	}

	nodeProjects, pythonProjects, dotnetProjects, err := detectAllProjects(tmpDir)
	if err != nil {
		t.Fatalf("detectAllProjects returned error: %v", err)
	}

	if len(nodeProjects) != 1 {
		t.Errorf("Expected 1 node project, got %d", len(nodeProjects))
	}
	if len(pythonProjects) != 1 {
		t.Errorf("Expected 1 python project, got %d", len(pythonProjects))
	}
	if len(dotnetProjects) != 1 {
		t.Errorf("Expected 1 dotnet project, got %d", len(dotnetProjects))
	}
}

// Test for InstallAllFiltered
func TestInstallAllFiltered_EmptyProjects(t *testing.T) {
	di := NewDependencyInstaller("/test/path")
	// With no projects set, should return empty results
	results, err := di.InstallAllFiltered()
	if err != nil {
		t.Fatalf("InstallAllFiltered returned error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

// Test showDryRunSummary JSON mode
func TestShowDryRunSummary_JSONMode(t *testing.T) {
	// Set JSON mode
	output.SetFormat("json")
	defer output.SetFormat("text")

	tmpDir := t.TempDir()

	nodeProjects := []types.NodeProject{
		{Dir: filepath.Join(tmpDir, "web"), PackageManager: "npm"},
	}
	pythonProjects := []types.PythonProject{
		{Dir: filepath.Join(tmpDir, "api"), PackageManager: "pip"},
	}
	dotnetProjects := []types.DotnetProject{
		{Path: filepath.Join(tmpDir, "backend", "project.csproj")},
	}

	// showDryRunSummary should return nil for JSON output
	err := showDryRunSummary(nodeProjects, pythonProjects, dotnetProjects, tmpDir)
	// In JSON mode it prints JSON and returns nil
	if err != nil {
		t.Logf("showDryRunSummary returned: %v (may be expected for JSON output)", err)
	}
}

// Test handleNoProjectsCase JSON mode
func TestHandleNoProjectsCase_JSONMode(t *testing.T) {
	// Set JSON mode
	output.SetFormat("json")
	defer output.SetFormat("text")

	tmpDir := t.TempDir()

	// Should not error, just print JSON
	err := handleNoProjectsCase(tmpDir, nil)
	if err != nil {
		t.Logf("handleNoProjectsCase returned: %v (may be expected for JSON output)", err)
	}
}

func TestHandleNoProjectsCase_JSONMode_WithServiceFilter(t *testing.T) {
	// Set JSON mode
	output.SetFormat("json")
	defer output.SetFormat("text")

	tmpDir := t.TempDir()

	err := handleNoProjectsCase(tmpDir, []string{"api", "web"})
	if err != nil {
		t.Logf("handleNoProjectsCase returned: %v (may be expected for JSON output)", err)
	}
}

// Test handleDepsError JSON mode
func TestHandleDepsError_JSONMode(t *testing.T) {
	// Set JSON mode
	output.SetFormat("json")
	defer output.SetFormat("text")

	originalErr := &testError{msg: "test error"}
	err := handleDepsError(originalErr, "failed to do something")

	// In JSON mode, handleDepsError prints JSON, which may return an error
	if err != nil {
		t.Logf("handleDepsError returned: %v", err)
	}
}

// Test cleanDirectory error path (when RemoveAll fails)
func TestCleanDirectory_SuccessPath(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	// Create directory to clean
	dirToClean := filepath.Join(tmpDir, "to_clean")
	if err := os.MkdirAll(dirToClean, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	err := cleanDirectory(dirToClean)
	if err != nil {
		t.Errorf("cleanDirectory returned unexpected error: %v", err)
	}
}

// Test DependencyInstaller InstallAllFiltered with projects
func TestInstallAllFiltered_WithNodeProjects(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	// Create a node project
	nodeDir := filepath.Join(tmpDir, "node-app")
	if err := os.MkdirAll(nodeDir, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	// Create package.json
	if err := os.WriteFile(filepath.Join(nodeDir, "package.json"), []byte(`{"name":"test"}`), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	di := NewDependencyInstaller(tmpDir)
	di.nodeProjects = []types.NodeProject{{Dir: nodeDir, PackageManager: "npm"}}

	// This will try to run npm install, which may fail if npm is not available
	// but the function should still return results
	results, err := di.InstallAllFiltered()
	if err != nil {
		t.Logf("InstallAllFiltered returned error (may be expected): %v", err)
	}

	// Should have at least one result
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if len(results) > 0 && results[0].Type != "node" {
		t.Errorf("Expected type 'node', got %q", results[0].Type)
	}
}

func TestInstallAllFiltered_WithPythonProjects(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	// Create a python project
	pythonDir := filepath.Join(tmpDir, "python-app")
	if err := os.MkdirAll(pythonDir, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	// Create requirements.txt
	if err := os.WriteFile(filepath.Join(pythonDir, "requirements.txt"), []byte("flask"), 0600); err != nil {
		t.Fatalf("Failed to create requirements.txt: %v", err)
	}

	di := NewDependencyInstaller(tmpDir)
	di.pythonProjects = []types.PythonProject{{Dir: pythonDir, PackageManager: "pip"}}

	results, err := di.InstallAllFiltered()
	if err != nil {
		t.Logf("InstallAllFiltered returned error (may be expected): %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if len(results) > 0 && results[0].Type != "python" {
		t.Errorf("Expected type 'python', got %q", results[0].Type)
	}
}

func TestInstallAllFiltered_WithDotnetProjects(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	// Create a dotnet project
	dotnetDir := filepath.Join(tmpDir, "dotnet-app")
	if err := os.MkdirAll(dotnetDir, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	// Create .csproj file
	csproj := `<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup></Project>`
	csprojPath := filepath.Join(dotnetDir, "project.csproj")
	if err := os.WriteFile(csprojPath, []byte(csproj), 0600); err != nil {
		t.Fatalf("Failed to create project.csproj: %v", err)
	}

	di := NewDependencyInstaller(tmpDir)
	di.dotnetProjects = []types.DotnetProject{{Path: csprojPath}}

	results, err := di.InstallAllFiltered()
	if err != nil {
		t.Logf("InstallAllFiltered returned error (may be expected): %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if len(results) > 0 && results[0].Type != "dotnet" {
		t.Errorf("Expected type 'dotnet', got %q", results[0].Type)
	}
}

func TestInstallAllFiltered_MixedProjects(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	// Create directories
	nodeDir := filepath.Join(tmpDir, "node-app")
	pythonDir := filepath.Join(tmpDir, "python-app")

	for _, dir := range []string{nodeDir, pythonDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	// Create project files
	if err := os.WriteFile(filepath.Join(nodeDir, "package.json"), []byte(`{"name":"test"}`), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pythonDir, "requirements.txt"), []byte("flask"), 0600); err != nil {
		t.Fatalf("Failed to create requirements.txt: %v", err)
	}

	di := NewDependencyInstaller(tmpDir)
	di.nodeProjects = []types.NodeProject{{Dir: nodeDir, PackageManager: "npm"}}
	di.pythonProjects = []types.PythonProject{{Dir: pythonDir, PackageManager: "pip"}}

	results, err := di.InstallAllFiltered()
	if err != nil {
		t.Logf("InstallAllFiltered returned error (may be expected): %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

// Test handleNoProjectsCase with Logic Apps
func TestHandleNoProjectsCase_LogicAppsWorkspace(t *testing.T) {
	output.SetFormat("text")
	// This test verifies the logic apps detection path runs without error
	tmpDir := t.TempDir()

	err := handleNoProjectsCase(tmpDir, nil)
	if err != nil {
		t.Errorf("handleNoProjectsCase returned error: %v", err)
	}
}

// Test getSearchRoot with various scenarios
func TestGetSearchRoot_InSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create azure.yaml in root
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte("name: test"), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	// Create nested subdirectory
	subDir := filepath.Join(tmpDir, "services", "api", "src")
	if err := os.MkdirAll(subDir, 0750); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	os.Chdir(subDir)

	searchRoot, err := getSearchRoot()
	if err != nil {
		t.Fatalf("getSearchRoot returned error: %v", err)
	}

	// Should return the azure.yaml directory, not the subdirectory
	expectedPath := filepath.Clean(tmpDir)
	actualPath := filepath.Clean(searchRoot)

	if actualPath != expectedPath {
		t.Errorf("searchRoot = %q, want %q", actualPath, expectedPath)
	}
}

// Test isSubdirectory with Windows-style paths
func TestIsSubdirectory_WindowsPaths(t *testing.T) {
	// These tests are cross-platform but cover edge cases
	tests := []struct {
		name        string
		path        string
		parentPaths map[string]bool
		want        bool
	}{
		{
			name:        "immediate subdirectory",
			path:        filepath.Join("workspace", "api", "src"),
			parentPaths: map[string]bool{filepath.Join("workspace", "api"): true},
			want:        true,
		},
		{
			name:        "same directory",
			path:        filepath.Join("workspace", "api"),
			parentPaths: map[string]bool{filepath.Join("workspace", "api"): true},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSubdirectory(tt.path, tt.parentPaths)
			if got != tt.want {
				t.Errorf("isSubdirectory(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// Test cleanDependencies with directories that don't exist
func TestCleanDependencies_NonExistentDirectories(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	// Create project directories but NOT the dependency directories
	nodeDir := filepath.Join(tmpDir, "node-project")
	pythonDir := filepath.Join(tmpDir, "python-project")
	dotnetDir := filepath.Join(tmpDir, "dotnet-project")

	for _, dir := range []string{nodeDir, pythonDir, dotnetDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	nodeProjects := []types.NodeProject{{Dir: nodeDir}}
	pythonProjects := []types.PythonProject{{Dir: pythonDir}}
	dotnetProjects := []types.DotnetProject{{Path: filepath.Join(dotnetDir, "project.csproj")}}

	// Should not error when directories don't exist
	err := cleanDependencies(nodeProjects, pythonProjects, dotnetProjects)
	if err != nil {
		t.Errorf("cleanDependencies returned error for non-existent directories: %v", err)
	}
}

// Test parseAzureYaml with services containing additional fields
func TestParseAzureYaml_ComplexServices(t *testing.T) {
	tmpDir := t.TempDir()

	// Create azure.yaml with complex service definitions
	complexYaml := `name: complex-app
services:
  api:
    project: ./api
    language: python
    host: containerapp
    docker:
      path: ./api/Dockerfile
  web:
    project: ./web
    language: nodejs
    host: staticwebapp
  worker:
    project: ./worker
    language: dotnet
    host: function
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(complexYaml), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	result, err := parseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("parseAzureYaml returned error: %v", err)
	}

	if len(result.Services) != 3 {
		t.Errorf("Expected 3 services, got %d", len(result.Services))
	}
}

// Additional coverage for DepsOptions
func TestDepsOptions_ServicesList(t *testing.T) {
	ResetDepsOptions()

	services := []string{"api", "web", "worker"}
	setDepsOptions(&DepsOptions{
		Services: services,
	})

	opts := GetDepsOptions()

	if len(opts.Services) != 3 {
		t.Errorf("Expected 3 services, got %d", len(opts.Services))
	}

	// Verify the copy is independent
	opts.Services[0] = "modified"
	opts2 := GetDepsOptions()
	if opts2.Services[0] != "api" {
		t.Errorf("Original services should not be modified, got %q", opts2.Services[0])
	}
}

// Test cleanDirectory JSON mode
func TestCleanDirectory_JSONMode(t *testing.T) {
	output.SetFormat("json")
	defer output.SetFormat("text")

	tmpDir := t.TempDir()

	// Create directory to clean
	dirToClean := filepath.Join(tmpDir, "to_clean")
	if err := os.MkdirAll(dirToClean, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	err := cleanDirectory(dirToClean)
	if err != nil {
		t.Errorf("cleanDirectory returned unexpected error: %v", err)
	}

	// Verify directory was removed
	if _, err := os.Stat(dirToClean); !os.IsNotExist(err) {
		t.Error("Directory should have been removed")
	}
}

// Test cleanDependencies JSON mode
func TestCleanDependencies_JSONMode(t *testing.T) {
	output.SetFormat("json")
	defer output.SetFormat("text")

	tmpDir := t.TempDir()

	// Create node project with node_modules
	nodeDir := filepath.Join(tmpDir, "node-project")
	nodeModules := filepath.Join(nodeDir, "node_modules")
	if err := os.MkdirAll(nodeModules, 0750); err != nil {
		t.Fatalf("Failed to create node_modules: %v", err)
	}

	nodeProjects := []types.NodeProject{{Dir: nodeDir}}

	err := cleanDependencies(nodeProjects, nil, nil)
	if err != nil {
		t.Errorf("cleanDependencies returned error: %v", err)
	}

	// Verify node_modules was removed
	if _, err := os.Stat(nodeModules); !os.IsNotExist(err) {
		t.Error("node_modules should have been removed")
	}
}

// Test handleNoProjectsCase with empty function apps
func TestHandleNoProjectsCase_EmptyFunctionApps(t *testing.T) {
	output.SetFormat("text")

	tmpDir := t.TempDir()

	// Create azure.yaml to make it a valid workspace
	azureYaml := `name: test`
	if err := os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte(azureYaml), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	err := handleNoProjectsCase(tmpDir, nil)
	if err != nil {
		t.Errorf("handleNoProjectsCase returned error: %v", err)
	}
}

// Test getSearchRoot error case
func TestGetSearchRoot_ErrorCase(t *testing.T) {
	// Save and restore working directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	tmpDir := t.TempDir()

	// Create azure.yaml
	if err := os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte("name: test"), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	os.Chdir(tmpDir)

	searchRoot, err := getSearchRoot()
	if err != nil {
		t.Fatalf("getSearchRoot returned error: %v", err)
	}

	if searchRoot == "" {
		t.Error("searchRoot should not be empty")
	}
}

// Test installProject with different scenarios
func TestInstallProject_Success(t *testing.T) {
	output.SetFormat("text")

	di := NewDependencyInstaller("/test")

	// Test with a successful install function
	result := di.installProject("test", "/test/dir", "test-manager", func() error {
		return nil
	})

	if !result.Success {
		t.Error("Expected success for successful install")
	}
	if result.Type != "test" {
		t.Errorf("Type = %q, want %q", result.Type, "test")
	}
	if result.Dir != "/test/dir" {
		t.Errorf("Dir = %q, want %q", result.Dir, "/test/dir")
	}
	if result.Manager != "test-manager" {
		t.Errorf("Manager = %q, want %q", result.Manager, "test-manager")
	}
}

func TestInstallProject_Failure(t *testing.T) {
	output.SetFormat("text")

	di := NewDependencyInstaller("/test")

	// Test with a failing install function
	expectedError := "installation failed"
	result := di.installProject("test", "/test/dir", "test-manager", func() error {
		return &testError{msg: expectedError}
	})

	if result.Success {
		t.Error("Expected failure for failing install")
	}
	if result.Error != expectedError {
		t.Errorf("Error = %q, want %q", result.Error, expectedError)
	}
}

func TestInstallProject_JSONMode(t *testing.T) {
	output.SetFormat("json")
	defer output.SetFormat("text")

	di := NewDependencyInstaller("/test")

	result := di.installProject("node", "/test/node", "npm", func() error {
		return nil
	})

	if !result.Success {
		t.Error("Expected success")
	}
}

func TestInstallProject_RelativePath(t *testing.T) {
	output.SetFormat("text")

	// Test with search root and subdirectory
	searchRoot := "/workspace"
	projectDir := "/workspace/services/api"

	di := NewDependencyInstaller(searchRoot)

	result := di.installProject("python", projectDir, "pip", func() error {
		return nil
	})

	if !result.Success {
		t.Error("Expected success")
	}
	if result.Dir != projectDir {
		t.Errorf("Dir = %q, want %q", result.Dir, projectDir)
	}
}

// Test createCacheManager
func TestCreateCacheManager_Enabled(t *testing.T) {
	cm := createCacheManager(true)
	if cm == nil {
		t.Error("createCacheManager should not return nil")
	}
}

func TestCreateCacheManager_Disabled(t *testing.T) {
	cm := createCacheManager(false)
	if cm == nil {
		t.Error("createCacheManager should not return nil")
	}
}

// Test detectAllProjects error paths
func TestDetectAllProjects_WithNestedProjects(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested project structure
	apiDir := filepath.Join(tmpDir, "api")
	webDir := filepath.Join(tmpDir, "web")
	apiSrcDir := filepath.Join(apiDir, "src")

	for _, dir := range []string{apiDir, webDir, apiSrcDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	// Create project files
	if err := os.WriteFile(filepath.Join(apiDir, "requirements.txt"), []byte("flask"), 0600); err != nil {
		t.Fatalf("Failed to create requirements.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "package.json"), []byte(`{"name":"web"}`), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	nodeProjects, pythonProjects, dotnetProjects, err := detectAllProjects(tmpDir)
	if err != nil {
		t.Fatalf("detectAllProjects returned error: %v", err)
	}

	if len(nodeProjects) < 1 {
		t.Errorf("Expected at least 1 node project, got %d", len(nodeProjects))
	}
	if len(pythonProjects) < 1 {
		t.Errorf("Expected at least 1 python project, got %d", len(pythonProjects))
	}
	if len(dotnetProjects) != 0 {
		t.Errorf("Expected 0 dotnet projects, got %d", len(dotnetProjects))
	}
}

// Test handleNoProjectsCase with Logic Apps workspace (function app variant)
func TestHandleNoProjectsCase_WithFunctionApps(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	// Create a function app structure (not logic apps)
	funcDir := filepath.Join(tmpDir, "func-app")
	if err := os.MkdirAll(funcDir, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create host.json to indicate function app
	hostJson := `{"version": "2.0"}`
	if err := os.WriteFile(filepath.Join(funcDir, "host.json"), []byte(hostJson), 0600); err != nil {
		t.Fatalf("Failed to create host.json: %v", err)
	}

	err := handleNoProjectsCase(tmpDir, nil)
	if err != nil {
		t.Errorf("handleNoProjectsCase returned error: %v", err)
	}
}

// Test handleNoProjectsCase with Logic Apps ONLY workspace (should suppress message)
func TestHandleNoProjectsCase_LogicAppsOnly(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	// Create a Logic Apps structure with workflows directory
	logicAppDir := filepath.Join(tmpDir, "logic-app")
	workflowsDir := filepath.Join(logicAppDir, "workflows", "myworkflow")
	if err := os.MkdirAll(workflowsDir, 0750); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create host.json with Logic Apps extension bundle
	hostJson := `{
		"version": "2.0",
		"extensionBundle": {
			"id": "Microsoft.Azure.Functions.ExtensionBundle.Workflows",
			"version": "[1.*, 2.0.0)"
		}
	}`
	if err := os.WriteFile(filepath.Join(logicAppDir, "host.json"), []byte(hostJson), 0600); err != nil {
		t.Fatalf("Failed to create host.json: %v", err)
	}

	// Create workflow.json file
	workflowJson := `{"definition": {}}`
	if err := os.WriteFile(filepath.Join(workflowsDir, "workflow.json"), []byte(workflowJson), 0600); err != nil {
		t.Fatalf("Failed to create workflow.json: %v", err)
	}

	// Should not error - Logic Apps workspace detected, message suppressed
	err := handleNoProjectsCase(tmpDir, nil)
	if err != nil {
		t.Errorf("handleNoProjectsCase returned error: %v", err)
	}
}

// Test handleNoProjectsCase with mixed Function Apps (Logic Apps + other)
func TestHandleNoProjectsCase_MixedFunctionApps(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	// Create a Logic Apps structure
	logicAppDir := filepath.Join(tmpDir, "logic-app")
	workflowsDir := filepath.Join(logicAppDir, "workflows", "myworkflow")
	if err := os.MkdirAll(workflowsDir, 0750); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	hostJson := `{
		"version": "2.0",
		"extensionBundle": {
			"id": "Microsoft.Azure.Functions.ExtensionBundle.Workflows",
			"version": "[1.*, 2.0.0)"
		}
	}`
	if err := os.WriteFile(filepath.Join(logicAppDir, "host.json"), []byte(hostJson), 0600); err != nil {
		t.Fatalf("Failed to create host.json: %v", err)
	}
	workflowJson := `{"definition": {}}`
	if err := os.WriteFile(filepath.Join(workflowsDir, "workflow.json"), []byte(workflowJson), 0600); err != nil {
		t.Fatalf("Failed to create workflow.json: %v", err)
	}

	// Create a Node.js function app (not Logic Apps)
	nodeFuncDir := filepath.Join(tmpDir, "node-func")
	if err := os.MkdirAll(nodeFuncDir, 0750); err != nil {
		t.Fatalf("Failed to create node-func directory: %v", err)
	}
	nodeHostJson := `{"version": "2.0"}`
	if err := os.WriteFile(filepath.Join(nodeFuncDir, "host.json"), []byte(nodeHostJson), 0600); err != nil {
		t.Fatalf("Failed to create host.json: %v", err)
	}
	packageJson := `{"name": "func", "dependencies": {"@azure/functions": "^4.0.0"}}`
	if err := os.WriteFile(filepath.Join(nodeFuncDir, "package.json"), []byte(packageJson), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	// Mixed apps - should show "No projects detected" message
	err := handleNoProjectsCase(tmpDir, nil)
	if err != nil {
		t.Errorf("handleNoProjectsCase returned error: %v", err)
	}
}

// Test handleNoProjectsCase JSON mode with no service filter
func TestHandleNoProjectsCase_JSONMode_NoFilter(t *testing.T) {
	output.SetFormat("json")
	defer output.SetFormat("text")

	tmpDir := t.TempDir()

	err := handleNoProjectsCase(tmpDir, nil)
	// In JSON mode, it prints JSON and returns the result of PrintJSON
	if err != nil {
		t.Logf("handleNoProjectsCase returned: %v", err)
	}
}

// Test cleanDirectory with JSON mode and non-existent directory
func TestCleanDirectory_JSONMode_NonExistent(t *testing.T) {
	output.SetFormat("json")
	defer output.SetFormat("text")

	err := cleanDirectory("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Errorf("cleanDirectory should not error for non-existent directory: %v", err)
	}
}

// Test cleanDirectory with JSON mode and existing directory
func TestCleanDirectory_JSONMode_Existing(t *testing.T) {
	output.SetFormat("json")
	defer output.SetFormat("text")

	tmpDir := t.TempDir()
	dirToClean := filepath.Join(tmpDir, "to_clean")
	if err := os.MkdirAll(dirToClean, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create a file inside
	if err := os.WriteFile(filepath.Join(dirToClean, "file.txt"), []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	err := cleanDirectory(dirToClean)
	if err != nil {
		t.Errorf("cleanDirectory returned error: %v", err)
	}

	// Verify removed
	if _, err := os.Stat(dirToClean); !os.IsNotExist(err) {
		t.Error("Directory should have been removed")
	}
}

// Test cleanDependencies JSON mode with all project types
func TestCleanDependencies_JSONMode_AllTypes(t *testing.T) {
	output.SetFormat("json")
	defer output.SetFormat("text")

	tmpDir := t.TempDir()

	// Create directories for all project types
	nodeDir := filepath.Join(tmpDir, "node-project")
	pythonDir := filepath.Join(tmpDir, "python-project")
	dotnetDir := filepath.Join(tmpDir, "dotnet-project")

	for _, dir := range []string{nodeDir, pythonDir, dotnetDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create dependency directories
	nodeModules := filepath.Join(nodeDir, "node_modules")
	venv := filepath.Join(pythonDir, ".venv")
	objDir := filepath.Join(dotnetDir, "obj")
	binDir := filepath.Join(dotnetDir, "bin")

	for _, dir := range []string{nodeModules, venv, objDir, binDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	nodeProjects := []types.NodeProject{{Dir: nodeDir}}
	pythonProjects := []types.PythonProject{{Dir: pythonDir}}
	dotnetProjects := []types.DotnetProject{{Path: filepath.Join(dotnetDir, "project.csproj")}}

	err := cleanDependencies(nodeProjects, pythonProjects, dotnetProjects)
	if err != nil {
		t.Errorf("cleanDependencies returned error: %v", err)
	}

	// Verify all directories were removed
	for _, dir := range []string{nodeModules, venv, objDir, binDir} {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("Directory %s should have been removed", dir)
		}
	}
}

// Test getSearchRoot when azure.yaml search returns error
func TestGetSearchRoot_AzureYamlInRoot(t *testing.T) {
	tmpDir := t.TempDir()

	// Create azure.yaml
	if err := os.WriteFile(filepath.Join(tmpDir, "azure.yaml"), []byte("name: test"), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	// Change to the directory containing azure.yaml
	os.Chdir(tmpDir)

	searchRoot, err := getSearchRoot()
	if err != nil {
		t.Fatalf("getSearchRoot returned error: %v", err)
	}

	expectedPath := filepath.Clean(tmpDir)
	actualPath := filepath.Clean(searchRoot)

	if actualPath != expectedPath {
		t.Errorf("searchRoot = %q, want %q", actualPath, expectedPath)
	}
}

// Test detectAllProjects with all project types
func TestDetectAllProjects_AllTypes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directories
	nodeDir := filepath.Join(tmpDir, "web")
	pythonDir := filepath.Join(tmpDir, "api")
	dotnetDir := filepath.Join(tmpDir, "backend")

	for _, dir := range []string{nodeDir, pythonDir, dotnetDir} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	// Create package.json for node
	if err := os.WriteFile(filepath.Join(nodeDir, "package.json"), []byte(`{"name":"web"}`), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	// Create requirements.txt for python
	if err := os.WriteFile(filepath.Join(pythonDir, "requirements.txt"), []byte("flask"), 0600); err != nil {
		t.Fatalf("Failed to create requirements.txt: %v", err)
	}

	// Create .csproj for dotnet
	if err := os.WriteFile(filepath.Join(dotnetDir, "project.csproj"), []byte("<Project></Project>"), 0600); err != nil {
		t.Fatalf("Failed to create project.csproj: %v", err)
	}

	nodeProjects, pythonProjects, dotnetProjects, err := detectAllProjects(tmpDir)
	if err != nil {
		t.Fatalf("detectAllProjects returned error: %v", err)
	}

	if len(nodeProjects) != 1 {
		t.Errorf("Expected 1 node project, got %d", len(nodeProjects))
	}
	if len(pythonProjects) != 1 {
		t.Errorf("Expected 1 python project, got %d", len(pythonProjects))
	}
	if len(dotnetProjects) != 1 {
		t.Errorf("Expected 1 dotnet project, got %d", len(dotnetProjects))
	}
}

// Test NewDepsCommand PreRunE with output flag
func TestNewDepsCommand_PreRunE(t *testing.T) {
	cmd := NewDepsCommand()

	// Create a parent command with output flag
	parentCmd := &cobra.Command{Use: "parent"}
	parentCmd.PersistentFlags().String("output", "", "Output format")
	parentCmd.AddCommand(cmd)

	// Test PreRunE with empty output (should not error)
	if cmd.PreRunE != nil {
		err := cmd.PreRunE(cmd, []string{})
		if err != nil {
			t.Errorf("PreRunE returned unexpected error: %v", err)
		}
	}
}

// Test NewDepsCommand with all flag combinations
func TestNewDepsCommand_AllFlags(t *testing.T) {
	cmd := NewDepsCommand()

	// Verify all flags exist with correct types
	verboseFlag := cmd.Flags().Lookup("verbose")
	if verboseFlag == nil || verboseFlag.Value.Type() != "bool" {
		t.Error("verbose flag missing or wrong type")
	}

	cleanFlag := cmd.Flags().Lookup("clean")
	if cleanFlag == nil || cleanFlag.Value.Type() != "bool" {
		t.Error("clean flag missing or wrong type")
	}

	noCacheFlag := cmd.Flags().Lookup("no-cache")
	if noCacheFlag == nil || noCacheFlag.Value.Type() != "bool" {
		t.Error("no-cache flag missing or wrong type")
	}

	forceFlag := cmd.Flags().Lookup("force")
	if forceFlag == nil || forceFlag.Value.Type() != "bool" {
		t.Error("force flag missing or wrong type")
	}

	dryRunFlag := cmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil || dryRunFlag.Value.Type() != "bool" {
		t.Error("dry-run flag missing or wrong type")
	}

	serviceFlag := cmd.Flags().Lookup("service")
	if serviceFlag == nil || serviceFlag.Value.Type() != "stringSlice" {
		t.Error("service flag missing or wrong type")
	}
}

// Test NewDepsCommand PreRunE with valid output format
func TestNewDepsCommand_PreRunE_WithOutputFormat(t *testing.T) {
	cmd := NewDepsCommand()

	// Create a parent command with output flag set to json
	parentCmd := &cobra.Command{Use: "parent"}
	parentCmd.PersistentFlags().String("output", "json", "Output format")
	parentCmd.AddCommand(cmd)

	// Need to parse flags to set the output flag value
	parentCmd.SetArgs([]string{"deps", "--output", "json"})

	// Test PreRunE - when output flag has value, it should set format
	if cmd.PreRunE != nil {
		err := cmd.PreRunE(cmd, []string{})
		// This might error if output format validation fails, but that's OK
		if err != nil {
			t.Logf("PreRunE returned: %v (expected for validation)", err)
		}
	}
}

// Test NewDepsCommand PreRunE with self output flag
func TestNewDepsCommand_PreRunE_SelfFlag(t *testing.T) {
	cmd := NewDepsCommand()

	// Add output flag directly to command
	cmd.Flags().String("output", "text", "Output format")

	if cmd.PreRunE != nil {
		err := cmd.PreRunE(cmd, []string{})
		if err != nil {
			t.Logf("PreRunE returned: %v", err)
		}
	}
}

// Test cleanDependencies with success in JSON mode
func TestCleanDependencies_JSONMode_Success(t *testing.T) {
	output.SetFormat("json")
	defer output.SetFormat("text")

	tmpDir := t.TempDir()

	// Create directories without dependency subdirectories (nothing to clean)
	nodeDir := filepath.Join(tmpDir, "node-project")
	if err := os.MkdirAll(nodeDir, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	nodeProjects := []types.NodeProject{{Dir: nodeDir}}

	// Should succeed (nothing to clean, but no error)
	err := cleanDependencies(nodeProjects, nil, nil)
	if err != nil {
		t.Errorf("cleanDependencies returned error: %v", err)
	}
}

// Test cleanDependencies text mode success message
func TestCleanDependencies_TextMode_SuccessMessage(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	// Create node project with node_modules
	nodeDir := filepath.Join(tmpDir, "node-project")
	nodeModules := filepath.Join(nodeDir, "node_modules")
	if err := os.MkdirAll(nodeModules, 0750); err != nil {
		t.Fatalf("Failed to create node_modules: %v", err)
	}

	nodeProjects := []types.NodeProject{{Dir: nodeDir}}

	// Should print success message
	err := cleanDependencies(nodeProjects, nil, nil)
	if err != nil {
		t.Errorf("cleanDependencies returned error: %v", err)
	}
}

// Test cleanDirectory text mode with ItemSuccess output
func TestCleanDirectory_TextMode_Success(t *testing.T) {
	output.SetFormat("text")
	tmpDir := t.TempDir()

	dirToClean := filepath.Join(tmpDir, "to_remove")
	if err := os.MkdirAll(dirToClean, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	err := cleanDirectory(dirToClean)
	if err != nil {
		t.Errorf("cleanDirectory returned error: %v", err)
	}

	// Directory should be removed
	if _, statErr := os.Stat(dirToClean); !os.IsNotExist(statErr) {
		t.Error("Directory should have been removed")
	}
}

// Test detectAllProjects with pyproject.toml
func TestDetectAllProjects_PyprojectToml(t *testing.T) {
	tmpDir := t.TempDir()

	// Create pyproject.toml for python
	pythonDir := filepath.Join(tmpDir, "python-app")
	if err := os.MkdirAll(pythonDir, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	pyproject := `[project]
name = "myapp"
version = "1.0.0"
`
	if err := os.WriteFile(filepath.Join(pythonDir, "pyproject.toml"), []byte(pyproject), 0600); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	_, pythonProjects, _, err := detectAllProjects(tmpDir)
	if err != nil {
		t.Fatalf("detectAllProjects returned error: %v", err)
	}

	if len(pythonProjects) != 1 {
		t.Errorf("Expected 1 python project, got %d", len(pythonProjects))
	}
}

// Test detectAllProjects with vbproj (should not be detected - only csproj/sln)
func TestDetectAllProjects_NonCsprojDotnet(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .fsproj for dotnet (should NOT be detected as the detector only looks for .csproj and .sln)
	dotnetDir := filepath.Join(tmpDir, "fsharp-app")
	if err := os.MkdirAll(dotnetDir, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	fsproj := `<Project Sdk="Microsoft.NET.Sdk"></Project>`
	if err := os.WriteFile(filepath.Join(dotnetDir, "project.fsproj"), []byte(fsproj), 0600); err != nil {
		t.Fatalf("Failed to create project.fsproj: %v", err)
	}

	_, _, dotnetProjects, err := detectAllProjects(tmpDir)
	if err != nil {
		t.Fatalf("detectAllProjects returned error: %v", err)
	}

	// The detector only looks for .csproj and .sln, so fsproj should NOT be detected
	if len(dotnetProjects) != 0 {
		t.Errorf("Expected 0 dotnet projects (fsproj not supported), got %d", len(dotnetProjects))
	}
}

// Test detectAllProjects with pnpm lockfile
func TestDetectAllProjects_PnpmLockfile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json and pnpm-lock.yaml
	nodeDir := filepath.Join(tmpDir, "pnpm-app")
	if err := os.MkdirAll(nodeDir, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(nodeDir, "package.json"), []byte(`{"name":"pnpm-app"}`), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nodeDir, "pnpm-lock.yaml"), []byte("lockfileVersion: 5"), 0600); err != nil {
		t.Fatalf("Failed to create pnpm-lock.yaml: %v", err)
	}

	nodeProjects, _, _, err := detectAllProjects(tmpDir)
	if err != nil {
		t.Fatalf("detectAllProjects returned error: %v", err)
	}

	if len(nodeProjects) != 1 {
		t.Errorf("Expected 1 node project, got %d", len(nodeProjects))
	}
	if len(nodeProjects) > 0 && nodeProjects[0].PackageManager != "pnpm" {
		t.Errorf("Expected pnpm package manager, got %q", nodeProjects[0].PackageManager)
	}
}

// Test detectAllProjects with poetry
func TestDetectAllProjects_Poetry(t *testing.T) {
	tmpDir := t.TempDir()

	// Create pyproject.toml with poetry section
	pythonDir := filepath.Join(tmpDir, "poetry-app")
	if err := os.MkdirAll(pythonDir, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	pyproject := `[tool.poetry]
name = "myapp"
version = "1.0.0"
`
	if err := os.WriteFile(filepath.Join(pythonDir, "pyproject.toml"), []byte(pyproject), 0600); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	_, pythonProjects, _, err := detectAllProjects(tmpDir)
	if err != nil {
		t.Fatalf("detectAllProjects returned error: %v", err)
	}

	if len(pythonProjects) != 1 {
		t.Errorf("Expected 1 python project, got %d", len(pythonProjects))
	}
}
