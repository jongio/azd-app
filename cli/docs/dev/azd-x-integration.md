# azd x Integration Analysis

## Overview

This document analyzes what the `cli` project needs to adopt from `cli2/jongio.azd.app` to fully support `azd x` extension commands, and how to integrate these into the existing build and release workflows.

## Current State Comparison

### jongio.azd.app (Built with `azd x init`)
- ✅ **extension.yaml**: Complete with `lifecycle-events` and `mcp-server` capabilities
- ✅ **Build script**: Simple `build.ps1` that uses environment variables
- ✅ **Minimal structure**: Only context, prompt, listen, version commands
- ✅ **Capabilities**: `custom-commands`, `lifecycle-events`, `mcp-server`

### cli (Legacy approach)
- ⚠️ **extension.yaml**: Only has `custom-commands` capability
- ✅ **Mage-based build**: Comprehensive build system with version bumping
- ✅ **Rich features**: Dashboard, service orchestration, health checks, etc.
- ✅ **GitHub Actions**: Automated release workflows
- ❌ **Missing**: `lifecycle-events` and `mcp-server` capabilities declaration

## What's Missing from `cli`

### 1. Extension Capabilities Declaration

**File**: `cli/extension.yaml`

**Missing capabilities**:
```yaml
capabilities:
  - custom-commands
  - lifecycle-events  # ← MISSING
  - mcp-server        # ← MISSING
```

**Impact**: These capabilities unlock additional functionality but may not be needed initially.

- `lifecycle-events`: Allows extensions to hook into azd lifecycle (preprovision, postdeploy, etc.)
- `mcp-server`: Enables Model Context Protocol server for AI integrations

**Status**: The `cli` already has lifecycle event handling via the `listen` command and `azdext.NewExtensionHost()`, so adding this capability is just declaring what already exists.

### 2. Platform-specific Executables

**File**: `cli/extension.yaml`

**jongio.azd.app has**:
```yaml
platforms:
  windows:
    executable: app.exe
  linux:
    executable: app
  darwin:
    executable: app
```

**cli has**:
```yaml
entryPoint: app  # Only one entry point
```

**Impact**: The `platforms` section is more explicit but may not be required. The GitHub Actions workflow already builds platform-specific binaries.

### 3. Build Environment Variables

**jongio.azd.app build.ps1** expects:
- `$env:EXTENSION_ID` - Extension identifier
- `$env:EXTENSION_VERSION` - Version to build
- `$env:EXTENSION_PLATFORM` - Optional platform override
- `$env:OUTPUT_DIR` - Output directory

**cli** currently:
- Reads version from `version.txt`
- Auto-bumps version on each build
- Uses hardcoded paths

**Gap**: The `cli` doesn't use the standard environment variables that `azd x build` provides.

## Integration Strategy

### Option 1: Adopt `azd x` Commands Fully (Recommended)

Replace the current Mage-based workflow with `azd x` commands for extension-specific tasks.

**Pros**:
- Uses official azd tooling
- Automatic packaging and registry management
- Watch mode built-in
- Simpler for contributors

**Cons**:
- Lose some custom Mage features (test coverage, linting, etc.)
- Need to migrate existing workflows

**Implementation**:

1. **Update extension.yaml**:
```yaml
capabilities:
  - custom-commands
  - lifecycle-events
  - mcp-server

platforms:
  windows:
    executable: app.exe
  linux:
    executable: app
  darwin:
    executable: app
```

2. **Create `azd-build.ps1`** (wrapper for `azd x build`):
```powershell
# Sets up environment for azd x build
$env:EXTENSION_ID = "jongio.azd.app"
$env:EXTENSION_VERSION = (Get-Content version.txt).Trim()
azd x build --all
```

3. **Keep Mage for development tasks**:
- `mage lint` - Linting
- `mage test` - Testing
- `mage coverage` - Coverage
- `mage dashboardbuild` - Dashboard build
- `mage preflight` - Pre-commit checks

4. **Use `azd x` for extension tasks**:
- `azd x build` - Build extension
- `azd x watch` - Watch and rebuild
- `azd x pack` - Package artifacts
- `azd x publish` - Publish to registry

5. **Update GitHub Actions**:
```yaml
# Instead of manual Go builds, use:
- name: Build extension
  run: |
    cd cli
    $env:EXTENSION_ID = "jongio.azd.app"
    $env:EXTENSION_VERSION = "${{ inputs.version }}"
    azd x build --all
```

### Option 2: Hybrid Approach (Keep Both)

Keep Mage for full control, use `azd x` only for local development convenience.

**Pros**:
- No disruption to existing workflows
- Full control over release process
- Custom build options remain

**Cons**:
- Dual maintenance
- Confusion about which to use

### Option 3: Mage Only (Current State)

Keep everything as-is, don't adopt `azd x` commands.

**Pros**:
- No changes needed
- Known, working process

**Cons**:
- Manual packaging
- No official azd extension tooling
- Contributors can't use `azd x watch`

