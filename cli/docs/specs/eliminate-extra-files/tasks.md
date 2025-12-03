# Tasks: Eliminate Extra Files

## Progress: 8/8 tasks complete ✅

---

## Task 1: Create azdconfig package
**Agent**: Developer
**Status**: ✅ DONE

Create `internal/azdconfig` package that wraps azd's gRPC UserConfig service for storing/retrieving configuration.

**Deliverables**:
- `internal/azdconfig/config.go` - Config client wrapper ✅
- Functions: `GetDashboardPort`, `SetDashboardPort`, `GetServicePort`, `SetServicePort`, `GetPreference`, `SetPreference` ✅
- Project hash function for unique project identification ✅

---

## Task 2: Migrate portmanager to use azdconfig
**Agent**: Developer  
**Status**: ✅ DONE
**Depends on**: Task 1

Replace file-based port storage with azdconfig gRPC calls.

**Deliverables**:
- Remove `ports.json` file I/O from `internal/portmanager/portmanager.go` ✅
- Use `azdconfig.SetServicePort()` and `azdconfig.GetServicePort()` ✅
- Update tests to work through azd extension framework ✅
- Delete file-based persistence code ✅

---

## Task 3: Migrate logs_config to use azdconfig
**Agent**: Developer
**Status**: ✅ DONE
**Depends on**: Task 1

Replace file-based preference storage with azdconfig gRPC calls.

**Deliverables**:
- Update `internal/dashboard/logs_config.go` to use azdconfig ✅
- Remove `~/.azure/logs-dashboard/` directory creation ✅
- Store preferences at `app.preferences.logs.*` ✅
- Update tests ✅

---

## Task 4: Add /api/ping endpoint to dashboard
**Agent**: Developer
**Status**: ✅ DONE

Add a simple ping endpoint for checking if dashboard is running.

**Deliverables**:
- Add `GET /api/ping` endpoint to `internal/dashboard/server.go` ✅
- Returns `{"status": "ok"}` with 200 status ✅
- Add test ✅

---

## Task 5: Update azd app info to query dashboard API
**Agent**: Developer
**Status**: ✅ DONE
**Depends on**: Task 1, Task 4

Modify info command to query live dashboard instead of reading files.

**Deliverables**:
- Update `cmd/app/commands/info.go` ✅
- Get dashboard port from azdconfig ✅
- Query `http://localhost:<port>/api/services` ✅
- Show "No services running" if dashboard unreachable ✅
- Remove registry file reading ✅
- Update tests ✅

---

## Task 6: Update azd app stop to call dashboard API
**Agent**: Developer
**Status**: ✅ DONE
**Depends on**: Task 1

Modify stop command to call dashboard API instead of reading registry.

**Deliverables**:
- Update `cmd/app/commands/stop.go` ✅
- Get dashboard port from azdconfig ✅
- POST to `http://localhost:<port>/api/services/stop` ✅
- Show "No services running" if dashboard unreachable ✅
- Update tests ✅

---

## Task 7: Remove registry file persistence
**Agent**: Developer
**Status**: ✅ DONE
**Depends on**: Task 5, Task 6

Convert registry to in-memory only (no services.json).

**Deliverables**:
- Update registry to be in-memory only ✅
- Update orchestrator to use in-memory state only ✅
- Update dashboard to serve state from orchestrator ✅
- Update healthcheck monitor (registry now in-memory only) ✅
- Update all affected tests ✅

**Notes**: Instead of deleting the registry package, we converted it to be in-memory only.
This achieves the goal of eliminating services.json without a major rewrite of the codebase.

---

## Task 8: Final cleanup and testing
**Agent**: Tester
**Status**: ✅ DONE
**Depends on**: Task 7

Verify all functionality works and no extra files are created.

**Deliverables**:
- Run full test suite ✅ (all 26 packages pass)
- Verify no `.azure/services.json` created ✅ (registry is now in-memory only)
- Verify no `.azure/ports.json` created ✅ (portmanager uses azdconfig)
- Verify no `~/.azure/logs-dashboard/` created ✅ (logs_config uses azdconfig)
- Verify `.azure/logs/*.log` still works ✅ (unchanged, not affected)
- Update documentation ✅ (see summary below)

---

## Implementation Summary

The following changes were made to eliminate extra files:

### New Files Created
1. **`cli/src/internal/azdconfig/config.go`** - Wrapper for azd's gRPC UserConfig service
   - Provides `ConfigClient` interface with in-memory fallback for testing
   - Handles project hash generation for config keys
   - Methods: `Get/Set/Clear DashboardPort`, `Get/Set/Clear ServicePort`, `GetAllServicePorts`, `Get/Set Preference`

2. **`cli/src/internal/dashboard/client.go`** - Dashboard API client helper
   - Methods: `Ping`, `GetServices`, `StopService`, `StopAllServices`, `IsDashboardRunning`, `GetDashboardPort`

### Files Modified
1. **`cli/src/internal/portmanager/portmanager.go`**
   - Uses `azdconfig` for port storage instead of `ports.json`
   - Falls back to in-memory storage when gRPC unavailable (for tests)
   - Removed all file I/O code

2. **`cli/src/internal/dashboard/logs_config.go`**
   - Uses `azdconfig` for preferences instead of file storage
   - Removed `getUserConfigDir()` and file-based load/save

3. **`cli/src/internal/dashboard/server.go`**
   - Added `/api/ping` endpoint
   - Added `registerPortInConfig()` and `clearPortFromConfig()` helpers
   - Start/Stop register/clear dashboard port in azdconfig

4. **`cli/src/cmd/app/commands/info.go`**
   - Queries dashboard API instead of reading registry files
   - Shows "No services running" if dashboard unreachable

5. **`cli/src/cmd/app/commands/stop.go`**
   - Calls dashboard API instead of reading registry
   - Shows "No services running" if dashboard unreachable

6. **`cli/src/internal/registry/registry.go`**
   - Converted to in-memory only (no file persistence)
   - Removed `filePath` field, `load()`, `save()` functions
   - Maintains API compatibility for orchestrator

### Test Updates
- Updated tests that expected `ports.json` or `services.json` files
- Tests now use `azdconfig.NewInMemoryClient()` for isolation
- Portmanager automatically falls back to in-memory storage during tests

### Architecture Summary
```
Before:
  azd app → registry → .azure/services.json
  azd app → portmanager → .azure/ports.json
  dashboard → logs_config → ~/.azure/logs-dashboard/preferences.json

After:
  azd app → dashboard API → live in-memory state
  dashboard → azdconfig → azd UserConfig gRPC → ~/.azd/config.yaml
  portmanager → azdconfig → azd UserConfig gRPC → ~/.azd/config.yaml
  logs_config → azdconfig → azd UserConfig gRPC → ~/.azd/config.yaml
```
