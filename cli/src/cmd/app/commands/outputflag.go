package commands

import (
	"strings"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/spf13/cobra"
)

// outputFormatDefault is azd's own default value for --output. It means "no
// specific format requested", not a format the extension renders.
const outputFormatDefault = "default"

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
func registerOutputFormats(cmd *cobra.Command, defaultFormat string, formats ...string) *cobra.Command {
	allowed := make([]string, 0, len(formats)+1)
	allowed = append(allowed, outputFormatDefault)
	allowed = append(allowed, formats...)

	azdext.RegisterFlagOptions(cmd, azdext.FlagOptions{
		Name:          "output",
		AllowedValues: allowed,
		Default:       defaultFormat,
	})

	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[outputFormatsAnnotation] = strings.Join(formats, ",")
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
	if format == "" || format == outputFormatDefault || format == jsonOutputVal {
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
