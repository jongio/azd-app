# Requirements Generate Test Project

This project tests the `azd app reqs --generate` command with various scenarios.

## Test Directories

| Directory | Scenario | Expected Behavior |
|-----------|----------|-------------------|
| `no-reqs/` | azure.yaml with NO reqs section | Should ADD reqs section with detected requirements |
| `empty-reqs/` | azure.yaml with `reqs: []` | Should REPLACE empty array with detected requirements |
| `partial-reqs/` | azure.yaml with only `node` req | Should ADD missing `pnpm` without duplicating `node` |
| `complete-reqs/` | azure.yaml with ALL reqs present | Should NOT modify the file |

## Usage

```bash
# Test no reqs scenario
cd no-reqs && azd app reqs -g

# Test empty reqs scenario
cd empty-reqs && azd app reqs -g

# Test partial reqs scenario (should add pnpm)
cd partial-reqs && azd app reqs -g

# Test complete reqs scenario (no changes expected)
cd complete-reqs && azd app reqs -g
```

## Expected Detections

Each subdirectory contains a `package.json` with `packageManager: pnpm@9.0.0`, so:
- `node` should be detected (from package.json)
- `pnpm` should be detected (from packageManager field)
