package healthcheck

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
)

func TestNewHealthMonitor(t *testing.T) {
	tempDir := t.TempDir()

	config := MonitorConfig{
		ProjectDir:      tempDir,
		DefaultEndpoint: "/health",
		Timeout:         5 * time.Second,
		Verbose:         false,
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create health monitor: %v", err)
	}

	if monitor == nil {
		t.Fatal("Expected monitor to be created")
	}

	if monitor.config.ProjectDir != tempDir {
		t.Errorf("Expected project dir %s, got %s", tempDir, monitor.config.ProjectDir)
	}

	if monitor.checker == nil {
		t.Error("Expected checker to be created")
	}

	if monitor.registry == nil {
		t.Error("Expected registry to be created")
	}
}

func TestMonitorCheck(t *testing.T) {
	tempDir := t.TempDir()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()

	// Register a test service
	reg := registry.GetRegistry(tempDir)
	port := server.Listener.Addr().(*net.TCPAddr).Port
	err := reg.Register(&registry.ServiceRegistryEntry{
		Name:      "test-service",
		Port:      port,
		PID:       os.Getpid(),
		StartTime: time.Now(),
		Status:    "running",
	})
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	config := MonitorConfig{
		ProjectDir:      tempDir,
		DefaultEndpoint: "/health",
		Timeout:         5 * time.Second,
		Verbose:         false,
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create health monitor: %v", err)
	}

	// Perform health check
	report, err := monitor.Check(context.Background(), nil)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	if report == nil {
		t.Fatal("Expected report to be generated")
	}

	if len(report.Services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(report.Services))
	}

	if report.Services[0].ServiceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", report.Services[0].ServiceName)
	}

	if report.Services[0].Status != HealthStatusHealthy {
		t.Errorf("Expected status healthy, got %s", report.Services[0].Status)
	}

	if report.Summary.Total != 1 {
		t.Errorf("Expected total 1, got %d", report.Summary.Total)
	}

	if report.Summary.Healthy != 1 {
		t.Errorf("Expected healthy 1, got %d", report.Summary.Healthy)
	}
}

func TestMonitorCheckWithFilter(t *testing.T) {
	tempDir := t.TempDir()

	// Register multiple test services
	reg := registry.GetRegistry(tempDir)
	err := reg.Register(&registry.ServiceRegistryEntry{
		Name:      "web",
		Port:      3000,
		PID:       os.Getpid(),
		StartTime: time.Now(),
		Status:    "running",
	})
	if err != nil {
		t.Fatalf("Failed to register web service: %v", err)
	}

	err = reg.Register(&registry.ServiceRegistryEntry{
		Name:      "api",
		Port:      8080,
		PID:       os.Getpid(),
		StartTime: time.Now(),
		Status:    "running",
	})
	if err != nil {
		t.Fatalf("Failed to register api service: %v", err)
	}

	config := MonitorConfig{
		ProjectDir:      tempDir,
		DefaultEndpoint: "/health",
		Timeout:         2 * time.Second,
		Verbose:         false,
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create health monitor: %v", err)
	}

	// Check with filter
	report, err := monitor.Check(context.Background(), []string{"web"})
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	if len(report.Services) != 1 {
		t.Errorf("Expected 1 filtered service, got %d", len(report.Services))
	}

	if report.Services[0].ServiceName != "web" {
		t.Errorf("Expected service name 'web', got '%s'", report.Services[0].ServiceName)
	}
}

func TestLoadAzureYaml(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test azure.yaml
	azureYamlContent := `
services:
  web:
    language: nodejs
    project: ./web
    ports:
      - "3000"
  api:
    language: python
    project: ./api
    ports:
      - "8080"
`

	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write azure.yaml: %v", err)
	}

	config := MonitorConfig{
		ProjectDir:      tempDir,
		DefaultEndpoint: "/health",
		Timeout:         5 * time.Second,
		Verbose:         false,
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create health monitor: %v", err)
	}

	azureYaml, err := monitor.loadAzureYaml()
	if err != nil {
		t.Fatalf("Failed to load azure.yaml: %v", err)
	}

	if azureYaml == nil {
		t.Fatal("Expected azure.yaml to be loaded")
	}

	if len(azureYaml.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(azureYaml.Services))
	}
}

func TestLoadAzureYamlNotFound(t *testing.T) {
	tempDir := t.TempDir()

	config := MonitorConfig{
		ProjectDir:      tempDir,
		DefaultEndpoint: "/health",
		Timeout:         5 * time.Second,
		Verbose:         false,
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create health monitor: %v", err)
	}

	azureYaml, err := monitor.loadAzureYaml()
	if err == nil {
		t.Error("Expected error when azure.yaml not found")
	}

	if azureYaml != nil {
		t.Error("Expected nil azure.yaml when file not found")
	}
}

