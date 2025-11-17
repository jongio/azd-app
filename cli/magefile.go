//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	binaryName         = "app"
	srcDir             = "src/cmd/app"
	binDir             = "bin"
	coverageDir        = "coverage"
	extensionFile      = "extension.yaml"
	dashboardDir       = "dashboard"
	defaultTestTimeout = "10m"
	extensionID        = "jongio.azd.app"
)

// Default target runs all checks and builds.
var Default = All

// getVersion reads the current version from extension.yaml.
func getVersion() (string, error) {
	data, err := os.ReadFile(extensionFile)
	if err != nil {
		return "", fmt.Errorf("failed to read extension.yaml: %w", err)
	}

	// Simple regex to extract version: line
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "version:") {
			version := strings.TrimSpace(strings.TrimPrefix(line, "version:"))
			return version, nil
		}
	}
	return "", fmt.Errorf("version not found in extension.yaml")
}

// All runs lint, test, and build in dependency order.
func All() error {
	mg.Deps(Fmt, DashboardBuild, Lint, Test)
	return Build()
}

// Build compiles the app binary for the current platform with version info.
func Build() error {
	fmt.Println("Building", binaryName+"...")

	// Get version from extension.yaml (don't bump on every build)
	version, err := getVersion()
	if err != nil {
		return err
	}

	// Detect current platform
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	platform := fmt.Sprintf("%s/%s", goos, goarch)

	// Set environment variables for build script
	env := map[string]string{
		"EXTENSION_ID":       extensionID,
		"EXTENSION_VERSION":  version,
		"EXTENSION_PLATFORM": platform,
	}

	// Call build script directly
	var buildScript string
	if runtime.GOOS == "windows" {
		buildScript = "build.ps1"
		if err := sh.RunWithV(env, "pwsh", "-File", buildScript); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}
	} else {
		buildScript = "build.sh"
		if err := sh.RunWithV(env, "bash", buildScript); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}
	}

	fmt.Printf("✅ Build complete! Version: %s\n", version)
	return nil
}

