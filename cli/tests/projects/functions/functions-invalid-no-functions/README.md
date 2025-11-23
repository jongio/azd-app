# Invalid Functions Project - No Functions Defined

This project is intentionally invalid to test error detection.

## Problem

No function definitions found - empty `src/functions/` directory.

## Expected Behavior

- **Detection**: Should fail with clear error message
- **Error Message**: Should mention no functions found
- **azd app run**: Should not start service or warn about empty project

## Project Structure

```
functions-invalid-no-functions/
├── host.json              # Valid
├── package.json           # Valid
└── src/
    └── functions/         # EMPTY - no function files
```

## Testing

```bash
azd app run
# Expected: Error or warning indicating no functions defined
```

## Use Cases

- Tests error detection for empty functions directory
- Validates helpful error messages
- Ensures project validation catches missing functions
