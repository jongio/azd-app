# App Extension for Azure Developer CLI
# App Extension for Azure Developer CLI

[![CI](https://github.com/jongio/azd-app-extension/actions/workflows/ci.yml/badge.svg)](https://github.com/jongio/azd-app-extension/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jongio/azd-app-extension)](https://goreportcard.com/report/github.com/jongio/azd-app-extension)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

An Azure Developer CLI (azd) extension that automates development environment setup by detecting project types and running appropriate commands across multiple languages and frameworks.
[![CI](https://github.com/jongio/azd-app-extension/actions/workflows/ci.yml/badge.svg)](https://github.com/jongio/azd-app-extension/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jongio/azd-app-extension)](https://goreportcard.com/report/github.com/jongio/azd-app-extension)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

An Azure Developer CLI (azd) extension that automates development environment setup by detecting project types and running appropriate commands across multiple languages and frameworks.

## Overview

App automatically detects and manages dependencies for:
- **Node.js**: npm, pnpm, yarn
- **Python**: uv, poetry, pip (with automatic virtual environment setup)
- **.NET**: dotnet restore for projects and solutions
- **Aspire**: .NET Aspire application orchestration
- **Docker Compose**: Container orchestration

## Features

- 🔍 **Smart Detection**: Automatically identifies project types and package managers
- 📦 **Multi-Language Support**: Works with Node.js, Python, and .NET projects
- 🚀 **One-Command Setup**: Install all dependencies with a single command
- 🎯 **Environment-Aware**: Creates and manages virtual environments for Python
- ⚡ **Fast Iteration**: Minimal test dependencies for quick validation

## Installation

### For End Users

First, add the extension registry:

```bash
azd config set extension.registry https://raw.githubusercontent.com/jongio/azd-app-extension/main/registry.json
```

Then install the extension:

```bash
azd extension install app
```

Or install from a specific version:

```bash
azd extension install app --version 0.1.0
```

To uninstall:

```bash
azd extension uninstall app
```

### For Development & Testing

**Quick Start - Recommended Method:**

```powershell
# Clone and navigate to the project
git clone https://github.com/jongio/azd-app-extension.git
cd azd-app-extension

# Run the installation script (builds and installs locally)
.\install-local.ps1

# Test it!
azd app reqs
```

**Development Workflow:**

After making changes to your code:

```powershell
# Option 1: Quick rebuild with install script
.\install-local.ps1

# Option 2: Use mage (cross-platform build tool)
mage build
mage install

# Option 3: Build manually
go build -o bin/app.exe ./src/cmd/app  # Windows
go build -o bin/app ./src/cmd/app      # Linux/macOS
```

**Uninstalling Development Build:**

```powershell
.\install-local.ps1 -Uninstall
```

### Prerequisites

- [Azure Developer CLI (azd)](https://learn.microsoft.com/azure/developer/azure-developer-cli/install-azd) installed
- Go 1.21 or later (for building from source)
- PowerShell (for Windows development)

## Quick Start

Once installed, you can use these commands:

### `azd app reqs`

Verifies that all required tools are installed and optionally checks if they are running.

```bash
azd app reqs
```

**Features:**
- ✅ Checks if required tools are installed
- ✅ Validates minimum version requirements
- ✅ Verifies if services are running (e.g., Docker daemon)
- ✅ Supports custom tool configurations
- ✅ Built-in support for Node.js, Python, .NET, Aspire, Docker, Azure CLI, and more

**Example azure.yaml configuration:**

```yaml
name: my-project
reqs:
  - id: docker
    minVersion: "20.0.0"
    checkRunning: true
  - id: nodejs
    minVersion: "20.0.0"
  - id: python
    minVersion: "3.12.0"
```

**Output:**
```
🔍 Checking requirements...

✅ docker: 24.0.5 (required: 20.0.0) - ✅ RUNNING
✅ nodejs: 22.19.0 (required: 20.0.0)
✅ python: 3.13.9 (required: 3.12.0)

✅ All requirements are satisfied!
```

See [docs/reqs-command.md](docs/reqs-command.md) for detailed documentation.

### `azd app deps`

Automatically detects your project type and installs all dependencies.

```bash
azd app deps
```

**Features:**
- 🔍 Detects Node.js, Python, and .NET projects
- 📦 Identifies package manager (npm/pnpm/yarn, uv/poetry/pip, dotnet)
- 🚀 Installs dependencies with the correct tool
- 🐍 Creates Python virtual environments automatically

### `azd app run`

Starts your development environment based on project type.

```bash
azd app run
```

**Supports:**
- .NET Aspire projects
- pnpm dev servers
- Docker Compose orchestration

## Commands

### AZD Context Access

All commands automatically have access to azd environment variables when invoked via azd:

```go
import "os"

// Access Azure subscription ID
subscriptionId := os.Getenv("AZURE_SUBSCRIPTION_ID")

// Access resource group name
resourceGroup := os.Getenv("AZURE_RESOURCE_GROUP_NAME")

// Access any azd environment variable
envName := os.Getenv("AZURE_ENV_NAME")
```

See [docs/azd-context.md](docs/azd-context.md) for comprehensive documentation on accessing azd context and environment variables.

## Development

### Project Structure

```
azd-app-extension/
├── src/
│   ├── cmd/
│   │   └── app/
│   │       ├── commands/       # Command implementations
│   │       │   ├── core.go     # Orchestrator setup
│   │       │   ├── deps.go     # Dependency installation
│   │       │   ├── reqs.go     # Prerequisites check
│   │       │   └── run.go      # Dev environment runner
│   │       └── main.go         # Main entry point
│   └── internal/
│       ├── detector/           # Project detection logic
│       │   └── detector.go
│       ├── installer/          # Dependency installation
│       │   └── installer.go
│       ├── runner/             # Project execution
│       │   └── runner.go
│       ├── executor/           # Safe command execution
│       │   └── executor.go
│       ├── orchestrator/       # Command dependency chain
│       │   └── orchestrator.go
│       ├── security/           # Input validation
│       │   └── validation.go
│       └── types/              # Shared types
│           └── types.go
├── tests/
│   └── projects/               # Test project fixtures
│       ├── node/
│       ├── python/
│       ├── aspire-test/
│       └── azure/
├── docs/                       # Documentation
│   ├── quickstart.md
│   ├── add-command-guide.md
│   ├── command-dependency-chain.md
│   ├── azd-context.md
│   └── reqs-command.md
├── .github/
│   └── workflows/
│       ├── ci.yml              # CI pipeline
│       └── release.yml         # Release automation
├── extension.yaml              # Extension manifest
├── registry.json               # Extension registry entry
├── go.mod                      # Go module definition
├── magefile.go                 # Mage build targets (cross-platform)
├── .golangci.yml              # Linter configuration
├── build.ps1                   # Windows build script
├── build.sh                    # Unix build script
├── install-local.ps1           # Local installation script
├── AGENTS.md                   # AI agent guidelines
├── CONTRIBUTING.md             # Contribution guidelines
├── LICENSE                     # MIT License
└── README.md                   # This file
```

### Development Commands

This project uses [Mage](https://magefile.org/) for build automation. Mage is a make/rake-like build tool using Go.

```bash
# Install mage (if not already installed)
go install github.com/magefile/mage@latest

# List all available targets
mage -l

# Build the extension
mage build

# Run unit tests
mage test

# Run integration tests (requires external tools)
mage testIntegration

# Run all tests (unit + integration)
mage testAll

# Run tests with coverage
mage testCoverage

# Run linter
mage lint

# Format code
mage fmt

# Clean build artifacts
mage clean

# Install locally
mage install

# Run everything (lint, test, build)
mage all
```

**Learn more about Mage:**
- [Mage Build Tool Guide](docs/mage-build-tool.md) - Complete guide to using Mage in this project
- [Integration Tests](docs/integration-tests.md) - Running and writing integration tests

### Adding New Commands

Use the command generator script:

```powershell
.\new-command.ps1 yourcommand
```

This creates:
1. A new command file in `src/cmd/app/commands/`
2. Registers it in the root command

Or manually:

1. Create a new file in `src/cmd/app/commands/` (e.g., `yourcommand.go`)
2. Implement the command following the pattern:

```go
package commands

import (
    "fmt"
    "github.com/spf13/cobra"
)

func NewYourCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "yourcommand",
        Short: "Description of your command",
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Println("Your command logic here")
            return nil
        },
    }
}
```

3. Register the command in `src/cmd/app/main.go`:

```go
rootCmd.AddCommand(commands.NewYourCommand())
```

4. Update `extension.yaml` to include the new command

## Testing

Test projects are located in `tests/projects/` with minimal dependencies:

```bash
# Test Node.js detection and dependency installation
cd tests/projects/node/test-npm-project
azd app deps

# Test Python detection and dependency installation
cd tests/projects/python/test-uv-project
azd app deps
```

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Code Quality Requirements

- All tests must pass: `go test ./...`
- Code coverage minimum: 80%
- Linter must pass: `golangci-lint run`
- Code must be formatted: `gofmt`
- Follow Go best practices and idioms

## License

MIT License - Copyright (c) 2024 Jon Gallant (jongio)

See [LICENSE](LICENSE) for details.

## Resources

- [Azure Developer CLI Documentation](https://learn.microsoft.com/azure/developer/azure-developer-cli/)
- [azd Extension Framework](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md)
- [Extension Registry](https://github.com/Azure/azure-dev/blob/main/cli/azd/extensions/registry.json)
- [Go Testing](https://go.dev/doc/tutorial/add-a-test)
- [Cobra CLI](https://github.com/spf13/cobra)

## Support

- **Issues**: [GitHub Issues](https://github.com/jongio/azd-app-extension/issues)
- **Discussions**: [GitHub Discussions](https://github.com/jongio/azd-app-extension/discussions)
- **Contributing**: See [CONTRIBUTING.md](CONTRIBUTING.md)

## Acknowledgments

This extension is built with:
- [Azure Developer CLI](https://github.com/Azure/azure-dev) - The azd CLI framework
- [Cobra](https://github.com/spf13/cobra) - Command-line interface framework
- [Mage](https://magefile.org/) - Build automation tool
