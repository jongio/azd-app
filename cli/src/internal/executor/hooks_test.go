package executor

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestGetDefaultShell(t *testing.T) {
	shell := getDefaultShell()
	if shell == "" {
		t.Error("Expected default shell to be non-empty")
	}

	if runtime.GOOS == "windows" {
		// On Windows, expect pwsh, powershell, or cmd
		if shell != "pwsh" && shell != "powershell" && shell != "cmd" {
			t.Errorf("Expected Windows shell (pwsh/powershell/cmd), got: %s", shell)
		}
	} else {
		// On POSIX, expect bash or sh
		if shell != "bash" && shell != "sh" {
			t.Errorf("Expected POSIX shell (bash/sh), got: %s", shell)
		}
	}
}

func TestPrepareHookCommand(t *testing.T) {
	tests := []struct {
		name             string
		shell            string
		script           string
		wantArg1         string
		wantArg2Contains string
		isPowerShell     bool
	}{
		{
			name:             "sh shell",
			shell:            "sh",
			script:           "echo test",
			wantArg1:         "-c",
			wantArg2Contains: "echo test",
			isPowerShell:     false,
		},
		{
			name:             "bash shell",
			shell:            "bash",
			script:           "ls -la",
			wantArg1:         "-c",
			wantArg2Contains: "ls -la",
			isPowerShell:     false,
		},
		{
			name:             "pwsh shell",
			shell:            "pwsh",
			script:           "Get-ChildItem",
			wantArg1:         "-Command",
			wantArg2Contains: "Get-ChildItem",
			isPowerShell:     true,
		},
		{
			name:             "powershell shell",
			shell:            "powershell",
			script:           "Write-Host test",
			wantArg1:         "-Command",
			wantArg2Contains: "Write-Host test",
			isPowerShell:     true,
		},
		{
			name:             "cmd shell",
			shell:            "cmd",
			script:           "dir",
			wantArg1:         "/c",
			wantArg2Contains: "dir",
			isPowerShell:     false,
		},
		{
			name:             "zsh shell (posix)",
			shell:            "zsh",
			script:           "echo zsh",
			wantArg1:         "-c",
			wantArg2Contains: "echo zsh",
			isPowerShell:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cmd := prepareHookCommand(ctx, tt.shell, tt.script, "/tmp", nil)

			if cmd.Dir != "/tmp" {
				t.Errorf("Expected Dir='/tmp', got: %s", cmd.Dir)
			}

			if len(cmd.Args) < 3 {
				t.Fatalf("Expected at least 3 args, got: %v", cmd.Args)
			}

			if cmd.Args[1] != tt.wantArg1 {
				t.Errorf("Expected arg[1]=%q, got: %q", tt.wantArg1, cmd.Args[1])
			}

			// For PowerShell, check that the script contains the expected command (may have UTF-8 wrapper)
			if tt.isPowerShell {
				if !strings.Contains(cmd.Args[2], tt.wantArg2Contains) {
					t.Errorf("Expected arg[2] to contain %q, got: %q", tt.wantArg2Contains, cmd.Args[2])
				}
			} else {
				if cmd.Args[2] != tt.wantArg2Contains {
					t.Errorf("Expected arg[2]=%q, got: %q", tt.wantArg2Contains, cmd.Args[2])
				}
			}

			if cmd.Env == nil {
				t.Error("Expected cmd.Env to be set")
			}
		})
	}
}

func TestResolveHookConfig_Nil(t *testing.T) {
	config := ResolveHookConfig(nil)
	if config != nil {
		t.Error("Expected nil config for nil hook")
	}
}

func TestResolveHookConfig_BaseOnly(t *testing.T) {
	hook := &Hook{
		Run:             "echo test",
		Shell:           "bash",
		ContinueOnError: true,
		Interactive:     false,
	}

	config := ResolveHookConfig(hook)
	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	if config.Run != "echo test" {
		t.Errorf("Expected Run='echo test', got: %s", config.Run)
	}
	if config.Shell != "bash" {
		t.Errorf("Expected Shell='bash', got: %s", config.Shell)
	}
	if !config.ContinueOnError {
		t.Error("Expected ContinueOnError=true")
	}
	if config.Interactive {
		t.Error("Expected Interactive=false")
	}
}

