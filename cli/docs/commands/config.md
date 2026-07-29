# azd app config

Show the configuration azd app resolved from azure.yaml for each service.

## Synopsis

```
azd app config [service] [flags]
```

## Description

The `config` command reads azure.yaml and prints the configuration for each
service the way azd app resolves it. This is a static view of the parsed model,
so you can confirm what the tool sees without starting anything.

The `project` value is the resolved absolute path for the service project, not
the literal string written in azure.yaml.

For every service it shows:

- `host` (when set)
- `type`, marked `explicit` when set in azure.yaml or `inferred` when derived from the service shape
- `language`, `project`, `command`, and `image` (when set)
- `ports` and `uses` (when set)
- `configured`: the optional blocks present on the service

The optional blocks reported are `docker`, `healthcheck`, `restart`,
`resources`, `logs`, `local`, and `azure`.

## Flags

| Flag | Description |
|------|-------------|
| `-o, --output` | Output format (default, json) |

## Examples

### Configuration for every service

```bash
azd app config
```

### Configuration for a single service

```bash
azd app config api
```

### JSON object keyed by service name

```bash
azd app config --output json
```

The JSON output is an object keyed by service name. Each value uses the same
field names as the text output where possible:

```json
{
  "api": {
    "type": "http",
    "typeSource": "explicit",
    "language": "python",
    "project": "C:\\repo\\api",
    "command": "python -m uvicorn main:app",
    "ports": ["8000"],
    "uses": ["redis"],
    "configured": ["healthcheck", "azure"]
  }
}
```

Unset optional fields are omitted. The `project` field is the resolved absolute
path.

## Notes

- Passing an unknown service name returns an error that lists the available services.
- `--output json` emits an object keyed by service name, which pairs well with tools like `jq`.
- Service-level hooks are not part of the azure.yaml schema. Lifecycle hooks are defined at the project level.
