<!-- NEXT: #task-2-2-realtime-streaming -->
# Azure Cloud Log Streaming Tasks

## Overview

Implementation tasks for streaming Azure-deployed service logs into the azd-app dashboard. Reference [spec.md](spec.md) for full technical specification.

**Key Requirements:**
- Leverage existing `logs` schema section in azure.yaml
- Easy enable with defaults, full customization via KQL queries
- Dashboard mode switching: Local / Azure / All
- MCP server respects dashboard mode with override capability

---

## HOTFIX: Connection & Credential Issues (P0)

*Discovered during integration testing - blocking user experience*

### HF-1: Workspace GUID Auto-Detection
**Assigned**: Developer
**Status**: IN PROGRESS

**Problem**: Log Analytics API requires workspace GUID (customerId), not resource ID. Users must manually set `AZURE_LOG_ANALYTICS_WORKSPACE_GUID` env var.

**Solution**: Auto-detect GUID from workspace ID/name using Azure Resource Manager API.

**Implementation**:
1. ✅ Add `AZURE_LOG_ANALYTICS_WORKSPACE_GUID` env var support in discovery.go
2. ✅ Update bicep to output `logAnalyticsWorkspaceGuid` using `workspace.properties.customerId`
3. ⬜ Auto-detect GUID from resource ID via ARM API when env var missing
4. ⬜ Update documentation with required bicep outputs

**Files**:
- `cli/src/internal/azure/discovery.go` - Add ARM API call to get workspace properties
- `cli/docs/features/azure-logs.md` - Document setup requirements

**Acceptance Criteria**:
- Works without manual GUID env var (auto-detects from workspace ID)
- Clear error message if workspace not found
- Example bicep in documentation

---

### HF-2: Log Analytics Credential Scope
**Assigned**: Developer
**Status**: IN PROGRESS

**Problem**: AZD_ACCESS_TOKEN is ARM-scoped, doesn't work for Log Analytics API (api.loganalytics.io).

**Solution**: Use separate credential chain for Log Analytics that skips AZD token.

**Implementation**:
1. ✅ Create `NewLogAnalyticsCredential()` in credentials.go
2. ✅ Update AzureLogBuffer to use separate credential for Log Analytics
3. ⬜ Verify logs actually appear after credential fix
4. ⬜ Add integration test for credential scoping

**Files**:
- `cli/src/internal/azure/credentials.go` - NewLogAnalyticsCredential()
- `cli/src/internal/service/azure_log_buffer.go` - Use correct credential

**Acceptance Criteria**:
- Log Analytics queries succeed with DefaultAzureCredential
- AZD token still used for ARM queries (resource discovery)
- No auth errors in dashboard

---

### HF-3: Connection Error UX
**Assigned**: Developer
**Status**: IN PROGRESS

**Problem**: Red dot doesn't help users fix issues. Need actionable guidance.

**Solution**: Show amber indicator with tooltip explaining what's missing and how to fix.

**Implementation**:
1. ✅ Update AzureStatus to include ConnectionIssue and ConnectionMessage
2. ✅ Change StatusDot to StatusIndicator with tooltip
3. ⬜ Add setup guide panel when not configured
4. ⬜ Link to documentation in error messages

**Files**:
- `cli/src/internal/dashboard/mode.go` - Add detailed status fields
- `cli/dashboard/src/components/ModeToggle.tsx` - StatusIndicator with tooltip
- `cli/dashboard/src/components/AzureSetupGuide.tsx` - New component

**Acceptance Criteria**:
- Tooltip shows specific issue (missing workspace, auth error, etc.)
- Users can click to see setup instructions
- Links to docs for detailed guidance

---

### HF-4: Verify Log Pipeline
**Assigned**: Developer
**Status**: DONE

**Problem**: Even with correct credentials, logs may not appear if apps aren't sending to Log Analytics.

**Solution**: Verify diagnostic settings and provide guidance.

**Checklist**:
1. ✅ Container App has diagnostic settings pointing to Log Analytics
2. ✅ App Service has diagnostic settings enabled
3. ✅ Log Analytics workspace is receiving data
4. ✅ KQL query matches actual log schema

**Documentation Created**: `cli/docs/features/azure-logs.md` includes:
- Complete bicep examples for diagnostic settings
- Verification commands
- Troubleshooting guide

**Acceptance Criteria**:
- ✅ Diagnostic settings documented in example bicep
- ✅ Verification steps documented
- ✅ Dashboard shows logs when apps are properly configured