// BuildAll builds for all platforms.
func BuildAll() error {
	fmt.Println("Building for all platforms...")

	// Get version from extension.yaml (don't bump on every build)
	version, err := getVersion()
	if err != nil {
		return err
	}

	// Set environment variables for build script
	// When EXTENSION_PLATFORM is not set, the script builds for all platforms
	env := map[string]string{
		"EXTENSION_ID":      extensionID,
		"EXTENSION_VERSION": version,
	}

	// Call build script directly
	var buildScript string
	if runtime.GOOS == "windows" {
		buildScript = "build.ps1"
		if err := sh.RunWithV(env, "pwsh", "-File", buildScript); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}
	} else {
		buildScript = "build.sh"
		if err := sh.RunWithV(env, "bash", buildScript); err != nil {
			return fmt.Errorf("build failed: %w", err)
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
		timeout = defaultTestTimeout
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

// TestVisual runs visual tests for progress bar rendering at multiple terminal widths.
// Generates an HTML report with screenshots showing terminal output at 50, 80, and 120 characters.
// Analyzes duplicate progress bar detection to ensure proper terminal width handling.
func TestVisual() error {
	fmt.Println("Running visual tests for progress bars...")

	visualTestDir := filepath.Join("tests", "visual-test")

	// Check if visual test exists
	if _, err := os.Stat(visualTestDir); os.IsNotExist(err) {
		fmt.Println("⚠️  Visual test directory not found:", visualTestDir)
		return nil
	}

	// Build the test binary first (Windows workaround for go run PATH issues)
	testBinary := filepath.Join(visualTestDir, "visual-test.exe")
	if runtime.GOOS != "windows" {
		testBinary = filepath.Join(visualTestDir, "visual-test")
	}

	buildCmd := exec.Command("go", "build", "-o", testBinary, "main.go")
	buildCmd.Dir = visualTestDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	// Ensure we build for the host platform by explicitly clearing cross-compile vars
	env := []string{}
	for _, e := range os.Environ() {
		// Skip GOOS and GOARCH from parent environment
		if !strings.HasPrefix(e, "GOOS=") && !strings.HasPrefix(e, "GOARCH=") {
			env = append(env, e)
		}
	}
	buildCmd.Env = env

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build visual test: %w", err)
	}

	// Run the built binary
	cmd := exec.Command(testBinary)
	cmd.Dir = visualTestDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("visual tests failed: %w", err)
	}

	// Report location
	reportPath := filepath.Join(visualTestDir, "test-output", "visual-report.html")
	absReportPath, _ := filepath.Abs(reportPath)
	fmt.Printf("\n📊 Visual test report: %s\n", absReportPath)

	return nil
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
	_ = os.RemoveAll(absCoverageDir) // Ignore error if directory doesn't exist

	// Create coverage directory
	if err := os.MkdirAll(absCoverageDir, 0o755); err != nil {
		return fmt.Errorf("failed to create coverage directory at %s: %w", absCoverageDir, err)
	}

	coverageOut := filepath.Join(absCoverageDir, "coverage.out")
	coverageHTML := filepath.Join(absCoverageDir, "coverage.html")

	// Run tests with coverage (use -short to skip integration tests)
	if err := sh.RunV("go", "test", "-v", "-short", "-coverprofile="+coverageOut, "./src/..."); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}

	// Generate HTML report
	// Note: Go 1.25+ changed cover tool syntax, use output redirection
	coverCmd := exec.Command("go", "tool", "cover", "-html="+coverageOut)
	coverOutput, err := coverCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to generate HTML coverage: %w", err)
	}
	// Use restrictive permissions (0600) to protect potentially sensitive coverage data
	if err := os.WriteFile(coverageHTML, coverOutput, 0o600); err != nil {
		return fmt.Errorf("failed to write HTML coverage file: %w", err)
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
	// Use same command as CI to ensure consistency
	if err := sh.RunV("golangci-lint", "run", "--timeout=5m"); err != nil {
		fmt.Println("⚠️  Linting failed. Ensure golangci-lint is installed:")
		fmt.Println("    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest")
		return err
	}
	return nil
}

