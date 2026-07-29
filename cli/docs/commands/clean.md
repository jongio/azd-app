# azd app clean

Reclaim disk space from build artifacts and caches.

## Synopsis

```
azd app clean [flags]
```

## Description

Remove build output and cache directories for the services defined in `azure.yaml`.

By default `clean` removes build artifacts and caches only. Dependency directories such as `node_modules` and `.venv` are left in place unless you pass `--deps`, because reinstalling them is slower and usually needs the network.

Only directories inside a detected service directory are ever removed, and only when their base name is on the allow list below. Anything outside the project, and any directory whose name is not on the list, is refused.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | `-s` | stringArray | | Limit to specific services (can be specified multiple times) |
| `--deps` | | bool | `false` | Also remove dependency directories (`node_modules`, `.venv`) |
| `--older-than` | | string | `0s` | Only remove artifacts older than this duration (for example, `24h`) |
| `--dry-run` | | bool | `false` | List what would be removed and the reclaimable size without deleting |

## Removable Directories

The removal allow list is fixed. A candidate directory is removed only when its base name appears here.

| Stack | Build output and caches | Dependencies (requires `--deps`) |
|-------|-------------------------|----------------------------------|
| Node | `dist`, `build`, `.next`, `.nuxt`, `out`, `coverage`, `.turbo` | `node_modules` |
| Python | `__pycache__`, `.pytest_cache`, `.mypy_cache`, `.ruff_cache`, `build`, `dist`, `htmlcov`, `.tox` | `.venv`, `venv` |
| .NET | `bin`, `obj` | |

## Examples

### Preview what would be removed

```bash
azd app clean --dry-run
```

The dry run prints each candidate directory, its service, its category, and its size, plus the total reclaimable space. Nothing is deleted.

### Remove build artifacts across all services

```bash
azd app clean
```

### Also remove dependency directories

```bash
azd app clean --deps
```

Run `azd app deps` afterward to reinstall.

### Only remove stale artifacts

```bash
azd app clean --older-than 24h
```

A directory is skipped when it was modified more recently than the given duration. This keeps a clean sweep from deleting output you just built.

### Limit to one service

```bash
azd app clean --service api
```

Repeat `--service` or pass a comma-separated list to target several services.

## Safety

- Every candidate is verified to live inside a detected service directory before removal.
- Every candidate's base name is checked against the allow list. A non-matching name is refused rather than skipped silently.
- `--dry-run` never writes.

## Exit Codes

| Code | Description |
|------|-------------|
| `0` | Clean completed, or there was nothing to clean |
| `1` | Clean failed, for example a directory could not be removed |

## Related Commands

- [azd app deps](deps.md) - Install dependencies for detected projects
- [azd app outdated](outdated.md) - Report outdated dependencies across services
- [azd app run](run.md) - Run the development environment
