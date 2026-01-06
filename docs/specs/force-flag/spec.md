# Add --force Flag to azd app run

## Overview

Add a `--force` flag to the `azd app run` command that propagates down to `azd app deps --force`, providing a convenient way to force clean reinstall of all dependencies before starting services.

## Background

Currently, users who want to force a clean dependency reinstall before running their services must manually run:

```bash
azd app deps --force
azd app run
```

This spec proposes adding `--force` to `azd app run` to streamline this common workflow.

## Goals

1. Add `--force` flag to `azd app run` command
2. Propagate the flag to `azd app deps` via the orchestrator
3. Update all CLI documentation (markdown files)
4. Update all web documentation (Astro pages)
5. Ensure consistency across all documentation

## Non-Goals

- Changing the behavior of `--force` flag itself
- Modifying the deps command beyond flag propagation
- Adding any other new flags

## Design

### Flag Definition

Add to `run.go`:

```go
var (
    // ... existing flags ...
    runForce bool  // Force clean dependency reinstall
)
```

```go
cmd.Flags().BoolVar(&runForce, "force", false, "Force clean dependency reinstall (passes --force to deps)")
```

### Flag Propagation

The orchestrator already handles command dependencies. The flag needs to be propagated through the orchestrator's context or the global `DepsOptions`.

Since `deps.go` already has a global `DepsOptions` structure that is accessed via `GetDepsOptions()`, we can set the force flag before the orchestrator runs deps:

```go
func runWithServices(ctx context.Context, _ *cobra.Command, _ []string) error {
    // ... existing validation ...
    
    // Set deps options before orchestrator runs
    if runForce {
        opts := GetDepsOptions()
        opts.Force = true
        setDepsOptions(opts)
    }
    
    // Execute dependencies first (reqs -> deps -> run)
    if err := cmdOrchestrator.Run("run"); err != nil {
        return fmt.Errorf("failed to execute command dependencies: %w", err)
    }
    
    // ... rest of run logic ...
}
```

### Documentation Updates

#### CLI Documentation

Files to update:
- `cli/docs/commands/run.md` - Add `--force` flag to table and examples
- `cli/docs/commands/deps.md` - No changes needed (already documents --force)

#### Web Documentation

Files to update:
- `web/src/pages/reference/cli/run.astro` - Add `--force` flag to flags section
- `web/src/pages/reference/cli/deps.astro` - No changes needed (already documents --force)

#### Other Documentation

Search for any mentions of `azd app run` or `azd app deps` workflow and update if relevant:
- README.md
- CONTRIBUTING.md
- Any guides or tutorials

## Implementation Plan

### Phase 1: Code Changes

1. Add `runForce` variable to `run.go`
2. Add flag definition to `NewRunCommand()`
3. Add flag propagation logic in `runWithServices()`
4. Test locally

### Phase 2: Documentation Updates

1. Update `cli/docs/commands/run.md`
   - Add to flags table
   - Add usage examples
2. Update `web/src/pages/reference/cli/run.astro`
   - Add to flags section
   - Add usage examples
3. Search for and update any other mentions

### Phase 3: Verification

1. Build and test the CLI
2. Verify flag works as expected
3. Verify docs render correctly
4. Check all links and references

## Testing

### Manual Testing

```bash
# Test force flag
azd app run --force

# Should output something like:
# Running deps with --force flag
# Cleaning dependencies...
# Installing dependencies...
# Starting services...

# Test that it actually cleans
# 1. Run without --force
azd app run
# 2. Check that node_modules exists
# 3. Run with --force
azd app run --force
# 4. Verify deps were cleaned and reinstalled
```

### Documentation Testing

1. Build web docs: `cd web && pnpm build`
2. Check rendered output
3. Verify all code examples are accurate
4. Check all internal links work

## Documentation Patterns

### Flags Table Format

```markdown
| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--force` | | bool | `false` | Force clean dependency reinstall (passes --force to deps) |
```

### Usage Examples Format

```bash
# Force clean dependency reinstall before running
azd app run --force

# Combine with other flags
azd app run --force --web
azd app run --force --service api
```

## Success Criteria

- [ ] `--force` flag added to `azd app run` command
- [ ] Flag correctly propagates to `azd app deps`
- [ ] All CLI markdown documentation updated
- [ ] All web Astro documentation updated
- [ ] Manual testing passes
- [ ] Documentation renders correctly
- [ ] All internal links work
