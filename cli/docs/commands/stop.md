# azd app stop

Stop running services and tear down the app.

## Synopsis

```
azd app stop
```

## Description

Sends a shutdown signal to the running `azd app run` process. This triggers graceful shutdown including prestop/poststop hooks, port release, and process cleanup, identical to pressing Ctrl+C in the run terminal.

Run this from **any terminal** in the project directory while `azd app run` is active in another terminal.

### How It Works

1. Discovers the running dashboard port (stored in a per-project temp file)
2. Authenticates with the dashboard using a per-session token
3. Sends an authenticated shutdown request
4. The run process executes the full graceful shutdown path

### Lifecycle Hooks

The stop command supports `prestop` and `poststop` hooks defined in `azure.yaml`:

- **`prestop`**: Runs before services are stopped (e.g., drain connections, flush caches)
- **`poststop`**: Runs after all services are stopped (e.g., cleanup temp files, remove tunnels)

Hook failures are non-fatal: services will still be stopped even if a hook fails.

## Examples

### Stop the running app

```bash
azd app stop
```

### With lifecycle hooks in azure.yaml

```yaml
# azure.yaml
hooks:
  prestop:
    run: echo "Draining connections..."
    continueOnError: true
  poststop:
    run: echo "Cleanup complete"
```

## Graceful Shutdown Sequence

1. Execute `prestop` hook (if configured)
2. Stop all service processes gracefully (10s timeout)
3. Stop the dashboard server
4. Release all port assignments
5. Execute `poststop` hook (if configured)
6. Clean up port discovery file

## Exit Codes

| Code | Description |
|------|-------------|
| `0` | Shutdown signal sent successfully |
| `1` | No running app found or communication failed |

## Related Commands

- [azd app run](run.md) - Run the development environment
- [azd app health](health.md) - Monitor service health
