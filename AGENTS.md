# AI Agent Guidelines

This document provides guidelines for AI agents (like GitHub Copilot) working with the App project.

## Project Overview

App is an Azure Developer CLI (azd) extension that automates development environment setup by detecting project types and running appropriate commands.

### Supported Project Types
- **Node.js**: npm, pnpm, yarn
- **Python**: uv, poetry, pip
- **.NET**: dotnet restore for .csproj and .sln
- **Aspire**: dotnet run for AppHost.cs projects
- **Docker Compose**: Detected from package.json scripts

## Repository Structure

```
src/
├── cmd/app/          # CLI entry point
│   └── commands/          # Command implementations (deps, run, reqs, listen)
│       ├── core.go        # Command orchestrator setup & core logic
│       ├── deps.go        # Deps command (depends on reqs)
│       ├── run.go         # Run command (depends on deps)
│       ├── reqs.go        # Reqs command (no dependencies)
│       └── listen.go      # Extension framework integration (azd lifecycle)
├── internal/
│   ├── detector/          # Project type & package manager detection
│   ├── installer/         # Dependency installation logic
│   ├── runner/            # Project execution logic
│   ├── executor/          # Safe command execution with context/timeout
│   ├── orchestrator/      # Command dependency chain management
│   ├── security/          # Input validation & sanitization
│   └── types/             # Shared type definitions
```

## Command Dependency Chain

Commands use an orchestrator pattern for dependency management:

```
run → deps → reqs
```

- **reqs**: Checks prerequisites (no dependencies)
- **deps**: Installs dependencies (depends on reqs)
- **run**: Starts dev environment (depends on deps)
- **listen**: Extension framework integration (hidden, invoked by azd)

The orchestrator provides:
- **Automatic dependency resolution**: Dependencies run in correct order
- **Memoization**: Each command runs only once per execution
- **Cycle detection**: Prevents infinite loops
- **Error propagation**: Failed dependencies prevent dependent commands

The `listen` command is required for proper azd extension framework integration. It establishes a bidirectional connection with azd and registers the extension's capabilities. This prevents azd from incorrectly trying to invoke the extension for unsupported operations (like service-target-provider).

See `docs/command-dependency-chain.md` for detailed design documentation.

## Code Patterns

### Detection Logic
- **File-based detection**: Use lock files and config files as markers
- **Priority order**: More specific package managers first (e.g., uv > poetry > pip)
- **Absolute paths**: Always resolve to absolute paths to avoid scanning parents
- **Error handling**: Return errors, don't panic

### Package Manager Detection
```go
// Lock file presence determines package manager
if exists("pnpm-lock.yaml") { return "pnpm" }
if exists("yarn.lock") { return "yarn" }
return "npm" // default
```

### Command Execution
- **ALWAYS use `executor.RunCommand()`** instead of raw `exec.Command()`
- Provides context-aware execution with 30-minute default timeout
- **Automatically inherits azd environment context** (AZD_SERVER, AZD_ACCESS_TOKEN, environment variables)
- Signature: `executor.RunCommand(name string, args []string, dir string) error`
- Example: `executor.RunCommand("npm", []string{"install"}, projectDir)`

### Environment Context Propagation
- **All spawned commands inherit azd environment variables** via `cmd.Env = os.Environ()`
- Critical variables set by azd:
  - `AZD_SERVER`: gRPC server address for extension ↔ azd communication
  - `AZD_ACCESS_TOKEN`: JWT token for authenticating gRPC requests
  - Environment-specific variables: Deployment context, Azure resources, configuration
- **Never manually create commands with `exec.Command()`** - always use executor package
- See `docs/azd-environment-context.md` for detailed documentation

### Security Validation
- **Path validation**: Use `security.ValidatePath()` before any file operations
  - Prevents path traversal attacks (blocks `..`)
  - Example: `if err := security.ValidatePath(filePath); err != nil { return err }`
- **Package manager validation**: Use `security.ValidatePackageManager()` 
  - Whitelist: npm, pnpm, yarn, pip, poetry, uv, dotnet
- **Script name sanitization**: Use `security.SanitizeScriptName()` for user input
  - Blocks shell metacharacters: `;`, `&`, `|`, `` ` ``, `$`, etc.
- **Error wrapping**: Always use `%w` in `fmt.Errorf()` for error chains

## Testing Strategy

- **Unit tests**: Table-driven tests with multiple scenarios
- **Coverage target**: 80% minimum (detector, runner, security packages)
- **Test file naming**: `*_test.go` in same package
- **Security tests**: Test attack vectors (path traversal, injection)
- **File permissions**: Use `0600` for files, `0750` for directories in tests
- **Test before commit**: ALWAYS run `go test ./...` after adding features
- **All tests must pass**: No commits with failing tests

## Common Tasks

### Adding a New Command
```bash
.\new-command.ps1 <command-name>
```

### Running Tests
```bash
go test ./...
go test -cover ./...
```

### Local Installation
```bash
.\install-local.ps1
azd app <command>
```

## Code Style

- Follow standard Go conventions (use `gofmt`)
- Run `golangci-lint` before committing (22 linters enabled including gosec)
- Keep functions focused and testable
- Document exported functions with godoc comments ending in periods

## Best Practices

1. **Detect before acting**: Always detect project type before running commands
2. **Fail gracefully**: Provide helpful error messages
3. **Support multiple package managers**: Don't assume npm/pip
4. **Test with real projects**: Use fixtures in tests/projects/
5. **Document decisions**: Add comments for non-obvious logic
6. **Validate inputs**: Use security package for all user-controlled inputs
7. **Safe execution**: Use executor package instead of raw exec.Command
8. **Write tests first**: For every new feature, write tests and ensure they pass before committing

## Extension Development

- **Extension manifest**: extension.yaml defines CLI integration
- **Local testing**: Use install-local.ps1 for quick iteration
- **azd integration**: Commands appear as `azd app <cmd>`
