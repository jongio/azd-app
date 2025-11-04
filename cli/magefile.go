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

	platforms := []struct {
		goos   string
		goarch string
		ext    string
	}{
		{"windows", "amd64", ".exe"},
		{"windows", "arm64", ".exe"},
		{"linux", "amd64", ""},
		{"linux", "arm64", ""},
		{"darwin", "amd64", ""},
		{"darwin", "arm64", ""},
	}

	for _, p := range platforms {
		platformDir := filepath.Join(binDir, fmt.Sprintf("%s-%s", p.goos, p.goarch))
		if err := os.MkdirAll(platformDir, 0o755); err != nil {
			return fmt.Errorf("failed to create platform directory: %w", err)
		}

		output := filepath.Join(platformDir, binaryName+p.ext)
		fmt.Printf("Building for %s/%s...\n", p.goos, p.goarch)

		env := map[string]string{
			"GOOS":   p.goos,
			"GOARCH": p.goarch,
		}

		if err := sh.RunWith(env, "go", "build", "-o", output, "./"+srcDir); err != nil {
			return fmt.Errorf("build failed for %s/%s: %w", p.goos, p.goarch, err)
		}
	}

	fmt.Println("✅ Build complete for all platforms!")
	return nil
}

// Test runs unit tests only (with -short flag).
func Test() error {
	fmt.Println("Running unit tests...")
	return sh.RunV("go", "test", "-v", "-short", "./src/...")
}

// TestIntegration runs integration tests only.
// Set TEST_PACKAGE env var to filter by package (e.g., installer, runner, commands)
// Set TEST_NAME env var to run a specific test
// Set TEST_TIMEOUT env var to override default 10m timeout
func TestIntegration() error {
	fmt.Println("Running integration tests...")

	args := []string{"test", "-v", "-tags=integration"}

	// Handle timeout
	timeout := os.Getenv("TEST_TIMEOUT")
	if timeout == "" {
		timeout = "10m"
	}
	args = append(args, "-timeout="+timeout)

	// Handle test filtering
	testName := os.Getenv("TEST_NAME")
	if testName != "" {
		args = append(args, "-run="+testName)
	}

	// Handle package filtering
	pkg := os.Getenv("TEST_PACKAGE")
	testPath := "./src/..."
	if pkg != "" {
		switch pkg {
		case "installer":
			testPath = "./src/internal/installer"
		case "runner":
			testPath = "./src/internal/runner"
		case "commands":
			testPath = "./src/cmd/app/commands"
		default:
			return fmt.Errorf("unknown package: %s (valid: installer, runner, commands)", pkg)
		}
	}
	args = append(args, testPath)

	return sh.RunV("go", args...)
}

// TestAll runs all tests (unit + integration).
func TestAll() error {
	fmt.Println("Running all tests...")
	return sh.RunV("go", "test", "-v", "-tags=integration", "./src/...")
}

