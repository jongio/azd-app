# Testing Framework Design

## Overview

This document outlines the complete architecture and design for the `azd app test` command suite, including test execution, coverage aggregation, and reporting across multi-language projects.

## Goals

1. **Unified Testing Experience**: Single command to test entire application
2. **Multi-Language Support**: Node.js, Python, .NET with framework detection
3. **Test Type Separation**: Independent execution of unit, integration, e2e tests
4. **Aggregated Coverage**: Unified coverage reporting across all services
5. **CI/CD Ready**: Integration with GitHub Actions, Azure Pipelines, etc.
6. **Developer Friendly**: Fast feedback, watch mode, clear error messages

## Commands

### Primary Commands

| Command | Purpose | Example |
|---------|---------|---------|
| `azd app test` | Run all tests | `azd app test --coverage` |
| `azd app test --type unit` | Run unit tests only | `azd app test --type unit --watch` |
| `azd app test --type integration` | Run integration tests | `azd app test --type integration --service api` |
| `azd app test --type e2e` | Run end-to-end tests | `azd app test --type e2e` |

### Supporting Commands

| Command | Purpose | Example |
|---------|---------|---------|
| `azd app coverage` | Generate coverage report | `azd app coverage --threshold 80` |
| `azd app coverage --merge` | Merge existing coverage reports | `azd app coverage --merge` |

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     azd app test                             │
│                  (Command Entry Point)                       │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   Test Orchestrator                          │
│  - Parses azure.yaml                                         │
│  - Detects test frameworks                                   │
│  - Manages parallel/sequential execution                     │
│  - Aggregates results                                        │
└─────────────────────────────────────────────────────────────┘
                            ↓
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Node.js    │     │   Python    │     │    .NET     │
│  Test       │     │   Test      │     │    Test     │
│  Runner     │     │   Runner    │     │    Runner   │
└─────────────┘     └─────────────┘     └─────────────┘
        │                   │                   │
        └───────────────────┼───────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                  Coverage Aggregator                         │
│  - Collects coverage from all services                       │
│  - Converts to common format (Cobertura)                     │
│  - Merges coverage data                                      │
│  - Generates unified reports                                 │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   Report Generator                           │
│  - Console output (default, JSON, JUnit)                     │
│  - Coverage reports (HTML, XML, JSON)                        │
│  - CI/CD integration (GitHub Actions, Azure Pipelines)       │
└─────────────────────────────────────────────────────────────┘
```

### Component Details

#### 1. Test Orchestrator

**Responsibilities:**
- Parse `azure.yaml` to get service configurations
- Detect test frameworks if not configured
- Determine test commands for each service
- Manage parallel vs sequential execution
- Collect and aggregate results
- Handle errors and timeouts

**Key Functions:**
```go
// TestOrchestrator manages test execution across services
type TestOrchestrator struct {
    config       *TestConfig
    services     []ServiceConfig
    parallel     bool
    failFast     bool
    resultsChan  chan TestResult
}

func (o *TestOrchestrator) Run(testType string) error
func (o *TestOrchestrator) DetectTests(serviceDir string) (*ServiceTestConfig, error)
func (o *TestOrchestrator) ExecuteTests(service ServiceConfig, testType string) (*TestResult, error)
func (o *TestOrchestrator) AggregateResults(results []TestResult) *AggregateResult
```

#### 2. Language-Specific Test Runners

##### Node.js Test Runner

**Responsibilities:**
- Detect framework (Jest, Vitest, Mocha)
- Execute tests with appropriate commands
- Parse test output
- Collect coverage

**Key Functions:**
```go
type NodeTestRunner struct {
    framework    string  // "jest", "vitest", "mocha"
    packageMgr   string  // "npm", "pnpm", "yarn"
    projectDir   string
}

