# azd app doctor

## Overview

The `doctor` command checks whether your machine is set up to run this project. It is entirely read-only: no services are started, no containers are created, and no files are written.

Where `azd app validate` asks "is `azure.yaml` correct?", `doctor` asks "can this machine actually run it?". Validation inspects the configuration; doctor inspects the configuration *and* the environment around it, including tools on `PATH` and whether the dashboard is already up.

## Purpose

- **Fast triage**: Find the missing tool or bad path before `azd app run` fails partway through startup
- **Environment aware**: Detects which package manager and language toolchain each service actually needs, rather than demanding a fixed list
- **Deterministic**: Checks are sorted by severity, then service, then check ID, so output is stable across runs

## Command Usage

```bash
azd app doctor
```

### Flags

The command takes no positional arguments and adds no flags of its own. It accepts the [global flags](../cli-reference.md#global-flags), plus `--output` from `azd` for machine-readable output.

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | `default` | Output format: 'default' or 'json' (inherited from parent) |

`azure.yaml` is located the same way `azd app run` locates it, by searching the current directory and its parents.

## Severities

| Severity | Meaning | Affects exit code |
|----------|---------|-------------------|
| `pass` | The check succeeded | No |
| `warn` | Advisory. The project can still run, but something is worth knowing | No |
| `fail` | The project is unlikely to run correctly until this is fixed | Yes |

Checks are sorted by severity first, and the severity names sort so that `fail` comes before `pass` and `warn`. Failures therefore appear at the top of the output.

## Checks

| Check ID | Severity | Description | Typical fix |
|----------|----------|-------------|-------------|
| `project.azure_yaml` | fail | `azure.yaml` was not found | Run from an azd project, or create one with `azd app init` |
| `project.root` | pass | Reports the resolved project root | - |
| `config.exists` | pass | Reports the resolved `azure.yaml` path | - |
| `config.parse` | pass / fail | Whether `azure.yaml` parsed | Fix the configuration error reported by the parser |
| `services.defined` | pass / warn | Whether any services are defined | Add services before running `azd app run` |
| `service.project` | pass / warn / fail | Whether each service `project` path exists and is a directory | Create the folder or update `azure.yaml` |
| `tool.available` | pass / fail | Whether each required executable resolves on `PATH` | Install the reported tool |
| `port.valid` | pass / fail | Whether each declared port parses and is in range | Use a host port between 1 and 65535 |
| `port.unique` | fail | Two services declare the same host port and protocol | Give one of the services a unique host port |
| `port.declared` | warn | No host ports are declared anywhere | Declare ports for browser-openable HTTP services |
| `dashboard.state` | pass / warn | Whether the dashboard is currently running | Run `azd app run` to start it |

When `azure.yaml` cannot be found or cannot be parsed, doctor stops there and reports only that failure. Later checks would just be noise built on a configuration it could not read.

### Required tools

`azd` and `git` are always required. Everything else is derived from the services you actually declare, so a Go-only project is never told to install Node.

| Service language | Tool required | How it is chosen |
|------------------|---------------|------------------|
| `node`, `javascript`, `typescript` | `pnpm`, `yarn`, or `npm` | `pnpm-lock.yaml` selects pnpm, `yarn.lock` selects yarn, otherwise npm |
| `python` | `uv`, `poetry`, `pipenv`, or `python` | The detected package manager wins. A service with a `.venv` or `venv` needs nothing, because the virtual environment ships its own interpreter |
| `dotnet`, `.net`, `csharp` | `dotnet` | Always |
| `java` | `gradle` or `mvn` | `gradlew`, `build.gradle`, or `build.gradle.kts` selects Gradle, otherwise Maven |
| `go` | `go` | Always |
| `rust` | `cargo` | Always |

A requirement can accept more than one executable name. Plain Python is satisfied by either `python` or `python3`, since many Linux and macOS distributions ship only `python3`.

Docker is required only for services that `azd app run` actually starts as containers. Declaring `ports:` on an ordinary process service does not pull in a Docker requirement.

### Port checks

Only the same host port on the same protocol conflicts, so `3000/tcp` and `3000/udp` coexist without complaint.

A container service that declares only a container port has its host port assigned at run time, so it can never collide and is always reported as a pass.

## Output

### Standard Output

```bash
$ azd app doctor

azd app doctor
──────────────────────────────

   [FAIL] tool.available: docker was not found on PATH
         fix: Install Docker or a compatible container runtime.
   [PASS] project.root: project root: /path/to/project
   [PASS] config.exists: found /path/to/project/azure.yaml
   [PASS] config.parse: azure.yaml parsed successfully
   [PASS] services.defined: 2 service(s) defined
   [PASS] api service.project: project path exists: ./api
   [PASS] tool.available: azd is available
   [PASS] tool.available: git is available
   [WARN] dashboard.state: dashboard is not running
         fix: Run azd app run to start the dashboard.
```

### JSON Output

```bash
$ azd app doctor --output json
[
  {
    "checkId": "tool.available",
    "severity": "fail",
    "message": "docker was not found on PATH",
    "hint": "Install Docker or a compatible container runtime.",
    "tool": "docker"
  }
]
```

`service`, `hint`, and `tool` are omitted when empty. The `tool` field names the executable a `tool.available` check probed, so consumers don't have to parse `message` to learn which tool is missing.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No failing checks. Warnings may still be present |
| 1 | At least one failing check, or `azure.yaml` could not be found or read |

Warnings never fail the command, so `azd app doctor` stays usable as a required CI check while still surfacing advisory issues.

## Common Use Cases

### 1. Check a machine before the first run

```bash
$ azd app doctor && azd app run
```

### 2. Onboard a new contributor

```bash
# Reports every missing tool at once instead of one failure per run attempt
azd app doctor
```

### 3. List only the missing tools

```bash
azd app doctor --output json | jq -r '.[] | select(.checkId == "tool.available" and .severity == "fail") | .tool'
```

### 4. Count checks by severity

```bash
azd app doctor --output json | jq -r 'group_by(.severity)[] | "\(.[0].severity): \(length)"'
```

## Related Commands

- `azd app validate` - Check `azure.yaml` itself for configuration errors
- `azd app reqs` - Verify and optionally generate the project requirements list
- `azd app ports` - List the host ports each service binds
- `azd app run` - Run the development environment
