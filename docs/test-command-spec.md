# `azd app test` Command Specification

## Overview

The `azd app test` command provides a unified interface for running tests across polyglot projects (Node.js, Python, .NET, etc.). It automatically detects project types, their package managers, and executes both unit and end-to-end (e2e) tests using each ecosystem's conventions.

## Design Principles

1. **Convention over Configuration**: Use standard test script names and locations per ecosystem
2. **Parallel Execution**: Run tests for different projects/languages concurrently when possible
3. **Fail-Fast Option**: Stop on first failure or continue to collect all failures
4. **Unified Output**: Consistent reporting format across all project types
5. **Flexible Filtering**: Support running specific test types, suites, or projects

## Command Signature

```bash
azd app test [flags]
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--type` | `-t` | string | `all` | Test type: `unit`, `e2e`, `integration`, `all` |
| `--project` | `-p` | string | `""` | Specific project path (relative or absolute) |
| `--parallel` | | bool | `true` | Run tests for different projects in parallel |
| `--fail-fast` | `-f` | bool | `false` | Stop on first test failure |
| `--coverage` | `-c` | bool | `false` | Collect coverage reports |
| `--coverage-format` | | string | `terminal` | Coverage format: `terminal`, `html`, `json`, `all` |
| `--coverage-threshold` | | float | `0` | Minimum coverage % required (fails if below) |
| `--coverage-files` | | bool | `false` | Show per-file coverage breakdown |
| `--report-format` | | string | `terminal` | Test results format: `terminal`, `html`, `json`, `junit`, `all` |
| `--watch` | `-w` | bool | `false` | Watch mode for development |
| `--verbose` | `-v` | bool | `false` | Verbose output (show all test logs) |
| `--filter` | | string | `""` | Test name/pattern filter (passed to test runner) |
| `--output-dir` | | string | `.test-results` | Directory for reports and results |

## Test Detection Strategy

### 1. Discovery Phase

The command walks the workspace directory tree and identifies test projects using the same detection logic as other commands:

```
workspace/
├── frontend/           # Node.js (npm) - package.json with test scripts
├── api/               # Python (poetry) - pyproject.toml with pytest
├── services/
│   └── auth/          # .NET - .csproj with xunit
└── e2e/               # Node.js (pnpm) - package.json with playwright
```

**Detection Criteria:**
- **Node.js**: `package.json` with `scripts.test`, `scripts.test:unit`, `scripts.test:e2e`
- **Python**: `pytest.ini`, `pyproject.toml` with `[tool.pytest]`, `tests/` directory
- **.NET**: `*.csproj` files with test framework references (xunit, nunit, mstest)
- **Go**: `*_test.go` files
- **Aspire**: Skip AppHost projects unless explicitly testing orchestration

### 2. Test Script Mapping

Each ecosystem has conventional test script names:

#### Node.js (npm/pnpm/yarn)

| Test Type | Script Name (Priority Order) |
|-----------|------------------------------|
| Unit | `test:unit`, `test` |
| E2E | `test:e2e`, `e2e` |
| Integration | `test:integration`, `test:int` |
| All | `test` (if no specific scripts exist) |

**Command Construction:**
```bash
# If --type=unit
npm run test:unit || npm test

# If --type=e2e
npm run test:e2e || npm run e2e

# If --coverage
npm run test:coverage || npm test -- --coverage
```

#### Python (pip/poetry/uv)

| Test Type | Convention |
|-----------|------------|
| Unit | `pytest tests/unit` or `pytest -m unit` |
| E2E | `pytest tests/e2e` or `pytest -m e2e` |
| Integration | `pytest tests/integration` or `pytest -m integration` |
| All | `pytest` |

**Command Construction:**
```bash
# Poetry example
poetry run pytest tests/unit

# Uv example
uv run pytest -m unit --cov

# Pip/venv example
.venv/Scripts/python -m pytest tests/e2e
```

#### .NET (dotnet)

| Test Type | Convention |
|-----------|------------|
| Unit | `dotnet test --filter Category=Unit` |
| E2E | `dotnet test --filter Category=E2E` |
| Integration | `dotnet test --filter Category=Integration` |
| All | `dotnet test` |

**Command Construction:**
```bash
# Unit tests only
dotnet test --filter "Category=Unit" --no-restore

# E2E with coverage
dotnet test --filter "Category=E2E" --collect:"XPlat Code Coverage"

# All tests
dotnet test --no-restore
```

#### Go

| Test Type | Convention |
|-----------|------------|
| Unit | `go test ./... -short` |
| E2E | `go test ./... -run E2E` |
| Integration | `go test ./... -run Integration` |
| All | `go test ./...` |

**Command Construction:**
```bash
# Unit tests
go test ./... -short -v

# E2E with coverage
go test ./... -run E2E -coverprofile=coverage.out

# All tests
go test ./...
```

## Execution Flow

```
┌─────────────────────────────────────┐
│ 1. Parse Flags & Validate           │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│ 2. Discover Test Projects           │
│    - Walk directory tree             │
│    - Detect project types            │
│    - Identify test scripts/patterns  │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│ 3. Filter Projects (if --project)   │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│ 4. Build Test Plans                 │
│    - Map test type to commands       │
│    - Validate test runners exist     │
│    - Check dependencies installed    │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│ 5. Execute Tests                     │
│    - Parallel: Goroutine per project │
│    - Sequential: If --parallel=false │
│    - Fail-fast: Cancel on failure    │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│ 6. Aggregate & Report Results       │
│    - Success/failure counts          │
│    - Coverage summaries (if enabled) │
│    - Execution times                 │
└─────────────────────────────────────┘
```

## Implementation Structure

### File: `src/internal/tester/tester.go`

```go
package tester

import (
    "app/src/internal/types"
)

// TestType represents the kind of tests to run.
type TestType string

const (
    TestTypeUnit        TestType = "unit"
    TestTypeE2E         TestType = "e2e"
    TestTypeIntegration TestType = "integration"
    TestTypeAll         TestType = "all"
)

// TestProject represents a project with runnable tests.
type TestProject struct {
    Dir            string
    Type           string // "node", "python", "dotnet", "go"
    PackageManager string
    TestScripts    map[TestType]string // Test type -> command to run
    HasTests       bool
}

// TestResult captures the outcome of running tests.
type TestResult struct {
    Project      string
    TestType     TestType
    Passed       bool
    Duration     time.Duration
    Output       string
    ExitCode     int
    CoverageFile string // Path to coverage report if generated
}

// FindTestProjects discovers all projects with tests.
func FindTestProjects(rootDir string) ([]TestProject, error)

// RunTests executes tests for a single project.
func RunTests(project TestProject, testType TestType, opts TestOptions) (*TestResult, error)

// RunAllTests executes tests across multiple projects.
func RunAllTests(projects []TestProject, opts TestOptions) ([]TestResult, error)
```

### File: `src/internal/tester/node.go`

```go
// DetectNodeTests checks package.json for test scripts.
func DetectNodeTests(projectDir string) (map[TestType]string, error) {
    scripts := make(map[TestType]string)
    
    packageJSON := readPackageJSON(projectDir)
    
    // Check for specific test scripts
    if script, exists := packageJSON.Scripts["test:unit"]; exists {
        scripts[TestTypeUnit] = script
    } else if script, exists := packageJSON.Scripts["test"]; exists {
        scripts[TestTypeUnit] = script
    }
    
    if script, exists := packageJSON.Scripts["test:e2e"]; exists {
        scripts[TestTypeE2E] = script
    } else if script, exists := packageJSON.Scripts["e2e"]; exists {
        scripts[TestTypeE2E] = script
    }
    
    // ... similar for integration, coverage, etc.
    
    return scripts, nil
}

// RunNodeTests executes npm/pnpm/yarn test scripts.
func RunNodeTests(project TestProject, testType TestType, opts TestOptions) (*TestResult, error) {
    script := project.TestScripts[testType]
    if script == "" {
        return nil, fmt.Errorf("no test script found for %s", testType)
    }
    
    args := []string{"run", script}
    
    // Add coverage flag if requested
    if opts.Coverage {
        args = append(args, "--", "--coverage")
    }
    
    // Add filter if provided
    if opts.Filter != "" {
        args = append(args, "--", "--testNamePattern", opts.Filter)
    }
    
    return executor.RunCommand(project.PackageManager, args, project.Dir)
}
```

### File: `src/internal/tester/python.go`

```go
// DetectPythonTests checks for pytest configuration and test directories.
func DetectPythonTests(projectDir string) (map[TestType]string, error) {
    scripts := make(map[TestType]string)
    
    // Check for test directories
    testDirs := []string{"tests/unit", "tests/e2e", "tests/integration"}
    for _, dir := range testDirs {
        fullPath := filepath.Join(projectDir, dir)
        if _, err := os.Stat(fullPath); err == nil {
            testType := getTestTypeFromPath(dir)
            scripts[testType] = dir
        }
    }
    
    // Fallback to marker-based tests
    if len(scripts) == 0 && hasTestFiles(projectDir) {
        scripts[TestTypeAll] = "."
    }
    
    return scripts, nil
}

// RunPythonTests executes pytest with appropriate arguments.
func RunPythonTests(project TestProject, testType TestType, opts TestOptions) (*TestResult, error) {
    var args []string
    
    switch project.PackageManager {
    case "poetry":
        args = []string{"run", "pytest"}
    case "uv":
        args = []string{"run", "pytest"}
    default: // pip/venv
        args = []string{"-m", "pytest"}
    }
    
    // Add test path or marker
    if testPath, exists := project.TestScripts[testType]; exists {
        args = append(args, testPath)
    } else {
        args = append(args, "-m", string(testType))
    }
    
    // Add coverage
    if opts.Coverage {
        args = append(args, "--cov", "--cov-report=xml")
    }
    
    // Add filter
    if opts.Filter != "" {
        args = append(args, "-k", opts.Filter)
    }
    
    cmd := getPythonCommand(project.PackageManager)
    return executor.RunCommand(cmd, args, project.Dir)
}
```

### File: `src/internal/tester/dotnet.go`

```go
// DetectDotnetTests finds .csproj files with test framework references.
func DetectDotnetTests(projectDir string) ([]string, error) {
    var testProjects []string
    
    err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
        if filepath.Ext(path) == ".csproj" {
            if isTestProject(path) {
                testProjects = append(testProjects, path)
            }
        }
        return nil
    })
    
    return testProjects, err
}

// isTestProject checks if csproj contains test framework references.
func isTestProject(csprojPath string) bool {
    content, _ := os.ReadFile(csprojPath)
    str := string(content)
    return strings.Contains(str, "xunit") ||
           strings.Contains(str, "nunit") ||
           strings.Contains(str, "MSTest")
}

// RunDotnetTests executes dotnet test with filters.
func RunDotnetTests(project TestProject, testType TestType, opts TestOptions) (*TestResult, error) {
    args := []string{"test", "--no-restore"}
    
    // Add category filter
    if testType != TestTypeAll {
        args = append(args, "--filter", fmt.Sprintf("Category=%s", testType))
    }
    
    // Add coverage
    if opts.Coverage {
        args = append(args, "--collect:\"XPlat Code Coverage\"")
    }
    
    // Add name filter
    if opts.Filter != "" {
        filterArg := fmt.Sprintf("FullyQualifiedName~%s", opts.Filter)
        args = append(args, "--filter", filterArg)
    }
    
    return executor.RunCommand("dotnet", args, project.Dir)
}
```

### File: `src/cmd/app/commands/test.go`

```go
package commands

import (
    "fmt"
    "os"
    "sync"
    
    "app/src/internal/orchestrator"
    "app/src/internal/tester"
    "github.com/spf13/cobra"
)

type TestOptions struct {
    TestType   string
    Project    string
    Parallel   bool
    FailFast   bool
    Coverage   bool
    Watch      bool
    Verbose    bool
    Filter     string
}

func newTestCommand() *cobra.Command {
    opts := TestOptions{}
    
    cmd := &cobra.Command{
        Use:   "test",
        Short: "Run tests across all detected projects",
        Long: `Runs unit, integration, and e2e tests across polyglot projects.
        
Automatically detects Node.js, Python, .NET, and Go projects and runs
their respective test suites using ecosystem conventions.

