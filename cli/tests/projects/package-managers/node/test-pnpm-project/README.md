# pnpm Package Manager Test Project

## Purpose

This test project validates that `azd app` correctly detects and uses **pnpm** as the package manager for Node.js projects, especially when the `packageManager` field is explicitly set in `package.json`.

## What Is Being Tested

### Package Manager Detection Priority
When multiple package managers could be applicable, azd detects in this order:
1. **packageManager field** in package.json (highest priority) ← This project tests this
2. Lock files (pnpm-lock.yaml, yarn.lock, package-lock.json)
3. Default to npm (lowest priority)

### Validation Points
- ✅ packageManager field is correctly parsed from package.json
- ✅ pnpm is explicitly selected via packageManager field
- ✅ `azd app deps` uses `pnpm install`
- ✅ `azd app run` starts the app with pnpm scripts
- ✅ pnpm lockfile (pnpm-lock.yaml) is respected
- ✅ pnpm-specific features work (workspace support, shared node_modules)

## Project Structure

```
test-pnpm-project/
├── package.json           # pnpm configuration with packageManager field
├── pnpm-lock.yaml        # pnpm lockfile
├── README.md             # This file
└── server.js            # Simple Node.js Express server
```

## Key Configuration

In `package.json`:
```json
{
  "packageManager": "pnpm@8.15.0",
  "scripts": {
    "install": "pnpm install",
    "start": "node server.js"
  }
}
```

The `packageManager` field explicitly declares pnpm v8.15.0, ensuring azd uses pnpm even with alternative lock files present.

## Running Tests

### Manual Test
```bash
cd cli/tests/projects/node/test-pnpm-project

# Check detection
azd app reqs          # Should show pnpm detection

# Install dependencies
azd app deps          # Should use pnpm install

# Run the service
azd app run           # Should start with pnpm start
```

### Expected Behavior
1. Detection identifies pnpm v8.15.0 from packageManager field
2. `azd app deps` executes: `pnpm install`
3. Service starts on port 3000
4. Logs show: "Server running on http://localhost:3000"
5. Dependencies installed in pnpm's shared node_modules (.pnpm)

### Automated Tests
This project is tested via:
- `cli/src/internal/service/detection_test.go` - Detection logic
- `cli/src/internal/executor/pnpm_executor_test.go` - pnpm execution
- Integration tests in CI/CD pipeline

## Why This Test Exists

### Problem It Solves
Without this test, we wouldn't validate:
- Correct parsing of packageManager field for pnpm
- pnpm-specific installation behavior (shared packages)
- pnpm lockfile compatibility
- Explicit pnpm version selection

### Real-World Scenario
A monorepo uses pnpm workspaces for performance and storage efficiency. The packageManager field ensures pnpm is used, enabling workspace support and faster installations compared to npm.

## Test Matrix

| Aspect | Expected | Status |
|--------|----------|--------|
| Detection | pnpm v8.15.0 | ✅ |
| Lock File | pnpm-lock.yaml | ✅ |
| Command | pnpm install | ✅ |
| Start Script | pnpm start | ✅ |
| Port | 3000 | ✅ |
| Workspaces | Supported | ✅ |

## Troubleshooting

**"pnpm not found"**
- Install pnpm globally: `npm install -g pnpm@8.15.0`
- Verify: `pnpm --version`

**"packageManager field not recognized"**
- Ensure Node.js version supports packageManager (18.19+)
- Verify package.json syntax is valid

**"pnpm-lock.yaml conflicts"**
- pnpm-lock.yaml is managed by pnpm, don't edit manually
- Run `pnpm install` to regenerate if corrupted
- Version mismatch may require updating pnpm version

## Related Test Projects

- [test-npm-project](../test-npm-project/) - npm variant
- [test-yarn-project](../test-yarn-project/) - yarn variant
- [test-pnpm-workspace](../test-pnpm-workspace/) - pnpm workspaces
- [test-package-manager-override](../test-package-manager-override/) - Override behavior
