# Package Manager Override Test Project

## Purpose

This test project validates that the `packageManager` field in `package.json` correctly **overrides any existing lock files**, ensuring the specified package manager is used regardless of what lock files are present.

## What Is Being Tested

### Package Manager Detection Priority
When multiple package managers could be applicable, azd detects in this order:
1. **packageManager field** in package.json (highest priority) ← This project tests override behavior
2. Lock files (pnpm-lock.yaml, yarn.lock, package-lock.json)
3. Default to npm (lowest priority)

### Validation Points
- ✅ packageManager field is correctly parsed
- ✅ packageManager field takes precedence over lock files
- ✅ yarn is explicitly selected via packageManager field despite pnpm-lock.yaml existing
- ✅ `azd app deps` uses `yarn install` (not pnpm)
- ✅ `azd app run` starts the app with yarn scripts
- ✅ Old lock file is ignored and replaced by yarn.lock

## Project Structure

```
test-package-manager-override/
├── package.json           # yarn packageManager field
├── pnpm-lock.yaml        # Existing pnpm lockfile (should be ignored)
├── README.md             # This file
└── server.js            # Simple Node.js Express server
```

## Key Configuration

In `package.json`:
```json
{
  "packageManager": "yarn@4.1.0",
  "scripts": {
    "start": "node server.js"
  }
}
```

Even though `pnpm-lock.yaml` exists, the `packageManager` field explicitly declares yarn v4.1.0. azd should use yarn, not pnpm.

## Running Tests

### Manual Test
```bash
cd cli/tests/projects/node/test-package-manager-override

# Check detection
azd app reqs          # Should show yarn detection (not pnpm)

# Install dependencies
azd app deps          # Should use yarn install (not pnpm)

# Verify lock file replacement
ls -la               # Should see pnpm-lock.yaml AND yarn.lock

# Run the service
azd app run          # Should start with yarn start
```

### Expected Behavior
1. Detection identifies yarn v4.1.0 from packageManager field
2. Existing pnpm-lock.yaml is ignored
3. `azd app deps` executes: `yarn install` (not `pnpm install`)
4. yarn creates its own yarn.lock file
5. Service starts on port 3000
6. Logs show: "Server running on http://localhost:3000"

### Automated Tests
This project is tested via:
- `cli/src/internal/service/detection_test.go` - Override behavior validation
- `cli/src/internal/executor/yarn_executor_test.go` - yarn execution with lock override
- Integration tests in CI/CD pipeline

## Why This Test Exists

### Problem It Solves
Without this test, we wouldn't validate:
- Correct override behavior of packageManager field
- Lock file priority handling
- Switching between package managers (migration scenario)
- Explicit version selection overriding lock file hints

### Real-World Scenario
A team migrates from pnpm to yarn. They update the packageManager field but the old pnpm-lock.yaml still exists in version control. This test ensures azd uses yarn (per packageManager field) rather than pnpm (per lock file).

## Test Matrix

| Aspect | Expected | Status |
|--------|----------|--------|
| packageManager Field | yarn@4.1.0 | ✅ |
| Existing Lock File | pnpm-lock.yaml | ✅ |
| Detection | yarn (not pnpm) | ✅ |
| Command | yarn install | ✅ |
| New Lock File | yarn.lock | ✅ |
| Start Script | yarn start | ✅ |
| Port | 3000 | ✅ |

## Troubleshooting

**"pnpm used instead of yarn"**
- Verify packageManager field is in package.json
- Check field is spelled correctly: `packageManager`
- Ensure package.json is valid JSON
- Clear npm cache: `npm cache clean --force`

**"yarn not found"**
- yarn v4+ uses Corepack (built into Node.js 16.9+)
- Verify Node.js version: `node --version` (should be 18+)
- Verify Corepack enabled: `corepack enable`

**"Both lock files present"**
- This is normal and expected
- pnpm-lock.yaml is kept for history/version control
- yarn.lock is created by yarn install
- Both can coexist safely; yarn is used per packageManager field

## Related Test Projects

- [test-npm-project](../test-npm-project/) - Explicit npm
- [test-pnpm-project](../test-pnpm-project/) - Explicit pnpm
- [test-yarn-project](../test-yarn-project/) - Explicit yarn
- [test-no-packagemanager](../test-no-packagemanager/) - Default detection
