# azd app restart

Restart services.

## Synopsis

```
azd app restart [flags]
```

## Description

Restart one or more services.

This command stops and then starts services. It works on both running and stopped services. Use `--service` to restart a specific service, or `--all` to restart all services.

Services are stopped gracefully before being restarted. If a service doesn't respond to graceful shutdown, it will be forcefully terminated.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | `-s` | string | | Service name(s) to restart (comma-separated) |
| `--all` | | bool | `false` | Restart all services |
| `--yes` | `-y` | bool | `false` | Skip confirmation prompt for `--all` |

## Examples

### Restart a specific service

```bash
azd app restart --service api
```

### Restart multiple services

```bash
azd app restart --service "api,web,worker"
```

### Restart all services

```bash
azd app restart --all
```

### Restart all without confirmation

```bash
azd app restart --all --yes
```

### JSON output

```bash
azd app restart --service api --output json
```

Output:

```json
{
  "serviceName": "api",
  "success": true,
  "message": "Service 'api' restarted",
  "status": "running",
  "duration": "2.345s"
}
```

## Restart Process

For each service, the restart command:

1. Stops the service gracefully (if running)
2. Waits for the process to exit
3. Starts the service with the same configuration

This ensures a clean restart without leftover state from the previous instance.

## Use Cases

- **Code changes**: Restart a service after making code changes (for languages without hot reload)
- **Configuration updates**: Apply new environment variables or configuration
- **Error recovery**: Restart a service that's in an error state
- **Resource refresh**: Clear memory or reset connections

## Exit Codes

| Code | Description |
|------|-------------|
| `0` | All services restarted successfully |
| `1` | One or more services failed to restart |

## Related Commands

- [azd app start](start.md) - Start stopped services
- [azd app stop](stop.md) - Stop running services
- [azd app run](run.md) - Run the development environment
- [azd app health](health.md) - Monitor service health
