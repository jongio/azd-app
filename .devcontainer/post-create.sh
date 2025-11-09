#!/bin/bash
set -e

echo "🚀 Setting up development environment..."

# Ensure mage is in PATH
if ! command -v mage &> /dev/null; then
    echo "⚠️  mage not found in PATH, installing via go install..."
    go install github.com/magefile/mage@latest
fi

# Install golangci-lint v1.64.8 to match local development
echo "📦 Installing golangci-lint v1.64.8..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8

# Install gosec for security scanning
echo "📦 Installing gosec for security scanning..."
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Install Aspire CLI
echo "📦 Installing Aspire CLI..."
ASPIRE_VERSION="9.5.2"
if dotnet tool list --global | grep -q "^aspire.cli"; then
    dotnet tool update --global aspire.cli --version $ASPIRE_VERSION
else
    dotnet tool install --global aspire.cli --version $ASPIRE_VERSION
fi

# Add dotnet tools to PATH for this session
export PATH="$HOME/.dotnet/tools:$PATH"

# Determine the CLI directory path (works in both devcontainer and CI)
# In devcontainer it's /workspaces/azd-app/cli
# In GitHub Actions it might be different
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLI_DIR="$REPO_ROOT/cli"

# Navigate to CLI directory for Go operations
cd "$CLI_DIR" || exit 1

# Download Go dependencies
echo "📦 Downloading Go dependencies..."
go mod download

# Build dashboard assets (required for tests)
echo "🎨 Building dashboard assets..."
cd dashboard
npm install
npm run build
cd ..

# Run quick tests to verify setup
echo "🧪 Running quick tests..."
go test ./... -short

echo "✅ Development environment ready!"

