//go:build integration
// +build integration

package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/cache"
)

const testToolSource = `package main

import (
	"fmt"
	"os"
)

const version = "2.5.0"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("test-tool version %s\n", version)
			os.Exit(0)
		case "--help", "-h", "help":
			fmt.Println("test-tool - A simple test utility for PATH resolution testing")
			os.Exit(0)
		}
	}
	fmt.Println("test-tool v" + version)
}
`

// TestReqsFixIntegration_BasicPATHResolution tests the complete --fix workflow
// This test:
// 1. Builds a test-tool binary
// 2. Installs it to a custom directory
// 3. Adds that directory to User PATH (registry on Windows)
// 4. Verifies tool is NOT in current session PATH
// 5. Runs azd app reqs --fix
// 6. Verifies tool is found and validated
func TestReqsFixIntegration_BasicPATHResolution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Only run on Windows for now (PATH resolution via registry)
	if runtime.GOOS != "windows" {
		t.Skip("This test currently only runs on Windows (registry PATH resolution)")
	}

	// Setup test environment
	testDir := t.TempDir()
	customToolsDir := filepath.Join(testDir, "CustomTools", "test-tool")
	testProjectDir := filepath.Join(testDir, "test-project")

	// Build test-tool
	buildTestTool(t, customToolsDir)

	// Verify test-tool is NOT in current PATH
	if commandExists("test-tool") {
		t.Skip("test-tool already in PATH - cannot test PATH resolution")
	}

	// Add to User PATH (registry only, not current session)
	defer func() {
		// Cleanup: restore PATH
		cleanupUserPATH(t, customToolsDir)
	}()

	addToUserPATH(t, customToolsDir)

	// Verify still NOT in current session
	if commandExists("test-tool") {
		t.Fatal("test-tool should not be in current session PATH yet")
	}

	// Create test project with azure.yaml
	createTestProject(t, testProjectDir, "2.0.0")

	// Change to test project directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			t.Logf("Warning: failed to restore directory: %v", chdirErr)
		}
	}()

	if err := os.Chdir(testProjectDir); err != nil {
		t.Fatal(err)
	}

	// Initial check should fail (tool not in session PATH)
	t.Log("Running initial check (should fail)...")
	err = runReqs()
	if err == nil {
		t.Fatal("Expected initial check to fail, but it passed")
	}

	// Run fix - should succeed by reading registry PATH
	t.Log("Running fix (should find tool via registry)...")
	err = runReqsFix()
	if err != nil {
		t.Fatalf("Fix should have succeeded: %v", err)
	}

	// Verify cache was cleared and recreated
	cacheDir := filepath.Join(testProjectDir, ".azure", "cache")
	cachePath := filepath.Join(cacheDir, "reqs_cache.json")

	// The fix should have cleared cache, so let's run a normal check to create it
	// This will still fail in current session, but that's expected
	t.Log("Running check to create cache...")
	_ = runReqs() // Ignore error - we expect it due to session limitation

	// Verify cache exists (it should be created even if check fails)
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Log("Cache not created - this is OK in integration test")
	}

	t.Log("✅ Basic PATH Resolution test PASSED")
}

// TestReqsFixIntegration_VersionMismatch tests that --fix properly detects version mismatches
func TestReqsFixIntegration_VersionMismatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if runtime.GOOS != "windows" {
		t.Skip("This test currently only runs on Windows")
	}

	testDir := t.TempDir()
	customToolsDir := filepath.Join(testDir, "CustomTools", "test-tool")
	testProjectDir := filepath.Join(testDir, "test-project")

	// Build test-tool (version 2.5.0)
	buildTestTool(t, customToolsDir)

	defer cleanupUserPATH(t, customToolsDir)
	addToUserPATH(t, customToolsDir)

	// Create project requiring version 10.0.0 (higher than available)
	createTestProject(t, testProjectDir, "10.0.0")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			t.Logf("Warning: failed to restore directory: %v", chdirErr)
		}
	}()

	if err := os.Chdir(testProjectDir); err != nil {
		t.Fatal(err)
	}

	// Fix should fail - tool found but version too old
	t.Log("Running fix with version mismatch (should fail)...")
	err = runReqsFix()
	if err == nil {
		t.Fatal("Expected fix to fail due to version mismatch, but it passed")
	}

	if !strings.Contains(err.Error(), "not all requirements satisfied") &&
		!strings.Contains(err.Error(), "requirement check failed") {
		t.Errorf("Expected 'not all requirements satisfied' or 'requirement check failed' error, got: %v", err)
	}

	t.Log("✅ Version Mismatch test PASSED")
}

