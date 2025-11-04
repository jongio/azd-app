#!/bin/bash
set -e

echo "🚀 Setting up development environment..."

# Ensure mage is in PATH
if ! command -v mage &> /dev/null; then
    echo "⚠️  mage not found in PATH, installing via go install..."
    go install github.com/magefile/mage@latest
fi

# Navigate to CLI directory for Go operations
cd /workspaces/azd-app/cli || exit 1

# Download Go dependencies
echo "📦 Downloading Go dependencies..."
go mod download

# Run quick tests to verify setup
echo "🧪 Running quick tests..."
go test ./... -short

echo "✅ Development environment ready!"

