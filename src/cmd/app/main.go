package main

import (
	"fmt"
	"os"

	"app/src/cmd/app/commands"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "app",
		Short: "App - Automate your development environment setup",
		Long:  `App is an Azure Developer CLI extension that automatically detects and sets up your development environment across multiple languages and frameworks.`,
	}

	// Register all commands
	rootCmd.AddCommand(
		commands.NewReqsCommand(),
		commands.NewRunCommand(),
		commands.NewDepsCommand(),
		commands.NewInfoCommand(),
		commands.NewVersionCommand(),
		commands.NewListenCommand(), // Required for azd extension framework
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
