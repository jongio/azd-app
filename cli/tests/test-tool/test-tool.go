package main

import (
	"fmt"
	"os"
)

const version = "2.5.0"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("test-tool version %s\n", version)
			os.Exit(0)
		case "--help", "-h", "help":
			fmt.Println("test-tool - A simple test utility for PATH resolution testing")
			fmt.Println()
			fmt.Println("Usage:")
			fmt.Println("  test-tool --version    Show version")
			fmt.Println("  test-tool --help       Show this help")
			fmt.Println("  test-tool hello        Print a greeting")
			os.Exit(0)
		case "hello":
			fmt.Println("Hello from test-tool!")
			os.Exit(0)
		}
	}

	fmt.Println("test-tool v" + version)
	fmt.Println("Run 'test-tool --help' for usage information")
}
