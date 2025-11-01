# Test Projects

This directory contains test projects used to validate the App Extension commands.

## Structure

```
test-projects/
├── node/               # Node.js test projects
│   ├── test-node-project/    (npm with dependencies)
│   └── test-npm-project/     (simple npm project)
├── python/             # Python test projects
│   ├── test-poetry-project/  (poetry)
│   ├── test-python-project/  (pip)
│   └── test-uv-project/      (uv)
└── azure/              # Azure configuration test files
    ├── azure.yaml
    ├── azure-backup.yaml
    └── azure-fail.yaml
```

## Usage

These projects are used to test:
- `azd app deps` - Installing dependencies across different package managers
- `azd app run` - Running development environments
- Detection logic for package managers (npm, pnpm, pip, poetry, uv)

## Running Tests

From the root directory:
```bash
# Test deps command
azd app deps

# Test run command
azd app run
```
