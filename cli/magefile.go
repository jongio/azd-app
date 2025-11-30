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
// Set ALL_PLATFORMS=true to build for all platforms instead of current platform.
func All() error {
	mg.Deps(Fmt, DashboardBuild, Lint, Test)
	return Build()
}

// Build builds the dashboard and CLI binary for the current platform.
// Set ALL_PLATFORMS=true to build for all platforms instead of current platform.
func Build() error {
	mg.Deps(DashboardBuild)

	if os.Getenv("ALL_PLATFORMS") == "true" {
		return buildAllPlatforms()
	}
	return buildCurrentPlatform()
}

// buildCurrentPlatform compiles the CLI binary for the current platform.
func buildCurrentPlatform() error {
	fmt.Println("Building CLI for current platform...")

	version, err := getVersion()
	if err != nil {
		return err
	}

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	platform := fmt.Sprintf("%s/%s", goos, goarch)

	env := map[string]string{
		"EXTENSION_ID":       extensionID,
		"EXTENSION_VERSION":  version,
		"EXTENSION_PLATFORM": platform,
	}

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

	fmt.Printf("✅ Build complete! Version: %s, Platform: %s\n", version, platform)
	return nil
}

// buildAllPlatforms compiles the CLI binary for all platforms.
func buildAllPlatforms() error {
	fmt.Println("Building CLI for all platforms...")

	version, err := getVersion()
	if err != nil {
		return err
	}

	// When EXTENSION_PLATFORM is not set, the script builds for all platforms
	env := map[string]string{
		"EXTENSION_ID":      extensionID,
		"EXTENSION_VERSION": version,
	}

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

	fmt.Printf("✅ Build complete for all platforms! Version: %s\n", version)
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

// TestE2E runs end-to-end integration tests for the health command.
func TestE2E() error {
	fmt.Println("Running E2E integration tests...")

	timeout := os.Getenv("TEST_TIMEOUT")
	if timeout == "" {
		timeout = "15m"
	}

	args := []string{
		"test",
		"-v",
		"-tags=integration",
		"-timeout=" + timeout,
		"./src/cmd/app/commands",
		"-run=TestHealthCommandE2E",
	}

	return sh.RunV("go", args...)
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

// CheckDeps checks for outdated Go modules and pnpm packages.
// It warns about available updates but does not fail the build.
func CheckDeps() error {
	fmt.Println("Checking for outdated dependencies...")
	fmt.Println()

	hasIssues := false

	// Check Go module updates
	fmt.Println("📦 Checking Go modules for updates...")
	goOutput, err := sh.Output("go", "list", "-u", "-m", "-f", "{{if .Update}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}", "all")
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to check Go module updates: %v\n", err)
	} else {
		// Filter out empty lines
		var updates []string
		for _, line := range strings.Split(goOutput, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				updates = append(updates, line)
			}
		}

		if len(updates) > 0 {
			fmt.Println("   Available Go module updates:")
			for _, update := range updates {
				fmt.Printf("   • %s\n", update)
			}
			hasIssues = true
		} else {
			fmt.Println("   ✅ All Go modules are up to date!")
		}
	}
	fmt.Println()

	// Check for deprecated Go modules
	fmt.Println("🔍 Checking Go modules for deprecation notices...")
	deprecatedOutput, err := sh.Output("go", "list", "-u", "-m", "-f", "{{if .Deprecated}}{{.Path}}: DEPRECATED - {{.Deprecated}}{{end}}", "all")
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to check for deprecated Go modules: %v\n", err)
	} else {
		// Filter out empty lines
		var deprecated []string
		for _, line := range strings.Split(deprecatedOutput, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				deprecated = append(deprecated, line)
			}
		}

		if len(deprecated) > 0 {
			fmt.Println("   ⚠️  Deprecated Go modules found:")
			for _, dep := range deprecated {
				fmt.Printf("   • %s\n", dep)
			}
			hasIssues = true
		} else {
			fmt.Println("   ✅ No deprecated Go modules found!")
		}
	}
	fmt.Println()

	// Check pnpm package updates for dashboard
	fmt.Println("📦 Checking dashboard pnpm packages for updates...")
	// pnpm outdated returns exit code 1 when there are outdated packages, so we capture output differently
	pnpmOutput, _ := sh.Output("pnpm", "outdated", "--dir", dashboardDir)
	if pnpmOutput != "" {
		fmt.Println("   Available pnpm package updates:")
		fmt.Println("   " + strings.ReplaceAll(pnpmOutput, "\n", "\n   "))
		hasIssues = true
	} else {
		fmt.Println("   ✅ All pnpm packages are up to date!")
	}
	fmt.Println()

	// Check for pnpm audit vulnerabilities
	fmt.Println("🔒 Checking dashboard pnpm packages for security vulnerabilities...")
	auditOutput, auditErr := sh.Output("pnpm", "audit", "--dir", dashboardDir, "--json")
	if auditErr != nil {
		// pnpm audit exits with non-zero when vulnerabilities found
		// Parse the JSON to get a summary
		if strings.Contains(auditOutput, "\"vulnerabilities\"") {
			fmt.Println("   ⚠️  Security vulnerabilities found in pnpm packages!")
			fmt.Println("   Run 'pnpm audit --dir dashboard' for details")
			fmt.Println("   Run 'pnpm audit --fix --dir dashboard' to fix automatically")
			hasIssues = true
		}
	} else {
		fmt.Println("   ✅ No known pnpm security vulnerabilities!")
	}
	fmt.Println()

	if hasIssues {
		fmt.Println("💡 Tip: Run 'go get -u ./...' to update Go modules")
		fmt.Println("💡 Tip: Run 'pnpm update --dir dashboard' to update pnpm packages")
		fmt.Println("⚠️  Dependency updates available (continuing with preflight)")
	} else {
		fmt.Println("✅ All dependencies are up to date!")
	}

	// Don't fail the build - just warn
	return nil
}

