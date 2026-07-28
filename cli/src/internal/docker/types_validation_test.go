package docker

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateNetworkName_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty is valid", "", false},
		{"simple name", "my-network", false},
		{"alphanumeric start", "a123", false},
		{"dots and underscores", "my.net_work", false},
		{"max length 128", strings.Repeat("a", 128), false},
		{"too long 129", strings.Repeat("a", 129), true},
		{"starts with dash", "-bad", true},
		{"starts with dot", ".bad", true},
		{"contains space", "bad name", true},
		{"contains slash", "bad/name", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNetworkName(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateVolumeSpec_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{"valid named volume", "pgdata:/var/lib/data", false, ""},
		{"valid bind mount", "/host/path:/container/path", false, ""},
		{"empty string", "", true, "empty"},
		{"whitespace only", "   ", true, "empty"},
		{"contains null byte", "bad\x00path:/app", true, "control"},
		{"contains newline", "bad\npath:/app", true, "control"},
		{"contains carriage return", "bad\rpath:/app", true, "control"},
		{"max length 4096", strings.Repeat("a", 4096), false, ""},
		{"too long 4097", strings.Repeat("a", 4097), true, "too long"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVolumeSpec(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateImageName_SingleChar(t *testing.T) {
	// Regression: single-character image names must be accepted.
	require.NoError(t, ValidateImageName("a"))
	require.NoError(t, ValidateImageName("x"))
	require.NoError(t, ValidateImageName("a:latest"))
}

func TestContainerConfigValidate_MissingImage(t *testing.T) {
	cfg := ContainerConfig{Name: "valid-name"}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "image")
}

func TestContainerConfigValidate_InvalidPort(t *testing.T) {
	cfg := ContainerConfig{
		Name:  "valid-name",
		Image: "nginx",
		Ports: []PortMapping{{HostPort: -1, ContainerPort: 80}},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port")
}

func TestContainerConfigValidate_InvalidVolumeControlChars(t *testing.T) {
	cfg := ContainerConfig{
		Name:    "valid-name",
		Image:   "nginx",
		Volumes: []string{"bad\x00vol:/data"},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "volume")
}
