# Dependency Management

This document describes how to manage dependencies across the azd-app project.

## Overview

The project has dependencies in multiple locations:
- **Go modules** (`cli/go.mod`) - Backend CLI code
- **Dashboard npm packages** (`cli/dashboard/package.json`) - React/TypeScript UI
- **Website npm packages** (`web/package.json`) - Astro documentation site

## Checking for Outdated Dependencies

To check for outdated dependencies across all projects:

```bash
cd cli
mage checkDeps
```

This command will:
1. Check Go modules for available updates
2. Check for deprecated Go modules
3. Check dashboard pnpm packages for updates
4. Check website pnpm packages for updates
5. Check for security vulnerabilities in pnpm packages

The command **does not fail the build** - it only reports available updates.

### Example Output

```
📦 Checking Go modules for updates...
   Available Go module updates:
   • github.com/spf13/cobra: v1.10.1 -> v1.10.2
   • golang.org/x/net: v0.47.0 -> v0.48.0

🔍 Checking Go modules for deprecation notices...
   ⚠️  Deprecated Go modules found:
   • github.com/golang/protobuf: DEPRECATED - Use google.golang.org/protobuf

📦 Checking dashboard pnpm packages for updates...
   Available pnpm package updates:
   ┌────────────────────────────────────────┬─────────┬─────────┐
   │ Package                                │ Current │ Latest  │
   ├────────────────────────────────────────┼─────────┼─────────┤
   │ react                                  │ 19.2.1  │ 19.2.3  │
   │ vite (dev)                             │ 7.2.6   │ 7.3.0   │
   └────────────────────────────────────────┴─────────┴─────────┘

📦 Checking website pnpm packages for updates...
   Available pnpm package updates:
   ┌────────────────────────────────────────┬─────────┬─────────┐
   │ Package                                │ Current │ Latest  │
   ├────────────────────────────────────────┼─────────┼─────────┤
   │ astro                                  │ 5.16.4  │ 5.16.6  │
   └────────────────────────────────────────┴─────────┴─────────┘

🔒 Checking dashboard pnpm packages for security vulnerabilities...
   ✅ No known dashboard security vulnerabilities!

🔒 Checking website pnpm packages for security vulnerabilities...
   ✅ No known website security vulnerabilities!

💡 Tip: Run 'mage updateDeps' to update all dependencies
```

## Updating Dependencies

### Update All Dependencies

To update all dependencies to their latest versions:

```bash
cd cli
mage updateDeps
```

This will:
1. Update all Go modules to latest versions
2. Run `go mod tidy` to clean up
3. Update dashboard pnpm packages to latest
4. Update website pnpm packages to latest
5. Run `pnpm audit --fix` to fix security vulnerabilities

### Safe Update Mode (Minor Only)

To only update to latest minor/patch versions (avoids breaking major version changes):

```bash
cd cli
$env:MINOR_ONLY="true"
mage updateDeps
```

This is safer and less likely to introduce breaking changes.

### Preview Updates (Dry Run)

To see what would be updated without making changes:

```bash
cd cli
$env:DRY_RUN="true"
mage updateDeps
```

### Update Specific Categories

To update only specific dependency types:

**Go modules only:**
```bash
cd cli
go get -u ./...
go mod tidy
```

**Dashboard packages only:**
```bash
cd cli
pnpm update --dir dashboard --latest
```

**Website packages only:**
```bash
cd cli
pnpm update --dir ../web --latest
```

## Workflow After Updating

After running `mage updateDeps`, follow these steps:

```bash
# 1. Review changes
git diff

# 2. Test the build
mage build

# 3. Run all tests
mage test

# 4. Run linter
mage lint

# 5. Run preflight checks
mage preflight

# 6. Commit changes
git add .
git commit -m "chore: update dependencies"
```

## Understanding Update Strategies

### Go Modules

- **Latest patch**: `go get -u=patch ./...` - Only updates to latest patch versions (1.2.3 → 1.2.4)
- **Latest minor**: Default behavior - Updates to latest minor/patch (1.2.3 → 1.3.0)
- **Latest major**: `go get -u ./...` - Updates to latest major version (1.2.3 → 2.0.0)

