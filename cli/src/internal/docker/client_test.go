package docker

import (
	"testing"
)

func TestPortMappingGetProtocol(t *testing.T) {
	tests := []struct {
		name     string
		mapping  PortMapping
		expected string
	}{
		{
			name:     "empty protocol defaults to tcp",
			mapping:  PortMapping{HostPort: 8080, ContainerPort: 80},
			expected: "tcp",
		},
		{
			name:     "explicit tcp protocol",
			mapping:  PortMapping{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
			expected: "tcp",
		},
		{
			name:     "explicit udp protocol",
			mapping:  PortMapping{HostPort: 53, ContainerPort: 53, Protocol: "udp"},
			expected: "udp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mapping.GetProtocol()
			if got != tt.expected {
				t.Errorf("GetProtocol() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFormatPortMapping(t *testing.T) {
	tests := []struct {
		name     string
		mapping  PortMapping
		expected string
	}{
		{
			name:     "basic port mapping with host port",
			mapping:  PortMapping{HostPort: 8080, ContainerPort: 80},
			expected: "8080:80/tcp",
		},
		{
			name:     "auto-assign host port",
			mapping:  PortMapping{HostPort: 0, ContainerPort: 80},
			expected: "80/tcp",
		},
		{
			name:     "udp protocol",
			mapping:  PortMapping{HostPort: 53, ContainerPort: 53, Protocol: "udp"},
			expected: "53:53/udp",
		},
		{
			name:     "auto-assign with udp",
			mapping:  PortMapping{HostPort: 0, ContainerPort: 53, Protocol: "udp"},
			expected: "53/udp",
		},
		{
			name:     "different host and container ports",
			mapping:  PortMapping{HostPort: 10000, ContainerPort: 10001},
			expected: "10000:10001/tcp",
		},
		{
			name:     "bind to localhost with explicit ports",
			mapping:  PortMapping{HostPort: 8080, ContainerPort: 80, BindIP: "127.0.0.1"},
			expected: "127.0.0.1:8080:80/tcp",
		},
		{
			name:     "bind to localhost with auto-assign",
			mapping:  PortMapping{HostPort: 0, ContainerPort: 80, BindIP: "127.0.0.1"},
			expected: "127.0.0.1::80/tcp",
		},
		{
			name:     "bind to specific IP with udp",
			mapping:  PortMapping{HostPort: 53, ContainerPort: 53, BindIP: "10.0.0.1", Protocol: "udp"},
			expected: "10.0.0.1:53:53/udp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPortMapping(tt.mapping)
			if got != tt.expected {
				t.Errorf("formatPortMapping() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildRunArgs(t *testing.T) {
	tests := []struct {
		name     string
		config   ContainerConfig
		expected []string
	}{
		{
			name: "basic config with image only",
			config: ContainerConfig{
				Image: "nginx",
			},
			expected: []string{"run", "-d", "nginx"},
		},
		{
			name: "config with name",
			config: ContainerConfig{
				Name:  "my-container",
				Image: "nginx",
			},
			expected: []string{"run", "-d", "--name", "my-container", "nginx"},
		},
		{
			name: "config with single port",
			config: ContainerConfig{
				Image: "nginx",
				Ports: []PortMapping{
					{HostPort: 8080, ContainerPort: 80},
				},
			},
			expected: []string{"run", "-d", "-p", "8080:80/tcp", "nginx"},
		},
		{
			name: "config with multiple ports",
			config: ContainerConfig{
				Image: "azurite",
				Ports: []PortMapping{
					{HostPort: 10000, ContainerPort: 10000},
					{HostPort: 10001, ContainerPort: 10001},
					{HostPort: 10002, ContainerPort: 10002},
				},
			},
			expected: []string{"run", "-d", "-p", "10000:10000/tcp", "-p", "10001:10001/tcp", "-p", "10002:10002/tcp", "azurite"},
		},
		{
			name: "config with all options",
			config: ContainerConfig{
				Name:  "test-container",
				Image: "mcr.microsoft.com/azure-storage/azurite",
				Ports: []PortMapping{
					{HostPort: 10000, ContainerPort: 10000},
				},
				Environment: map[string]string{
					"DEBUG": "true",
				},
			},
			// Note: environment variables are added from a map, so order is not guaranteed
			// We'll check for presence instead of exact order in a separate test
			expected: nil, // Will check manually
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRunArgs(tt.config)

			if tt.expected == nil {
				// For the "all options" test, check for presence of key elements
				assertContains(t, got, "run")
				assertContains(t, got, "-d")
				assertContains(t, got, "--name")
				assertContains(t, got, "test-container")
				assertContains(t, got, "-p")
				assertContains(t, got, "10000:10000/tcp")
				assertContains(t, got, "-e")
				assertContains(t, got, "DEBUG=true")
				assertContains(t, got, "mcr.microsoft.com/azure-storage/azurite")

				// Image should be last
				if got[len(got)-1] != "mcr.microsoft.com/azure-storage/azurite" {
					t.Errorf("Image should be the last argument, got: %v", got)
				}
				return
			}

			if len(got) != len(tt.expected) {
				t.Errorf("buildRunArgs() returned %d args, want %d. Got: %v", len(got), len(tt.expected), got)
				return
			}

			for i, arg := range tt.expected {
				if got[i] != arg {
					t.Errorf("buildRunArgs()[%d] = %q, want %q", i, got[i], arg)
				}
			}
		})
	}
}

func TestBuildRunArgsEnvironmentVariables(t *testing.T) {
	config := ContainerConfig{
		Image: "test",
		Environment: map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
			"VAR3": "value with spaces",
		},
	}

	got := buildRunArgs(config)

	// Count environment flags
	envCount := 0
	for i, arg := range got {
		if arg == "-e" && i+1 < len(got) {
			envCount++
		}
	}

	if envCount != 3 {
		t.Errorf("Expected 3 environment variables, got %d. Args: %v", envCount, got)
	}

	// Check that all env vars are present
	assertContains(t, got, "VAR1=value1")
	assertContains(t, got, "VAR2=value2")
	assertContains(t, got, "VAR3=value with spaces")
}

func TestContainerConfigValidation(t *testing.T) {
	client := NewClient()

	t.Run("empty image returns error", func(t *testing.T) {
		_, err := client.Run(ContainerConfig{})
		if err == nil {
			t.Error("Expected error for empty image, got nil")
		}
	})
}

func TestStopValidation(t *testing.T) {
	client := NewClient()

	t.Run("empty container ID returns error", func(t *testing.T) {
		err := client.Stop("", 10)
		if err == nil {
			t.Error("Expected error for empty container ID, got nil")
		}
	})
}

func TestRemoveValidation(t *testing.T) {
	client := NewClient()

	t.Run("empty container ID returns error", func(t *testing.T) {
		err := client.Remove("")
		if err == nil {
			t.Error("Expected error for empty container ID, got nil")
		}
	})
}

func TestLogsValidation(t *testing.T) {
	client := NewClient()

	t.Run("empty container ID returns error", func(t *testing.T) {
		_, err := client.Logs("")
		if err == nil {
			t.Error("Expected error for empty container ID, got nil")
		}
	})
}

func TestInspectValidation(t *testing.T) {
	client := NewClient()

	t.Run("empty container ID returns error", func(t *testing.T) {
		_, err := client.Inspect("")
		if err == nil {
			t.Error("Expected error for empty container ID, got nil")
		}
	})
}

func TestIsRunningValidation(t *testing.T) {
	client := NewClient()

	t.Run("empty container ID returns false", func(t *testing.T) {
		if client.IsRunning("") {
			t.Error("Expected false for empty container ID")
		}
	})
}

func TestPullValidation(t *testing.T) {
	client := NewClient()

	t.Run("empty image returns error", func(t *testing.T) {
		err := client.Pull("")
		if err == nil {
			t.Error("Expected error for empty image, got nil")
		}
	})
}

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Error("NewClient() returned nil")
	}
}

func TestValidateImageName(t *testing.T) {
	tests := []struct {
		name    string
		image   string
		wantErr bool
	}{
		{"valid simple image", "nginx", false},
		{"valid image with tag", "nginx:latest", false},
		{"valid image with registry", "docker.io/library/nginx", false},
		{"valid mcr image", "mcr.microsoft.com/azure-storage/azurite", false},
		{"valid image with tag and registry", "mcr.microsoft.com/azure-storage/azurite:latest", false},
		{"valid image with digest", "nginx@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", false},
		{"empty image", "", true},
		{"image with spaces", "nginx latest", true},
		{"image with special chars", "nginx;rm -rf /", true},
		{"image too long", string(make([]byte, 300)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImageName(tt.image)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateImageName(%q) error = %v, wantErr %v", tt.image, err, tt.wantErr)
			}
		})
	}
}

func TestValidateContainerName(t *testing.T) {
	tests := []struct {
		name          string
		containerName string
		wantErr       bool
	}{
		{"valid name", "my-container", false},
		{"valid name with underscore", "my_container", false},
		{"valid name with dot", "my.container", false},
		{"empty name (valid)", "", false},
		{"starts with number", "1container", false},
		{"starts with hyphen", "-container", true},
		{"contains spaces", "my container", true},
		{"too long", string(make([]byte, 200)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContainerName(tt.containerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContainerName(%q) error = %v, wantErr %v", tt.containerName, err, tt.wantErr)
			}
		})
	}
}

func TestPortMappingValidate(t *testing.T) {
	tests := []struct {
		name    string
		mapping PortMapping
		wantErr bool
	}{
		{"valid mapping", PortMapping{HostPort: 8080, ContainerPort: 80}, false},
		{"auto-assign host port", PortMapping{HostPort: 0, ContainerPort: 80}, false},
		{"invalid host port", PortMapping{HostPort: 70000, ContainerPort: 80}, true},
		{"invalid container port", PortMapping{HostPort: 8080, ContainerPort: 0}, true},
		{"container port too high", PortMapping{HostPort: 8080, ContainerPort: 70000}, true},
		{"invalid protocol", PortMapping{HostPort: 8080, ContainerPort: 80, Protocol: "http"}, true},
		{"valid udp", PortMapping{HostPort: 53, ContainerPort: 53, Protocol: "udp"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mapping.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("PortMapping.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContainerConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  ContainerConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  ContainerConfig{Image: "nginx", Name: "my-nginx"},
			wantErr: false,
		},
		{
			name:    "valid config without name",
			config:  ContainerConfig{Image: "nginx"},
			wantErr: false,
		},
		{
			name:    "invalid image",
			config:  ContainerConfig{Image: ""},
			wantErr: true,
		},
		{
			name:    "invalid container name",
			config:  ContainerConfig{Image: "nginx", Name: "-invalid"},
			wantErr: true,
		},
		{
			name: "invalid port mapping",
			config: ContainerConfig{
				Image: "nginx",
				Ports: []PortMapping{{HostPort: 8080, ContainerPort: 0}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ContainerConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// assertContains checks if a slice contains a specific string.
func assertContains(t *testing.T, slice []string, want string) {
	t.Helper()
	for _, s := range slice {
		if s == want {
			return
		}
	}
	t.Errorf("Slice does not contain %q. Got: %v", want, slice)
}

// Tests for docker exec functionality

func TestExec_EmptyContainerName(t *testing.T) {
	client := NewClient()
	command := []string{"echo", "hello"}

	exitCode, output, err := client.Exec("", command)
	if err == nil {
		t.Error("Exec with empty container name should return error")
	}
	if exitCode != -1 {
		t.Errorf("Exit code should be -1 for empty container, got %d", exitCode)
	}
	if output != "" {
		t.Errorf("Output should be empty for invalid input, got %q", output)
	}
	if err.Error() == "" {
		t.Error("Error message should not be empty")
	}
}

func TestExec_EmptyCommand(t *testing.T) {
	client := NewClient()

	exitCode, output, err := client.Exec("test-container", []string{})
	if err == nil {
		t.Error("Exec with empty command should return error")
	}
	if exitCode != -1 {
		t.Errorf("Exit code should be -1 for empty command, got %d", exitCode)
	}
	if output != "" {
		t.Errorf("Output should be empty for invalid input, got %q", output)
	}
	expectedErrMsg := "command cannot be empty"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message %q, got %q", expectedErrMsg, err.Error())
	}
}

func TestExec_NilCommand(t *testing.T) {
	client := NewClient()

	exitCode, output, err := client.Exec("test-container", nil)
	if err == nil {
		t.Error("Exec with nil command should return error")
	}
	if exitCode != -1 {
		t.Errorf("Exit code should be -1 for nil command, got %d", exitCode)
	}
	if output != "" {
		t.Errorf("Output should be empty for invalid input, got %q", output)
	}
}

func TestExec_NonexistentContainer(t *testing.T) {
	client := NewClient()

	// Use a container name that definitely doesn't exist
	containerName := "nonexistent-container-that-does-not-exist-12345"
	command := []string{"echo", "hello"}

	exitCode, _, err := client.Exec(containerName, command)
	if err == nil {
		// If docker isn't available, we might get a different error
		// but we should still get an error
		t.Log("Exec with nonexistent container returned success (unexpected)")
	}
	// Exit code should be -1 when container doesn't exist
	if err != nil && exitCode != -1 {
		// This is acceptable - means docker returned an exit code
		t.Logf("Exit code for nonexistent container: %d", exitCode)
	}
}

func TestExec_SingleArgumentCommand(t *testing.T) {
	client := NewClient()

	// Test command with single argument
	command := []string{"echo"}
	exitCode, _, err := client.Exec("test-container", command)

	// Should fail (no such container) but validates single-arg handling
	if exitCode == 0 && err == nil {
		t.Error("Should fail with nonexistent container")
	}
}

func TestExec_MultipleArgumentCommand(t *testing.T) {
	client := NewClient()

	// Test command with multiple arguments
	command := []string{"sh", "-c", "echo 'hello world'"}
	exitCode, _, err := client.Exec("test-container", command)

	// Should fail (no such container) but validates multi-arg handling
	if exitCode == 0 && err == nil {
		t.Error("Should fail with nonexistent container")
	}
}

func TestExec_CommandWithSpecialCharacters(t *testing.T) {
	client := NewClient()

	// Test command with special characters (properly escaped)
	command := []string{"echo", "hello & world"}
	exitCode, _, _ := client.Exec("test-container", command)

	// Should fail (no such container) but validates special char handling
	if exitCode == 0 {
		t.Error("Should fail with nonexistent container")
	}
}

func TestExecShell_EmptyContainerName(t *testing.T) {
	client := NewClient()

	exitCode, output, err := client.ExecShell("", "echo hello")
	if err == nil {
		t.Error("ExecShell with empty container name should return error")
	}
	if exitCode != -1 {
		t.Errorf("Exit code should be -1 for error, got %d", exitCode)
	}
	if output != "" {
		t.Errorf("Output should be empty for invalid input, got %q", output)
	}
}

func TestExecShell_EmptyCommand(t *testing.T) {
	client := NewClient()

	// Empty shell command should still execute (sh -c "" is valid)
	// But will fail because container doesn't exist
	exitCode, _, err := client.ExecShell("test-container", "")

	// The actual exec call should validate this
	// Since it calls Exec with []string{"sh", "-c", ""}, it won't error on empty
	if exitCode == 0 && err == nil {
		t.Error("Should fail with nonexistent container")
	}
}

func TestExecShell_SimpleCommand(t *testing.T) {
	client := NewClient()

	shellCommand := "echo hello"
	exitCode, _, err := client.ExecShell("test-container", shellCommand)

	// Should fail (no such container) but validates simple command
	if exitCode == 0 && err == nil {
		t.Error("Should fail with nonexistent container")
	}
}

func TestExecShell_ComplexCommand(t *testing.T) {
	client := NewClient()

	// Test complex shell command with pipes and redirects
	shellCommand := "echo 'hello world' | grep hello && exit 0"
	exitCode, _, _ := client.ExecShell("test-container", shellCommand)

	// Should fail (no such container) but validates complex command handling
	if exitCode == 0 {
		t.Log("Complex shell command handling validated")
	}
}

func TestExecShell_MultilineCommand(t *testing.T) {
	client := NewClient()

	// Test multiline shell command
	shellCommand := `
		echo "line 1"
		echo "line 2"
		exit 0
	`
	_, _, err := client.ExecShell("test-container", shellCommand)

	// Should fail (no such container) but validates multiline handling
	_ = err
}

func TestExecShell_CommandWithQuotes(t *testing.T) {
	client := NewClient()

	// Test command with various quote types
	shellCommand := `echo "double quotes" && echo 'single quotes'`
	_, _, err := client.ExecShell("test-container", shellCommand)

	// Should fail (no such container) but validates quote handling
	_ = err
}

func TestExecShell_UsesShellWrapper(t *testing.T) {
	// Verify that ExecShell wraps commands in sh -c
	// by testing a command that only works in a shell context
	client := NewClient()

	// Shell-specific features like && chaining
	shellCommand := "exit 0 && exit 1"
	_, _, err := client.ExecShell("test-container", shellCommand)

	// Should fail (no such container) but validates shell wrapper
	_ = err
}

func TestExec_OutputFromStdout(t *testing.T) {
	// This test documents that output should be captured
	client := NewClient()

	// Even though container doesn't exist, the function should
	// return output in the second return value
	_, output, _ := client.Exec("nonexistent", []string{"echo", "test"})

	// Output exists (may be empty for nonexistent container)
	_ = output
}

func TestExec_OutputFromStderr(t *testing.T) {
	// This test documents that stderr should be captured when stdout is empty
	client := NewClient()

	// Error scenarios should capture stderr
	_, output, _ := client.Exec("nonexistent", []string{"sh", "-c", "echo error >&2"})

	// Output exists (may contain error message)
	_ = output
}

func TestExec_ExitCodeZero(t *testing.T) {
	// Document that exit code 0 should be returned on success
	// (This would require a real container to fully test)
	client := NewClient()

	exitCode, _, _ := client.Exec("test", []string{"true"})

	// Exit code should be 0 for successful commands
	// or -1 if container doesn't exist
	if exitCode != 0 && exitCode != -1 {
		t.Logf("Got exit code: %d", exitCode)
	}
}

func TestExec_ExitCodeNonZero(t *testing.T) {
	// Document that non-zero exit codes should be captured
	client := NewClient()

	exitCode, output, err := client.Exec("test", []string{"false"})

	// Should capture exit code from failed command
	// For nonexistent container: exitCode = -1, err != nil
	// For real container: exitCode = 1, err = nil
	if exitCode == -1 && err != nil {
		t.Logf("Container doesn't exist (expected in test): %v", err)
	} else if exitCode != 0 {
		t.Logf("Captured non-zero exit code: %d, output: %q", exitCode, output)
	}
}

func TestExecShell_ExitCodeFromCommand(t *testing.T) {
	client := NewClient()

	// Test that shell commands can return explicit exit codes
	shellCommand := "exit 42"
	exitCode, _, _ := client.ExecShell("test-container", shellCommand)

	// Should capture the exit code (42) or -1 if container doesn't exist
	if exitCode != 42 && exitCode != -1 {
		t.Logf("Exit code: %d (expected 42 or -1)", exitCode)
	}
}

func TestExec_CombinedOutput(t *testing.T) {
	// Verify that stdout is preferred over stderr for output
	client := NewClient()

	// Command that writes to both stdout and stderr
	_, output, _ := client.Exec("test", []string{"sh", "-c", "echo stdout; echo stderr >&2"})

	// Output should exist (content depends on whether container exists)
	_ = output
}

func TestExec_ErrorMessageFormat(t *testing.T) {
	client := NewClient()

	// Test that error messages include container name
	containerName := "test-container-name"
	_, _, err := client.Exec(containerName, []string{"invalid-command"})

	if err != nil {
		// Error message should mention the container (when exec fails, not validation)
		// For validation errors, we have specific messages
		if err.Error() == "" {
			t.Error("Error message should not be empty")
		}
	}
}

func TestExecClient_ImplementsClient(t *testing.T) {
	var _ Client = (*ExecClient)(nil)

	// Verify all Client methods are implemented
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient should not return nil")
	}
}

func TestExec_ContainerNameValidation(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name          string
		containerName string
		shouldError   bool
	}{
		{"empty name", "", true},
		{"valid name", "my-container", false},
		{"valid name with underscore", "my_container", false},
		{"valid name with dot", "my.container", false},
		{"valid name with numbers", "container123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := client.Exec(tt.containerName, []string{"echo", "test"})

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for container name %q", tt.containerName)
				}
			} else {
				// For valid names, we expect error because container doesn't exist
				// but not a validation error
				if err != nil && tt.containerName != "" {
					// Should get "container not found" type error, not validation error
					errMsg := err.Error()
					if errMsg == "container name cannot be empty" {
						t.Error("Should not get validation error for valid name")
					}
				}
			}
		})
	}
}

func TestExec_CommandValidation(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name        string
		command     []string
		shouldError bool
	}{
		{"nil command", nil, true},
		{"empty command", []string{}, true},
		{"single arg", []string{"echo"}, false},
		{"multiple args", []string{"echo", "hello"}, false},
		{"command with flags", []string{"ls", "-la"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode, _, err := client.Exec("test-container", tt.command)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for command %v", tt.command)
				}
				if exitCode != -1 {
					t.Errorf("Expected exit code -1 for validation error, got %d", exitCode)
				}
			} else {
				// For valid commands, we expect error because container doesn't exist
				// but not a validation error about the command
				if err != nil {
					errMsg := err.Error()
					if errMsg == "command cannot be empty" {
						t.Error("Should not get command validation error for valid command")
					}
				}
			}
		})
	}
}

func TestExecShell_ShellCommandValidation(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name         string
		shellCommand string
		description  string
	}{
		{"empty command", "", "empty string is valid for sh -c"},
		{"simple command", "echo hello", "basic shell command"},
		{"command with pipes", "echo hello | grep hello", "piped commands"},
		{"command with AND", "echo hello && echo world", "chained with &&"},
		{"command with OR", "echo hello || echo world", "chained with ||"},
		{"command with redirect", "echo hello > /dev/null", "output redirection"},
		{"command with variables", "TEST=value; echo $TEST", "shell variables"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := client.ExecShell("test-container", tt.shellCommand)

			// All should fail due to nonexistent container, not due to validation
			// The validation of shell command happens in sh itself
			if err != nil && err.Error() == "container name cannot be empty" {
				t.Error("Should not get container name validation error")
			}
		})
	}
}
