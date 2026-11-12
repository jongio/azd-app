package dashboard

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestReadDashboardPortFromAzdConfig(t *testing.T) {
	// Create a temporary home directory for the test
	tmpHome := t.TempDir()
	azdDir := filepath.Join(tmpHome, ".azd")
	if err := os.MkdirAll(azdDir, 0o700); err != nil {
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
			if err := os.WriteFile(configPath, []byte(tt.config), 0o600); err != nil {
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

func TestNewClientForPortUsesProvidedSessionToken(t *testing.T) {
	client, err := newClientForPort(45678, "0123456789abcdef")
	if err != nil {
		t.Fatalf("newClientForPort() error = %v", err)
	}
	if client.baseURL != "http://127.0.0.1:45678" {
		t.Errorf("newClientForPort() baseURL = %q, want %q", client.baseURL, "http://127.0.0.1:45678")
	}
}

func TestNewClientForPortRejectsMissingSessionToken(t *testing.T) {
	client, err := newClientForPort(45678, "")
	if err == nil {
		t.Fatal("newClientForPort() error = nil, want missing-token error")
	}
	if client != nil {
		t.Fatalf("newClientForPort() client = %#v, want nil", client)
	}
}

func TestNewClientUsesCrossProcessPortAndTokenFiles(t *testing.T) {
	nonceDirBase = t.TempDir()
	projectDir := t.TempDir()
	t.Cleanup(func() {
		removePortFile(projectDir)
		removeTokenFile(projectDir)
		nonceDirBase = ""
	})

	writePortFile(projectDir, 45678)
	writeTokenFile(projectDir, "0123456789abcdef")

	client, err := NewClient(context.Background(), projectDir)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.baseURL != "http://127.0.0.1:45678" {
		t.Errorf("NewClient() baseURL = %q, want %q", client.baseURL, "http://127.0.0.1:45678")
	}
}

func TestProtoToServiceInfo_PreservesTerminalStatusAndExitCode(t *testing.T) {
	endTime := time.Date(2026, 11, 11, 22, 0, 0, 0, time.UTC)
	meta, err := structpb.NewStruct(map[string]any{
		"rawStatus":   "built",
		"exitCode":    float64(0),
		"endTime":     endTime.Format(time.RFC3339Nano),
		"serviceType": "process",
		"serviceMode": "build",
	})
	if err != nil {
		t.Fatalf("NewStruct error: %v", err)
	}

	got := protoToServiceInfo(&v1.ServiceInfo{
		Name:     "builder",
		Status:   v1.ServiceStatus_SERVICE_STATUS_UNSPECIFIED,
		Metadata: meta,
	})
	if got == nil || got.Local == nil {
		t.Fatal("expected LocalServiceInfo to be reconstructed from terminal metadata")
	}
	if got.Local.RawStatus != "built" {
		t.Fatalf("RawStatus = %q, want %q", got.Local.RawStatus, "built")
	}
	if got.Local.ExitCode == nil || *got.Local.ExitCode != 0 {
		t.Fatalf("ExitCode = %v, want 0", got.Local.ExitCode)
	}
	if got.Local.EndTime == nil || !got.Local.EndTime.Equal(endTime) {
		t.Fatalf("EndTime = %v, want %v", got.Local.EndTime, endTime)
	}
}