Examples:
  azd app test                    # Run all tests
  azd app test --type unit        # Run only unit tests
  azd app test --type e2e -c      # Run e2e tests with coverage
  azd app test --project ./api    # Test specific project
  azd app test --fail-fast        # Stop on first failure
  azd app test --watch            # Watch mode for development`,
        
        RunE: func(cmd *cobra.Command, args []string) error {
            return executeTest(opts)
        },
    }
    
    // Add flags
    cmd.Flags().StringVarP(&opts.TestType, "type", "t", "all", 
        "Test type: unit, e2e, integration, all")
    cmd.Flags().StringVarP(&opts.Project, "project", "p", "", 
        "Specific project path")
    cmd.Flags().BoolVar(&opts.Parallel, "parallel", true, 
        "Run tests for different projects in parallel")
    cmd.Flags().BoolVarP(&opts.FailFast, "fail-fast", "f", false, 
        "Stop on first test failure")
    cmd.Flags().BoolVarP(&opts.Coverage, "coverage", "c", false, 
        "Collect coverage reports")
    cmd.Flags().BoolVarP(&opts.Watch, "watch", "w", false, 
        "Watch mode for development")
    cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, 
        "Verbose output")
    cmd.Flags().StringVar(&opts.Filter, "filter", "", 
        "Test name/pattern filter")
    
    return cmd
}

// executeTest is the core logic for the test command.
func executeTest(opts TestOptions) error {
    fmt.Println("🧪 Running tests...")
    fmt.Println()
    
    // Get current working directory
    cwd, err := os.Getwd()
    if err != nil {
        return fmt.Errorf("failed to get current directory: %w", err)
    }
    
    // Override if specific project provided
    searchDir := cwd
    if opts.Project != "" {
        searchDir = opts.Project
    }
    
    // Discover all test projects
    testProjects, err := tester.FindTestProjects(searchDir)
    if err != nil {
        return fmt.Errorf("failed to discover test projects: %w", err)
    }
    
    if len(testProjects) == 0 {
        fmt.Println("ℹ️  No test projects found")
        return nil
    }
    
    // Display discovered projects
    fmt.Printf("📋 Found %d test project(s):\n", len(testProjects))
    for _, proj := range testProjects {
        fmt.Printf("   • %s (%s)\n", proj.Dir, proj.Type)
    }
    fmt.Println()
    
    // Convert test type string to enum
    testType := tester.TestType(opts.TestType)
    
    // Run tests
    var results []tester.TestResult
    if opts.Parallel {
        results, err = runTestsParallel(testProjects, testType, opts)
    } else {
        results, err = runTestsSequential(testProjects, testType, opts)
    }
    
    if err != nil {
        return err
    }
    
    // Display summary
    displayTestSummary(results, opts)
    
    // Check if any tests failed
    for _, result := range results {
        if !result.Passed {
            return fmt.Errorf("tests failed")
        }
    }
    
    return nil
}

// runTestsParallel executes tests for all projects concurrently.
func runTestsParallel(projects []tester.TestProject, testType tester.TestType, 
                      opts TestOptions) ([]tester.TestResult, error) {
    var wg sync.WaitGroup
    resultsChan := make(chan tester.TestResult, len(projects))
    errorsChan := make(chan error, len(projects))
    
    for _, project := range projects {
        wg.Add(1)
        go func(proj tester.TestProject) {
            defer wg.Done()
            
            result, err := tester.RunTests(proj, testType, convertOptions(opts))
            if err != nil {
                errorsChan <- err
                return
            }
            
            resultsChan <- *result
            
            // Fail-fast: if this test failed and fail-fast is enabled
            if opts.FailFast && !result.Passed {
                // Cancel other goroutines (requires context)
                return
            }
        }(project)
    }
    
    wg.Wait()
    close(resultsChan)
    close(errorsChan)
    
    // Collect results
    var results []tester.TestResult
    for result := range resultsChan {
        results = append(results, result)
    }
    
    // Check for errors
    select {
    case err := <-errorsChan:
        return results, err
    default:
        return results, nil
    }
}

// runTestsSequential executes tests for all projects one by one.
func runTestsSequential(projects []tester.TestProject, testType tester.TestType, 
                        opts TestOptions) ([]tester.TestResult, error) {
    var results []tester.TestResult
    
    for _, project := range projects {
        fmt.Printf("🔍 Testing %s...\n", project.Dir)
        
        result, err := tester.RunTests(project, testType, convertOptions(opts))
        if err != nil {
            if opts.FailFast {
                return results, err
            }
            fmt.Printf("❌ Error: %v\n\n", err)
            continue
        }
        
        results = append(results, *result)
        
        if opts.FailFast && !result.Passed {
            return results, fmt.Errorf("test failed: %s", project.Dir)
        }
    }
    
    return results, nil
}

// displayTestSummary shows aggregated test results.
func displayTestSummary(results []tester.TestResult, opts TestOptions) {
    fmt.Println()
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println("📊 Test Summary")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    
    passed := 0
    failed := 0
    totalDuration := time.Duration(0)
    
    for _, result := range results {
        status := "✓"
        if !result.Passed {
            status = "✗"
            failed++
        } else {
            passed++
        }
        
        fmt.Printf("%s %s (%s) - %v\n", 
            status, result.Project, result.TestType, result.Duration)
        
        totalDuration += result.Duration
        
        if opts.Coverage && result.CoverageFile != "" {
            fmt.Printf("   Coverage: %s\n", result.CoverageFile)
        }
    }
    
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Printf("Total: %d passed, %d failed (%v)\n", 
        passed, failed, totalDuration)
    
    if failed > 0 {
        fmt.Println("❌ Tests failed")
    } else {
        fmt.Println("✅ All tests passed!")
    }
}
```

## Example Workflows

### Scenario 1: Monorepo with Multiple Languages

```
workspace/
├── frontend/          # React app (npm) with jest
├── api/              # Python FastAPI (poetry) with pytest
├── services/
│   └── auth/         # .NET service with xunit
└── e2e/              # Playwright tests (pnpm)
```

**Command:**
```bash
azd app test
```

**Output:**
```
🧪 Running tests...

📋 Found 4 test project(s):
   • frontend (node)
   • api (python)
   • services/auth (dotnet)
   • e2e (node)

🔍 Testing frontend...
   ✓ 24 tests passed (3.2s)

🔍 Testing api...
   ✓ 15 tests passed (2.1s)

🔍 Testing services/auth...
   ✓ 8 tests passed (1.8s)

🔍 Testing e2e...
   ✓ 6 tests passed (12.4s)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Test Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ frontend (all) - 3.2s
✓ api (all) - 2.1s
✓ services/auth (all) - 1.8s
✓ e2e (all) - 12.4s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total: 4 passed, 0 failed (19.5s)
✅ All tests passed!
```

### Scenario 2: Run Only E2E Tests with Coverage

**Command:**
```bash
azd app test --type e2e --coverage
```

**Output:**
```
🧪 Running tests...

📋 Found 2 test project(s):
   • frontend (node)
   • e2e (node)

🔍 Testing frontend...
   Running e2e tests with coverage...
   ✓ 8 e2e tests passed (5.4s)
   Coverage: frontend/coverage/lcov.info

🔍 Testing e2e...
   Running e2e tests with coverage...
   ✓ 6 tests passed (14.2s)
   Coverage: e2e/coverage/coverage.xml

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Test Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ frontend (e2e) - 5.4s
   Coverage: frontend/coverage/lcov.info
✓ e2e (e2e) - 14.2s
   Coverage: e2e/coverage/coverage.xml
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total: 2 passed, 0 failed (19.6s)
✅ All tests passed!
```

### Scenario 3: Test Specific Project Only

**Command:**
```bash
azd app test --project ./api --type unit
```

**Output:**
```
🧪 Running tests...

📋 Found 1 test project(s):
   • api (python)

🔍 Testing api...
   Running unit tests...
   ✓ 15 tests passed (2.3s)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Test Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ api (unit) - 2.3s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total: 1 passed, 0 failed (2.3s)
✅ All tests passed!
```

## Dependencies in Orchestrator

The `test` command should depend on `deps` to ensure all dependencies are installed before running tests:

```go
// In commands/core.go init()
if err := cmdOrchestrator.Register(&orchestrator.Command{
    Name:         "test",
    Dependencies: []string{"deps"},
    Execute:      executeTest,
}); err != nil {
    fmt.Fprintf(os.Stderr, "Failed to register test command: %v\n", err)
    os.Exit(1)
}
```

**Dependency Chain:**
```
test → deps → reqs
```

## Edge Cases & Considerations

### 1. No Tests Found
```go
if len(testProjects) == 0 {
    fmt.Println("ℹ️  No test projects found")
    fmt.Println("   Add test scripts to package.json, pytest configuration,")
    fmt.Println("   or test projects with appropriate frameworks")
    return nil
}
```

### 2. Missing Test Runners
```go
// Check if test runner exists before running
if project.Type == "python" {
    if _, err := exec.LookPath("pytest"); err != nil {
        return fmt.Errorf("pytest not found. Install it with: %s install pytest", 
                         project.PackageManager)
    }
}
```

### 3. Watch Mode Limitation
Watch mode is challenging in a polyglot environment. Initial implementation should:
- Only support single project at a time: `azd app test --watch --project ./api`
- Delegate to the underlying tool's watch mode (jest --watch, pytest-watch, dotnet watch test)

### 4. Parallel Test Output Interleaving
Use buffered output per project and display after completion:
```go
type TestResult struct {
    // ... other fields
    StdOut string
    StdErr string
}

// After test completes, print buffered output
if opts.Verbose {
    fmt.Println(result.StdOut)
    if result.StdErr != "" {
        fmt.Fprintln(os.Stderr, result.StdErr)
    }
}
```

### 5. Coverage Report Aggregation

See the dedicated **Unified Coverage & Reporting** section below for comprehensive coverage aggregation strategy.

## Testing the Test Command

### Unit Tests
```go
// src/internal/tester/tester_test.go
func TestFindTestProjects(t *testing.T) {
    // Test discovery of Node, Python, .NET projects
}

func TestDetectNodeTests(t *testing.T) {
    // Test package.json script detection
}

func TestDetectPythonTests(t *testing.T) {
    // Test pytest configuration detection
}
```

### Integration Tests
```go
// src/cmd/app/commands/test_integration_test.go
func TestTestCommand_NodeProject(t *testing.T) {
    // Test against tests/projects/node/test-npm-project
}

func TestTestCommand_MultipleProjects(t *testing.T) {
    // Test parallel execution across node + python projects
}
```

### Test Fixtures
Create test fixture projects with actual tests:

```
tests/projects/
├── node-with-tests/
│   ├── package.json
│   │   "scripts": {
│   │     "test": "jest",
│   │     "test:unit": "jest --testPathPattern=unit",
│   │     "test:e2e": "jest --testPathPattern=e2e"
│   │   }
│   └── __tests__/
│       ├── unit/
│       │   └── example.test.js
│       └── e2e/
│           └── flow.test.js
├── python-with-tests/
│   ├── pyproject.toml
│   ├── tests/
│   │   ├── unit/
│   │   │   └── test_example.py
│   │   └── e2e/
│   │       └── test_flow.py
└── dotnet-with-tests/
    └── TestProject/
        ├── TestProject.csproj
        └── UnitTests.cs