func TestResolveHookConfig_EmptyShell(t *testing.T) {
	hook := &Hook{
		Run:   "echo test",
		Shell: "", // Empty shell should work
	}

	config := ResolveHookConfig(hook)
	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	if config.Run != "echo test" {
		t.Errorf("Expected Run='echo test', got: %s", config.Run)
	}
	if config.Shell != "" {
		t.Errorf("Expected Shell='', got: %s", config.Shell)
	}
}

func TestResolveHookConfig_WithPlatformOverride(t *testing.T) {
	continueOnError := false
	interactive := true

	hook := &Hook{
		Run:             "echo base",
		Shell:           "sh",
		ContinueOnError: true,
		Interactive:     false,
	}

	if runtime.GOOS == "windows" {
		hook.Windows = &PlatformHook{
			Run:             "echo windows",
			Shell:           "pwsh",
			ContinueOnError: &continueOnError,
			Interactive:     &interactive,
		}
	} else {
		hook.Posix = &PlatformHook{
			Run:             "echo posix",
			Shell:           "bash",
			ContinueOnError: &continueOnError,
			Interactive:     &interactive,
		}
	}

	config := ResolveHookConfig(hook)
	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	// Check platform-specific overrides applied
	if runtime.GOOS == "windows" {
		if config.Run != "echo windows" {
			t.Errorf("Expected Windows override Run='echo windows', got: %s", config.Run)
		}
		if config.Shell != "pwsh" {
			t.Errorf("Expected Windows override Shell='pwsh', got: %s", config.Shell)
		}
	} else {
		if config.Run != "echo posix" {
			t.Errorf("Expected POSIX override Run='echo posix', got: %s", config.Run)
		}
		if config.Shell != "bash" {
			t.Errorf("Expected POSIX override Shell='bash', got: %s", config.Shell)
		}
	}

	if config.ContinueOnError {
		t.Error("Expected platform override ContinueOnError=false")
	}
	if !config.Interactive {
		t.Error("Expected platform override Interactive=true")
	}
}

func TestResolveHookConfig_PartialPlatformOverride(t *testing.T) {
	hook := &Hook{
		Run:             "echo base",
		Shell:           "sh",
		ContinueOnError: true,
		Interactive:     false,
	}

	if runtime.GOOS == "windows" {
		hook.Windows = &PlatformHook{
			Run: "echo windows", // Only override Run
		}
	} else {
		hook.Posix = &PlatformHook{
			Run: "echo posix", // Only override Run
		}
	}

	config := ResolveHookConfig(hook)
	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	// Run should be overridden
	if runtime.GOOS == "windows" {
		if config.Run != "echo windows" {
			t.Errorf("Expected Run='echo windows', got: %s", config.Run)
		}
	} else {
		if config.Run != "echo posix" {
			t.Errorf("Expected Run='echo posix', got: %s", config.Run)
		}
	}

	// Other fields should keep base values
	if config.Shell != "sh" {
		t.Errorf("Expected Shell='sh' from base, got: %s", config.Shell)
	}
	if !config.ContinueOnError {
		t.Error("Expected ContinueOnError=true from base")
	}
	if config.Interactive {
		t.Error("Expected Interactive=false from base")
	}
}

func TestResolveHookConfig_EmptyPlatformOverride(t *testing.T) {
	hook := &Hook{
		Run:   "echo base",
		Shell: "sh",
	}

	// Set platform override but with empty values
	if runtime.GOOS == "windows" {
		hook.Windows = &PlatformHook{}
	} else {
		hook.Posix = &PlatformHook{}
	}

	config := ResolveHookConfig(hook)
	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	// Should use base values when override is empty
	if config.Run != "echo base" {
		t.Errorf("Expected Run='echo base', got: %s", config.Run)
	}
	if config.Shell != "sh" {
		t.Errorf("Expected Shell='sh', got: %s", config.Shell)
	}
}

func TestExecuteHook_NoHook(t *testing.T) {
	ctx := context.Background()
	config := HookConfig{
		Run: "", // Empty run command
	}

	err := ExecuteHook(ctx, "test", config, "/tmp")
	if err != nil {
		t.Errorf("Expected no error for empty hook, got: %v", err)
	}
}

func TestExecuteHook_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hook execution test in short mode")
	}

	ctx := context.Background()
	config := HookConfig{
		Run:   "echo 'test successful'",
		Shell: getDefaultShell(),
	}

	err := ExecuteHook(ctx, "test", config, ".")
	if err != nil {
		t.Errorf("Expected successful execution, got: %v", err)
	}
}

