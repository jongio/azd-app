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
   
   # First time setup: Add the local registry source (one-time)
   azd extension source add -n app -t file -l "<path-to-repo>/registry.json"
   # Example: azd extension source add -n app -t file -l "C:\code\azd-app\registry.json"
   
   # Install the extension from the local registry
   azd extension install jongio.azd.app --source app --force
   
   # Verify installation
   azd app version
   ```

   After the initial setup, you can rebuild and reinstall with:
   ```bash
   # Build and install extension
   mage build
   # or
   azd x build
   
   # Verify installation
   azd app version
   ```

4. **Development workflow with watch mode**:
   ```bash
   # For active development, use watch mode to auto-rebuild on file changes
   azd x watch
   ```

## Development Workflow

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
# Build and install for testing
azd x build --skip-install=false

# Or use watch mode during active development
azd x watch

# Test the extension
azd app <your-command>

# Run unit tests (legacy - use azd x build instead)
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
3. Create test project in `tests/projects/package-managers/`
4. Add unit tests
5. Update documentation

## Testing with Test Projects

Use the test projects in `tests/projects/` for integration testing:

```bash
cd tests/projects/package-managers/node/test-npm-project
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

## Website Screenshots

The documentation website uses automated screenshot capture for visual consistency, but these must be run manually (they are not part of CI/CD).

### Website Page Screenshots

To update screenshots of the documentation website pages:

```bash
cd web
pnpm install
pnpm run dev  # Start the dev server in another terminal
npx playwright test --update-snapshots --project=chromium
```

This captures screenshots of key pages in light/dark mode and at various viewport sizes.

### Dashboard Screenshots

To capture screenshots of the azd app dashboard for marketing purposes:

```bash
cd web
npx tsx scripts/capture-screenshots.ts
```

This script:
1. Starts the demo project with `azd app run`
2. Waits for services to be ready
3. Captures screenshots of different dashboard views
4. Saves optimized images to `web/public/screenshots/`

**Note:** Dashboard screenshots require a working `azd-app` binary in `cli/bin/`.

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
