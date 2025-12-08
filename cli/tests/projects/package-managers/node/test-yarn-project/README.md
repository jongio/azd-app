# Yarn Package Manager Test Project

## Purpose

This test project validates that `azd app` correctly detects and uses **yarn** as the package manager for Node.js projects, especially when the `packageManager` field is explicitly set in `package.json`.

## What Is Being Tested

### Package Manager Detection Priority
When multiple package managers could be applicable, azd detects in this order:
1. **packageManager field** in package.json (highest priority) ← This project tests this
2. Lock files (pnpm-lock.yaml, yarn.lock, package-lock.json)
3. Default to npm (lowest priority)

### Validation Points
- ✅ packageManager field is correctly parsed from package.json
- ✅ yarn is explicitly selected via packageManager field
- ✅ `azd app deps` uses `yarn install`
- ✅ `azd app run` starts the app with yarn scripts
- ✅ yarn.lock file is properly used and updated
- ✅ yarn v4 (Corepack) is properly detected and executed

## Project Structure

```
test-yarn-project/
├── package.json           # yarn configuration with packageManager field
├── yarn.lock             # yarn lockfile
├── README.md             # This file
└── server.js            # Simple Node.js Express server
```

## Key Configuration

In `package.json`:
```json
{
  "packageManager": "yarn@4.1.0",
  "scripts": {
    "install": "yarn install",
    "start": "node server.js"
  }
}
```

The `packageManager` field explicitly declares yarn v4.1.0, ensuring azd uses yarn via Corepack.

## Running Tests

### Manual Test
```bash
cd cli/tests/projects/node/test-yarn-project

# Check detection
azd app reqs          # Should show yarn detection

# Install dependencies
azd app deps          # Should use yarn install

# Run the service
azd app run           # Should start with yarn start
```

### Expected Behavior
1. Detection identifies yarn v4.1.0 from packageManager field
2. Yarn is executed via Node.js Corepack (automatic v4 shim)
3. `azd app deps` executes: `yarn install`
4. Service starts on port 3000
5. Logs show: "Server running on http://localhost:3000"
6. yarn.lock is respected and dependencies installed

### Automated Tests
This project is tested via:
- `cli/src/internal/service/detection_test.go` - Detection logic
- `cli/src/internal/executor/yarn_executor_test.go` - yarn execution
- Integration tests in CI/CD pipeline

## Why This Test Exists

### Problem It Solves
Without this test, we wouldn't validate:
- Correct parsing of packageManager field for yarn
- Yarn v4 Corepack integration
- yarn.lock file handling
- Explicit yarn version selection (especially v4)

### Real-World Scenario
A team standardizes on yarn v4 for its improved performance, security features, and workspace support. The packageManager field ensures yarn is used consistently across all environments, including CI/CD pipelines.

## Test Matrix

| Aspect | Expected | Status |
|--------|----------|--------|
| Detection | yarn v4.1.0 | ✅ |
| Lock File | yarn.lock | ✅ |
| Command | yarn install | ✅ |
| Start Script | yarn start | ✅ |
| Corepack | Auto-shimmed | ✅ |
| Port | 3000 | ✅ |

## Troubleshooting

**"yarn not found"**
- yarn v4+ uses Corepack (built into Node.js 16.9+)
- Verify Node.js version: `node --version` (should be 18+)
- Verify Corepack enabled: `corepack enable`

**"packageManager field not recognized"**
- Ensure Node.js version supports packageManager (18.19+)
- Verify package.json syntax is valid

**"Unknown Yarn version"**
- yarn.lock version might not match packageManager field
- Regenerate yarn.lock: `yarn install`
- Update package.json to match actual yarn version

## Related Test Projects

- [test-npm-project](../test-npm-project/) - npm variant
- [test-pnpm-project](../test-pnpm-project/) - pnpm variant
- [test-package-manager-override](../test-package-manager-override/) - Override behavior
- [test-no-packagemanager](../test-no-packagemanager/) - Default npm detection
