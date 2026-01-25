package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/jongio/azd-app/cli/src/cmd/app/commands"
	"github.com/jongio/azd-app/cli/src/internal/logging"
	"github.com/jongio/azd-core/cliout"

	"github.com/spf13/cobra"
)

var (
	outputFormat   string
	debugMode      bool
	structuredLogs bool
	cwdFlag        string
	environment    string
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

			// Handle environment selection
			if environment != "" {
				// Load environment variables from the specified environment
				if err := loadAzdEnvironment(cmd.Context(), environment); err != nil {
					return fmt.Errorf("failed to load environment '%s': %w", environment, err)
				}
			}

			// Set global output format and debug mode
			if debugMode {
				os.Setenv("AZD_DEBUG", "true")
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
				if !cliout.IsJSON() {
					fmt.Fprintf(os.Stderr, "%s[DEBUG]%s Build: %s (built on %s, commit: %.8s)\n",
						cliout.Dim, cliout.Reset, commands.Version, commands.BuildTime, commands.Commit)
				}
			}

			return cliout.SetFormat(outputFormat)
		},
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "default", "Output format (default, json)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&structuredLogs, "structured-logs", false, "Enable structured JSON logging to stderr")
	rootCmd.PersistentFlags().StringVarP(&cwdFlag, "cwd", "C", "", "Sets the current working directory")
	rootCmd.PersistentFlags().StringVarP(&environment, "environment", "e", "", "The name of the environment to use")

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
		commands.NewAddCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// loadAzdEnvironment loads all environment variables from the specified azd environment.
// This ensures that when -e flag is used, we get the correct environment values.
//
// WORKAROUND: This is a workaround for a limitation in the azd extension framework.
// Ideally, azd should honor the -e flag before invoking the extension and inject
// the correct environment variables. However, currently azd injects the default
// environment from config.json, then passes the -e flag to the extension.
// This forces us to manually reload the environment using 'azd env get-values'.
//
// See: https://github.com/Azure/azure-dev/issues/[TBD] for tracking the framework enhancement.
func loadAzdEnvironment(ctx context.Context, envName string) error {
	// Validate envName to prevent injection attacks
	if envName == "" {
		return fmt.Errorf("environment name cannot be empty")
	}
	if strings.Contains(envName, " ") || strings.Contains(envName, ";") || strings.Contains(envName, "&") {
		return fmt.Errorf("invalid environment name: %q", envName)
	}

	// Use 'azd env get-values' with the -e flag to get environment variables
	cmd := exec.CommandContext(ctx, "azd", "env", "get-values", "-e", envName, "--output", "json")

	output, err := cmd.Output()
	if err != nil {
		// If azd env get-values fails, try without JSON output (older azd versions)
		cmd = exec.CommandContext(ctx, "azd", "env", "get-values", "-e", envName)
		output, err = cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get environment values for '%s': %w", envName, err)
		}

		// Parse key=value format
		values, parseErr := parseKeyValueFormat(output)
		if parseErr != nil {
			return fmt.Errorf("failed to parse environment values: %w", parseErr)
		}

		// Set all environment variables
		for key, value := range values {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("failed to set environment variable %s: %w", key, err)
			}
		}
		return nil
	}

	// Parse JSON output
	var values map[string]string
	if err := json.Unmarshal(output, &values); err != nil {
		return fmt.Errorf("failed to parse environment values as JSON: %w", err)
	}

	// Set all environment variables
	for key, value := range values {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", key, err)
		}
	}

	return nil
}

// parseKeyValueFormat parses output in "KEY=value" format (one per line).
// Handles quoted values and skips empty lines and comments.
func parseKeyValueFormat(output []byte) (map[string]string, error) {
	values := make(map[string]string)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Find the first '=' separator
		idx := strings.Index(line, "=")
		if idx <= 0 || idx == len(line)-1 {
			// Invalid line: no '=', '=' at start, or '=' at end
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := line[idx+1:]

		// Validate key (must be non-empty and contain only allowed characters)
		if key == "" {
			continue
		}

		// Remove surrounding quotes if present (handles both " and ')
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		values[key] = value
	}

	// Log for debugging if we got zero values (likely an error condition)
	if len(values) == 0 {
		slog.Debug("No environment variables parsed from key=value output")
	}

	return values, nil
}
