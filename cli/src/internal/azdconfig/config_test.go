package azdconfig

import (
	"testing"
)

// TestProjectHash tests the project hash generation
func TestProjectHash(t *testing.T) {
	tests := []struct {
		name        string
		path1       string
		path2       string
		shouldMatch bool
	}{
		{
			name:        "identical paths should produce same hash",
			path1:       "/home/user/project",
			path2:       "/home/user/project",
			shouldMatch: true,
		},
		{
			name:        "different paths should produce different hashes",
			path1:       "/home/user/project1",
			path2:       "/home/user/project2",
			shouldMatch: false,
		},
		{
			name:        "relative and absolute paths should match after normalization",
			path1:       ".",
			path2:       ".",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := ProjectHash(tt.path1)
			hash2 := ProjectHash(tt.path2)

			// Hash should be 16 hex chars (8 bytes * 2)
			if len(hash1) != 16 {
				t.Errorf("ProjectHash() returned hash of length %d, expected 16", len(hash1))
			}

			if tt.shouldMatch {
				if hash1 != hash2 {
					t.Errorf("ProjectHash() = %v and %v, expected to match", hash1, hash2)
				}
			} else {
				if hash1 == hash2 {
					t.Errorf("ProjectHash() = %v and %v, expected to differ", hash1, hash2)
				}
			}
		})
	}
}

// TestInMemoryClient_DashboardPort tests dashboard port operations
func TestInMemoryClient_DashboardPort(t *testing.T) {
	client := NewInMemoryClient()
	defer client.Close()

	projectHash := "test123"

	// Test getting non-existent port returns 0
	port, err := client.GetDashboardPort(projectHash)
	if err != nil {
		t.Fatalf("GetDashboardPort() error = %v", err)
	}
	if port != 0 {
		t.Errorf("GetDashboardPort() = %v, want 0 for non-existent port", port)
	}

	// Test setting port
	err = client.SetDashboardPort(projectHash, 8080)
	if err != nil {
		t.Fatalf("SetDashboardPort() error = %v", err)
	}

	// Test getting the set port
	port, err = client.GetDashboardPort(projectHash)
	if err != nil {
		t.Fatalf("GetDashboardPort() error = %v", err)
	}
	if port != 8080 {
		t.Errorf("GetDashboardPort() = %v, want 8080", port)
	}

	// Test updating port
	err = client.SetDashboardPort(projectHash, 9090)
	if err != nil {
		t.Fatalf("SetDashboardPort() error = %v", err)
	}

	port, err = client.GetDashboardPort(projectHash)
	if err != nil {
		t.Fatalf("GetDashboardPort() error = %v", err)
	}
	if port != 9090 {
		t.Errorf("GetDashboardPort() = %v, want 9090", port)
	}

	// Test clearing port
	err = client.ClearDashboardPort(projectHash)
	if err != nil {
		t.Fatalf("ClearDashboardPort() error = %v", err)
	}

	port, err = client.GetDashboardPort(projectHash)
	if err != nil {
		t.Fatalf("GetDashboardPort() error = %v", err)
	}
	if port != 0 {
		t.Errorf("GetDashboardPort() = %v, want 0 after clear", port)
	}
}

// TestInMemoryClient_ServicePort tests service port operations
func TestInMemoryClient_ServicePort(t *testing.T) {
	client := NewInMemoryClient()
	defer client.Close()

	projectHash := "test123"
	serviceName := "api"

	// Test getting non-existent port returns 0
	port, err := client.GetServicePort(projectHash, serviceName)
	if err != nil {
		t.Fatalf("GetServicePort() error = %v", err)
	}
	if port != 0 {
		t.Errorf("GetServicePort() = %v, want 0 for non-existent port", port)
	}

	// Test setting port
	err = client.SetServicePort(projectHash, serviceName, 3000)
	if err != nil {
		t.Fatalf("SetServicePort() error = %v", err)
	}

	// Test getting the set port
	port, err = client.GetServicePort(projectHash, serviceName)
	if err != nil {
		t.Fatalf("GetServicePort() error = %v", err)
	}
	if port != 3000 {
		t.Errorf("GetServicePort() = %v, want 3000", port)
	}

	// Test clearing port
	err = client.ClearServicePort(projectHash, serviceName)
	if err != nil {
		t.Fatalf("ClearServicePort() error = %v", err)
	}

	port, err = client.GetServicePort(projectHash, serviceName)
	if err != nil {
		t.Fatalf("GetServicePort() error = %v", err)
	}
	if port != 0 {
		t.Errorf("GetServicePort() = %v, want 0 after clear", port)
	}
}

