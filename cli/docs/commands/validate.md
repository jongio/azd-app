# azd app validate

## Overview

The `validate` command inspects `azure.yaml` and reports configuration problems **before** you run anything. It is entirely read-only: no services are started, no containers are created, and no files are written.

## Purpose

- **Fast feedback**: Catch configuration mistakes without waiting for `azd app run` to fail
- **CI friendly**: Non-zero exit code plus machine-readable JSON makes it easy to gate pull requests
- **Deterministic**: Findings are sorted by service, then by check ID, so output is stable across runs

## Command Usage

```bash
azd app validate
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | `default` | Output format: 'default' or 'json' (inherited from parent) |

The command takes no positional arguments. `azure.yaml` is located the same way `azd app run` locates it — by searching the current directory and its parents.

## Execution Flow

```
┌─────────────────────────────────────────────────────────────┐
│                 azd app validate                             │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Locate and read azure.yaml                                  │
│  - Search current directory and parents                      │
│  - Parse YAML (yaml.parse on syntax errors)                  │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Run structured per-service checks                           │
│  - Service names, project paths, uses, ports, type, mode     │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Parser backstop (only when no errors were found)            │
│  - Catches problems the structured checks do not cover       │
│    (for example invalid customUrl / customDomain values)     │
│  - Skipped when errors exist so one root cause yields        │
│    exactly one finding                                       │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Sort findings and render (text or JSON)                     │
│  - Exit non-zero if any finding has severity "error"         │
└─────────────────────────────────────────────────────────────┘
```

## Checks

| Check ID | Severity | Description | Typical fix |
|----------|----------|-------------|-------------|
| `yaml.parse` | error | `azure.yaml` is not valid YAML | Fix the YAML syntax |
| `schema.parse` | error | The parser rejected the configuration | Fix the service configuration reported by the parser |
| `services.empty` | warning | No services are defined | Add services before running `azd app run` |
| `service.name` | error | Service name contains unsupported characters | Use letters, numbers, dot, underscore, or hyphen |
| `project.missing` | error | `project` path does not exist | Create the folder or update the project path |
| `project.not-directory` | error | `project` path is not a directory | Point `project` to a directory |
| `project.outside-root` | error | `project` path resolves outside the project root | Keep service project paths inside the repository |
| `uses.unknown` | error | `uses` entry is not a defined service or resource | Add the dependency or remove it from `uses` |
| `port.invalid` | error | Host port is not between 1 and 65535 | Use a host port such as `8080:80` |
| `port.duplicate` | error | Two services request the same host port | Assign a unique host port to one of the services |
| `type.unsupported` | error | Service `type` is not recognized | Use `http`, `tcp`, `process`, or `container` |
| `mode.unsupported` | error | Service `mode` is not recognized | Use `watch`, `build`, `daemon`, or `task` |
| `command.missing` | warning | Process service has no command in `azure.yaml` | Set `command`, or rely on detection to infer it |

### Port parsing

Host ports are read from the second-to-last `:`-separated segment, so all of these resolve to host port `8080`:

```yaml
ports:
  - "8080"              # bare host port
  - "8080:80"           # host:container
  - "127.0.0.1:8080:80" # bind address
  - "[::1]:8080:80"     # IPv6 bind address
```

A `/tcp` or `/udp` protocol suffix is ignored.

### Single finding per root cause

The structured checks are precise and per-service; the parser backstop is fail-fast and reports a single opaque error. To avoid reporting one problem twice, the backstop runs **only when the structured checks found no errors**. An out-of-root `project` path therefore produces one `project.outside-root` finding, not a `project.outside-root` plus a duplicate `schema.parse`.

## Output

### Standard Output

A valid configuration:

```bash
$ azd app validate

azd app validate
──────────────────────────────

✓ azure.yaml is valid
```

A configuration with findings:

```bash
$ azd app validate

azd app validate
──────────────────────────────

   [FAIL] web port.duplicate: host port 8080 is also used by service "api"
         fix: Assign a unique host port to one of the services.
   [WARN] worker command.missing: process service has no command in azure.yaml
         fix: Set command or make sure detection can infer how to run it.
```

### JSON Output

```bash
$ azd app validate --output json
[
  {
    "file": "/path/to/azure.yaml",
    "service": "web",
    "checkId": "port.duplicate",
    "severity": "error",
    "message": "host port 8080 is also used by service \"api\"",
    "hint": "Assign a unique host port to one of the services."
  }
]
```

`service` and `hint` are omitted when empty. A valid configuration emits `[]`.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No error-severity findings (warnings may still be present) |
| 1 | At least one error-severity finding, or `azure.yaml` could not be found or read |

Warnings never fail the command. This keeps `azd app validate` usable as a required CI check while still surfacing advisory issues.

## Common Use Cases

### 1. Check a project before the first run

```bash
$ azd app validate && azd app run
```

### 2. Gate a pull request in CI

```bash
# Fails the job when azure.yaml has error-severity findings
azd app validate
```

### 3. Report only the failing check IDs

```bash
azd app validate --output json | jq -r '.[] | select(.severity == "error") | .checkId'
```

### 4. Count findings by severity

```bash
azd app validate --output json | jq -r 'group_by(.severity)[] | "\(.[0].severity): \(length)"'
```

## Related Commands

- `azd app init` - Generate or enrich `azure.yaml` from your project
- `azd app run` - Run the development environment (performs the same parsing at startup)
- `azd app reqs` - Verify that required tools are installed
