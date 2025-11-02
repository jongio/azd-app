package main

import (
	"fmt"
	"os"

	"github.com/jongio/azd-app/cli/src/cmd/app/commands"
	"github.com/jongio/azd-app/cli/src/internal/output"

	"github.com/spf13/cobra"
)

var outputFormat string

func main() {
	rootCmd := &cobra.Command{
		Use:   "app",
		Short: "App - Automate your development environment setup",
		Long:  `App is an Azure Developer CLI extension that automatically detects and sets up your development environment across multiple languages and frameworks.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Set global output format from the flag
			return output.SetFormat(outputFormat)
		},
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "default", "Output format (default, json)")

	// Register all commands
	rootCmd.AddCommand(
		commands.NewReqsCommand(),
		commands.NewRunCommand(),
		commands.NewDepsCommand(),
		commands.NewLogsCommand(),
		commands.NewInfoCommand(),
		commands.NewVersionCommand(),
		commands.NewListenCommand(), // Required for azd extension framework
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
