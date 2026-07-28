---
title: Ports
description: List host port bindings and detect conflicts before running services.
command: azd app ports
category: info
order: 60
---

Use `azd app ports` to inspect the host ports configured in `azure.yaml` for each service.

The command shows each binding as `host -> container/protocol`. Bind IPs are included when configured, for example `127.0.0.1:8080 -> 80/tcp`. Ports that are left for the runtime to assign are shown as `auto`.

When explicit host bindings overlap, the command marks the conflicting bindings and exits non-zero. Use it as a preflight check before `azd app run` or in CI.

```bash
azd app ports
azd app ports --output json
```