func (r *NodeTestRunner) DetectFramework() (string, error)
func (r *NodeTestRunner) RunTests(testType string, coverage bool) (*TestResult, error)
func (r *NodeTestRunner) ParseOutput(output string) (*TestResult, error)
func (r *NodeTestRunner) CollectCoverage() (*CoverageData, error)
```

**Framework Detection:**
```
Check for configuration files:
  jest.config.js, jest.config.ts     → Jest
  vitest.config.ts, vitest.config.js → Vitest
  .mocharc.js, .mocharc.json         → Mocha

Check package.json dependencies:
  "jest": "*"                        → Jest
  "vitest": "*"                      → Vitest
  "mocha": "*"                       → Mocha

Default: npm test (whatever is configured in package.json)
```

##### Python Test Runner

**Responsibilities:**
- Detect framework (pytest, unittest)
- Manage virtual environment
- Execute tests with markers
- Collect coverage

**Key Functions:**
```go
type PythonTestRunner struct {
    framework    string  // "pytest", "unittest"
    packageMgr   string  // "pip", "poetry", "uv"
    projectDir   string
    venvPath     string
}

func (r *PythonTestRunner) DetectFramework() (string, error)
func (r *PythonTestRunner) ActivateVenv() error
func (r *PythonTestRunner) RunTests(testType string, coverage bool) (*TestResult, error)
func (r *PythonTestRunner) ParseOutput(output string) (*TestResult, error)
func (r *PythonTestRunner) CollectCoverage() (*CoverageData, error)
```

**Framework Detection:**
```
Check for configuration files:
  pytest.ini                         → pytest
  pyproject.toml with [tool.pytest]  → pytest
  
Check for test directories:
  tests/ with test_*.py              → pytest or unittest
  
Default: pytest (most common)
```

##### .NET Test Runner

**Responsibilities:**
- Detect framework (xUnit, NUnit, MSTest)
- Find test projects
- Execute tests with filters
- Collect coverage

**Key Functions:**
```go
type DotnetTestRunner struct {
    framework    string  // "xunit", "nunit", "mstest"
    testProjects []string
    projectDir   string
}

func (r *DotnetTestRunner) DetectFramework() (string, error)
func (r *DotnetTestRunner) FindTestProjects() ([]string, error)
func (r *DotnetTestRunner) RunTests(testType string, coverage bool) (*TestResult, error)
func (r *DotnetTestRunner) ParseOutput(output string) (*TestResult, error)
func (r *DotnetTestRunner) CollectCoverage() (*CoverageData, error)
```

**Framework Detection:**
```
Scan *.csproj files for package references:
  <PackageReference Include="xunit" /> → xUnit
  <PackageReference Include="NUnit" /> → NUnit
  <PackageReference Include="MSTest" />→ MSTest

Look for test project naming:
  *.Tests.csproj, *.Test.csproj
  
Find test projects in:
  tests/, Tests/, test/, Test/
```

#### 3. Coverage Aggregator

**Responsibilities:**
- Collect coverage from all services
- Convert to common format (Cobertura XML)
- Merge coverage data
- Calculate aggregate metrics
- Generate unified reports

**Key Functions:**
```go
type CoverageAggregator struct {
    serviceCoverage map[string]*CoverageData
    threshold       float64
    outputDir       string
}

func (a *CoverageAggregator) AddCoverage(service string, data *CoverageData) error
func (a *CoverageAggregator) ConvertToCobertura(data *CoverageData, format string) (*CoberturaCoverage, error)
func (a *CoverageAggregator) Merge() (*AggregateCoverage, error)
func (a *CoverageAggregator) GenerateReports() error
func (a *CoverageAggregator) CheckThreshold() (bool, error)
```

**Coverage Conversion:**
```
Input Formats:
  - Jest:    lcov.info (LCOV format)
  - Vitest:  coverage/coverage-final.json (Istanbul format)
  - pytest:  coverage.xml (Cobertura XML)
  - coverlet: coverage.cobertura.xml (Cobertura XML)

