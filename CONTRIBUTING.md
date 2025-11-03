# Contributing to azd-app

Thank you for your interest in contributing to azd-app! This document provides guidelines for contributing to the project.

## Getting Started

1. **Fork the repository** and clone your fork
2. **Set up your development environment** based on the component you're working on:

### CLI Extension

```bash
# Navigate to CLI directory
cd cli

# Install Go 1.21 or later
go version

# Install dependencies
go mod download

# Build the extension
go build -o bin/app.exe ./src/cmd/app
   ```

   > **Note:** If you encounter an error like "Operation did not complete successfully because the file contains a virus or potentially unwanted software", you need to exclude `go.exe` from Windows Defender or your antivirus software. This is a common false positive when Go builds executables. See [Windows Defender exclusions](https://support.microsoft.com/en-us/windows/add-an-exclusion-to-windows-security-811816c0-4dfd-af4a-47e4-c301afe13b26) for instructions.

3. **Install locally for testing**:
   ```bash
   cd cli
   .\install-local.ps1
   ```

## Development Workflow

### 1. Create a Branch
```bash
git checkout -b feature/your-feature-name
```

### 2. Make Changes
- Follow Go code conventions
- Run `gofmt` to format your code
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes
```bash
# Run unit tests
go test ./...

# Run with coverage
go test -cover ./...

# Test locally
azd app <your-command>
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

1. Use the command generator:
   ```bash
   .\new-command.ps1 your-command
   ```

2. Implement the command logic in the generated file

3. Register in `main.go`:
   ```go
   rootCmd.AddCommand(newYourCommand())
   ```

4. Add tests in `cmd_your_command_test.go`

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
- [ ] All tests pass: `go test ./...`
- [ ] Code coverage is at least 80%: `go test -cover ./...`
- [ ] Linter passes: `golangci-lint run`
- [ ] Code is formatted: `gofmt -l .` returns nothing
- [ ] Documentation is updated
- [ ] Commit messages follow Conventional Commits

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
