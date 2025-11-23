# test-pnpm-workspace

Test project for validating pnpm workspace race condition fix in `azd app`.

## Purpose

This project tests the pnpm workspace handling implementation to ensure:
1. **Workspace detection** - `HasNpmWorkspaces()` correctly identifies `pnpm-workspace.yaml`
2. **Smart installation** - Only the workspace root is installed with `pnpm install --recursive`
3. **Race condition prevention** - No concurrent `pnpm` processes modifying shared `node_modules/`
4. **Dependency hoisting** - pnpm's workspace dependencies are properly linked

## Project Structure

```
test-pnpm-workspace/
├── pnpm-workspace.yaml    ← Defines workspace packages
├── package.json           ← Root package with packageManager field
├── azure.yaml             ← Defines services (api, webapp)
└── packages/
    ├── api/
    │   ├── package.json   ← @test-pnpm-workspace/api (express, dotenv)
    │   └── index.js       ← Express API server (port 3001)
    └── webapp/
        ├── package.json   ← @test-pnpm-workspace/webapp (express, axios)
        └── index.js       ← Web app calling API (port 3002)
```

## Workspace Configuration

### pnpm-workspace.yaml
```yaml
packages:
  - 'packages/*'
```

### package.json (root)
```json
{
  "name": "test-pnpm-workspace",
  "private": true,
  "packageManager": "pnpm@9.0.0"
}
```

## Dependencies

**Root:**
- `typescript` (dev dependency)

**packages/api:**
- `express` - Web server
- `dotenv` - Environment variables

**packages/webapp:**
- `express` - Web server
- `axios` - HTTP client to call API

## Testing the Fix

### 1. Workspace Detection

Run the integration test:
```bash
cd cli
go test ./src/internal/detector -v -run TestPnpmWorkspace
```

**Expected:**
- `HasNpmWorkspaces()` returns `true` for the workspace root
- `FindNodeProjects()` finds 3 projects (root + 2 packages)
- Root has `IsWorkspaceRoot = true`
- Children have `WorkspaceRoot = /path/to/root`

### 2. Dependency Installation

Test with the actual CLI:
```bash
cd cli/tests/projects/node/test-pnpm-workspace

# Clean first (optional)
rm -rf node_modules packages/*/node_modules pnpm-lock.yaml

# Run azd app deps
azd app deps
```

**Expected output:**
```
📦 Installing Dependencies

Found 1 Node.js project(s)

✓ test-pnpm-workspace (pnpm)  [━━━━━━━] 100% 10.5s

✓ Installed 1 project(s)
```

**What should happen:**
1. Only 1 project added to parallel installer (the workspace root)
2. Child projects (`packages/api`, `packages/webapp`) skipped
3. Single `pnpm install --recursive` command executed
4. All dependencies installed to workspace `node_modules/`

### 3. Verify Installation

Check that pnpm workspace structure is correct:
```bash
# Root node_modules should exist
ls node_modules/

# Should have express, axios, dotenv, typescript
ls node_modules/ | grep -E "express|axios|dotenv|typescript"

# Packages should have workspace links (pnpm creates .pnpm virtual store)
ls packages/api/node_modules/.pnpm/
ls packages/webapp/node_modules/.pnpm/
```

### 4. Run Services

Test service execution:
```bash
azd app run
```

**Expected:**
- Both services start successfully
- API service on http://localhost:3001
- WebApp service on http://localhost:3002
- WebApp can call API service

**Test endpoints:**
```bash
curl http://localhost:3001/        # API root
curl http://localhost:3001/health  # API health
curl http://localhost:3002/        # WebApp (calls API)
curl http://localhost:3002/health  # WebApp health
```

### 5. Race Condition Validation

**Before the fix:**
- 3 parallel `pnpm install` processes would run
- Race condition: EBUSY, ENOTEMPTY errors on Windows
- Shared `node_modules/` directory conflicts
- Installation failures

**After the fix:**
- Only 1 `pnpm install --recursive` at workspace root
- No concurrent pnpm processes
- No file locking errors
- Clean installation every time

## pnpm Workspace Behavior

### How pnpm Workspaces Work

1. **pnpm-workspace.yaml** defines workspace packages
2. **Workspace root** contains shared `node_modules/` and `node_modules/.pnpm/` (virtual store)
3. **Package dependencies** are hoisted to root when possible
4. **Workspace links** created in `packages/*/node_modules/`
5. **`pnpm install --recursive`** installs all workspace packages in one operation

### pnpm vs npm Workspaces

| Feature | npm | pnpm |
|---------|-----|------|
| Config file | `package.json` (`workspaces` field) | `pnpm-workspace.yaml` or `package.json` |
| Install flag | `npm install --workspaces` | `pnpm install --recursive` (or `-r`) |
| Hoisting | Default (can cause issues) | Strict (uses virtual store) |
| Disk usage | Higher (duplicates) | Lower (hard links) |
| Speed | Slower | Faster |

## Integration Tests

The project includes integration tests in `workspace_test.go`:

**Test: TestHasNpmWorkspaces_PnpmWorkspaceYaml**
- Verifies `pnpm-workspace.yaml` detection
- Tests both YAML and package.json workspace formats

**Test: TestFindNodeProjects_PnpmWorkspaces**
- Creates test pnpm workspace structure
- Verifies workspace root detection
- Confirms child projects are linked to workspace root

## Comparison with Other Test Projects

| Project | Package Manager | Workspace Type | Purpose |
|---------|----------------|----------------|---------|
| `test-npm-workspace` | npm | npm workspaces | Test npm workspace fix |
| **`test-pnpm-workspace`** | **pnpm** | **pnpm workspaces** | **Test pnpm workspace fix** |
| `test-yarn-project` | yarn | Independent | Test yarn detection |
| `test-pnpm-project` | pnpm | Independent | Test pnpm detection |

## Troubleshooting

### pnpm not found
```bash
npm install -g pnpm
```

### Lock file conflicts
```bash
rm pnpm-lock.yaml
pnpm install
```

### Virtual store issues
```bash
rm -rf node_modules
pnpm install
```

## References

- [pnpm Workspaces Documentation](https://pnpm.io/workspaces)
- [pnpm CLI](https://pnpm.io/cli/install)
- [Workspace fix documentation](../../docs/dev/npm-workspace-race-condition-fix.md)
