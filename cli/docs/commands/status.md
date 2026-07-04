# azd app status

Show the current run state for this project.

## Synopsis

```bash
azd app status [--output json]
```

## Description

`azd app status` reads the run-state file for the current project and reports:

1. Whether `azd app run` is active
2. The dashboard URL
3. Running services with URL and port details

If the run-state file exists but the recorded PID is no longer alive, status reports `not running` and removes the stale state file.

## Examples

### Show status in text output

```bash
azd app status
```

### Show status in JSON

```bash
azd app status --output json
```

### Expected JSON shape

```json
{
  "running": true,
  "pid": 12345,
  "dashboardUrl": "http://localhost:4280",
  "startTime": "2026-07-03T12:00:00Z",
  "services": [
    {
      "name": "api",
      "url": "http://localhost:8080",
      "port": 8080
    }
  ]
}
```

When not running:

```json
{
  "running": false
}
```

## Related Commands

- [azd app run](run.md) - Start the development environment
- [azd app stop](stop.md) - Stop the running app
