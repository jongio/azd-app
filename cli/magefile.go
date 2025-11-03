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

const (
	binaryName  = "app"
	srcDir      = "src/cmd/app"
	binDir      = "bin"
	coverageDir = "coverage"
	versionFile = "version.txt"
)

// Default target runs all checks and builds.
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
	if err := os.WriteFile(versionFile, []byte(newVersion+"\n"), 0o644); err != nil {
		return "", fmt.Errorf("failed to write version file: %w", err)
	}

	return newVersion, nil
}

// All runs lint, test, and build in dependency order.
func All() error {
	mg.Deps(DashboardBuild, Fmt, Lint, Test)
	return Build()
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

	if err := os.MkdirAll(binDir, 0o755); err != nil {
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
	return sh.RunV("pwsh", "-File", "build.ps1", "-All")
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

	if err := os.MkdirAll(coverageDir, 0o755); err != nil {
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
	if err := sh.RunV("golangci-lint", "run", "./..."); err != nil {
		fmt.Println("⚠️  Linting failed. Ensure golangci-lint is installed:")
		fmt.Println("    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest")
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

// Install builds and installs the extension locally.
func Install() error {
	if err := Build(); err != nil {
		return err
	}

	version, err := getVersion()
	if err != nil {
		return err
	}

	fmt.Println("Installing locally...")
	if err := sh.RunV("pwsh", "-File", "install-local.ps1"); err != nil {
		return fmt.Errorf("installation failed: %w", err)
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

	checks := []struct {
		name string
		step int
		fn   func() error
	}{
		{"Building and linting dashboard", 1, DashboardBuild},
		{"Formatting code", 2, Fmt},
		{"Running linter (golangci-lint with misspell)", 3, Lint},
		{"Running security scan (gosec)", 4, runGosec},
		{"Running all tests", 5, TestAll},
		{"Generating coverage report", 6, TestCoverage},
	}

	for _, check := range checks {
		fmt.Printf("📋 Step %d/%d: %s...\n", check.step, len(checks), check.name)
		if err := check.fn(); err != nil {
			return fmt.Errorf("%s failed: %w", check.name, err)
		}
		fmt.Println()
	}

	fmt.Println("✅ All preflight checks passed!")
	fmt.Println("🎉 Ready to ship!")
	return nil
}

// runGosec runs security scanning with gosec.
func runGosec() error {
	if err := sh.RunV("gosec", "-quiet", "./..."); err != nil {
		fmt.Println("⚠️  Security scan failed. Ensure gosec is installed:")
		fmt.Println("    go install github.com/securego/gosec/v2/cmd/gosec@latest")
		return err
	}
	return nil
}

// DashboardBuild builds the dashboard TypeScript/React code.
func DashboardBuild() error {
	fmt.Println("Building dashboard...")
	
	dashboardDir := "dashboard"
	
	// Install dependencies
	fmt.Println("Installing dashboard dependencies...")
	if err := sh.RunWith(map[string]string{"npm_config_update_notifier": "false"}, "npm", "install", "--prefix", dashboardDir); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}
	
	// Run TypeScript compilation and build
	fmt.Println("Building dashboard assets...")
	if err := sh.RunV("npm", "run", "build", "--prefix", dashboardDir); err != nil {
		return fmt.Errorf("dashboard build failed: %w", err)
	}
	
	fmt.Println("✅ Dashboard build complete!")
	return nil
}

// DashboardDev runs the dashboard in development mode with hot reload.
func DashboardDev() error {
	fmt.Println("Starting dashboard development server...")
	return sh.RunV("npm", "run", "dev", "--prefix", "dashboard")
}
