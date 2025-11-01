package commands

import (
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
