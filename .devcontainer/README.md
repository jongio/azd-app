# App Extension Devcontainer

This devcontainer provides a complete development environment for the App Extension project.

## What's Included

### Languages & Runtimes
- **Go 1.23** - Primary development language
- **Node.js LTS** - For Node.js project testing
- **Python 3.12** - For Python project testing
- **.NET 8.0** - For .NET and Aspire project testing

### Tools
- **Azure Developer CLI (azd)** - The target platform
- **Azure CLI** - For Azure operations
- **Docker** - For container testing (Docker-in-Docker)
- **PowerShell** - For running scripts

### Package Managers
- **npm, pnpm, yarn** - Node.js package managers
- **pip, poetry, uv** - Python package managers
- **dotnet** - .NET package manager

### Go Tools
- **golangci-lint** - Linter with 22 enabled linters
- **Mage** - Cross-platform build tool
- **gopls** - Go language server

### VS Code Extensions
- Go extension with language server support
- Azure Developer CLI extension
- GitHub Copilot
- PowerShell extension
- Makefile Tools

## Usage

### Opening in VS Code

1. Install the "Dev Containers" extension in VS Code
2. Open the project folder
3. When prompted, click "Reopen in Container"
   - Or use Command Palette: `Dev Containers: Reopen in Container`

### First Time Setup

The devcontainer will automatically:
1. Install all Go tools and dependencies
2. Install Azure Developer CLI
3. Install package managers (pnpm, poetry, uv)
4. Install .NET Aspire workload
5. Run tests to verify the setup

This takes about 3-5 minutes on first launch.

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
