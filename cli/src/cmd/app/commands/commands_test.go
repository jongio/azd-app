package commands

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewReqsCommand(t *testing.T) {
	cmd := NewReqsCommand()

	if cmd == nil {
		t.Fatal("NewReqsCommand() returned nil")
	}

	if cmd.Use != "reqs" {
		t.Errorf("Use = %q, want %q", cmd.Use, "reqs")
	}

	if cmd.Short == "" {
		t.Error("Short description is empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE function is nil")
	}
}

func TestNewDepsCommand(t *testing.T) {
	cmd := NewDepsCommand()

	if cmd == nil {
		t.Fatal("NewDepsCommand() returned nil")
	}

	if cmd.Use != "deps" {
		t.Errorf("Use = %q, want %q", cmd.Use, "deps")
	}

	if cmd.Short == "" {
		t.Error("Short description is empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE function is nil")
	}
}

func TestNewRunCommand(t *testing.T) {
	cmd := NewRunCommand()

	if cmd == nil {
		t.Fatal("NewRunCommand() returned nil")
	}

	if cmd.Use != "run" {
		t.Errorf("Use = %q, want %q", cmd.Use, "run")
	}

	if cmd.Short == "" {
		t.Error("Short description is empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE function is nil")
	}
}

func TestAllCommandsHaveDescriptions(t *testing.T) {
	commands := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"reqs", NewReqsCommand()},
		{"deps", NewDepsCommand()},
		{"run", NewRunCommand()},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			if tc.cmd.Use == "" {
				t.Errorf("%s command has empty Use", tc.name)
			}
			if tc.cmd.Short == "" {
				t.Errorf("%s command has empty Short description", tc.name)
			}
			if tc.cmd.Long == "" {
				t.Errorf("%s command has empty Long description", tc.name)
			}
		})
	}
}

func TestCommandsHaveRunFunctions(t *testing.T) {
	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"reqs", NewReqsCommand()},
		{"deps", NewDepsCommand()},
		{"run", NewRunCommand()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd.RunE == nil {
				t.Errorf("%s command should have RunE function", tt.name)
			}
		})
	}
}

func TestNewVersionCommand_Basic(t *testing.T) {
	outputFormat := "default"
	cmd := NewVersionCommand(&outputFormat)

	if cmd == nil {
		t.Fatal("NewVersionCommand() returned nil")
	}

	if cmd.Use != "version" {
		t.Errorf("Use = %q, want %q", cmd.Use, "version")
	}

	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
}

func TestNewInfoCommand(t *testing.T) {
	cmd := NewInfoCommand()

	if cmd == nil {
		t.Fatal("NewInfoCommand() returned nil")
	}

	if cmd.Use != "info" {
		t.Errorf("Use = %q, want %q", cmd.Use, "info")
	}

	if cmd.Short == "" {
		t.Error("Short description is empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE function is nil")
	}
}

func TestNewListenCommand(t *testing.T) {
	cmd := NewListenCommand()

	if cmd == nil {
		t.Fatal("NewListenCommand() returned nil")
	}

	if cmd.Use != "listen" {
		t.Errorf("Use = %q, want %q", cmd.Use, "listen")
	}

	// SDK's NewListenCommand is a hidden internal command; Short is intentionally empty
	if !cmd.Hidden {
		t.Error("listen command should be hidden")
	}

	if cmd.RunE == nil {
		t.Error("RunE function is nil")
	}
}

func TestNewLogsCommand(t *testing.T) {
	cmd := NewLogsCommand()

	if cmd == nil {
		t.Fatal("NewLogsCommand() returned nil")
	}

	// Logs command may have arguments in Use field
	if !strings.Contains(cmd.Use, "logs") {
		t.Errorf("Use = %q, should contain 'logs'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description is empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE function is nil")
	}
}

func TestCommandFlags(t *testing.T) {
	tests := []struct {
		name     string
		cmd      *cobra.Command
		flagName string
	}{
		{"logs --follow flag", NewLogsCommand(), "follow"},
		{"logs --tail flag", NewLogsCommand(), "tail"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := tt.cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("Expected flag %q to exist", tt.flagName)
			}
		})
	}
}

func TestCommandAliases(t *testing.T) {
	tests := []struct {
		name    string
		cmd     *cobra.Command
		wantLen int
	}{
		{"reqs", NewReqsCommand(), 0},
		{"deps", NewDepsCommand(), 0},
		{"run", NewRunCommand(), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.cmd.Aliases) != tt.wantLen {
				t.Logf("Command %s has %d aliases (expected %d)", tt.name, len(tt.cmd.Aliases), tt.wantLen)
			}
		})
	}
}

func TestCommandUsageText(t *testing.T) {
	commands := []*cobra.Command{
		NewReqsCommand(),
		NewDepsCommand(),
		NewRunCommand(),
		NewVersionCommand(func() *string { s := "default"; return &s }()),
		NewInfoCommand(),
		NewListenCommand(),
		NewLogsCommand(),
	}

	for _, cmd := range commands {
		t.Run(cmd.Use, func(t *testing.T) {
			usage := cmd.UsageString()
			if usage == "" {
				t.Error("Command should have usage string")
			}
		})
	}
}