// LintAll runs golangci-lint with all linters enabled for comprehensive checking.
// This is more strict than Lint() and may report many issues.
func LintAll() error {
	fmt.Println("Running comprehensive linting with all linters...")
	// Enable all linters except some noisy ones (ignore config file to avoid conflicts)
	excludeLinters := "exhaustruct,exhaustive,varnamelen,gochecknoglobals,gochecknoinits,wrapcheck,paralleltest,tparallel,nlreturn,wsl,funlen,cyclop,gocognit,maintidx,lll,tagliatelle"
	if err := sh.RunV("golangci-lint", "run", "--no-config", "--enable-all", "--disable="+excludeLinters, "--max-issues-per-linter=0", "--max-same-issues=0", "./..."); err != nil {
		fmt.Println("⚠️  Comprehensive linting found issues.")
		fmt.Println("    Some findings may be opinionated. Review and fix critical issues.")
		fmt.Println("    Ensure golangci-lint is installed: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest")
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

// Vet runs go vet to check for suspicious constructs.
func Vet() error {
	fmt.Println("Running go vet...")
	if err := sh.RunV("go", "vet", "./..."); err != nil {
		return fmt.Errorf("go vet found issues: %w", err)
	}
	fmt.Println("✅ go vet passed!")
	return nil
}

// Staticcheck runs staticcheck for advanced static analysis.
func Staticcheck() error {
	fmt.Println("Running staticcheck...")
	if err := sh.RunV("staticcheck", "./..."); err != nil {
		fmt.Println("⚠️  staticcheck found issues. Ensure staticcheck is installed:")
		fmt.Println("    go install honnef.co/go/tools/cmd/staticcheck@latest")
		return err
	}
	fmt.Println("✅ staticcheck passed!")
	return nil
}

// ModTidy ensures go.mod and go.sum are tidy.
func ModTidy() error {
	fmt.Println("Running go mod tidy...")
	if err := sh.RunV("go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}

	// Check if there are any changes
	if err := sh.RunV("git", "diff", "--exit-code", "go.mod", "go.sum"); err != nil {
		return fmt.Errorf("go.mod or go.sum has uncommitted changes after running go mod tidy - please review and commit these changes")
	}

	fmt.Println("✅ go mod tidy passed!")
	return nil
}

// ModVerify verifies dependencies have expected content.
func ModVerify() error {
	fmt.Println("Running go mod verify...")
	if err := sh.RunV("go", "mod", "verify"); err != nil {
		return fmt.Errorf("go mod verify failed: %w", err)
	}
	fmt.Println("✅ go mod verify passed!")
	return nil
}

// Vulncheck runs govulncheck to check for known vulnerabilities.
func Vulncheck() error {
	fmt.Println("Running govulncheck...")
	if err := sh.RunV("govulncheck", "./..."); err != nil {
		fmt.Println("⚠️  govulncheck found vulnerabilities. Ensure govulncheck is installed:")
		fmt.Println("    go install golang.org/x/vuln/cmd/govulncheck@latest")
		return err
	}
	fmt.Println("✅ No known vulnerabilities found!")
	return nil
}

// runVulncheck runs govulncheck if available, otherwise skips.
func runVulncheck() error {
	fmt.Println("Checking for known vulnerabilities...")
	// Check if govulncheck is installed
	if _, err := exec.LookPath("govulncheck"); err != nil {
		fmt.Println("⚠️  govulncheck not installed - skipping vulnerability check")
		fmt.Println("    Install with: go install golang.org/x/vuln/cmd/govulncheck@latest")
		return nil // Don't fail preflight if not installed
	}

	if err := sh.RunV("govulncheck", "./..."); err != nil {
		fmt.Println("⚠️  Known vulnerabilities found!")
		return err
	}
	fmt.Println("✅ No known vulnerabilities found!")
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

// Install builds and installs the extension locally using azd x build.
// Requires azd to be installed and available in PATH.
func Install() error {
	// Check if azd is available
	if _, err := sh.Output("azd", "version"); err != nil {
		return fmt.Errorf("azd is not installed or not in PATH. Install from https://aka.ms/azd")
	}

	// Get version
	version, err := getVersion()
	if err != nil {
		return err
	}

	fmt.Println("Installing locally...")

	// Set environment variables
	env := map[string]string{
		"EXTENSION_ID":      extensionID,
		"EXTENSION_VERSION": version,
	}

	// azd x build automatically installs unless --skip-install is passed
	if err := sh.RunWithV(env, "azd", "x", "build"); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Printf("✅ Installed version: %s\n", version)
	return nil
}

// Watch monitors files and rebuilds/reinstalls on changes using azd x watch.
// Requires azd to be installed and available in PATH.
func Watch() error {
	// Check if azd is available
	if _, err := sh.Output("azd", "version"); err != nil {
		return fmt.Errorf("azd is not installed or not in PATH. Install from https://aka.ms/azd")
	}

	fmt.Println("Starting file watcher with azd x watch...")

	// Set environment variables
	env := map[string]string{
		"EXTENSION_ID": extensionID,
	}

	return sh.RunWithV(env, "azd", "x", "watch")
}

// Uninstall removes the locally installed extension.
func Uninstall() error {
	fmt.Println("Uninstalling extension...")
	if err := sh.RunV("pwsh", "-File", "scripts/install.ps1", "-Uninstall"); err != nil {
		return fmt.Errorf("failed to uninstall extension: %w", err)
	}

	fmt.Println("✅ Extension uninstalled!")
	return nil
}

// Preflight runs all checks before shipping: format, build, lint, security, tests, and coverage.
func Preflight() error {
	fmt.Println("🚀 Running preflight checks...")
	fmt.Println()

	checks := []struct {
		name string
		fn   func() error
	}{
		{"Formatting code", Fmt},
		{"Verifying go.mod consistency", ModVerify},
		{"Tidying go.mod and go.sum", ModTidy},
		{"Building and linting dashboard", DashboardBuild},
		{"Running dashboard tests", DashboardTest},
		{"Building Go binary", Build},
		{"Running go vet", Vet},
		{"Running staticcheck", Staticcheck},
		{"Running standard linting", Lint},
		{"Running quick security scan", runQuickSecurity},
		{"Checking for known vulnerabilities", runVulncheck},
		{"Running all tests with coverage", TestCoverage},
	}

	for i, check := range checks {
		fmt.Printf("📋 Step %d/%d: %s...\n", i+1, len(checks), check.name)
		if err := check.fn(); err != nil {
			return fmt.Errorf("%s failed: %w", check.name, err)
		}
		fmt.Println()
	}

	fmt.Println("✅ All preflight checks passed!")
	fmt.Println("💡 Tips:")
	fmt.Println("   • Run 'mage security' for a full security scan (~4 minutes)")
	fmt.Println("🎉 Ready to ship!")
	return nil
}

// Security runs security scanning with gosec.
func Security() error {
	return runGosec()
}

// runQuickSecurity runs a fast security scan checking only high-severity, high-confidence issues.
func runQuickSecurity() error {
	fmt.Println("Running quick security scan (high severity only)...")
	// Only check HIGH severity and HIGH confidence issues for speed
	// This catches critical security problems without the 4-minute full scan
	if err := sh.RunV("gosec",
		"-tests=false",
		"-exclude-generated",
		"-severity=high",
		"-confidence=high",
		"-quiet",
		"./src/...",
	); err != nil {
		fmt.Println("⚠️  Quick security scan found HIGH severity issues!")
		fmt.Println("    Run 'mage security' for a full scan")
		return err
	}
	fmt.Println("✅ Quick security scan passed!")
	return nil
}

// runGosec runs security scanning with gosec.
func runGosec() error {
	fmt.Println("Running security scan...")
	// Use -tests=false to skip test files (major speed improvement)
	// Use -exclude-generated to skip generated code
	// Use -fmt=text for faster scanning (skip JSON formatting overhead)
	// Use -concurrency to parallelize (defaults to number of CPUs)
	// Only check specific high-priority rules to speed up scanning
	if err := sh.RunV("gosec",
		"-tests=false",
		"-exclude-generated",
		"-fmt=text",
		"-exclude=G304,G307", // Exclude file paths and deferred error checks (we handle these)
		"-nosec",             // Respect #nosec comments
		"./src/...",          // Only scan src directory
	); err != nil {
		fmt.Println("⚠️  Security scan failed. Ensure gosec is installed:")
		fmt.Println("    go install github.com/securego/gosec/v2/cmd/gosec@latest")
		return err
	}
	fmt.Println("✅ Security scan passed!")
	return nil
}

// DashboardBuild builds the dashboard TypeScript/React code.
func DashboardBuild() error {
	fmt.Println("Building dashboard...")

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

// DashboardTest runs the dashboard tests with vitest.
func DashboardTest() error {
	fmt.Println("Running dashboard tests...")

	// Run tests
	if err := sh.RunV("npm", "test", "--prefix", dashboardDir); err != nil {
		return fmt.Errorf("dashboard tests failed: %w", err)
	}

	fmt.Println("✅ Dashboard tests passed!")
	return nil
}

// DashboardDev runs the dashboard in development mode with hot reload.
func DashboardDev() error {
	fmt.Println("Starting dashboard development server...")
	return sh.RunV("npm", "run", "dev", "--prefix", dashboardDir)
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
	defer func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to restore directory: %v\n", chdirErr)
		}
	}()

	if err := os.Chdir(projectDir); err != nil {
		return fmt.Errorf("failed to change to project directory: %w", err)
	}

	fmt.Printf("🚀 Running in %s: %s %s\n\n", projectDir, binaryPath, command)

	// Get absolute binary path since we changed directories
	absBinaryPath := filepath.Join(originalDir, binaryPath)
	return sh.RunV(absBinaryPath, command)
}