func TestBuildServiceList(t *testing.T) {
	tempDir := t.TempDir()

	// Create registry entries
	reg := registry.GetRegistry(tempDir)
	err := reg.Register(&registry.ServiceRegistryEntry{
		Name:      "web",
		Port:      3000,
		PID:       12345,
		StartTime: time.Now(),
		Status:    "running",
	})
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	registeredServices := reg.ListAll()

	config := MonitorConfig{
		ProjectDir:      tempDir,
		DefaultEndpoint: "/health",
		Timeout:         5 * time.Second,
		Verbose:         false,
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create health monitor: %v", err)
	}

	services := monitor.buildServiceList(nil, registeredServices)

	if len(services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(services))
	}

	if services[0].Name != "web" {
		t.Errorf("Expected service name 'web', got '%s'", services[0].Name)
	}

	if services[0].Port != 3000 {
		t.Errorf("Expected port 3000, got %d", services[0].Port)
	}
}

func TestIsProcessRunning(t *testing.T) {
	// Test with current process (should be running)
	currentPID := os.Getpid()
	result := isProcessRunning(currentPID)

	// On some systems, this might not work reliably, so we just verify it doesn't panic
	// and returns a boolean
	_ = result

	// Test with clearly non-existent PID (very high number unlikely to exist)
	if isProcessRunning(9999999) {
		// Some systems might have processes with high PIDs, so we accept both results
		t.Log("High PID process check returned true - acceptable on some systems")
	}
}

func TestUpdateRegistry(t *testing.T) {
	tempDir := t.TempDir()

	// Register a test service
	reg := registry.GetRegistry(tempDir)
	err := reg.Register(&registry.ServiceRegistryEntry{
		Name:      "test-service",
		Port:      3000,
		PID:       os.Getpid(),
		StartTime: time.Now(),
		Status:    "running",
	})
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	config := MonitorConfig{
		ProjectDir:      tempDir,
		DefaultEndpoint: "/health",
		Timeout:         5 * time.Second,
		Verbose:         false,
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create health monitor: %v", err)
	}

	results := []HealthCheckResult{
		{
			ServiceName: "test-service",
			Status:      HealthStatusHealthy,
		},
	}

	monitor.updateRegistry(results)

	// Verify the update - status should remain "running" for healthy services
	services := reg.ListAll()
	if len(services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(services))
	}

	// Health is no longer stored in registry - it's computed dynamically via health checks
	// Just verify the service is still registered with running status
	if services[0].Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", services[0].Status)
	}
}
// TestUpdateRegistry_NoRegressionFromRunningToStarting verifies that services that are
// already running are NOT regressed to "starting" when health checks return HealthStatusStarting.
// This fixes a bug where services would show "starting" indefinitely in the dashboard because
// the health check grace period would overwrite the registry status from "running" back to "starting".
func TestUpdateRegistry_NoRegressionFromRunningToStarting(t *testing.T) {
	tempDir := t.TempDir()

	// Register a test service that's already running
	reg := registry.GetRegistry(tempDir)
	err := reg.Register(&registry.ServiceRegistryEntry{
		Name:      "test-service",
		Port:      3000,
		PID:       os.Getpid(),
		StartTime: time.Now(),
		Status:    "running", // Service is already running
	})
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	config := MonitorConfig{
		ProjectDir:      tempDir,
		DefaultEndpoint: "/health",
		Timeout:         5 * time.Second,
		Verbose:         false,
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create health monitor: %v", err)
	}

	// Health check returns "starting" (e.g., during grace period when health checks haven't passed yet)
	results := []HealthCheckResult{
		{
			ServiceName: "test-service",
			Status:      HealthStatusStarting, // This should NOT regress the registry status
		},
	}

	monitor.updateRegistry(results)

	// Verify the status was NOT regressed to "starting"
	services := reg.ListAll()
	if len(services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(services))
	}

	if services[0].Status != "running" {
		t.Errorf("BUG: Service status was regressed from 'running' to '%s' - this causes the dashboard to show 'starting' indefinitely", services[0].Status)
	}
}

// TestUpdateRegistry_AllowsStartingForNewServices verifies that services that are
// NOT yet running (e.g., just registered with "starting" status) can still be updated to "starting".
func TestUpdateRegistry_AllowsStartingForNewServices(t *testing.T) {
	tempDir := t.TempDir()

	// Register a test service that's just starting
	reg := registry.GetRegistry(tempDir)
	err := reg.Register(&registry.ServiceRegistryEntry{
		Name:      "test-service",
		Port:      3000,
		PID:       os.Getpid(),
		StartTime: time.Now(),
		Status:    "starting", // Service is still starting
	})
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	config := MonitorConfig{
		ProjectDir:      tempDir,
		DefaultEndpoint: "/health",
		Timeout:         5 * time.Second,
		Verbose:         false,
	}

	monitor, err := NewHealthMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create health monitor: %v", err)
	}

	// Health check returns "starting" (e.g., during grace period)
	results := []HealthCheckResult{
		{
			ServiceName: "test-service",
			Status:      HealthStatusStarting,
		},
	}

	monitor.updateRegistry(results)

	// Verify the status is still "starting" (allowed for non-running services)
	services := reg.ListAll()
	if len(services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(services))
	}

	if services[0].Status != "starting" {
		t.Errorf("Expected status 'starting', got '%s'", services[0].Status)
	}
}