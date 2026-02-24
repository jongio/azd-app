package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/jongio/azd-app/cli/src/cmd/app/commands"
	"github.com/jongio/azd-app/cli/src/internal/logging"
	"github.com/jongio/azd-app/cli/src/internal/skills"
	"github.com/jongio/azd-core/cliout"
	"github.com/jongio/azd-core/env"
	"github.com/spf13/cobra"
)

var (
	structuredLogs bool
)

func main() {
	// Use the standard extension root command which provides:
	// - Standard azd flags (--debug, --no-prompt, --cwd, -e, --output)
	// - AZD_* environment variable fallback for all flags
	// - OpenTelemetry trace context propagation from TRACEPARENT/TRACESTATE
	// - gRPC access token injection via WithAccessToken()
	rootCmd, extCtx := azdext.NewExtensionRootCommand(azdext.ExtensionCommandOptions{
		Name:    "app",
		Version: commands.Version,
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

		// Handle environment selection
		if extCtx.Environment != "" {
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
			logging.Debug("Starting azd app extension",
				"version", commands.Version,
				"command", cmd.Name(),
				"args", args,
				"cwd", extCtx.Cwd,
			)
			if !cliout.IsJSON() {
				fmt.Fprintf(os.Stderr, "%s[DEBUG]%s Build: %s (built on %s, commit: %.8s)\n",
					cliout.Dim, cliout.Reset, commands.Version, commands.BuildTime, commands.Commit)
			}
		}

		// Install Copilot skill
		if err := skills.InstallSkill(); err != nil {
			if extCtx.Debug {
				slog.Debug("Failed to install copilot skill", "error", err)
			}
		}

		return cliout.SetFormat(extCtx.OutputFormat)
	}

	// Register all commands
	rootCmd.AddCommand(
		commands.NewReqsCommand(),
		commands.NewRunCommand(),
		commands.NewDepsCommand(),
		commands.NewTestCommand(),
		commands.NewLogsCommand(),
		commands.NewInfoCommand(),
		commands.NewHealthCommand(),
		commands.NewVersionCommand(&extCtx.OutputFormat),
		commands.NewNotificationsCommand(),
		commands.NewListenCommand(), // Required for azd extension framework
		commands.NewMCPCommand(),    // Model Context Protocol server
		commands.NewStartCommand(),
		commands.NewStopCommand(),
		commands.NewRestartCommand(),
		commands.NewAddCommand(),
		commands.NewMetadataCommand(func() *cobra.Command { return rootCmd }),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