Conversion Process:
  1. Read coverage data in native format
  2. Parse coverage metrics (lines, branches, functions)
  3. Convert to Cobertura XML (common format)
  4. Merge all Cobertura files
  5. Generate HTML report from merged data
```

**Merge Algorithm:**
```
For each file covered:
  1. Collect all coverage entries for that file
  2. Calculate maximum coverage (union of covered lines)
  3. Merge branch coverage (union of covered branches)
  4. Update function coverage (union of covered functions)
  
Aggregate Metrics:
  Total Lines Covered = Sum(lines covered per service)
  Total Lines = Sum(total lines per service)
  Coverage % = (Total Lines Covered / Total Lines) * 100
```

#### 4. Report Generator

**Responsibilities:**
- Generate console output
- Create JSON/JUnit/GitHub formats
- Create HTML coverage reports
- Set CI/CD outputs

**Key Functions:**
```go
type ReportGenerator struct {
    format    string  // "default", "json", "junit", "github"
    outputDir string
}

func (g *ReportGenerator) GenerateTestReport(results *AggregateResult) error
func (g *ReportGenerator) GenerateCoverageReport(coverage *AggregateCoverage) error
func (g *ReportGenerator) GenerateHTML(coverage *AggregateCoverage) error
func (g *ReportGenerator) SetGitHubOutputs(results *AggregateResult, coverage *AggregateCoverage) error
```

## Data Structures

### Core Types

```go
// TestConfig represents the global test configuration
type TestConfig struct {
    Parallel          bool
    FailFast          bool
    CoverageThreshold float64
    OutputDir         string
    Verbose           bool
}

// ServiceConfig represents a service with test configuration
type ServiceConfig struct {
    Name     string
    Language string
    Dir      string
    Test     *ServiceTestConfig
}

// ServiceTestConfig represents test configuration for a service
type ServiceTestConfig struct {
    Framework string
    Unit      *TestTypeConfig
    Integration *TestTypeConfig
    E2E       *TestTypeConfig
    Coverage  *CoverageConfig
}

// TestTypeConfig represents configuration for a test type
type TestTypeConfig struct {
    Command  string
    Pattern  string
    Markers  []string
    Filter   string
    Projects []string
    Setup    []string  // Commands to run before tests
    Teardown []string  // Commands to run after tests
}

// CoverageConfig represents coverage configuration
type CoverageConfig struct {
    Enabled      bool
    Tool         string
    Threshold    float64
    OutputFormat string
    Source       string
    Exclude      []string
}

// TestResult represents the result of running tests for a service
type TestResult struct {
    Service      string
    TestType     string
    Passed       int
    Failed       int
    Skipped      int
    Total        int
    Duration     float64
    Failures     []TestFailure
    Coverage     *CoverageData
}

// TestFailure represents a single test failure
type TestFailure struct {
    Name       string
    Message    string
    StackTrace string
    File       string
    Line       int
}

// CoverageData represents coverage data for a service
type CoverageData struct {
    Lines        CoverageMetric
    Branches     CoverageMetric
    Functions    CoverageMetric
    Files        map[string]*FileCoverage
}

// CoverageMetric represents a coverage metric
type CoverageMetric struct {
    Covered int
    Total   int
    Percent float64
}

// FileCoverage represents coverage for a single file
type FileCoverage struct {
    Path         string
    Lines        CoverageMetric
    Branches     CoverageMetric
    Functions    CoverageMetric
    CoveredLines []int
}

// AggregateResult represents aggregated test results
type AggregateResult struct {
    Services  []*TestResult
    Passed    int
    Failed    int
    Skipped   int
    Total     int
    Duration  float64
    Success   bool
}

