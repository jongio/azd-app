# Discovery Test Project

This project is for manually testing the `azd app test` command's dynamic discovery feature.

## Services

| Service | Language | Framework | Has Tests | Notes |
|---------|----------|-----------|-----------|-------|
| web | js | vitest | ✅ | Standard vitest setup |
| api | ts | jest | ✅ | Standard jest setup |
| backend | python | pytest | ✅ | pytest with pyproject.toml |
| gateway | go | go test | ✅ | Standard go test |
| config | js | none | ❌ | No test script, no test files |
| nested | ts | jest | ✅ | Tests in deeply nested __tests__ folder |

## Expected Behavior

When running `azd app test` from this directory:

1. **Validation phase** should show:
   - ✓ web: vitest detected (1 test file)
   - ✓ api: jest detected (1 test file)
   - ✓ backend: pytest detected (1 test file)
   - ✓ gateway: gotest detected (1 test file)
   - ⚠ config: No test script in package.json (skipping)
   - ✓ nested: jest detected (1 test file)

2. **Config save prompt** should offer to save:
   ```yaml
   services:
     web:
       test:
         framework: vitest
     api:
       test:
         framework: jest
     # etc.
   ```

3. **Test execution** should run 5 services, skip 1

## Testing Commands

```bash
# Run with dry-run to see discovery
azd app test --dry-run

# Run with streaming to see output
azd app test --stream

# Run with auto-save
azd app test --save

# Run specific service
azd app test --service web
```
