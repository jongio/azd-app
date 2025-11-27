package installer

import (
	"errors"
	"testing"
)

func TestDependencyInstallError(t *testing.T) {
	underlyingErr := errors.New("command failed")
	err := &DependencyInstallError{
		ProjectType:    "Node.js",
		ProjectDir:     "/path/to/project",
		PackageManager: "npm",
		Command:        "npm install",
		Err:            underlyingErr,
	}

	// Test Error() method
	expected := "failed to install Node.js dependencies in /path/to/project using npm (command: npm install): command failed"
	got := err.Error()
	if got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}

	// Test Unwrap() method
	if unwrapped := err.Unwrap(); unwrapped != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}
}

func TestVirtualEnvError(t *testing.T) {
	underlyingErr := errors.New("venv creation failed")
	err := &VirtualEnvError{
		ProjectDir: "/path/to/python/project",
		Tool:       "uv",
		Err:        underlyingErr,
	}

	// Test Error() method
	expected := "failed to create virtual environment in /path/to/python/project using uv: venv creation failed"
	got := err.Error()
	if got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}

	// Test Unwrap() method
	if unwrapped := err.Unwrap(); unwrapped != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}
}

func TestDependencyInstallError_DifferentProjectTypes(t *testing.T) {
	tests := []struct {
		name        string
		projectType string
		manager     string
		command     string
	}{
		{
			name:        "Python with pip",
			projectType: "Python",
			manager:     "pip",
			command:     "pip install -r requirements.txt",
		},
		{
			name:        ".NET with dotnet",
			projectType: ".NET",
			manager:     "dotnet",
			command:     "dotnet restore",
		},
		{
			name:        "Node.js with pnpm",
			projectType: "Node.js",
			manager:     "pnpm",
			command:     "pnpm install",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &DependencyInstallError{
				ProjectType:    tt.projectType,
				ProjectDir:     "/test",
				PackageManager: tt.manager,
				Command:        tt.command,
				Err:            errors.New("test error"),
			}

			errStr := err.Error()
			// Verify key information is in error message
			if !contains(errStr, tt.projectType) {
				t.Errorf("Error message missing project type %q: %s", tt.projectType, errStr)
			}
			if !contains(errStr, tt.manager) {
				t.Errorf("Error message missing package manager %q: %s", tt.manager, errStr)
			}
			if !contains(errStr, tt.command) {
				t.Errorf("Error message missing command %q: %s", tt.command, errStr)
			}
		})
	}
}

func TestVirtualEnvError_DifferentTools(t *testing.T) {
	tools := []string{"uv", "poetry", "virtualenv", "venv"}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			err := &VirtualEnvError{
				ProjectDir: "/test/project",
				Tool:       tool,
				Err:        errors.New("test error"),
			}

			errStr := err.Error()
			if !contains(errStr, tool) {
				t.Errorf("Error message missing tool %q: %s", tool, errStr)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
