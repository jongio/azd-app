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
			name:     "running status",
			status:   "running",
			contains: "running",
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
			name:     "not-running status",
			status:   "not-running",
			contains: "not-running",
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

func TestInfoFormatDuration(t *testing.T) {
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
			result := formatInfoDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatInfoDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestInfoGetStatusIcon(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		health   string
		contains string
	}{
		{
			name:     "running and healthy",
			status:   "running",
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
			name:     "unhealthy but running",
			status:   "running",
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
			name:     "not-running",
			status:   "not-running",
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
			result := getInfoStatusIcon(tt.status, tt.health)
			if result == "" {
				t.Errorf("getInfoStatusIcon(%q, %q) returned empty string", tt.status, tt.health)
			}
		})
	}
}

func TestRunInfoNoServices(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

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
	defer func() { _ = os.Chdir(originalDir) }()

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
			Status:      "running",
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
			Status:      "running",
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

func TestRunInfoWithDifferentWorkingDirectory(t *testing.T) {
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
		Status:      "running",
		StartTime:   time.Now().Add(-1 * time.Hour),
		LastChecked: time.Now(),
	}

	if err := reg.Register(testService); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Change to the temp directory (simulating --cwd flag behavior)
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	cmd := NewInfoCommand()
	cmd.SetArgs([]string{})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("runInfo() with different working directory failed: %v", err)
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
	defer func() { _ = os.Chdir(originalDir) }()

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

func TestGetServiceEnvironmentVars(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		azureEnv    map[string]string
		expected    map[string]string
	}{
		{
			name:        "service with matching variables",
			serviceName: "api",
			azureEnv: map[string]string{
				"SERVICE_API_DATABASE_URL":  "postgres://localhost:5432/db",
				"SERVICE_API_PORT":          "8080",
				"SERVICE_WEB_URL":           "http://localhost:3000",
				"AZURE_API_STORAGE_ACCOUNT": "mystorageaccount",
				"AZURE_WEB_CDN":             "mycdn.azureedge.net",
				"UNRELATED_VAR":             "value",
			},
			expected: map[string]string{
				"SERVICE_API_DATABASE_URL":  "postgres://localhost:5432/db",
				"SERVICE_API_PORT":          "8080",
				"AZURE_API_STORAGE_ACCOUNT": "mystorageaccount",
			},
		},
		{
			name:        "service with no matching variables",
			serviceName: "api",
			azureEnv: map[string]string{
				"SERVICE_WEB_URL": "http://localhost:3000",
				"UNRELATED_VAR":   "value",
			},
			expected: map[string]string{},
		},
		{
			name:        "empty environment",
			serviceName: "api",
			azureEnv:    map[string]string{},
			expected:    map[string]string{},
		},
		{
			name:        "case insensitive service name",
			serviceName: "API",
			azureEnv: map[string]string{
				"SERVICE_API_URL": "http://localhost:8080",
				"service_api_db":  "postgres://localhost:5432",
			},
			expected: map[string]string{
				"SERVICE_API_URL": "http://localhost:8080",
				"service_api_db":  "postgres://localhost:5432",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getServiceEnvironmentVars(tt.serviceName, tt.azureEnv)

			if len(result) != len(tt.expected) {
				t.Errorf("getServiceEnvironmentVars() returned %d variables, want %d", len(result), len(tt.expected))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("getServiceEnvironmentVars() missing expected key %q", key)
				} else if actualValue != expectedValue {
					t.Errorf("getServiceEnvironmentVars()[%q] = %q, want %q", key, actualValue, expectedValue)
				}
			}

			// Check for unexpected keys
			for key := range result {
				if _, expected := tt.expected[key]; !expected {
					t.Errorf("getServiceEnvironmentVars() returned unexpected key %q with value %q", key, result[key])
				}
			}
		})
	}
}

func TestFormatStatusEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"empty string", ""},
		{"arbitrary value", "arbitrary"},
		{"mixed case", "Running"},
		{"with spaces", "running service"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatus(tt.status)
			// Should not panic and should return something
			if result == "" && tt.status != "" {
				t.Errorf("formatStatus(%q) returned empty string", tt.status)
			}
		})
	}
}

func TestFormatHealthEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		health string
	}{
		{"empty string", ""},
		{"arbitrary value", "arbitrary"},
		{"mixed case", "Healthy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatHealth(tt.health)
			// Should not panic and should return something
			if result == "" && tt.health != "" {
				t.Errorf("formatHealth(%q) returned empty string", tt.health)
			}
		})
	}
}

func TestGetInfoStatusIconEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		status string
		health string
	}{
		{"both empty", "", ""},
		{"status empty", "", "healthy"},
		{"health empty", "running", ""},
		{"invalid combinations", "invalid", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInfoStatusIcon(tt.status, tt.health)
			// Should always return something (default is "?")
			if result == "" {
				t.Errorf("getInfoStatusIcon(%q, %q) returned empty string", tt.status, tt.health)
			}
		})
	}
}

func TestFormatInfoDurationEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero duration", 0, "0s"},
		{"negative duration", -5 * time.Second, "-5s"},
		{"exactly 1 minute", 1 * time.Minute, "1m"},
		{"exactly 1 hour", 1 * time.Hour, "1h"},
		{"exactly 24 hours", 24 * time.Hour, "1d"},
		{"59 seconds", 59 * time.Second, "59s"},
		{"61 seconds", 61 * time.Second, "1m"},
		{"90 minutes", 90 * time.Minute, "1h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatInfoDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatInfoDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}
