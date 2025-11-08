# Health Monitoring Architecture Diagram

## Command Flow Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                       azd app health                                 │
│                                                                      │
│  Flags:                                                              │
│  --service, --stream, --interval, --output, --endpoint, --timeout   │
└──────────────────────────────────┬──────────────────────────────────┘
                                   │
                                   ↓
┌─────────────────────────────────────────────────────────────────────┐
│                        Health Monitor                                │
│                                                                      │
│  Responsibilities:                                                   │
│  • Coordinate health check execution                                │
│  • Manage streaming mode lifecycle                                  │
│  • Aggregate results and format output                              │
└──────────┬──────────────────────┬───────────────────┬──────────────┘
           │                      │                   │
           ↓                      ↓                   ↓
┌──────────────────┐   ┌──────────────────┐   ┌──────────────────┐
│ Health Checker   │   │ Service          │   │ Output           │
│                  │   │ Discovery        │   │ Formatter        │
│ • HTTP checks    │   │                  │   │                  │
│ • Port checks    │   │ • Load registry  │   │ • Text format    │
│ • Process checks │   │ • Load azure.yml │   │ • JSON format    │
│ • Cascading      │   │ • Apply filters  │   │ • Table format   │
│   strategy       │   │                  │   │                  │
└──────────────────┘   └──────────────────┘   └──────────────────┘
           │                      │                   │
           └──────────────────────┴───────────────────┘
                                   │
                                   ↓
┌─────────────────────────────────────────────────────────────────────┐
│                    Shared Components                                 │
│                                                                      │
│  • Service Registry (.azure/services.json)                          │
│  • Service Info (internal/serviceinfo)                              │
│  • Health Check Functions (internal/service/health.go)              │
│  • Output Utilities (internal/output)                               │
└─────────────────────────────────────────────────────────────────────┘
```

## Health Check Cascading Strategy

```
For each service:

┌─────────────────────────────────────┐
│ 1. Check azure.yaml Config          │
│    healthCheck.endpoint defined?    │
└─────────┬──────────────────┬────────┘
          │ YES              │ NO
          ↓                  ↓
   ┌─────────────┐    ┌──────────────────────┐
   │ Use Explicit│    │ 2. Try HTTP Endpoints│
   │ Endpoint    │    │    /health           │
   └─────────────┘    │    /healthz          │
                      │    /ready            │
                      │    /alive            │
                      │    /ping             │
                      └──────┬───────┬───────┘
                             │ Found │ Not Found
                             ↓       ↓
                      ┌──────────┐  │
                      │ Use HTTP │  │
                      │ Endpoint │  │
                      └──────────┘  ↓
                           ┌──────────────────┐
                           │ 3. Port Available?│
                           └────┬──────┬──────┘
                                │ YES  │ NO
                                ↓      ↓
                         ┌──────────┐ │
                         │ Use Port │ │
                         │ Check    │ │
                         └──────────┘ ↓
                              ┌────────────────┐
                              │ 4. Process     │
                              │    Check       │
                              └────────────────┘
```

## Static vs Streaming Mode

### Static Mode (Default)
```
User runs: azd app health

        ┌──────────────────┐
        │ Perform Health   │
        │ Checks (Parallel)│
        └────────┬─────────┘
                 │
        ┌────────▼─────────┐
        │ Aggregate        │
        │ Results          │
        └────────┬─────────┘
                 │
        ┌────────▼─────────┐
        │ Format Output    │
        │ (Text/JSON/Table)│
        └────────┬─────────┘
                 │
        ┌────────▼─────────┐
        │ Display to       │
        │ Console          │
        └────────┬─────────┘
                 │
        ┌────────▼─────────┐
        │ Exit with Code   │
        │ 0=healthy        │
        │ 1=unhealthy      │
        └──────────────────┘
```

### Streaming Mode (--stream)
```
User runs: azd app health --stream

        ┌──────────────────┐
        │ Initialize Stream│
        │ • Setup signals  │
        │ • Clear terminal │
        └────────┬─────────┘
                 │
        ┌────────▼─────────┐
        │ Monitoring Loop  │
        │ (Every interval) │
        └────────┬─────────┘
                 │
          ┌──────▼──────┐
          │ Perform     │
          │ Checks      │
          └──────┬──────┘
                 │
          ┌──────▼──────┐
          │ Compare with│
          │ Previous    │
          └──────┬──────┘
                 │
          ┌──────▼──────┐
          │ Update      │
          │ Display     │
          └──────┬──────┘
                 │
          ┌──────▼──────┐
          │ Wait for    │
          │ Interval    │
          └──────┬──────┘
                 │
                 └──────┐
                        │ Loop continues until Ctrl+C
                        │
                 ┌──────▼──────┐
                 │ Graceful    │
                 │ Shutdown    │
                 │ • Summary   │
                 │ • Stats     │
                 └─────────────┘
```

## Data Flow - Health Check Execution

```
Input: Service List
   │
   ↓
