package commands

import (
	"strings"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/spf13/cobra"
)

// outputFormatDefault is azd's own default value for --output. It means "no
// specific format requested", not a format the extension renders.
const outputFormatDefault = "default"

// The formats this extension renders. They live here, not beside the command
// that happens to use each one, because the same value was previously spelled
// three ways: a graph-local constant, a logs-local constant, and a bare
// literal. A switch that cases on a constant and an error message that lists a
// literal drift apart silently, and the compiler cannot help.
const (
	outputFormatText     = "text"
	outputFormatJSON     = "json"
	outputFormatTable    = "table"
	outputFormatMarkdown = "markdown"
	outputFormatMermaid  = "mermaid"
	outputFormatDOT      = "dot"
	outputFormatD2       = "d2"
	outputFormatPlantUML = "plantuml"
)

// outputFormatsAnnotation records the rich formats a command renders beyond
// azd's own default and json.
//
// The azdext SDK stores the same list in its own annotation, but under an
// unexported key, so reading it back would couple this extension to an
// implementation detail of a package we don't control. Recording it a second
// time under our own key costs one map entry and keeps the coupling here.
const outputFormatsAnnotation = "azd-app.output-formats"

// registerOutputFormats declares the values a command accepts for azd's global
// --output flag.
//
// azd reserves --output/-o and the SDK registers it on the root command, so a
// subcommand must never define its own. Doing so shadowed the real flag: cobra
// resolves a local flag ahead of an inherited persistent one, so azd's value
// never reached the command and the extension silently ignored it.
//
// One declaration drives the per-command help text, shell completion, the
// extension metadata azd reads, and parse-time rejection of anything outside
// the list, so by the time RunE executes the value is already known good.
//
// "default" is always accepted. The SDK seeds the flag with it and a user can
// pass it explicitly, and in both cases it means the command's own default.
//
// defaultFormat is folded into the accepted set, so a caller passes it once.
// Requiring it in both positions invited the failure where a command declared
// a default that its own AllowedValues did not contain, which azd would then
// reject at parse time.
func registerOutputFormats(cmd *cobra.Command, defaultFormat string, formats ...string) *cobra.Command {
	declared := make([]string, 0, len(formats)+1)
	seen := map[string]bool{}
	for _, format := range append([]string{defaultFormat}, formats...) {
		if format == "" || seen[format] {
			continue
		}
		seen[format] = true
		declared = append(declared, format)
	}

	azdext.RegisterFlagOptions(cmd, azdext.FlagOptions{
		Name:          "output",
		AllowedValues: append([]string{outputFormatDefault}, declared...),
		Default:       defaultFormat,
	})

	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[outputFormatsAnnotation] = strings.Join(declared, ",")
	return cmd
}

// inheritedOutputFormat reads azd's global --output flag for a subcommand.
//
// Pair it with registerOutputFormats, which validates the value before RunE
// runs. fallback covers the case where the command is executed detached from
// the SDK root, which is how much of the test suite drives these commands.
func inheritedOutputFormat(cmd *cobra.Command, fallback string) string {
	if cmd == nil {
		return fallback
	}
	value := ""
	if flag := cmd.InheritedFlags().Lookup("output"); flag != nil {
		value = flag.Value.String()
	}
	if value == "" {
		if flag := cmd.Flags().Lookup("output"); flag != nil {
			value = flag.Value.String()
		}
	}
	if value == "" || value == outputFormatDefault {
		return fallback
	}
	return value
}

// CliOutFormatFor translates azd's --output value into one that cliout
// understands.
//
// cliout is a two-state switch: human-readable or machine-readable JSON. Most
// commands only ever see "default" or "json", so forwarding the raw value is
// correct for them and keeps cliout rejecting typos. Commands that render
// richer formats, such as graph's mermaid or health's table, are still human
// output as far as cliout is concerned, so their formats map to "default".
//
// Only formats the invoked command actually declared are translated. Anything
// else passes through untouched so cliout still reports it as invalid.
func CliOutFormatFor(cmd *cobra.Command, format string) string {
	if format == "" || format == outputFormatDefault || format == outputFormatJSON {
		return format
	}
	for c := cmd; c != nil; c = c.Parent() {
		declared, ok := c.Annotations[outputFormatsAnnotation]
		if !ok {
			continue
		}
		for _, candidate := range strings.Split(declared, ",") {
			if candidate == format {
				return outputFormatDefault
			}
		}
		break
	}
	return format
}
