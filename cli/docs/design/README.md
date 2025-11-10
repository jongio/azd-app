# Health Monitoring Command - Documentation Index

This directory contains the complete specification and design for the `azd app health` command.

## Quick Start

**For Users/Product Managers**: Start with [Executive Summary](health-monitoring-summary.md)

**For Developers/Implementers**: Start with [Design Document](health-monitoring.md)

**For Command Reference**: See [Command Documentation](../commands/health.md)

**For Visual Overview**: See [Architecture Diagrams](health-monitoring-architecture.md)

## Document Overview

### 1. Command Documentation
**File**: [`../commands/health.md`](../commands/health.md)  
**Audience**: End users, documentation readers  
**Size**: ~850 lines

**Contents:**
- Command usage and syntax
- All flags and options explained
- Execution flows for both modes
- Health check strategies in detail
- Configuration schema in azure.yaml
- Output format examples (text, JSON, table)
- Error handling and troubleshooting
- 8+ detailed examples
- Common use cases and workflows
- Best practices
- Exit codes

**When to read**: When you want to understand how users will interact with the command.

### 2. Design Document
**File**: [`health-monitoring.md`](health-monitoring.md)  
**Audience**: Developers, technical leads, architects  
**Size**: ~700 lines

**Contents:**
- Architecture and component design
- Core types and interfaces (Go code)
- Data flow diagrams
- Health check implementation details
- Streaming mode implementation
- Integration with existing systems
- Performance considerations
- Security considerations
- Testing strategy
- Future enhancements

**When to read**: When you're implementing the command or reviewing the technical approach.

### 3. Executive Summary
**File**: [`health-monitoring-summary.md`](health-monitoring-summary.md)  
**Audience**: Stakeholders, product managers, reviewers  
**Size**: ~200 lines

**Contents:**
- High-level overview
- Key features summary
- Operational modes
- Configuration highlights
- Use cases
- Implementation status
- Review checklist
- Questions for discussion

**When to read**: For a quick overview or before approving the specification.

### 4. Architecture Diagrams
**File**: [`health-monitoring-architecture.md`](health-monitoring-architecture.md)  
**Audience**: Developers, architects, visual learners  
**Size**: ~320 lines of ASCII diagrams

**Contents:**
- Command flow overview
- Component architecture
- Health check cascading strategy
- Static vs streaming mode flows
- Data flow diagrams
- Component interaction sequences
- Integration points
- Proposed file structure

**When to read**: When you want to understand the system visually.

### 5. CLI Reference (Updated)
**File**: [`../cli-reference.md`](../cli-reference.md)  
**Audience**: CLI users, documentation readers  
**Updated Section**: Added health command entry

**Contents:**
- Command overview table (updated)
- Health command quick reference
- Flags summary
- Health check strategy summary
- Configuration example
- Output format examples
- Exit codes
- Link to full specification

**When to read**: When looking up command syntax or flags quickly.

## Feature Summary

### Two Operational Modes

**Static Mode** (default)
- Point-in-time health snapshot
- Quick status verification
- Exit code indicates health (0=healthy, 1=unhealthy)
- Perfect for CI/CD pipelines

**Streaming Mode** (`--stream`)
- Real-time continuous monitoring
- Configurable check intervals (default: 5s)
- Live terminal updates or JSON stream
- Graceful shutdown with summary

### Health Check Cascading Strategy

```
1. HTTP Health Endpoint ← Preferred
   ├─ Explicit endpoint from azure.yaml
   └─ Common paths: /health, /healthz, /ready, /alive, /ping

2. TCP Port Check ← Fallback
   └─ Verify port is listening

3. Process Check ← Last Resort
   └─ Verify process is running
```

### Output Formats

- **Text**: Human-readable with colors and icons
- **JSON**: Machine-readable for automation
- **Table**: Compact tabular view

### Configuration

Service-level configuration in `azure.yaml`:

```yaml
services:
  api:
    healthCheck:
      type: http
      endpoint: /api/health
      timeout: 5s
      interval: 10s
      headers:
        Authorization: Bearer token
```

## Implementation Status

**Status**: ✅ **IMPLEMENTED**

**Completed:**
- ✅ Complete specification documentation
- ✅ Architecture design
- ✅ Use cases defined
- ✅ Configuration schema
- ✅ Output formats specified
- ✅ Integration points identified
- ✅ Code implementation
- ✅ Unit tests
- ✅ Integration tests
- ✅ Comprehensive test project
- ✅ Cross-platform support (Windows/macOS/Linux)

