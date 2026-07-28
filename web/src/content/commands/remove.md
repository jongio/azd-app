---
title: "azd app remove"
description: "Remove a service from the services section of azure.yaml."
command: "remove"
category: "core"
order: 20
---

# azd app remove

Remove a service from the `services` section of `azure.yaml`.

## Description

The `remove` command is the inverse of `azd app add`. It removes the named service entry from `azure.yaml` while keeping the remaining services and settings semantically unchanged. yaml formatting may be normalized when the file is written.

The file is written atomically, so an interrupted or failed write leaves the original `azure.yaml` intact.

## Usage

```bash
azd app remove <service> [flags]
```

## Flags

| Flag | Description |
|------|-------------|
| `-o, --output` | Output format (default, json) |

## Examples

```bash
# Remove a service
azd app remove redis

# JSON output
azd app remove redis --output json
```

```json
{
  "service": "redis",
  "removed": true,
  "message": "Removed redis from azure.yaml"
}
```

## Behavior

- Removing a service that is not present fails in text mode and lists the current service names.
- In JSON mode, a missing service returns a structured result with `"removed": false` rather than printing nothing.
- Removing the last service keeps an empty `services: {}` mapping so tooling can still find the key.

## See Also

- `azd app add` adds a well-known service to `azure.yaml`
- `azd app run` starts services