// AggregateCoverage represents aggregated coverage across services
type AggregateCoverage struct {
    Services  map[string]*CoverageData
    Aggregate *CoverageData
    Threshold float64
    Met       bool
}
```

## File Structure

### New Files to Create

```
cli/
├── src/
│   ├── cmd/app/commands/
│   │   ├── test.go              # Main test command
│   │   ├── test_test.go         # Unit tests
│   │   ├── coverage.go          # Coverage command (optional)
│   │   └── coverage_test.go
│   └── internal/
│       ├── testing/
│       │   ├── orchestrator.go       # Test orchestration
│       │   ├── orchestrator_test.go
│       │   ├── node_runner.go        # Node.js test runner
│       │   ├── node_runner_test.go
│       │   ├── python_runner.go      # Python test runner
│       │   ├── python_runner_test.go
│       │   ├── dotnet_runner.go      # .NET test runner
│       │   ├── dotnet_runner_test.go
│       │   ├── coverage.go           # Coverage aggregation
│       │   ├── coverage_test.go
│       │   ├── reporter.go           # Report generation
│       │   ├── reporter_test.go
│       │   └── types.go              # Type definitions
│       └── types/
│           └── types.go              # Update with test types
├── docs/
│   ├── commands/
│   │   ├── test.md              # Test command spec (created)
│   │   └── coverage.md          # Coverage command spec
│   └── design/
│       └── testing-framework.md # This document
└── tests/
    └── projects/
        ├── node/
        │   └── test-jest-project/    # Test fixture
        ├── python/
        │   └── test-pytest-project/  # Test fixture
        └── dotnet/
            └── test-xunit-project/   # Test fixture
```

## Implementation Plan

### Phase 1: Core Infrastructure (Week 1)

1. **Create type definitions** (`internal/testing/types.go`)
   - Define all core data structures
   - Add to `internal/types/types.go` as needed

2. **Test Orchestrator** (`internal/testing/orchestrator.go`)
   - Parse azure.yaml for test configurations
   - Detect test frameworks
   - Manage execution flow
   - Aggregate results

3. **Basic Test Command** (`cmd/app/commands/test.go`)
   - Command definition and flags
   - Integration with orchestrator
   - Basic output formatting

### Phase 2: Language Support (Week 2)

4. **Node.js Test Runner** (`internal/testing/node_runner.go`)
   - Framework detection (Jest, Vitest, Mocha)
   - Test execution
   - Output parsing
   - Coverage collection

5. **Python Test Runner** (`internal/testing/python_runner.go`)
   - Framework detection (pytest, unittest)
   - Virtual environment handling
   - Test execution with markers
   - Coverage collection

6. **.NET Test Runner** (`internal/testing/dotnet_runner.go`)
   - Framework detection (xUnit, NUnit, MSTest)
   - Test project discovery
   - Test execution with filters
   - Coverage collection

### Phase 3: Coverage Aggregation (Week 3)

7. **Coverage Aggregator** (`internal/testing/coverage.go`)
   - Format conversion (LCOV, Cobertura, Istanbul)
   - Coverage merging
   - Aggregate metrics calculation

8. **Report Generator** (`internal/testing/reporter.go`)
   - Console output (default)
   - JSON output
   - JUnit XML output
   - HTML coverage reports

### Phase 4: Advanced Features (Week 4)

9. **Watch Mode**
   - File system monitoring
   - Incremental test execution
   - Smart test selection

10. **Parallel Execution**
    - Concurrent service testing
    - Resource management
    - Error handling

11. **CI/CD Integration**
    - GitHub Actions format
    - Azure Pipelines integration
    - Output formatting

### Phase 5: Testing & Documentation (Week 5)

12. **Unit Tests**
    - Test each component
    - Mock external dependencies
    - Achieve 80%+ coverage

13. **Integration Tests**
    - Test with real projects
    - Verify framework detection
    - Validate coverage aggregation

14. **Documentation**
    - Command reference
    - Configuration examples
    - Troubleshooting guide

## Auto-Detection Logic

### Node.js Detection

```go
func DetectNodeTestFramework(projectDir string) (string, error) {
    // 1. Check for config files
    if exists("jest.config.js") || exists("jest.config.ts") {
        return "jest", nil
    }
    if exists("vitest.config.ts") || exists("vitest.config.js") {
        return "vitest", nil
    }
    if exists(".mocharc.js") || exists(".mocharc.json") {
        return "mocha", nil
    }
    
    // 2. Check package.json dependencies
    pkg := readPackageJSON(projectDir)
    if pkg.DevDependencies["jest"] != "" {
        return "jest", nil
    }
    if pkg.DevDependencies["vitest"] != "" {
        return "vitest", nil
    }
    if pkg.DevDependencies["mocha"] != "" {
        return "mocha", nil
    }
    
    // 3. Default to npm test
    return "npm", nil
}