func TestExecuteHook_FailureWithContinue(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hook execution test in short mode")
	}

	ctx := context.Background()

	// Use a command that will fail on both Windows and POSIX
	failCmd := "exit 1"
	if runtime.GOOS == "windows" {
		failCmd = "exit /b 1"
	}

	config := HookConfig{
		Run:             failCmd,
		Shell:           getDefaultShell(),
		ContinueOnError: true,
	}

	err := ExecuteHook(ctx, "test", config, ".")
	// Should not error because continueOnError is true
	if err != nil {
		t.Errorf("Expected no error with continueOnError=true, got: %v", err)
	}
}

func TestExecuteHook_FailureWithoutContinue(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hook execution test in short mode")
	}

	ctx := context.Background()

	// Use a command that will fail on both Windows and POSIX
	failCmd := "exit 1"
	if runtime.GOOS == "windows" {
		failCmd = "exit /b 1"
	}

	config := HookConfig{
		Run:             failCmd,
		Shell:           getDefaultShell(),
		ContinueOnError: false,
	}

	err := ExecuteHook(ctx, "test", config, ".")
	// Should error because continueOnError is false
	if err == nil {
		t.Error("Expected error with continueOnError=false")
	}
}

func TestExecuteHook_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hook execution test in short mode")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	config := HookConfig{
		Run:   "sleep 10",
		Shell: getDefaultShell(),
	}

	err := ExecuteHook(ctx, "test", config, ".")
	if err == nil {
		t.Error("Expected error from cancelled context")
	}
}

func TestExecuteHook_InvalidShell(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hook execution test in short mode")
	}

	ctx := context.Background()
	config := HookConfig{
		Run:   "echo test",
		Shell: "nonexistent-shell-xyz",
	}

	err := ExecuteHook(ctx, "test", config, ".")
	if err == nil {
		t.Error("Expected error for nonexistent shell")
	}
}

func TestExecuteHook_InvalidWorkingDir(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hook execution test in short mode")
	}

	ctx := context.Background()
	config := HookConfig{
		Run:   "echo test",
		Shell: getDefaultShell(),
	}

	err := ExecuteHook(ctx, "test", config, "/nonexistent/directory/path")
	if err == nil {
		t.Error("Expected error for invalid working directory")
	}
}

func TestExecuteHook_LongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hook execution test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var sleepCmd string
	if runtime.GOOS == "windows" {
		sleepCmd = "timeout /t 10 /nobreak"
	} else {
		sleepCmd = "sleep 10"
	}

	config := HookConfig{
		Run:   sleepCmd,
		Shell: getDefaultShell(),
	}

	err := ExecuteHook(ctx, "test", config, ".")
	if err == nil {
		t.Error("Expected timeout error for long-running command")
	}
}

func TestResolveHookConfig_BothPlatformOverrides(t *testing.T) {
	continueOnError := true
	hook := &Hook{
		Run:   "echo base",
		Shell: "sh",
		Windows: &PlatformHook{
			Run:             "echo windows",
			Shell:           "pwsh",
			ContinueOnError: &continueOnError,
		},
		Posix: &PlatformHook{
			Run:             "echo posix",
			Shell:           "bash",
			ContinueOnError: &continueOnError,
		},
	}

	config := ResolveHookConfig(hook)
	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	// Should use appropriate platform override
	if runtime.GOOS == "windows" {
		if config.Run != "echo windows" {
			t.Errorf("Expected Windows override, got: %s", config.Run)
		}
		if config.Shell != "pwsh" {
			t.Errorf("Expected Windows shell pwsh, got: %s", config.Shell)
		}
	} else {
		if config.Run != "echo posix" {
			t.Errorf("Expected POSIX override, got: %s", config.Run)
		}
		if config.Shell != "bash" {
			t.Errorf("Expected POSIX shell bash, got: %s", config.Shell)
		}
	}

	if !config.ContinueOnError {
		t.Error("Expected ContinueOnError=true from platform override")
	}
}

func TestPrepareHookCommand_EnvironmentInheritance(t *testing.T) {
	// Set a test environment variable
	testKey := "TEST_HOOK_ENV_VAR"
	testValue := "test_value_123"
	os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	ctx := context.Background()
	cmd := prepareHookCommand(ctx, "sh", "echo test", "/tmp", nil)

	// Check that environment is inherited
	found := false
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, testKey+"=") {
			found = true
			if !strings.Contains(env, testValue) {
				t.Errorf("Expected env var to contain %s, got: %s", testValue, env)
			}
			break
		}
	}

	if !found {
		t.Errorf("Expected environment variable %s to be inherited", testKey)
	}
}

