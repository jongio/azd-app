package executor

import (
	"context"
	"runtime"
	"testing"
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

func TestPrepareHookCommand_Sh(t *testing.T) {
	ctx := context.Background()
	cmd := prepareHookCommand(ctx, "sh", "echo test", "/tmp")

	if cmd.Dir != "/tmp" {
		t.Errorf("Expected Dir='/tmp', got: %s", cmd.Dir)
	}

	// Check command structure
	if len(cmd.Args) < 3 {
		t.Fatalf("Expected at least 3 args, got: %v", cmd.Args)
	}
	if cmd.Args[1] != "-c" {
		t.Errorf("Expected second arg='-c', got: %s", cmd.Args[1])
	}
	if cmd.Args[2] != "echo test" {
		t.Errorf("Expected third arg='echo test', got: %s", cmd.Args[2])
	}
}

func TestPrepareHookCommand_Bash(t *testing.T) {
	ctx := context.Background()
	cmd := prepareHookCommand(ctx, "bash", "ls -la", "/home")

	if len(cmd.Args) < 3 {
		t.Fatalf("Expected at least 3 args, got: %v", cmd.Args)
	}
	if cmd.Args[1] != "-c" {
		t.Errorf("Expected second arg='-c', got: %s", cmd.Args[1])
	}
	if cmd.Args[2] != "ls -la" {
		t.Errorf("Expected third arg='ls -la', got: %s", cmd.Args[2])
	}
}

func TestPrepareHookCommand_PowerShell(t *testing.T) {
	ctx := context.Background()
	cmd := prepareHookCommand(ctx, "pwsh", "Get-ChildItem", "/tmp")

	if len(cmd.Args) < 3 {
		t.Fatalf("Expected at least 3 args, got: %v", cmd.Args)
	}
	if cmd.Args[1] != "-Command" {
		t.Errorf("Expected second arg='-Command', got: %s", cmd.Args[1])
	}
	if cmd.Args[2] != "Get-ChildItem" {
		t.Errorf("Expected third arg='Get-ChildItem', got: %s", cmd.Args[2])
	}
}

func TestPrepareHookCommand_Cmd(t *testing.T) {
	ctx := context.Background()
	cmd := prepareHookCommand(ctx, "cmd", "dir", "C:\\")

	if len(cmd.Args) < 3 {
		t.Fatalf("Expected at least 3 args, got: %v", cmd.Args)
	}
	if cmd.Args[1] != "/c" {
		t.Errorf("Expected second arg='/c', got: %s", cmd.Args[1])
	}
	if cmd.Args[2] != "dir" {
		t.Errorf("Expected third arg='dir', got: %s", cmd.Args[2])
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
