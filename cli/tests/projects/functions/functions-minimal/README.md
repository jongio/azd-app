# Azure Functions Minimal Valid Project

Absolutely minimal valid Azure Functions project for testing.

## Purpose

Tests the minimum requirements for a valid Azure Functions project:
- `host.json` present
- At least one function defined
- Valid configuration

## Project Structure

```
functions-minimal/
├── host.json              # Minimal configuration (version only)
├── package.json           # Minimal dependencies
└── src/
    └── functions/
        └── http.js        # Single minimal HTTP function
```

## Expected Behavior

- **Detection**: Should be detected as valid Functions project
- **Startup**: Should start successfully with `azd app run` or `func start`
- **Endpoint**: `http://localhost:7071/api/http` returns `200 OK`

## Testing

```bash
azd app run
curl http://localhost:7071/api/http
# Expected: "OK"
```

## Use Cases

- Validates minimum requirements checker
- Tests edge case of minimal configuration
- Baseline for error scenario comparisons
