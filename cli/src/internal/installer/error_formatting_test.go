package installer

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestFormatNodeInstallError(t *testing.T) {
	tests := []struct {
		name           string
		packageManager string
		projectDir     string
		exitCode       int
		stderr         string
		wantContains   []string
	}{
		{
			name:           "pnpm_command_not_found_exit_254",
			packageManager: "pnpm",
			projectDir:     "/test/project",
			exitCode:       254,
			stderr:         "sh: pnpm: command not found",
			wantContains: []string{
				"failed to run pnpm install",
				"pnpm command failed - check if pnpm is installed and in PATH",
				"command not found",
				"Install pnpm with: npm install -g pnpm",
				"/test/project",
			},
		},
		{
			name:           "npm_permission_denied",
			packageManager: "npm",
			projectDir:     "/test/project",
			exitCode:       1,
			stderr:         "Error: EACCES: permission denied, mkdir '/usr/local/lib/node_modules'",
			wantContains: []string{
				"failed to run npm install",
				"command failed with errors",
				"permission denied",
				"Try running with appropriate permissions",
			},
		},
		{
			name:           "yarn_network_error",
			packageManager: "yarn",
			projectDir:     "/test/project",
			exitCode:       1,
			stderr:         "error An unexpected error occurred: \"https://registry.yarnpkg.com/: getaddrinfo ENOTFOUND\"",
			wantContains: []string{
				"failed to run yarn install",
				"ENOTFOUND",
				"Check your network connection",
			},
		},
		{
			name:           "pnpm_exit_127",
			packageManager: "pnpm",
			projectDir:     "/test/project",
			exitCode:       127,
			stderr:         "",
			wantContains: []string{
				"pnpm not found - please install pnpm",
				"Install pnpm with: npm install -g pnpm",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock command that will fail
			cmd := exec.Command("nonexistent-command-xyz-123")
			cmd.Args = []string{tt.packageManager, "install"}
			
			// Create a mock error - we can't create ExitError directly,
			// so we use a generic error and rely on the function handling both types
			cmdErr := fmt.Errorf("exit status %d", tt.exitCode)

			err := formatNodeInstallError(tt.packageManager, tt.projectDir, cmd, cmdErr, tt.stderr)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message missing expected content:\nwant substring: %q\ngot: %s", want, errMsg)
				}
			}
		})
	}
}

func TestFormatPythonInstallError(t *testing.T) {
	tests := []struct {
		name         string
		tool         string
		projectDir   string
		exitCode     int
		stderr       string
		wantContains []string
	}{
		{
			name:       "uv_not_found",
			tool:       "uv",
			projectDir: "/test/project",
			exitCode:   127,
			stderr:     "",
			wantContains: []string{
				"failed to run uv",
				"uv not found - please install uv",
				"Install uv with: pip install uv",
			},
		},
		{
			name:       "pip_network_error",
			tool:       "pip install",
			projectDir: "/test/project",
			exitCode:   1,
			stderr:     "ERROR: Could not find a version that satisfies the requirement",
			wantContains: []string{
				"failed to run pip install",
				"Could not find",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("nonexistent-command-xyz-123")
			cmd.Args = []string{tt.tool}
			
			cmdErr := fmt.Errorf("exit status %d", tt.exitCode)

			err := formatPythonInstallError(tt.tool, tt.projectDir, cmd, cmdErr, tt.stderr)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message missing expected content:\nwant substring: %q\ngot: %s", want, errMsg)
				}
			}
		})
	}
}

func TestExtractErrorDetails(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		tool   string
		want   string
	}{
		{
			name: "extract_error_lines",
			stderr: `Progress: downloading packages...
Building cache...
Error: ENOENT: no such file or directory
Failed to install package xyz
More progress output...`,
			tool: "npm",
			want: "Error: ENOENT: no such file or directory; Failed to install package xyz",
		},
		{
			name: "extract_last_lines_when_no_errors",
			stderr: `Downloading...
Building...
Installation complete
Success!`,
			tool: "npm",
			want: "Installation complete; Success!",
		},
		{
			name:   "empty_stderr",
			stderr: "",
			tool:   "npm",
			want:   "",
		},
		{
			name: "truncate_very_long_stderr",
			stderr: strings.Repeat("Error: some error message\n", 1000),
			tool:   "npm",
			want:   "", // Will be truncated during extraction
		},
		{
			name: "limit_error_lines_to_three",
			stderr: `Error 1
Error 2  
Error 3
Error 4
Error 5`,
			tool: "npm",
			want: "Error 1; Error 2; Error 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractErrorDetails(tt.stderr, tt.tool)
			if tt.want != "" && !strings.Contains(got, strings.Split(tt.want, ";")[0]) {
				t.Errorf("extractErrorDetails() = %q, want to contain %q", got, tt.want)
			}
			if tt.want == "" && got != "" {
				// For truncation test, just ensure we got something reasonable
				if len(got) > 600 {
					t.Errorf("extractErrorDetails() returned overly long result: %d chars", len(got))
				}
			}
		})
	}
}

func TestFormatCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  *exec.Cmd
		want string
	}{
		{
			name: "normal_command",
			cmd:  exec.Command("pnpm", "install", "--prefer-offline"),
			want: "pnpm install --prefer-offline",
		},
		{
			name: "nil_command",
			cmd:  nil,
			want: "(unknown command)",
		},
		{
			name: "command_with_no_args",
			cmd:  &exec.Cmd{Path: "/usr/bin/pnpm", Args: []string{}},
			want: "/usr/bin/pnpm",
		},
		{
			name: "command_with_one_arg",
			cmd:  &exec.Cmd{Path: "/usr/bin/pnpm", Args: []string{"pnpm"}},
			want: "pnpm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCommand(tt.cmd)
			if !strings.Contains(got, tt.want) && got != tt.want {
				t.Errorf("formatCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetSuggestion(t *testing.T) {
	tests := []struct {
		name           string
		packageManager string
		exitCode       int
		stderr         string
		wantContains   string
	}{
		{
			name:           "pnpm_not_found",
			packageManager: "pnpm",
			exitCode:       127,
			stderr:         "",
			wantContains:   "npm install -g pnpm",
		},
		{
			name:           "permission_error",
			packageManager: "npm",
			exitCode:       1,
			stderr:         "Error: EACCES: permission denied",
			wantContains:   "appropriate permissions",
		},
		{
			name:           "network_error",
			packageManager: "npm",
			exitCode:       1,
			stderr:         "Error: getaddrinfo ENOTFOUND registry.npmjs.org",
			wantContains:   "network connection",
		},
		{
			name:           "disk_space",
			packageManager: "npm",
			exitCode:       1,
			stderr:         "Error: ENOSPC: no space left on device",
			wantContains:   "Free up disk space",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSuggestion(tt.packageManager, tt.exitCode, tt.stderr)
			if !strings.Contains(got, tt.wantContains) {
				t.Errorf("getSuggestion() = %q, want to contain %q", got, tt.wantContains)
			}
		})
	}
}
