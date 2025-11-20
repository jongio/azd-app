# pnpm Workspace Support

## Overview

Added full support for pnpm workspaces to the workspace race condition fix, including `pnpm-workspace.yaml` detection and proper installation with `pnpm install --recursive`.

## Changes Made

### 1. Workspace Detection (detector.go)

**Enhanced `HasNpmWorkspaces()` function:**
```go
func HasNpmWorkspaces(dir string) bool {
    // Check for pnpm-workspace.yaml first (pnpm-specific workspace configuration)
    pnpmWorkspacePath := filepath.Join(dir, "pnpm-workspace.yaml")
    if err := security.ValidatePath(pnpmWorkspacePath); err == nil {
        if _, err := os.Stat(pnpmWorkspacePath); err == nil {
            return true
        }
    }
    
    // Then check package.json workspaces field...
}
```

**What it detects:**
1. ✅ `pnpm-workspace.yaml` (pnpm-specific)
2. ✅ `package.json` with `workspaces` array (npm/yarn/pnpm)
3. ✅ `package.json` with `workspaces` object (Yarn compatibility)

### 2. pnpm Installation (installer.go)

**Added `--recursive` flag for pnpm workspaces:**
```go
case "pnpm":
    args = []string{"install", "--prefer-offline"}
    // If this is a workspace root, use --recursive flag to install all workspace packages
    if project.IsWorkspaceRoot {
        args = append(args, "--recursive")
    }
```

### 3. Workspace Handler Module (NEW)

**Created `src/internal/workspace` package:**
- Extracted workspace logic from `core.go` into dedicated module
- Cleaner separation of concerns
- Easier to test and maintain

**Key functions:**
- `FilterNodeProjects()` - Filters projects to prevent duplicate installations
- `GetWorkspaceRoots()` - Returns all workspace root directories
- `GetWorkspaceChildren()` - Returns children for a workspace root
- `HasWorkspaces()` - Checks if any projects define workspaces
- `CountWorkspaces()` - Returns counts of roots and children

### 4. Refactored core.go

**Before:**
```go
// 25 lines of nested logic with maps and conditionals
workspaceHandled := make(map[string]bool)
for _, project := range nodeProjects {
    if project.IsWorkspaceRoot {
        parallelInstaller.AddNodeProject(project)
        workspaceHandled[project.Dir] = true
    } else if project.WorkspaceRoot != "" {
        if !workspaceHandled[project.WorkspaceRoot] {
            parallelInstaller.AddNodeProject(project)
        }
    } else {
        parallelInstaller.AddNodeProject(project)
    }
}
```

**After:**
```go
// Clean, simple, testable
workspaceHandler := workspace.NewHandler()
filteredNodeProjects := workspaceHandler.FilterNodeProjects(nodeProjects)

for _, project := range filteredNodeProjects {
    parallelInstaller.AddNodeProject(project)
}
```

## pnpm Workspace Configuration

### Format 1: pnpm-workspace.yaml (pnpm-specific)
```yaml
packages:
  - 'packages/*'
  - 'apps/*'
```

### Format 2: package.json (shared with npm/yarn)
```json
{
  "name": "my-workspace",
  "workspaces": ["packages/*"]
}
```

### Format 3: Both (pnpm-workspace.yaml takes precedence)
pnpm can use both files, but `pnpm-workspace.yaml` is the canonical configuration.

## Test Coverage

### Unit Tests

**File:** `workspace_test.go`
- `TestHasNpmWorkspaces_PnpmWorkspaceYaml` - 4 test cases for pnpm detection
- `TestFindNodeProjects_PnpmWorkspaces` - Full workflow test

**Results:**
```
✓ pnpm-workspace.yaml exists
✓ pnpm-workspace.yaml with package.json workspaces
✓ only package.json workspaces (no pnpm-workspace.yaml)
✓ no workspace configuration
✓ TestFindNodeProjects_PnpmWorkspaces
```

### Integration Tests

**File:** `workspace_integration_test.go`
- `TestPnpmWorkspaceIntegration` - Tests with real pnpm workspace project
- `TestPnpmWorkspaceHasWorkspaces` - Verifies pnpm-workspace.yaml detection

**File:** `workspace/workspace_test.go`
- `TestFilterNodeProjects` - 5 test cases for filtering logic
- `TestGetWorkspaceRoots` - Workspace root extraction
- `TestGetWorkspaceChildren` - Child project extraction
- `TestHasWorkspaces` - Workspace detection
- `TestCountWorkspaces` - Counting roots and children
- `TestIsWorkspaceChild` - Child identification

### Test Project

**Created:** `cli/tests/projects/node/test-pnpm-workspace`

