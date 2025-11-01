//go:build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Variables
const (
	binaryName  = "app"
	srcDir      = "src/cmd/app"
	binDir      = "bin"
	coverageDir = "coverage"
)

// Default target runs Lint, Test, and Build.
var Default = All

// All runs lint, test, and build.
func All() error {
	mg.Deps(Lint, Test, Build)
	return nil
}

// Build compiles the app binary for the current platform.
func Build() error {
	fmt.Println("Building", binaryName+"...")

	output := filepath.Join(binDir, binaryName)
	if runtime.GOOS == "windows" {
		output += ".exe"
	}

	if err := os.MkdirAll(binDir, 0750); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	if err := sh.RunV("go", "build", "-o", output, "./"+srcDir); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Println("✅ Build complete!")
	return nil
}

// BuildAll builds for all platforms using build.ps1.
func BuildAll() error {
	fmt.Println("Building for all platforms...")
	return sh.RunV("powershell", "-File", "build.ps1", "-All")
}

// Test runs unit tests only (with -short flag).
func Test() error {
	fmt.Println("Running unit tests...")
	return sh.RunV("go", "test", "-v", "-short", "./src/...")
}

// TestIntegration runs integration tests only.
func TestIntegration() error {
	fmt.Println("Running integration tests...")
	return sh.RunV("go", "test", "-v", "-tags=integration", "./src/...")
}

// TestAll runs all tests (unit + integration).
func TestAll() error {
	fmt.Println("Running all tests...")
	return sh.RunV("go", "test", "-v", "-tags=integration", "./src/...")
}

// TestCoverage runs tests with coverage report.
func TestCoverage() error {
	fmt.Println("Running tests with coverage...")

	if err := os.MkdirAll(coverageDir, 0750); err != nil {
		return fmt.Errorf("failed to create coverage directory: %w", err)
	}

	coverageOut := filepath.Join(coverageDir, "coverage.out")
	coverageHTML := filepath.Join(coverageDir, "coverage.html")

	// Run tests with coverage
	if err := sh.RunV("go", "test", "-v", "-coverprofile="+coverageOut, "./src/..."); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}

	// Generate HTML report
	if err := sh.RunV("go", "tool", "cover", "-html="+coverageOut, "-o", coverageHTML); err != nil {
		return fmt.Errorf("failed to generate HTML coverage: %w", err)
	}

	// Display coverage summary
	if err := sh.RunV("go", "tool", "cover", "-func="+coverageOut); err != nil {
		return fmt.Errorf("failed to display coverage summary: %w", err)
	}

	fmt.Println("Coverage report:", coverageHTML)
	return nil
}

// Lint runs golangci-lint on the codebase.
func Lint() error {
	fmt.Println("Running golangci-lint...")

	// Check if golangci-lint is installed
	if err := sh.RunV("golangci-lint", "run", "./..."); err != nil {
		fmt.Println("⚠️  golangci-lint not installed or failed.")
		fmt.Println("To install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest")
		return err
	}

	return nil
}

// Fmt formats all Go code using gofmt.
func Fmt() error {
	fmt.Println("Formatting code...")

	if err := sh.RunV("gofmt", "-w", "-s", "."); err != nil {
		return fmt.Errorf("formatting failed: %w", err)
	}

	fmt.Println("✅ Code formatted!")
	return nil
}

// Clean removes build artifacts and coverage reports.
func Clean() error {
	fmt.Println("Cleaning build artifacts...")

	dirs := []string{binDir, coverageDir}
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("failed to remove %s: %w", dir, err)
		}
	}

	fmt.Println("✅ Clean complete!")
	return nil
}

// Install installs the extension locally using azd x build.
func Install() error {
	fmt.Println("Installing locally...")
	// Use pwsh (PowerShell 7) instead of powershell (Windows PowerShell 5.1) for better UTF-8 support
	return sh.RunV("pwsh", "-File", "install-local.ps1")
}
