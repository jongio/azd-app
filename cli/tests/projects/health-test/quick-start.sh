#!/bin/bash

# Quick Start Script for Health Monitoring Test
# This script helps you get started with manual testing quickly

set -e

echo "========================================"
echo "azd app health - Quick Start Test Guide"
echo "========================================"
echo ""

# Check if azd-app is built
AZD_APP="../../../azd-app"
if [ ! -f "$AZD_APP" ]; then
    echo "❌ azd-app not found. Building now..."
    cd ../../..
    go build -o azd-app ./src/cmd/app
    cd tests/projects/health-test
    echo "✅ azd-app built successfully"
else
    echo "✅ azd-app found"
fi

echo ""
echo "Starting all services..."
./start-all.sh

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
$AZD_APP health
EXIT_CODE=$?
echo ""
if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ Test 1 PASSED (Exit code: $EXIT_CODE)"
else
    echo "❌ Test 1 FAILED (Exit code: $EXIT_CODE, expected: 0)"
fi
echo ""

# Test 2: JSON output
echo "Test 2: JSON Output Format"
echo "----------------------------------------"
JSON_OUTPUT=$($AZD_APP health --output json)
if echo "$JSON_OUTPUT" | jq . > /dev/null 2>&1; then
    echo "✅ Test 2 PASSED (Valid JSON output)"
    echo "$JSON_OUTPUT" | jq .
else
    echo "❌ Test 2 FAILED (Invalid JSON)"
    echo "$JSON_OUTPUT"
fi
echo ""

# Test 3: Table output
echo "Test 3: Table Output Format"
echo "----------------------------------------"
$AZD_APP health --output table
echo "✅ Test 3 PASSED"
echo ""

# Test 4: Service filtering
echo "Test 4: Service Filtering"
echo "----------------------------------------"
$AZD_APP health --service web,api
echo "✅ Test 4 PASSED"
echo ""

# Test 5: Verbose mode
echo "Test 5: Verbose Mode"
echo "----------------------------------------"
$AZD_APP health --verbose
echo "✅ Test 5 PASSED"
echo ""

echo "========================================"
echo "Quick Tests Complete!"
echo "========================================"
echo ""
echo "All basic tests passed. Services are running correctly."
echo ""
echo "Next Steps:"
echo "  1. Try streaming mode:    $AZD_APP health --stream"
echo "  2. Test service failure:  kill \$(cat logs/web.pid) && $AZD_APP health"
echo "  3. Full manual testing:   See TESTING.md for comprehensive test guide"
echo ""
echo "To stop all services:"
echo "  ./stop-all.sh"
echo ""
echo "For detailed testing: cat TESTING.md"
echo ""
