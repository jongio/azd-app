# npm Workspace Test Project

This test project validates the npm workspace race condition fix.

## Structure

```
test-npm-workspace/
├── package.json          ← Root workspace configuration
├── azure.yaml
└── packages/
    ├── api/
    │   ├── package.json  ← Workspace package
    │   └── index.js
    └── webapp/
        ├── package.json  ← Workspace package
        └── index.js
```

## Workspace Configuration

The root `package.json` defines workspaces:
```json
{
  "workspaces": ["packages/*"]
}
```

This creates an npm workspace where:
- Shared dependencies are hoisted to root `node_modules/`
- Each package can have its own dependencies
- All packages are installed with a single `npm install --workspaces`

## Testing the Fix

### Test 1: Workspace Detection
```bash
cd cli/tests/projects/node/test-npm-workspace
go run ../../../src/cmd/app deps --verbose
```

**Expected behavior:**
- ✅ Detects workspace root at project root
- ✅ Detects 2 child packages (api, webapp)
- ✅ Only runs `npm install --workspaces` at root
- ✅ Does NOT run separate installs in packages/api and packages/webapp
- ✅ No EBUSY/ENOTEMPTY errors on Windows

### Test 2: Verify Workspace Root Installation
```bash
cd cli/tests/projects/node/test-npm-workspace

# Clean install
rm -rf node_modules packages/*/node_modules

# Run deps
../../../bin/azd.exe app deps
```

**Expected output:**
```
Installing Dependencies

✓ test-npm-workspace (npm)  [━━━━━━━] 100% X.Xs

✓ Installed 1 project(s)
```

**Verification:**
- Root `node_modules/` should exist
- Root `node_modules/lodash` should exist (root dependency)
- Root `node_modules/express` should exist (hoisted from api)
- Root `node_modules/axios` should exist (hoisted from webapp)
- `packages/api/node_modules/` should exist (symlinked workspaces)
- `packages/webapp/node_modules/` should exist (symlinked workspaces)

### Test 3: Run Services
```bash
../../../bin/azd.exe app run
```

**Expected behavior:**
- ✅ Both services start successfully
- ✅ No dependency installation errors
- ✅ Services can import shared dependencies (lodash)

## What This Tests

1. **Workspace Detection:**
   - `HasNpmWorkspaces()` correctly identifies workspace root
   - `FindNodeProjects()` detects 3 projects (root + 2 children)
   - Child projects have `WorkspaceRoot` field set

2. **Smart Installation:**
   - Parallel installer skips child workspace projects
   - Only workspace root is added to installation queue
   - Uses `npm install --workspaces` flag

3. **Race Condition Prevention:**
   - Single npm process instead of 3 parallel processes
   - No concurrent modification of shared `node_modules/`
   - No EBUSY/ENOTEMPTY errors on Windows

4. **Dependency Hoisting:**
   - Shared dependencies (lodash) in root
   - Package-specific dependencies (express, axios) hoisted to root
   - Workspace symlinks created correctly

## Manual Verification

To manually verify workspace detection:

```go
package main

import (
    "fmt"
    "github.com/jongio/azd-app/cli/src/internal/detector"
)

func main() {
    projects, _ := detector.FindNodeProjects(".")
    for _, p := range projects {
        fmt.Printf("Project: %s\n", p.Dir)
        fmt.Printf("  IsWorkspaceRoot: %v\n", p.IsWorkspaceRoot)
        fmt.Printf("  WorkspaceRoot: %s\n", p.WorkspaceRoot)
        fmt.Printf("  PackageManager: %s\n\n", p.PackageManager)
    }
}
```

**Expected output:**
```
Project: /path/to/test-npm-workspace
  IsWorkspaceRoot: true
  WorkspaceRoot: 
  PackageManager: npm

Project: /path/to/test-npm-workspace/packages/api
  IsWorkspaceRoot: false
  WorkspaceRoot: /path/to/test-npm-workspace
  PackageManager: npm

Project: /path/to/test-npm-workspace/packages/webapp
  IsWorkspaceRoot: false
  WorkspaceRoot: /path/to/test-npm-workspace
  PackageManager: npm
```

## Regression Tests

This project should also be used for regression testing:

1. **Before the fix:** Would fail with EBUSY/ENOTEMPTY on Windows
2. **After the fix:** Should succeed with single workspace install

## Related Test Projects

Compare with:
- `test-npm-project` - Independent npm project (not a workspace)
- `boundary-test` - Tests directory traversal boundaries
- `test-pnpm-project` - pnpm package manager (also supports workspaces)