```

## Future Enhancements

1. **Test Results Export**: Export results to JUnit XML, JSON, or other formats
2. **Coverage Thresholds**: Fail if coverage drops below specified percentage
3. **Flaky Test Detection**: Track test stability across runs
4. **Smart Test Selection**: Run only tests affected by changed files
5. **Cloud Test Reporting**: Integration with Azure Test Plans or GitHub Actions
6. **Container-based E2E**: Automatically spin up docker-compose for e2e tests
7. **Performance Testing**: Integrate with k6, artillery, or similar tools
8. **Differential Coverage**: Show coverage changes vs. base branch
9. **Coverage Trends**: Track coverage over time with historical data
10. **Mutation Testing**: Integrate with Stryker, PITest for mutation coverage

---

## Unified Test Results Reporting

In addition to coverage, test execution results need to be aggregated across all languages for a complete picture.

### Test Results Format Landscape

Different test frameworks produce results in different formats:

| Language | Framework | Format | Location |
|----------|-----------|--------|----------|
| Node.js | Jest | Jest JSON, JUnit XML | `junit.xml`, `test-results.json` |
| Python | pytest | JUnit XML, JSON | `junit.xml`, `.pytest_cache` |
| .NET | xunit/nunit/mstest | TRX, JUnit XML | `TestResults/*.trx` |
| Go | go test | Go test JSON | `test-output.json` |

### Unified Format: JUnit XML + JSON

**Primary Format**: JUnit XML for broad tool compatibility  
**Secondary Format**: Custom JSON for rich metadata and dashboard generation

### Test Results Structure

```go
// File: src/internal/tester/results.go

// TestResults represents aggregated results across all projects.
type TestResults struct {
    Summary      TestSummary
    Projects     []ProjectTestResults
    StartTime    time.Time
    EndTime      time.Time
    Duration     time.Duration
}

// TestSummary provides high-level aggregated metrics.
type TestSummary struct {
    TotalTests   int
    Passed       int
    Failed       int
    Skipped      int
    Errors       int
    SuccessRate  float64
}

// ProjectTestResults contains results for a single project.
type ProjectTestResults struct {
    Name         string
    Language     string
    Framework    string
    TestType     TestType
    Tests        int
    Passed       int
    Failed       int
    Skipped      int
    Duration     time.Duration
    TestCases    []TestCase
    ResultFile   string // Original result file
}

// TestCase represents a single test case execution.
type TestCase struct {
    Name         string
    ClassName    string
    Duration     time.Duration
    Status       TestStatus
    ErrorMessage string
    ErrorTrace   string
    Output       string
}

// TestStatus represents the outcome of a test.
type TestStatus string

const (
    TestStatusPassed  TestStatus = "passed"
    TestStatusFailed  TestStatus = "failed"
    TestStatusSkipped TestStatus = "skipped"
    TestStatusError   TestStatus = "error"
)
```

### JUnit XML Aggregation

```go
// File: src/internal/tester/junit.go

// JUnitTestSuites represents the root of JUnit XML.
type JUnitTestSuites struct {
    XMLName    xml.Name         `xml:"testsuites"`
    Name       string           `xml:"name,attr"`
    Tests      int              `xml:"tests,attr"`
    Failures   int              `xml:"failures,attr"`
    Errors     int              `xml:"errors,attr"`
    Skipped    int              `xml:"skipped,attr"`
    Time       float64          `xml:"time,attr"`
    TestSuites []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a suite of tests (typically a project).
type JUnitTestSuite struct {
    Name       string         `xml:"name,attr"`
    Tests      int            `xml:"tests,attr"`
    Failures   int            `xml:"failures,attr"`
    Errors     int            `xml:"errors,attr"`
    Skipped    int            `xml:"skipped,attr"`
    Time       float64        `xml:"time,attr"`
    Timestamp  string         `xml:"timestamp,attr"`
    TestCases  []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a single test.
type JUnitTestCase struct {
    Name       string          `xml:"name,attr"`
    ClassName  string          `xml:"classname,attr"`
    Time       float64         `xml:"time,attr"`
    Failure    *JUnitFailure   `xml:"failure,omitempty"`
    Error      *JUnitError     `xml:"error,omitempty"`
    Skipped    *JUnitSkipped   `xml:"skipped,omitempty"`
    SystemOut  string          `xml:"system-out,omitempty"`
    SystemErr  string          `xml:"system-err,omitempty"`
}

// ParseJUnitXML reads and parses a JUnit XML file.
func ParseJUnitXML(filePath string) (*JUnitTestSuites, error)

// MergeJUnitXML combines multiple JUnit XML files into one.
func MergeJUnitXML(files []string) (*JUnitTestSuites, error)

// ConvertToJUnitXML converts framework-specific formats to JUnit XML.
func ConvertToJUnitXML(sourceFile, format, projectName string) (string, error)
```

### Test Result Converters

```go
// File: src/internal/tester/converters_test.go

// ConvertJestToJUnit converts Jest JSON results to JUnit XML.
func ConvertJestToJUnit(jestFile, projectName string) (string, error) {
    // Parse Jest JSON format
    data, err := os.ReadFile(jestFile)
    if err != nil {
        return "", err
    }
    
    var jestResults JestResults
    if err := json.Unmarshal(data, &jestResults); err != nil {
        return "", err
    }
    
    // Convert to JUnit structure
    suite := &JUnitTestSuite{
        Name:      projectName,
        Tests:     jestResults.NumTotalTests,
        Failures:  jestResults.NumFailedTests,
        Skipped:   jestResults.NumPendingTests,
        Time:      float64(jestResults.Duration) / 1000.0,
        Timestamp: time.Now().Format(time.RFC3339),
    }
    
    for _, testSuite := range jestResults.TestResults {
        for _, test := range testSuite.AssertionResults {
            testCase := JUnitTestCase{
                Name:      test.Title,
                ClassName: testSuite.Name,
                Time:      float64(test.Duration) / 1000.0,
            }
            
            if test.Status == "failed" {
                testCase.Failure = &JUnitFailure{
                    Message: test.FailureMessages[0],
                    Type:    "AssertionError",
                    Content: strings.Join(test.FailureMessages, "\n"),
                }
            } else if test.Status == "pending" {
                testCase.Skipped = &JUnitSkipped{}
            }
            
            suite.TestCases = append(suite.TestCases, testCase)
        }
    }
    
    // Write to JUnit XML
    outputFile := filepath.Join(filepath.Dir(jestFile), "junit.xml")
    return writeJUnitXML(suite, outputFile)
}

// ConvertPytestToJUnit - pytest already outputs JUnit XML with --junit-xml flag
func ConvertPytestToJUnit(pytestFile string) (string, error) {
    // pytest already outputs JUnit XML, just validate and return path
    if _, err := ParseJUnitXML(pytestFile); err != nil {
        return "", fmt.Errorf("invalid JUnit XML: %w", err)
    }
    return pytestFile, nil
}

// ConvertTRXToJUnit converts .NET TRX format to JUnit XML.
func ConvertTRXToJUnit(trxFile, projectName string) (string, error) {
    // Parse TRX XML format
    data, err := os.ReadFile(trxFile)
    if err != nil {
        return "", err
    }
    
    var trx TRXTestRun
    if err := xml.Unmarshal(data, &trx); err != nil {
        return "", err
    }
    
    // Convert to JUnit
    suite := convertTRXToJUnitSuite(trx, projectName)
    
    outputFile := filepath.Join(filepath.Dir(trxFile), "junit.xml")
    return writeJUnitXML(suite, outputFile)
}

// ConvertGoTestToJUnit converts Go test JSON output to JUnit XML.
func ConvertGoTestToJUnit(goTestFile, projectName string) (string, error) {
    // Parse Go test JSON format (go test -json)
    tests := parseGoTestJSON(goTestFile)
    
    suite := &JUnitTestSuite{
        Name:      projectName,
        Timestamp: time.Now().Format(time.RFC3339),
    }
    
    for _, test := range tests {
        testCase := JUnitTestCase{
            Name:      test.Test,
            ClassName: test.Package,
            Time:      test.Elapsed,
        }
        
        suite.Tests++
        
        switch test.Action {
        case "pass":
            // No additional fields needed
        case "fail":
            suite.Failures++
            testCase.Failure = &JUnitFailure{
                Message: "Test failed",
                Content: test.Output,
            }
        case "skip":
            suite.Skipped++
            testCase.Skipped = &JUnitSkipped{}
        }
        
        suite.TestCases = append(suite.TestCases, testCase)
    }
    
    outputFile := filepath.Join(filepath.Dir(goTestFile), "junit.xml")
    return writeJUnitXML(suite, outputFile)
}
```

### Enhanced Test Execution with Result Collection

```go
// File: src/internal/tester/tester.go (updated)

// RunTests executes tests and collects results.
func RunTests(project TestProject, testType TestType, opts TestOptions) (*TestResult, error) {
    result := &TestResult{
        Project:   project.Dir,
        TestType:  testType,
        StartTime: time.Now(),
    }
    
    // Build test command based on language and package manager
    cmd, args := buildTestCommand(project, testType, opts)
    
    // Execute test command
    output, err := executor.RunCommandWithOutput(cmd, args, project.Dir)
    result.EndTime = time.Now()
    result.Duration = result.EndTime.Sub(result.StartTime)
    result.Output = output
    
    if err != nil {
        result.Passed = false
        result.ExitCode = getExitCode(err)
    } else {
        result.Passed = true
        result.ExitCode = 0
    }
    
    // Collect coverage file if coverage enabled
    if opts.Coverage {
        result.CoverageFile = findCoverageFile(project)
    }
    
    // Collect test result file
    result.ResultFile = findTestResultFile(project)
    
    // Parse test results for detailed metrics
    if result.ResultFile != "" {
        testResults, err := parseTestResults(result.ResultFile, project.Type)
        if err == nil {
            result.TestResults = testResults
        }
    }
    
    return result, err
}

// buildTestCommand constructs the appropriate test command with result output flags.
func buildTestCommand(project TestProject, testType TestType, opts TestOptions) (string, []string) {
    switch project.Type {
    case "node":
        return buildNodeTestCommand(project, testType, opts)
    case "python":
        return buildPythonTestCommand(project, testType, opts)
    case "dotnet":
        return buildDotnetTestCommand(project, testType, opts)
    case "go":
        return buildGoTestCommand(project, testType, opts)
    default:
        return "", nil
    }
}

// buildNodeTestCommand adds JUnit reporter to npm/jest.
func buildNodeTestCommand(project TestProject, testType TestType, opts TestOptions) (string, []string) {
    args := []string{"run", "test"}
    
    // Add Jest JUnit reporter
    args = append(args, "--", 
        "--reporters=default",
        "--reporters=jest-junit",
        "--coverageReporters=cobertura",
        "--coverageReporters=text")
    
    if opts.Coverage {
        args = append(args, "--coverage")
    }
    
    return project.PackageManager, args
}

// buildPythonTestCommand adds JUnit XML output to pytest.
func buildPythonTestCommand(project TestProject, testType TestType, opts TestOptions) (string, []string) {
    var cmd string
    var baseArgs []string
    
    switch project.PackageManager {
    case "poetry":
        cmd = "poetry"
        baseArgs = []string{"run", "pytest"}
    case "uv":
        cmd = "uv"
        baseArgs = []string{"run", "pytest"}
    default:
        cmd = "python"
        baseArgs = []string{"-m", "pytest"}
    }
    
    args := append(baseArgs, "--junit-xml=junit.xml")
    
    if opts.Coverage {
        args = append(args, "--cov", "--cov-report=xml")
    }
    
    return cmd, args
}

// buildDotnetTestCommand adds test logger to dotnet test.
func buildDotnetTestCommand(project TestProject, testType TestType, opts TestOptions) (string, []string) {
    args := []string{"test", "--no-restore"}
    
    // Add JUnit logger
    args = append(args, "--logger:junit")
    
    if opts.Coverage {
        args = append(args, "--collect:\"XPlat Code Coverage\"")
    }
    
    if testType != TestTypeAll {
        args = append(args, "--filter", fmt.Sprintf("Category=%s", testType))
    }
    
    return "dotnet", args
}

// buildGoTestCommand adds JSON output to go test.
func buildGoTestCommand(project TestProject, testType TestType, opts TestOptions) (string, []string) {
    args := []string{"test", "./...", "-json"}
    
    if opts.Coverage {
        args = append(args, "-coverprofile=coverage.out")
    }
    
    if testType == TestTypeUnit {
        args = append(args, "-short")
    }
    
    return "go", args
}
```

### Unified Report Generation

```go
// File: src/internal/tester/reporter.go

// GenerateUnifiedReport creates a comprehensive test + coverage report.
func GenerateUnifiedReport(testResults []TestResult, coverageReport *CoverageReport, opts ReportOptions) error {
    switch opts.Format {
    case "terminal":
        return generateTerminalUnifiedReport(testResults, coverageReport, opts)
    case "html":
        return generateHTMLUnifiedReport(testResults, coverageReport, opts)
    case "json":
        return generateJSONUnifiedReport(testResults, coverageReport, opts)
    case "junit":
        return generateJUnitReport(testResults, opts)
    case "all":
        generateTerminalUnifiedReport(testResults, coverageReport, opts)
        generateHTMLUnifiedReport(testResults, coverageReport, opts)
        generateJSONUnifiedReport(testResults, coverageReport, opts)
        return generateJUnitReport(testResults, opts)
    default:
        return fmt.Errorf("unsupported format: %s", opts.Format)
    }
}

// generateJUnitReport creates a merged JUnit XML file.
func generateJUnitReport(results []TestResult, opts ReportOptions) error {
    var junitFiles []string
    
    // Collect all JUnit files
    for _, result := range results {
        if result.ResultFile != "" {
            junitFiles = append(junitFiles, result.ResultFile)
        }
    }
    
    // Merge all JUnit files
    merged, err := MergeJUnitXML(junitFiles)
    if err != nil {
        return fmt.Errorf("failed to merge JUnit files: %w", err)
    }
    
    // Write merged file
    outputPath := filepath.Join(opts.OutputDir, "test-results.xml")
    file, err := os.Create(outputPath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    encoder := xml.NewEncoder(file)
    encoder.Indent("", "  ")
    if err := encoder.Encode(merged); err != nil {
        return err
    }
    
    fmt.Printf("📄 JUnit test results: %s\n", outputPath)
    return nil
}
```

### Complete Example with Coverage + Results

**Command:**
```bash
azd app test --coverage --coverage-format=all --report-format=all --coverage-threshold=80
```

**Output:**
```
🧪 Running tests with coverage...

📋 Found 4 test project(s):
   • frontend (node)
   • api (python)
   • services/auth (dotnet)
   • shared (go)

🔍 Testing frontend...
   ✓ 24/24 tests passed (3.2s)
   📊 Coverage: 85.3%
   📝 Results: frontend/junit.xml

🔍 Testing api...
   ✓ 15/15 tests passed (2.1s)
   📊 Coverage: 92.1%
   📝 Results: api/junit.xml

🔍 Testing services/auth...
   ⚠️  7/8 tests passed, 1 failed (1.8s)
   📊 Coverage: 78.5%
   📝 Results: services/auth/TestResults/junit.xml

🔍 Testing shared...
   ✓ 12/12 tests passed (0.9s)
   📊 Coverage: 88.7%
   📝 Results: shared/junit.xml

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Test Results Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ frontend          24/24 passed   (3.2s)
✓ api               15/15 passed   (2.1s)
❌ services/auth     7/8 passed    (1.8s)
✓ shared            12/12 passed   (0.9s)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total: 58/59 tests passed (98.3%), 1 failed (8.0s)

Failed Tests:
❌ services/auth > UserServiceTests > Should_ValidateEmail_WithInvalidFormat
   Expected: true
   Actual: false
   at UserServiceTests.Should_ValidateEmail_WithInvalidFormat() in UserServiceTests.cs:line 42

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Coverage Report
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ frontend                     85.3% (234/274 lines)
✅ api                          92.1% (156/169 lines)
⚠️  services/auth                78.5% (102/130 lines) ⬇️ Below 80%
✅ shared                       88.7% (142/160 lines)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Overall Coverage: 86.4% (634/733 lines, 89/102 branches 87.3%)
✅ Coverage meets threshold 80.0%

⚠️  Note: services/auth has coverage below threshold (78.5% < 80.0%)

📄 Reports generated:
   Test Results:
   • JUnit XML: .coverage/test-results.xml
   • JSON: .coverage/test-results.json
   • HTML: .coverage/test-report.html
   
   Coverage:
   • Terminal summary (above)
   • HTML: .coverage/coverage-report.html
   • JSON: .coverage/coverage-report.json
   • Cobertura XML: .coverage/coverage-merged.xml

❌ Tests failed (1 failure)
```

## Future Enhancements

### Unified Format: Cobertura XML

**Choice**: Cobertura XML as the unified interchange format

**Rationale**:
- Widely supported across ecosystems
- Good tooling support (Azure DevOps, SonarQube, Codecov, Coveralls)
- Structured XML format that's easy to parse and merge
- Contains all necessary metadata (file paths, line coverage, branch coverage)

### Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                   azd app test --coverage                     │
└────────────┬─────────────────────────────────────────────────┘
             │
             ├──> Node.js Project
             │    ├─> Run: npm test -- --coverage
             │    ├─> Generate: coverage/lcov.info
             │    └─> Convert: lcov → cobertura.xml
             │
             ├──> Python Project  
             │    ├─> Run: pytest --cov --cov-report=xml
             │    └─> Already Cobertura: coverage.xml ✓
             │
             ├──> .NET Project
             │    ├─> Run: dotnet test --collect:"XPlat Code Coverage"
             │    └─> Already Cobertura: coverage.cobertura.xml ✓
             │
             └──> Go Project
                  ├─> Run: go test -coverprofile=coverage.out
                  └─> Convert: coverage.out → cobertura.xml
                  
┌──────────────────────────────────────────────────────────────┐
│              Coverage Aggregation Pipeline                    │
└────────────┬─────────────────────────────────────────────────┘
             │
             ├──> Normalize Paths (relative to workspace root)
             ├──> Merge Cobertura XML files
             ├──> Calculate Overall Metrics
             │    ├─> Total lines covered/total lines
             │    ├─> Total branches covered/total branches
             │    └─> Per-project breakdown
             │
             └──> Generate Reports
                  ├─> Terminal Summary (default)
                  ├─> HTML Report (--coverage-html)
                  ├─> JSON Report (--coverage-json)
                  └─> Merged Cobertura XML (coverage-merged.xml)
```

### Implementation Structure

#### File: `src/internal/coverage/coverage.go`

```go
package coverage

import (
    "encoding/xml"
    "path/filepath"
)

// CoverageReport represents a unified coverage report.
type CoverageReport struct {
    Projects       []ProjectCoverage
    TotalLines     int
    CoveredLines   int
    TotalBranches  int
    CoveredBranches int
    Percentage     float64
}

// ProjectCoverage represents coverage for a single project.
type ProjectCoverage struct {
    Name           string
    Language       string
    SourceFile     string // Original coverage file
    Lines          int
    CoveredLines   int
    Branches       int
    CoveredBranches int
    Percentage     float64
    Files          []FileCoverage
}

// FileCoverage represents coverage for a single source file.
type FileCoverage struct {
    Path           string
    Lines          int
    CoveredLines   int
    Branches       int
    CoveredBranches int
    Percentage     float64
}

// Cobertura XML structures for parsing and generation
type Cobertura struct {
    XMLName     xml.Name           `xml:"coverage"`
    LineRate    float64            `xml:"line-rate,attr"`
    BranchRate  float64            `xml:"branch-rate,attr"`
    Version     string             `xml:"version,attr"`
    Timestamp   int64              `xml:"timestamp,attr"`
    Sources     []string           `xml:"sources>source"`
    Packages    []CoberturaPackage `xml:"packages>package"`
}

type CoberturaPackage struct {
    Name       string            `xml:"name,attr"`
    LineRate   float64           `xml:"line-rate,attr"`
    BranchRate float64           `xml:"branch-rate,attr"`
    Classes    []CoberturaClass  `xml:"classes>class"`
}

type CoberturaClass struct {
    Name       string            `xml:"name,attr"`
    Filename   string            `xml:"filename,attr"`
    LineRate   float64           `xml:"line-rate,attr"`
    BranchRate float64           `xml:"branch-rate,attr"`
    Lines      []CoberturaLine   `xml:"lines>line"`
}

type CoberturaLine struct {
    Number int     `xml:"number,attr"`
    Hits   int     `xml:"hits,attr"`
    Branch bool    `xml:"branch,attr,omitempty"`
}

// ConvertToCobertura converts various coverage formats to Cobertura XML.
func ConvertToCobertura(sourceFile, format, projectDir string) (string, error)

// MergeCoberturaFiles combines multiple Cobertura XML files into one.
func MergeCoberturaFiles(files []string, workspaceRoot string) (*Cobertura, error)

// GenerateCoverageReport creates a unified coverage report from multiple sources.
func GenerateCoverageReport(coverageSources []CoverageSource, workspaceRoot string) (*CoverageReport, error)
```

#### File: `src/internal/coverage/converters.go`

```go
package coverage

import (
    "bufio"
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"
)

// CoverageSource represents a coverage file from a test project.
type CoverageSource struct {
    ProjectName string
    Language    string
    FilePath    string
    Format      string // "lcov", "cobertura", "go", "json"
    ProjectDir  string
}

// ConvertLcovToCobertura converts LCOV format to Cobertura XML.
func ConvertLcovToCobertura(lcovFile, projectDir string) (string, error) {
    // Parse LCOV file
    coverage := parseLcovFile(lcovFile)
    
    // Build Cobertura structure
    cobertura := &Cobertura{
        Version:   "1.0",
        Timestamp: time.Now().Unix(),
        Sources:   []string{projectDir},
    }
    
    // Group files into packages (by directory)
    packages := make(map[string]*CoberturaPackage)
    
    for filePath, fileCov := range coverage {
        packageName := filepath.Dir(filePath)
        if packageName == "." {
            packageName = "root"
        }
        
        pkg, exists := packages[packageName]
        if !exists {
            pkg = &CoberturaPackage{
                Name:    packageName,
                Classes: []CoberturaClass{},
            }
            packages[packageName] = pkg
        }
        
        // Convert to Cobertura class
        class := CoberturaClass{
            Name:     filepath.Base(filePath),
            Filename: filePath,
            Lines:    convertLinesToCobertura(fileCov.Lines),
        }
        
        // Calculate rates
        class.LineRate = calculateLineRate(class.Lines)
        class.BranchRate = calculateBranchRate(class.Lines)
        
        pkg.Classes = append(pkg.Classes, class)
    }
    
    // Add packages to cobertura
    for _, pkg := range packages {
        pkg.LineRate = calculatePackageLineRate(pkg)
        pkg.BranchRate = calculatePackageBranchRate(pkg)
        cobertura.Packages = append(cobertura.Packages, *pkg)
    }
    
    // Calculate overall rates
    cobertura.LineRate = calculateOverallLineRate(cobertura)
    cobertura.BranchRate = calculateOverallBranchRate(cobertura)
    
    // Write to XML file
    outputFile := filepath.Join(projectDir, "coverage", "cobertura.xml")
    if err := writeCoberturaXML(cobertura, outputFile); err != nil {
        return "", err
    }
    
    return outputFile, nil
}

// parseLcovFile parses an LCOV coverage file.
func parseLcovFile(lcovFile string) map[string]*LcovFileCoverage {
    coverage := make(map[string]*LcovFileCoverage)
    
    file, err := os.Open(lcovFile)
    if err != nil {
        return coverage
    }
    defer file.Close()
    
    var currentFile *LcovFileCoverage
    var currentPath string
    
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        
        if strings.HasPrefix(line, "SF:") {
            // Source file
            currentPath = strings.TrimPrefix(line, "SF:")
            currentFile = &LcovFileCoverage{
                Path:  currentPath,
                Lines: make(map[int]int),
            }
            coverage[currentPath] = currentFile
        } else if strings.HasPrefix(line, "DA:") {
            // Line coverage data
            parts := strings.Split(strings.TrimPrefix(line, "DA:"), ",")
            if len(parts) == 2 {
                lineNum, _ := strconv.Atoi(parts[0])
                hits, _ := strconv.Atoi(parts[1])
                currentFile.Lines[lineNum] = hits
            }
        } else if strings.HasPrefix(line, "BRDA:") {
            // Branch coverage data
            currentFile.HasBranches = true
        }
    }
    
    return coverage
}

// ConvertGoCoverageToCobertura converts Go coverage format to Cobertura XML.
func ConvertGoCoverageToCobertura(coverageFile, projectDir string) (string, error) {
    // Parse Go coverage file (format: mode: set\nfile.go:startLine.startCol,endLine.endCol numStmt count)
    coverage := parseGoCoverageFile(coverageFile)
    
    // Build Cobertura structure similar to LCOV conversion
    cobertura := buildCoberturaFromGoCoverage(coverage, projectDir)
    
    // Write to XML
    outputFile := filepath.Join(projectDir, "coverage", "cobertura.xml")
    if err := writeCoberturaXML(cobertura, outputFile); err != nil {
        return "", err
    }
    
    return outputFile, nil
}

// ConvertJSONCoverageToCobertura converts Jest JSON coverage to Cobertura.
func ConvertJSONCoverageToCobertura(jsonFile, projectDir string) (string, error) {
    // Parse coverage-final.json format
    coverage := parseJestJSONCoverage(jsonFile)
    
    cobertura := buildCoberturaFromJSON(coverage, projectDir)
    
    outputFile := filepath.Join(projectDir, "coverage", "cobertura.xml")
    if err := writeCoberturaXML(cobertura, outputFile); err != nil {
        return "", err
    }
    
    return outputFile, nil
}
```

#### File: `src/internal/coverage/merger.go`

```go
package coverage

import (
    "encoding/xml"
    "fmt"
    "os"
    "path/filepath"
)

// MergeCoberturaFiles combines multiple Cobertura XML files into one unified report.
func MergeCoberturaFiles(files []string, workspaceRoot string) (*Cobertura, error) {
    if len(files) == 0 {
        return nil, fmt.Errorf("no coverage files to merge")
    }
    
    merged := &Cobertura{
        Version:   "1.0",
        Timestamp: time.Now().Unix(),
        Sources:   []string{workspaceRoot},
        Packages:  []CoberturaPackage{},
    }
    
    packageMap := make(map[string]*CoberturaPackage)
    
    for _, file := range files {
        cov, err := parseCoberturaFile(file)
        if err != nil {
            fmt.Printf("Warning: failed to parse %s: %v\n", file, err)
            continue
        }
        
        // Merge packages
        for _, pkg := range cov.Packages {
            // Normalize package paths relative to workspace root
            normalizedName := normalizePath(pkg.Name, workspaceRoot)
            
            existingPkg, exists := packageMap[normalizedName]
            if !exists {
                pkg.Name = normalizedName
                packageMap[normalizedName] = &pkg
            } else {
                // Merge classes from same package
                existingPkg.Classes = append(existingPkg.Classes, pkg.Classes...)
            }
        }
    }
    
    // Convert map back to slice and recalculate rates
    for _, pkg := range packageMap {
        pkg.LineRate = calculatePackageLineRate(pkg)
        pkg.BranchRate = calculatePackageBranchRate(pkg)
        merged.Packages = append(merged.Packages, *pkg)
    }
    
    // Calculate overall rates
    merged.LineRate = calculateOverallLineRate(merged)
    merged.BranchRate = calculateOverallBranchRate(merged)
    
    return merged, nil
}

// parseCoberturaFile reads and parses a Cobertura XML file.
func parseCoberturaFile(filePath string) (*Cobertura, error) {
    data, err := os.ReadFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }
    
    var cov Cobertura
    if err := xml.Unmarshal(data, &cov); err != nil {
        return nil, fmt.Errorf("failed to parse XML: %w", err)
    }
    
    return &cov, nil
}

// normalizePath converts absolute or project-relative paths to workspace-relative.
func normalizePath(path, workspaceRoot string) string {
    // If path is absolute, make it relative to workspace
    if filepath.IsAbs(path) {
        rel, err := filepath.Rel(workspaceRoot, path)
        if err != nil {
            return path
        }
        return rel
    }
    
    // Already relative
    return path
}

// calculateOverallLineRate computes the overall line coverage rate.
func calculateOverallLineRate(cov *Cobertura) float64 {
    totalLines := 0
    coveredLines := 0
    
    for _, pkg := range cov.Packages {
        for _, class := range pkg.Classes {
            for _, line := range class.Lines {
                totalLines++
                if line.Hits > 0 {
                    coveredLines++
                }
            }
        }
    }
    
    if totalLines == 0 {
        return 0.0
    }
    
    return float64(coveredLines) / float64(totalLines)
}
```

#### File: `src/internal/coverage/reporter.go`

```go
package coverage

import (
    "fmt"
    "os"
    "path/filepath"
    "text/template"
)

// ReportOptions configures coverage report generation.
type ReportOptions struct {
    Format        string // "terminal", "html", "json"
    OutputDir     string
    MinCoverage   float64 // Minimum coverage threshold (0-100)
    ShowFiles     bool    // Show per-file breakdown
    ShowUncovered bool    // Highlight uncovered lines
}

// GenerateReport creates a coverage report in the specified format.
func GenerateReport(report *CoverageReport, opts ReportOptions) error {
    switch opts.Format {
    case "terminal":
        return generateTerminalReport(report, opts)
    case "html":
        return generateHTMLReport(report, opts)
    case "json":
        return generateJSONReport(report, opts)
    default:
        return fmt.Errorf("unsupported format: %s", opts.Format)
    }
}

// generateTerminalReport prints coverage to terminal with colors.
func generateTerminalReport(report *CoverageReport, opts ReportOptions) error {
    fmt.Println()
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println("📊 Coverage Report")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    
    // Per-project coverage
    for _, proj := range report.Projects {
        coverageStr := formatCoverage(proj.Percentage)
        icon := getCoverageIcon(proj.Percentage)
        
        fmt.Printf("%s %-30s %s (%d/%d lines)\n",
            icon,
            proj.Name,
            coverageStr,
            proj.CoveredLines,
            proj.Lines)
        
        // Show per-file breakdown if requested
        if opts.ShowFiles && len(proj.Files) > 0 {
            for _, file := range proj.Files {
                fileIcon := getCoverageIcon(file.Percentage)
                fmt.Printf("   %s %-40s %s\n",
                    fileIcon,
                    file.Path,
                    formatCoverage(file.Percentage))
            }
            fmt.Println()
        }
    }
    
    // Overall coverage
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    overallIcon := getCoverageIcon(report.Percentage)
    fmt.Printf("%s Overall Coverage: %s (%d/%d lines",
        overallIcon,
        formatCoverage(report.Percentage),
        report.CoveredLines,
        report.TotalLines)
    
    if report.TotalBranches > 0 {
        branchPct := float64(report.CoveredBranches) / float64(report.TotalBranches) * 100
        fmt.Printf(", %d/%d branches %.1f%%", 
            report.CoveredBranches, 
            report.TotalBranches, 
            branchPct)
    }
    fmt.Println(")")
    
    // Check threshold
    if opts.MinCoverage > 0 && report.Percentage < opts.MinCoverage {
        fmt.Printf("❌ Coverage %.1f%% is below threshold %.1f%%\n", 
            report.Percentage, opts.MinCoverage)
        return fmt.Errorf("coverage threshold not met")
    } else if opts.MinCoverage > 0 {
        fmt.Printf("✅ Coverage meets threshold %.1f%%\n", opts.MinCoverage)
    }
    
    return nil
}

// formatCoverage formats a percentage with color coding.
func formatCoverage(pct float64) string {
    color := getCoverageColor(pct)
    return fmt.Sprintf("%s%.1f%%%s", color, pct, colorReset)
}

// getCoverageColor returns ANSI color code based on coverage percentage.
func getCoverageColor(pct float64) string {
    switch {
    case pct >= 80:
        return "\033[32m" // Green
    case pct >= 60:
        return "\033[33m" // Yellow
    default:
        return "\033[31m" // Red
    }
}

// getCoverageIcon returns an emoji based on coverage level.
func getCoverageIcon(pct float64) string {
    switch {
    case pct >= 80:
        return "✅"
    case pct >= 60:
        return "⚠️ "
    default:
        return "❌"
    }
}

const colorReset = "\033[0m"

// generateHTMLReport creates an interactive HTML coverage report.
func generateHTMLReport(report *CoverageReport, opts ReportOptions) error {
    outputPath := filepath.Join(opts.OutputDir, "coverage-report.html")
    
    tmpl := template.Must(template.New("coverage").Parse(htmlTemplate))
    
    file, err := os.Create(outputPath)
    if err != nil {
        return fmt.Errorf("failed to create HTML report: %w", err)
    }
    defer file.Close()
    
    if err := tmpl.Execute(file, report); err != nil {
        return fmt.Errorf("failed to generate HTML: %w", err)
    }
    
    fmt.Printf("📄 HTML coverage report: %s\n", outputPath)
    return nil
}

// HTML template for coverage report
const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>Coverage Report</title>
    <style>
        body { font-family: system-ui; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; }
        h1 { color: #333; }
        .overall { font-size: 24px; padding: 20px; background: #f0f0f0; border-radius: 4px; margin: 20px 0; }
        .project { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 4px; }
        .high { color: #28a745; }
        .medium { color: #ffc107; }
        .low { color: #dc3545; }
        .progress { height: 20px; background: #e9ecef; border-radius: 4px; overflow: hidden; }
        .progress-bar { height: 100%; transition: width 0.3s; }
        table { width: 100%; border-collapse: collapse; margin-top: 10px; }
        th, td { padding: 8px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #f8f9fa; font-weight: 600; }
    </style>
</head>
<body>
    <div class="container">
        <h1>📊 Coverage Report</h1>
        
        <div class="overall">
            <strong>Overall Coverage:</strong>
            <span class="{{if ge .Percentage 80.0}}high{{else if ge .Percentage 60.0}}medium{{else}}low{{end}}">
                {{printf "%.1f%%" .Percentage}}
            </span>
            ({{.CoveredLines}}/{{.TotalLines}} lines)
            
            <div class="progress" style="margin-top: 10px;">
                <div class="progress-bar {{if ge .Percentage 80.0}}high{{else if ge .Percentage 60.0}}medium{{else}}low{{end}}" 
                     style="width: {{.Percentage}}%; background-color: {{if ge .Percentage 80.0}}#28a745{{else if ge .Percentage 60.0}}#ffc107{{else}}#dc3545{{end}};">
                </div>
            </div>
        </div>
        
        <h2>Projects</h2>
        {{range .Projects}}
        <div class="project">
            <h3>{{.Name}} <span style="color: #666;">({{.Language}})</span></h3>
            <p>
                Coverage: 
                <strong class="{{if ge .Percentage 80.0}}high{{else if ge .Percentage 60.0}}medium{{else}}low{{end}}">
                    {{printf "%.1f%%" .Percentage}}
                </strong>
                ({{.CoveredLines}}/{{.Lines}} lines)
            </p>
            
            <div class="progress">
                <div class="progress-bar" style="width: {{.Percentage}}%; background-color: {{if ge .Percentage 80.0}}#28a745{{else if ge .Percentage 60.0}}#ffc107{{else}}#dc3545{{end}};"></div>
            </div>
            
            {{if .Files}}
            <table>
                <thead>
                    <tr>
                        <th>File</th>
                        <th>Coverage</th>
                        <th>Lines</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Files}}
                    <tr>
                        <td>{{.Path}}</td>
                        <td class="{{if ge .Percentage 80.0}}high{{else if ge .Percentage 60.0}}medium{{else}}low{{end}}">
                            {{printf "%.1f%%" .Percentage}}
                        </td>
                        <td>{{.CoveredLines}}/{{.Lines}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
            {{end}}
        </div>
        {{end}}
    </div>
</body>
</html>`

// generateJSONReport exports coverage data as JSON.
func generateJSONReport(report *CoverageReport, opts ReportOptions) error {
    outputPath := filepath.Join(opts.OutputDir, "coverage-report.json")
    
    data, err := json.MarshalIndent(report, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal JSON: %w", err)
    }
    
    if err := os.WriteFile(outputPath, data, 0644); err != nil {
        return fmt.Errorf("failed to write JSON: %w", err)
    }
    
    fmt.Printf("📄 JSON coverage report: %s\n", outputPath)
    return nil
}
```

### Updated Command Flags

Add new flags for coverage reporting:

```go
cmd.Flags().BoolVarP(&opts.Coverage, "coverage", "c", false, 
    "Collect coverage reports")
cmd.Flags().StringVar(&opts.CoverageFormat, "coverage-format", "terminal", 
    "Coverage report format: terminal, html, json, all")
cmd.Flags().Float64Var(&opts.CoverageThreshold, "coverage-threshold", 0, 
    "Minimum coverage percentage required (0-100, fails if below)")
cmd.Flags().BoolVar(&opts.CoverageShowFiles, "coverage-files", false, 
    "Show per-file coverage breakdown")
cmd.Flags.StringVar(&opts.CoverageOutputDir, "coverage-output", ".coverage", 
    "Directory for coverage reports")
```

### Enhanced Workflow with Coverage

```
┌─────────────────────────────────────┐
│ 1. Run Tests with Coverage          │
│    - Execute tests per project      │
│    - Collect coverage files          │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│ 2. Detect Coverage Formats           │
│    - LCOV (Node.js)                  │
│    - Cobertura (Python, .NET)        │
│    - Go coverage format              │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│ 3. Convert to Unified Format         │
│    - LCOV → Cobertura                │
│    - Go → Cobertura                  │
│    - JSON → Cobertura                │
│    - Keep existing Cobertura         │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│ 4. Merge Coverage Reports            │
│    - Normalize file paths            │
│    - Combine Cobertura XMLs          │
│    - Calculate aggregate metrics     │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│ 5. Generate Reports                  │
│    - Terminal: Colored summary       │
│    - HTML: Interactive dashboard     │
│    - JSON: Machine-readable export   │
│    - Merged Cobertura XML            │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│ 6. Validate Thresholds               │
│    - Check against --coverage-threshold│
│    - Fail build if below threshold   │
└─────────────────────────────────────┘
```

### Example: Complete Coverage Flow

**Command:**
```bash
azd app test --coverage --coverage-format=all --coverage-threshold=80
```

**Output:**
```
🧪 Running tests with coverage...

📋 Found 4 test project(s):
   • frontend (node)
   • api (python)
   • services/auth (dotnet)
   • shared (go)

🔍 Testing frontend...
   Running tests with coverage...
   ✓ 24 tests passed (3.2s)
   📊 Coverage: 85.3% (frontend/coverage/lcov.info)
   🔄 Converting LCOV → Cobertura...

🔍 Testing api...
   Running tests with coverage...
   ✓ 15 tests passed (2.1s)
   📊 Coverage: 92.1% (api/coverage.xml) ✓

🔍 Testing services/auth...
   Running tests with coverage...
   ✓ 8 tests passed (1.8s)
   📊 Coverage: 78.5% (services/auth/TestResults/.../coverage.cobertura.xml) ✓

🔍 Testing shared...
   Running tests with coverage...
   ✓ 12 tests passed (0.9s)
   📊 Coverage: 88.7% (shared/coverage.out)
   🔄 Converting Go → Cobertura...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Test Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ frontend (all) - 3.2s
✓ api (all) - 2.1s
✓ services/auth (all) - 1.8s
✓ shared (all) - 0.9s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total: 4 passed, 0 failed (8.0s)
✅ All tests passed!

🔄 Merging coverage reports...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Coverage Report
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ frontend                     85.3% (234/274 lines)
✅ api                          92.1% (156/169 lines)
⚠️  services/auth                78.5% (102/130 lines)
✅ shared                       88.7% (142/160 lines)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Overall Coverage: 86.4% (634/733 lines, 89/102 branches 87.3%)
✅ Coverage meets threshold 80.0%

📄 Reports generated:
   • Terminal summary (above)
   • HTML report: .coverage/coverage-report.html
   • JSON report: .coverage/coverage-report.json
   • Merged Cobertura: .coverage/coverage-merged.xml
```

### CI/CD Integration

The unified coverage report can be easily integrated with CI/CD systems:

#### GitHub Actions

```yaml
- name: Run tests with coverage
  run: azd app test --coverage --coverage-format=all --coverage-threshold=80

- name: Upload coverage to Codecov
  uses: codecov/codecov-action@v3
  with:
    files: .coverage/coverage-merged.xml
    flags: unittests
    name: codecov-umbrella
```

#### Azure DevOps

```yaml
- task: Bash@3
  displayName: 'Run tests with coverage'
  inputs:
    targetType: 'inline'
    script: 'azd app test --coverage --coverage-format=all'

- task: PublishCodeCoverageResults@1
  inputs:
    codeCoverageTool: 'Cobertura'
    summaryFileLocation: '.coverage/coverage-merged.xml'
    reportDirectory: '.coverage'
```

### Coverage Exclusions

Support for excluding files/directories from coverage:

**Configuration file: `.coveragerc` in workspace root**

```ini
[coverage]
exclude_dirs =
    **/node_modules/**
    **/dist/**
    **/build/**
    **/__tests__/**
    **/test/**
    **/*.test.*
    **/*.spec.*

exclude_patterns =
    **/migrations/**
    **/*_pb.ts
    **/*.config.js
```

Load and apply during coverage aggregation to avoid inflating uncovered lines.

### Benefits of This Approach

1. **Single Source of Truth**: One unified coverage report across all languages
2. **Standard Format**: Cobertura XML is industry-standard and widely supported
3. **Tool Compatibility**: Works with Azure DevOps, GitHub Actions, SonarQube, Codecov, Coveralls
4. **Developer-Friendly**: Beautiful terminal output with colors and emojis
5. **CI/CD Ready**: Easy integration with existing pipelines
6. **Threshold Enforcement**: Built-in support for failing builds on low coverage
7. **Detailed Insights**: HTML report provides interactive file-level analysis
8. **Machine-Readable**: JSON export for custom tooling and dashboards

```ini
[coverage]
exclude_dirs =
    **/node_modules/**
    **/dist/**
    **/build/**
    **/__tests__/**
    **/test/**
    **/*.test.*
    **/*.spec.*

exclude_patterns =
    **/migrations/**
    **/*_pb.ts
    **/*.config.js
```

Load and apply during coverage aggregation to avoid inflating uncovered lines.

### Benefits of This Approach

1. **Single Source of Truth**: One unified coverage report across all languages
2. **Standard Format**: Cobertura XML is industry-standard and widely supported
3. **Tool Compatibility**: Works with Azure DevOps, GitHub Actions, SonarQube, Codecov, Coveralls
4. **Developer-Friendly**: Beautiful terminal output with colors and emojis
5. **CI/CD Ready**: Easy integration with existing pipelines
6. **Threshold Enforcement**: Built-in support for failing builds on low coverage
7. **Detailed Insights**: HTML report provides interactive file-level analysis
8. **Machine-Readable**: JSON export for custom tooling and dashboards

---

## Local Testing Strategy

To ensure the `azd app test` command works correctly, we need comprehensive local testing with real projects across all supported languages.

### Test Project Structure

Create test fixture projects in `tests/projects/with-tests/`:

```
tests/projects/with-tests/
├── node-jest/                  # Node.js with Jest
│   ├── package.json
│   ├── jest.config.js
│   ├── src/
│   │   ├── math.js
│   │   └── user.js
│   └── __tests__/
│       ├── unit/
│       │   ├── math.test.js
│       │   └── user.test.js
│       └── e2e/
│           └── api.test.js
│
├── node-vitest/                # Node.js with Vitest
│   ├── package.json
│   ├── vitest.config.js
│   ├── src/
│   │   └── utils.ts
│   └── tests/
│       ├── unit/
│       │   └── utils.test.ts
│       └── integration/
│           └── service.test.ts
│
├── python-pytest/              # Python with pytest
│   ├── pyproject.toml
│   ├── pytest.ini
│   ├── src/
│   │   └── calculator.py
│   └── tests/
│       ├── unit/
│       │   └── test_calculator.py
│       └── e2e/
│           └── test_api.py
│
├── dotnet-xunit/               # .NET with xUnit
│   ├── src/
│   │   └── MyApp/
│   │       ├── MyApp.csproj
│   │       └── Services/
│   │           └── UserService.cs
│   └── tests/
│       └── MyApp.Tests/
│           ├── MyApp.Tests.csproj
│           └── UserServiceTests.cs
│
├── go-testing/                 # Go with standard testing
│   ├── go.mod
│   ├── calculator.go
│   ├── calculator_test.go
│   └── integration_test.go
│
└── polyglot-workspace/         # Mixed languages
    ├── frontend/               # Node.js
    ├── backend/                # Python
    ├── services/               # .NET
    └── shared/                 # Go
```

### Test Fixture Implementation

#### 1. Node.js Jest Project

**`tests/projects/with-tests/node-jest/package.json`**
```json
{
  "name": "test-node-jest-project",
  "version": "1.0.0",
  "scripts": {
    "test": "jest",
    "test:unit": "jest --testPathPattern=unit",
    "test:e2e": "jest --testPathPattern=e2e",
    "test:coverage": "jest --coverage"
  },
  "devDependencies": {
    "jest": "^29.7.0",
    "jest-junit": "^16.0.0",
    "@types/jest": "^29.5.0"
  },
  "jest": {
    "testEnvironment": "node",
    "coverageDirectory": "coverage",
    "collectCoverageFrom": ["src/**/*.js"],
    "reporters": [
      "default",
      ["jest-junit", {
        "outputDirectory": ".",
        "outputName": "junit.xml"
      }]
    ]
  }
}
```

**`tests/projects/with-tests/node-jest/src/math.js`**
```javascript
function add(a, b) {
    return a + b;
}

