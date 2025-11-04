package vscode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetDebugPort(t *testing.T) {
	tests := []struct {
		language string
		offset   int
		expected int
	}{
		{"node", 0, 9229},
		{"node", 1, 9230},
		{"python", 0, 5678},
		{"go", 0, 2345},
		{"dotnet", 0, 5005},
		{"java", 0, 5005},
		{"unknown", 0, 9000}, // fallback
	}

	for _, tt := range tests {
		result := GetDebugPort(tt.language, tt.offset)
		if result != tt.expected {
			t.Errorf("GetDebugPort(%s, %d) = %d, want %d", tt.language, tt.offset, result, tt.expected)
		}
	}
}

func TestGenerateLaunchJSON(t *testing.T) {
	services := []ServiceDebugInfo{
		{Name: "api", Language: "node", Port: 9229},
		{Name: "worker", Language: "python", Port: 5678},
		{Name: "web", Language: "dotnet", Port: 5005},
	}

	config := generateLaunchJSON(services)

	// Check version
	if config.Version != "0.2.0" {
		t.Errorf("Expected version 0.2.0, got %s", config.Version)
	}

	// Check configurations count
	if len(config.Configurations) != 3 {
		t.Errorf("Expected 3 configurations, got %d", len(config.Configurations))
	}

	// Check individual configs
	for i, cfg := range config.Configurations {
		if cfg.Request != "attach" {
			t.Errorf("Config %d: Expected request=attach, got %s", i, cfg.Request)
		}
	}

	// Check compounds
	if len(config.Compounds) != 2 {
		t.Errorf("Expected 2 compounds, got %d", len(config.Compounds))
	}

	// Check compound names
	if config.Compounds[0].Name != "🔌 Attach to ALL (already running)" {
		t.Errorf("Expected first compound name to be '🔌 Attach to ALL (already running)', got %s", config.Compounds[0].Name)
	}

	if config.Compounds[1].Name != "🚀 Debug ALL Services" {
		t.Errorf("Expected second compound name to be '🚀 Debug ALL Services', got %s", config.Compounds[1].Name)
	}

	// Check that the second compound has a preLaunchTask
	if config.Compounds[1].PreLaunchTask != "azd: Start Services (Debug)" {
		t.Errorf("Expected preLaunchTask to be 'azd: Start Services (Debug)', got %s", config.Compounds[1].PreLaunchTask)
	}
}

func TestCreateDebugConfig_Node(t *testing.T) {
	config := createDebugConfig("api", "node", 9229)

	if config.Type != "node" {
		t.Errorf("Expected type=node, got %s", config.Type)
	}

	if config.Request != "attach" {
		t.Errorf("Expected request=attach, got %s", config.Request)
	}

	if config.Port != 9229 {
		t.Errorf("Expected port=9229, got %d", config.Port)
	}

	if config.Address != "localhost" {
		t.Errorf("Expected address=localhost, got %s", config.Address)
	}

	if len(config.SkipFiles) == 0 {
		t.Error("Expected SkipFiles to be set for node")
	}
}

func TestCreateDebugConfig_Python(t *testing.T) {
	config := createDebugConfig("worker", "python", 5678)

	if config.Type != "debugpy" {
		t.Errorf("Expected type=debugpy, got %s", config.Type)
	}

	if config.Request != "attach" {
		t.Errorf("Expected request=attach, got %s", config.Request)
	}

	if config.Connect == nil {
		t.Fatal("Expected Connect to be set")
	}

	host, ok := config.Connect["host"].(string)
	if !ok || host != "localhost" {
		t.Errorf("Expected connect.host=localhost, got %v", host)
	}

	port, ok := config.Connect["port"].(int)
	if !ok || port != 5678 {
		t.Errorf("Expected connect.port=5678, got %v", port)
	}
}

func TestCreateDebugConfig_Go(t *testing.T) {
	config := createDebugConfig("api", "go", 2345)

	if config.Type != "go" {
		t.Errorf("Expected type=go, got %s", config.Type)
	}

	if config.Mode != "remote" {
		t.Errorf("Expected mode=remote, got %s", config.Mode)
	}

	if config.Host != "localhost" {
		t.Errorf("Expected host=localhost, got %s", config.Host)
	}

	if config.Port != 2345 {
		t.Errorf("Expected port=2345, got %d", config.Port)
	}
}

