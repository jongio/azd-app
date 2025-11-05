#!/bin/bash
set -e

echo "🚀 Setting up development environment..."

# Ensure Go tools are in PATH (added by devcontainer but might not be set yet)
export PATH="$HOME/go/bin:$PATH"

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

# Build dashboard to create dist directory (required for embed directive)
echo "🎨 Building dashboard..."
if command -v npm &> /dev/null; then
    cd dashboard
    npm install --silent --no-progress
    npm run build --silent
    cd ..
else
    echo "⚠️  npm not found, skipping dashboard build"
fi

# Run quick tests to verify setup
echo "🧪 Running quick tests..."
go test ./... -short

echo "✅ Development environment ready!"