function subtract(a, b) {
    return a - b;
}

function multiply(a, b) {
    return a * b;
}

function divide(a, b) {
    if (b === 0) {
        throw new Error('Division by zero');
    }
    return a / b;
}

module.exports = { add, subtract, multiply, divide };
```

**`tests/projects/with-tests/node-jest/__tests__/unit/math.test.js`**
```javascript
const { add, subtract, multiply, divide } = require('../../src/math');

describe('Math Utils', () => {
    describe('add', () => {
        test('should add two positive numbers', () => {
            expect(add(2, 3)).toBe(5);
        });

        test('should handle negative numbers', () => {
            expect(add(-5, 3)).toBe(-2);
        });
    });

    describe('subtract', () => {
        test('should subtract two numbers', () => {
            expect(subtract(10, 4)).toBe(6);
        });
    });

    describe('multiply', () => {
        test('should multiply two numbers', () => {
            expect(multiply(3, 4)).toBe(12);
        });
    });

    describe('divide', () => {
        test('should divide two numbers', () => {
            expect(divide(10, 2)).toBe(5);
        });

        test('should throw error on division by zero', () => {
            expect(() => divide(10, 0)).toThrow('Division by zero');
        });
    });
});
```

#### 2. Python pytest Project

**`tests/projects/with-tests/python-pytest/pyproject.toml`**
```toml
[project]
name = "test-pytest-project"
version = "0.1.0"
description = "Test project for pytest"