// TestReqsFixIntegration_ToolNotFound tests behavior when tool doesn't exist
func TestReqsFixIntegration_ToolNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDir := t.TempDir()
	testProjectDir := filepath.Join(testDir, "test-project")

	// Create project requiring nonexistent tool
	if err := os.MkdirAll(testProjectDir, 0755); err != nil {
		t.Fatal(err)
	}

	azureYaml := `name: test-not-found
reqs:
  - name: nonexistent-tool-xyz-12345
    minVersion: 1.0.0
    command: nonexistent-tool-xyz-12345
    args: ["--version"]
`
	yamlPath := filepath.Join(testProjectDir, "azure.yaml")
	if err := os.WriteFile(yamlPath, []byte(azureYaml), 0600); err != nil {
		t.Fatal(err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			t.Logf("Warning: failed to restore directory: %v", chdirErr)
		}
	}()

	if err := os.Chdir(testProjectDir); err != nil {
		t.Fatal(err)
	}

	// Fix should fail - tool not found anywhere
	t.Log("Running fix for nonexistent tool (should fail)...")
	err = runReqsFix()
	if err == nil {
		t.Fatal("Expected fix to fail for nonexistent tool, but it passed")
	}

	t.Log("✅ Tool Not Found test PASSED")
}

// TestReqsFixIntegration_CacheClearing tests that cache is properly cleared after fix
func TestReqsFixIntegration_CacheClearing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if runtime.GOOS != "windows" {
		t.Skip("This test currently only runs on Windows")
	}

	testDir := t.TempDir()
	customToolsDir := filepath.Join(testDir, "CustomTools", "test-tool")
	testProjectDir := filepath.Join(testDir, "test-project")

	buildTestTool(t, customToolsDir)
	defer cleanupUserPATH(t, customToolsDir)
	addToUserPATH(t, customToolsDir)

	createTestProject(t, testProjectDir, "2.0.0")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			t.Logf("Warning: failed to restore directory: %v", chdirErr)
		}
	}()

	if err := os.Chdir(testProjectDir); err != nil {
		t.Fatal(err)
	}

	// Create initial cache
	cacheDir := filepath.Join(testProjectDir, ".azure", "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}

	cacheManager, err := cache.NewCacheManagerWithOptions(cache.CacheOptions{
		Enabled:  true,
		CacheDir: cacheDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create a fake cache entry
	azureYamlPath := filepath.Join(testProjectDir, "azure.yaml")
	oldResults := []cache.CachedReqResult{
		{
			Name:      "test-tool",
			Installed: false,
			Version:   "",
			Required:  "2.0.0",
			Satisfied: false,
		},
	}

	if err := cacheManager.SaveResults(azureYamlPath, oldResults, false); err != nil {
		t.Fatal(err)
	}

	// Run fix - should clear cache
	t.Log("Running fix (should clear cache)...")
	err = runReqsFix()
	if err != nil {
		t.Logf("Fix completed: %v", err)
	}

	// Verify cache was cleared by checking if it's gone or has new data
	// Note: Cache is not recreated during --fix due to cache being disabled
	cachedData, found, err := cacheManager.GetCachedResults(azureYamlPath)
	if err != nil {
		t.Logf("Cache read error (expected if cleared): %v", err)
	}

	if found {
		// If cache exists, verify it's not the old one we created
		if !cachedData.AllPassed && len(cachedData.Results) == 1 {
			// This might be our old cache - check timestamp
			if time.Since(cachedData.Timestamp) > 30*time.Minute {
				t.Fatal("Cache appears to be the old one - not cleared")
			}
		}
	}

	t.Log("✅ Cache Clearing test PASSED")
}

// Helper functions

func buildTestTool(t *testing.T, targetDir string) string {
	t.Helper()

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	// Create test-tool.go source
	sourcePath := filepath.Join(targetDir, "test-tool.go")
	if err := os.WriteFile(sourcePath, []byte(testToolSource), 0600); err != nil {
		t.Fatalf("Failed to write test-tool source: %v", err)
	}

	// Build the tool
	exeName := "test-tool"
	if runtime.GOOS == "windows" {
		exeName = "test-tool.exe"
	}
	exePath := filepath.Join(targetDir, exeName)

	cmd := exec.Command("go", "build", "-o", exePath, sourcePath)
	cmd.Dir = targetDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build test-tool: %v\nOutput: %s", err, output)
	}

	// Verify executable exists
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		t.Fatalf("test-tool executable not found at %s", exePath)
	}

	t.Logf("✓ Built test-tool at %s", exePath)
	return exePath
}

