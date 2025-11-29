# azd app stop

Stop running services.

## Synopsis

```
azd app stop [flags]
```

## Description

Stop one or more running services gracefully.

This command stops services that are currently running. Use `--service` to stop a specific service, or `--all` to stop all running services.

Services are stopped gracefully with a timeout. If a service doesn't respond to graceful shutdown, it will be forcefully terminated.

## Options

| Flag | Alias | Description |
|------|-------|-------------|
| `--service` | `-s` | Service name(s) to stop (comma-separated) |
| `--all` | | Stop all running services |
| `--yes` | `-y` | Skip confirmation prompt for `--all` |
| `--output` | `-o` | Output format: `default`, `json` |

## Examples

### Stop a specific service

```bash
azd app stop --service api
```

### Stop multiple services

```bash
azd app stop --service "api,web,worker"
```

### Stop all running services

```bash
azd app stop --all
```

### Stop all without confirmation

```bash
azd app stop --all --yes
```

### JSON output

```bash
azd app stop --service api --output json
```

Output:

```json
{
  "serviceName": "api",
  "success": true,
  "message": "Service 'api' stopped",
  "status": "stopped",
  "duration": "0.856s"
}
```

## Graceful Shutdown

The stop command uses a graceful shutdown process:

1. Send SIGTERM signal to the service process
2. Wait up to 30 seconds for the service to exit cleanly
3. If the service doesn't exit, send SIGKILL to force termination

This allows services to complete in-flight requests and clean up resources before stopping.

## Exit Codes

| Code | Description |
|------|-------------|
| `0` | All services stopped successfully |
| `1` | One or more services failed to stop |

## Related Commands

- [azd app start](start.md) - Start stopped services
- [azd app restart](restart.md) - Restart services
- [azd app run](run.md) - Run the development environment
- [azd app health](health.md) - Monitor service health
