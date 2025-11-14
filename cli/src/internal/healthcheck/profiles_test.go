package healthcheck

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadHealthProfiles(t *testing.T) {
	tempDir := t.TempDir()

	// Test loading when no profile file exists (should return defaults)
	profiles, err := LoadHealthProfiles(tempDir)
	if err != nil {
		t.Fatalf("Failed to load default profiles: %v", err)
	}

	if profiles == nil {
		t.Fatal("Expected profiles, got nil")
	}

	// Verify default profiles exist
	expectedProfiles := []string{"development", "production", "ci", "staging"}
	for _, name := range expectedProfiles {
		if _, exists := profiles.Profiles[name]; !exists {
			t.Errorf("Expected default profile '%s' to exist", name)
		}
	}
}

func TestLoadHealthProfilesFromFile(t *testing.T) {
	tempDir := t.TempDir()
	azdDir := filepath.Join(tempDir, ".azd")
	if err := os.MkdirAll(azdDir, 0755); err != nil {
		t.Fatalf("Failed to create .azd directory: %v", err)
	}

	// Create custom profile file
	profileContent := `profiles:
  custom:
    name: custom
    interval: 10s
    timeout: 15s
    retries: 5
    circuitBreaker: true
    circuitBreakerFailures: 10
    circuitBreakerTimeout: 30s
    rateLimit: 5
    verbose: true
    logLevel: debug
    logFormat: json
    metrics: true
    metricsPort: 8080
    cacheTTL: 10s
`

	profilePath := filepath.Join(azdDir, "health-profiles.yaml")
	if err := os.WriteFile(profilePath, []byte(profileContent), 0644); err != nil {
		t.Fatalf("Failed to write profile file: %v", err)
	}

	// Load profiles
	profiles, err := LoadHealthProfiles(tempDir)
	if err != nil {
		t.Fatalf("Failed to load profiles: %v", err)
	}

	// Verify custom profile
	custom, exists := profiles.Profiles["custom"]
	if !exists {
		t.Fatal("Expected custom profile to exist")
	}

	if custom.Name != "custom" {
		t.Errorf("Expected name 'custom', got '%s'", custom.Name)
	}

	if custom.Interval != 10*time.Second {
		t.Errorf("Expected interval 10s, got %v", custom.Interval)
	}

	if custom.Timeout != 15*time.Second {
		t.Errorf("Expected timeout 15s, got %v", custom.Timeout)
	}

	if custom.Retries != 5 {
		t.Errorf("Expected retries 5, got %d", custom.Retries)
	}

	// Verify default profiles are still present
	if _, exists := profiles.Profiles["development"]; !exists {
		t.Error("Expected development profile to exist (merged with defaults)")
	}
}

func TestGetDefaultProfiles(t *testing.T) {
	profiles := getDefaultProfiles()

	if profiles == nil {
		t.Fatal("Expected profiles, got nil")
	}

	// Test development profile
	dev, exists := profiles.Profiles["development"]
	if !exists {
		t.Fatal("Expected development profile")
	}

	if dev.Interval != 5*time.Second {
		t.Errorf("Development: expected interval 5s, got %v", dev.Interval)
	}

	if dev.CircuitBreaker {
		t.Error("Development: expected circuit breaker to be disabled")
	}

	if dev.LogLevel != "debug" {
		t.Errorf("Development: expected log level 'debug', got '%s'", dev.LogLevel)
	}

	// Test production profile
	prod, exists := profiles.Profiles["production"]
	if !exists {
		t.Fatal("Expected production profile")
	}

	if prod.Interval != 30*time.Second {
		t.Errorf("Production: expected interval 30s, got %v", prod.Interval)
	}

	if !prod.CircuitBreaker {
		t.Error("Production: expected circuit breaker to be enabled")
	}

	if prod.LogLevel != "info" {
		t.Errorf("Production: expected log level 'info', got '%s'", prod.LogLevel)
	}

	if !prod.Metrics {
		t.Error("Production: expected metrics to be enabled")
	}

	// Test CI profile
	ci, exists := profiles.Profiles["ci"]
	if !exists {
		t.Fatal("Expected ci profile")
	}

	if ci.Timeout != 30*time.Second {
		t.Errorf("CI: expected timeout 30s, got %v", ci.Timeout)
	}

	if ci.Retries != 5 {
		t.Errorf("CI: expected retries 5, got %d", ci.Retries)
	}

	// Test staging profile
	staging, exists := profiles.Profiles["staging"]
	if !exists {
		t.Fatal("Expected staging profile")
	}

	if staging.RateLimit != 20 {
		t.Errorf("Staging: expected rate limit 20, got %d", staging.RateLimit)
	}
}

func TestGetProfile(t *testing.T) {
	profiles := getDefaultProfiles()

	// Test getting existing profile
	dev, err := profiles.GetProfile("development")
	if err != nil {
		t.Fatalf("Failed to get development profile: %v", err)
	}

	if dev.Name != "development" {
		t.Errorf("Expected name 'development', got '%s'", dev.Name)
	}

	// Test getting non-existent profile
	_, err = profiles.GetProfile("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent profile")
	}

	if err != nil && err.Error() == "" {
		t.Error("Expected error message for non-existent profile")
	}
}

