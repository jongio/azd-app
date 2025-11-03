# azd app

**Supercharge your local development with Azure Developer CLI.**

[![CI](https://github.com/jongio/azd-app/actions/workflows/ci.yml/badge.svg)](https://github.com/jongio/azd-app/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jongio/azd-app/cli)](https://goreportcard.com/report/github.com/jongio/azd-app/cli)
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

## �📦 Components

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
  - Live web dashboard with service monitoring
  - Real-time log streaming
  - Azure environment integration
  - Python entry point auto-detection

👉 [CLI Documentation](./cli/README.md)

### VS Code Extension
**Status:** 🚧 Coming Soon

Visual Studio Code extension for enhanced azd workflows and project management.

### MCP Server
**Status:** 🚧 Coming Soon

Model Context Protocol server for AI-assisted development with Azure Developer CLI.

---

## 🚀 Quick Start

### Prerequisites

First, install Azure Developer CLI if you haven't already:

```bash
# Windows (PowerShell)
winget install microsoft.azd

# macOS (Homebrew)
brew tap azure/azd && brew install azd

# Linux
curl -fsSL https://aka.ms/install-azd.sh | bash
```

Then enable azd extensions:

```bash
azd config set alpha.extension.enabled on
```

Learn more: [Install Azure Developer CLI](https://learn.microsoft.com/azure/developer/azure-developer-cli/install-azd) | [Extensions Documentation](https://learn.microsoft.com/azure/developer/azure-developer-cli/azd-extensions)

### Install the CLI Extension

```bash
# Add the extension registry
azd config set extension.registry https://raw.githubusercontent.com/jongio/azd-app/main/registry.json

# Install the extension
azd extension install app
```

### Try It Out

```bash
# Option 1: Use an existing azd project
cd your-azd-project
azd app reqs  # Check prerequisites
azd app deps  # Install dependencies
azd app run   # Start services with dashboard

# Option 2: Create a new sample project
azd init -t hello-azd
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
├── mcp/              # MCP Server (TypeScript) - Coming Soon
├── docs/             # Documentation
└── .github/          # CI/CD workflows
```

---

## 🤝 Contributing

Contributions are welcome! See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

### Development Setup

1. **CLI Extension**: See [cli/README.md](./cli/README.md#for-development--testing)
2. **VS Code Extension**: Coming soon
3. **MCP Server**: Coming soon

---

## 📄 License

MIT License - see [LICENSE](./LICENSE) for details.

---

## 🔗 Links

- [CLI Extension](./cli/)
- [Documentation](./cli/docs/)
- [Contributing Guide](./CONTRIBUTING.md)
- [Issue Tracker](https://github.com/jongio/azd-app/issues)

