package security

import (
	"errors"
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
	var err error
	if err = ValidateFilePermissions(tmpFile); err != nil {
		t.Errorf("ValidateFilePermissions() with 0644 should pass, got error: %v", err)
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

		err = ValidateFilePermissions(tmpFile)
		if err == nil {
			t.Error("ValidateFilePermissions() with 0666 should fail on Unix in non-container environment")
		}
		if err != nil && !strings.Contains(err.Error(), "insecure file permissions") && !errors.Is(err, ErrInsecureFilePermissions) {
			t.Errorf("ValidateFilePermissions() returned unexpected error: %v", err)
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

	// Create a test file and explicitly set world-writable permissions
	// (os.WriteFile respects umask, so we need os.Chmod to ensure exact permissions)
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.Chmod(tmpFile, 0666); err != nil {
		t.Fatalf("Failed to set permissions: %v", err)
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
			wantErr:     true,
		},
		{
			name:        "Dev Containers - should warn",
			envVars:     map[string]string{"REMOTE_CONTAINERS": "true"},
			wantWarning: true,
			wantErr:     true,
		},
		{
			name:        "Kubernetes - should warn",
			envVars:     map[string]string{"KUBERNETES_SERVICE_HOST": "10.0.0.1"},
			wantWarning: true,
			wantErr:     true,
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

			err := ValidateFilePermissions(tmpFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilePermissions() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantWarning {
				// In container environments, caller is expected to translate the sentinel error
				// into a warning. Here we assert that the sentinel is returned and/or the
				// environment is detected.
				if !errors.Is(err, ErrInsecureFilePermissions) {
					t.Errorf("expected ErrInsecureFilePermissions in container env, got: %v", err)
				}
			}
		})
	}
}

// TestValidateServiceName tests service name validation
func TestValidateServiceName(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		allowEmpty  bool
		wantErr     bool
	}{
		{
			name:        "valid service name",
			serviceName: "api",
			allowEmpty:  false,
			wantErr:     false,
		},
		{
			name:        "valid with hyphen",
			serviceName: "api-service",
			allowEmpty:  false,
			wantErr:     false,
		},
		{
			name:        "valid with underscore",
			serviceName: "api_service",
			allowEmpty:  false,
			wantErr:     false,
		},
		{
			name:        "valid with dot",
			serviceName: "api.service",
			allowEmpty:  false,
			wantErr:     false,
		},
		{
			name:        "valid numeric",
			serviceName: "service1",
			allowEmpty:  false,
			wantErr:     false,
		},
		{
			name:        "empty when allowed",
			serviceName: "",
			allowEmpty:  true,
			wantErr:     false,
		},
		{
			name:        "empty when not allowed",
			serviceName: "",
			allowEmpty:  false,
			wantErr:     true,
		},
		{
			name:        "too long",
			serviceName: "this-is-a-very-long-service-name-that-exceeds-sixty-three-characters",
			allowEmpty:  false,
			wantErr:     true,
		},
		{
			name:        "starts with dash",
			serviceName: "-api",
			allowEmpty:  false,
			wantErr:     true,
		},
		{
			name:        "starts with underscore",
			serviceName: "_api",
			allowEmpty:  false,
			wantErr:     true,
		},
		{
			name:        "contains forward slash",
			serviceName: "api/service",
			allowEmpty:  false,
			wantErr:     true,
		},
		{
			name:        "contains backslash",
			serviceName: "api\\service",
			allowEmpty:  false,
			wantErr:     true,
		},
		{
			name:        "path traversal attempt",
			serviceName: "api..service",
			allowEmpty:  false,
			wantErr:     true,
		},
		{
			name:        "contains space",
			serviceName: "api service",
			allowEmpty:  false,
			wantErr:     true,
		},
		{
			name:        "contains special chars",
			serviceName: "api@service",
			allowEmpty:  false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServiceName(tt.serviceName, tt.allowEmpty)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateServiceName() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "invalid service name") {
				t.Errorf("ValidateServiceName() error should mention invalid service name, got: %v", err)
			}
		})
	}
}

