# Invalid Functions Project - Corrupted host.json

This project is intentionally invalid to test error detection.

## Problem

Corrupted `host.json` with invalid JSON syntax.

## Expected Behavior

- **Detection**: Should fail with clear error message
- **Error Message**: Should mention JSON parse error in host.json
- **azd app run**: Should not start service

## Project Structure

```
functions-invalid-corrupt-host/
├── host.json              # CORRUPTED - invalid JSON
├── package.json           # Valid
└── src/
    └── functions/
        └── http.js        # Valid function definition
```

## Testing

```bash
azd app run
# Expected: Error indicating invalid JSON in host.json
```

## Use Cases

- Tests error detection for malformed host.json
- Validates JSON parsing error messages
- Ensures proper validation of configuration files