// Install builds and installs the extension locally using azd x build.
// Requires azd to be installed and available in PATH.
func Install() error {
	// Ensure azd extensions are set up
	if err := ensureAzdExtensions(); err != nil {
		return err
	}

	// Get version
	version, err := getVersion()
	if err != nil {
		return err
	}

	fmt.Println("Building extension...")

	// Set environment variables
	env := map[string]string{
		"EXTENSION_ID":      extensionID,
		"EXTENSION_VERSION": version,
	}

	// Build the extension (skip install - we'll do it via extension install for proper registration)
	if err := sh.RunWithV(env, "azd", "x", "build", "--skip-install"); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Get the absolute path to registry.json (cross-platform)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	registryPath := filepath.Join(filepath.Dir(cwd), "registry.json")

	// Check if registry.json exists, if not use the one in the current directory's parent
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		// Try current directory
		registryPath = filepath.Join(cwd, "..", "registry.json")
		registryPath, _ = filepath.Abs(registryPath)
	}

	// Ensure the local extension source exists
	if err := ensureLocalExtensionSource(registryPath); err != nil {
		return err
	}

	// Install the extension from the local source
	fmt.Println("📦 Installing extension from local registry...")
	if err := sh.RunV("azd", "extension", "install", extensionID, "--source", "local", "--force"); err != nil {
		return fmt.Errorf("extension install failed: %w", err)
	}

	fmt.Printf("✅ Installed version: %s\n", version)
	fmt.Println("   Run 'azd app version' to verify")
	return nil
}

// ensureLocalExtensionSource adds a local extension source if it doesn't exist.
func ensureLocalExtensionSource(registryPath string) error {
	// Check if local source already exists
	sourcesOutput, err := sh.Output("azd", "extension", "source", "list")
	if err != nil {
		sourcesOutput = ""
	}

	// Check if "local" source already exists
	if strings.Contains(sourcesOutput, "local") {
		fmt.Println("✅ Local extension source already configured")
		return nil
	}

	fmt.Println("📦 Adding local extension source...")

	// Add the local source
	if err := sh.RunV("azd", "extension", "source", "add", "-n", "local", "-t", "file", "-l", registryPath); err != nil {
		return fmt.Errorf("failed to add local extension source: %w", err)
	}

	return nil
}

