# azd app start

Start stopped services.

## Synopsis

```
azd app start [flags]
```

## Description

Start one or more stopped services that were previously running.

This command starts services that are currently in a stopped or error state. Use `--service` to start a specific service, or `--all` to start all stopped services.

The start command operates on the service registry maintained by `azd app run`. If no services are registered, use `azd app run` to start your development environment first.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | `-s` | string | | Service name(s) to start (comma-separated) |
| `--all` | | bool | `false` | Start all stopped services |

## Examples

### Start a specific service

```bash
azd app start --service api
```

### Start multiple services

```bash
azd app start --service "api,web,worker"
```

### Start all stopped services

```bash
azd app start --all
```

### JSON output

```bash
azd app start --service api --output json
```

Output:

```json
{
  "serviceName": "api",
  "success": true,
  "message": "Service 'api' started",
  "status": "running",
  "duration": "1.234s"
}
```

## Exit Codes

| Code | Description |
|------|-------------|
| `0` | All services started successfully |
| `1` | One or more services failed to start |

## Related Commands

- [azd app stop](stop.md) - Stop running services
- [azd app restart](restart.md) - Restart services
- [azd app run](run.md) - Run the development environment
- [azd app health](health.md) - Monitor service health