[tool.pytest.ini_options]
testpaths = ["tests"]
python_files = ["test_*.py"]
python_classes = ["Test*"]
python_functions = ["test_*"]
markers = [
    "unit: Unit tests",
    "e2e: End-to-end tests",
    "integration: Integration tests"
]
addopts = "--strict-markers -v"

[tool.coverage.run]
source = ["src"]
omit = ["tests/*", "**/__pycache__/*"]

[tool.coverage.report]
exclude_lines = [
    "pragma: no cover",
    "def __repr__",
    "raise AssertionError",
    "raise NotImplementedError"
]
```

**`tests/projects/with-tests/python-pytest/src/calculator.py`**
```python
class Calculator:
    """Simple calculator with basic operations."""
    
    def add(self, a: float, b: float) -> float:
        """Add two numbers."""
        return a + b
    
    def subtract(self, a: float, b: float) -> float:
        """Subtract b from a."""
        return a - b
    
    def multiply(self, a: float, b: float) -> float:
        """Multiply two numbers."""
        return a * b
    
    def divide(self, a: float, b: float) -> float:
        """Divide a by b."""
        if b == 0:
            raise ValueError("Cannot divide by zero")
        return a / b
```

**`tests/projects/with-tests/python-pytest/tests/unit/test_calculator.py`**
```python
import pytest
from src.calculator import Calculator