**Structure:**
```
test-pnpm-workspace/
├── pnpm-workspace.yaml    ← Workspace config
├── package.json           ← Root with packageManager field
├── azure.yaml             ← Service definitions
└── packages/
    ├── api/               ← Express API service
    └── webapp/            ← Web app with axios
```

**What it tests:**
1. pnpm-workspace.yaml detection
2. Single workspace install (not 3 parallel)
3. `pnpm install --recursive` usage
4. No EBUSY/ENOTEMPTY errors
5. Proper dependency hoisting

## How It Works

### Detection Flow

```
1. FindNodeProjects(root)
   ├── First pass: Find all package.json files
   │   └── For each: Check HasNpmWorkspaces()
   │       ├── Check pnpm-workspace.yaml exists? → IsWorkspaceRoot = true
   │       └── Check package.json workspaces? → IsWorkspaceRoot = true
   │
   └── Second pass: Link children to workspace roots
       └── For each child: Set WorkspaceRoot if parent has IsWorkspaceRoot

2. FilterNodeProjects(projects)
   ├── Workspace root? → Include
   ├── Workspace child with root in list? → Skip
   └── Independent project? → Include
```

### Installation Flow

```
pnpm workspace root detected
    ↓
FilterNodeProjects() returns only root
    ↓
ParallelInstaller adds only 1 project
    ↓
pnpm install --recursive --prefer-offline
    ↓
All workspace packages installed
    ↓
No race conditions!
```

## Comparison: npm vs pnpm Workspaces

| Feature | npm | pnpm |
|---------|-----|------|
| **Config** | `package.json` only | `pnpm-workspace.yaml` or `package.json` |
| **Install Command** | `npm install --workspaces` | `pnpm install --recursive` |
| **Hoisting** | Default, can cause conflicts | Strict, uses virtual store `.pnpm/` |
| **Disk Usage** | Higher (duplicates dependencies) | Lower (hard links, content-addressable store) |
| **Speed** | Slower | Faster (parallel, cached) |
| **Lock File** | `package-lock.json` | `pnpm-lock.yaml` |
| **Node Modules** | Flat structure | Nested with symlinks |

## Benefits of This Implementation

### 1. Unified Handling
- Single codebase handles npm, yarn, and pnpm workspaces
- Same race condition fix applies to all package managers
- Consistent behavior across different projects

### 2. Clean Architecture
- Workspace logic separated into dedicated module
- Easy to test individual components
- Clear separation of concerns

### 3. Comprehensive Testing
- Unit tests for each function
- Integration tests with real workspace projects
- Test coverage for all workspace scenarios

### 4. Performance
- Single install operation instead of N parallel
- Faster on Windows (no file locking errors)
- Leverages pnpm's built-in workspace efficiency

## Migration Guide

### If you have an npm workspace:
No changes needed - existing detection still works.

### If you have a pnpm workspace:
1. **Using `pnpm-workspace.yaml`:** Automatically detected ✅
2. **Using `package.json` workspaces:** Automatically detected ✅
3. **Run:** `azd app deps` → Single `pnpm install --recursive`

### If you're creating a new pnpm workspace:

**Option 1: pnpm-workspace.yaml (recommended)**
```yaml
packages:
  - 'packages/*'
  - 'apps/*'
```

**Option 2: package.json**
```json
{
  "workspaces": ["packages/*"]
}
```

## Documentation Updates

- ✅ Created `pnpm-workspace-support.md` (this file)
- ✅ Updated `npm-workspace-race-condition-fix.md` to mention pnpm
- ✅ Updated `npm-workspace-fix-summary.md` to include pnpm
- ✅ Created test project README in `test-pnpm-workspace/README.md`
- ✅ Added workspace handler documentation in code comments

## Verification

### Run All Tests
```bash
# Workspace handler tests
go test ./src/internal/workspace -v

# Detector tests (includes pnpm)
go test ./src/internal/detector -v -run "Pnpm|Workspace"

# Integration tests
go test ./src/internal/detector -v -run "Integration"

# Build verification
go build -o bin/azd.exe ./src/cmd/app
```

### Manual Testing
```bash
cd cli/tests/projects/node/test-pnpm-workspace

# Clean
rm -rf node_modules packages/*/node_modules pnpm-lock.yaml

# Test detection
azd app deps

# Expected: Single install operation, no errors

# Test services
azd app run

# Expected: Both services start successfully
```

## References

- [pnpm Workspaces Documentation](https://pnpm.io/workspaces)
- [pnpm CLI - install](https://pnpm.io/cli/install)
- [npm Workspaces](https://docs.npmjs.com/cli/v10/using-npm/workspaces)
- [Yarn Workspaces](https://classic.yarnpkg.com/en/docs/workspaces/)
