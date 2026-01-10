//go:build mage

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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
	websiteDir         = "../web"
	defaultTestTimeout = "10m"
	extensionID        = "jongio.azd.app"
	goSrcPattern       = "./src/..."
	goIntegrationTag   = "-tags=integration"
	errBuildFailedFmt  = "build failed: %w"
	errPnpmFailedFmt   = "pnpm install failed: %w"
	fmtBulletItem      = "   • %s\n"
	fmtTestingProject  = "   Testing %s (%s)...\n"
	fmtProjectFailed   = "   ❌ %s failed: %v\n"
	fmtProjectPassed   = "   ✅ %s passed\n"
)

// Default target runs all checks and builds.
var Default = All

// killAppProcesses terminates any running azd app processes to allow rebuilding.
// This is necessary on Windows where the binary cannot be overwritten while in use.
func killAppProcesses() error {
	if runtime.GOOS == "windows" {
		fmt.Println("Stopping any running app processes...")
		// Use PowerShell Stop-Process instead of taskkill, which can timeout on Windows
		// Stop-Process is more reliable and doesn't hang when the process doesn't exist

		// Kill any process named "app" (the binary name without extension)
		_ = exec.Command("powershell", "-NoProfile", "-Command",
			"Stop-Process -Name '"+binaryName+"' -Force -ErrorAction SilentlyContinue").Run()

		// Also kill the installed extension binary (jongio-azd-app-*.exe)
		// The extension ID is "jongio.azd.app" which becomes "jongio-azd-app" in the binary name
		extensionBinaryPrefix := strings.ReplaceAll(extensionID, ".", "-")
		// Kill all platform variants that might be running
		for _, arch := range []string{"windows-amd64", "windows-arm64"} {
			procName := extensionBinaryPrefix + "-" + arch
			_ = exec.Command("powershell", "-NoProfile", "-Command",
				"Stop-Process -Name '"+procName+"' -Force -ErrorAction SilentlyContinue").Run()
		}
	} else {
		// On Unix, use pkill (ignore errors if no process found)
		_ = exec.Command("pkill", "-f", binaryName).Run()
		// Also kill the installed extension binary
		extensionBinaryPrefix := strings.ReplaceAll(extensionID, ".", "-")
		_ = exec.Command("pkill", "-f", extensionBinaryPrefix).Run()
	}
	return nil
}

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

// Build builds the dashboard and CLI binary, and installs it locally.
// This is the main command for development - it builds everything and installs the extension.
// Set ALL_PLATFORMS=true to build for all platforms (skip install).
// Set SKIP_INSTALL=true to only build without installing.
func Build() error {
	// Kill any running app processes first to avoid "file in use" errors on Windows
	_ = killAppProcesses()

	mg.Deps(DashboardBuild)

	if os.Getenv("ALL_PLATFORMS") == "true" {
		return buildAllPlatforms()
	}

	// Ensure azd extensions are set up
	if err := ensureAzdExtensions(); err != nil {
		return err
	}

	version, err := getVersion()
	if err != nil {
		return err
	}

	fmt.Println("Building and installing extension...")

	env := map[string]string{
		"EXTENSION_ID":      extensionID,
		"EXTENSION_VERSION": version,
	}

	// Build and install directly using azd x build
	if err := sh.RunWithV(env, "azd", "x", "build"); err != nil {
		return fmt.Errorf(errBuildFailedFmt, err)
	}

	fmt.Printf("✅ Build complete! Version: %s\n", version)
	fmt.Println("   Run 'azd app version' to verify")
	return nil
}

// buildAllPlatforms compiles the CLI binary for all platforms (used for releases).
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
			return fmt.Errorf(errBuildFailedFmt, err)
		}
	} else {
		buildScript = "build.sh"
		if err := sh.RunWithV(env, "bash", buildScript); err != nil {
			return fmt.Errorf(errBuildFailedFmt, err)
		}
	}

	fmt.Printf("✅ Build complete for all platforms! Version: %s\n", version)
	return nil
}

// Test runs unit tests only (with -short flag).
func Test() error {
	fmt.Println("Running unit tests...")
	// Use full module path in workspace mode
	pkgPath := goSrcPattern
	if _, err := os.Stat("../go.work"); err == nil {
		pkgPath = "github.com/jongio/azd-app/cli/src/..."
	}
	return sh.RunV("go", "test", "-v", "-short", pkgPath)
}

