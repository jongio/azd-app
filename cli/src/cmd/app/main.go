package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/jongio/azd-app/cli/src/cmd/app/commands"
	"github.com/jongio/azd-app/cli/src/internal/logging"
	"github.com/jongio/azd-app/cli/src/internal/skills"
	internalversion "github.com/jongio/azd-app/cli/src/internal/version"
	"github.com/jongio/azd-core/cliout"
	"github.com/jongio/azd-core/env"
	"github.com/spf13/cobra"
)

var structuredLogs bool

// isDetachedChild reports whether this process is the background run spawned by
// `azd app run --detach`. It is a variable rather than a direct call so tests
// can exercise both sides of the environment-loading guard below. The real
// detection caches under sync.Once and consumes its marker, so it cannot be
// toggled more than once within a single process.
var isDetachedChild = commands.IsDetachedChild

func main() {
	// azdext.Run owns the lifecycle: FORCE_COLOR handling, cobra SilenceErrors,
	// context creation with tracing propagation, gRPC access token injection,
	// reserved-flag validation, structured error reporting back to the azd host,
	// error and suggestion display, and the exit status.
	azdext.Run(newRootCmd())
}

// newRootCmd builds the full command tree. It is a function rather than inline
// setup in main so tests can construct the same tree the binary runs.
func newRootCmd() *cobra.Command {
	// Use the standard extension root command which provides:
	// - Standard azd flags (--debug, --no-prompt, --cwd, -e, --output)
	// - AZD_* environment variable fallback for all flags
	// - OpenTelemetry trace context propagation from TRACEPARENT/TRACESTATE
	// - gRPC access token injection via WithAccessToken()
	rootCmd, extCtx := azdext.NewExtensionRootCommand(azdext.ExtensionCommandOptions{
		Name:    "app",
		Version: internalversion.Version,
		Short:   "App - Automate your development environment setup",
		Long:    `App is an Azure Developer CLI extension that automatically detects and sets up your development environment across multiple languages and frameworks.`,
	})

	// Add app-specific flags not covered by the standard set
	rootCmd.PersistentFlags().BoolVar(&structuredLogs, "structured-logs", false, "Enable structured JSON logging to stderr")

	// Chain app-specific setup after the standard PersistentPreRunE
	origPreRun := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Run standard extension setup first (env var fallback, cwd, tracing, access token)
		if origPreRun != nil {
			if err := origPreRun(cmd, args); err != nil {
				return err
			}
		}

		// Handle environment selection.
		//
		// The detached background run already inherits the values its parent
		// resolved, so reloading them is redundant. It is also actively harmful:
		// LoadAzdEnvironment shells out to `azd env get-values`, and the azd host
		// that served the parent exits as soon as the parent returns. That left
		// the detached child blocked and completely silent for seconds, which is
		// why its run.log was empty before it died.
		//
		// The azdext SDK does not remove the need for this guard. Neither
		// NewExtensionRootCommand nor azdext.LoadAzdEnvironment changes the fact
		// that reading environment values requires a live azd host, and the SDK
		// loader shells out to the same `azd env get-values` (without even an -e
		// flag). See TestDetachedChildSkipsEnvironmentLoad.
		if extCtx.Environment != "" && !isDetachedChild() {
			if err := env.LoadAzdEnvironment(cmd.Context(), extCtx.Environment); err != nil {
				return fmt.Errorf("failed to load environment '%s': %w", extCtx.Environment, err)
			}
		}

		// Configure logging
		if extCtx.Debug {
			slog.SetLogLoggerLevel(slog.LevelDebug)
		}
		logging.SetupLogger(extCtx.Debug, structuredLogs)

		if extCtx.Debug {
			logging.Debug(
				"Starting azd app extension",
				"version", internalversion.Version,
				"command", cmd.Name(),
				"args", args,
				"cwd", extCtx.Cwd,
			)
			if !cliout.IsJSON() {
				fmt.Fprintf(os.Stderr, "%s[DEBUG]%s Build: %s (built on %s, commit: %.8s)\n",
					cliout.Dim, cliout.Reset, internalversion.Version, commands.BuildTime, commands.Commit)
			}
		}

		// Install Copilot skill
		if err := skills.InstallSkill(); err != nil {
			if extCtx.Debug {
				slog.Debug("Failed to install copilot skill", "error", err)
			}
		}

		return cliout.SetFormat(commands.CliOutFormatFor(cmd, extCtx.OutputFormat))
	}

	// Register all commands
	rootCmd.AddCommand(
		commands.NewInitCommand(),
		commands.NewReqsCommand(),
		commands.NewRunCommand(),
		commands.NewDepsCommand(),
		commands.NewOutdatedCommand(),
		commands.NewCleanCommand(),
		commands.NewTestCommand(),
		commands.NewLogsCommand(),
		commands.NewInfoCommand(),
		commands.NewEnvCommand(),
		commands.NewHealthCommand(),
		commands.NewVersionCommand(&extCtx.OutputFormat),
		commands.NewNotificationsCommand(),
		commands.NewListenCommand(), // Required for azd extension framework
		commands.NewMCPCommand(),    // Model Context Protocol server
		commands.NewStartCommand(),
		commands.NewStopCommand(),
		commands.NewStatusCommand(),
		commands.NewRestartCommand(),
		commands.NewProxyCommand(),
		commands.NewCertCommand(),
		commands.NewAddCommand(),
		commands.NewConfigCommand(),
		commands.NewRemoveCommand(),
		commands.NewSupportBundleCommand(),
		commands.NewGraphCommand(),
		commands.NewValidateCommand(),
		commands.NewDoctorCommand(),
		commands.NewHooksCommand(),
		commands.NewPortsCommand(),
		commands.NewOpenCommand(),
		commands.NewMetadataCommand(func() *cobra.Command { return rootCmd }),
	)

	return rootCmd
}
