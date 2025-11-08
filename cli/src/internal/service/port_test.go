package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPort_ExplicitFromAzureYaml(t *testing.T) {
	service := Service{
		Ports: []string{"3000"},
	}

	port, isExplicit, err := DetectPort("test-service", service, ".", "Next.js", nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port != 3000 {
		t.Errorf("Expected port 3000, got %d", port)
	}

	if !isExplicit {
		t.Error("Expected isExplicit to be true for azure.yaml ports field")
	}
}

func TestDetectPort_ExplicitFromAzureYaml_HostContainer(t *testing.T) {
	// Test host:container port mapping
	service := Service{
		Ports: []string{"3000:8080"},
	}

	port, isExplicit, err := DetectPort("test-service", service, ".", "Next.js", nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port != 3000 {
		t.Errorf("Expected port 3000, got %d", port)
	}

	if !isExplicit {
		t.Error("Expected isExplicit to be true for azure.yaml ports field")
	}
}

func TestDetectPort_NoExplicitConfig(t *testing.T) {
	service := Service{}

	usedPorts := make(map[int]bool)
	port, isExplicit, err := DetectPort("test-service", service, ".", "Next.js", usedPorts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port == 0 {
		t.Error("Expected non-zero port")
	}

	if isExplicit {
		t.Error("Expected isExplicit to be false when no explicit config")
	}
}

func TestDetectPort_EmptyConfig(t *testing.T) {
	service := Service{}

	usedPorts := make(map[int]bool)
	port, isExplicit, err := DetectPort("test-service", service, ".", "React", usedPorts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port == 0 {
		t.Error("Expected non-zero port")
	}

	if isExplicit {
		t.Error("Expected isExplicit to be false for empty config")
	}
}

func TestDetectPort_FrameworkDefault_NotExplicit(t *testing.T) {
	service := Service{
		Language: "TypeScript",
	}

	usedPorts := make(map[int]bool)
	port, isExplicit, err := DetectPort("test-service", service, ".", "Next.js", usedPorts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Next.js default is 3000
	if port != 3000 {
		t.Errorf("Expected default port 3000 for Next.js, got %d", port)
	}

	if isExplicit {
		t.Error("Expected isExplicit to be false for framework default")
	}
}

func TestDetectPort_DynamicAssignment_NotExplicit(t *testing.T) {
	service := Service{}

	usedPorts := map[int]bool{
		3000: true, // Framework default in use
		3001: true,
		3002: true,
	}

	port, isExplicit, err := DetectPort("test-service", service, ".", "Unknown", usedPorts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port < 3000 || port > 9999 {
		t.Errorf("Expected port in range 3000-9999, got %d", port)
	}

	if isExplicit {
		t.Error("Expected isExplicit to be false for dynamic assignment")
	}
}

func TestDetectPort_PackageJSON_NotExplicit(t *testing.T) {
	tempDir := t.TempDir()

	// Create package.json with port in script
	packageJSON := `{
		"scripts": {
			"dev": "next dev -p 3333"
		}
	}`

	packagePath := filepath.Join(tempDir, "package.json")
	if err := os.WriteFile(packagePath, []byte(packageJSON), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	service := Service{}
	port, isExplicit, err := DetectPort("test-service", service, tempDir, "Next.js", nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// -p flag is not extracted by regex (only --port is supported)
	// So it falls back to framework default for Next.js which is 3000
	if port != 3000 {
		t.Errorf("Expected port 3000 (Next.js default), got %d", port)
	}

	if isExplicit {
		t.Error("Expected isExplicit to be false for package.json config")
	}
}

func TestDetectPort_ExplicitOverridesFrameworkConfig(t *testing.T) {
	tempDir := t.TempDir()

	// Create package.json with port 3333
	packageJSON := `{
		"scripts": {
			"dev": "next dev -p 3333"
		}
	}`

	packagePath := filepath.Join(tempDir, "package.json")
	if err := os.WriteFile(packagePath, []byte(packageJSON), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	// But azure.yaml has explicit port 4000
	service := Service{
		Ports: []string{"4000"},
	}

	port, isExplicit, err := DetectPort("test-service", service, tempDir, "Next.js", nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port != 4000 {
		t.Errorf("Expected explicit port 4000 to override framework config, got %d", port)
	}

	if !isExplicit {
		t.Error("Expected isExplicit to be true")
	}
}

func TestDetectPort_UsedPortsRespected(t *testing.T) {
	service := Service{}

	usedPorts := map[int]bool{
		3000: true, // Default port is in use
	}

	port, isExplicit, err := DetectPort("test-service", service, ".", "Next.js", usedPorts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port == 3000 {
		t.Error("Expected to avoid used port 3000")
	}

	if isExplicit {
		t.Error("Expected isExplicit to be false")
	}
}

func TestGetFrameworkDefaultPort(t *testing.T) {
	tests := []struct {
		framework    string
		language     string
		expectedPort int
	}{
		{"Next.js", "TypeScript", 3000},
		{"React", "JavaScript", 5173},
		{"Vue", "TypeScript", 5173},
		{"Angular", "TypeScript", 4200},
		{"Django", "Python", 8000},
		{"Flask", "Python", 5000},
		{"Spring Boot", "Java", 8080},
		{"Unknown", "Go", 8080}, // Falls back to DefaultPorts["go"] = 8080
	}

	for _, tt := range tests {
		port := getFrameworkDefaultPort(tt.framework, tt.language)
		if port != tt.expectedPort {
			t.Errorf("Framework %s: expected port %d, got %d", tt.framework, tt.expectedPort, port)
		}
	}
}

func TestFindAvailablePort_ServicePort(t *testing.T) {
	usedPorts := map[int]bool{
		3000: true,
		3001: true,
		3002: true,
	}

	port, err := findAvailablePort(3000, usedPorts)
	if err != nil {
		t.Fatalf("Expected to find available port, got error: %v", err)
	}

	if port < 3000 || port > 9999 {
		t.Errorf("Expected port in range 3000-9999, got %d", port)
	}

	if usedPorts[port] {
		t.Errorf("Port %d should not be in usedPorts map", port)
	}
}

func TestDetectPortFromEnv(t *testing.T) {
	// Set environment variable
	os.Setenv("TEST_SERVICE_PORT", "5555")
	defer os.Unsetenv("TEST_SERVICE_PORT")

	port := detectPortFromEnv("TEST_SERVICE")
	if port != 5555 {
		t.Errorf("Expected port 5555 from env var, got %d", port)
	}
}

func TestDetectPortFromEnv_GenericPORT(t *testing.T) {
	// Set generic PORT variable
	os.Setenv("PORT", "6666")
	defer os.Unsetenv("PORT")

	port := detectPortFromEnv("any-service")
	if port != 6666 {
		t.Errorf("Expected port 6666 from PORT env var, got %d", port)
	}
}

func TestDetectPortFromEnv_NotSet(t *testing.T) {
	port := detectPortFromEnv("NONEXISTENT_SERVICE")
	if port != 0 {
		t.Errorf("Expected 0 for nonexistent env var, got %d", port)
	}
}

func TestDetectPortFromEnv_Invalid(t *testing.T) {
	os.Setenv("TEST_SERVICE_PORT", "invalid")
	defer os.Unsetenv("TEST_SERVICE_PORT")

	port := detectPortFromEnv("TEST_SERVICE")
	if port != 0 {
		t.Errorf("Expected 0 for invalid port value, got %d", port)
	}
}

// TestExtractPortFromURL tests the extractPortFromURL function
func TestExtractPortFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected int
	}{
		{"http with port", "http://localhost:3000", 3000},
		{"https with port", "https://example.com:8443", 8443},
		{"no port", "http://localhost", 0},
		{"port only", ":9000", 9000},
		{"invalid port", ":invalid", 0},
		{"empty string", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPortFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("extractPortFromURL(%q) = %d, want %d", tt.url, result, tt.expected)
			}
		})
	}
}

// TestIsPortAvailable tests the IsPortAvailable function
func TestIsPortAvailable(t *testing.T) {
	// Test with a port that should be available
	// Use a high port number to avoid conflicts
	testPort := 59999

	if !IsPortAvailable(testPort) {
		t.Logf("Port %d is in use (this may vary by system)", testPort)
	}

	// Note: Port 0 is a special case - it asks the OS to assign a free port
	// The OS will bind it successfully, so it appears "available"
	// This is expected behavior, not a bug
}

// TestDetectPortFromSpringConfig tests Spring Boot configuration detection
func TestDetectPortFromSpringConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure for Spring Boot project
	resourcesDir := filepath.Join(tmpDir, "src", "main", "resources")
	if err := os.MkdirAll(resourcesDir, 0750); err != nil {
		t.Fatalf("failed to create resources dir: %v", err)
	}

	t.Run("application.properties", func(t *testing.T) {
		propsPath := filepath.Join(resourcesDir, "application.properties")
		content := "server.port=8090\nserver.name=myapp\n"
		if err := os.WriteFile(propsPath, []byte(content), 0600); err != nil {
			t.Fatalf("failed to create application.properties: %v", err)
		}

		port, err := detectPortFromSpringConfig(tmpDir)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if port != 8090 {
			t.Errorf("Expected port 8090, got %d", port)
		}

		// Clean up
		os.Remove(propsPath)
	})

	t.Run("application.yml", func(t *testing.T) {
		ymlPath := filepath.Join(resourcesDir, "application.yml")
		content := "server:\n  port: 8091\n"
		if err := os.WriteFile(ymlPath, []byte(content), 0600); err != nil {
			t.Fatalf("failed to create application.yml: %v", err)
		}

		port, err := detectPortFromSpringConfig(tmpDir)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if port != 8091 {
			t.Errorf("Expected port 8091, got %d", port)
		}

		// Clean up
		os.Remove(ymlPath)
	})

	t.Run("no config", func(t *testing.T) {
		_, err := detectPortFromSpringConfig(tmpDir)
		if err == nil {
			t.Error("Expected error when no config files exist")
		}
	})
}