func DetectNodeTestCommands(framework, testType string) (string, error) {
    switch framework {
    case "jest":
        switch testType {
        case "unit":
            return "npm test -- --testPathPattern=unit", nil
        case "integration":
            return "npm test -- --testPathPattern=integration", nil
        case "e2e":
            return "npm test -- --testPathPattern=e2e", nil
        default:
            return "npm test", nil
        }
    case "vitest":
        switch testType {
        case "unit":
            return "npm test -- --run --testNamePattern=unit", nil
        // ... similar for other types
        }
    }
}
```

### Python Detection

```go
func DetectPythonTestFramework(projectDir string) (string, error) {
    // 1. Check for pytest config
    if exists("pytest.ini") {
        return "pytest", nil
    }
    
    // 2. Check pyproject.toml
    if exists("pyproject.toml") {
        content := readFile("pyproject.toml")
        if strings.Contains(content, "[tool.pytest]") {
            return "pytest", nil
        }
    }
    
    // 3. Check for test directory
    if exists("tests/") || exists("test/") {
        // Look for pytest markers or imports
        files := findFiles("tests/", "test_*.py")
        for _, file := range files {
            content := readFile(file)
            if strings.Contains(content, "import pytest") {
                return "pytest", nil
            }
        }
    }
    
    // Default to pytest (most common)
    return "pytest", nil
}

func DetectPythonTestCommands(framework, testType string) (string, error) {
    switch framework {
    case "pytest":
        switch testType {
        case "unit":
            return "pytest tests/unit -v", nil
        case "integration":
            return "pytest tests/integration -v", nil
        case "e2e":
            return "pytest tests/e2e -v", nil
        default:
            return "pytest -v", nil
        }
    case "unittest":
        return "python -m unittest discover", nil
    }
}
```

### .NET Detection

```go
func DetectDotnetTestFramework(projectDir string) (string, error) {
    testProjects := findFiles(projectDir, "*.Tests.csproj")
    
    for _, proj := range testProjects {
        content := readFile(proj)
        
        // Check package references
        if strings.Contains(content, `PackageReference Include="xunit"`) {
            return "xunit", nil
        }
        if strings.Contains(content, `PackageReference Include="NUnit"`) {
            return "nunit", nil
        }
        if strings.Contains(content, `PackageReference Include="MSTest"`) {
            return "mstest", nil
        }
    }
    
    // Default to xUnit (most common for modern .NET)
    return "xunit", nil
}