After updating, always run `go mod tidy` to clean up unused dependencies.

### pnpm Packages

- **Respect semver**: `pnpm update` - Updates within version ranges in package.json
- **Latest versions**: `pnpm update --latest` - Updates to latest regardless of semver ranges
- **Interactive**: `pnpm update --interactive` - Choose which packages to update

### Security Updates

Security vulnerabilities should be fixed immediately:

```bash
# Dashboard
pnpm audit --dir dashboard
pnpm audit --fix --dir dashboard

# Website
pnpm audit --dir ../web
pnpm audit --fix --dir ../web

# Go modules
govulncheck ./...
```

## Deprecated Dependencies

When `checkDeps` reports deprecated modules:

1. **Check replacement**: Read the deprecation message for recommended replacement
2. **Update imports**: Replace old import paths with new ones
3. **Test thoroughly**: Breaking changes are likely
4. **Document**: Note the migration in CHANGELOG.md

Example:
```
⚠️  Deprecated Go modules found:
• github.com/golang/protobuf: DEPRECATED - Use google.golang.org/protobuf
```

Action: Replace all `github.com/golang/protobuf` imports with `google.golang.org/protobuf`.

## Best Practices

### Regular Updates

- **Weekly**: Run `mage checkDeps` to monitor outdated dependencies
- **Monthly**: Run `mage updateDeps` with `MINOR_ONLY=true` for safe updates
- **Quarterly**: Run `mage updateDeps` for full updates including major versions

### Before Releases

Always run `mage preflight` before releases, which includes `checkDeps`.

### Testing After Updates

Priority testing after dependency updates:

1. **Build**: Ensure clean build
2. **Unit tests**: Catch breaking API changes
3. **Integration tests**: Verify external integrations still work
4. **E2E tests**: Ensure dashboard and website work end-to-end
5. **Manual testing**: Test critical user flows

### Breaking Changes

When major version updates introduce breaking changes:

1. Read the CHANGELOG of the updated dependency
2. Update code to match new APIs
3. Add migration notes to our CHANGELOG.md
4. Consider creating a separate PR for major updates

## Automation

The `checkDeps` command is included in the `preflight` target, so it runs automatically before releases.

To add it to CI:

```yaml
# .github/workflows/ci.yml
- name: Check Dependencies
  run: |
    cd cli
    mage checkDeps
```

## Troubleshooting

### Update Fails

If `mage updateDeps` fails:

1. Check error messages for specific packages
2. Try updating categories individually
3. Check for breaking changes in changelogs
4. Revert and update problematic packages manually

### Conflicts After Update

If tests fail after updating:

```bash
# Revert all changes
git checkout go.mod go.sum
git checkout dashboard/package.json dashboard/pnpm-lock.yaml
git checkout ../web/package.json ../web/pnpm-lock.yaml

# Update one category at a time
go get -u ./...
go mod tidy
mage test  # Test after each update

pnpm update --dir dashboard --latest
mage dashboardtest

pnpm update --dir ../web --latest
mage websitetest
```

### Version Pinning

To pin a specific version (prevent updates):

**Go modules** - Use `replace` directive in go.mod:
```go
replace github.com/problematic/module => github.com/problematic/module v1.2.3
```

**pnpm packages** - Use exact version in package.json:
```json
{
  "dependencies": {
    "problematic-package": "1.2.3"  // No ^ or ~ prefix
  }
}
```

## Related Commands

- `mage checkDeps` - Check for outdated dependencies
- `mage updateDeps` - Update all dependencies
- `mage preflight` - Run all pre-release checks (includes checkDeps)
- `mage vulncheck` - Check for security vulnerabilities in Go modules

## References

- [Go Modules Documentation](https://go.dev/doc/modules)
- [pnpm Update Documentation](https://pnpm.io/cli/update)
- [Semantic Versioning](https://semver.org/)
- [Go Vulnerability Database](https://pkg.go.dev/vuln/)