---

### HF-5: Documentation Update
**Assigned**: Developer
**Status**: DONE

**Required Documentation**:
1. ✅ Required bicep outputs:
   - `AZURE_LOG_ANALYTICS_WORKSPACE_ID` (resource ID)
   - `AZURE_LOG_ANALYTICS_WORKSPACE_NAME`
   - `AZURE_LOG_ANALYTICS_WORKSPACE_GUID` (customerId)
2. ✅ Example bicep for monitoring module with outputs
3. ✅ Diagnostic settings configuration for each service type
4. ✅ Troubleshooting guide for common issues
5. ✅ Update CLI reference docs
6. ✅ Update /web docs with Azure logs section

**Files Created/Updated**:
- `cli/docs/features/azure-logs.md` - Main feature documentation (DONE)
- `web/src/pages/reference/cli/logs.astro` - CLI reference with --source flag (DONE)
- `web/src/pages/tour/6-logs.astro` - Tour page mentions Azure logs (already present)

**Acceptance Criteria**:
- ✅ Complete setup guide from scratch
- ✅ All required outputs documented
- ✅ Troubleshooting covers credential and workspace issues

---

## Phase 1: Foundation (P0)

### Task 1.1: Azure SDK Integration
**Assigned**: Developer
**Status**: DONE

Add Azure SDK dependencies to Go module:
- `azure-sdk-for-go/sdk/azidentity` for credential handling
- `azure-sdk-for-go/sdk/monitor/query/azlogs` for Log Analytics queries
- `azure-sdk-for-go/sdk/resourcemanager/resources/armresources` for resource discovery

Update `go.mod` and `go.sum` in `cli/` directory.

**Acceptance Criteria**:
- Dependencies added and importable
- No breaking changes to existing functionality
- All existing tests pass

---

### Task 1.2: Schema Extension for Azure Logs
**Assigned**: Developer
**Status**: DONE

Extend azure.yaml schema (`schemas/v1.1/azure.yaml.json`) to add Azure log config under existing `logs` section:

```yaml
logs: { filters: { ... }, analytics: { workspace: "", pollingInterval: 10s, defaultTimespan: 30m, realtime: false } }
```

Also support service-level `logs.analytics` override.

**Acceptance Criteria**:
- Schema validates new `logs.analytics` section
- Service-level config inherits/overrides project-level
- Backward compatible (existing configs work)
- Schema documentation updated

---

### Task 1.3: Azure Credential Provider
**Assigned**: Developer
**Status**: DONE

Create credential provider that chains:
1. AZD extension token (`AZD_ACCESS_TOKEN` from extension context)
2. DefaultAzureCredential fallback (CLI, env vars, managed identity)

Implement in `cli/src/internal/azure/credentials.go`:
- `NewAzdCredential()` function
- Token refresh handling
- Error wrapping with user-friendly messages

**Acceptance Criteria**:
- Works with existing `azd auth login` credentials
- Clear error messages for authentication failures
- Unit tests with mocked credentials

---

### Task 1.4: Azure Resource Discovery
**Assigned**: Developer
**Status**: DONE

Implement Azure resource discovery in `cli/src/internal/azure/discovery.go`:
- Parse `azd env get-values` for `SERVICE_*` patterns
- Query Azure Resource Manager for resource details
- Detect resource type (containerApp, appService, function, etc.)
- Auto-detect Log Analytics workspace from diagnostic settings
- Cache resource metadata

Data structures:
```
AzureResource {
  ServiceName, ResourceID, ResourceType, ResourceGroup,
  SubscriptionID, LogAnalyticsWorkspaceID
}
```

**Acceptance Criteria**:
- Discovers Container Apps and App Service resources
- Returns resource type and ID for each service
- Auto-detects workspace (with manual override support)
- Handles missing/undeployed services gracefully
- Caches results for 5 minutes

---

### Task 1.5: Log Analytics Query Client
**Assigned**: Developer
**Status**: DONE

Implement Log Analytics query client in `cli/src/internal/azure/loganalytics.go`:
- Default KQL queries per resource type
- Custom query support from azure.yaml config
- Query execution with pagination
- Response parsing to `LogEntry` format
- Timestamp tracking for incremental queries

Methods:
- `QueryLogs(resourceType, resourceID, since time.Time) []LogEntry`
- `BuildKQLQuery(resourceType, serviceName, timespan, customQuery) string`
- `GetDefaultQuery(resourceType) string`

