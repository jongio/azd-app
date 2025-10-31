package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newHiCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "hi",
		Short: "Say hello from DevStack extension",
		Long: `The hi command displays a friendly greeting and shows that the 
DevStack extension is working correctly.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("👋 Hi from DevStack Extension!")
			fmt.Println()
			fmt.Println("🚀 DevStack Extension v0.1.0")
			fmt.Println("   A collection of developer productivity commands for Azure Developer CLI")
			fmt.Println()
			fmt.Println("   Ready to help you build amazing things! 💡")
			return nil
		},
	}
}