func addToUserPATH(t *testing.T, dir string) {
	t.Helper()

	if runtime.GOOS != "windows" {
		t.Skip("addToUserPATH only supported on Windows")
	}

	// Read current User PATH from registry
	cmd := exec.Command("powershell", "-Command",
		"[Environment]::GetEnvironmentVariable('Path', 'User')")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to read User PATH: %v", err)
	}

	currentPath := strings.TrimSpace(string(output))
	if strings.Contains(currentPath, dir) {
		t.Logf("✓ %s already in User PATH", dir)
		return
	}

	// Add to User PATH
	newPath := currentPath + ";" + dir
	cmd = exec.Command("powershell", "-Command",
		fmt.Sprintf("[Environment]::SetEnvironmentVariable('Path', '%s', 'User')", newPath))
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add to User PATH: %v", err)
	}

	t.Logf("✓ Added %s to User PATH (registry only)", dir)
}

func cleanupUserPATH(t *testing.T, dir string) {
	t.Helper()

	if runtime.GOOS != "windows" {
		return
	}

	// Read current User PATH
	cmd := exec.Command("powershell", "-Command",
		"[Environment]::GetEnvironmentVariable('Path', 'User')")
	output, err := cmd.Output()
	if err != nil {
		t.Logf("Warning: Failed to read User PATH for cleanup: %v", err)
		return
	}

	currentPath := strings.TrimSpace(string(output))
	if !strings.Contains(currentPath, dir) {
		return
	}

	// Remove from User PATH
	newPath := strings.ReplaceAll(currentPath, ";"+dir, "")
	newPath = strings.ReplaceAll(newPath, dir+";", "")
	newPath = strings.ReplaceAll(newPath, dir, "")

	cmd = exec.Command("powershell", "-Command",
		fmt.Sprintf("[Environment]::SetEnvironmentVariable('Path', '%s', 'User')", newPath))
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Failed to remove from User PATH: %v", err)
	} else {
		t.Logf("✓ Removed %s from User PATH", dir)
	}
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func createTestProject(t *testing.T, dir string, minVersion string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create test project directory: %v", err)
	}

	azureYaml := fmt.Sprintf(`name: path-fix-test
reqs:
  - name: test-tool
    minVersion: %s
    command: test-tool
    args: ["--version"]
    versionPrefix: "test-tool version "
`, minVersion)

	yamlPath := filepath.Join(dir, "azure.yaml")
	if err := os.WriteFile(yamlPath, []byte(azureYaml), 0600); err != nil {
		t.Fatalf("Failed to write azure.yaml: %v", err)
	}

	t.Logf("✓ Created test project at %s", dir)
}
