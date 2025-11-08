# azd app

**Supercharge your local development with Azure Developer CLI.**

[![CI](https://github.com/jongio/azd-app/actions/workflows/ci.yml/badge.svg)](https://github.com/jongio/azd-app/actions/workflows/ci.yml)
[![Release](https://github.com/jongio/azd-app/actions/workflows/release.yml/badge.svg)](https://github.com/jongio/azd-app/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/jongio/azd-app/branch/main/graph/badge.svg)](https://codecov.io/gh/jongio/azd-app)
[![Go Report Card](https://goreportcard.com/badge/github.com/jongio/azd-app/cli?refresh=1)](https://goreportcard.com/report/github.com/jongio/azd-app/cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A suite of productivity tools that extend [Azure Developer CLI](https://learn.microsoft.com/azure/developer/azure-developer-cli/) with powerful local development capabilities.

---

## Why azd app?

Azure Developer CLI (azd) is fantastic for provisioning and deploying to Azure. But what about **local development**?

azd app fills that gap with intelligent automation:

- ✅ **Verify prerequisites** - Check all required tools are installed
- 📦 **Install dependencies** - Recursively install across all projects and languages  
- 🚀 **Run locally** - Start your entire application with one command
- 📊 **Live dashboard** - Monitor services, view URLs, stream logs
- 🔄 **Multi-language support** - Node.js, Python, .NET, Aspire, and more

## Quick Example

```bash
# Check prerequisites
azd app reqs

# Install all dependencies automatically
azd app deps

# Start your application with live dashboard
azd app run
```

That's it. No configuration needed. azd app detects your project structure and does the right thing.

---

## 📦 Components

This monorepo contains multiple tools that work together to enhance your Azure Developer CLI experience:

### [CLI Extension](./cli/) 
**Status:** ✅ Active

An Azure Developer CLI (azd) extension that automates development environment setup by detecting project types and running appropriate commands across multiple languages and frameworks.

- **Languages**: Node.js, Python, .NET, Aspire
- **Package Managers**: npm, pnpm, yarn, uv, poetry, pip, dotnet
- **Features**: 
  - Smart project and dependency detection
  - Prerequisite checking with caching
  - Automatic dependency installation
  - Service orchestration from azure.yaml
  - Lifecycle hooks (prerun/postrun automation)
  - Live web dashboard with service monitoring
  - Real-time log streaming
  - Azure environment integration
  - Python entry point auto-detection

👉 [CLI Documentation](./cli/README.md)

### VS Code Extension
**Status:** 🚧 Coming Soon

Visual Studio Code extension for enhanced azd workflows and project management.

### MCP Server
**Status:** ✅ Active

Model Context Protocol server for AI-assisted development with Azure Developer CLI.

- **Implementation**: Native Go implementation using `mark3labs/mcp-go`
- **AI Integration**: Enables AI assistants to monitor and operate on running applications
- **Tools**: 6 tools (3 observability + 3 operational)
  - **Observability**: get_services, get_service_logs, get_project_info
  - **Operations**: run_services, install_dependencies, check_requirements
- **Features**:
  - Real-time service status and health monitoring
  - Log streaming with filtering capabilities
  - Start/stop services through AI commands
  - Automatic dependency installation
  - Prerequisite checking and validation
  - Azure deployment information access
  - Zero external dependencies (no Node.js required)

👉 [Usage Guide: Using with AI Assistants](./docs/mcp-usage.md)

---

## 🚀 Quick Start

### 1. Install Azure Developer CLI

If you haven't already, install Azure Developer CLI:

```bash
# Windows (PowerShell)
winget install microsoft.azd

# macOS (Homebrew)
brew tap azure/azd && brew install azd

# Linux
curl -fsSL https://aka.ms/install-azd.sh | bash
```

Learn more: [Install Azure Developer CLI](https://learn.microsoft.com/azure/developer/azure-developer-cli/install-azd)

### 2. Enable azd Extensions

```bash
azd config set alpha.extension.enabled on
```

Learn more: [Extensions Documentation](https://learn.microsoft.com/azure/developer/azure-developer-cli/azd-extensions)

### 3. Add the Extension Registry

```bash
azd extension source add -n app -t url -l "https://raw.githubusercontent.com/jongio/azd-app/refs/heads/main/registry.json"
```

### 4. Install the Extension

```bash
azd extension install jongio.azd.app
```

### 5. Try It Out

```bash
# Option 1: Use an existing azd project
cd your-azd-project
azd app reqs  # Check prerequisites
azd app deps  # Install dependencies
azd app run   # Start services with dashboard

# Option 2: Create a new sample project
azd init -t hello-azd
azd up
azd app run

# View service information
azd app info

# Stream logs
azd app logs           # All services
azd app logs api       # Specific service
azd app logs -f        # Follow mode
```

For detailed installation and usage instructions, see the [CLI documentation](./cli/README.md).

---

## 📂 Repository Structure

```
azd-app/
├── cli/              # Azure Developer CLI Extension (Go)
├── vsc/              # VS Code Extension (TypeScript) - Coming Soon
├── docs/             # Documentation
└── .github/          # CI/CD workflows
```

---

## 🤝 Contributing

Contributions are welcome! See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

### Development Setup

1. **CLI Extension**: See [cli/README.md](./cli/README.md#for-development--testing)
2. **VS Code Extension**: Coming soon

---

## 📄 License

MIT License - see [LICENSE](./LICENSE) for details.

---

## 🔗 Links

- [CLI Extension](./cli/)
- [Documentation](./cli/docs/)
- [Contributing Guide](./CONTRIBUTING.md)
- [Issue Tracker](https://github.com/jongio/azd-app/issues)

