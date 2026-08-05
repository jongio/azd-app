package commands

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// attachInheritedOutput mirrors how the azdext SDK wires azd's global output
// flag: a persistent string flag on the root command that every subcommand
// inherits. Commands under test are built standalone, so without a parent they
// see no --output at all and fall back to their own default.
func attachInheritedOutput(t *testing.T, cmd *cobra.Command, value string) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "app"}
	root.PersistentFlags().StringP("output", "o", outputFormatDefault, "The output format")
	root.AddCommand(cmd)
	if value != "" {
		if err := root.PersistentFlags().Set("output", value); err != nil {
			t.Fatalf("set output: %v", err)
		}
	}
	return root
}

// TestCliOutFormatFor covers the translation from azd's --output value to the
// two-state format cliout accepts. Getting this wrong made every rich format
// fail at startup, before the command ever ran.
func TestCliOutFormatFor(t *testing.T) {
	tests := []struct {
		name   string
		cmd    *cobra.Command
		format string
		want   string
	}{
		{"empty passes through", NewGraphCommand(), "", ""},
		{"azd default passes through", NewGraphCommand(), outputFormatDefault, outputFormatDefault},
		{"json passes through", NewGraphCommand(), "json", "json"},
		{"declared graph format maps to default", NewGraphCommand(), "mermaid", outputFormatDefault},
		{"declared health format maps to default", NewHealthCommand(), "table", outputFormatDefault},
		{"declared logs format maps to default", NewLogsCommand(), "text", outputFormatDefault},
		{"undeclared value passes through", NewGraphCommand(), "xml", "xml"},
		{"format declared elsewhere still passes through", NewHealthCommand(), "mermaid", "mermaid"},
		{"command without declarations passes through", &cobra.Command{Use: "plain"}, "table", "table"},
		{"nil command passes through", nil, "table", "table"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CliOutFormatFor(tt.cmd, tt.format); got != tt.want {
				t.Fatalf("CliOutFormatFor(%q) = %q, want %q", tt.format, got, tt.want)
			}
		})
	}
}

// TestCliOutFormatForInheritsFromParent covers a subcommand of a command that
// declared formats, which is how the annotation lookup walks upward.
func TestCliOutFormatForInheritsFromParent(t *testing.T) {
	parent := registerOutputFormats(&cobra.Command{Use: "parent"}, "text", "text", "table")
	child := &cobra.Command{Use: "child"}
	parent.AddCommand(child)

	if got := CliOutFormatFor(child, "table"); got != outputFormatDefault {
		t.Fatalf("CliOutFormatFor = %q, want %q", got, outputFormatDefault)
	}
}

// TestRegisteredOutputFormatsAreCliOutSafe is the regression guard for the
// startup failure: every format a command advertises must survive translation
// into something cliout accepts, otherwise the command is unreachable.
func TestRegisteredOutputFormatsAreCliOutSafe(t *testing.T) {
	commands := map[string]*cobra.Command{
		"graph":  NewGraphCommand(),
		"health": NewHealthCommand(),
		"logs":   NewLogsCommand(),
	}
	for name, cmd := range commands {
		t.Run(name, func(t *testing.T) {
			declared := cmd.Annotations[outputFormatsAnnotation]
			if declared == "" {
				t.Fatalf("%s declares no output formats", name)
			}
			for _, format := range append(strings.Split(declared, ","), outputFormatDefault) {
				translated := CliOutFormatFor(cmd, format)
				if translated != outputFormatDefault && translated != "json" {
					t.Fatalf("%s advertises %q, which cliout rejects (translated to %q)", name, format, translated)
				}
			}
		})
	}
}

func TestInheritedOutputFormat(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback string
		want     string
	}{
		{"unset falls back", "", "text", "text"},
		{"azd default falls back", outputFormatDefault, "text", "text"},
		{"explicit value wins", "json", "text", "json"},
		{"fallback is per command", "", "table", "table"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "child"}
			attachInheritedOutput(t, cmd, tt.value)
			if got := inheritedOutputFormat(cmd, tt.fallback); got != tt.want {
				t.Fatalf("inheritedOutputFormat = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInheritedOutputFormatNilCommand(t *testing.T) {
	if got := inheritedOutputFormat(nil, "text"); got != "text" {
		t.Fatalf("inheritedOutputFormat(nil) = %q, want text", got)
	}
}

func TestInheritedOutputFormatWithoutParent(t *testing.T) {
	cmd := &cobra.Command{Use: "child"}
	if got := inheritedOutputFormat(cmd, "text"); got != "text" {
		t.Fatalf("inheritedOutputFormat = %q, want text", got)
	}
}

// TestInheritedOutputFormatLocalFlag covers the branch where a command owns the
// output flag directly rather than inheriting it, which is how azd's own root
// command sees it.
func TestInheritedOutputFormatLocalFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "child"}
	cmd.Flags().StringP("output", "o", "", "The output format")
	if err := cmd.Flags().Set("output", "json"); err != nil {
		t.Fatalf("set output: %v", err)
	}
	if got := inheritedOutputFormat(cmd, "text"); got != "json" {
		t.Fatalf("inheritedOutputFormat = %q, want json", got)
	}
}

// TestOutputCommandsDoNotShadowGlobalFlag is the regression guard for the bug
// this helper exists to fix. A local --output or --format on these commands
// takes precedence over azd's inherited flag, so azd's value would be dropped.
func TestOutputCommandsDoNotShadowGlobalFlag(t *testing.T) {
	commands := map[string]*cobra.Command{
		"graph":  NewGraphCommand(),
		"health": NewHealthCommand(),
		"logs":   NewLogsCommand(),
	}
	for name, cmd := range commands {
		t.Run(name, func(t *testing.T) {
			for _, flag := range []string{"output", "format"} {
				if cmd.Flags().Lookup(flag) != nil {
					t.Fatalf("%s defines a local --%s, which shadows azd's global output flag", name, flag)
				}
			}
		})
	}
}
