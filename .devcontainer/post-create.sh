#!/bin/bash
set -e

echo "🚀 Setting up development environment..."

# Ensure mage is in PATH
if ! command -v mage &> /dev/null; then
    echo "⚠️  mage not found in PATH, installing via go install..."
    go install github.com/magefile/mage@latest
fi

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

# Navigate to CLI directory for Go operations
cd /workspaces/azd-app/cli || exit 1

# Download Go dependencies
echo "📦 Downloading Go dependencies..."
go mod download

# Run quick tests to verify setup
echo "🧪 Running quick tests..."
go test ./... -short

echo "✅ Development environment ready!"