func TestSaveSampleProfiles(t *testing.T) {
	tempDir := t.TempDir()

	// Save sample profiles
	err := SaveSampleProfiles(tempDir)
	if err != nil {
		t.Fatalf("Failed to save sample profiles: %v", err)
	}

	// Verify file was created
	profilePath := filepath.Join(tempDir, ".azd", "health-profiles.yaml")
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Error("Expected profile file to be created")
	}

	// Verify file content
	content, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatalf("Failed to read profile file: %v", err)
	}

	contentStr := string(content)

	// Should contain header comments
	if !contains(contentStr, "Health Check Profiles") {
		t.Error("Expected header comment in profile file")
	}

	// Should contain all default profiles
	for _, profile := range []string{"development", "production", "ci", "staging"} {
		if !contains(contentStr, profile) {
			t.Errorf("Expected profile '%s' in saved file", profile)
		}
	}

	// Test saving when file already exists
	err = SaveSampleProfiles(tempDir)
	if err == nil {
		t.Error("Expected error when saving to existing file")
	}
}

func TestLoadHealthProfilesInvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	azdDir := filepath.Join(tempDir, ".azd")
	if err := os.MkdirAll(azdDir, 0755); err != nil {
		t.Fatalf("Failed to create .azd directory: %v", err)
	}

	// Write invalid YAML
	invalidYAML := `profiles:
  invalid:
    - this is not valid
    - yaml structure
`

	profilePath := filepath.Join(azdDir, "health-profiles.yaml")
	if err := os.WriteFile(profilePath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write invalid profile: %v", err)
	}

	// Loading should fail
	_, err := LoadHealthProfiles(tempDir)
	if err == nil {
		t.Error("Expected error when loading invalid YAML")
	}
}

func TestProfileSettings(t *testing.T) {
	tests := []struct {
		name     string
		profile  string
		validate func(*testing.T, HealthProfile)
	}{
		{
			name:    "development has no cache",
			profile: "development",
			validate: func(t *testing.T, p HealthProfile) {
				if p.CacheTTL != 0 {
					t.Error("Development should have no caching")
				}
			},
		},
		{
			name:    "production has caching enabled",
			profile: "production",
			validate: func(t *testing.T, p HealthProfile) {
				if p.CacheTTL == 0 {
					t.Error("Production should have caching enabled")
				}
			},
		},
		{
			name:    "ci has longer timeout",
			profile: "ci",
			validate: func(t *testing.T, p HealthProfile) {
				if p.Timeout != 30*time.Second {
					t.Errorf("CI should have 30s timeout, got %v", p.Timeout)
				}
			},
		},
		{
			name:    "staging has higher rate limit",
			profile: "staging",
			validate: func(t *testing.T, p HealthProfile) {
				prod, _ := getDefaultProfiles().GetProfile("production")
				if p.RateLimit <= prod.RateLimit {
					t.Error("Staging should have higher rate limit than production")
				}
			},
		},
	}

	profiles := getDefaultProfiles()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, err := profiles.GetProfile(tt.profile)
			if err != nil {
				t.Fatalf("Failed to get %s profile: %v", tt.profile, err)
			}
			tt.validate(t, profile)
		})
	}
}

func TestProfileMerging(t *testing.T) {
	tempDir := t.TempDir()
	azdDir := filepath.Join(tempDir, ".azd")
	if err := os.MkdirAll(azdDir, 0755); err != nil {
		t.Fatalf("Failed to create .azd directory: %v", err)
	}

	// Create file with only one custom profile
	profileContent := `profiles:
  myprofile:
    name: myprofile
    interval: 20s
    timeout: 25s
    retries: 3
    circuitBreaker: false
    circuitBreakerFailures: 3
    circuitBreakerTimeout: 60s
    rateLimit: 0
    verbose: false
    logLevel: info
    logFormat: text
    metrics: false
    metricsPort: 9091
    cacheTTL: 0s
`

	profilePath := filepath.Join(azdDir, "health-profiles.yaml")
	if err := os.WriteFile(profilePath, []byte(profileContent), 0644); err != nil {
		t.Fatalf("Failed to write profile: %v", err)
	}

	// Load profiles
	profiles, err := LoadHealthProfiles(tempDir)
	if err != nil {
		t.Fatalf("Failed to load profiles: %v", err)
	}

	// Custom profile should exist
	custom, exists := profiles.Profiles["myprofile"]
	if !exists {
		t.Error("Expected custom profile 'myprofile'")
	}

	if custom.Interval != 20*time.Second {
		t.Errorf("Expected custom interval 20s, got %v", custom.Interval)
	}

	// Default profiles should also exist (merged)
	for _, name := range []string{"development", "production", "ci", "staging"} {
		if _, exists := profiles.Profiles[name]; !exists {
			t.Errorf("Expected default profile '%s' to be merged", name)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