// TestCoverage runs tests with coverage report.
func TestCoverage() error {
	fmt.Println("Running tests with coverage...")

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create absolute path for coverage directory
	absCoverageDir := filepath.Join(cwd, coverageDir)

	// Remove existing coverage directory to ensure clean state
	os.RemoveAll(absCoverageDir)

	// Create coverage directory
	if err := os.MkdirAll(absCoverageDir, 0o755); err != nil {
		return fmt.Errorf("failed to create coverage directory at %s: %w", absCoverageDir, err)
	}

	coverageOut := filepath.Join(absCoverageDir, "coverage.out")
	coverageHTML := filepath.Join(absCoverageDir, "coverage.html")

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

// Coverage is an alias for TestCoverage for easier access.
func Coverage() error {
	return TestCoverage()
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
	mg.Deps(DashboardBuild)

	if err := Build(); err != nil {
		return err
	}

	version, err := getVersion()
	if err != nil {
		return err
	}

	fmt.Println("Installing locally with azd x build...")

	// Run azd x build which handles the installation
	if err := sh.RunV("azd", "x", "build"); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Println("Installing extension with azd extension install...")

	// Install the extension to make it available as 'azd app'
	if err := sh.RunV("azd", "extension", "install", "jongio.azd.app"); err != nil {
		// If it fails because already installed, that's ok - try updating instead
		fmt.Println("Extension already installed, updating...")
		if err := sh.RunV("azd", "extension", "update", "jongio.azd.app"); err != nil {
			return fmt.Errorf("failed to install/update extension: %w", err)
		}
	}

	fmt.Printf("✅ Installed version: %s\n", version)
	fmt.Println("\nTest it: azd app version")
	return nil
}

// Uninstall removes the locally installed extension.
func Uninstall() error {
	fmt.Println("Uninstalling extension...")
	if err := sh.RunV("azd", "extension", "uninstall", "jongio.azd.app"); err != nil {
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
		{"Running all tests with coverage", 5, TestCoverage},
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

// InstallFast builds and installs quickly without dashboard build (for rapid iteration).
func InstallFast() error {
	fmt.Println("Fast install mode - skipping dashboard build...")

	if err := Build(); err != nil {
		return err
	}

	version, err := getVersion()
	if err != nil {
		return err
	}

	fmt.Println("Installing locally with azd x build...")

	if err := sh.RunV("azd", "x", "build"); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Println("Installing extension with azd extension install...")

	// Install the extension to make it available as 'azd app'
	if err := sh.RunV("azd", "extension", "install", "jongio.azd.app"); err != nil {
		// If it fails because already installed, that's ok - try updating instead
		fmt.Println("Extension already installed, updating...")
		if err := sh.RunV("azd", "extension", "update", "jongio.azd.app"); err != nil {
			return fmt.Errorf("failed to install/update extension: %w", err)
		}
	}

	fmt.Printf("✅ Fast install complete! Version: %s\n", version)
	return nil
}

// Run builds and runs the app directly in a test project (without installing as extension).
func Run() error {
	projectDir := os.Getenv("PROJECT_DIR")
	if projectDir == "" {
		projectDir = "tests/projects/fullstack-test"
	}

	command := os.Getenv("COMMAND")
	if command == "" {
		command = "run"
	}

	fmt.Printf("Building and running: %s in %s\n", command, projectDir)

	// Build first
	if err := Build(); err != nil {
		return err
	}

	// Get the binary path
	binaryExt := ""
	if runtime.GOOS == "windows" {
		binaryExt = ".exe"
	}
	binaryPath := filepath.Join(binDir, binaryName+binaryExt)

	// Change to project directory
	originalDir, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(projectDir); err != nil {
		return fmt.Errorf("failed to change to project directory: %w", err)
	}

	fmt.Printf("🚀 Running in %s: %s %s\n\n", projectDir, binaryPath, command)

	// Get absolute binary path since we changed directories
	absBinaryPath := filepath.Join(originalDir, binaryPath)
	return sh.RunV(absBinaryPath, command)
}

// Release creates a new release with automatic version bumping.
func Release() error {
	fmt.Println("Starting release process...")
	return sh.RunV("pwsh", "-File", "scripts/release.ps1")
}

// ReleasePatch creates a patch release (bug fixes).
func ReleasePatch() error {
	fmt.Println("Creating patch release...")
	return sh.RunV("pwsh", "-File", "scripts/release.ps1", "-BumpType", "Patch")
}

// ReleaseMinor creates a minor release (new features).
func ReleaseMinor() error {
	fmt.Println("Creating minor release...")
	return sh.RunV("pwsh", "-File", "scripts/release.ps1", "-BumpType", "Minor")
}

// ReleaseMajor creates a major release (breaking changes).
func ReleaseMajor() error {
	fmt.Println("Creating major release...")
	return sh.RunV("pwsh", "-File", "scripts/release.ps1", "-BumpType", "Major")
}
