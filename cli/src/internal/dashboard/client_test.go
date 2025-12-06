package dashboard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
)

func TestReadDashboardPortFromAzdConfig(t *testing.T) {
	// Create a temporary home directory for the test
	tmpHome := t.TempDir()
	azdDir := filepath.Join(tmpHome, ".azd")
	if err := os.MkdirAll(azdDir, 0700); err != nil {
		t.Fatalf("failed to create .azd directory: %v", err)
	}

	// Create a project dir to get its hash
	projectDir := filepath.Join(tmpHome, "myproject")
	projectHash := azdconfig.ProjectHash(projectDir)

	tests := []struct {
		name      string
		config    string
		wantPort  int
		wantError bool
	}{
		{
			name:      "valid config with dashboard port",
			config:    `{"app":{"projects":{"` + projectHash + `":{"dashboardPort":12345}}}}`,
			wantPort:  12345,
			wantError: false,
		},
		{
			name:      "empty config",
			config:    `{}`,
			wantPort:  0,
			wantError: false,
		},
		{
			name:      "config without app section",
			config:    `{"other":"value"}`,
			wantPort:  0,
			wantError: false,
		},
		{
			name:      "config without projects section",
			config:    `{"app":{}}`,
			wantPort:  0,
			wantError: false,
		},
		{
			name:      "config with different project hash",
			config:    `{"app":{"projects":{"differenthash":{"dashboardPort":99999}}}}`,
			wantPort:  0,
			wantError: false,
		},
		{
			name:      "invalid json",
			config:    `invalid`,
			wantPort:  0,
			wantError: true,
		},
		{
			name:      "project without dashboardPort",
			config:    `{"app":{"projects":{"` + projectHash + `":{"ports":{"api":3000}}}}}`,
			wantPort:  0,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(azdDir, "config.json")
			if err := os.WriteFile(configPath, []byte(tt.config), 0600); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			// Override the azdConfigPath function for this test
			originalFunc := azdConfigPath
			azdConfigPath = func() (string, error) {
				return configPath, nil
			}
			defer func() { azdConfigPath = originalFunc }()

			port, err := readDashboardPortFromAzdConfig(projectHash)
			if (err != nil) != tt.wantError {
				t.Errorf("readDashboardPortFromAzdConfig() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if port != tt.wantPort {
				t.Errorf("readDashboardPortFromAzdConfig() port = %v, want %v", port, tt.wantPort)
			}
		})
	}
}

func TestReadDashboardPortFromAzdConfig_NoFile(t *testing.T) {
	tmpHome := t.TempDir()
	configPath := filepath.Join(tmpHome, ".azd", "config.json")

	// Override the azdConfigPath function for this test
	originalFunc := azdConfigPath
	azdConfigPath = func() (string, error) {
		return configPath, nil
	}
	defer func() { azdConfigPath = originalFunc }()

	// File doesn't exist - should return 0 without error
	port, err := readDashboardPortFromAzdConfig("somehash")
	if err != nil {
		t.Errorf("readDashboardPortFromAzdConfig() error = %v, want nil", err)
	}
	if port != 0 {
		t.Errorf("readDashboardPortFromAzdConfig() port = %v, want 0", port)
	}
}
