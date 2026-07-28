# azd app remove

Remove a service from the services section of azure.yaml.

## Synopsis

```
azd app remove <service> [flags]
```

## Description

The `remove` command is the inverse of `azd app add`. It removes the named service entry from azure.yaml while keeping the remaining services and settings semantically unchanged. yaml formatting may be normalized when the file is written.

## Flags

| Flag | Description |
|------|-------------|
| `-o, --output` | Output format (default, json) |

## Examples

### Remove Redis

```bash
azd app remove redis
```

### Get JSON output

```bash
azd app remove redis --output json
```

```json
{
  "service": "redis",
  "removed": true,
  "message": "Removed redis from azure.yaml"
}
```

If the service is not present, JSON output returns `"removed": false` with a message.

## Behavior

- Removing a service that is not present fails in text mode and lists the current service names.
- In JSON mode, a missing service returns a structured result with `"removed": false`.
- Removing the last service keeps an empty `services: {}` mapping so tooling can still find the key.

## See Also

- [azd app add](add.md) - Add a well-known service to azure.yaml
- [azd app run](run.md) - Start services
