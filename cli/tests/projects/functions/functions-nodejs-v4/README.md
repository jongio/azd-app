# Node.js v4 Azure Functions Test Project

## Purpose

This test project validates that `azd app` correctly detects and runs **Azure Functions with Node.js v4** runtime, the modern and recommended Node.js programming model for Azure Functions.

## What Is Being Tested

### Azure Functions Detection
When a directory is detected as an Azure Functions project:
1. Correct runtime identification (Node.js v4)
2. Programming model validation (modern decorator-based)
3. Function definitions are properly parsed
4. `azd app reqs` validates prerequisites
5. `azd app run` successfully starts the Functions host

### Validation Points
- ✅ host.json is valid and recognized
- ✅ functions/ directory with function definitions
- ✅ @azure/functions v4 dependency
- ✅ HTTP triggers properly configured
- ✅ Timer triggers properly configured
- ✅ Azure Functions Core Tools v4.0+ detected
- ✅ Node.js 18+ and npm detected
- ✅ func CLI properly starts the host

## Project Structure

```
functions-nodejs-v4/
├── host.json                    # Azure Functions host configuration
├── package.json                 # npm with @azure/functions v4
├── src/
│   └── functions/
│       ├── httpTrigger.js      # HTTP-triggered function
│       └── timerTrigger.js     # Timer-triggered function
└── README.md                    # This file
```

## Key Configuration

In `package.json`:
```json
{
  "dependencies": {
    "@azure/functions": "^4.0.0"
  }
}
```

In `host.json`:
```json
{
  "version": "2.0",
  "functionTimeout": "00:05:00"
}
```

This is the current standard for Azure Functions Node.js v4.

## Running Tests

### Prerequisites Check
```bash
cd cli/tests/projects/functions/functions-nodejs-v4

# Verify all requirements met
azd app reqs

# Expected output:
# ✓ Azure Functions Core Tools: 4.x.x
# ✓ Node.js: v18.x or v20.x
# ✓ npm: 9.x or higher
# All prerequisites met.
```

### Manual Test
```bash
# Install dependencies
azd app deps

# Start the Functions host
azd app run

# Expected output:
# Worker process started and initialized.
# Available Functions:
# - httpTrigger: [GET,POST] http://localhost:7071/api/httpTrigger
# - timerTrigger: [Timer Schedule Pattern: "0 */5 * * * *"]
```

### Testing HTTP Trigger
```bash
# In another terminal
curl http://localhost:7071/api/httpTrigger?name=World

# Expected response:
# Welcome, World!
```

### Automated Tests
This project is tested via:
- `cli/src/internal/service/functions_test.go` - Detection logic
- `cli/src/internal/service/functions_integration_test.go` - End-to-end tests
- CI/CD pipeline integration tests

## Why This Test Exists

### Problem It Solves
Without this test, we wouldn't validate:
- Correct detection of Node.js v4 Functions
- Modern async/await patterns
- Proper trigger configuration
- HTTP and Timer trigger support
- Correct health check endpoints

### Real-World Scenario
Node.js v4 is the current standard for Azure Functions. New projects should use this model. This test ensures production-ready support for the modern runtime.

## Test Matrix

| Aspect | Expected | Status |
|--------|----------|--------|
| Runtime | Node.js v4 | ✅ |
| Framework | @azure/functions ^4.0.0 | ✅ |
| Host Version | 2.0 | ✅ |
| HTTP Triggers | Supported | ✅ |
| Timer Triggers | Supported | ✅ |
| Decorators | Supported | ✅ |
| Prerequisites | func 4.0+, Node.js 18+ | ✅ |
| Port | 7071 (default) | ✅ |

## Troubleshooting

**"Azure Functions Core Tools not found"**
- Install func CLI: `winget install Microsoft.Azure.FunctionsCoreTools` (Windows)
- Verify: `func --version` (should be 4.0+)

**"No functions found"**
- Ensure functions/ directory exists
- Verify functions are exported with @Function decorators
- Check function.json exists for each function

**"HTTP trigger not responding"**
- Check URL is correct: http://localhost:7071/api/httpTrigger
- Verify function is listed in "Available Functions" output
- Check logs for errors

**"Timer trigger not firing"**
- Verify schedule pattern in function.json: `"0 */5 * * * *"`
- Check logs for trigger events
- Note: Timer triggers log to console every 5 minutes

## Related Test Projects

- [functions-nodejs-v3](../functions-nodejs-v3/) - Legacy Node.js v3 model
- [functions-typescript-v4](../functions-typescript-v4/) - TypeScript variant
- [functions-python-v2](../functions-python-v2/) - Python alternative
- [functions-dotnet-isolated](../functions-dotnet-isolated/) - .NET alternative

