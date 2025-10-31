package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "devstack",
		Short: "DevStack - Developer productivity commands for Azure Developer CLI",
		Long: `DevStack Extension provides a collection of developer productivity 
commands and utilities to enhance your Azure Developer CLI experience.`,
	}

	// Add commands
	rootCmd.AddCommand(newHiCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
