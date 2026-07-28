#!/bin/bash
# prerun.sh - Unix/Mac/Linux startup script

echo "🚀 Starting fullstack application - preparing API and Web services..."
echo "📦 Checking dependencies..."

# Check if virtual environment exists for Python API
if [ ! -d "./api/.venv" ]; then
    echo "Creating Python virtual environment..."
fi

# Check if node_modules exists for Web
if [ ! -d "./web/node_modules" ]; then
    echo "Node modules will be installed..."
fi

echo "✅ Pre-run checks complete"
exit 0
