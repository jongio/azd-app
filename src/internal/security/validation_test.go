package security

import (
	"strings"
	"testing"
)

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid path",
			path:    "/tmp/test",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "path with dots",
			path:    "../../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "current directory",
			path:    ".",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePackageManager(t *testing.T) {
	tests := []struct {
		name    string
		pm      string
		wantErr bool
	}{
		{
			name:    "valid npm",
			pm:      "npm",
			wantErr: false,
		},
		{
			name:    "valid pnpm",
			pm:      "pnpm",
			wantErr: false,
		},
		{
			name:    "valid yarn",
			pm:      "yarn",
			wantErr: false,
		},
		{
			name:    "valid pip",
			pm:      "pip",
			wantErr: false,
		},
		{
			name:    "valid poetry",
			pm:      "poetry",
			wantErr: false,
		},
		{
			name:    "valid uv",
			pm:      "uv",
			wantErr: false,
		},
		{
			name:    "invalid package manager",
			pm:      "malicious-pm",
			wantErr: true,
		},
		{
			name:    "shell command injection",
			pm:      "npm; rm -rf /",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePackageManager(tt.pm)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePackageManager() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeScriptName(t *testing.T) {
	tests := []struct {
		name       string
		scriptName string
		wantErr    bool
	}{
		{
			name:       "valid script name",
			scriptName: "dev",
			wantErr:    false,
		},
		{
			name:       "valid script with dash",
			scriptName: "build-prod",
			wantErr:    false,
		},
		{
			name:       "semicolon injection",
			scriptName: "dev; rm -rf /",
			wantErr:    true,
		},
		{
			name:       "pipe injection",
			scriptName: "dev | cat /etc/passwd",
			wantErr:    true,
		},
		{
			name:       "ampersand injection",
			scriptName: "dev & malicious",
			wantErr:    true,
		},
		{
			name:       "backtick injection",
			scriptName: "dev`whoami`",
			wantErr:    true,
		},
		{
			name:       "dollar sign injection",
			scriptName: "dev$(whoami)",
			wantErr:    true,
		},
		{
			name:       "redirect injection",
			scriptName: "dev > /tmp/pwned",
			wantErr:    true,
		},
		{
			name:       "newline injection",
			scriptName: "dev\nrm -rf /",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SanitizeScriptName(tt.scriptName)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizeScriptName() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "dangerous character") {
				t.Errorf("SanitizeScriptName() error message should mention dangerous character, got: %v", err)
			}
		})
	}
}
