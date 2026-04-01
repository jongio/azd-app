package commands

import (
	"bytes"
	"testing"

	internalversion "github.com/jongio/azd-app/cli/src/internal/version"
)

func TestVersionConstants(t *testing.T) {
	if internalversion.Version == "" {
		t.Error("Version should not be empty")
	}
	if BuildTime == "" {
		t.Error("BuildTime should not be empty")
	}
	if Commit == "" {
		t.Error("Commit should not be empty")
	}
}

func TestVersionInfo(t *testing.T) {
	if VersionInfo == nil {
		t.Fatal("VersionInfo should not be nil")
	}
	if VersionInfo.ExtensionID != "jongio.azd.app" {
		t.Errorf("VersionInfo.ExtensionID = %q, want %q", VersionInfo.ExtensionID, "jongio.azd.app")
	}
	if VersionInfo.Name != "azd app" {
		t.Errorf("VersionInfo.Name = %q, want %q", VersionInfo.Name, "azd app")
	}
	if VersionInfo.Version == "" {
		t.Error("VersionInfo.Version should not be empty")
	}
}

func TestNewVersionCommand(t *testing.T) {
	outputFormat := "default"
	cmd := NewVersionCommand(&outputFormat)

	if cmd == nil {
		t.Fatal("NewVersionCommand() returned nil")
	}
	if cmd.Use != "version" {
		t.Errorf("cmd.Use = %q, want %q", cmd.Use, "version")
	}
	if cmd.Short == "" {
		t.Error("cmd.Short should not be empty")
	}

	flag := cmd.Flags().Lookup("quiet")
	if flag == nil {
		t.Error("--quiet flag should exist")
	}
}

func TestVersionCommand_DefaultOutput(t *testing.T) {
	outputFormat := "default"
	cmd := NewVersionCommand(&outputFormat)

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("cmd.Execute() error = %v", err)
	}
}
