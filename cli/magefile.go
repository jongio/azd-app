//go:build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Variables
const (
	binaryName  = "app"
	srcDir      = "src/cmd/app"
	binDir      = "bin"
	coverageDir = "coverage"
	versionFile = "version.txt"
)

// Default target runs Lint, Test, and Build.
var Default = All

// getVersion reads the current version from version.txt.
func getVersion() (string, error) {
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "", fmt.Errorf("failed to read version file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// bumpVersion increments the patch version and writes it back.
func bumpVersion() (string, error) {
	version, err := getVersion()
	if err != nil {
		return "", err
	}

	// Parse version (simple semver: major.minor.patch)
	var major, minor, patch int
	if _, err := fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &patch); err != nil {
		return "", fmt.Errorf("failed to parse version %s: %w", version, err)
	}

	// Increment patch
	patch++
	newVersion := fmt.Sprintf("%d.%d.%d", major, minor, patch)

	// Write back
	if err := os.WriteFile(versionFile, []byte(newVersion+"\n"), 0644); err != nil {
		return "", fmt.Errorf("failed to write version file: %w", err)
	}

	return newVersion, nil
}

// All runs lint, test, and build.
func All() error {
	mg.Deps(Lint, Test, Build)
	return nil
}

// Build compiles the app binary for the current platform with version info.
func Build() error {
	fmt.Println("Building", binaryName+"...")

	// Bump version
	version, err := bumpVersion()
	if err != nil {
		return err
	}

	output := filepath.Join(binDir, binaryName)
	if runtime.GOOS == "windows" {
		output += ".exe"
	}

	if err := os.MkdirAll(binDir, 0750); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Build with version injected via ldflags
	buildTime := time.Now().Format(time.RFC3339)
	ldflags := fmt.Sprintf("-X github.com/jongio/azd-app/cli/src/cmd/app/commands.Version=%s -X github.com/jongio/azd-app/cli/src/cmd/app/commands.BuildTime=%s", version, buildTime)

	if err := sh.RunV("go", "build", "-ldflags", ldflags, "-o", output, "./"+srcDir); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Printf("✅ Build complete! Version: %s\n", version)
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
	// First build to get the version
	if err := Build(); err != nil {
		return err
	}

	version, err := getVersion()
	if err != nil {
		return err
	}

	fmt.Println("Installing locally...")
	// Use pwsh (PowerShell 7) instead of powershell (Windows PowerShell 5.1) for better UTF-8 support
	if err := sh.RunV("pwsh", "-File", "install-local.ps1"); err != nil {
		return err
	}

	fmt.Printf("✅ Installed version: %s\n", version)
	return nil
}

// Uninstall removes the locally installed extension.
func Uninstall() error {
	fmt.Println("Uninstalling extension...")
	if err := sh.RunV("azd", "extension", "uninstall", "app"); err != nil {
		return fmt.Errorf("failed to uninstall extension: %w", err)
	}

	fmt.Println("✅ Extension uninstalled!")
	return nil
}

// Preflight runs all checks before shipping: format, lint, security, tests, and coverage.
func Preflight() error {
	fmt.Println("🚀 Running preflight checks...")
	fmt.Println()

	// 1. Format check
	fmt.Println("📝 Step 1/6: Checking code formatting...")
	if err := Fmt(); err != nil {
		return fmt.Errorf("formatting failed: %w", err)
	}
	fmt.Println()

	// 2. Lint
	fmt.Println("🔍 Step 2/6: Running linter...")
	if err := Lint(); err != nil {
		return fmt.Errorf("linting failed: %w", err)
	}
	fmt.Println()

	// 3. Security scan with gosec
	fmt.Println("🔒 Step 3/6: Running security scan (gosec)...")
	if err := sh.RunV("gosec", "-quiet", "./..."); err != nil {
		fmt.Println("⚠️  gosec not installed or found security issues.")
		fmt.Println("To install: go install github.com/securego/gosec/v2/cmd/gosec@latest")
		return err
	}
	fmt.Println()

	// 4. Spell check with typos
	fmt.Println("✏️  Step 4/6: Running spell check (typos)...")
	if err := sh.RunV("typos", "."); err != nil {
		fmt.Println("⚠️  typos not installed or found spelling errors.")
		fmt.Println("To install: cargo install typos-cli")
		fmt.Println("Or download from: https://github.com/crate-ci/typos")
		return err
	}
	fmt.Println()

	// 5. Run all tests
	fmt.Println("🧪 Step 5/6: Running all tests...")
	if err := TestAll(); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}
	fmt.Println()

	// 6. Generate coverage report
	fmt.Println("📊 Step 6/6: Generating coverage report...")
	if err := TestCoverage(); err != nil {
		return fmt.Errorf("coverage generation failed: %w", err)
	}
	fmt.Println()

	fmt.Println("✅ All preflight checks passed!")
	fmt.Println("🎉 Ready to ship!")
	return nil
}
