# App Extension for Azure Developer CLI

[![CI](https://github.com/jongio/azd-app/actions/workflows/ci.yml/badge.svg)](https://github.com/jongio/azd-app/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jongio/azd-app/cli)](https://goreportcard.com/report/github.com/jongio/azd-app/cli)
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
azd config set extension.registry https://raw.githubusercontent.com/jongio/azd-app/main/registry.json
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
git clone https://github.com/jongio/azd-app.git
cd azd-app/cli

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

### Using Devcontainer

For a consistent, pre-configured development environment:

1. Install [Docker](https://docs.docker.com/get-docker/) and [VS Code Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. Open the project in VS Code
3. Click "Reopen in Container" when prompted

The devcontainer includes:
- Go, Node.js, Python, .NET pre-installed
- All package managers (npm, pnpm, yarn, pip, poetry, uv)
- Azure Developer CLI and Azure CLI
- Mage, golangci-lint, and all development tools
- Your Azure credentials automatically mounted

See [.devcontainer/README.md](.devcontainer/README.md) for details.

### Prerequisites

- [Azure Developer CLI (azd)](https://learn.microsoft.com/azure/developer/azure-developer-cli/install-azd) installed
- Go 1.25 or later (for building from source)
- Node.js 20.0.0 or later (for building dashboard)
- npm 10.0.0 or later (for building dashboard)
- PowerShell 7.4 or later (recommended: 7.5.4 for full compatibility with build scripts)
- TypeScript 5.9.3 (installed via npm, required for dashboard)

## Quick Start

Once installed, you can use these commands:

### `azd app reqs`

Verifies that all required tools are installed and optionally checks if they are running. Can also auto-generate requirements based on your project.

```bash
# Check requirements defined in azure.yaml
azd app reqs

# Auto-generate requirements from your project
azd app reqs --generate

# Preview what would be generated without making changes
azd app reqs --generate --dry-run
```

**Features:**
- ✅ Checks if required tools are installed
- ✅ Validates minimum version requirements
- ✅ Verifies if services are running (e.g., Docker daemon)
- ✅ **Auto-generates requirements from detected project dependencies**
- ✅ **Smart version normalization** (e.g., Node: major only, Python: major.minor)
- ✅ **Merges with existing requirements** without duplicates
- ✅ Supports custom tool configurations
- ✅ Built-in support for Node.js, Python, .NET, Aspire, Docker, Git, Azure CLI, and more

**Auto-Generation Example:**

When you run `azd app reqs --generate` in a Node.js project:

```bash
🔍 Scanning project for dependencies...

Found:
  ✓ Node.js project (pnpm)

📝 Detected requirements:
  • node (22.19.0 installed) → minVersion: "22.0.0"
  • pnpm (10.20.0 installed) → minVersion: "10.0.0"

✅ Created azure.yaml with 2 requirements
```

The generated `azure.yaml`:
```yaml
name: my-project
reqs:
  - id: node
    minVersion: 22.0.0
  - id: pnpm
    minVersion: 10.0.0
```

**Supported Detection:**
- **Node.js**: Automatically detects npm, pnpm, or yarn based on lock files
- **Python**: Detects pip, poetry, uv, or pipenv based on project files
- **.NET**: Detects dotnet SDK and Aspire workloads
- **Docker**: Detects from Dockerfile or docker-compose files
- **Git**: Detects from .git directory

**Manual Configuration Example:**

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
# Run with default azd dashboard
azd app run

# Run specific services only
azd app run --service web,api

# Use native Aspire dashboard (for .NET Aspire projects)
azd app run --runtime aspire

# Preview what would run without starting
azd app run --dry-run

# Enable verbose logging
azd app run --verbose

# Load environment variables from custom file
azd app run --env-file .env.local
```

**Flags:**
- `--service, -s`: Run specific service(s) only (comma-separated)
- `--runtime`: Runtime mode - `azd` (default, uses azd dashboard) or `aspire` (native Aspire with dotnet run)
- `--env-file`: Load environment variables from .env file
- `--verbose, -v`: Enable verbose logging
- `--dry-run`: Show what would be run without starting services

**Runtime Modes:**
- **azd** (default): Runs services through azd's built-in dashboard, works with all project types
- **aspire**: Uses native .NET Aspire dashboard via `dotnet run` (only for Aspire projects)

**Supports:**
- Services defined in azure.yaml (multi-service orchestration)
- .NET Aspire projects (AppHost.cs)
- pnpm dev servers
- Docker Compose orchestration

### `azd app auth`

Manage authentication server for secure container-to-container communication.

```bash
# Start authentication server
azd app auth server start --secret mysecret

# Get token from auth server
azd app auth token get --server http://auth-server:8080 --secret mysecret

# Check server health
azd app auth token get --server http://auth-server:8080 --secret mysecret --health-check
```

**Server Commands:**
- `azd app auth server start` - Start the authentication server
- `azd app auth server status` - Check server status

**Client Commands:**
- `azd app auth token get` - Fetch tokens from auth server

**Features:**
- 🔐 Centralized authentication for container networks
- 🔑 JWT-wrapped Azure tokens for security
- 💾 Smart caching to minimize API calls
- 🚦 Rate limiting to prevent abuse
- 🔒 TLS/HTTPS support for production
- 🐳 Compatible with Docker, Podman, and Kubernetes

**Use Cases:**
- Distribute Azure credentials across multiple containers
- Sidecar authentication pattern
- Standalone auth server for microservices
- Eliminate credential duplication in container environments

See [docs/auth-server.md](docs/auth-server.md) for comprehensive documentation including Docker Compose and Kubernetes examples.

## Commands

For complete command reference with all flags and options, see the [CLI Reference Documentation](docs/cli-reference.md).

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

# Run preflight checks before committing/releasing
# This includes: dashboard build, formatting, linting, security scan, all tests, and coverage
mage preflight
```

**Dashboard Development:**

The dashboard is a React + TypeScript application that provides a web UI for monitoring services.

```bash
# Build the dashboard (TypeScript compilation + Vite build)
mage dashboardBuild

# Start dashboard development server with hot reload
mage dashboardDev

# Build dashboard manually
cd dashboard
npm install
npm run build
```

The dashboard build is automatically included in:
- `mage all` - Ensures dashboard is built before Go compilation
- `mage preflight` - Validates TypeScript compilation and catches type errors
- CI/CD workflows - Dashboard is built and validated in all pipelines

**Learn more about Mage:**
- [Mage Build Tool Guide](docs/mage-build-tool.md) - Complete guide to using Mage in this project
- [Integration Tests](docs/integration-tests.md) - Running and writing integration tests

### Adding New Commands

To add a new command, describe what you want to **GitHub Copilot** and it will generate the implementation for you.

**Important**: Always request both **unit tests** and **integration tests** for your command.

Example prompt:
```
Add a new 'azd app validate' command that checks project configuration.
Include unit tests and integration tests.
```

Copilot will:
1. Create the command file in `src/cmd/app/commands/`
2. Register it in `src/cmd/app/main.go`
3. Update `extension.yaml`
4. Generate unit tests in the same directory
5. Generate integration tests in `src/internal/*/` directories as needed

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

### Documentation

- **[CLI Reference](docs/cli-reference.md)**: Complete command reference with all flags and options
- **[Release Process](docs/release-process.md)**: Guide for publishing new versions
- **[Quick Release](docs/release-quick.md)**: Quick reference for releases
- **[Development Guides](docs/dev/)**: Internal development documentation

### External Resources

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
