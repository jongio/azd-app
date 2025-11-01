#!/bin/bash
set -e

echo "🚀 Setting up App Extension development environment..."

# Install Go tools (not included in base image)
echo "📦 Installing Go tools..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/magefile/mage@latest

# Install additional package managers for testing (not in devcontainer features)
echo "📦 Installing additional package managers..."

# Install pnpm (npm is already installed via node feature)
npm install -g pnpm

# Install poetry (pip is already installed via python feature)
curl -sSL https://install.python-poetry.org | python3 -

# Install uv
curl -LsSf https://astral.sh/uv/install.sh | sh

# Add to PATH for current session
export PATH="$HOME/.local/bin:$PATH"

# Download Go dependencies
echo "📦 Downloading Go dependencies..."
go mod download

# Run quick tests to verify setup
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
