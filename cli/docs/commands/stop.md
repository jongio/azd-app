# azd app stop

Stop running services and tear down the app.

## Synopsis

```
azd app stop [flags]
```

## Description

Stop running services, kill all associated ports, and tear down the app.

By default (no flags), stops ALL running services and releases their ports. Use `--service` to stop only specific services.

Services are stopped gracefully with a timeout. If a service doesn't respond to graceful shutdown, it will be forcefully terminated.

### Lifecycle Hooks

The stop command supports `prestop` and `poststop` hooks defined in `azure.yaml`:

- **`prestop`** — Runs before services are stopped (e.g., drain connections, flush caches)
- **`poststop`** — Runs after all services are stopped (e.g., cleanup temp files, remove tunnels)

Hook failures are non-fatal: services will still be stopped even if a hook fails.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | `-s` | string | | Service name(s) to stop (comma-separated) |
| `--all` | | bool | `false` | Stop all running services (same as default) |
| `--yes` | `-y` | bool | `false` | Skip confirmation prompt |

## Examples

### Stop all services (default)

```bash
azd app stop
```

### Stop a specific service

```bash
azd app stop --service api
```

### Stop multiple services

```bash
azd app stop --service "api,web,worker"
```

### Stop all without confirmation

```bash
azd app stop --yes
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

### With lifecycle hooks

```yaml
# azure.yaml
hooks:
  prestop:
    run: echo "Draining connections..."
    continueOnError: true
  poststop:
    run: ./scripts/cleanup.sh
    shell: bash
```

## Graceful Shutdown

The stop command uses a graceful shutdown process:

1. Execute `prestop` hook (if configured)
2. Send SIGTERM signal to each service process
3. Wait up to 30 seconds for services to exit cleanly
4. If a service doesn't exit, send SIGKILL to force termination
5. Release all port assignments
6. Execute `poststop` hook (if configured)

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
