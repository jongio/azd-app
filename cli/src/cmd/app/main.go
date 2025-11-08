package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/jongio/azd-app/cli/src/cmd/app/commands"
	"github.com/jongio/azd-app/cli/src/internal/logging"
	"github.com/jongio/azd-app/cli/src/internal/output"

	"github.com/spf13/cobra"
)

var (
	outputFormat   string
	debugMode      bool
	structuredLogs bool
	cwdFlag        string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "app",
		Short: "App - Automate your development environment setup",
		Long:  `App is an Azure Developer CLI extension that automatically detects and sets up your development environment across multiple languages and frameworks.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Change working directory if --cwd is specified
			if cwdFlag != "" {
				if err := os.Chdir(cwdFlag); err != nil {
					return fmt.Errorf("failed to change to directory '%s': %w", cwdFlag, err)
				}
			}

			// Set global output format and debug mode
			if debugMode {
				os.Setenv("AZD_APP_DEBUG", "true")
				// Configure slog to show debug messages
				slog.SetLogLoggerLevel(slog.LevelDebug)
			}

			// Configure logging
			logging.SetupLogger(debugMode, structuredLogs)

			// Log startup in debug mode
			if debugMode {
				logging.Debug("Starting azd app extension",
					"version", commands.Version,
					"command", cmd.Name(),
					"args", args,
					"cwd", cwdFlag,
				)
				// Print build info in debug mode (before command output)
				if !output.IsJSON() {
					fmt.Fprintf(os.Stderr, "%s[DEBUG]%s Build: %s (built on %s, commit: %.8s)\n",
						output.Dim, output.Reset, commands.Version, commands.BuildTime, commands.Commit)
				}
			}

			return output.SetFormat(outputFormat)
		},
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "default", "Output format (default, json)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&structuredLogs, "structured-logs", false, "Enable structured JSON logging to stderr")
	rootCmd.PersistentFlags().StringVarP(&cwdFlag, "cwd", "C", "", "Sets the current working directory")

	// Register all commands
	rootCmd.AddCommand(
		commands.NewReqsCommand(),
		commands.NewRunCommand(),
		commands.NewDepsCommand(),
		commands.NewTestCommand(),
		commands.NewLogsCommand(),
		commands.NewInfoCommand(),
		commands.NewHealthCommand(),
		commands.NewVersionCommand(),
		commands.NewNotificationsCommand(),
		commands.NewListenCommand(), // Required for azd extension framework
		commands.NewMCPCommand(),    // Model Context Protocol server
		commands.NewStartCommand(),
		commands.NewStopCommand(),
		commands.NewRestartCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