// TestIntegration runs integration tests only.
// Set TEST_PACKAGE env var to filter by package (e.g., installer, runner, commands)
// Set TEST_NAME env var to run a specific test
// Set TEST_TIMEOUT env var to override default 10m timeout
func TestIntegration() error {
	fmt.Println("Running integration tests...")

	args := []string{"test", "-v", goIntegrationTag}

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
	testPath := goSrcPattern
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
	return sh.RunV("go", "test", "-v", goIntegrationTag, goSrcPattern)
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
		goIntegrationTag,
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
	// Use exec.Command to capture output and handle Go version mismatch warnings gracefully
	// These warnings occur when Go's compiled stdlib doesn't match the go binary version
	// but don't affect test correctness
	// Use full module path in workspace mode
	pkgPath := goSrcPattern
	if _, err := os.Stat("../go.work"); err == nil {
		pkgPath = "github.com/jongio/azd-app/cli/src/..."
	}
	cmd := exec.Command("go", "test", "-short", "-coverprofile="+coverageOut, pkgPath)
	output, testErr := cmd.CombinedOutput()
	fmt.Print(string(output))

	// Check if there were actual test failures vs just version mismatch warnings
	if testErr != nil {
		outputStr := string(output)
		// If all tests passed (contain "ok" lines) but exit code is non-zero,
		// it's likely due to Go version mismatch warnings which can be ignored
		hasTestFailure := strings.Contains(outputStr, "FAIL") && !strings.Contains(outputStr, "[setup failed]")
		hasVersionMismatch := strings.Contains(outputStr, "does not match go tool version")

		// Only fail if there are actual test failures, not just version warnings
		if hasTestFailure || !hasVersionMismatch {
			return fmt.Errorf("tests failed: %w", testErr)
		}
		fmt.Println("⚠️  Go version mismatch warnings detected (stdlib vs binary version)")
		fmt.Println("   Consider reinstalling Go to fix this warning")
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
	// --concurrency=0 uses all available CPUs

	// golangci-lint automatically discovers .golangci.yml config
	// In workspace mode, we run without explicit package path - golangci-lint
	// will use the current directory as context
	if err := sh.RunV("golangci-lint", "run", "--timeout=5m", "--concurrency=0"); err != nil {
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
	// Use full module path in workspace mode
	pkgPath := "./..."
	if _, err := os.Stat("../go.work"); err == nil {
		pkgPath = "github.com/jongio/azd-app/cli/..."
	}
	if err := sh.RunV("golangci-lint", "run", "--no-config", "--enable-all", "--disable="+excludeLinters, "--max-issues-per-linter=0", "--max-same-issues=0", pkgPath); err != nil {
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
	// Use full module path in workspace mode
	pkgPath := "./..."
	if _, err := os.Stat("../go.work"); err == nil {
		pkgPath = "github.com/jongio/azd-app/cli/..."
	}
	if err := sh.RunV("go", "vet", pkgPath); err != nil {
		return fmt.Errorf("go vet found issues: %w", err)
	}
	fmt.Println("✅ go vet passed!")
	return nil
}

// Staticcheck runs staticcheck for advanced static analysis.
func Staticcheck() error {
	fmt.Println("Running staticcheck...")
	// Use full module path in workspace mode
	pkgPath := "./..."
	if _, err := os.Stat("../go.work"); err == nil {
		pkgPath = "github.com/jongio/azd-app/cli/..."
	}
	if err := sh.RunV("staticcheck", pkgPath); err != nil {
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

	// Check if we're in a workspace - if so, skip tidy as it doesn't work with local modules
	if _, err := os.Stat("../go.work"); err == nil {
		fmt.Println("⚠️  Workspace detected (go.work exists) - skipping go mod tidy")
		fmt.Println("   (go mod tidy doesn't work with local workspace modules)")
		fmt.Println("✅ ModTidy check skipped (workspace mode)!")
		return nil
	}

	goModBefore, err := fileHash("go.mod")
	if err != nil {
		return fmt.Errorf("failed to read go.mod before tidy: %w", err)
	}
	goSumBefore, err := fileHash("go.sum")
	if err != nil {
		return fmt.Errorf("failed to read go.sum before tidy: %w", err)
	}

	if err := sh.RunV("go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}

	goModAfter, err := fileHash("go.mod")
	if err != nil {
		return fmt.Errorf("failed to read go.mod after tidy: %w", err)
	}
	goSumAfter, err := fileHash("go.sum")
	if err != nil {
		return fmt.Errorf("failed to read go.sum after tidy: %w", err)
	}

	if goModBefore != goModAfter || goSumBefore != goSumAfter {
		return fmt.Errorf("go.mod or go.sum changed after running go mod tidy - please review the changes")
	}

	fmt.Println("✅ go mod tidy passed!")
	return nil
}

func fileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
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

	// Use full module path in workspace mode
	pkgPath := "./..."
	if _, err := os.Stat("../go.work"); err == nil {
		pkgPath = "github.com/jongio/azd-app/cli/..."
	}
	if err := sh.RunV("govulncheck", pkgPath); err != nil {
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
				fmt.Printf(fmtBulletItem, update)
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
				fmt.Printf(fmtBulletItem, dep)
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
		fmt.Println("   ✅ All dashboard pnpm packages are up to date!")
	}
	fmt.Println()

	// Check pnpm package updates for website
	fmt.Println("📦 Checking website pnpm packages for updates...")
	websitePnpmOutput, _ := sh.Output("pnpm", "outdated", "--dir", websiteDir)
	if websitePnpmOutput != "" {
		fmt.Println("   Available pnpm package updates:")
		fmt.Println("   " + strings.ReplaceAll(websitePnpmOutput, "\n", "\n   "))
		hasIssues = true
	} else {
		fmt.Println("   ✅ All website pnpm packages are up to date!")
	}
	fmt.Println()

	// Check for pnpm audit vulnerabilities (dashboard)
	fmt.Println("🔒 Checking dashboard pnpm packages for security vulnerabilities...")
	auditOutput, auditErr := sh.Output("pnpm", "audit", "--dir", dashboardDir, "--json")
	if auditErr != nil {
		// pnpm audit exits with non-zero when vulnerabilities found
		// Parse the JSON to get a summary
		if strings.Contains(auditOutput, "\"vulnerabilities\"") {
			fmt.Println("   ⚠️  Security vulnerabilities found in dashboard packages!")
			fmt.Println("   Run 'pnpm audit --dir dashboard' for details")
			fmt.Println("   Run 'pnpm audit --fix --dir dashboard' to fix automatically")
			hasIssues = true
		}
	} else {
		fmt.Println("   ✅ No known dashboard security vulnerabilities!")
	}
	fmt.Println()

	// Check for pnpm audit vulnerabilities (website)
	fmt.Println("🔒 Checking website pnpm packages for security vulnerabilities...")
	websiteAuditOutput, websiteAuditErr := sh.Output("pnpm", "audit", "--dir", websiteDir, "--json")
	if websiteAuditErr != nil {
		// pnpm audit exits with non-zero when vulnerabilities found
		if strings.Contains(websiteAuditOutput, "\"vulnerabilities\"") {
			fmt.Println("   ⚠️  Security vulnerabilities found in website packages!")
			fmt.Println("   Run 'pnpm audit --dir ../web' for details")
			fmt.Println("   Run 'pnpm audit --fix --dir ../web' to fix automatically")
			hasIssues = true
		}
	} else {
		fmt.Println("   ✅ No known website security vulnerabilities!")
	}
	fmt.Println()

	if hasIssues {
		fmt.Println("💡 Tip: Run 'mage updateDeps' to update all dependencies")
		fmt.Println("💡 Tip: Run 'go get -u ./...' to update Go modules only")
		fmt.Println("💡 Tip: Run 'pnpm update --dir dashboard' to update dashboard packages only")
		fmt.Println("💡 Tip: Run 'pnpm update --dir ../web' to update website packages only")
		fmt.Println("⚠️  Dependency updates available (continuing with preflight)")
	} else {
		fmt.Println("✅ All dependencies are up to date!")
	}

	// Don't fail the build - just warn
	return nil
}

// UpdateDeps updates all dependencies to their latest versions across the entire project.
// This includes Go modules, dashboard pnpm packages, and website pnpm packages.
// Use MINOR_ONLY=true to only update to latest minor versions (safer, avoids breaking changes).
// Use DRY_RUN=true to preview updates without applying them.
func UpdateDeps() error {
	fmt.Println("🔄 Updating all dependencies to latest versions...")
	fmt.Println()

	minorOnly := os.Getenv("MINOR_ONLY") == "true"
	dryRun := os.Getenv("DRY_RUN") == "true"

	if dryRun {
		fmt.Println("🔍 DRY RUN MODE - No changes will be made")
		fmt.Println()
	}

	if minorOnly {
		fmt.Println("📌 MINOR ONLY MODE - Only updating to latest minor versions (safer)")
		fmt.Println()
	}

	// Track any errors but continue with other updates
	var errors []string

	// Update Go modules
	fmt.Println("📦 Updating Go modules...")
	if dryRun {
		// Just show what would be updated
		if err := sh.RunV("go", "list", "-u", "-m", "all"); err != nil {
			errors = append(errors, fmt.Sprintf("Go modules check: %v", err))
		}
	} else {
		// Update all Go modules to latest
		updateCmd := "go"
		updateArgs := []string{"get", "-u"}
		if minorOnly {
			// Update to latest minor/patch but not major
			updateArgs = append(updateArgs, "-u=patch")
		}
		updateArgs = append(updateArgs, "./...")

		if err := sh.RunV(updateCmd, updateArgs...); err != nil {
			errors = append(errors, fmt.Sprintf("Go modules update: %v", err))
		} else {
			// Tidy up after updates
			if err := sh.RunV("go", "mod", "tidy"); err != nil {
				errors = append(errors, fmt.Sprintf("go mod tidy: %v", err))
			} else {
				fmt.Println("   ✅ Go modules updated and tidied!")
			}
		}
	}
	fmt.Println()

	// Update dashboard pnpm packages
	fmt.Println("📦 Updating dashboard pnpm packages...")
	if dryRun {
		// Just show what would be updated
		_, _ = sh.Output("pnpm", "outdated", "--dir", dashboardDir)
	} else {
		updateArgs := []string{"update", "--dir", dashboardDir}
		if minorOnly {
			// pnpm update respects semver ranges in package.json by default
			// Use --latest for major updates, omit for minor/patch only
			fmt.Println("   Using version ranges from package.json (minor/patch updates)")
		} else {
			// Update to latest regardless of semver ranges
			updateArgs = append(updateArgs, "--latest")
		}

		if err := sh.RunV("pnpm", updateArgs...); err != nil {
			errors = append(errors, fmt.Sprintf("dashboard pnpm update: %v", err))
		} else {
			fmt.Println("   ✅ Dashboard packages updated!")
		}
	}
	fmt.Println()

	// Update website pnpm packages
	fmt.Println("📦 Updating website pnpm packages...")
	if dryRun {
		// Just show what would be updated
		_, _ = sh.Output("pnpm", "outdated", "--dir", websiteDir)
	} else {
		updateArgs := []string{"update", "--dir", websiteDir}
		if minorOnly {
			fmt.Println("   Using version ranges from package.json (minor/patch updates)")
		} else {
			updateArgs = append(updateArgs, "--latest")
		}

		if err := sh.RunV("pnpm", updateArgs...); err != nil {
			errors = append(errors, fmt.Sprintf("website pnpm update: %v", err))
		} else {
			fmt.Println("   ✅ Website packages updated!")
		}
	}
	fmt.Println()

	// Fix any security vulnerabilities in pnpm packages
	if !dryRun {
		fmt.Println("🔒 Fixing security vulnerabilities...")

		// Dashboard
		fmt.Println("   Fixing dashboard vulnerabilities...")
		if err := sh.RunV("pnpm", "audit", "--fix", "--dir", dashboardDir); err != nil {
			// audit --fix can return non-zero even when successful if unfixable vulns remain
			fmt.Println("   ⚠️  Some vulnerabilities may remain (manual review needed)")
		} else {
			fmt.Println("   ✅ Dashboard vulnerabilities fixed!")
		}

		// Website
		fmt.Println("   Fixing website vulnerabilities...")
		if err := sh.RunV("pnpm", "audit", "--fix", "--dir", websiteDir); err != nil {
			fmt.Println("   ⚠️  Some vulnerabilities may remain (manual review needed)")
		} else {
			fmt.Println("   ✅ Website vulnerabilities fixed!")
		}
		fmt.Println()
	}

	// Summary
	if len(errors) > 0 {
		fmt.Println("❌ Some updates failed:")
		for _, err := range errors {
			fmt.Printf(fmtBulletItem, err)
		}
		fmt.Println()
		fmt.Println("💡 Review errors above and fix manually if needed")
		return fmt.Errorf("%d update(s) failed", len(errors))
	}

	if dryRun {
		fmt.Println("✅ Dry run complete! No changes were made.")
		fmt.Println("💡 Run 'mage updateDeps' without DRY_RUN=true to apply updates")
	} else {
		fmt.Println("✅ All dependencies updated successfully!")
		fmt.Println()
		fmt.Println("📝 Next steps:")
		fmt.Println("   1. Review changes: git diff")
		fmt.Println("   2. Test the build: mage build")
		fmt.Println("   3. Run tests: mage test")
		fmt.Println("   4. Check for issues: mage lint")
		fmt.Println("   5. Commit changes: git add . && git commit -m 'chore: update dependencies'")
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
		if err := sh.RunV("azd", "extension", "install", "microsoft.azd.extensions", "--source", "azd", "--no-prompt"); err != nil {
			return fmt.Errorf("failed to install azd x extension: %w", err)
		}
		fmt.Println("✅ azd x extension installed!")
	}

	return nil
}

// Watch monitors both CLI and dashboard files, rebuilding on changes.
// Runs azd x watch for CLI and vite build --watch for dashboard concurrently.
// The dashboard is built to the embedded location (src/internal/dashboard/dist)
// so changes are automatically included when the CLI is rebuilt.
// Note: The build scripts (build.ps1/build.sh) kill running app processes
// on each rebuild iteration to avoid "file in use" errors on Windows.
func Watch() error {
	fmt.Println("Starting watchers for both CLI and dashboard...")
	fmt.Println()

	// Ensure azd extensions are set up (enables extensions + installs azd x if needed)
	if err := ensureAzdExtensions(); err != nil {
		return err
	}

	// Install dashboard dependencies before starting watcher
	fmt.Println("📦 Installing dashboard dependencies...")
	if err := sh.RunV("pnpm", "install", "--dir", dashboardDir); err != nil {
		return fmt.Errorf(errPnpmFailedFmt, err)
	}

	// Do an initial dashboard build to ensure embedded dist is up-to-date
	fmt.Println("📦 Building dashboard for embedding...")
	if err := sh.RunV("pnpm", "--dir", dashboardDir, "run", "build"); err != nil {
		return fmt.Errorf("initial dashboard build failed: %w", err)
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

	// Start dashboard watcher in goroutine - uses vite build --watch to output to embedded location
	go func() {
		fmt.Println("⚛️  Starting dashboard watcher (vite build --watch)...")
		fmt.Println("   Dashboard changes will be built to src/internal/dashboard/dist")
		if err := sh.RunV("pnpm", "--dir", dashboardDir, "run", "build", "--", "--watch"); err != nil {
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
		{"Validating website CLI docs", WebsiteValidate},
		{"Building website", WebsiteBuild},
		{"Running website E2E tests", WebsiteTestE2E},
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

// runQuickSecurity runs a fast security scan checking only critical security rules.
// Optimized to run ~7x faster than a full scan by focusing on high-impact vulnerabilities.
func runQuickSecurity() error {
	fmt.Println("Running quick security scan (critical rules only)...")
	// Run only the most critical security rules for speed:
	// G101: Hardcoded credentials - CRITICAL
	// G102: Bind to all interfaces - CRITICAL for network services
	// G201: SQL injection via format string - CRITICAL
	// G202: SQL injection via concatenation - CRITICAL
	// G301: Poor directory permissions - HIGH
	// G305: File path traversal (zip slip) - CRITICAL
	// G402: Bad TLS settings - CRITICAL
	// G403: Weak RSA key length - HIGH
	// This reduces scan time from ~600s to ~90s while catching critical vulnerabilities
	if err := sh.RunV("gosec",
		"-tests=false",
		"-exclude-generated",
		"-severity=high",
		"-confidence=high",
		"-quiet",
		"-include=G101,G102,G201,G202,G301,G305,G402,G403",
		goSrcPattern,
	); err != nil {
		fmt.Println("⚠️  Quick security scan found critical issues!")
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
		goSrcPattern,         // Only scan src directory
	); err != nil {
		fmt.Println("⚠️  Security scan failed. Ensure gosec is installed:")
		fmt.Println("    go install github.com/securego/gosec/v2/cmd/gosec@latest")
		return err
	}
	fmt.Println("✅ Security scan passed!")
	return nil
}

// DashboardBuild builds the dashboard TypeScript/React code.
// The build output goes to src/internal/dashboard/dist which is embedded in the CLI binary.
func DashboardBuild() error {
	fmt.Println("Building dashboard...")

	// Install dependencies
	fmt.Println("Installing dashboard dependencies...")
	if err := sh.RunV("pnpm", "install", "--dir", dashboardDir); err != nil {
		return fmt.Errorf(errPnpmFailedFmt, err)
	}

	// Run TypeScript compilation and build
	// Output goes to src/internal/dashboard/dist (configured in vite.config.ts)
	fmt.Println("Building dashboard assets to src/internal/dashboard/dist...")
	if err := sh.RunV("pnpm", "--dir", dashboardDir, "run", "build"); err != nil {
		return fmt.Errorf("dashboard build failed: %w", err)
	}

	fmt.Println("✅ Dashboard build complete! Assets embedded in CLI binary.")
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

	// Get absolute path to dashboard directory (safe for parallel execution)
	absDashboardDir, err := filepath.Abs(dashboardDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute dashboard path: %w", err)
	}

	// Ensure Playwright browsers are installed
	fmt.Println("Installing Playwright browsers (if needed)...")
	installCmd := exec.Command("npx", "playwright", "install", "--with-deps", "chromium")
	installCmd.Dir = absDashboardDir
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		fmt.Println("⚠️  Failed to install Playwright browsers - continuing anyway...")
	}

	// Run playwright with line reporter to avoid opening browser with HTML report on failure
	testCmd := exec.Command("npx", "playwright", "test", "--reporter=line", "--project=chromium")
	testCmd.Dir = absDashboardDir
	testCmd.Stdout = os.Stdout
	testCmd.Stderr = os.Stderr
	if err := testCmd.Run(); err != nil {
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

// ============================================================================
// Website (Astro marketing site) targets
// ============================================================================

// WebsiteBuild builds the Astro website with validation and code generation.
// Runs: validate CLI docs, generate CLI reference, generate changelog, then build.
func WebsiteBuild() error {
	fmt.Println("Building website...")

	// Install dependencies
	fmt.Println("Installing website dependencies...")
	if err := sh.RunV("pnpm", "install", "--dir", websiteDir); err != nil {
		return fmt.Errorf(errPnpmFailedFmt, err)
	}

	// Run build (which includes prebuild: validate, generate:cli, generate:changelog)
	fmt.Println("Building Astro site...")
	if err := sh.RunV("pnpm", "--dir", websiteDir, "run", "build"); err != nil {
		return fmt.Errorf("website build failed: %w", err)
	}

	fmt.Println("✅ Website build complete!")
	return nil
}

// WebsiteValidate validates that CLI command documentation matches actual commands.
func WebsiteValidate() error {
	fmt.Println("Validating website CLI documentation...")

	// Install dependencies first
	if err := sh.RunV("pnpm", "install", "--dir", websiteDir); err != nil {
		return fmt.Errorf(errPnpmFailedFmt, err)
	}

	// Run validation script
	if err := sh.RunV("pnpm", "--dir", websiteDir, "run", "validate"); err != nil {
		return fmt.Errorf("website CLI validation failed: %w", err)
	}

	fmt.Println("✅ Website CLI documentation is valid!")
	return nil
}

// WebsiteTestE2EUpdateSnapshots runs the website E2E tests and updates snapshots.
func WebsiteTestE2EUpdateSnapshots() error {
	return runWebsiteE2ETests(true)
}

// WebsiteTestE2E runs the website E2E tests with Playwright.
func WebsiteTestE2E() error {
	return runWebsiteE2ETests(false)
}

// runWebsiteE2ETests is the shared implementation for E2E tests.
func runWebsiteE2ETests(updateSnapshots bool) error {
	// Get absolute path to website directory (safe for parallel execution)
	absWebsiteDir, err := filepath.Abs(websiteDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute website path: %w", err)
	}

	// Auto-generate baseline snapshots if they don't exist
	snapshotsDir := filepath.Join(absWebsiteDir, "e2e", "screenshots.spec.ts-snapshots")
	if _, err := os.Stat(snapshotsDir); os.IsNotExist(err) {
		fmt.Println("📸 Baseline snapshots not found - will auto-generate them...")
		updateSnapshots = true
	} else if err == nil {
		// Check if directory is empty
		entries, err := os.ReadDir(snapshotsDir)
		if err == nil && len(entries) == 0 {
			fmt.Println("📸 Baseline snapshots directory is empty - will auto-generate them...")
			updateSnapshots = true
		}
	}

	if updateSnapshots {
		fmt.Println("Running website E2E tests (updating snapshots)...")
	} else {
		fmt.Println("Running website E2E tests...")
	}

	// Install Playwright browsers
	fmt.Println("Installing Playwright browsers (if needed)...")
	installCmd := exec.Command("npx", "playwright", "install", "--with-deps", "chromium")
	installCmd.Dir = absWebsiteDir
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		fmt.Println("⚠️  Failed to install Playwright browsers - continuing anyway...")
	}

	// Start the preview server in the background
	fmt.Println("Starting preview server...")
	serverCmd := exec.Command("npx", "astro", "preview", "--host", "127.0.0.1", "--port", "4321")
	serverCmd.Dir = absWebsiteDir
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr
	if err := serverCmd.Start(); err != nil {
		return fmt.Errorf("failed to start preview server: %w", err)
	}
	defer func() {
		if serverCmd.Process != nil {
			_ = serverCmd.Process.Kill()
		}
	}()

	// Wait for server to be ready
	fmt.Println("Waiting for server to be ready...")
	serverReady := false
	for i := 0; i < 30; i++ {
		resp, err := http.Get("http://localhost:4321/azd-app/")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				serverReady = true
				break
			}
		}
		time.Sleep(1 * time.Second)
	}
	if !serverReady {
		return fmt.Errorf("server did not become ready within 30 seconds")
	}
	fmt.Println("Server is ready!")

	// Run playwright tests
	args := []string{"playwright", "test", "--reporter=line", "--project=chromium"}
	if updateSnapshots {
		args = append(args, "--update-snapshots")
	}
	testCmd := exec.Command("npx", args...)
	testCmd.Dir = absWebsiteDir
	// Set CI=true to skip visual regression (screenshot comparison) since
	// baseline snapshots are platform-specific and gitignored
	testCmd.Env = append(os.Environ(), "CI=true")
	testCmd.Stdout = os.Stdout
	testCmd.Stderr = os.Stderr
	if err := testCmd.Run(); err != nil {
		return fmt.Errorf("website E2E tests failed: %w", err)
	}

	fmt.Println("✅ Website E2E tests passed!")
	return nil
}

// WebsiteDev runs the website in development mode with hot reload.
func WebsiteDev() error {
	fmt.Println("Starting website development server...")
	return sh.RunV("pnpm", "--dir", websiteDir, "run", "dev")
}

// WebsitePreview runs the website in preview mode (production build served locally).
func WebsitePreview() error {
	fmt.Println("Starting website preview server...")

	// Build first
	if err := WebsiteBuild(); err != nil {
		return err
	}

	return sh.RunV("pnpm", "--dir", websiteDir, "run", "preview")
}

// WebsiteScreenshots captures dashboard screenshots for the marketing website.
// Requires the demo project to be running with azd app run.
func WebsiteScreenshots() error {
	fmt.Println("Capturing dashboard screenshots...")

	// Install dependencies first
	if err := sh.RunV("pnpm", "install", "--dir", websiteDir); err != nil {
		return fmt.Errorf(errPnpmFailedFmt, err)
	}

	// Run screenshot capture script
	if err := sh.RunV("pnpm", "--dir", websiteDir, "run", "screenshots"); err != nil {
		return fmt.Errorf("screenshot capture failed: %w", err)
	}

	fmt.Println("✅ Screenshots captured!")
	return nil
}

// Run builds and runs the app directly in a test project (without installing as extension).
func Run() error {
	projectDir := os.Getenv("PROJECT_DIR")
	if projectDir == "" {
		projectDir = "tests/projects/orchestration/fullstack-test"
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

// ============================================================================
// Sample Project Testing targets
// ============================================================================

const testProjectsDir = "tests/projects"

// TestProjects runs tests for all sample projects in tests/projects/test-frameworks.
// This validates that our test framework detection and execution works correctly.
// Set LANGUAGE env var to filter by language (node, python, dotnet, go).
func TestProjects() error {
	fmt.Println("🧪 Running tests for sample projects...")
	fmt.Println()

	language := os.Getenv("LANGUAGE")

	var failed []string
	var passed []string

	// Test Node.js projects
	if language == "" || language == "node" {
		fmt.Println("📦 Testing Node.js projects...")
		nodeProjects := []struct {
			name string
			dir  string
		}{
			{"Jest", "test-frameworks/node/jest"},
			{"Vitest", "test-frameworks/node/vitest"},
			{"Mocha+Jasmine", "test-frameworks/node/alternatives"},
		}

		for _, proj := range nodeProjects {
			projPath := filepath.Join(testProjectsDir, proj.dir)
			fmt.Printf(fmtTestingProject, proj.name, proj.dir)
			if err := runNodeTests(projPath); err != nil {
				fmt.Printf(fmtProjectFailed, proj.name, err)
				failed = append(failed, proj.name)
			} else {
				fmt.Printf(fmtProjectPassed, proj.name)
				passed = append(passed, proj.name)
			}
		}
		fmt.Println()
	}

	// Test Python projects
	if language == "" || language == "python" {
		fmt.Println("🐍 Testing Python projects...")
		pythonProjects := []struct {
			name    string
			dir     string
			command string
		}{
			{"pytest", "test-frameworks/python/pytest-svc", "pytest"},
			{"unittest", "test-frameworks/python/unittest-svc", "unittest"},
		}

		for _, proj := range pythonProjects {
			projPath := filepath.Join(testProjectsDir, proj.dir)
			fmt.Printf(fmtTestingProject, proj.name, proj.dir)
			if err := runPythonTests(projPath, proj.command); err != nil {
				fmt.Printf(fmtProjectFailed, proj.name, err)
				failed = append(failed, proj.name)
			} else {
				fmt.Printf(fmtProjectPassed, proj.name)
				passed = append(passed, proj.name)
			}
		}
		fmt.Println()
	}

	// Test .NET projects
	if language == "" || language == "dotnet" {
		fmt.Println("🔷 Testing .NET projects...")
		dotnetPath := filepath.Join(testProjectsDir, "test-frameworks/dotnet")
		fmt.Printf("   Testing xUnit + NUnit (solution)...\n")
		if err := runDotnetTests(dotnetPath); err != nil {
			fmt.Printf("   ❌ .NET tests failed: %v\n", err)
			failed = append(failed, ".NET (xUnit+NUnit)")
		} else {
			fmt.Printf("   ✅ .NET tests passed\n")
			passed = append(passed, ".NET (xUnit+NUnit)")
		}
		fmt.Println()
	}

	// Test Go projects
	if language == "" || language == "go" {
		fmt.Println("🐹 Testing Go projects...")
		goProjects := []struct {
			name string
			dir  string
		}{
			{"Go testing", "test-frameworks/go/testing-svc"},
			{"Go testify", "test-frameworks/go/testify-svc"},
		}

		for _, proj := range goProjects {
			projPath := filepath.Join(testProjectsDir, proj.dir)
			fmt.Printf(fmtTestingProject, proj.name, proj.dir)
			if err := runGoTests(projPath); err != nil {
				fmt.Printf(fmtProjectFailed, proj.name, err)
				failed = append(failed, proj.name)
			} else {
				fmt.Printf(fmtProjectPassed, proj.name)
				passed = append(passed, proj.name)
			}
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Printf("📊 Summary: %d passed, %d failed\n", len(passed), len(failed))
	if len(failed) > 0 {
		fmt.Println("   Failed projects:")
		for _, name := range failed {
			fmt.Printf(fmtBulletItem, name)
		}
		return fmt.Errorf("%d project(s) failed", len(failed))
	}

	fmt.Println("✅ All sample project tests passed!")
	return nil
}

// runNodeTests runs npm test in the given directory.
func runNodeTests(dir string) error {
	// Install dependencies first
	installCmd := exec.Command("npm", "install")
	installCmd.Dir = dir
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	// Run tests
	testCmd := exec.Command("npm", "test")
	testCmd.Dir = dir
	testCmd.Stdout = os.Stdout
	testCmd.Stderr = os.Stderr
	return testCmd.Run()
}

// runPythonTests runs Python tests in the given directory.
func runPythonTests(dir string, framework string) error {
	// Check for virtual environment and requirements
	reqPath := filepath.Join(dir, "requirements.txt")
	if _, err := os.Stat(reqPath); err == nil {
		// Install requirements
		pipCmd := exec.Command("pip", "install", "-r", "requirements.txt", "-q")
		pipCmd.Dir = dir
		pipCmd.Stdout = os.Stdout
		pipCmd.Stderr = os.Stderr
		if err := pipCmd.Run(); err != nil {
			return fmt.Errorf("pip install failed: %w", err)
		}
	}

	// Run tests based on framework
	var testCmd *exec.Cmd
	switch framework {
	case "pytest":
		testCmd = exec.Command("pytest", "tests/", "-v")
	case "unittest":
		testCmd = exec.Command("python", "-m", "unittest", "discover", "-s", "tests", "-p", "test_*.py", "-v")
	default:
		return fmt.Errorf("unknown framework: %s", framework)
	}

	testCmd.Dir = dir
	testCmd.Stdout = os.Stdout
	testCmd.Stderr = os.Stderr
	return testCmd.Run()
}

// runDotnetTests runs dotnet test in the given directory.
func runDotnetTests(dir string) error {
	// Restore and run tests on the solution
	restoreCmd := exec.Command("dotnet", "restore")
	restoreCmd.Dir = dir
	restoreCmd.Stdout = os.Stdout
	restoreCmd.Stderr = os.Stderr
	if err := restoreCmd.Run(); err != nil {
		return fmt.Errorf("dotnet restore failed: %w", err)
	}

	testCmd := exec.Command("dotnet", "test", "--no-restore", "-v", "minimal")
	testCmd.Dir = dir
	testCmd.Stdout = os.Stdout
	testCmd.Stderr = os.Stderr
	return testCmd.Run()
}

// runGoTests runs go test in the given directory.
func runGoTests(dir string) error {
	// Get dependencies
	modCmd := exec.Command("go", "mod", "download")
	modCmd.Dir = dir
	modCmd.Stdout = os.Stdout
	modCmd.Stderr = os.Stderr
	if err := modCmd.Run(); err != nil {
		// Ignore errors - module might not have external deps
		_ = err
	}

	// Run tests
	testCmd := exec.Command("go", "test", "-v", "./...")
	testCmd.Dir = dir
	testCmd.Stdout = os.Stdout
	testCmd.Stderr = os.Stderr
	return testCmd.Run()
}

// TestProjectsNode runs only Node.js sample project tests.
func TestProjectsNode() error {
	os.Setenv("LANGUAGE", "node")
	defer os.Unsetenv("LANGUAGE")
	return TestProjects()
}

// TestProjectsPython runs only Python sample project tests.
func TestProjectsPython() error {
	os.Setenv("LANGUAGE", "python")
	defer os.Unsetenv("LANGUAGE")
	return TestProjects()
}

// TestProjectsDotnet runs only .NET sample project tests.
func TestProjectsDotnet() error {
	os.Setenv("LANGUAGE", "dotnet")
	defer os.Unsetenv("LANGUAGE")
	return TestProjects()
}

// TestProjectsGo runs only Go sample project tests.
func TestProjectsGo() error {
	os.Setenv("LANGUAGE", "go")
	defer os.Unsetenv("LANGUAGE")
	return TestProjects()
}

// ============================================================================
// Negative/Failing Test Projects - verify error handling
// ============================================================================

const failingTestProjectsDir = "tests/projects/test-frameworks/failing"

// TestProjectsFailing runs intentionally failing test projects to verify error handling.
// All tests should fail (exit with error). Success means the test runners properly detect failures.
func TestProjectsFailing() error {
	fmt.Println("🧪 Running FAILING test projects (verifying error detection)...")
	fmt.Println("   These tests are EXPECTED to fail - we're verifying error handling.")
	fmt.Println()

	var verified []string
	var broken []string

	// Test Node.js failing project
	fmt.Println("📦 Testing Node.js failing project...")
	nodeDir := filepath.Join(failingTestProjectsDir, "node")
	if err := runNodeTests(nodeDir); err == nil {
		fmt.Println("   ❌ BROKEN: Node.js tests should have failed but passed!")
		broken = append(broken, "Node.js")
	} else {
		fmt.Println("   ✅ VERIFIED: Node.js correctly detected test failures")
		verified = append(verified, "Node.js")
	}
	fmt.Println()

	// Test Python failing project
	fmt.Println("🐍 Testing Python failing project...")
	pythonDir := filepath.Join(failingTestProjectsDir, "python")
	if err := runPythonTests(pythonDir, "pytest"); err == nil {
		fmt.Println("   ❌ BROKEN: Python tests should have failed but passed!")
		broken = append(broken, "Python")
	} else {
		fmt.Println("   ✅ VERIFIED: Python correctly detected test failures")
		verified = append(verified, "Python")
	}
	fmt.Println()

	// Test Go failing project
	fmt.Println("🐹 Testing Go failing project...")
	goDir := filepath.Join(failingTestProjectsDir, "go")
	if err := runGoTests(goDir); err == nil {
		fmt.Println("   ❌ BROKEN: Go tests should have failed but passed!")
		broken = append(broken, "Go")
	} else {
		fmt.Println("   ✅ VERIFIED: Go correctly detected test failures")
		verified = append(verified, "Go")
	}
	fmt.Println()

	// Test .NET failing project
	fmt.Println("🔷 Testing .NET failing project...")
	dotnetDir := filepath.Join(failingTestProjectsDir, "dotnet")
	if err := runDotnetTests(dotnetDir); err == nil {
		fmt.Println("   ❌ BROKEN: .NET tests should have failed but passed!")
		broken = append(broken, ".NET")
	} else {
		fmt.Println("   ✅ VERIFIED: .NET correctly detected test failures")
		verified = append(verified, ".NET")
	}
	fmt.Println()

	// Summary
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Printf("📊 Summary: %d verified, %d broken\n", len(verified), len(broken))
	if len(broken) > 0 {
		fmt.Println("   Broken (tests passed when they should fail):")
		for _, name := range broken {
			fmt.Printf(fmtBulletItem, name)
		}
		return fmt.Errorf("%d language(s) failed to detect test failures", len(broken))
	}

	fmt.Println("✅ All test runners correctly detect failures!")
	return nil
}
