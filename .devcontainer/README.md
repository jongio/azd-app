# App Extension Devcontainer

This devcontainer provides a complete development environment for the App Extension project, following the latest [VS Code Dev Container specification](https://containers.dev/).

## What's Included

### Languages & Runtimes
- **Go 1.23** - Primary development language
- **Node.js LTS** - For Node.js project testing
- **Python 3.12** - For Python project testing
- **.NET 8.0** - For .NET and Aspire project testing

### Tools
All tools are installed declaratively via [devcontainer features](https://containers.dev/features):

- **Azure Developer CLI (azd)** - The target platform
- **Azure CLI** - For Azure operations
- **Docker-in-Docker** - For container testing with non-root support
- **PowerShell** - For running scripts
- **golangci-lint** - Linter with 22 enabled linters
- **Mage** - Cross-platform build tool

### Package Managers
All package managers are installed automatically via devcontainer features:

- **npm, pnpm, yarn** - Node.js package managers (via Node feature)
- **pip, poetry, uv** - Python package managers (via Python and uv features)
- **dotnet** - .NET package manager (via .NET feature)

### Go Tools
- **gopls** - Go language server
- **dlv** - Delve debugger
- Standard Go toolchain

### VS Code Extensions
- Go extension with language server support
- Azure Developer CLI extension
- GitHub Copilot
- PowerShell extension
- Makefile Tools

## Resource Requirements

This devcontainer is configured with the following minimum requirements:
- **CPUs**: 4 cores
- **Memory**: 8GB
- **Storage**: 32GB

For complex testing scenarios with multiple containers, you may need to increase these limits in Docker Desktop settings.

## Usage

### Opening in VS Code

1. Install the "Dev Containers" extension in VS Code
2. Open the project folder
3. When prompted, click "Reopen in Container"
   - Or use Command Palette: `Dev Containers: Reopen in Container`

### Container Lifecycle

The devcontainer uses the following lifecycle commands:

- **onCreateCommand**: Runs once when the container is created (downloads Go dependencies, runs verification tests)
- **postStartCommand**: Runs each time the container starts (currently not configured)

See [Dev Container specification](https://containers.dev/implementors/json_reference/#lifecycle-scripts) for more details.

### First Time Setup

The devcontainer will automatically:
1. Install all language runtimes via devcontainer features
2. Install development tools (golangci-lint, mage, uv) via devcontainer features
3. Install package managers (pnpm, yarn, poetry) via devcontainer features
4. Download Go modules (via `onCreateCommand`)
5. Run verification tests (via `onCreateCommand`)

Most installations happen declaratively through devcontainer features, making the setup more reliable and faster. The `onCreateCommand` only handles project-specific setup (Go dependencies and tests).

This process happens once during container creation and takes about 2-3 minutes.

### Azure Credentials

Your local Azure credentials are automatically mounted into the container:
- `~/.azure` - Azure CLI credentials
- `~/.azd` - Azure Developer CLI configuration

This means you don't need to re-authenticate inside the container.

### Quick Commands

Once inside the container:

```bash
# Build the extension
mage build

# Run tests
mage test

# Install locally
mage install

# Test the extension
azd app reqs
azd app deps
azd app run

# Run linter
mage lint

# View coverage
mage coverage
```

## Development Workflow

1. Make code changes in VS Code
2. Save files (auto-format on save is enabled)
3. Run `mage build` to build
4. Run `mage test` to test
5. Run `mage install` to install locally
6. Test with `azd app` commands

## Troubleshooting

### Container fails to start

- Ensure Docker is running
- Check Docker has enough resources (4GB RAM recommended)

### Azure CLI not authenticated

If mounted credentials don't work:
```bash
az login
azd auth login
```

### Package managers not found

Re-run the post-create script:
```bash
bash .devcontainer/post-create.sh
```

### Go tools missing

Reinstall Go tools:
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/magefile/mage@latest
```

## Customization

To customize the devcontainer:
1. Edit `.devcontainer/devcontainer.json`
2. Rebuild container: `Dev Containers: Rebuild Container`

## Performance

The devcontainer uses bind mounts for Azure credentials, which provides:
- Fast credential access
- Automatic sync with host
- No need to re-authenticate

For best performance on Windows, ensure Docker Desktop is using WSL 2 backend.
