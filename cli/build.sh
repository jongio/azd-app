#!/bin/bash
# Build script called by azd x build
# This handles pre-build steps like dashboard compilation

set -e

# Get the directory of the script (cli folder)
EXTENSION_DIR="$(cd "$(dirname "$0")" && pwd)"

# Change to the script directory
cd "$EXTENSION_DIR" || exit

# Check if .go files have changed by comparing timestamps
# Only kill CLI processes if Go files need rebuilding
SHOULD_KILL_PROCESSES=false

if [ -d "src" ]; then
    # Find newest .go file
    NEWEST_GO_FILE=$(find src -name "*.go" -type f -exec stat -c %Y {} \; 2>/dev/null | sort -n | tail -1 || \
                     find src -name "*.go" -type f -exec stat -f %m {} \; 2>/dev/null | sort -n | tail -1)
    
    if [ -d "bin" ]; then
        # Find newest binary (excluding .old files)
        NEWEST_BINARY=$(find bin -type f ! -name "*.old" -exec stat -c %Y {} \; 2>/dev/null | sort -n | tail -1 || \
                        find bin -type f ! -name "*.old" -exec stat -f %m {} \; 2>/dev/null | sort -n | tail -1)
        
        if [ -n "$NEWEST_GO_FILE" ] && [ -n "$NEWEST_BINARY" ]; then
            if [ "$NEWEST_GO_FILE" -gt "$NEWEST_BINARY" ]; then
                SHOULD_KILL_PROCESSES=true
            fi
        elif [ -n "$NEWEST_GO_FILE" ]; then
            # No binary exists, will need to build
            SHOULD_KILL_PROCESSES=true
        fi
    else
        # No bin directory, will need to build
        SHOULD_KILL_PROCESSES=true
    fi
fi

if [ "$SHOULD_KILL_PROCESSES" = true ]; then
    # Kill any running app processes to allow rebuilding
    echo "Go files changed - stopping any running app processes..."
    BINARY_NAME="app"
    EXTENSION_ID_FOR_KILL="jongio.azd.app"
    EXTENSION_BINARY_PREFIX="${EXTENSION_ID_FOR_KILL//./-}"

    # Kill processes silently (ignore errors if not running)
    pkill -f "$BINARY_NAME" 2>/dev/null || true
    pkill -f "$EXTENSION_BINARY_PREFIX" 2>/dev/null || true
fi

echo "Building App Extension..."

# Build dashboard first (if needed)
DASHBOARD_DIST_PATH="src/internal/dashboard/dist"
DASHBOARD_SRC_PATH="dashboard/src"

SHOULD_BUILD_DASHBOARD=false

if [ ! -d "$DASHBOARD_DIST_PATH" ]; then
    SHOULD_BUILD_DASHBOARD=true
    echo "Dashboard not built yet"
elif [ -d "$DASHBOARD_SRC_PATH" ]; then
    DIST_TIME=$(stat -c %Y "$DASHBOARD_DIST_PATH" 2>/dev/null || stat -f %m "$DASHBOARD_DIST_PATH" 2>/dev/null)
    NEWEST_SRC=$(find "$DASHBOARD_SRC_PATH" -type f -printf '%T@\n' 2>/dev/null | sort -n | tail -1 || find "$DASHBOARD_SRC_PATH" -type f -exec stat -f %m {} \; 2>/dev/null | sort -n | tail -1)
    
    if [ -n "$NEWEST_SRC" ] && [ "${NEWEST_SRC%.*}" -gt "$DIST_TIME" ]; then
        SHOULD_BUILD_DASHBOARD=true
        echo "Dashboard source changed, rebuild needed"
    fi
fi

if [ "$SHOULD_BUILD_DASHBOARD" = true ]; then
    echo "Building dashboard..."
    pushd dashboard > /dev/null
    
    if [ ! -d "node_modules" ]; then
        echo "  Installing dashboard dependencies..."
        npm install --silent
        if [ $? -ne 0 ]; then
            echo "ERROR: npm install failed"
            exit 1
        fi
    fi

    echo "  Building dashboard bundle..."
    npm run build --silent
    if [ $? -ne 0 ]; then
        echo "ERROR: Dashboard build failed"
        exit 1
    fi
    echo "  ✓ Dashboard built successfully"
    
    popd > /dev/null
else
    echo "  ✓ Dashboard up to date"
fi

# Create a safe version of EXTENSION_ID replacing dots with dashes
EXTENSION_ID_SAFE="${EXTENSION_ID//./-}"

# Define output directory
OUTPUT_DIR="${OUTPUT_DIR:-$EXTENSION_DIR/bin}"

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Get Git commit hash and build date
COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Read version from extension.yaml if EXTENSION_VERSION not set
if [ -z "$EXTENSION_VERSION" ]; then
    if [ -f "extension.yaml" ]; then
        EXTENSION_VERSION=$(grep -E '^version:' extension.yaml | awk '{print $2}' | tr -d '[:space:]')
        if [ -z "$EXTENSION_VERSION" ]; then
            EXTENSION_VERSION="0.0.0-dev"
        fi
    else
        EXTENSION_VERSION="0.0.0-dev"
    fi
fi

echo "Building version $EXTENSION_VERSION"

# List of OS and architecture combinations
if [ -n "$EXTENSION_PLATFORM" ]; then
    PLATFORMS=("$EXTENSION_PLATFORM")
else
    PLATFORMS=(
        "windows/amd64"
        "windows/arm64"
        "darwin/amd64"
        "darwin/arm64"
        "linux/amd64"
        "linux/arm64"
    )
fi

APP_PATH="github.com/jongio/azd-app/cli/src/cmd/app/commands"

# Loop through platforms and build
for PLATFORM in "${PLATFORMS[@]}"; do
    OS=$(echo "$PLATFORM" | cut -d'/' -f1)
    ARCH=$(echo "$PLATFORM" | cut -d'/' -f2)

    OUTPUT_NAME="$OUTPUT_DIR/$EXTENSION_ID_SAFE-$OS-$ARCH"

    if [ "$OS" = "windows" ]; then
        OUTPUT_NAME+='.exe'
    fi

    echo "  Building for $OS/$ARCH..."

    # Delete the output file if it already exists
    [ -f "$OUTPUT_NAME" ] && rm -f "$OUTPUT_NAME"

    LDFLAGS="-s -w -X '$APP_PATH.Version=$EXTENSION_VERSION' -X '$APP_PATH.BuildTime=$BUILD_DATE' -X '$APP_PATH.Commit=$COMMIT'"

    # Set environment variables for Go build
    GOOS=$OS GOARCH=$ARCH go build \
        -ldflags="$LDFLAGS" \
        -o "$OUTPUT_NAME" \
        ./src/cmd/app

    if [ $? -ne 0 ]; then
        echo "ERROR: Build failed for $OS/$ARCH"
        exit 1
    fi
done

echo ""
echo "✓ Build completed successfully!"
echo "  Binaries are located in the $OUTPUT_DIR directory."
