package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestGetConfigPath(t *testing.T) {
	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() error = %v", err)
	}

	if path == "" {
		t.Error("GetConfigPath() returned empty path")
	}

	// Should end with .azd/config.json
	if !endsWithPath(path, ".azd", "config.json") {
		t.Errorf("GetConfigPath() = %q, should end with .azd/config.json", path)
	}
}

func TestLoadSaveConfig(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".azd", "config.json")

	// Override GetConfigPath for testing
	originalGetConfigPath := GetConfigPath
	GetConfigPath = func() (string, error) {
		return configPath, nil
	}
	defer func() {
		GetConfigPath = originalGetConfigPath
	}()

	// Load non-existent config (should return empty)
	config, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config == nil {
		t.Fatal("Load() returned nil config")
	}

	// Set a value
	config.App = &AppConfig{
		Dashboard: &DashboardConfig{
			Browser: "system",
		},
	}

	// Save config
	if err := Save(config); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load again and verify
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() after Save error = %v", err)
	}

	if loaded.App == nil || loaded.App.Dashboard == nil {
		t.Fatal("Loaded config missing dashboard settings")
	}

	if loaded.App.Dashboard.Browser != "system" {
		t.Errorf("Browser = %q, want %q", loaded.App.Dashboard.Browser, "system")
	}
}

func TestGetSetUnset(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".azd", "config.json")

	// Override GetConfigPath for testing
	originalGetConfigPath := GetConfigPath
	GetConfigPath = func() (string, error) {
		return configPath, nil
	}
	defer func() {
		GetConfigPath = originalGetConfigPath
	}()

	// Reset global config for test
	globalConfig = nil
	globalConfigOnce = sync.Once{}

	// Get when not set (should return empty)
	value, err := Get("app.dashboard.browser")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if value != "" {
		t.Errorf("Get() = %q, want empty string", value)
	}

	// Set value
	if err := Set("app.dashboard.browser", "system"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get value (need to reload global config)
	globalConfig = nil
	globalConfigOnce = sync.Once{}

	value, err = Get("app.dashboard.browser")
	if err != nil {
		t.Fatalf("Get() after Set error = %v", err)
	}
	if value != "system" {
		t.Errorf("Get() = %q, want %q", value, "system")
	}

	// Unset value
	if err := Unset("app.dashboard.browser"); err != nil {
		t.Fatalf("Unset() error = %v", err)
	}

	// Get after unset (need to reload)
	globalConfig = nil
	globalConfigOnce = sync.Once{}

	value, err = Get("app.dashboard.browser")
	if err != nil {
		t.Fatalf("Get() after Unset error = %v", err)
	}
	if value != "" {
		t.Errorf("Get() after Unset = %q, want empty string", value)
	}
}

func TestGetSetUnsetInvalidKey(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".azd", "config.json")

	// Override GetConfigPath for testing
	originalGetConfigPath := GetConfigPath
	GetConfigPath = func() (string, error) {
		return configPath, nil
	}
	defer func() {
		GetConfigPath = originalGetConfigPath
	}()

	// Reset global config for test
	globalConfig = nil
	globalConfigOnce = sync.Once{}

	// Get invalid key
	_, err := Get("invalid.key")
	if err == nil {
		t.Error("Get() with invalid key should return error")
	}

	// Set invalid key
	err = Set("invalid.key", "value")
	if err == nil {
		t.Error("Set() with invalid key should return error")
	}

	// Unset invalid key
	err = Unset("invalid.key")
	if err == nil {
		t.Error("Unset() with invalid key should return error")
	}
}

func TestDashboardBrowserHelpers(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".azd", "config.json")

	// Override GetConfigPath for testing
	originalGetConfigPath := GetConfigPath
	GetConfigPath = func() (string, error) {
		return configPath, nil
	}
	defer func() {
		GetConfigPath = originalGetConfigPath
	}()

	// Reset global config for test
	globalConfig = nil
	globalConfigOnce = sync.Once{}

	// Get when not set
	value := GetDashboardBrowser()
	if value != "" {
		t.Errorf("GetDashboardBrowser() = %q, want empty string", value)
	}

	// Set value
	if err := SetDashboardBrowser("system"); err != nil {
		t.Fatalf("SetDashboardBrowser() error = %v", err)
	}

	// Get value (need to reload)
	globalConfig = nil
	globalConfigOnce = sync.Once{}

	value = GetDashboardBrowser()
	if value != "system" {
		t.Errorf("GetDashboardBrowser() = %q, want %q", value, "system")
	}

	// Unset value
	if err := UnsetDashboardBrowser(); err != nil {
		t.Fatalf("UnsetDashboardBrowser() error = %v", err)
	}

	// Get after unset (need to reload)
	globalConfig = nil
	globalConfigOnce = sync.Once{}

	value = GetDashboardBrowser()
	if value != "" {
		t.Errorf("GetDashboardBrowser() after unset = %q, want empty string", value)
	}
}

// Helper functions

func endsWithPath(fullPath string, parts ...string) bool {
	for i := len(parts) - 1; i >= 0; i-- {
		if !endsWith(fullPath, parts[i]) {
			return false
		}
		fullPath = filepath.Dir(fullPath)
	}
	return true
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
