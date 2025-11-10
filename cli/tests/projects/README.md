# Test Projects

This directory contains test projects used to validate the App Extension commands.

## Structure

```
test-projects/
├── node/               # Node.js test projects
│   ├── test-node-project/              # Basic Node.js project (npm default)
│   ├── test-npm-project/               # npm project with packageManager field
│   ├── test-pnpm-project/              # pnpm project with packageManager field
│   ├── test-yarn-project/              # yarn project with packageManager field
│   ├── test-no-packagemanager/         # No packageManager field, defaults to npm
│   └── test-package-manager-override/  # packageManager field overrides lock files
├── python/             # Python test projects
│   ├── test-poetry-project/  (poetry)
│   ├── test-python-project/  (pip)
│   └── test-uv-project/      (uv)
├── boundary-test/      # Tests boundary checking (no parent traversal)
│   ├── package.json          (parent - should NOT be found)
│   └── workspace/
│       ├── azure.yaml        (workspace root)
│       ├── web/              (should be found)
│       └── api/              (should be found)
└── azure/              # Azure configuration test files
    ├── azure.yaml
    ├── azure-backup.yaml
    └── azure-fail.yaml
```

## Node.js Test Projects

The Node.js test projects validate package manager detection with the following priority:
1. **packageManager field** in package.json (highest priority)
2. **Lock files** (pnpm-lock.yaml > pnpm-workspace.yaml > yarn.lock > package-lock.json)
3. **Default to npm** if neither is found

### Test Coverage

- **test-npm-project**: Tests npm with explicit `packageManager: "npm@10.5.0"` field
- **test-pnpm-project**: Tests pnpm with explicit `packageManager: "pnpm@8.15.0"` field
- **test-yarn-project**: Tests yarn with explicit `packageManager: "yarn@4.1.0"` field
- **test-no-packagemanager**: Tests default npm behavior when no packageManager field exists
- **test-package-manager-override**: Tests that `packageManager: "yarn@4.1.0"` overrides existing `pnpm-lock.yaml`
- **test-node-project**: Basic Node.js project setup

## Usage

These projects are used to test:
- `azd app deps` - Installing dependencies across different package managers
- `azd app run` - Running development environments
- Detection logic for package managers (npm, pnpm, pip, poetry, uv)
- **Boundary checking** - Ensuring projects outside `azure.yaml` workspace are not detected

### Boundary Test Project

The `boundary-test/` project specifically tests that the detector functions:
- ✅ Only search within the workspace defined by `azure.yaml` location
- ✅ Do NOT traverse outside the workspace to parent directories
- ❌ Do NOT detect projects in sibling/parent directories

See `boundary-test/README.md` for detailed test instructions.

## Running Tests

From the root directory:
```bash
# Test deps command
azd app deps

# Test run command
azd app run
```
