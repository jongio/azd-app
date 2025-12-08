# Boundary Test Project

This test project verifies that the `app deps` and `app run` commands correctly respect the boundary set by the `azure.yaml` file location.

## Directory Structure

```
boundary-test/
├── package.json              ← PARENT PROJECT (should NOT be found)
└── workspace/
    ├── azure.yaml            ← Workspace root
    ├── web/
    │   └── package.json      ← Should be found
    └── api/
        └── requirements.txt  ← Should be found
```

## Expected Behavior

When running commands from within the `workspace` directory:
- ✅ The `web/package.json` should be detected
- ✅ The `api/requirements.txt` should be detected
- ❌ The parent `package.json` should NOT be detected (it's outside the workspace)

## Testing

1. Navigate to the workspace directory:
   ```bash
   cd workspace
   ```

2. Run the deps command:
   ```bash
   app deps
   ```

3. Verify output shows only 2 projects:
   - 1 Node.js project (web)
   - 1 Python project (api)
   
   The parent `package.json` should NOT appear in the output.

## Why This Matters

Without proper boundary checking, the detector would traverse up the directory tree and find projects outside the workspace, leading to:
- Running services from unintended directories
- Installing dependencies in the wrong locations
- Confusion about which projects are part of the application
