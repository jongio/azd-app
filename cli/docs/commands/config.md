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

## Notes

- Passing an unknown service name returns an error that lists the available services.
- `--output json` emits an object keyed by service name, which pairs well with tools like `jq`.
- Service-level hooks are not part of the azure.yaml schema. Lifecycle hooks are defined at the project level.
