# Eliminate Extra Files Using azd Extension Framework

## Overview

Remove all custom file-based state storage (`.azure/services.json`, `.azure/ports.json`, preference files) and use azd's built-in gRPC services for configuration, with live dashboard API queries for runtime state.

## Problem Statement

Currently `azd app` creates multiple JSON files in `.azure/` for state management:
- `services.json` - Running service registry (stale data issues)
- `ports.json` - Port assignments
- `~/.azure/logs-dashboard/*` - User preferences

This causes:
1. **Stale data** - Cached state becomes inconsistent with reality
2. **File clutter** - Multiple files in project directory
3. **Sync issues** - Multiple processes can have conflicting views
4. **Inconsistent patterns** - Different from how azd stores config

## Solution

**Principle**: `azd app` is an extension that cannot exist without `azd`. All configuration goes through azd's gRPC services. Runtime state lives in the dashboard server (single source of truth).

### Files to Eliminate

| File | Current Purpose | New Approach |
|------|-----------------|--------------|
| `.azure/services.json` | Running service registry | **Live API** - Dashboard server is source of truth |
| `.azure/ports.json` | Port assignments | **UserConfig Service** - `app.projects.<hash>.ports` |
| `~/.azure/logs-dashboard/*` | User preferences | **UserConfig Service** - `app.preferences.*` |
| `.azure/cache/reqs_cache.json` | Requirements cache | **In-memory** during session |

### Files to Keep

| File | Reason |
|------|--------|
| `.azure/logs/*.log` | Logs must persist to disk for history/debugging |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    azd (invokes extension)                       │
│                           │                                      │
│                     gRPC Server                                  │
│         (UserConfig, Environment, Project services)             │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                      gRPC calls
                            │
┌───────────────────────────▼─────────────────────────────────────┐
│                      azd app extension                           │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  azd app run                                               │  │
│  │  ├─ Orchestrator (in-memory service state)                 │  │
│  │  ├─ Dashboard Server (REST API for live state)             │  │
│  │  └─ Stores ports via UserConfig gRPC                       │  │
│  └───────────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  azd app info / stop / logs                                │  │
│  │  ├─ Get dashboard port from UserConfig gRPC                │  │
│  │  └─ Query dashboard REST API for live state                │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Config Structure

```json
// In ~/.azd/config.json (via UserConfig Service)
{
  "app": {
    "projects": {
      "<project-path-hash>": {
        "dashboardPort": 54321,
        "ports": {
          "api": 8080,
          "web": 3000,
          "worker": 5000
        }
      }
    },
    "preferences": {
      "logs": {
        "levelFilter": "info",
        "collapsedServices": ["worker"],
        "classifications": {}
      }
    }
  }
}
```

## Command Flows

### `azd app run`
1. Get/assign ports via `UserConfig.Set("app.projects.<hash>.ports.<service>", port)`
2. Start services (in-memory state in orchestrator)
3. Start dashboard server
4. Store dashboard port via `UserConfig.Set("app.projects.<hash>.dashboardPort", port)`
5. Dashboard serves live state via REST API

### `azd app info`
1. Get dashboard port via `UserConfig.GetString("app.projects.<hash>.dashboardPort")`
2. Query `http://localhost:<port>/api/services`
3. If unreachable → "No services running. Use 'azd app run' to start."

### `azd app stop`
1. Get dashboard port from UserConfig
2. POST `http://localhost:<port>/api/services/stop`
3. If unreachable → "No services running to stop."

### `azd app health`
1. Get dashboard port from UserConfig
2. Query `http://localhost:<port>/api/health`

### `azd app logs`
1. Get dashboard port from UserConfig
2. Connect to `ws://localhost:<port>/api/logs/stream`

## API Additions

Add to dashboard server:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/ping` | GET | Check if dashboard is running (returns 200) |

## Acceptance Criteria

1. No `.azure/services.json` file created
2. No `.azure/ports.json` file created
3. No `~/.azure/logs-dashboard/` directory created
4. Port assignments stored in azd config via gRPC
5. User preferences stored in azd config via gRPC
6. `azd app info` queries live dashboard API
7. `azd app stop` calls dashboard API
8. All existing functionality preserved
9. All tests pass through `azd app` commands
