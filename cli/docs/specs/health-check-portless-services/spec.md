# Health Check for Portless Services

## Problem Statement

Services defined in `azure.yaml` that don't have explicit `ports` configuration (e.g., TypeScript compilers in watch mode like `api-tsc`, `preload-tsc`, `main-tsc`) are incorrectly getting ports auto-assigned and HTTP health checks attempted.

**Current Behavior:**
- Services without `ports` in `azure.yaml` still return `NeedsPort() = true` by default
- Port manager assigns a random available port
- Health checks attempt HTTP requests to that port, which fails since the service doesn't serve HTTP

**Expected Behavior:**
- Services without explicit `ports` in `azure.yaml` should NOT get ports assigned
- Health checks should use process-based monitoring for portless services
- Services that need ports must explicitly declare them in `azure.yaml`

## Example

Example `azure.yaml` with portless services:

```yaml
services:
  # TypeScript compiler for API (watch mode) - NO ports defined
  api-tsc:
    language: ts
    project: ./services/api-tsc
    environment:
      NODE_ENV: development

  # API server with nodemon - HAS ports defined
  api:
    language: ts
    project: ./services/api
    ports:
      - "3001"
    uses:
      - api-tsc
```

`api-tsc` is a `tsc --watch` process. It:
- Compiles TypeScript files
- Outputs to disk
- Does NOT serve HTTP
- Should NOT have a port assigned
- Should use process-based health checks (is the process running?)

## Functional Requirements

### FR-1: Port Assignment Logic
- Services with explicit `ports` in `azure.yaml` get those ports assigned
- Services without `ports` get NO port assigned (port = 0)
- Remove default behavior of auto-assigning ports to services without explicit port configuration

### FR-2: Health Check Type Selection
For services without ports:
- Default to `process` health check type (verify process is running)
- Support `output` health check type via `healthcheck.type: output` with pattern matching
- Support `healthcheck: false` or `healthcheck.disable: true` to skip all health checks

For services with ports:
- Default to `http` health check type
- Fall back to `port` (TCP) check if HTTP fails
- Fall back to `process` check as last resort

### FR-3: Service Registry Updates
- Services without ports should be registered with `port: 0`
- Health status should be based on process check results
- Dashboard should display portless services without port information

### FR-4: Dashboard Display
- Show portless services in the services list
- Display health status based on process check
- Don't show port column or show "-" for portless services
- Response time shows as "-" for process-based checks

## Acceptance Criteria

1. **No port auto-assignment**: Services without `ports` in `azure.yaml` do not get a port assigned
2. **Process health checks**: Portless services use process-based health monitoring
3. **Existing behavior preserved**: Services with explicit ports continue to work as before
4. **Health report accuracy**: Health reports correctly show portless services as healthy when process is running
5. **Dashboard compatibility**: Dashboard displays portless services correctly without errors
6. **Backward compatible**: Existing `azure.yaml` files with `healthcheck: false` continue to work

## Out of Scope

- Output-based health checks (pattern matching in stdout) - tracked separately
- Changes to dashboard UI components beyond basic compatibility
- Health check documentation updates (already documented correctly)
