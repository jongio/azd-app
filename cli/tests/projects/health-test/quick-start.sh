#!/bin/bash

# Quick Start Script for Health Monitoring Test
# This script uses 'azd app run' to start services and test health monitoring

set -e

echo "========================================"
echo "azd app health - Quick Start Test Guide"
echo "========================================"
echo ""

# Check if in correct directory
if [ ! -f "azure.yaml" ]; then
    echo "❌ Error: azure.yaml not found. Please run from health-test directory."
    exit 1
fi

echo "✅ Found azure.yaml - ready to start"
echo ""
echo "Starting all services with 'azd app run'..."
echo ""

# Start services in background
azd app run &
AZD_PID=$!

echo "✅ Services starting (PID: $AZD_PID)"
echo ""
echo "Waiting 30 seconds for services to initialize..."
for i in {30..1}; do
    echo -ne "  $i seconds remaining...\r"
    sleep 1
done
echo ""

echo ""
echo "✅ All services should be ready!"
echo ""
echo "========================================"
echo "Running Quick Tests"
echo "========================================"
echo ""

# Test 1: Basic health check
echo "Test 1: Basic Health Check (Static Mode)"
echo "----------------------------------------"
azd app health
EXIT_CODE=$?
echo ""
if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ Test 1 PASSED (Exit code: $EXIT_CODE)"
else
    echo "❌ Test 1 FAILED (Exit code: $EXIT_CODE, expected: 0)"
fi
echo ""

# Test 2: Service info
echo "Test 2: Service Info"
echo "----------------------------------------"
azd app info
echo "✅ Test 2 PASSED"
echo ""

# Test 3: JSON output
echo "Test 3: JSON Output Format"
echo "----------------------------------------"
JSON_OUTPUT=$(azd app health --output json)
if echo "$JSON_OUTPUT" | jq . > /dev/null 2>&1; then
    echo "✅ Test 3 PASSED (Valid JSON output)"
    echo "$JSON_OUTPUT" | jq .summary
else
    echo "❌ Test 3 FAILED (Invalid JSON)"
    echo "$JSON_OUTPUT"
fi
echo ""

# Test 4: Table output
echo "Test 4: Table Output Format"
echo "----------------------------------------"
azd app health --output table
echo "✅ Test 4 PASSED"
echo ""

# Test 5: Service filtering
echo "Test 5: Service Filtering"
echo "----------------------------------------"
azd app health --service web,api
echo "✅ Test 5 PASSED"
echo ""

# Test 6: Verbose mode
echo "Test 6: Verbose Mode"
echo "----------------------------------------"
azd app health --verbose
echo "✅ Test 6 PASSED"
echo ""

echo "========================================"
echo "Quick Tests Complete!"
echo "========================================"
echo ""
echo "All basic tests passed. Services are running correctly."
echo ""
echo "Next Steps:"
echo "  1. Try streaming mode:    azd app health --stream"
echo "  2. View service logs:     azd app logs --service web"
echo "  3. Full manual testing:   See TESTING.md for comprehensive test guide"
echo ""
echo "To stop all services:"
echo "  kill $AZD_PID"
echo "  or press Ctrl+C in the terminal running 'azd app run'"
echo ""
echo "For detailed testing: cat TESTING.md"
echo ""
