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
	warning, err := ValidateFilePermissions(tmpFile)
	if err != nil {
		t.Errorf("ValidateFilePermissions() with 0644 should pass, got error: %v", err)
	}
	if warning != "" {
		t.Errorf("ValidateFilePermissions() with 0644 should not have warning, got: %v", warning)
	}

	// Skip world-writable test on Windows (uses ACLs)
	if runtime.GOOS != "windows" {
		// Make file world-writable
		if err := os.Chmod(tmpFile, 0666); err != nil {
			t.Fatalf("Failed to chmod file: %v", err)
		}

		// Test with insecure permissions (0666) - should fail in non-container environment
		// Clear any container env vars to ensure we're testing non-container behavior
		originalCodespaces := os.Getenv("CODESPACES")
		originalRemoteContainers := os.Getenv("REMOTE_CONTAINERS")
		originalK8s := os.Getenv("KUBERNETES_SERVICE_HOST")
		os.Unsetenv("CODESPACES")
		os.Unsetenv("REMOTE_CONTAINERS")
		os.Unsetenv("KUBERNETES_SERVICE_HOST")
		defer func() {
			if originalCodespaces != "" {
				os.Setenv("CODESPACES", originalCodespaces)
			}
			if originalRemoteContainers != "" {
				os.Setenv("REMOTE_CONTAINERS", originalRemoteContainers)
			}
			if originalK8s != "" {
				os.Setenv("KUBERNETES_SERVICE_HOST", originalK8s)
			}
		}()

		warning, err = ValidateFilePermissions(tmpFile)
		if err == nil {
			t.Error("ValidateFilePermissions() with 0666 should fail on Unix in non-container environment")
		}
		if warning != "" {
			t.Errorf("ValidateFilePermissions() error case should not have warning, got: %v", warning)
		}
	}

	// Test with non-existent file (only fails on Unix, Windows returns nil)
	if runtime.GOOS != "windows" {
		_, err = ValidateFilePermissions("/nonexistent/file")
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

func TestIsContainerEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected bool
	}{
		{
			name:     "no container environment",
			envVars:  map[string]string{},
			expected: false,
		},
		{
			name:     "GitHub Codespaces",
			envVars:  map[string]string{"CODESPACES": "true"},
			expected: true,
		},
		{
			name:     "CODESPACES not true",
			envVars:  map[string]string{"CODESPACES": "false"},
			expected: false,
		},
		{
			name:     "VS Code Dev Containers",
			envVars:  map[string]string{"REMOTE_CONTAINERS": "true"},
			expected: true,
		},
		{
			name:     "REMOTE_CONTAINERS not true",
			envVars:  map[string]string{"REMOTE_CONTAINERS": "false"},
			expected: false,
		},
		{
			name:     "Kubernetes",
			envVars:  map[string]string{"KUBERNETES_SERVICE_HOST": "10.0.0.1"},
			expected: true,
		},
		{
			name:     "Multiple container indicators",
			envVars:  map[string]string{"CODESPACES": "true", "REMOTE_CONTAINERS": "true"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			originalCodespaces := os.Getenv("CODESPACES")
			originalRemoteContainers := os.Getenv("REMOTE_CONTAINERS")
			originalK8s := os.Getenv("KUBERNETES_SERVICE_HOST")

			// Clear all container env vars
			os.Unsetenv("CODESPACES")
			os.Unsetenv("REMOTE_CONTAINERS")
			os.Unsetenv("KUBERNETES_SERVICE_HOST")

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Restore original env vars after test
			defer func() {
				os.Unsetenv("CODESPACES")
				os.Unsetenv("REMOTE_CONTAINERS")
				os.Unsetenv("KUBERNETES_SERVICE_HOST")
				if originalCodespaces != "" {
					os.Setenv("CODESPACES", originalCodespaces)
				}
				if originalRemoteContainers != "" {
					os.Setenv("REMOTE_CONTAINERS", originalRemoteContainers)
				}
				if originalK8s != "" {
					os.Setenv("KUBERNETES_SERVICE_HOST", originalK8s)
				}
			}()

			result := IsContainerEnvironment()
			if result != tt.expected {
				t.Errorf("IsContainerEnvironment() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestValidateFilePermissions_ContainerEnvironment(t *testing.T) {
	// Skip on Windows as it uses ACLs
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	tmpFile := t.TempDir() + "/test.txt"

	// Create a test file with world-writable permissions
	if err := os.WriteFile(tmpFile, []byte("test"), 0666); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		envVars     map[string]string
		wantWarning bool
		wantErr     bool
	}{
		{
			name:        "non-container environment - should error",
			envVars:     map[string]string{},
			wantWarning: false,
			wantErr:     true,
		},
		{
			name:        "Codespaces - should warn",
			envVars:     map[string]string{"CODESPACES": "true"},
			wantWarning: true,
			wantErr:     false,
		},
		{
			name:        "Dev Containers - should warn",
			envVars:     map[string]string{"REMOTE_CONTAINERS": "true"},
			wantWarning: true,
			wantErr:     false,
		},
		{
			name:        "Kubernetes - should warn",
			envVars:     map[string]string{"KUBERNETES_SERVICE_HOST": "10.0.0.1"},
			wantWarning: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			originalCodespaces := os.Getenv("CODESPACES")
			originalRemoteContainers := os.Getenv("REMOTE_CONTAINERS")
			originalK8s := os.Getenv("KUBERNETES_SERVICE_HOST")

			// Clear all container env vars
			os.Unsetenv("CODESPACES")
			os.Unsetenv("REMOTE_CONTAINERS")
			os.Unsetenv("KUBERNETES_SERVICE_HOST")

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Restore original env vars after test
			defer func() {
				os.Unsetenv("CODESPACES")
				os.Unsetenv("REMOTE_CONTAINERS")
				os.Unsetenv("KUBERNETES_SERVICE_HOST")
				if originalCodespaces != "" {
					os.Setenv("CODESPACES", originalCodespaces)
				}
				if originalRemoteContainers != "" {
					os.Setenv("REMOTE_CONTAINERS", originalRemoteContainers)
				}
				if originalK8s != "" {
					os.Setenv("KUBERNETES_SERVICE_HOST", originalK8s)
				}
			}()

			warning, err := ValidateFilePermissions(tmpFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilePermissions() error = %v, wantErr %v", err, tt.wantErr)
			}

			if (warning != "") != tt.wantWarning {
				t.Errorf("ValidateFilePermissions() warning = %q, wantWarning %v", warning, tt.wantWarning)
			}

			// Verify warning message format
			if tt.wantWarning && warning != "" {
				if !strings.Contains(warning, "world-writable permissions") {
					t.Errorf("Warning should mention 'world-writable permissions', got: %s", warning)
				}
				if !strings.Contains(warning, "container environments") {
					t.Errorf("Warning should mention 'container environments', got: %s", warning)
				}
				if !strings.Contains(warning, "chmod 644") {
					t.Errorf("Warning should include fix command 'chmod 644', got: %s", warning)
				}
			}
		})
	}
}

func TestValidateFilePermissions_SecurePermissions_ContainerEnvironment(t *testing.T) {
	// Skip on Windows as it uses ACLs
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	tmpFile := t.TempDir() + "/test.txt"

	// Create a test file with secure permissions
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Save original env vars
	originalCodespaces := os.Getenv("CODESPACES")
	os.Setenv("CODESPACES", "true")
	defer func() {
		if originalCodespaces != "" {
			os.Setenv("CODESPACES", originalCodespaces)
		} else {
			os.Unsetenv("CODESPACES")
		}
	}()

	// Even in container environment, secure permissions should not produce warning
	warning, err := ValidateFilePermissions(tmpFile)
	if err != nil {
		t.Errorf("ValidateFilePermissions() with 0644 should not error, got: %v", err)
	}
	if warning != "" {
		t.Errorf("ValidateFilePermissions() with 0644 should not warn, got: %v", warning)
	}
}