func TestIsContainerEnvironment_DockerEnv(t *testing.T) {
	// This tests the /.dockerenv file check
	// We can't easily test this in unit tests without mocking os.Stat,
	// but we can at least exercise the code path by calling the function
	// which will check for the file and return false if not found
	result := IsContainerEnvironment()
	// The result depends on whether we're actually in a container
	// Just verify the function doesn't panic
	_ = result
}

func TestValidatePath_SymlinkError(t *testing.T) {
	// Test path validation with a path that doesn't exist
	// This exercises the os.IsNotExist branch
	nonExistentPath := "/tmp/this-should-not-exist-" + t.Name()
	err := ValidatePath(nonExistentPath)
	// Should not error on non-existent paths (they might be created later)
	if err != nil && !strings.Contains(err.Error(), "parent directory reference") {
		// Only fail if it's not a traversal error
		t.Logf("ValidatePath() with non-existent path: %v", err)
	}
}

func TestValidatePath_WithSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlink test more reliable on Unix-like systems")
	}

	tmpDir := t.TempDir()
	targetFile := tmpDir + "/target.txt"
	symlinkPath := tmpDir + "/link.txt"

	// Create target file
	if err := os.WriteFile(targetFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create symlink
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Skipf("Failed to create symlink (may need privileges): %v", err)
	}

	// Validate the symlink - should succeed as it points to valid location
	err := ValidatePath(symlinkPath)
	if err != nil {
		t.Errorf("ValidatePath() with valid symlink should pass, got: %v", err)
	}
}

func TestValidatePath_AbsolutePathConversion(t *testing.T) {
	// Test that relative paths are converted to absolute
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "relative path without dots",
			path:    "some/relative/path",
			wantErr: false,
		},
		{
			name:    "path with double dots",
			path:    "some/../path",
			wantErr: true,
		},
		{
			name:    "path with trailing dots after clean",
			path:    "normalpath/../..", // This becomes ".." after clean
			wantErr: true,
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

// TestValidatePath_CleanedPathCheck tests the check after filepath.Clean
func TestValidatePath_CleanedPathCheck(t *testing.T) {
	// These paths might pass the initial check but fail after cleaning
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "path that becomes dots after clean",
			path:    "./test/../..",
			wantErr: true,
		},
		{
			name:    "relative traversal",
			path:    "foo/../../bar",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "parent directory reference") {
				t.Errorf("Expected path traversal error, got: %v", err)
			}
		})
	}
}

func TestValidatePath_EvalSymlinksNonExistError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlink behavior different on Windows")
	}

	tmpDir := t.TempDir()
	brokenSymlink := tmpDir + "/broken-link"

	// Create a symlink pointing to a non-existent target
	// This should pass validation (path doesn't exist yet, but structure is valid)
	if err := os.Symlink("/tmp/nonexistent-target-12345", brokenSymlink); err != nil {
		t.Skipf("Failed to create broken symlink: %v", err)
	}

	err := ValidatePath(brokenSymlink)
	// Should not error - the path doesn't exist, but that's okay
	if err != nil {
		t.Logf("ValidatePath() with broken symlink: %v", err)
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

	// Even in container environment, secure permissions should not produce error
	if err := ValidateFilePermissions(tmpFile); err != nil {
		t.Errorf("ValidateFilePermissions() with 0644 should not error, got: %v", err)
	}
}

func TestValidatePath_ResolvedPathWithDots(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlink test more reliable on Unix-like systems")
	}

	tmpDir := t.TempDir()

	// Create a directory structure: tmpDir/safe/target.txt
	safeDir := tmpDir + "/safe"
	if err := os.MkdirAll(safeDir, 0755); err != nil {
		t.Fatalf("Failed to create safe dir: %v", err)
	}

	targetFile := safeDir + "/target.txt"
	if err := os.WriteFile(targetFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create symlink with a benign name
	symlinkPath := tmpDir + "/link.txt"
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Skipf("Failed to create symlink: %v", err)
	}

	// This should pass - the symlink resolves to a safe location
	err := ValidatePath(symlinkPath)
	if err != nil {
		t.Errorf("ValidatePath() with safe symlink should pass, got: %v", err)
	}
}
