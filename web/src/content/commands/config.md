---
title: "azd app config"
description: "Show the effective service configuration resolved from azure.yaml."
command: "azd app config"
category: "config"
order: 0
---

# azd app config

Show the configuration that azd app resolved from azure.yaml for each service.

```bash
azd app config [service] [flags]
```

The command prints a static view of the parsed service model so you can confirm
what azd app sees without starting services.

## Output

For each service, the text output shows fields such as `host`, `type`,
`language`, `project`, `command`, `image`, `ports`, `uses`, and `configured`.
The `configured` field lists optional blocks present on the service, such as
`docker`, `healthcheck`, `restart`, `resources`, `logs`, `local`, and `azure`.

The `project` value is the resolved absolute path for the service project, not
the literal string written in azure.yaml.

## JSON output

Use `--output json` for a machine-readable object keyed by service name:

```bash
azd app config --output json
```

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

Unset optional fields are omitted.

## Examples

```bash
azd app config
azd app config api
azd app config --output json
```
