# Invalid Functions Project - Missing host.json

This project is intentionally invalid to test error detection.

## Problem

Missing `host.json` file - required for all Azure Functions projects.

## Expected Behavior

- **Detection**: Should fail with clear error message
- **Error Message**: Should mention missing `host.json`
- **azd app run**: Should not start service

## Project Structure

```
functions-invalid-no-host/
├── package.json           # Valid
└── src/
    └── functions/
        └── http.js        # Valid function definition
# MISSING: host.json
```

## Testing

```bash
azd app run
# Expected: Error indicating host.json is required
```

## Use Cases

- Tests error detection for missing host.json
- Validates helpful error messages
- Ensures proper validation before startup