┌────────────────────────────────────────────┐
│ Parallel Health Checks                     │
│ (Max 10 concurrent)                        │
│                                            │
│  Service A ─┐                              │
│  Service B ─┤──→ Semaphore (rate limit)   │
│  Service C ─┘                              │
└────────────┬───────────────────────────────┘
             │
    ┌────────┼────────┐
    ↓        ↓        ↓
┌────────┐ ┌────────┐ ┌────────┐
│HTTP    │ │Port    │ │Process │
│Check   │ │Check   │ │Check   │
└───┬────┘ └───┬────┘ └───┬────┘
    │          │          │
    └──────────┴──────────┘
               │
    ┌──────────▼─────────┐
    │ HealthCheckResult  │
    │ • status           │
    │ • responseTime     │
    │ • error            │
    │ • details          │
    └──────────┬─────────┘
               │
    ┌──────────▼─────────┐
    │ Aggregate Results  │
    │ • Calculate summary│
    │ • Determine overall│
    └──────────┬─────────┘
               │
    ┌──────────▼─────────┐
    │ HealthReport       │
    │ • timestamp        │
    │ • services[]       │
    │ • summary          │
    └────────────────────┘
```

## Integration with Existing Systems

```
┌─────────────────────────────────────────────────────────────┐
│                     azd app health                           │
└────────────┬───────────────────────────────┬────────────────┘
             │                               │
    ┌────────▼────────┐           ┌─────────▼────────┐
    │ Service Registry│           │ Service Info     │
    │ (Read)          │           │ (Read)           │
    │                 │           │                  │
    │ • Get services  │           │ • Get metadata   │
    │ • Read status   │           │ • Get ports      │
    │ • Read health   │           │ • Get env vars   │
    └────────┬────────┘           └─────────┬────────┘
             │                               │
    ┌────────▼────────────────────────────────▼────────┐
    │           Perform Health Checks                  │
    └────────┬────────────────────────────────┬────────┘
             │                                │
    ┌────────▼────────┐           ┌──────────▼────────┐
    │ Service Registry│           │ Dashboard API     │
    │ (Update)        │           │ (Provide)         │
    │                 │           │                   │
    │ • Update health │           │ GET /api/health   │
    │ • Update status │           │ GET /api/health/  │
    │ • Update time   │           │     stream (SSE)  │
    └─────────────────┘           └───────────────────┘
```

## Component Interaction Sequence

### Static Mode Sequence
```
User          Command       Monitor      Checker     Registry    Output
 │              │             │             │           │          │
 │──health──────>│            │             │           │          │
 │              │──Check()────>│            │           │          │
 │              │             │──Load───────>           │          │
 │              │             │<──Services──│           │          │
 │              │             │──CheckSvc()─>           │          │
 │              │             │<──Result────│           │          │
 │              │             │──Update─────────────────>          │
 │              │<──Report────│             │           │          │
 │              │──Format()───────────────────────────────────────>│
 │<──Output─────────────────────────────────────────────────────────
```

### Streaming Mode Sequence
```
User          Command       Monitor      Checker     Registry    Output
 │              │             │             │           │          │
 │──health──────>│            │             │           │          │
 │  --stream    │──Stream()───>            │           │          │
 │              │             │──Init()─────>           │          │
 │              │             │             │           │          │
 │              │          ╔══▼═══════════════════════════════╗    │
 │              │          ║ Loop (every interval)            ║    │
 │              │          ║  │──Check()────>                 ║    │
 │              │          ║  │──Load───────>                 ║    │
 │              │          ║  │<──Services──│                 ║    │
 │              │          ║  │──CheckSvc()─>                 ║    │
 │              │          ║  │<──Result────│                 ║    │
 │              │          ║  │──Update─────────────────────> ║    │
 │              │          ║  │──Format()─────────────────────────>│
 │<──Update─────────────────────────────────────────────────────────
 │              │          ║  │──Wait(interval)               ║    │
 │              │          ╚══▲═══════════════════════════════╝    │
 │──Ctrl+C──────>             │                                     │
 │              │──Stop()─────>                                     │
 │              │──Summary()───────────────────────────────────────>│
 │<──Final──────────────────────────────────────────────────────────
```

## File Structure

```
cli/
├── docs/
│   ├── commands/
│   │   └── health.md                    # User-facing command docs
│   ├── design/
│   │   ├── health-monitoring.md         # Implementation design
│   │   └── health-monitoring-summary.md # Executive summary
│   └── cli-reference.md                 # Updated with health command
│
└── src/
    ├── cmd/app/commands/
    │   └── health.go                    # Command implementation (future)
    │
    └── internal/
        ├── health/                      # New package (future)
        │   ├── monitor.go              # Health monitor
        │   ├── checker.go              # Health checker
        │   ├── stream.go               # Stream manager
        │   └── formatter.go            # Output formatters
        │
        ├── service/
        │   └── health.go               # Existing health functions
        │
        ├── registry/
        │   └── registry.go             # Service registry (existing)
        │
        └── serviceinfo/
            └── serviceinfo.go          # Service info (existing)
```
