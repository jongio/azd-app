#!/bin/bash
set -e

# Configuration
DASHBOARD_DIR="dashboard"
ASPIRE_VERSION="9.5.2"

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
if dotnet tool list --global | grep -q "^aspire.cli"; then
    dotnet tool update --global aspire.cli --version $ASPIRE_VERSION
else
    dotnet tool install --global aspire.cli --version $ASPIRE_VERSION
fi

# Add dotnet tools to PATH for this session
export PATH="$HOME/.dotnet/tools:$PATH"

# Navigate to CLI directory for Go operations
# Try devcontainer path first, then fallback to script location
if [ -d "/workspaces/azd-app/cli" ]; then
    cd /workspaces/azd-app/cli
elif [ -f "$(dirname "$0")/../cli/go.mod" ]; then
    cd "$(dirname "$0")/../cli"
else
    echo "❌ Could not find CLI directory"
    exit 1
fi

# Download Go dependencies
echo "📦 Downloading Go dependencies..."
go mod download

# Build dashboard to create dist directory (required for embed directive)
echo "🎨 Building dashboard..."
if command -v npm &> /dev/null; then
    if [ -d "$DASHBOARD_DIR" ]; then
        CLI_DIR=$(pwd)
        cd "$DASHBOARD_DIR"
        
        # Install dependencies (show errors but reduce noise)
        if ! npm install --quiet --no-progress 2>&1; then
            echo "❌ Dashboard npm install failed"
            cd "$CLI_DIR"
            exit 1
        fi
        
        # Build dashboard (show errors but reduce noise)
        if ! npm run build --quiet 2>&1; then
            echo "❌ Dashboard build failed"
            cd "$CLI_DIR"
            exit 1
        fi
        
        cd "$CLI_DIR"
    else
        echo "⚠️  Dashboard directory not found at $DASHBOARD_DIR, skipping dashboard build"
    fi
else
    echo "⚠️  npm not found, skipping dashboard build"
fi

# Run quick tests to verify setup
echo "🧪 Running quick tests..."
go test ./... -short

echo "✅ Development environment ready!"