func TestExecuteHook_DefaultShell(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hook execution test in short mode")
	}

	ctx := context.Background()
	config := HookConfig{
		Run:   "echo test",
		Shell: "", // Empty shell should use default
	}

	err := ExecuteHook(ctx, "test", config, ".")
	if err != nil {
		t.Errorf("Expected successful execution with default shell, got: %v", err)
	}
}

func TestPrepareHookCommand_WithAdditionalEnv(t *testing.T) {
	testDir := "/test/path"
	additionalEnv := []string{
		"CUSTOM_VAR=custom_value",
		"ANOTHER_VAR=another_value",
	}

	ctx := context.Background()
	cmd := prepareHookCommand(ctx, "sh", "echo test", testDir, additionalEnv)

	// Verify additional environment variables are appended
	found := make(map[string]bool)
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "CUSTOM_VAR=") {
			found["CUSTOM_VAR"] = true
			if !strings.Contains(env, "custom_value") {
				t.Errorf("Expected CUSTOM_VAR=custom_value, got: %s", env)
			}
		}
		if strings.HasPrefix(env, "ANOTHER_VAR=") {
			found["ANOTHER_VAR"] = true
			if !strings.Contains(env, "another_value") {
				t.Errorf("Expected ANOTHER_VAR=another_value, got: %s", env)
			}
		}
	}

	if !found["CUSTOM_VAR"] {
		t.Error("Expected CUSTOM_VAR to be in environment")
	}
	if !found["ANOTHER_VAR"] {
		t.Error("Expected ANOTHER_VAR to be in environment")
	}
}

func TestExecuteHook_WithCustomEnvVars(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hook execution test in short mode")
	}

	ctx := context.Background()
	config := HookConfig{
		Run:   "echo test",
		Shell: getDefaultShell(),
		Env: []string{
			EnvProjectDir + "=/custom/path",
			EnvProjectName + "=test-project",
			EnvServiceCount + "=5",
		},
	}

	err := ExecuteHook(ctx, "test", config, ".")
	if err != nil {
		t.Errorf("Expected successful execution with env vars, got: %v", err)
	}
}

func TestNewHook(t *testing.T) {
	continueOnError := true
	interactive := false

	windows := NewPlatformHook("echo windows", "pwsh", &continueOnError, &interactive)
	posix := NewPlatformHook("echo posix", "bash", nil, nil)

	hook := NewHook("echo test", "sh", false, true, windows, posix)

	if hook == nil {
		t.Fatal("Expected non-nil hook")
	}

	if hook.Run != "echo test" {
		t.Errorf("Expected Run='echo test', got: %s", hook.Run)
	}
	if hook.Shell != "sh" {
		t.Errorf("Expected Shell='sh', got: %s", hook.Shell)
	}
	if hook.ContinueOnError {
		t.Error("Expected ContinueOnError=false")
	}
	if !hook.Interactive {
		t.Error("Expected Interactive=true")
	}
	if hook.Windows == nil {
		t.Error("Expected Windows platform hook to be non-nil")
	}
	if hook.Posix == nil {
		t.Error("Expected POSIX platform hook to be non-nil")
	}
}

func TestNewPlatformHook(t *testing.T) {
	continueOnError := true
	interactive := false

	hook := NewPlatformHook("echo test", "bash", &continueOnError, &interactive)

	if hook == nil {
		t.Fatal("Expected non-nil platform hook")
	}

	if hook.Run != "echo test" {
		t.Errorf("Expected Run='echo test', got: %s", hook.Run)
	}
	if hook.Shell != "bash" {
		t.Errorf("Expected Shell='bash', got: %s", hook.Shell)
	}
	if hook.ContinueOnError == nil {
		t.Fatal("Expected ContinueOnError to be non-nil")
	}
	if !*hook.ContinueOnError {
		t.Error("Expected ContinueOnError=true")
	}
	if hook.Interactive == nil {
		t.Fatal("Expected Interactive to be non-nil")
	}
	if *hook.Interactive {
		t.Error("Expected Interactive=false")
	}
}

func TestNewPlatformHook_WithNilValues(t *testing.T) {
	hook := NewPlatformHook("echo test", "sh", nil, nil)

	if hook == nil {
		t.Fatal("Expected non-nil platform hook")
	}

	if hook.ContinueOnError != nil {
		t.Error("Expected ContinueOnError to be nil")
	}
	if hook.Interactive != nil {
		t.Error("Expected Interactive to be nil")
	}
}