func DetectDotnetTestCommands(framework, testType string) (string, error) {
    switch testType {
    case "unit":
        return "dotnet test --filter Category=Unit", nil
    case "integration":
        return "dotnet test --filter Category=Integration", nil
    case "e2e":
        return "dotnet test --filter Category=E2E", nil
    default:
        return "dotnet test", nil
    }
}
```

## Coverage Conversion

### LCOV to Cobertura (Jest)

```go
func ConvertLCOVToCobertura(lcovPath string) (*CoberturaCoverage, error) {
    // Parse LCOV format
    lcov := parseLCOV(lcovPath)
    
    // Convert to Cobertura structure
    cov := &CoberturaCoverage{
        LineRate:   lcov.CalculateLineRate(),
        BranchRate: lcov.CalculateBranchRate(),
        Packages:   make([]*CoberturaPackage, 0),
    }
    
    // Group files by package (directory)
    packages := groupFilesByPackage(lcov.Files)
    
    for pkgName, files := range packages {
        pkg := &CoberturaPackage{
            Name:    pkgName,
            Classes: make([]*CoberturaClass, 0),
        }
        
        for _, file := range files {
            class := convertFileToClass(file)
            pkg.Classes = append(pkg.Classes, class)
        }
        
        cov.Packages = append(cov.Packages, pkg)
    }
    
    return cov, nil
}
```

### Coverage Merging

```go
func MergeCoverage(coverageFiles []*CoberturaCoverage) (*AggregateCoverage, error) {
    merged := &AggregateCoverage{
        Services:  make(map[string]*CoverageData),
        Aggregate: &CoverageData{
            Files: make(map[string]*FileCoverage),
        },
    }
    
    totalLines := 0
    coveredLines := 0
    totalBranches := 0
    coveredBranches := 0
    
    for _, cov := range coverageFiles {
        // Add service coverage
        service := extractServiceName(cov)
        merged.Services[service] = extractCoverageData(cov)
        
        // Merge into aggregate
        for _, pkg := range cov.Packages {
            for _, class := range pkg.Classes {
                filePath := class.Filename
                
                if existing, ok := merged.Aggregate.Files[filePath]; ok {
                    // File already covered, merge coverage
                    existing = mergeFileCoverage(existing, class)
                } else {
                    // New file
                    merged.Aggregate.Files[filePath] = convertClassToFileCoverage(class)
                }
                
                totalLines += class.Lines.Total
                coveredLines += class.Lines.Covered
                totalBranches += class.Branches.Total
                coveredBranches += class.Branches.Covered
            }
        }
    }
    
    // Calculate aggregate metrics
    merged.Aggregate.Lines = CoverageMetric{
        Total:   totalLines,
        Covered: coveredLines,
        Percent: float64(coveredLines) / float64(totalLines) * 100,
    }
    
    merged.Aggregate.Branches = CoverageMetric{
        Total:   totalBranches,
        Covered: coveredBranches,
        Percent: float64(coveredBranches) / float64(totalBranches) * 100,
    }
    
    return merged, nil
}
```

## Error Handling

### Test Execution Errors

```go
type TestExecutionError struct {
    Service string
    Type    string
    Err     error
}

func (e *TestExecutionError) Error() string {
    return fmt.Sprintf("test execution failed for %s (%s): %v", 
        e.Service, e.Type, e.Err)
}

// Error handling in orchestrator
func (o *TestOrchestrator) ExecuteTests(service ServiceConfig, testType string) (*TestResult, error) {
    runner := o.getRunner(service.Language)
    
    result, err := runner.RunTests(testType, o.config.Coverage)
    if err != nil {
        if o.config.FailFast {
            return nil, &TestExecutionError{
                Service: service.Name,
                Type:    testType,
                Err:     err,
            }
        }
        // Continue on error, collect result
        result.Success = false
        result.Error = err.Error()
    }
    
    return result, nil
}
```

### Coverage Errors

```go
type CoverageError struct {
    Service string
    Err     error
}

func (e *CoverageError) Error() string {
    return fmt.Sprintf("coverage generation failed for %s: %v", 
        e.Service, e.Err)
}

