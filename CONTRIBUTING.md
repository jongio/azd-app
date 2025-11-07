# Contributing to azd-app

Thank you for your interest in contributing to azd-app! This document provides guidelines for contributing to the project.

## Getting Started

### Prerequisites

Before contributing, ensure you have the following installed:

- **Go**: 1.25 or later
- **Node.js**: 20.0.0 or later
- **npm**: 10.0.0 or later  
- **PowerShell**: 7.4 or later (recommended: 7.5.4 for full compatibility)
- **TypeScript**: 5.9.3 (installed via npm when building dashboard)
- **Azure Developer CLI (azd)**: Latest version

You can verify your versions:
```bash
go version                  # Should be 1.25+
node --version             # Should be v20.0.0+
npm --version              # Should be 10.0.0+
pwsh --version            # Should be 7.4+ or 7.5.4
tsc --version             # Should be 5.9.3 (after npm install in dashboard/)
azd version               # Should be latest
```

### Setup

1. **Fork the repository** and clone your fork
2. **Set up your development environment** based on the component you're working on:

### CLI Extension

```bash
# Navigate to CLI directory
cd cli

# Install Go dependencies
go mod download

# Install dashboard dependencies
cd dashboard
npm install
cd ..

# Build the extension
go build -o bin/app.exe ./src/cmd/app
   ```

   > **Note:** If you encounter an error like "Operation did not complete successfully because the file contains a virus or potentially unwanted software", you need to exclude `go.exe` from Windows Defender or your antivirus software. This is a common false positive when Go builds executables. See [Windows Defender exclusions](https://support.microsoft.com/en-us/windows/add-an-exclusion-to-windows-security-811816c0-4dfd-af4a-47e4-c301afe13b26) for instructions.

3. **Install locally for testing**:
   ```bash
   cd cli
   # Install mage if not already installed
   go install github.com/magefile/mage@latest
   
   # Build and install
   mage install
   ```

## Development Workflow

### Resetting to Local Development

If you've previously installed the extension from the registry and want to switch back to using your local development version:

```bash
# Uninstall the registry extension
azd extension uninstall jongio.azd.app

# Build and install your local version
azd x build --skip-install=false

# Start watching for changes
azd x watch
```

### Dashboard Development Workflow

The dashboard is a React/TypeScript application built with Vite. For dashboard development:

#### Option 1: Full Stack Development with Live Reload (Recommended)

Run both watchers in separate terminals for the complete development experience:

**Terminal 1 - Watch Go Backend:**
```bash
cd cli
mage watch
```

**Terminal 2 - Watch Dashboard:**
```bash
cd cli
mage dashboardWatch
```

This setup:
- **Terminal 1**: Watches Go files, rebuilds the extension, and reinstalls when Go code changes
- **Terminal 2**: Watches dashboard files and rebuilds the production bundle when frontend files change
- Both watchers run independently for maximum flexibility

#### Option 2: Dashboard Development with Mock Data

For rapid UI iteration without a running backend:

```bash
cd cli/dashboard
npm run dev
```

This starts the Vite dev server with hot module replacement (HMR) at `http://localhost:5173` and uses mock service data for the UI.

#### Option 3: One-Time Dashboard Build

If you just need to rebuild the dashboard once:

```bash
cd cli
mage dashboardbuild
```

### Recommended VS Code Settings

For the best development experience, add these settings to your `.vscode/settings.json`:

```json
{
  "go.lintFlags": ["--fast"],
  "go.lintTool": "golangci-lint",
  "go.vetOnSave": "package",
  "gopls": {
    "analyses": {
      "nilness": true,
      "shadow": true,
      "ST1003": true,
      "unusedparams": true,
      "unusedwrite": true,
      "useany": true
    },
    "staticcheck": true
  }
}
```

These settings enable:
- **nilness**: Catch potential nil pointer dereferences
- **shadow**: Find variable shadowing issues
- **ST1003**: Check for proper naming conventions
- **unusedparams**: Detect unused function parameters
- **unusedwrite**: Identify writes to variables that are never read
- **useany**: Suggest using `any` instead of `interface{}`
- **staticcheck**: Enable comprehensive static analysis

### 1. Create a Branch
```bash
git checkout -b feature/your-feature-name
```

### 2. Make Changes
- Follow Go code conventions
- Run `mage fmt` to format your code
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes
```bash
# Run unit tests
mage test

# Run with coverage
mage testcoverage

# Run all tests including integration
mage testall

# Run linter
mage lint

# Test locally
mage install
azd app <your-command>

# Alternative: Use go directly
go test ./...
go test -cover ./...
```

### 4. Commit Your Changes
```bash
git add .
git commit -m "feat: add support for X"
```

Follow [Conventional Commits](https://www.conventionalcommits.org/):
- `feat:` New features
- `fix:` Bug fixes
- `docs:` Documentation changes
- `test:` Adding or updating tests
- `refactor:` Code refactoring
- `chore:` Maintenance tasks

### 5. Push and Create Pull Request
```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub.

## Code Guidelines

### Go Style
- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting
- Run `golangci-lint run` before committing
- Keep functions small and focused
- Add comments for exported functions

### Testing
- Write tests for new functionality
- Aim for 80% code coverage minimum
- Use table-driven tests where appropriate
- Mock external dependencies (file system, exec.Command)

### Documentation
- Update README.md for user-facing changes
- Add/update docs/ files for new features
- Document non-obvious code with comments
- Update CHANGELOG.md for notable changes

## Project Structure

```
src/
├── cmd/              # Command implementations
├── internal/
│   ├── detector/     # Project detection logic
│   ├── installer/    # Dependency installation
│   └── runner/       # Project execution
└── types/            # Shared types

tests/
└── projects/         # Test fixtures

docs/                 # Documentation
```

## Adding a New Command

1. Create a new file in `src/cmd/app/commands/` (e.g., `your_command.go`)

2. Implement the command following the existing patterns (see `run.go`, `reqs.go`, etc.)

3. Register the command in `src/cmd/app/commands/root.go`:
   ```go
   rootCmd.AddCommand(newYourCommand())
   ```

4. Add tests in `src/cmd/app/commands/your_command_test.go`

5. Update documentation

## Adding Support for a New Package Manager

1. Add detection logic in `src/internal/detector/`
2. Add installation logic in `src/internal/installer/`
3. Create test project in `tests/projects/`
4. Add unit tests
5. Update documentation

## Testing with Test Projects

Use the test projects in `tests/projects/` for integration testing:

```bash
cd tests/projects/node/test-npm-project
azd app install
azd app run
```

Create new test projects with minimal dependencies for faster testing.

## Quality Gates

Before submitting a PR, ensure:
- [ ] All tests pass: `mage test` (or `go test ./...`)
- [ ] Code coverage is maintained: `mage testcoverage`
- [ ] Linter passes: `mage lint`
- [ ] Code is formatted: `mage fmt`
- [ ] Documentation is updated
- [ ] Commit messages follow Conventional Commits

**Recommended:** Run `mage preflight` to execute all quality checks at once.

## Pull Request Process

1. Update documentation with details of changes
2. Update CHANGELOG.md with notable changes
3. Ensure all tests pass and coverage meets requirements
4. Request review from maintainers
5. Address review feedback
6. Once approved, maintainer will merge

## Code Review Guidelines

When reviewing code:
- Check for test coverage
- Verify error handling
- Ensure documentation is clear
- Look for edge cases
- Confirm code follows Go conventions

## Getting Help

- Open an issue for bugs or feature requests
- Start a discussion for questions
- Check existing issues and documentation first

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
