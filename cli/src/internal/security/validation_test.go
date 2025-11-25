package security

import (
	"os"
	"runtime"
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

func TestValidateFilePermissions(t *testing.T) {
	tmpFile := t.TempDir() + "/test.txt"

	// Create a test file
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test with secure permissions (0644)
	err := ValidateFilePermissions(tmpFile)
	if err != nil {
		t.Errorf("ValidateFilePermissions() with 0644 should pass, got error: %v", err)
	}

	// Skip world-writable test on Windows (uses ACLs)
	if runtime.GOOS != "windows" {
		// Make file world-writable
		if err := os.Chmod(tmpFile, 0666); err != nil {
			t.Fatalf("Failed to chmod file: %v", err)
		}

		// Test with insecure permissions (0666)
		err = ValidateFilePermissions(tmpFile)
		if err == nil {
			t.Error("ValidateFilePermissions() with 0666 should fail on Unix")
		}
	}

	// Test with non-existent file (only fails on Unix, Windows returns nil)
	if runtime.GOOS != "windows" {
		err = ValidateFilePermissions("/nonexistent/file")
		if err == nil {
			t.Error("ValidateFilePermissions() with non-existent file should fail on Unix")
		}
	}
}

func TestValidatePath_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "double dots in middle",
			path:    "/usr/../etc/passwd",
			wantErr: true,
		},
		{
			name:    "double dots at end",
			path:    "/tmp/..",
			wantErr: true,
		},
		{
			name:    "normal relative path",
			path:    "relative/path",
			wantErr: false,
		},
		{
			name:    "absolute windows path",
			path:    "C:\\Users\\test",
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

func TestSanitizeScriptName_AllDangerousChars(t *testing.T) {
	dangerousChars := []string{";", "&", "|", ">", "<", "`", "$", "(", ")", "{", "}", "[", "]", "\n", "\r"}

	for _, char := range dangerousChars {
		t.Run("char_"+char, func(t *testing.T) {
			scriptName := "test" + char + "malicious"
			err := SanitizeScriptName(scriptName)
			if err == nil {
				t.Errorf("SanitizeScriptName() should reject script with %q", char)
			}
		})
	}
}

func TestValidatePackageManager_Dotnet(t *testing.T) {
	err := ValidatePackageManager("dotnet")
	if err != nil {
		t.Errorf("ValidatePackageManager(\"dotnet\") should be valid, got error: %v", err)
	}
}
