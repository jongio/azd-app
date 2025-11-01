#!/bin/bash

# Build script for App Extension (Unix/Linux/macOS)

set -e

EXTENSION_NAME="app"
OUTPUT_DIR="bin"

echo "Building App Extension..."

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Get build arguments
BUILD_ALL=false
while [[ $# -gt 0 ]]; do
  case $1 in
    --all)
      BUILD_ALL=true
      shift
      ;;
    *)
      shift
      ;;
  esac
done

if [ "$BUILD_ALL" = true ]; then
  echo "Building for all platforms..."
  
  # Windows
  echo "Building for Windows (AMD64)..."
  GOOS=windows GOARCH=amd64 go build -o "$OUTPUT_DIR/windows-amd64/$EXTENSION_NAME.exe" .
  
  echo "Building for Windows (ARM64)..."
  GOOS=windows GOARCH=arm64 go build -o "$OUTPUT_DIR/windows-arm64/$EXTENSION_NAME.exe" .
  
  # Linux
  echo "Building for Linux (AMD64)..."
  GOOS=linux GOARCH=amd64 go build -o "$OUTPUT_DIR/linux-amd64/$EXTENSION_NAME" .
  
  echo "Building for Linux (ARM64)..."
  GOOS=linux GOARCH=arm64 go build -o "$OUTPUT_DIR/linux-arm64/$EXTENSION_NAME" .
  
  # macOS
  echo "Building for macOS (AMD64)..."
  GOOS=darwin GOARCH=amd64 go build -o "$OUTPUT_DIR/darwin-amd64/$EXTENSION_NAME" .
  
  echo "Building for macOS (ARM64)..."
  GOOS=darwin GOARCH=arm64 go build -o "$OUTPUT_DIR/darwin-arm64/$EXTENSION_NAME" .
  
  echo "✅ Build complete for all platforms!"
else
  echo "Building for current platform..."
  go build -o "$OUTPUT_DIR/$EXTENSION_NAME" .
  echo "✅ Build complete!"
fi

echo "Binaries are in the '$OUTPUT_DIR' directory"
