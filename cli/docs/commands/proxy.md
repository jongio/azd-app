# azd app proxy

Route local requests to running services through one local origin.

## Synopsis

```bash
azd app proxy [flags]
```

## Description

`azd app proxy` starts one local HTTP listener and path-routes requests to services that are currently running.

Each service is exposed as:

`/<service>/... -> http://localhost:<service-port>/...`

The proxy strips the `/<service>` prefix before forwarding. For example, `/api/users` forwards to `/users` on the `api` service.

Requests to `/` return a list of available routes.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--port` | | int | `8080` | Port for the proxy listener |

## Examples

### Start proxy on the default port

```bash
azd app proxy
```

### Start proxy on a custom port

```bash
azd app proxy --port 9090
```

### Example route table

```text
Proxy listening on http://localhost:8080
/api/    -> http://localhost:5001
/web/    -> http://localhost:3000
```

### Call a proxied endpoint

```bash
curl http://localhost:8080/api/users
```

## Error Behavior

- If no services are running with valid local ports, the command exits with an error.
- If a route exists but the target service is unavailable, the proxy returns `502 Bad Gateway`.

## Related Commands

- [azd app run](run.md) - Start services and populate the local service registry
- [azd app info](info.md) - Show currently running services and ports
