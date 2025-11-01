#!/bin/bash
set -e

echo "🚀 Setting up App Extension development environment..."

# Install Go tools
echo "📦 Installing Go tools..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/magefile/mage@latest

# Install Azure Developer CLI
echo "📦 Installing Azure Developer CLI..."
curl -fsSL https://aka.ms/install-azd.sh | bash

# Install additional package managers for testing
echo "📦 Installing package managers..."

# Install pnpm
npm install -g pnpm

# Install poetry
curl -sSL https://install.python-poetry.org | python3 -

# Install uv
curl -LsSf https://astral.sh/uv/install.sh | sh

# Add poetry and uv to PATH
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
export PATH="$HOME/.local/bin:$PATH"

# Install .NET Aspire workload
echo "📦 Installing .NET Aspire workload..."
dotnet workload update
dotnet workload install aspire

# Run go mod download
echo "📦 Downloading Go dependencies..."
go mod download

# Run tests to verify setup
echo "✅ Running tests to verify setup..."
go test ./... -short

echo ""
echo "✅ Development environment setup complete!"
echo ""
echo "Quick start commands:"
echo "  mage build    - Build the extension"
echo "  mage test     - Run tests"
echo "  mage install  - Install locally"
echo "  azd app reqs  - Check prerequisites"
echo ""
