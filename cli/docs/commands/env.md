# azd app env

Print the resolved environment for a service.

## Synopsis

```
azd app env [service] [flags]
```

## Description

Print the effective environment a service receives when it runs.

The output merges the process environment, the azd environment values, an optional `.env` file, generated service URLs, and the service-specific variables from `azure.yaml`, the same way `azd app run` resolves them. Because the resolution code is shared, what `env` prints is what the service gets.

Pass a service name to print its environment, or run without a name to list the available services.

Secret-shaped values are masked by default. Use `--no-mask` to print raw values, for example when piping the output into another command.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--all` | | bool | `false` | Print the resolved environment for every service |
| `--format` | | string | `dotenv` | Output format: `dotenv`, `shell`, `powershell`, or `json` |
| `--keys` | | bool | `false` | Print variable names only |
| `--no-mask` | | bool | `false` | Print raw values instead of masking secret-shaped values |
| `--explain` | | bool | `false` | Show the source of each effective value and any sources it overrode |
| `--diff` | | bool | `false` | Compare the resolved environment of two services (pass two service names) |
| `--env-file` | | string | | Path to a `.env` file to merge, matching `azd app run` |
| `--write` | | bool | `false` | Write the resolved environment to a `.env` file instead of printing it |
| `--out` | | string | | Destination folder for `--write` files (writes `<service>.env`); defaults to each service directory |

The global `--json` flag takes precedence over `--format`.

## Resolution Order

Sources are applied from lowest to highest precedence. A later source overwrites an earlier one.

| Precedence | Source | Origin |
|------------|--------|--------|
| 1 (lowest) | `os` | The process environment |
| 2 | `azd` | The selected azd environment's values |
| 3 | `.env` | The file passed to `--env-file` |
| 4 | `service-url` | URLs generated for sibling services |
| 5 (highest) | `azure.yaml` | The service's own `env` block |

`--explain` prints the winning source for each variable, plus the sources it overrode, ordered highest priority first.

## Output Formats

| Format | Shape |
|--------|-------|
| `dotenv` | `KEY=value` lines |
| `shell` | `export KEY=value` statements |
| `powershell` | `$env:KEY = "value"` assignments |
| `json` | A JSON object; with `--all`, an object keyed by service name |

With `--all`, the `dotenv`, `shell`, and `powershell` formats group each service under a `# <service>` header.

## Examples

### Resolved environment for one service

```bash
azd app env api
```

### Load the environment into your shell

```bash
# bash / zsh
eval "$(azd app env api --format shell --no-mask)"
```

```powershell
# PowerShell
azd app env api --format powershell --no-mask | iex
```

### Machine-readable output

```bash
azd app env api --format json
```

### Show raw values

```bash
azd app env api --no-mask
```

### Explain where each value came from

```bash
azd app env api --explain
```

### Compare two services

```bash
azd app env --diff api web
```

### List variable names only

```bash
azd app env api --keys
```

### Write a .env file

```bash
# Writes api/.env
azd app env api --write

# Writes build/env/<service>.env for every service
azd app env --all --write --out build/env
```

### Resolved environment for every service

```bash
azd app env --all
```

## Masking

Values whose names look secret are replaced with a masked placeholder unless `--no-mask` is passed. Masking applies to every format, including `json`, so piping the default output into a log or an issue report is safe.

## Exit Codes

| Code | Description |
|------|-------------|
| `0` | The environment was resolved and printed or written |
| `1` | The service was not found, the arguments were invalid, or the environment could not be resolved |

## Related Commands

- [azd app run](run.md) - Run the development environment
- [azd app info](info.md) - Show information about running services
- [azd app graph](graph.md) - Show the service dependency graph
