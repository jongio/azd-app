package healthcheck

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetDefaultProfiles(t *testing.T) {
	profiles := getDefaultProfiles()

	if profiles == nil {
		t.Fatal("Expected non-nil profiles")
	}

	expectedProfiles := []string{"development", "production", "ci", "staging"}
	for _, name := range expectedProfiles {
		if _, exists := profiles.Profiles[name]; !exists {
			t.Errorf("Expected profile '%s' to exist in defaults", name)
		}
	}

	// Verify development profile defaults
	dev := profiles.Profiles["development"]
	if dev.Name != "development" {
		t.Errorf("Development profile name = %s, want development", dev.Name)
	}
	if dev.Interval != 5*time.Second {
		t.Errorf("Development interval = %v, want 5s", dev.Interval)
	}
	if dev.Verbose != true {
		t.Error("Development profile should be verbose")
	}
	if dev.CircuitBreaker != false {
		t.Error("Development profile should not have circuit breaker enabled")
	}

	// Verify production profile defaults
	prod := profiles.Profiles["production"]
	if prod.CircuitBreaker != true {
		t.Error("Production profile should have circuit breaker enabled")
	}
	if prod.Metrics != true {
		t.Error("Production profile should have metrics enabled")
	}
	if prod.RateLimit != 10 {
		t.Errorf("Production rate limit = %d, want 10", prod.RateLimit)
	}
}

func TestHealthProfiles_GetProfile(t *testing.T) {
	profiles := getDefaultProfiles()

	tests := []struct {
		name     string
		profile  string
		wantErr  bool
		wantName string
	}{
		{
			name:     "get development",
			profile:  "development",
			wantErr:  false,
			wantName: "development",
		},
		{
			name:     "get production",
			profile:  "production",
			wantErr:  false,
			wantName: "production",
		},
		{
			name:    "get non-existent",
			profile: "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, err := profiles.GetProfile(tt.profile)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && profile.Name != tt.wantName {
				t.Errorf("Profile name = %s, want %s", profile.Name, tt.wantName)
			}
		})
	}
}

func TestLoadHealthProfiles_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	profiles, err := LoadHealthProfiles(tmpDir)
	if err != nil {
		t.Fatalf("LoadHealthProfiles() error = %v, want nil (should return defaults)", err)
	}

	// Should return default profiles
	if profiles == nil {
		t.Fatal("Expected non-nil profiles")
	}

	if len(profiles.Profiles) == 0 {
		t.Error("Expected default profiles, got empty")
	}
}

func TestLoadHealthProfiles_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	azdDir := filepath.Join(tmpDir, ".azd")
	if err := os.MkdirAll(azdDir, 0755); err != nil {
		t.Fatalf("Failed to create .azd directory: %v", err)
	}

	profilePath := filepath.Join(azdDir, "health-profiles.yaml")
	content := `profiles:
  custom:
    name: custom
    interval: 2s
    timeout: 5s
    retries: 2
    circuitBreaker: true
    circuitBreakerFailures: 3
    circuitBreakerTimeout: 30s
    rateLimit: 5
    verbose: true
    logLevel: debug
    logFormat: json
    metrics: true
    metricsPort: 9091
    cacheTTL: 1s
`

	if err := os.WriteFile(profilePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test profile: %v", err)
	}

	profiles, err := LoadHealthProfiles(tmpDir)
	if err != nil {
		t.Fatalf("LoadHealthProfiles() error = %v", err)
	}

	// Should have custom profile
	custom, exists := profiles.Profiles["custom"]
	if !exists {
		t.Fatal("Expected custom profile to exist")
	}

	if custom.Name != "custom" {
		t.Errorf("Custom profile name = %s, want custom", custom.Name)
	}
	if custom.Interval != 2*time.Second {
		t.Errorf("Custom interval = %v, want 2s", custom.Interval)
	}

	// Should also have defaults merged in
	if _, exists := profiles.Profiles["development"]; !exists {
		t.Error("Expected development profile to be merged from defaults")
	}
}

func TestLoadHealthProfiles_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	azdDir := filepath.Join(tmpDir, ".azd")
	if err := os.MkdirAll(azdDir, 0755); err != nil {
		t.Fatalf("Failed to create .azd directory: %v", err)
	}

	profilePath := filepath.Join(azdDir, "health-profiles.yaml")
	invalidContent := `this is not valid yaml: [[[`

	if err := os.WriteFile(profilePath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write test profile: %v", err)
	}

	_, err := LoadHealthProfiles(tmpDir)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestSaveSampleProfiles(t *testing.T) {
	tmpDir := t.TempDir()

	err := SaveSampleProfiles(tmpDir)
	if err != nil {
		t.Fatalf("SaveSampleProfiles() error = %v", err)
	}

	// Verify file was created
	profilePath := filepath.Join(tmpDir, ".azd", "health-profiles.yaml")
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Error("Expected health-profiles.yaml to be created")
	}

	// Verify content is valid
	profiles, err := LoadHealthProfiles(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load saved profiles: %v", err)
	}

	if len(profiles.Profiles) == 0 {
		t.Error("Expected profiles in saved file")
	}
}

func TestSaveSampleProfiles_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file first
	err := SaveSampleProfiles(tmpDir)
	if err != nil {
		t.Fatalf("First SaveSampleProfiles() error = %v", err)
	}

	// Try to save again - should error
	err = SaveSampleProfiles(tmpDir)
	if err == nil {
		t.Error("Expected error when file already exists, got nil")
	}
}

func TestSaveSampleProfiles_DirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()

	// Remove .azd directory if it exists
	azdDir := filepath.Join(tmpDir, ".azd")
	os.RemoveAll(azdDir)

	err := SaveSampleProfiles(tmpDir)
	if err != nil {
		t.Fatalf("SaveSampleProfiles() error = %v", err)
	}

	// Verify .azd directory was created
	if _, err := os.Stat(azdDir); os.IsNotExist(err) {
		t.Error("Expected .azd directory to be created")
	}
}