// ensureAzdExtensions checks that azd is installed, extensions are enabled, and the azd x extension is installed.
// This is a prerequisite for commands that use azd x (build, watch, etc.).
func ensureAzdExtensions() error {
	// Check if azd is available
	if _, err := sh.Output("azd", "version"); err != nil {
		return fmt.Errorf("azd is not installed or not in PATH. Install from https://aka.ms/azd")
	}

	// Check if extensions are enabled by looking at config
	configOutput, err := sh.Output("azd", "config", "show")
	if err != nil {
		// Config might not exist yet, that's okay
		configOutput = ""
	}

	// Enable extensions if not already enabled
	if !strings.Contains(configOutput, `"enabled": "on"`) && !strings.Contains(configOutput, `"enabled":"on"`) {
		fmt.Println("📦 Enabling azd extensions...")
		if err := sh.RunV("azd", "config", "set", "alpha.extension.enabled", "on"); err != nil {
			return fmt.Errorf("failed to enable azd extensions: %w", err)
		}
		fmt.Println("✅ Extensions enabled!")
	}

	// Check if azd x extension is available
	if _, err := sh.Output("azd", "x", "--help"); err != nil {
		fmt.Println("📦 Installing azd x extension (developer kit)...")
		if err := sh.RunV("azd", "extension", "install", "microsoft.azd.extensions", "--source", "azd"); err != nil {
			return fmt.Errorf("failed to install azd x extension: %w", err)
		}
		fmt.Println("✅ azd x extension installed!")
	}

	return nil
}

// Watch monitors files and rebuilds/reinstalls on changes using azd x watch.
// Requires azd to be installed and available in PATH.
func Watch() error {
	// Ensure azd extensions are set up
	if err := ensureAzdExtensions(); err != nil {
		return err
	}

	fmt.Println("Starting file watcher with azd x watch...")

	// Set environment variables
	env := map[string]string{
		"EXTENSION_ID": extensionID,
	}

	return sh.RunWithV(env, "azd", "x", "watch")
}

