# azd app outdated

Report outdated dependencies across services.

## Synopsis

```
azd app outdated [flags]
```

## Description

Check every service in `azure.yaml` for outdated dependencies and print one aggregated report.

The package manager is detected per service from the files in the service directory. A service whose package manager is not installed is skipped with a warning rather than failing the whole run, so a mixed-language project still reports on the services it can inspect.

`outdated` only reads. It never installs, upgrades, or edits a manifest.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | `-s` | stringArray | | Limit to specific services (repeatable or comma-separated) |
| `--manager` | | stringArray | | Limit to package managers: `npm`, `pnpm`, `yarn`, `pip`, `dotnet`, or `go` |
| `--ignore` | | stringArray | | Package names to exclude from the report (repeatable or comma-separated) |
| `--format` | | string | `text` | Output format: `text` or `json` |
| `--exit-code` | | bool | `false` | Return a non-zero exit code when any dependency is outdated |

## Detection

| Marker in the service directory | Language | Package manager |
|---------------------------------|----------|-----------------|
| `pnpm-lock.yaml` | Node | `pnpm` |
| `yarn.lock` | Node | `yarn` |
| `package.json` (no other lockfile) | Node | `npm` |
| `requirements.txt`, `pyproject.toml`, `setup.py`, or `Pipfile` | Python | `pip` |
| `*.csproj`, `*.fsproj`, or `*.sln` | .NET | `dotnet` |
| `go.mod` | Go | `go` |

## Underlying Queries

| Package manager | Command |
|-----------------|---------|
| `npm`, `pnpm`, `yarn` | `<manager> outdated --json` |
| `pip` | `pip list --outdated --format=json` |
| `dotnet` | `dotnet list package --outdated --format json` |
| `go` | `go list -u -m -json all` |

## Examples

### Report outdated dependencies for every service

```bash
azd app outdated
```

### Limit to one service

```bash
azd app outdated --service api
```

### Limit to selected package managers

```bash
azd app outdated --manager npm,pip
```

### Machine-readable output

```bash
azd app outdated --format json
```

### Gate CI on outdated dependencies

```bash
azd app outdated --exit-code
```

### Ignore packages you have pinned on purpose

```bash
azd app outdated --exit-code --ignore react,typescript
```

`--ignore` matches names case-insensitively. An ignored package neither shows up in the report nor trips `--exit-code`.

## Exit Codes

| Code | Description |
|------|-------------|
| `0` | The report was produced. Without `--exit-code` this is returned even when dependencies are outdated |
| `1` | The report could not be produced, or `--exit-code` was set and at least one dependency is outdated |

## Related Commands

- [azd app deps](deps.md) - Install dependencies for detected projects
- [azd app clean](clean.md) - Reclaim disk space from build artifacts and caches
- [azd app reqs](reqs.md) - Check and verify required tools