@pytest.fixture
def calc():
    return Calculator()

@pytest.mark.unit
class TestCalculator:
    def test_add(self, calc):
        assert calc.add(2, 3) == 5
        assert calc.add(-1, 1) == 0
    
    def test_subtract(self, calc):
        assert calc.subtract(10, 4) == 6
        assert calc.subtract(0, 5) == -5
    
    def test_multiply(self, calc):
        assert calc.multiply(3, 4) == 12
        assert calc.multiply(-2, 3) == -6
    
    def test_divide(self, calc):
        assert calc.divide(10, 2) == 5
        assert calc.divide(9, 3) == 3
    
    def test_divide_by_zero(self, calc):
        with pytest.raises(ValueError, match="Cannot divide by zero"):
            calc.divide(10, 0)
```

#### 3. .NET xUnit Project

**`tests/projects/with-tests/dotnet-xunit/tests/MyApp.Tests/MyApp.Tests.csproj`**
```xml
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <IsPackable>false</IsPackable>
  </PropertyGroup>

  <ItemGroup>
    <PackageReference Include="Microsoft.NET.Test.Sdk" Version="17.8.0" />
    <PackageReference Include="xunit" Version="2.6.2" />
    <PackageReference Include="xunit.runner.visualstudio" Version="2.5.4" />
    <PackageReference Include="coverlet.collector" Version="6.0.0" />
    <PackageReference Include="JUnitTestLogger" Version="1.1.0" />
  </ItemGroup>

  <ItemGroup>
    <ProjectReference Include="..\..\src\MyApp\MyApp.csproj" />
  </ItemGroup>
</Project>
```

**`tests/projects/with-tests/dotnet-xunit/tests/MyApp.Tests/UserServiceTests.cs`**
```csharp
using Xunit;
using MyApp.Services;

namespace MyApp.Tests
{
    [Trait("Category", "Unit")]
    public class UserServiceTests
    {
        [Fact]
        public void ValidateEmail_WithValidEmail_ReturnsTrue()
        {
            var service = new UserService();
            var result = service.ValidateEmail("test@example.com");
            Assert.True(result);
        }

        [Fact]
        public void ValidateEmail_WithInvalidEmail_ReturnsFalse()
        {
            var service = new UserService();
            var result = service.ValidateEmail("invalid-email");
            Assert.False(result);
        }

        [Theory]
        [InlineData("user@domain.com", true)]
        [InlineData("user.name@domain.co.uk", true)]
        [InlineData("", false)]
        [InlineData("@domain.com", false)]
        public void ValidateEmail_WithVariousInputs_ReturnsExpected(string email, bool expected)
        {
            var service = new UserService();
            var result = service.ValidateEmail(email);
            Assert.Equal(expected, result);
        }
    }
}
```

#### 4. Go Testing Project

**`tests/projects/with-tests/go-testing/calculator_test.go`**
```go
package calculator

import "testing"

func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive numbers", 2, 3, 5},
        {"negative numbers", -5, -3, -8},
        {"mixed signs", 10, -4, 6},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, result, tt.expected)
            }
        })
    }
}

func TestDivide(t *testing.T) {
    result, err := Divide(10, 2)
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
    if result != 5 {
        t.Errorf("Divide(10, 2) = %d; want 5", result)
    }
}

func TestDivideByZero(t *testing.T) {
    _, err := Divide(10, 0)
    if err == nil {
        t.Error("expected error for division by zero")
    }
}
```

### Integration Test Implementation

**`src/cmd/app/commands/test_integration_test.go`**
```go
//go:build integration
// +build integration

