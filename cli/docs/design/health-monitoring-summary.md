# Health Monitoring Command - Specification Summary

## Purpose

This document provides a high-level summary of the `azd app health` command specification. For complete details, see:
- [Command Documentation](../commands/health.md) - User-facing command documentation
- [Design Document](health-monitoring.md) - Implementation architecture and design

## Overview

The `azd app health` command provides comprehensive health monitoring for local development services with two operational modes:

1. **Static Mode** (default): Point-in-time health snapshot
2. **Streaming Mode** (`--stream`): Real-time continuous monitoring

## Key Features

### Intelligent Health Detection

The command uses a cascading strategy to determine the best health check method:

```
1. HTTP Health Endpoint (Preferred)
   ├─ Explicit endpoint from azure.yaml
   ├─ Common paths: /health, /healthz, /ready, /alive, /ping
   └─ Accept 2xx/3xx status codes

2. TCP Port Check (Fallback)
   └─ Verify port is listening

3. Process Check (Last Resort)
   └─ Verify process is running
```

### Two Operational Modes

#### Static Mode
- Single point-in-time health check
- Quick status verification
- Suitable for CI/CD pipelines
- Exit codes: 0 (healthy), 1 (unhealthy), 2 (error)

#### Streaming Mode
- Continuous health monitoring
- Configurable check intervals (default: 5s)
- Real-time status updates
- Interactive terminal display or JSON stream
- Ctrl+C for graceful shutdown

### Multiple Output Formats

- **Text**: Human-readable with colors and icons (default)
- **JSON**: Machine-readable for automation
- **Table**: Compact tabular view

## Command Usage

```bash
# Basic health check
azd app health

# Specific services
azd app health --service web,api

# Real-time monitoring
azd app health --stream

# Custom interval
azd app health --stream --interval 3s

# JSON output for automation
azd app health --output json

# Verbose details
azd app health --verbose
```

## Configuration

Services can specify health check configuration in `azure.yaml`:

```yaml
services:
  api:
    language: python
    project: ./api
    ports:
      - "8080"
    healthCheck:
      type: http              # http, port, process
      endpoint: /api/health   # HTTP endpoint path
      timeout: 5s             # Timeout for each check
      interval: 10s           # Interval for streaming mode
      startPeriod: 30s        # Grace period after startup
      retries: 3              # Retries before marking unhealthy
      headers:                # Optional HTTP headers
        Authorization: Bearer token
```

## Health Status Values

| Status | Meaning | Icon |
|--------|---------|------|
| `healthy` | Service fully operational | ✓ |
| `degraded` | Service running with issues | ⚠ |
| `unhealthy` | Service not functioning | ✗ |
| `starting` | Service initializing | ○ |
| `unknown` | Cannot determine health | ? |

## Integration

### Service Registry
- Reads service list from `.azure/services.json`
- Updates health status in registry
- Cleans stale entries

### Dashboard
- Provides `/api/health` endpoint
- Server-Sent Events for real-time updates
- Displays health in dashboard UI

### Existing Commands
- `azd app run`: Performs health checks during service startup
- `azd app info`: Shows last known health status
- `azd app logs`: Useful for debugging unhealthy services

## Use Cases

### Development
```bash
# Monitor services during development
azd app health --stream
```

### CI/CD
```bash
# Wait for services to be healthy
azd app run &
sleep 10
azd app health || exit 1
```

### Debugging
```bash
# Check why service is failing
azd app health --verbose --service api
azd app logs --service api --level error
```

### Automation
```bash
# Get health as JSON for processing
azd app health --output json | jq '.summary.unhealthy'
```

## Implementation Status

**Current Status**: ⚠️ SPECIFICATION ONLY

This is a design specification document. Implementation will come after:
1. PR27 is merged (as mentioned in the requirements)
2. This specification is reviewed and approved
3. Implementation work begins

## Dependencies

### Existing Infrastructure
- Health check functions in `cli/src/internal/service/health.go`
- Service registry in `cli/src/internal/registry/`
- Service info package in `cli/src/internal/serviceinfo/`
- Output formatting in `cli/src/internal/output/`

### New Components Required
- Health monitor coordinator
- Health checker with cascading strategy
- Streaming manager
- Output formatters (text, JSON, table)
- Command implementation in `cli/src/cmd/app/commands/health.go`

## Future Enhancements

After initial implementation, potential enhancements include:

1. **Health History**: Store and query historical health data
2. **Alerting**: Email/webhook notifications on health changes
3. **Metrics Export**: Prometheus format export
4. **Predictive Analysis**: ML-based failure prediction
5. **Distributed Tracing**: Correlate health with traces
6. **Custom Checks**: Plugin system for custom health logic

## Related Documents

- [Command Documentation](../commands/health.md) - Complete command specification
- [Design Document](health-monitoring.md) - Implementation architecture
- [CLI Reference](../cli-reference.md) - Command reference with health command
- [Run Command](../commands/run.md) - Related service orchestration
- [Info Command](../commands/info.md) - Service information display

## Review Checklist

Before implementation, verify:

- [ ] Specification is complete and clear
- [ ] All use cases are covered
- [ ] Configuration schema is well-defined
- [ ] Output formats meet requirements
- [ ] Integration points are identified
- [ ] Error handling is comprehensive
- [ ] Performance considerations addressed
- [ ] Security implications reviewed
- [ ] Testing strategy defined
- [ ] Documentation is thorough

## Questions for Review

1. Is the cascading health check strategy appropriate?
2. Should we support custom health check plugins from the start?
3. Is the streaming interval range (1s-60s) reasonable?
4. Do we need rate limiting for health checks?
5. Should health history be stored persistently or in-memory only?
6. Is the configuration schema flexible enough for future needs?
7. Should we support health check dependencies (e.g., don't check API if DB is down)?
8. Do we need circuit breaker patterns for failing health checks?

## Approval

This specification requires review and approval before implementation begins.

**Reviewers**:
- [ ] Product Owner
- [ ] Technical Lead
- [ ] DevOps Team
- [ ] Documentation Team

**Approved**: ___________  **Date**: ___________
