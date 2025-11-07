package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/cache"
	"github.com/jongio/azd-app/cli/src/internal/output"
)

func TestSetCacheEnabled(t *testing.T) {
	// Save original state
	original := execContext.CacheEnabled

	defer func() {
		execContext.CacheEnabled = original
	}()

	tests := []struct {
		name    string
		enabled bool
	}{
		{"enable cache", true},
		{"disable cache", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetCacheEnabled(tt.enabled)
			if execContext.CacheEnabled != tt.enabled {
				t.Errorf("SetCacheEnabled(%v) failed, got %v", tt.enabled, execContext.CacheEnabled)
			}
		})
	}
}

func TestCreateCacheManager(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled cache", true},
		{"disabled cache", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := createCacheManager(tt.enabled)
			if cm == nil {
				t.Fatal("createCacheManager returned nil")
			}
			if cm.IsEnabled() != tt.enabled {
				t.Errorf("cache enabled = %v, want %v", cm.IsEnabled(), tt.enabled)
			}
		})
	}
}

func TestNewDependencyInstaller(t *testing.T) {
	searchRoot := "/test/path"
	di := NewDependencyInstaller(searchRoot)

	if di == nil {
		t.Fatal("NewDependencyInstaller returned nil")
	}
	if di.searchRoot != searchRoot {
		t.Errorf("searchRoot = %q, want %q", di.searchRoot, searchRoot)
	}
}

func TestCheckAllSuccess(t *testing.T) {
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
			},
			want: true,
		},
		{
			name: "one failure",
			results: []InstallResult{
				{Success: true},
				{Success: false},
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

func TestGetSearchRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that requires file system access")
	}

	// Test 1: Current directory (no azure.yaml)
	got, err := getSearchRoot()
	if err != nil {
		t.Fatalf("getSearchRoot() error = %v", err)
	}
	if got == "" {
		t.Error("getSearchRoot() returned empty string")
	}
}

func TestExecuteRun(t *testing.T) {
	// Set JSON mode to avoid console output
	_ = output.SetFormat("json")
	defer func() { _ = output.SetFormat("default") }()

	err := executeRun()
	if err != nil {
		t.Errorf("executeRun() error = %v", err)
	}
}

func TestConvertCachedResults(t *testing.T) {
	cached := []cache.CachedReqResult{
		{
			Name:       "node",
			Installed:  true,
			Version:    "20.0.0",
			Required:   "18.0.0",
			Satisfied:  true,
			Running:    false,
			CheckedRun: false,
			Message:    "test message",
		},
	}

	results := convertCachedResults(cached)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Name != "node" {
		t.Errorf("Name = %q, want %q", r.Name, "node")
	}
	if r.Version != "20.0.0" {
		t.Errorf("Version = %q, want %q", r.Version, "20.0.0")
	}
	if !r.Satisfied {
		t.Error("Satisfied should be true")
	}
}

func TestPerformReqsCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that checks for installed tools")
	}

	reqs := []Prerequisite{
		{
			Name:       "git",
			MinVersion: "2.0.0",
		},
	}

	results, allSatisfied := performReqsCheck(reqs)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Name != "git" {
		t.Errorf("Name = %q, want %q", results[0].Name, "git")
	}

	// Git should typically be installed in dev environments
	t.Logf("git check: installed=%v, satisfied=%v, all=%v",
		results[0].Installed, results[0].Satisfied, allSatisfied)
}

func TestNewResultFormatter(t *testing.T) {
	formatter := NewResultFormatter()
	if formatter == nil {
		t.Fatal("NewResultFormatter returned nil")
	}
}

func TestResultFormatterPrintAll(t *testing.T) {
	// Set JSON mode to suppress console output during test
	_ = output.SetFormat("json")
	defer func() { _ = output.SetFormat("default") }()

	formatter := NewResultFormatter()
	results := []ReqResult{
		{
			Name:      "test",
			Installed: true,
			Version:   "1.0.0",
			Required:  "1.0.0",
			Satisfied: true,
		},
	}

	// Should not panic
	formatter.PrintAll(results)
}

func TestResultFormatterPrint(t *testing.T) {
	// Set JSON mode to suppress console output
	_ = output.SetFormat("json")
	defer func() { _ = output.SetFormat("default") }()

	formatter := NewResultFormatter()

	tests := []struct {
		name   string
		result ReqResult
	}{
		{
			name: "not installed",
			result: ReqResult{
				Name:      "node",
				Installed: false,
				Required:  "18.0.0",
			},
		},
		{
			name: "installed with unknown version",
			result: ReqResult{
				Name:      "npm",
				Installed: true,
				Version:   "",
				Required:  "8.0.0",
			},
		},
		{
			name: "satisfied",
			result: ReqResult{
				Name:      "git",
				Installed: true,
				Version:   "2.40.0",
				Required:  "2.0.0",
				Satisfied: true,
			},
		},
		{
			name: "not satisfied",
			result: ReqResult{
				Name:       "python",
				Installed:  true,
				Version:    "3.8.0",
				Required:   "3.9.0",
				Satisfied:  false,
				CheckedRun: false,
			},
		},
		{
			name: "running check - running",
			result: ReqResult{
				Name:       "docker",
				Installed:  true,
				Version:    "24.0.0",
				Required:   "20.0.0",
				Satisfied:  true,
				CheckedRun: true,
				Running:    true,
			},
		},
		{
			name: "running check - not running",
			result: ReqResult{
				Name:       "docker",
				Installed:  true,
				Version:    "24.0.0",
				Required:   "20.0.0",
				Satisfied:  true,
				CheckedRun: true,
				Running:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			formatter.Print(tt.result)
		})
	}
}

func TestResultFormatterPrintRunningStatus(t *testing.T) {
	// Set JSON mode to suppress console output
	_ = output.SetFormat("json")
	defer func() { _ = output.SetFormat("default") }()

	formatter := NewResultFormatter()

	// Should not panic
	formatter.printRunningStatus(true)
	formatter.printRunningStatus(false)
}

func TestInstallProject(t *testing.T) {
	di := NewDependencyInstaller("/test")

	// Test successful install
	result := di.installProject("node", "/test/dir", "npm", func() error {
		return nil
	})

	if !result.Success {
		t.Error("expected success")
	}
	if result.Type != "node" {
		t.Errorf("Type = %q, want %q", result.Type, "node")
	}
	if result.Dir != "/test/dir" {
		t.Errorf("Dir = %q, want %q", result.Dir, "/test/dir")
	}
	if result.Manager != "npm" {
		t.Errorf("Manager = %q, want %q", result.Manager, "npm")
	}

	// Test failed install (will suppress output in JSON mode)
	_ = output.SetFormat("json")
	defer func() { _ = output.SetFormat("default") }()

	failResult := di.installProject("python", "/test/dir2", "pip", func() error {
		return os.ErrNotExist
	})

	if failResult.Success {
		t.Error("expected failure")
	}
	if failResult.Error == "" {
		t.Error("expected error message")
	}
}

func TestSaveToCache(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that writes to file system")
	}

	// Create temp directory for cache
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	// Create a simple azure.yaml
	err := os.WriteFile(azureYamlPath, []byte("name: test\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create cache manager
	cacheManager, err := cache.NewCacheManagerWithOptions(cache.CacheOptions{
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("failed to create cache manager: %v", err)
	}

	results := []ReqResult{
		{
			Name:      "git",
			Installed: true,
			Version:   "2.40.0",
			Satisfied: true,
		},
	}

	// Should not panic
	saveToCache(azureYamlPath, results, true, cacheManager)
}