**Acceptance Criteria**:
- Default queries work for Container Apps, App Service
- Custom queries from config override defaults
- Transforms results to existing LogEntry format
- Handles query errors and empty results

---

### Task 1.6: Azure Log Buffer with Mode Support
**Assigned**: Developer
**Status**: DONE

Extend logging infrastructure for Azure logs in `cli/src/internal/service/`:
- Create `AzureLogBuffer` similar to existing `LogBuffer`
- Implement polling mechanism (default 10s interval, configurable)
- Add `Source` field to `LogEntry` type
- Support mode switching: local, azure, all
- Merge Azure logs into unified stream when mode=all

Modify:
- `logentry.go` - add Source and AzureMetadata fields
- `logbuffer.go` - support source filtering
- Create `azurelogbuffer.go` for Azure-specific logic
- Create `logmode.go` for mode management

**Acceptance Criteria**:
- Azure logs stored in buffer with same interface
- Polling interval configurable via azure.yaml
- Logs tagged with source=azure
- Mode switching works without reconnect
- Memory limits same as local logs (1000 entries)

---

### Task 1.7: Mode API Endpoints
**Assigned**: Developer
**Status**: DONE

Add mode management API in `cli/src/internal/server/`:
- `GET /api/mode` - get current mode and available modes
- `PUT /api/mode` - set dashboard mode
- Store mode in server state (shared with MCP)

Extend existing endpoints with source parameter:
- `/api/logs/stream?source=all|local|azure`
- `/api/logs?source=all|local|azure`
- `/api/services?source=all|local|azure`

**Acceptance Criteria**:
- Mode API documented and tested
- Source parameter filters log output
- Mode persists for session lifetime
- Broadcasts mode change to WebSocket clients

---

### Task 1.8: Azure Log API Endpoints
**Assigned**: Developer
**Status**: DONE

Add Azure-specific API endpoints in `cli/src/internal/server/`:
- `GET /api/azure/services` - list Azure-deployed services
- `GET /api/azure/logs?service={name}` - fetch Azure logs
- `WS /api/azure/logs/stream?service={name}` - stream Azure logs
- `POST /api/azure/logs/query` - execute custom KQL
- `GET /api/azure/status` - Azure connection status

**Acceptance Criteria**:
- New endpoints registered and documented
- Azure status returns connection state, subscription, workspace
- Error responses include actionable guidance
- Query endpoint validates KQL safety

---

### Task 1.9: Dashboard Mode Toggle UI
**Assigned**: Designer → Developer
**Status**: DONE

Add mode toggle to dashboard header:
- Button group: `[Local] [Azure] [All]`
- Position: ConsoleView toolbar
- Keyboard shortcut: `Ctrl+Shift+M` to cycle
- Persist preference in session storage
- Read default from azure.yaml via API

Visual design:
- Local: blue highlight
- Azure: purple highlight
- All: green highlight
- Clear active state indicator

Files to modify:
- `cli/dashboard/src/components/ConsoleView.tsx`
- `cli/dashboard/src/components/ModeToggle.tsx` (new)
- `cli/dashboard/src/hooks/useMode.ts` (new)
- `cli/dashboard/src/contexts/ModeContext.tsx` (new)

**Acceptance Criteria**:
- Toggle visible and functional
- Mode changes reflected immediately
- Keyboard shortcut works
- Default from config respected on load

---

### Task 1.10: Azure Connection Status UI
**Assigned**: Designer → Developer
**Status**: DONE

Add Azure connection status indicator:
- Icon in header: ● green (connected), ○ yellow (connecting), ✕ red (error)
- Tooltip with: subscription, resource group, workspace, last sync
- Click to retry on error
- Show auth guidance on auth failure

**Acceptance Criteria**:
- Status visible in dashboard header
- Updates reflect actual connection state
- Clear call-to-action for errors
- Non-blocking - local logs still work

---

### Task 1.11: Log Entry Source Badges
**Assigned**: Developer
**Status**: DONE

Update log display to show source badges:
- `[Azure]` badge on Azure-sourced logs
- `[Local]` badge only in All mode
- Different background tint per source
- Click badge to filter by source

Files to modify:
- `cli/dashboard/src/components/LogsPane.tsx`
- `cli/dashboard/src/components/LogsView.tsx`
- `cli/dashboard/src/types.ts`

