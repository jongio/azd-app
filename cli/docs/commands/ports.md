# azd app ports

List the host port each service binds, resolved from azure.yaml.

## Synopsis

```
azd app ports [flags]
```

## Description

The `ports` command reads `azure.yaml` and prints the host port each service binds. For every service it shows each port binding as `host -> container/protocol`.

An explicit host port is shown as its number. A port left for the tool to assign (for example a Docker service that only names the container port) is shown as `auto`. When two bindings claim the same explicit host port the command marks the conflict, prints a warning, and exits non-zero. That makes `azd app ports` a quick preflight check before `azd app run`, and it composes well in scripts and CI where a non-zero exit stops the pipeline.

Only explicit host ports can conflict. Auto-assigned ports never count as a conflict because the runtime picks a free port for each one.

## Flags

| Flag | Description |
|------|-------------|
| `-o, --output` | Output format (text, json) |

## Examples

### List ports for every service

```bash
azd app ports
```

```
web
  port: 3000 -> 8080/tcp
api
  port: auto -> 9090/tcp
```

### Report a conflict

When two services bind the same explicit host port, each conflicting line is marked and a warning is printed:

```
web
  port: 3000 -> 8080/tcp  (conflict)
worker
  port: 3000 -> 9090/tcp  (conflict)

Host port 3000 is bound by more than one service: web, worker
```

The command exits non-zero in this case.

### Get JSON output

```bash
azd app ports --output json
```

```json
{
  "api": {
    "ports": [
      { "host": "auto", "container": 9090, "protocol": "tcp" }
    ]
  },
  "web": {
    "ports": [
      { "host": "3000", "hostPort": 3000, "container": 8080, "protocol": "tcp" }
    ]
  }
}
```

## See Also

- [azd app run](run.md) - Start services using these port bindings
- [azd app info](info.md) - Show information about running services
- [azd app add](add.md) - Add a container service with preset ports
