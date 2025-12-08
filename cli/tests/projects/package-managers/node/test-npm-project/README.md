# npm Package Manager Test Project

## Purpose

This test project validates that `azd app` correctly detects and uses **npm** as the package manager for Node.js projects, especially when the `packageManager` field is explicitly set in `package.json`.

## What Is Being Tested

### Package Manager Detection Priority
When multiple package managers could be applicable, azd detects in this order:
1. **packageManager field** in package.json (highest priority) ← This project tests this
2. Lock files (pnpm-lock.yaml, yarn.lock, package-lock.json)
3. Default to npm (lowest priority)

### Validation Points
- ✅ packageManager field is correctly parsed from package.json
- ✅ npm is explicitly selected despite lock files or other indicators
- ✅ `azd app deps` uses `npm install`
- ✅ `azd app run` starts the app with npm scripts
- ✅ Dependencies are properly installed in node_modules

## Project Structure

```
test-npm-project/
├── package.json           # npm configuration with packageManager field
├── README.md             # This file
└── server.js            # Simple Node.js Express server
```

## Key Configuration

In `package.json`:
```json
{
  "packageManager": "npm@10.5.0",
  "scripts": {
    "install": "npm install",
    "start": "node server.js"
  }
}
```

The `packageManager` field explicitly declares npm v10.5.0, ensuring azd uses npm regardless of any lock files.

## Running Tests

### Manual Test
```bash
cd cli/tests/projects/node/test-npm-project

# Check detection
azd app reqs          # Should show npm detection

# Install dependencies
azd app deps          # Should use npm install

# Run the service
azd app run           # Should start with npm start
```

### Expected Behavior
1. Detection identifies npm v10.5.0 from packageManager field
2. `azd app deps` executes: `npm install`
3. Service starts on port 3000
4. Logs show: "Server running on http://localhost:3000"

### Automated Tests
This project is tested via:
- `cli/src/internal/service/detection_test.go` - Detection logic
- `cli/src/internal/executor/npm_executor_test.go` - npm execution
- Integration tests in CI/CD pipeline

## Why This Test Exists

### Problem It Solves
Without this test, we wouldn't validate:
- Correct parsing of packageManager field
- Priority of packageManager field over other indicators
- Explicit version matching

### Real-World Scenario
A team uses npm exclusively but has git history with pnpm-lock.yaml files or other package managers' artifacts. The packageManager field ensures npm is always used, preventing installation errors.

## Test Matrix

| Aspect | Expected | Status |
|--------|----------|--------|
| Detection | npm v10.5.0 | ✅ |
| Lock File | package-lock.json | ✅ |
| Command | npm install | ✅ |
| Start Script | npm start | ✅ |
| Port | 3000 | ✅ |

## Troubleshooting

**"packageManager field not recognized"**
- Ensure Node.js version supports packageManager (18.19+)
- Verify package.json syntax is valid

**"npm not found"**
- Install Node.js 18+ which includes npm
- Verify: `npm --version`

**"Unexpected package manager detected"**
- Ensure packageManager field is set correctly
- Remove conflicting lock files (pnpm-lock.yaml, yarn.lock)
- Clear npm cache: `npm cache clean --force`

## Related Test Projects

- [test-pnpm-project](../test-pnpm-project/) - pnpm variant
- [test-yarn-project](../test-yarn-project/) - yarn variant
- [test-package-manager-override](../test-package-manager-override/) - Override behavior
- [test-no-packagemanager](../test-no-packagemanager/) - Default npm detection