**Acceptance Criteria**:
- Badges render correctly
- Visual distinction clear
- Click filtering works
- Performance acceptable with mixed sources

---

### Task 1.12: MCP Tool Extensions
**Assigned**: Developer
**Status**: DONE (partial - dedicated Azure tools deferred to P2)

Extend MCP `get_service_logs` tool with source parameter:
- Add `source: auto|local|azure` parameter
- Implement source resolution logic:
  1. Parameter override
  2. Dashboard mode (if running)
  3. azure.yaml config
  4. Auto-detect

~~Add new MCP tools:~~
~~- `get_azure_logs` - dedicated Azure query tool~~
~~- `get_azure_status` - Azure connection and resource info~~
*Note: Dedicated Azure MCP tools deferred to Phase 3 (3.3 KQL Query Builder). Core functionality available via `get_service_logs --source azure`.*

Modify:
- `cli/src/cmd/app/commands/mcp_tools.go`
- Add mode synchronization with dashboard API

**Acceptance Criteria**:
- Existing tool backward compatible
- Source parameter works correctly
- ~~New tools documented with schemas~~
- Mode sync works when dashboard running

---

### Task 1.13: Container Apps + App Service Tests
**Assigned**: Tester
**Status**: DONE

Write integration tests for core Azure log functionality:
- Mock Azure API responses
- Test log parsing and transformation
- Test mode switching logic
- Test error handling (auth failure, resource not found)
- Test config parsing from azure.yaml

**Acceptance Criteria**:
- 80% code coverage on new Azure code
- Tests run in CI without Azure credentials
- Edge cases covered (empty logs, rate limits, custom queries)

---

## Phase 2: Service Expansion (P1)

### Task 2.1: Azure Functions Log Support
**Assigned**: Developer
**Status**: DONE

Add Azure Functions log streaming:
- Detect Function App resources
- Query FunctionAppLogs and traces tables
- Default query with function name context
- Support custom queries via config

**Acceptance Criteria**:
- Function invocation logs visible
- Function name shown in log context
- Error/exception logs highlighted

---

### Task 2.2: Real-time Streaming Mode
**Assigned**: Developer
**Status**: TODO

Implement low-latency streaming using service-specific APIs:
- Container Apps: Real-time log stream API
- App Service: Kudu logstream endpoint
- Toggle between polling and real-time modes
- Configurable via `logs.analytics.realtime: true`
- Automatic fallback on API failures

**Acceptance Criteria**:
- Logs appear within 5 seconds of generation
- Manual toggle between modes in UI
- Graceful degradation to polling
- Config option documented

---

### Task 2.3: Historical Log Query Panel
**Assigned**: Designer → Developer
**Status**: TODO

Add historical log query UI:
- Time range picker (last 15m, 30m, 6h, 24h, custom)
- KQL query input (collapsible advanced section)
- Load more pagination
- Export to file (JSON/text)

**Acceptance Criteria**:
- Time range selection functional
- Custom KQL executes correctly
- Export produces valid output
- Loading state clearly indicated

---

### Task 2.4: Error State UI Enhancement
**Assigned**: Designer → Developer
**Status**: TODO

Improve error handling UI for Azure logs:
- Permission denied: show required roles
- Resource not found: suggest `azd provision`
- Network errors: retry button with backoff indicator
- Rate limiting: cooldown timer display

**Acceptance Criteria**:
- Each error type has specific UI treatment
- Actionable guidance provided
- Non-blocking to other dashboard functions

---

## Phase 3: Advanced Features (P2)

### Task 3.1: AKS Container Insights Integration
**Assigned**: Developer
**Status**: TODO

Add AKS log support via Container Insights:
- Query ContainerLogV2 table
- Map pod/container names to services
- Default query with namespace filtering
- Support custom queries

**Acceptance Criteria**:
- AKS container logs visible
- Pod and container context shown
- Works with Container Insights enabled clusters

---

### Task 3.2: Azure Container Instances Support
**Assigned**: Developer
**Status**: TODO

Add ACI log streaming:
- Container logs API integration
- Real-time attach mode
- Handle container group structure

**Acceptance Criteria**:
- ACI logs visible in dashboard
- Multiple containers in group supported

---

### Task 3.3: KQL Query Builder UI
**Assigned**: Designer → Developer
**Status**: TODO

Advanced query interface for power users:
- KQL input with syntax highlighting
- Query templates dropdown per resource type
- Query history (session storage)
- Results in sortable table view

