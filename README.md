# azd-app

A suite of app tools for Azure Developer CLI.

[![CI](https://github.com/jongio/azd-app/actions/workflows/ci.yml/badge.svg)](https://github.com/jongio/azd-app/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jongio/azd-app/cli)](https://goreportcard.com/report/github.com/jongio/azd-app/cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## � The Problem

You have your full Azure Developer CLI (azd) projects set up - they provision and deploy beautifully to Azure. But what about running them **locally**?

- ❓ How do you verify all prerequisites are installed?
- ❓ How do you recursively install all your dependencies across multiple languages?
- ❓ How do you actually run the application on your machine?
- ❓ How do you view all available URLs and monitor the health of your running services?

**Azure Developer CLI doesn't handle this today.** Until now.

## ✨ The Solution

With the **azd app extension**, local development becomes effortless with just a few simple commands:

```bash
azd app reqs  # ✅ Check all prerequisites are installed
azd app deps  # 📦 Install all dependencies automatically
azd app run   # 🚀 Start your application locally with a live dashboard
```

The `azd app run` command automatically launches a **web dashboard** where you can:
- 📊 View all available service URLs in one place
- 💚 Monitor the health status of your locally running services
- 🔗 Click to open any service directly in your browser

**Smart. Simple. Automatic.** Works with Node.js, Python, .NET, Aspire, and more.

---

## �📦 Components

This monorepo contains multiple tools that work together to enhance your Azure Developer CLI experience:

### [CLI Extension](./cli/) 
**Status:** ✅ Active

An Azure Developer CLI (azd) extension that automates development environment setup by detecting project types and running appropriate commands across multiple languages and frameworks.

- **Languages**: Node.js, Python, .NET, Aspire
- **Package Managers**: npm, pnpm, yarn, uv, poetry, pip, dotnet
- **Features**: Smart detection, dependency installation, dev environment orchestration

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
azd app run

# Option 2: Create a new sample project
azd init -t hello-azd
azd app run
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