// TestInMemoryClient_GetAllServicePorts tests getting all service ports
func TestInMemoryClient_GetAllServicePorts(t *testing.T) {
	client := NewInMemoryClient()
	defer client.Close()

	projectHash := "test123"

	// Test getting empty ports
	ports, err := client.GetAllServicePorts(projectHash)
	if err != nil {
		t.Fatalf("GetAllServicePorts() error = %v", err)
	}
	if len(ports) != 0 {
		t.Errorf("GetAllServicePorts() = %v, want empty map", ports)
	}

	// Set multiple service ports
	services := map[string]int{
		"api":      3000,
		"web":      8080,
		"database": 5432,
	}

	for name, port := range services {
		err = client.SetServicePort(projectHash, name, port)
		if err != nil {
			t.Fatalf("SetServicePort(%s, %d) error = %v", name, port, err)
		}
	}

	// Get all ports
	ports, err = client.GetAllServicePorts(projectHash)
	if err != nil {
		t.Fatalf("GetAllServicePorts() error = %v", err)
	}

	if len(ports) != len(services) {
		t.Errorf("GetAllServicePorts() returned %d ports, want %d", len(ports), len(services))
	}

	for name, expectedPort := range services {
		if port, ok := ports[name]; !ok {
			t.Errorf("GetAllServicePorts() missing service %s", name)
		} else if port != expectedPort {
			t.Errorf("GetAllServicePorts()[%s] = %v, want %v", name, port, expectedPort)
		}
	}

	// Test isolation between projects
	otherProjectHash := "other456"
	otherPorts, err := client.GetAllServicePorts(otherProjectHash)
	if err != nil {
		t.Fatalf("GetAllServicePorts() error = %v", err)
	}
	if len(otherPorts) != 0 {
		t.Errorf("GetAllServicePorts(other project) = %v, want empty map", otherPorts)
	}
}

// TestInMemoryClient_Preferences tests user preference operations
func TestInMemoryClient_Preferences(t *testing.T) {
	client := NewInMemoryClient()
	defer client.Close()

	key := "theme"

	// Test getting non-existent preference returns empty string
	value, err := client.GetPreference(key)
	if err != nil {
		t.Fatalf("GetPreference() error = %v", err)
	}
	if value != "" {
		t.Errorf("GetPreference() = %v, want empty string for non-existent key", value)
	}

	// Test setting preference
	err = client.SetPreference(key, "dark")
	if err != nil {
		t.Fatalf("SetPreference() error = %v", err)
	}

	// Test getting the set preference
	value, err = client.GetPreference(key)
	if err != nil {
		t.Fatalf("GetPreference() error = %v", err)
	}
	if value != "dark" {
		t.Errorf("GetPreference() = %v, want 'dark'", value)
	}

	// Test clearing preference
	err = client.ClearPreference(key)
	if err != nil {
		t.Fatalf("ClearPreference() error = %v", err)
	}

	value, err = client.GetPreference(key)
	if err != nil {
		t.Fatalf("GetPreference() error = %v", err)
	}
	if value != "" {
		t.Errorf("GetPreference() = %v, want empty string after clear", value)
	}
}

// TestInMemoryClient_PreferenceSection tests preference section operations
func TestInMemoryClient_PreferenceSection(t *testing.T) {
	client := NewInMemoryClient()
	defer client.Close()

	key := "notifications"

	// Test getting non-existent section returns nil
	data, err := client.GetPreferenceSection(key)
	if err != nil {
		t.Fatalf("GetPreferenceSection() error = %v", err)
	}
	if data != nil {
		t.Errorf("GetPreferenceSection() = %v, want nil for non-existent section", data)
	}

	// Test setting section
	jsonData := []byte(`{"enabled":true,"level":"info"}`)
	err = client.SetPreferenceSection(key, jsonData)
	if err != nil {
		t.Fatalf("SetPreferenceSection() error = %v", err)
	}

	// Test getting the set section
	data, err = client.GetPreferenceSection(key)
	if err != nil {
		t.Fatalf("GetPreferenceSection() error = %v", err)
	}
	if string(data) != string(jsonData) {
		t.Errorf("GetPreferenceSection() = %v, want %v", string(data), string(jsonData))
	}

	// Test clearing section
	err = client.ClearPreference(key)
	if err != nil {
		t.Fatalf("ClearPreference() error = %v", err)
	}

	data, err = client.GetPreferenceSection(key)
	if err != nil {
		t.Fatalf("GetPreferenceSection() error = %v", err)
	}
	if data != nil {
		t.Errorf("GetPreferenceSection() = %v, want nil after clear", data)
	}
}

// TestInMemoryClient_ConcurrentAccess tests thread safety
func TestInMemoryClient_ConcurrentAccess(t *testing.T) {
	client := NewInMemoryClient()
	defer client.Close()

	projectHash := "test123"
	const goroutines = 10

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			serviceName := "service" + string(rune('0'+id))
			err := client.SetServicePort(projectHash, serviceName, 3000+id)
			if err != nil {
				t.Errorf("SetServicePort() error = %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all writes to complete
	for i := 0; i < goroutines; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			serviceName := "service" + string(rune('0'+id))
			port, err := client.GetServicePort(projectHash, serviceName)
			if err != nil {
				t.Errorf("GetServicePort() error = %v", err)
			}
			expectedPort := 3000 + id
			if port != expectedPort {
				t.Errorf("GetServicePort() = %v, want %v", port, expectedPort)
			}
			done <- true
		}(i)
	}

	// Wait for all reads to complete
	for i := 0; i < goroutines; i++ {
		<-done
	}
}

// TestPathFunctions tests path building functions
func TestPathFunctions(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() string
		expected string
	}{
		{
			name: "project config path",
			fn: func() string {
				return projectConfigPath("abc123", "dashboardPort")
			},
			expected: "app.projects.abc123.dashboardPort",
		},
		{
			name: "project config path with nested key",
			fn: func() string {
				return projectConfigPath("abc123", "ports.api")
			},
			expected: "app.projects.abc123.ports.api",
		},
		{
			name: "preference path",
			fn: func() string {
				return preferencePath("theme")
			},
			expected: "app.preferences.theme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn()
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}
