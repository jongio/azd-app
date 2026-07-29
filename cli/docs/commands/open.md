# azd app open

Open a service URL in the default browser.

## Synopsis

```
azd app open <service> [flags]
```

## Description

The `open` command resolves the URL for a single service and opens it in the default browser. It takes exactly one service name.

Use it to jump straight to a service you are working on without remembering which port it bound to.

## Flags

| Flag | Description |
|------|-------------|
| `--path` | Path to append to the service URL |
| `--print` | Print the URL without opening a browser |

## URL resolution order

The running app state wins over `azure.yaml`, because it knows the port a service actually bound to. Within each source the first match is used:

| Order | Source | Value |
|-------|--------|-------|
| 1 | Running app state | `local.customUrl` |
| 2 | Running app state | The URL reported by the service |
| 3 | Running app state | `http://localhost:<bound port>` |
| 4 | `azure.yaml` | `local.customUrl` |
| 5 | `azure.yaml` | `http://localhost:<first published TCP host port>` |

Only ports actually published on the host are considered. A Docker mapping whose host port is auto-assigned at runtime yields no URL rather than a URL pointing at the container port. Non-TCP mappings are skipped.

## Examples

### Open a service

```bash
azd app open api
```

### Open a specific route

```bash
azd app open api --path /health
```

A trailing slash in `--path` is preserved:

```bash
azd app open api --path /docs/
```

### Print the URL instead of opening a browser

Useful in scripts, in containers, and over SSH where no browser exists:

```bash
azd app open api --print
```

```
http://localhost:3000
```

## Exit behavior

| Condition | Result |
|-----------|--------|
| Service resolves to a URL | Opens the browser (or prints with `--print`) and exits zero |
| Service exists but has no known URL | Fails and suggests starting it with `azd app run`, or setting `local.customUrl` or ports in `azure.yaml` |
| Service is not defined | Fails and lists the available service names |
| No services defined in `azure.yaml` | Fails and reports that no services are defined |
| `azure.yaml` is malformed | Fails with the parse error rather than reporting the service as missing |

## See Also

- [azd app run](run.md) - Start services so their bound ports are known
- [azd app info](info.md) - Show the resolved URL for every service
- [azd app status](status.md) - Check whether the app is running