package commands

import (
    "os"
    "path/filepath"
    "testing"
    
    "app/src/internal/tester"
)

func TestTestCommand_NodeJestProject(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    projectDir := filepath.Join("..", "..", "..", "tests", "projects", "with-tests", "node-jest")
    
    // Install dependencies first
    if err := os.Chdir(projectDir); err != nil {
        t.Fatalf("Failed to change directory: %v", err)
    }
    
    // Discover test projects
    projects, err := tester.FindTestProjects(projectDir)
    if err != nil {
        t.Fatalf("Failed to discover projects: %v", err)
    }
    
    if len(projects) != 1 {
        t.Fatalf("Expected 1 project, got %d", len(projects))
    }
    
    project := projects[0]
    if project.Type != "node" {
        t.Errorf("Expected type 'node', got '%s'", project.Type)
    }
    
    // Run unit tests
    opts := tester.TestOptions{
        Coverage: true,
        Verbose:  true,
    }
    
    result, err := tester.RunTests(project, tester.TestTypeUnit, opts)
    if err != nil {
        t.Fatalf("Test execution failed: %v", err)
    }
    
    if !result.Passed {
        t.Errorf("Tests failed: %s", result.Output)
    }
    
    if result.CoverageFile == "" {
        t.Error("Expected coverage file to be generated")
    }
}

func TestTestCommand_PythonPytestProject(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    projectDir := filepath.Join("..", "..", "..", "tests", "projects", "with-tests", "python-pytest")
    
    projects, err := tester.FindTestProjects(projectDir)
    if err != nil {
        t.Fatalf("Failed to discover projects: %v", err)
    }
    
    if len(projects) != 1 {
        t.Fatalf("Expected 1 project, got %d", len(projects))
    }
    
    project := projects[0]
    if project.Type != "python" {
        t.Errorf("Expected type 'python', got '%s'", project.Type)
    }
    
    opts := tester.TestOptions{
        Coverage: true,
    }
    
    result, err := tester.RunTests(project, tester.TestTypeUnit, opts)
    if err != nil {
        t.Fatalf("Test execution failed: %v", err)
    }
    
    if !result.Passed {
        t.Errorf("Tests failed")
    }
}

func TestTestCommand_PolyglotWorkspace(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    projectDir := filepath.Join("..", "..", "..", "tests", "projects", "with-tests", "polyglot-workspace")
    
    projects, err := tester.FindTestProjects(projectDir)
    if err != nil {
        t.Fatalf("Failed to discover projects: %v", err)
    }
    
    // Should find frontend (node), backend (python), services (dotnet), shared (go)
    if len(projects) < 3 {
        t.Errorf("Expected at least 3 projects, got %d", len(projects))
    }
    
    // Test parallel execution
    opts := tester.TestOptions{
        Coverage: true,
        Parallel: true,
    }
    
    results, err := tester.RunAllTests(projects, tester.TestTypeAll, opts)
    if err != nil {
        t.Fatalf("Test execution failed: %v", err)
    }
    
    passedCount := 0
    for _, result := range results {
        if result.Passed {
            passedCount++
        }
    }
    
    if passedCount != len(projects) {
        t.Errorf("Expected all projects to pass, got %d/%d", passedCount, len(projects))
    }
}

func TestCoverageAggregation(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Test coverage merging across different formats
    projectDir := filepath.Join("..", "..", "..", "tests", "projects", "with-tests", "polyglot-workspace")
    
    projects, _ := tester.FindTestProjects(projectDir)
    
    opts := tester.TestOptions{
        Coverage: true,
    }
    
    results, err := tester.RunAllTests(projects, tester.TestTypeAll, opts)
    if err != nil {
        t.Fatalf("Failed to run tests: %v", err)
    }
    
    // Collect coverage files
    var coverageFiles []string
    for _, result := range results {
        if result.CoverageFile != "" {
            coverageFiles = append(coverageFiles, result.CoverageFile)
        }
    }
    
    if len(coverageFiles) == 0 {
        t.Fatal("No coverage files generated")
    }
    
    // Generate unified report
    report, err := coverage.GenerateCoverageReport(coverageFiles, projectDir)
    if err != nil {
        t.Fatalf("Failed to generate coverage report: %v", err)
    }
    
    if report.TotalLines == 0 {
        t.Error("Expected coverage data to be collected")
    }
    
    if report.Percentage < 0 || report.Percentage > 100 {
        t.Errorf("Invalid coverage percentage: %.1f", report.Percentage)
    }
}
```

### Manual Testing Script

**`scripts/test-locally.ps1`**
```powershell
#!/usr/bin/env pwsh
# Test the azd app test command locally with fixture projects

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("node", "python", "dotnet", "go", "polyglot", "all")]
    [string]$ProjectType = "all",
    
    [switch]$Coverage,
    [switch]$Verbose
)

$ErrorActionPreference = "Stop"

Write-Host "🧪 Testing azd app test command locally" -ForegroundColor Cyan
Write-Host ""

# Build the extension first
Write-Host "🔨 Building extension..." -ForegroundColor Yellow
& "$PSScriptRoot\..\build.ps1"
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Build failed" -ForegroundColor Red
    exit 1
}

# Install locally
Write-Host "📦 Installing extension locally..." -ForegroundColor Yellow
& "$PSScriptRoot\..\install-local.ps1"
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Installation failed" -ForegroundColor Red
    exit 1
}

$testProjects = @()

if ($ProjectType -eq "all" -or $ProjectType -eq "node") {
    $testProjects += @{
        Name = "Node.js Jest"
        Path = "tests\projects\with-tests\node-jest"
        Setup = "npm install"
    }
}

if ($ProjectType -eq "all" -or $ProjectType -eq "python") {
    $testProjects += @{
        Name = "Python pytest"
        Path = "tests\projects\with-tests\python-pytest"
        Setup = "pip install -e . pytest pytest-cov"
    }
}

if ($ProjectType -eq "all" -or $ProjectType -eq "dotnet") {
    $testProjects += @{
        Name = ".NET xUnit"
        Path = "tests\projects\with-tests\dotnet-xunit"
        Setup = "dotnet restore"
    }
}

if ($ProjectType -eq "all" -or $ProjectType -eq "go") {
    $testProjects += @{
        Name = "Go testing"
        Path = "tests\projects\with-tests\go-testing"
        Setup = "go mod download"
    }
}

if ($ProjectType -eq "polyglot") {
    $testProjects += @{
        Name = "Polyglot Workspace"
        Path = "tests\projects\with-tests\polyglot-workspace"
        Setup = $null  # Multiple setup steps
    }
}

foreach ($project in $testProjects) {
    Write-Host ""
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
    Write-Host "Testing: $($project.Name)" -ForegroundColor Cyan
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
    
    $projectPath = Join-Path $PSScriptRoot ".." $project.Path
    
    if (-not (Test-Path $projectPath)) {
        Write-Host "⚠️  Project not found: $projectPath" -ForegroundColor Yellow
        continue
    }
    
    Push-Location $projectPath
    
    try {
        # Setup dependencies if needed
        if ($project.Setup) {
            Write-Host "📦 Setting up dependencies: $($project.Setup)" -ForegroundColor Yellow
            Invoke-Expression $project.Setup
        }
        
        # Build test command
        $testCmd = "azd app test"
        
        if ($Coverage) {
            $testCmd += " --coverage --coverage-format=all"
        }
        
        if ($Verbose) {
            $testCmd += " --verbose"
        }
        
        Write-Host ""
        Write-Host "Running: $testCmd" -ForegroundColor Green
        Write-Host ""
        
        Invoke-Expression $testCmd
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host ""
            Write-Host "✅ $($project.Name) tests passed!" -ForegroundColor Green
        } else {
            Write-Host ""
            Write-Host "❌ $($project.Name) tests failed!" -ForegroundColor Red
        }
    }
    finally {
        Pop-Location
    }
}

Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "✅ Local testing complete!" -ForegroundColor Green
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
```

### Running Local Tests

```bash
# Test all project types
.\scripts\test-locally.ps1 -ProjectType all -Coverage

# Test specific project type
.\scripts\test-locally.ps1 -ProjectType node -Coverage -Verbose

# Test polyglot workspace
.\scripts\test-locally.ps1 -ProjectType polyglot -Coverage

# Run Go integration tests
cd src
go test -tags=integration ./cmd/app/commands -v

# Run all tests including integration
go test -tags=integration ./... -v
```

---

## User Guide: Setting Up Projects for `azd app test`

This section provides guidance for users to configure their projects to work seamlessly with `azd app test`.

### Option 1: Follow Conventions (Recommended)

The easiest approach is to follow the naming conventions that `azd app test` expects. This requires no configuration.

#### Node.js Projects

**Add these scripts to `package.json`:**
```json
{
  "scripts": {
    "test": "jest",
    "test:unit": "jest --testPathPattern=unit",
    "test:e2e": "jest --testPathPattern=e2e",
    "test:integration": "jest --testPathPattern=integration"
  },
  "devDependencies": {
    "jest": "^29.7.0",
    "jest-junit": "^16.0.0"
  },
  "jest": {
    "reporters": ["default", "jest-junit"],
    "coverageReporters": ["text", "cobertura"]
  }
}
```

**Directory structure:**
```
your-project/
├── package.json
├── src/
└── __tests__/          or  tests/
    ├── unit/
    ├── e2e/
    └── integration/
```

**Commands that work:**
```bash
azd app test                    # Runs all tests
azd app test --type unit        # Runs test:unit script
azd app test --type e2e         # Runs test:e2e script
azd app test --coverage         # Runs with --coverage flag
```

#### Python Projects

**Add pytest configuration to `pyproject.toml`:**
```toml
[tool.pytest.ini_options]
testpaths = ["tests"]
markers = [
    "unit: Unit tests",
    "e2e: End-to-end tests",
    "integration: Integration tests"
]

[tool.coverage.run]
source = ["src"]

[tool.coverage.report]
skip_empty = true
```

**Directory structure:**
```
your-project/
├── pyproject.toml
├── src/
└── tests/
    ├── unit/
    ├── e2e/
    └── integration/
```

**Or use markers in tests:**
```python
import pytest

@pytest.mark.unit
def test_something():
    assert True

@pytest.mark.e2e
def test_end_to_end():
    assert True
```

**Commands that work:**
```bash
azd app test                    # pytest
azd app test --type unit        # pytest tests/unit OR pytest -m unit
azd app test --coverage         # pytest --cov --cov-report=xml
```

#### .NET Projects

**Add categories to your tests:**
```csharp
using Xunit;

public class MyTests
{
    [Fact]
    [Trait("Category", "Unit")]
    public void UnitTest()
    {
        Assert.True(true);
    }

    [Fact]
    [Trait("Category", "E2E")]
    public void E2ETest()
    {
        Assert.True(true);
    }
}
```

**Add test logger to `.csproj`:**
```xml
<ItemGroup>
  <PackageReference Include="JUnitTestLogger" Version="1.1.0" />
  <PackageReference Include="coverlet.collector" Version="6.0.0" />
</ItemGroup>
```

**Commands that work:**
```bash
azd app test                    # dotnet test
azd app test --type unit        # dotnet test --filter Category=Unit
azd app test --coverage         # dotnet test --collect:"XPlat Code Coverage"
```

#### Go Projects

**Use short tests for unit tests:**
```go
func TestUnit(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping test in short mode")
    }
    // ... test code
}

func TestE2E(t *testing.T) {
    // Name the function with E2E prefix or suffix
}
```

**Commands that work:**
```bash
azd app test                    # go test ./...
azd app test --type unit        # go test ./... -short
azd app test --type e2e         # go test ./... -run E2E
azd app test --coverage         # go test ./... -coverprofile=coverage.out
```

### Option 2: Custom Configuration

If you have existing test scripts with different names, use an `azd-test.yaml` configuration file.

**Create `azd-test.yaml` in your project root:**

```yaml
# azd-test.yaml - Custom test configuration
version: 1

# Define test commands for different test types
tests:
  # Unit tests
  unit:
    command: "npm run test:specs"  # Your custom script name
    coverage: "npm run coverage:unit"
    
  # E2E tests
  e2e:
    command: "npm run e2e:tests"
    coverage: "npm run e2e:coverage"
    
  # Integration tests
  integration:
    command: "npm run integration"
    coverage: "npm run integration:coverage"
    
  # Default test command (for azd app test with no --type flag)
  all:
    command: "npm run test:all"
    coverage: "npm run test:coverage"

# Coverage configuration
coverage:
  # Output format (auto-detected if not specified)
  format: "lcov"  # or "cobertura", "go", "json"
  
  # Coverage file location (auto-detected if not specified)
  file: "coverage/lcov.info"
  
  # Exclude patterns
  exclude:
    - "**/node_modules/**"
    - "**/*.test.js"
    - "**/*.spec.js"

