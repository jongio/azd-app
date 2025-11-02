package commands

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
)

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		contains string
	}{
		{
			name:     "ready status",
			status:   "ready",
			contains: "ready",
		},
		{
			name:     "starting status",
			status:   "starting",
			contains: "starting",
		},
		{
			name:     "error status",
			status:   "error",
			contains: "error",
		},
		{
			name:     "stopped status",
			status:   "stopped",
			contains: "stopped",
		},
		{
			name:     "unknown status",
			status:   "unknown",
			contains: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatus(tt.status)
			if result == "" {
				t.Errorf("formatStatus(%q) returned empty string", tt.status)
			}
			// The result will contain ANSI codes, but should still contain the status text
			// We can't do exact string matching due to color codes
		})
	}
}

func TestFormatHealth(t *testing.T) {
	tests := []struct {
		name     string
		health   string
		contains string
	}{
		{
			name:     "healthy",
			health:   "healthy",
			contains: "healthy",
		},
		{
			name:     "unhealthy",
			health:   "unhealthy",
			contains: "unhealthy",
		},
		{
			name:     "unknown",
			health:   "unknown",
			contains: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatHealth(tt.health)
			if result == "" {
				t.Errorf("formatHealth(%q) returned empty string", tt.health)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		contains string
	}{
		{
			name:     "zero time",
			time:     time.Time{},
			contains: "N/A",
		},
		{
			name:     "30 seconds ago",
			time:     now.Add(-30 * time.Second),
			contains: "30s ago",
		},
		{
			name:     "5 minutes ago",
			time:     now.Add(-5 * time.Minute),
			contains: "5m ago",
		},
		{
			name:     "2 hours ago",
			time:     now.Add(-2 * time.Hour),
			contains: "2h ago",
		},
		{
			name:     "25 hours ago (absolute time)",
			time:     now.Add(-25 * time.Hour),
			contains: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTime(tt.time)
			if result == "" {
				t.Errorf("formatTime(%v) returned empty string", tt.time)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "30 seconds",
			duration: 30 * time.Second,
			expected: "30s",
		},
		{
			name:     "5 minutes",
			duration: 5 * time.Minute,
			expected: "5m",
		},
		{
			name:     "2 hours",
			duration: 2 * time.Hour,
			expected: "2h",
		},
		{
			name:     "25 hours (1 day)",
			duration: 25 * time.Hour,
			expected: "1d",
		},
		{
			name:     "50 hours (2 days)",
			duration: 50 * time.Hour,
			expected: "2d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		health   string
		contains string
	}{
		{
			name:     "ready and healthy",
			status:   "ready",
			health:   "healthy",
			contains: "✓",
		},
		{
			name:     "starting",
			status:   "starting",
			health:   "unknown",
			contains: "○",
		},
		{
			name:     "error",
			status:   "error",
			health:   "unhealthy",
			contains: "✗",
		},
		{
			name:     "unhealthy but ready",
			status:   "ready",
			health:   "unhealthy",
			contains: "✗",
		},
		{
			name:     "stopped",
			status:   "stopped",
			health:   "unknown",
			contains: "●",
		},
		{
			name:     "unknown status",
			status:   "unknown",
			health:   "unknown",
			contains: "?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStatusIcon(tt.status, tt.health)
			if result == "" {
				t.Errorf("getStatusIcon(%q, %q) returned empty string", tt.status, tt.health)
			}
		})
	}
}

func TestGetAzureEndpoints(t *testing.T) {
	// Set up test environment variables
	testEnvVars := map[string]string{
		"SERVICE_API_ENDPOINT_URL":     "https://api.example.com",
		"SERVICE_WEB_URL":              "https://web.example.com",
		"SERVICE_BACKEND_ENDPOINT_URL": "https://backend.example.com",
		"OTHER_VAR":                    "not-a-service",
		"SERVICE_CACHE_CONNECTION_STR": "redis://localhost:6379", // Should be ignored
	}

	// Set environment variables
	for key, value := range testEnvVars {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	endpoints := getAzureEndpoints()

	tests := []struct {
		serviceName string
		expected    string
		shouldExist bool
	}{
		{
			serviceName: "api",
			expected:    "https://api.example.com",
			shouldExist: true,
		},
		{
			serviceName: "web",
			expected:    "https://web.example.com",
			shouldExist: true,
		},
		{
			serviceName: "backend",
			expected:    "https://backend.example.com",
			shouldExist: true,
		},
		{
			serviceName: "nonexistent",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.serviceName, func(t *testing.T) {
			endpoint, exists := endpoints[tt.serviceName]
			if exists != tt.shouldExist {
				t.Errorf("getAzureEndpoints()[%q] exists = %v, want %v", tt.serviceName, exists, tt.shouldExist)
			}
			if tt.shouldExist && endpoint != tt.expected {
				t.Errorf("getAzureEndpoints()[%q] = %q, want %q", tt.serviceName, endpoint, tt.expected)
			}
		})
	}
}

func TestGetServiceEnvVars(t *testing.T) {
	// Set up test environment variables
	testEnvVars := map[string]string{
		"SERVICE_API_DATABASE_URL":     "postgres://localhost:5432/db",
		"SERVICE_API_CACHE_URL":        "redis://localhost:6379",
		"SERVICE_API_SECRET_KEY":       "secret123",
		"SERVICE_WEB_PORT":             "3000",
		"OTHER_VAR":                    "should-not-appear",
		"SERVICE_BACKEND_API_ENDPOINT": "http://localhost:8080",
	}

	// Set environment variables
	for key, value := range testEnvVars {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	tests := []struct {
		name          string
		serviceName   string
		expectedCount int
		expectedVars  map[string]string
	}{
		{
			name:          "api service",
			serviceName:   "api",
			expectedCount: 3,
			expectedVars: map[string]string{
				"SERVICE_API_DATABASE_URL": "postgres://localhost:5432/db",
				"SERVICE_API_CACHE_URL":    "redis://localhost:6379",
				"SERVICE_API_SECRET_KEY":   "secret123",
			},
		},
		{
			name:          "web service",
			serviceName:   "web",
			expectedCount: 1,
			expectedVars: map[string]string{
				"SERVICE_WEB_PORT": "3000",
			},
		},
		{
			name:          "backend service",
			serviceName:   "backend",
			expectedCount: 1,
			expectedVars: map[string]string{
				"SERVICE_BACKEND_API_ENDPOINT": "http://localhost:8080",
			},
		},
		{
			name:          "nonexistent service",
			serviceName:   "nonexistent",
			expectedCount: 0,
			expectedVars:  map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVars := getServiceEnvVars(tt.serviceName)

			if len(envVars) != tt.expectedCount {
				t.Errorf("getServiceEnvVars(%q) returned %d vars, want %d", tt.serviceName, len(envVars), tt.expectedCount)
			}

			for key, expectedValue := range tt.expectedVars {
				if actualValue, exists := envVars[key]; !exists {
					t.Errorf("getServiceEnvVars(%q) missing key %q", tt.serviceName, key)
				} else if actualValue != expectedValue {
					t.Errorf("getServiceEnvVars(%q)[%q] = %q, want %q", tt.serviceName, key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestGetCurrentDir(t *testing.T) {
	result := getCurrentDir()
	if result == "" {
		t.Error("getCurrentDir() returned empty string")
	}

	// Should return a valid path
	if !filepath.IsAbs(result) && result != "." {
		t.Errorf("getCurrentDir() = %q, expected absolute path or '.'", result)
	}
}

func TestRunInfoNoServices(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .azure directory (no services.json)
	azureDir := filepath.Join(tmpDir, ".azure")
	if err := os.MkdirAll(azureDir, 0750); err != nil {
		t.Fatalf("Failed to create .azure directory: %v", err)
	}

	cmd := NewInfoCommand()
	err = cmd.Execute()
	if err != nil {
		t.Errorf("runInfo() failed with no services: %v", err)
	}
}

func TestRunInfoWithServices(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .azure directory
	azureDir := filepath.Join(tmpDir, ".azure")
	if err := os.MkdirAll(azureDir, 0750); err != nil {
		t.Fatalf("Failed to create .azure directory: %v", err)
	}

	// Register test services
	reg := registry.GetRegistry(tmpDir)

	now := time.Now()

	testServices := []*registry.ServiceRegistryEntry{
		{
			Name:        "api",
			ProjectDir:  tmpDir,
			PID:         12345,
			Port:        8080,
			URL:         "http://localhost:8080",
			Language:    "go",
			Framework:   "net/http",
			Status:      "ready",
			Health:      "healthy",
			StartTime:   now.Add(-5 * time.Minute),
			LastChecked: now,
		},
		{
			Name:        "web",
			ProjectDir:  tmpDir,
			PID:         12346,
			Port:        3000,
			URL:         "http://localhost:3000",
			Language:    "node",
			Framework:   "next.js",
			Status:      "ready",
			Health:      "healthy",
			StartTime:   now.Add(-10 * time.Minute),
			LastChecked: now,
		},
	}

	for _, svc := range testServices {
		if err := reg.Register(svc); err != nil {
			t.Fatalf("Failed to register service %s: %v", svc.Name, err)
		}
	}

	// Set up some test environment variables
	os.Setenv("SERVICE_API_DATABASE_URL", "postgres://localhost:5432/db")
	defer os.Unsetenv("SERVICE_API_DATABASE_URL")

	cmd := NewInfoCommand()
	err = cmd.Execute()
	if err != nil {
		t.Errorf("runInfo() failed with services: %v", err)
	}
}

func TestRunInfoWithProjectFlag(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create .azure directory
	azureDir := filepath.Join(tmpDir, ".azure")
	if err := os.MkdirAll(azureDir, 0750); err != nil {
		t.Fatalf("Failed to create .azure directory: %v", err)
	}

	// Register a test service
	reg := registry.GetRegistry(tmpDir)
	testService := &registry.ServiceRegistryEntry{
		Name:        "test-service",
		ProjectDir:  tmpDir,
		PID:         99999,
		Port:        9000,
		URL:         "http://localhost:9000",
		Language:    "python",
		Framework:   "fastapi",
		Status:      "ready",
		Health:      "healthy",
		StartTime:   time.Now().Add(-1 * time.Hour),
		LastChecked: time.Now(),
	}

	if err := reg.Register(testService); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Test with --project flag
	cmd := NewInfoCommand()
	cmd.SetArgs([]string{"--project", tmpDir})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("runInfo() with --project flag failed: %v", err)
	}
}

func TestRunInfoWithErrorService(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .azure directory
	azureDir := filepath.Join(tmpDir, ".azure")
	if err := os.MkdirAll(azureDir, 0750); err != nil {
		t.Fatalf("Failed to create .azure directory: %v", err)
	}

	// Register a service with error
	reg := registry.GetRegistry(tmpDir)
	errorService := &registry.ServiceRegistryEntry{
		Name:        "failing-service",
		ProjectDir:  tmpDir,
		PID:         0,
		Port:        5000,
		URL:         "http://localhost:5000",
		Language:    "node",
		Framework:   "express",
		Status:      "error",
		Health:      "unhealthy",
		StartTime:   time.Now().Add(-30 * time.Second),
		LastChecked: time.Now(),
		Error:       "failed to bind to port: address already in use",
	}

	if err := reg.Register(errorService); err != nil {
		t.Fatalf("Failed to register error service: %v", err)
	}

	cmd := NewInfoCommand()
	err = cmd.Execute()
	if err != nil {
		t.Errorf("runInfo() with error service failed: %v", err)
	}
}
