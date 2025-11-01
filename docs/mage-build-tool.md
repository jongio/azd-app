# Mage Build Tool

This project uses [Mage](https://magefile.org/) for build automation instead of Make. Mage is a make/rake-like build tool using Go, which provides several advantages:

## Why Mage?

- **Pure Go**: Build scripts are Go code, not shell scripts
- **Cross-platform**: Works identically on Windows, Linux, and macOS without requiring WSL or MinGW
- **IDE support**: Full autocomplete, refactoring, and debugging for build scripts
- **Type safety**: Compile-time checking prevents runtime errors
- **No dependencies**: Users can run build scripts with `go run` even without installing mage

## Installation

```bash
# Install globally (recommended)
go install github.com/magefile/mage@latest

# Or use go run (no installation needed)
go run magefile.go <target>
```

## Available Targets

List all available targets:

```bash
mage -l
```

Output:
```
Targets:
  all*               runs lint, test, and build.
  build              compiles the app binary for the current platform.
  buildAll           builds for all platforms using build.ps1.
  clean              removes build artifacts and coverage reports.
  fmt                formats all Go code using gofmt.
  install            installs the extension locally using azd x build.
  lint               runs golangci-lint on the codebase.
  test               runs unit tests only (with -short flag).
  testAll            runs all tests (unit + integration).
  testCoverage       runs tests with coverage report.
  testIntegration    runs integration tests only.

* default target
```

## Common Tasks

### Build the Extension

```bash
# Current platform only
mage build

# All platforms (Windows, Linux, macOS for AMD64 and ARM64)
mage buildAll
```

### Run Tests

```bash
# Unit tests only (fast, no external dependencies)
mage test

# Integration tests only (requires external tools)
mage testIntegration

# All tests (unit + integration)
mage testAll

# Tests with coverage report
mage testCoverage
```

### Code Quality

```bash
# Run linter
mage lint

# Format code
mage fmt

# Run everything (lint, test, build)
mage all
# Or simply:
mage
```

### Install Locally

```bash
# Install extension to azd
mage install
```

### Clean Build Artifacts

```bash
mage clean
```

## Using Without Installing Mage

If you don't have mage installed globally, you can still use it:

```bash
# List targets
go run magefile.go -l

# Run a target
go run magefile.go build
go run magefile.go test
```

## Magefile Structure

The `magefile.go` at the project root defines all build targets as exported Go functions. Each function represents a target that can be executed.

```go
// Example: Build target
func Build() error {
    fmt.Println("Building app...")
    return sh.RunV("go", "build", "-o", "bin/app.exe", "./src/cmd/app")
}
```

### Dependencies Between Targets

Targets can depend on other targets using `mg.Deps()`:

```go
// All depends on Lint, Test, and Build
func All() error {
    mg.Deps(Lint, Test, Build)
    return nil
}
```

Mage automatically:
- Runs dependencies in the correct order
- Runs each dependency only once (memoization)
- Stops on first error

## Advantages Over Makefile

| Feature | Makefile | Magefile |
|---------|----------|----------|
| Cross-platform | ❌ Requires make/WSL on Windows | ✅ Pure Go, works everywhere |
| Language | Shell scripting | Go |
| IDE support | Limited | Full (autocomplete, refactoring, debugging) |
| Type safety | ❌ Runtime errors | ✅ Compile-time checking |
| Debugging | Difficult | Use Go debugger |
| Error handling | Basic | Rich error wrapping with context |
| Dependency management | Tab-sensitive syntax | Clean Go code |

## Learning More

- [Mage Documentation](https://magefile.org/)
- [Mage GitHub](https://github.com/magefile/mage)
- [Example Magefiles](https://github.com/magefile/mage/tree/master/examples)

## Troubleshooting

### "mage: command not found"

Install mage or use `go run magefile.go` instead:

```bash
go install github.com/magefile/mage@latest
```

### Mage not in PATH

Add Go bin directory to your PATH:

**Windows (PowerShell):**
```powershell
$env:Path += ";$env:USERPROFILE\go\bin"
```

**Linux/macOS:**
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### Need to pass arguments to a target

Mage doesn't support arguments to targets directly. Instead, modify the magefile.go to use environment variables:

```go
func Build() error {
    output := os.Getenv("OUTPUT_PATH")
    if output == "" {
        output = "bin/app.exe"
    }
    return sh.RunV("go", "build", "-o", output, "./src/cmd/app")
}
```

Then use:
```bash
OUTPUT_PATH=custom/path mage build
```