// WatchAll monitors both CLI and dashboard files, rebuilding on changes.
// Runs azd x watch for CLI and vite dev server for dashboard concurrently.
// Note: On Windows, stop any running instances of the app before starting the watcher
// to avoid "file in use" errors during installation.
func WatchAll() error {
	fmt.Println("Starting watchers for both CLI and dashboard...")
	fmt.Println("⚠️  Tip: Stop any running instances of 'app' to avoid file-in-use errors")
	fmt.Println()

	// Ensure azd extensions are set up (enables extensions + installs azd x if needed)
	if err := ensureAzdExtensions(); err != nil {
		return err
	}

	// Install dashboard dependencies before starting watcher
	fmt.Println("📦 Installing dashboard dependencies...")
	if err := sh.RunV("pnpm", "install", "--dir", dashboardDir); err != nil {
		return fmt.Errorf("pnpm install failed: %w", err)
	}
	fmt.Println()

	// Create channels for error handling
	errChan := make(chan error, 2)

	// Start CLI watcher in goroutine
	go func() {
		fmt.Println("🔧 Starting CLI watcher (azd x watch)...")
		env := map[string]string{
			"EXTENSION_ID": extensionID,
		}
		if err := sh.RunWithV(env, "azd", "x", "watch"); err != nil {
			errChan <- fmt.Errorf("CLI watcher failed: %w", err)
		}
	}()

	// Start dashboard watcher in goroutine
	go func() {
		fmt.Println("⚛️  Starting dashboard watcher (vite dev server)...")
		if err := sh.RunV("pnpm", "--dir", dashboardDir, "run", "dev"); err != nil {
			errChan <- fmt.Errorf("dashboard watcher failed: %w", err)
		}
	}()

	// Wait for either watcher to fail
	return <-errChan
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

// CheckGitAttributes ensures .gitattributes file exists with proper line ending configuration.
func CheckGitAttributes() error {
	fmt.Println("Checking .gitattributes...")

	gitattributesPath := filepath.Join("..", ".gitattributes")
	if _, err := os.Stat(gitattributesPath); os.IsNotExist(err) {
		return fmt.Errorf(".gitattributes file not found - required for proper line ending configuration")
	}

	fmt.Println("✅ .gitattributes exists!")
	return nil
}

// CheckGitIgnore ensures .gitignore file exists.
func CheckGitIgnore() error {
	fmt.Println("Checking .gitignore...")

	gitignorePath := filepath.Join("..", ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		return fmt.Errorf(".gitignore file not found")
	}

	fmt.Println("✅ .gitignore exists!")
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
		{"Checking .gitignore", CheckGitIgnore},
		{"Checking .gitattributes", CheckGitAttributes},
		{"Checking for outdated dependencies", CheckDeps},
		{"Formatting code", Fmt},
		{"Verifying go.mod consistency", ModVerify},
		{"Tidying go.mod and go.sum", ModTidy},
		{"Building dashboard", DashboardBuild},
		{"Linting dashboard", DashboardLint},
		{"Running dashboard unit tests", DashboardTest},
		{"Running dashboard E2E tests", DashboardTestE2E},
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
	if err := sh.RunV("pnpm", "install", "--dir", dashboardDir); err != nil {
		return fmt.Errorf("pnpm install failed: %w", err)
	}

	// Run TypeScript compilation and build
	fmt.Println("Building dashboard assets...")
	if err := sh.RunV("pnpm", "--dir", dashboardDir, "run", "build"); err != nil {
		return fmt.Errorf("dashboard build failed: %w", err)
	}

	fmt.Println("✅ Dashboard build complete!")
	return nil
}

// DashboardTest runs the dashboard tests with vitest.
func DashboardTest() error {
	fmt.Println("Running dashboard tests...")

	// Run tests
	if err := sh.RunV("pnpm", "--dir", dashboardDir, "test"); err != nil {
		return fmt.Errorf("dashboard tests failed: %w", err)
	}

	fmt.Println("✅ Dashboard tests passed!")
	return nil
}

// DashboardLint runs ESLint on the dashboard code.
func DashboardLint() error {
	fmt.Println("Running dashboard linting...")

	if err := sh.RunV("pnpm", "--dir", dashboardDir, "run", "lint"); err != nil {
		return fmt.Errorf("dashboard linting failed: %w", err)
	}

	fmt.Println("✅ Dashboard linting passed!")
	return nil
}

// DashboardTestE2E runs the dashboard E2E tests with Playwright.
func DashboardTestE2E() error {
	fmt.Println("Running dashboard E2E tests...")

	// Ensure Playwright browsers are installed
	fmt.Println("Installing Playwright browsers (if needed)...")
	// Change to dashboard directory to run playwright install
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to restore directory: %v\n", chdirErr)
		}
	}()

	if err := os.Chdir(dashboardDir); err != nil {
		return fmt.Errorf("failed to change to dashboard directory: %w", err)
	}

	if err := sh.RunV("npx", "playwright", "install", "--with-deps", "chromium"); err != nil {
		fmt.Println("⚠️  Failed to install Playwright browsers - continuing anyway...")
	}

	// Run playwright with line reporter to avoid opening browser with HTML report on failure
	// Stay in dashboard directory where playwright.config.ts is located
	if err := sh.RunV("npx", "playwright", "test", "--reporter=line", "--project=chromium"); err != nil {
		return fmt.Errorf("dashboard E2E tests failed: %w", err)
	}

	fmt.Println("✅ Dashboard E2E tests passed!")
	return nil
}

// DashboardDev runs the dashboard in development mode with hot reload.
func DashboardDev() error {
	fmt.Println("Starting dashboard development server...")
	return sh.RunV("pnpm", "--dir", dashboardDir, "run", "dev")
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