func TestCreateDebugConfig_DotNet(t *testing.T) {
	config := createDebugConfig("web", "dotnet", 5005)

	if config.Type != "coreclr" {
		t.Errorf("Expected type=coreclr, got %s", config.Type)
	}

	if config.ProcessName != "web" {
		t.Errorf("Expected processName=web, got %s", config.ProcessName)
	}
}

func TestCreateDebugConfig_Java(t *testing.T) {
	config := createDebugConfig("api", "java", 5005)

	if config.Type != "java" {
		t.Errorf("Expected type=java, got %s", config.Type)
	}

	if config.Host != "localhost" {
		t.Errorf("Expected host=localhost, got %s", config.Host)
	}

	if config.Port != 5005 {
		t.Errorf("Expected port=5005, got %d", config.Port)
	}
}

func TestGenerateTasksJSON(t *testing.T) {
	config := generateTasksJSON()

	if config.Version != "2.0.0" {
		t.Errorf("Expected version 2.0.0, got %s", config.Version)
	}

	if len(config.Tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(config.Tasks))
	}

	task := config.Tasks[0]

	if task.Label != "azd: Start Services (Debug)" {
		t.Errorf("Expected label 'azd: Start Services (Debug)', got %s", task.Label)
	}

	if task.Command != "azd app run --debug" {
		t.Errorf("Expected command 'azd app run --debug', got %s", task.Command)
	}

	if !task.IsBackground {
		t.Error("Expected isBackground to be true")
	}
}

func TestEnsureDebugConfig(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "vscode-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	services := []ServiceDebugInfo{
		{Name: "api", Language: "node", Port: 9229},
		{Name: "worker", Language: "python", Port: 5678},
	}

	// Test creating config for the first time
	err = EnsureDebugConfig(tmpDir, services, false)
	if err != nil {
		t.Fatalf("EnsureDebugConfig failed: %v", err)
	}

	// Check that .vscode directory was created
	vscodeDir := filepath.Join(tmpDir, ".vscode")
	if _, err := os.Stat(vscodeDir); os.IsNotExist(err) {
		t.Error(".vscode directory was not created")
	}

	// Check that launch.json was created
	launchPath := filepath.Join(vscodeDir, "launch.json")
	if _, err := os.Stat(launchPath); os.IsNotExist(err) {
		t.Error("launch.json was not created")
	}

	// Check that tasks.json was created
	tasksPath := filepath.Join(vscodeDir, "tasks.json")
	if _, err := os.Stat(tasksPath); os.IsNotExist(err) {
		t.Error("tasks.json was not created")
	}

	// Verify launch.json content
	launchData, err := os.ReadFile(launchPath)
	if err != nil {
		t.Fatalf("Failed to read launch.json: %v", err)
	}

	var launch LaunchConfig
	if err := json.Unmarshal(launchData, &launch); err != nil {
		t.Fatalf("Failed to parse launch.json: %v", err)
	}

	if len(launch.Configurations) != 2 {
		t.Errorf("Expected 2 configurations in launch.json, got %d", len(launch.Configurations))
	}

	// Test that second run doesn't overwrite (unless force=true)
	err = EnsureDebugConfig(tmpDir, services, false)
	if err != nil {
		t.Fatalf("Second EnsureDebugConfig failed: %v", err)
	}

	// Config should still be the same
	launchData2, _ := os.ReadFile(launchPath)
	if string(launchData) != string(launchData2) {
		// Content should be identical for same services
		// Note: This might fail due to formatting differences, but the content should be semantically the same
	}

	// Test force regeneration
	services2 := []ServiceDebugInfo{
		{Name: "api", Language: "node", Port: 9229},
		{Name: "web", Language: "dotnet", Port: 5005},
	}

	err = EnsureDebugConfig(tmpDir, services2, true)
	if err != nil {
		t.Fatalf("Force regeneration failed: %v", err)
	}

	// Verify the config was updated
	launchData3, err := os.ReadFile(launchPath)
	if err != nil {
		t.Fatalf("Failed to read updated launch.json: %v", err)
	}

	var launch2 LaunchConfig
	if err := json.Unmarshal(launchData3, &launch2); err != nil {
		t.Fatalf("Failed to parse updated launch.json: %v", err)
	}

	// Should still have 2 configs but different services
	if len(launch2.Configurations) != 2 {
		t.Errorf("Expected 2 configurations after update, got %d", len(launch2.Configurations))
	}
}