# Test results configuration
results:
  # Result format (auto-detected if not specified)
  format: "junit"  # or "trx", "json"
  
  # Result file location (auto-detected if not specified)
  file: "test-results.xml"
```

**Python example:**
```yaml
version: 1

tests:
  unit:
    command: "make test-unit"  # Your custom command
    coverage: "make test-unit-coverage"
    
  e2e:
    command: "make test-e2e"
    coverage: "make test-e2e-coverage"
    
  all:
    command: "make test"
    coverage: "make test-coverage"

coverage:
  format: "cobertura"
  file: "coverage.xml"
```

**.NET example:**
```yaml
version: 1

tests:
  unit:
    command: "dotnet test --filter Category=Unit"
    coverage: "dotnet test --filter Category=Unit --collect:\"XPlat Code Coverage\""
    
  e2e:
    command: "./run-e2e-tests.sh"  # Custom script
    
  all:
    command: "dotnet test"
    coverage: "dotnet test --collect:\"XPlat Code Coverage\""
```

**Using custom configuration:**
```bash
# azd app test will automatically detect azd-test.yaml
azd app test --type unit

# Override with specific config file
azd app test --config custom-test-config.yaml
```

### Option 3: Workspace-Level Configuration

For monorepos with multiple projects, create a workspace-level configuration:

**`azd-test-workspace.yaml` in workspace root:**
```yaml
version: 1

# Define project-specific configurations
projects:
  - name: frontend
    path: ./frontend
    config: ./frontend/azd-test.yaml
    
  - name: backend
    path: ./backend
    config: ./backend/azd-test.yaml
    
  - name: services/auth
    path: ./services/auth
    # Inline configuration
    tests:
      unit:
        command: "dotnet test --filter Category=Unit"
      e2e:
        command: "dotnet test --filter Category=E2E"

# Global settings
settings:
  # Run tests in parallel by default
  parallel: true
  
  # Fail fast on first error
  fail_fast: false
  
  # Coverage threshold for all projects
  coverage_threshold: 80
  
  # Output directory for reports
  output_dir: ".test-results"
```

**Usage:**
```bash
# Test all projects
azd app test

# Test specific project
azd app test --project frontend

# Test with workspace config
azd app test --workspace-config azd-test-workspace.yaml
```

### Migration Guide

If you're migrating from other test runners:

#### From npm test to azd app test

**Before:**
```bash
npm test
npm run test:unit
npm run test:e2e
npm run test:coverage
```

**After (using conventions):**
Rename your package.json scripts to match conventions:
```json
{
  "scripts": {
    "test": "jest",
    "test:unit": "jest --testPathPattern=unit",
    "test:e2e": "jest --testPathPattern=e2e"
  }
}
```

Then use:
```bash
azd app test
azd app test --type unit
azd app test --type e2e
azd app test --coverage
```

#### From pytest to azd app test

**Before:**
```bash
pytest tests/unit
pytest tests/e2e
pytest --cov --cov-report=html
```

**After (using directory structure):**
Keep your directory structure:
```
tests/
├── unit/
└── e2e/
```

Then use:
```bash
azd app test --type unit    # Automatically runs pytest tests/unit
azd app test --type e2e     # Automatically runs pytest tests/e2e
azd app test --coverage     # Runs pytest --cov --cov-report=xml
```

#### From dotnet test to azd app test

**Before:**
```bash
dotnet test
dotnet test --filter Category=Unit
dotnet test --collect:"XPlat Code Coverage"
```

**After (using test categories):**
Add `[Trait("Category", "Unit")]` to your tests, then:
```bash
azd app test
azd app test --type unit
azd app test --coverage
```

### Troubleshooting

#### Tests not discovered

**Problem**: `azd app test` says "No test projects found"

**Solutions**:
1. Ensure your project has test indicators:
   - Node.js: `package.json` with test scripts
   - Python: `pytest.ini`, `pyproject.toml`, or `tests/` directory
   - .NET: `*.csproj` files with test framework references
   - Go: `*_test.go` files

2. Check file naming:
   ```bash
   # Node.js
   __tests__/**/*.test.js
   tests/**/*.spec.js
   
   # Python
   tests/**/test_*.py
   **/test_*.py
   
   # .NET
   **/*.Tests.csproj
   **/Test*.cs
   
   # Go
   **/*_test.go
   ```

#### Coverage not collected

**Problem**: Coverage reports are empty or not generated

**Solutions**:
1. Install coverage tools:
   ```bash
   # Node.js
   npm install --save-dev jest
   
   # Python
   pip install pytest-cov
   
   # .NET
   dotnet add package coverlet.collector
   
   # Go (built-in)
   ```

2. Ensure output formats are correct:
   - Node.js: Add `--coverageReporters=cobertura` to jest config
   - Python: Use `--cov-report=xml`
   - .NET: Use `--collect:"XPlat Code Coverage"`

#### Custom scripts not recognized

**Problem**: Your test scripts have different names

**Solution**: Use `azd-test.yaml`:
```yaml
version: 1
tests:
  unit:
    command: "npm run my-custom-unit-tests"
  all:
    command: "npm run my-custom-all-tests"
```

### Best Practices

1. **Use Standard Names**: Follow ecosystem conventions when possible
2. **Organize by Type**: Separate unit/e2e/integration tests in directories
3. **Enable Coverage**: Add coverage reporters to your test configuration
4. **Use Markers/Categories**: Tag tests with their type for easy filtering
5. **Keep Tests Fast**: Unit tests should run quickly (<1 second each)
6. **Document Conventions**: Add a README.md explaining your test structure
7. **CI/CD Integration**: Use `--coverage-threshold` and `--fail-fast` in pipelines

### Example: Complete Project Setup

**Node.js project with full `azd app test` support:**

```json
{
  "name": "my-app",
  "scripts": {
    "test": "jest",
    "test:unit": "jest --testPathPattern=unit",
    "test:e2e": "jest --testPathPattern=e2e",
    "test:integration": "jest --testPathPattern=integration",
    "test:watch": "jest --watch",
    "test:coverage": "jest --coverage"
  },
  "devDependencies": {
    "jest": "^29.7.0",
    "jest-junit": "^16.0.0",
    "@testing-library/react": "^14.0.0"
  },
  "jest": {
    "testEnvironment": "node",
    "coverageDirectory": "coverage",
    "coverageReporters": ["text", "cobertura", "html"],
    "collectCoverageFrom": [
      "src/**/*.{js,jsx,ts,tsx}",
      "!src/**/*.test.{js,jsx,ts,tsx}",
      "!src/**/*.spec.{js,jsx,ts,tsx}"
    ],
    "reporters": [
      "default",
      ["jest-junit", {
        "outputDirectory": ".",
        "outputName": "junit.xml"
      }]
    ]
  }
}
```

**Directory structure:**
```
my-app/
├── package.json
├── jest.config.js
├── src/
│   ├── components/
│   ├── services/
│   └── utils/
└── __tests__/
    ├── unit/
    │   ├── components/
    │   ├── services/
    │   └── utils/
    ├── integration/
    │   └── api/
    └── e2e/
        └── flows/
```

**Usage:**
```bash
# Development
azd app test --type unit --watch

# CI/CD
azd app test --coverage --coverage-threshold=80 --fail-fast --report-format=all

# Quick check
azd app test --type unit

# Full suite
azd app test --coverage
```

---

## Future Enhancements

1. **Differential Coverage**: Show coverage changes vs. base branch
2. **Coverage Trends**: Track coverage over time with historical data
3. **Mutation Testing**: Integrate with Stryker, PITest for mutation coverage
4. **Smart Test Selection**: Run only tests affected by changed files (Git diff analysis)
5. **Flaky Test Detection**: Track test stability across runs with retry logic
6. **Performance Metrics**: Collect and report test execution time trends
7. **Cloud Test Reporting**: Direct integration with Azure Test Plans
8. **Container-based E2E**: Automatically spin up docker-compose for e2e tests
9. **Parallel Test Distribution**: Distribute tests across multiple machines
10. **Visual Regression Testing**: Integrate with Percy, Chromatic for UI tests
11. **Test Impact Analysis**: AI-powered prediction of test failures based on code changes
12. **Custom Reporters**: Plugin system for custom test reporters
13. **Test Data Management**: Built-in test data seeding and cleanup
14. **Multi-version Testing**: Run tests against multiple dependency versions

---

## References

### Official Documentation
- [Jest CLI Options](https://jestjs.io/docs/cli)
- [Vitest Documentation](https://vitest.dev/)
- [pytest Documentation](https://docs.pytest.org/)
- [pytest-cov Documentation](https://pytest-cov.readthedocs.io/)
- [dotnet test](https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-test)
- [xUnit Documentation](https://xunit.net/)
- [Go Testing Package](https://pkg.go.dev/testing)
- [Go Coverage](https://go.dev/blog/cover)

### Coverage Tools
- [Cobertura Format Specification](https://cobertura.github.io/cobertura/)
- [LCOV Format](https://linux.die.net/man/1/geninfo)
- [Istanbul (nyc) Documentation](https://istanbul.js.org/)
- [Coverlet for .NET](https://github.com/coverlet-coverage/coverlet)
- [Coverage.py](https://coverage.readthedocs.io/)

### CI/CD Integration
- [GitHub Actions - Test Reporter](https://github.com/marketplace/actions/test-reporter)
- [Azure Pipelines - Publish Test Results](https://learn.microsoft.com/en-us/azure/devops/pipelines/tasks/reference/publish-test-results-v2)
- [Codecov Documentation](https://docs.codecov.com/)
- [SonarQube Test Coverage](https://docs.sonarqube.org/latest/analysis/coverage/)

### Test Frameworks
- [Jest Runners](https://jestjs.io/docs/configuration#runner-string)
- [Playwright](https://playwright.dev/)
- [Cypress](https://www.cypress.io/)
- [pytest Plugins](https://docs.pytest.org/en/latest/reference/plugin_list.html)
- [NUnit](https://nunit.org/)
- [MSTest](https://learn.microsoft.com/en-us/dotnet/core/testing/unit-testing-with-mstest)

---

## Summary

The `azd app test` command provides a comprehensive, polyglot testing solution that:

### ✅ Key Capabilities
- **Polyglot Support**: Node.js, Python, .NET, Go, and more
- **Unified Coverage**: Aggregates coverage across all languages into single report
- **Multiple Test Types**: Unit, E2E, integration with ecosystem conventions
- **Parallel Execution**: Fast test runs across multiple projects
- **Rich Reporting**: Terminal, HTML, JSON, JUnit XML outputs
- **CI/CD Ready**: Coverage thresholds, fail-fast, standard formats
- **Flexible Configuration**: Convention-based or custom `azd-test.yaml`

### 🎯 Design Goals Achieved
1. ✅ **Convention over Configuration**: Works out-of-box with standard project structures
2. ✅ **Unified Interface**: Single command for all test types across languages
3. ✅ **Comprehensive Reporting**: Coverage + test results in multiple formats
4. ✅ **Developer-Friendly**: Beautiful terminal output with colors and emojis
5. ✅ **Enterprise-Ready**: Threshold enforcement, coverage aggregation, audit trails

### 🚀 Getting Started

**For users following conventions:**
```bash
# Just works with standard project structures
azd app test --coverage
```

**For users with custom setups:**
```yaml
# Create azd-test.yaml
version: 1
tests:
  unit:
    command: "npm run my-test-script"
```

**For monorepos:**
```bash
# Tests all projects, aggregates results
azd app test --coverage --coverage-threshold=80 --report-format=all
```

### 📦 Deliverables

The implementation includes:
1. **Core Detection**: `src/internal/tester/tester.go` - Project and test discovery
2. **Coverage Engine**: `src/internal/coverage/` - Format conversion and aggregation
3. **Report Generator**: `src/internal/coverage/reporter.go` - Multi-format output
4. **Test Runners**: Language-specific test execution with result collection
5. **Integration Tests**: Comprehensive test suite with fixtures
6. **User Documentation**: Setup guides, migration paths, troubleshooting
7. **Local Testing**: Scripts for testing the implementation locally

### 🎓 For Contributors

When implementing this feature:
1. Start with detection logic (`detector/` package patterns)
2. Build test runners for each language
3. Implement coverage converters (LCOV→Cobertura, Go→Cobertura)
4. Create merger for unified reports
5. Add comprehensive integration tests
6. Test locally with all fixture projects
7. Validate CI/CD integration scenarios

This specification provides a complete blueprint for implementing a production-ready, polyglot testing solution that brings order to chaos in multi-language projects! 🎉