**In Progress:**
- 🟡 Dashboard integration (future enhancement)
- 🟡 CI/CD pipeline examples (future)

**Implementation Review:**
See [`../dev/health-code-review.md`](../dev/health-code-review.md) for detailed code review and fixes applied.

## Key Design Decisions

### 1. Cascading Health Check Strategy
**Decision**: Try HTTP first, fall back to port, then process  
**Rationale**: HTTP provides rich health information; graceful degradation for all service types

### 2. Streaming vs Static Modes
**Decision**: Separate modes rather than always streaming  
**Rationale**: Different use cases need different behaviors (CI/CD vs development)

### 3. Parallel Health Checks
**Decision**: Check services concurrently (max 10)  
**Rationale**: Faster results; rate limiting prevents overwhelming system

### 4. No Persistent Storage (Initially)
**Decision**: In-memory health history only  
**Rationale**: Simpler initial implementation; can add persistence later

### 5. Integrate with Existing Registry
**Decision**: Use current service registry structure  
**Rationale**: Avoid duplication; leverage existing service tracking

## Use Case Examples

### 1. Daily Development
```bash
# Monitor services while developing
azd app health --stream
```

### 2. CI/CD Pipeline
```bash
# Wait for services to be healthy
azd app run &
sleep 10
azd app health || exit 1
```

### 3. Debugging Issues
```bash
# Check why service is failing
azd app health --verbose --service api
azd app logs --service api --level error
```

### 4. Automation
```bash
# Get unhealthy services for alerting
azd app health --output json | jq '.services[] | select(.status != "healthy") | .serviceName'
```

## Review Checklist

Before implementation begins, verify:

- [ ] **Completeness**: All features are specified
- [ ] **Clarity**: Requirements are clear and unambiguous
- [ ] **Consistency**: Design aligns with existing patterns
- [ ] **Feasibility**: Implementation is technically viable
- [ ] **Performance**: Performance requirements are addressed
- [ ] **Security**: Security implications are considered
- [ ] **Testing**: Testing strategy is defined
- [ ] **Documentation**: User documentation is complete
- [ ] **Integration**: Integration points are identified
- [ ] **Extensibility**: Design supports future enhancements

## Questions for Review

1. **Health Check Strategy**: Is the cascading approach (HTTP → Port → Process) appropriate?
2. **Streaming Interval**: Is 1s-60s the right range for check intervals?
3. **Custom Plugins**: Should we support custom health check plugins in v1?
4. **Health History**: Should history be persistent or in-memory initially?
5. **Rate Limiting**: Do we need additional rate limiting beyond the 10 concurrent limit?
6. **Dependencies**: Should we support health check dependencies (e.g., skip API if DB is down)?
7. **Circuit Breakers**: Do we need circuit breaker patterns for repeatedly failing checks?
8. **Alerting**: Should alerting be in v1 or deferred to v2?

## Related Documentation

### Within This Repository
- [Port Management Design](ports.md) - Related port handling design
- [Run Command](../commands/run.md) - Service orchestration command
- [Info Command](../commands/info.md) - Service information display
- [Logs Command](../commands/logs.md) - Log viewing command

### External References
- [Azure Developer CLI](https://learn.microsoft.com/azure/developer/azure-developer-cli/)
- [Health Check Pattern](https://microservices.io/patterns/observability/health-check-api.html)
- [Kubernetes Health Checks](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [Docker Healthcheck](https://docs.docker.com/engine/reference/builder/#healthcheck)

## Contributing to This Specification

If you have feedback on this specification:

1. Review the relevant document(s)
2. Identify specific concerns or suggestions
3. Open an issue or PR with:
   - Which document(s) are affected
   - Specific section or paragraph
   - Your concern or suggestion
   - Rationale for the change

## Document History

| Date | Change | Author |
|------|--------|--------|
| 2024-11-08 | Initial specification created | GitHub Copilot |
| 2024-11-08 | Added architecture diagrams | GitHub Copilot |
| 2024-11-08 | Added documentation index | GitHub Copilot |
| 2024-11-09 | Implementation completed | GitHub Copilot |
| 2024-11-09 | Updated status and organization | GitHub Copilot |

## License

All documentation in this repository is covered by the repository license. See [LICENSE](../../../LICENSE) for details.