**Acceptance Criteria**:
- KQL queries execute successfully
- Templates cover common use cases
- Results display in readable format

---

### Task 3.4: Cross-service Log Correlation
**Assigned**: Developer
**Status**: TODO

Enable viewing logs across services in context:
- Timestamp-based alignment
- Request ID correlation (if available in logs)
- Highlight related entries on hover

**Acceptance Criteria**:
- Multiple services viewable simultaneously
- Timestamps aligned across sources
- Visual grouping of related logs

---

## Phase 4: Polish (P3)

### Task 4.1: Performance Optimization
**Assigned**: Developer
**Status**: TODO

Optimize Azure log performance:
- Connection pooling for Azure APIs
- Query result caching with TTL
- Incremental fetch optimization
- Memory usage profiling and limits

**Acceptance Criteria**:
- Dashboard responsive with Azure logs
- Memory within existing limits
- API calls minimized through caching

---

### Task 4.2: Log Filtering Enhancement
**Assigned**: Developer
**Status**: TODO

Advanced filtering for Azure logs:
- Filter by resource instance/replica
- Filter by log category
- Saved filter presets
- Regex search support

**Acceptance Criteria**:
- All filters work with Azure logs
- Filters combinable
- Performance acceptable with filters

---

### Task 4.3: Documentation
**Assigned**: Developer
**Status**: TODO

Write user and developer documentation:
- User guide for Azure log feature
- azure.yaml configuration reference
- Required Azure permissions guide
- Troubleshooting guide
- MCP tool documentation updates

**Acceptance Criteria**:
- Docs in cli/docs/features/azure-logs.md
- Schema reference updated
- Common issues documented

---

### Task 4.4: E2E Testing
**Assigned**: Tester
**Status**: TODO

End-to-end tests for Azure log feature:
- Dashboard tests with mocked Azure backend
- Mode toggle behavior tests
- MCP tool integration tests
- Accessibility tests for new UI

**Acceptance Criteria**:
- E2E tests cover happy path
- Error states tested
- Tests run in CI

---

## Task Dependencies

```
1.1 (SDK) ─┬─► 1.2 (Schema) ─► 1.3 (Credentials) ─► 1.4 (Discovery)
           │                                              │
           │                                              ▼
           │                                        1.5 (Query Client)
           │                                              │
           │                                              ▼
           │                                        1.6 (Log Buffer)
           │                                              │
           ├───────────────────────────────────────────────┤
           │                                              │
           ▼                                              ▼
     1.7 (Mode API) ◄──────────────────────────► 1.8 (Azure API)
           │                                              │
           └──────────────┬───────────────────────────────┘
                          │
           ┌──────────────┼──────────────┐
           ▼              ▼              ▼
     1.9 (Toggle)   1.10 (Status)  1.11 (Badges)
           │              │              │
           └──────────────┼──────────────┘
                          ▼
                   1.12 (MCP Tools)
                          │
                          ▼
                   1.13 (Tests)
                          │
                          ▼
           ┌──────────────┼──────────────┐
           ▼              ▼              ▼
     2.1 (Functions) 2.2 (Realtime) 2.3 (Historical)
           │              │              │
           └──────────────┼──────────────┘
                          ▼
                   2.4 (Error UI)
                          │
           ┌──────────────┼──────────────┐
           ▼              ▼              ▼
     3.1 (AKS)      3.2 (ACI)     3.3 (KQL UI)
           │              │              │
           └──────────────┼──────────────┘
                          ▼
                   3.4 (Correlation)
                          │
           ┌──────────────┼──────────────┐
           ▼              ▼              ▼
     4.1 (Perf)    4.2 (Filters)  4.3 (Docs)
           │              │              │
           └──────────────┼──────────────┘
                          ▼
                    4.4 (E2E)
```

---

## Milestone Summary

| Phase | Tasks | Focus |
|-------|-------|-------|
| Phase 1 (P0) | 1.1 - 1.13 | Foundation: Schema, Mode switching, Container Apps + App Service, MCP |
| Phase 2 (P1) | 2.1 - 2.4 | Functions, Real-time streaming, Historical queries, Error UX |
| Phase 3 (P2) | 3.1 - 3.4 | AKS, ACI, KQL builder, Cross-service correlation |
| Phase 4 (P3) | 4.1 - 4.4 | Performance, Advanced filters, Documentation, E2E tests |