## Recommended Implementation Plan

### Phase 1: Add Missing Capabilities ✅

1. Update `cli/extension.yaml`:
   - Add `lifecycle-events` capability
   - Add `mcp-server` capability (optional)
   - Add explicit `platforms` section

### Phase 2: Enable `azd x watch` 🎯

This is the biggest value-add for developers.

1. Create `cli/scripts/azd-watch.ps1`:
```powershell
$env:EXTENSION_ID = "jongio.azd.app"
azd x watch
```

2. Update Magefile:
```go
// AzdWatch uses azd x watch for automatic rebuilds
func AzdWatch() error {
    return sh.RunV("azd", "x", "watch")
}
```

3. Update `cli/README.md` to document both:
   - `mage watch` - Custom PowerShell watcher (current)
   - `mage azdwatch` - Official azd x watch

### Phase 3: Integrate with Release Workflow 🚀

Keep GitHub Actions as-is but add `azd x pack` and `azd x publish` as optional steps.

1. **Keep current release process** (GitHub Actions builds, creates releases)
2. **Add post-release automation**:
```yaml
- name: Publish to extension registry (optional)
  run: |
    cd cli
    azd x publish \
      --registry ../registry.json \
      --version ${{ inputs.version }} \
      --artifacts "./app-*.{zip,tar.gz}" \
      --repo jongio/azd-app
```

### Phase 4: Documentation Updates 📚

1. Update `cli/README.md` with `azd x` command usage
2. Document when to use Mage vs `azd x`
3. Add contributor guide for both workflows

## Version Management Strategy

### Current State

- `cli/version.txt` - Source of truth, auto-bumped by Mage
- `cli/extension.yaml` - Manually updated
- `registry.json` - Updated by GitHub Actions

### Recommended

1. **version.txt remains source of truth**
2. **Mage build reads it** (current behavior)
3. **azd x build reads it via env var**:
```powershell
$env:EXTENSION_VERSION = (Get-Content version.txt).Trim()
azd x build
```
4. **GitHub Actions syncs all three**

## Migration Checklist

- [ ] Update `cli/extension.yaml` with missing capabilities
- [ ] Test that `azd x build` works in `cli/` directory
- [ ] Create `AzdWatch()` Mage target
- [ ] Document `azd x` commands in README
- [ ] Add environment variable setup to build scripts
- [ ] Test `azd x pack` for packaging
- [ ] Evaluate `azd x publish` for registry updates
- [ ] Update contributor documentation

## Recommendation

**Start with Phase 1 and Phase 2** - these provide immediate value:
1. Declare all capabilities in `extension.yaml`
2. Enable `azd x watch` for developers
3. Keep existing Mage and GitHub Actions workflows unchanged

This gives developers the benefit of `azd x watch` without disrupting the proven release process.

Later, evaluate Phase 3 (`azd x publish`) if automating registry updates becomes valuable.

## Key Findings

### What `azd x` Commands Actually Do

1. **`azd x build`**:
   - Reads `extension.yaml` for configuration
   - Builds Go binary with version ldflags
   - Supports `--all` for multi-platform builds
   - Auto-installs locally unless `--skip-install`

2. **`azd x pack`**:
   - Creates platform-specific archives (zip/tar.gz)
   - Includes `extension.yaml` in each package
   - Outputs to registry-compatible structure

3. **`azd x publish`**:
   - Updates registry.json with checksums
   - Can create GitHub releases
   - Handles artifact uploads

4. **`azd x watch`**:
   - Monitors source files
   - Auto-rebuilds on changes
   - Auto-reinstalls extension

### What We Already Have

- ✅ Comprehensive Mage build system
- ✅ Dashboard build pipeline
- ✅ Multi-platform builds in GitHub Actions
- ✅ Automated releases
- ✅ Version management
- ✅ Custom watch script with debouncing

### What We Gain from `azd x`

- 🎯 **Standard extension workflow** - Other contributors familiar with `azd x`
- 🎯 **Built-in watch mode** - Simpler than custom PowerShell
- 🎯 **Official packaging** - Registry-compatible by default
- 🎯 **Integrated publish** - Can automate registry updates

### What We'd Lose

- ⚠️ Custom version bumping logic
- ⚠️ Dashboard-aware builds
- ⚠️ Integrated linting/testing
- ⚠️ Preflight checks

## Conclusion

**Recommended Approach**: **Hybrid with `azd x` for development**

1. **Keep Mage** for:
   - Linting, testing, coverage
   - Dashboard builds
   - Preflight checks
   - Release preparation

2. **Add `azd x`** for:
   - `azd x watch` - Developer convenience
   - `azd x build` - Quick local builds
   - Optional: `azd x publish` - Registry automation

3. **GitHub Actions** stays the same:
   - Current workflow is robust
   - Can optionally add `azd x publish` later

This gives the best of both worlds: powerful custom tooling where needed, standard azd tooling for common tasks.