// Graceful degradation for coverage
func (a *CoverageAggregator) AddCoverage(service string, data *CoverageData) error {
    if data == nil {
        log.Printf("Warning: no coverage data for %s, continuing...", service)
        return nil
    }
    
    a.serviceCoverage[service] = data
    return nil
}
```

## Performance Considerations

### Parallel Execution

```go
func (o *TestOrchestrator) RunParallel(services []ServiceConfig, testType string) ([]*TestResult, error) {
    resultsChan := make(chan *TestResult, len(services))
    errorsChan := make(chan error, len(services))
    
    // Create semaphore for concurrency control
    sem := make(chan struct{}, runtime.NumCPU())
    
    var wg sync.WaitGroup
    
    for _, service := range services {
        wg.Add(1)
        go func(svc ServiceConfig) {
            defer wg.Done()
            
            // Acquire semaphore
            sem <- struct{}{}
            defer func() { <-sem }()
            
            result, err := o.ExecuteTests(svc, testType)
            if err != nil {
                errorsChan <- err
                return
            }
            resultsChan <- result
        }(service)
    }
    
    // Wait for all tests to complete
    go func() {
        wg.Wait()
        close(resultsChan)
        close(errorsChan)
    }()
    
    // Collect results
    results := make([]*TestResult, 0, len(services))
    for result := range resultsChan {
        results = append(results, result)
    }
    
    // Check for errors
    var firstError error
    for err := range errorsChan {
        if firstError == nil {
            firstError = err
        }
    }
    
    if firstError != nil && o.config.FailFast {
        return results, firstError
    }
    
    return results, nil
}
```

### Caching

```go
// Cache test framework detection results
type FrameworkCache struct {
    cache map[string]string
    mu    sync.RWMutex
}

func (c *FrameworkCache) Get(projectDir string) (string, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    framework, ok := c.cache[projectDir]
    return framework, ok
}

func (c *FrameworkCache) Set(projectDir, framework string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.cache[projectDir] = framework
}
```

## Security Considerations

### Command Injection Prevention

```go
func (r *NodeTestRunner) RunTests(testType string, coverage bool) (*TestResult, error) {
    // Validate test type to prevent injection
    validTypes := map[string]bool{
        "unit":        true,
        "integration": true,
        "e2e":        true,
        "all":        true,
    }
    
    if !validTypes[testType] {
        return nil, fmt.Errorf("invalid test type: %s", testType)
    }
    
    // Use exec.Command with separated arguments (not shell)
    cmd := exec.Command(r.packageMgr, "test", "--", "--testPathPattern="+testType)
    
    // Don't use shell to prevent injection
    cmd.Dir = r.projectDir
    
    output, err := cmd.CombinedOutput()
    return r.ParseOutput(string(output)), err
}
```

### Path Validation

```go
func validateProjectPath(path string) error {
    // Ensure path is within project boundaries
    absPath, err := filepath.Abs(path)
    if err != nil {
        return err
    }
    
    projectRoot, err := findProjectRoot()
    if err != nil {
        return err
    }
    
    if !strings.HasPrefix(absPath, projectRoot) {
        return fmt.Errorf("path %s is outside project root", path)
    }
    
    return nil
}
```

## Future Enhancements

### 1. Test Sharding

Distribute tests across multiple machines for faster CI builds:

```bash
# Run shard 1 of 4
azd app test --shard 1/4

# In CI (GitHub Actions)
strategy:
  matrix:
    shard: [1, 2, 3, 4]
steps:
  - run: azd app test --shard ${{ matrix.shard }}/4
```

### 2. Test Impact Analysis

Run only tests affected by code changes:

```bash
# Run tests for files changed since last commit
azd app test --changed

# Run tests for files changed in PR
azd app test --since origin/main
```

### 3. Mutation Testing

Validate test quality by mutating code:

```bash
# Run mutation tests
azd app test --mutation

# Output: 85% mutation score (tests caught 85% of mutations)
```

### 4. Visual Regression Testing

For UI components:

```bash
# Capture and compare screenshots
azd app test --visual

# Update baseline screenshots
azd app test --visual --update-baseline
```

## Summary

This design provides a comprehensive testing framework for `azd app` that:

1. **Supports multiple languages** with automatic framework detection
2. **Separates test types** for fast feedback and focused testing
3. **Aggregates coverage** across all services for unified reporting
4. **Integrates with CI/CD** through multiple output formats
5. **Scales efficiently** with parallel execution and caching
6. **Maintains security** through proper input validation

The implementation follows the existing patterns in the codebase and integrates seamlessly with the command orchestration system.
