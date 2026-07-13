# azd app hooks

List the project-level lifecycle hooks configured in azure.yaml.

## Synopsis

```
azd app hooks [flags]
```

## Description

The `hooks` command reads `azure.yaml` and lists the lifecycle hooks it defines. For each configured hook it shows the command it runs, the shell it uses, whether it continues on error or needs user interaction, and any per-platform override for Windows or POSIX.

Hooks run around the development lifecycle:

| Hook | When it runs |
|------|--------------|
| `prerun` | before services start |
| `postrun` | after services stop following a run |
| `prestop` | before services are stopped |
| `poststop` | after services are stopped |

Use it to confirm what will run around `azd app run` and `azd app stop` without opening the file. When no hooks are configured the command prints a short message and exits zero.

## Flags

| Flag | Description |
|------|-------------|
| `-o, --output` | Output format (text, json) |

## Examples

### List configured hooks

```bash
azd app hooks
```

```
prerun
  run: npm run setup
  shell: bash
postrun
  run: ./cleanup.sh
  shell: (default)
```

### Hook with a platform override

A hook that defines a Windows or POSIX override shows it on its own line:

```
prerun
  run: ./setup.sh
  shell: (default)
  windows: setup.ps1 (shell: pwsh)
```

### Get JSON output

```bash
azd app hooks --output json
```

```json
[
  {
    "name": "prerun",
    "run": "npm run setup",
    "shell": "bash"
  }
]
```

## See Also

- [azd app run](run.md) - Runs prerun and postrun hooks around services
- [azd app stop](stop.md) - Runs prestop and poststop hooks around shutdown
