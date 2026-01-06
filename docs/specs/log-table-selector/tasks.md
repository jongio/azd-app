<!-- NEXT: -->
# Log Table Selector Tasks

## Done

All implementation tasks complete. Feature is fully functional and accessible via the Settings2 (gear) icon in the HistoricalLogPanel header.

### Backend - Table Discovery API {#backend-table-discovery}
**Assigned**: Developer
**File**: `cli/src/internal/azure/loganalytics.go`

Added ListAvailableTables method to LogAnalyticsClient:
- Query Log Analytics workspace for available tables using schema query
- Return TableInfo structs with name, category, description, columns
- Add recommended tables per resource type
- Handle workspace not configured error gracefully

### Backend - Table Categories {#backend-table-categories}
**Assigned**: Developer  
**File**: `cli/src/internal/azure/tables.go` (new)

Created predefined table categories:
- Define TableCategories map for containerapp, appservice, function, aks, aci
- Add GetRecommendedTables(resourceType) function
- Add GetTableCategory(tableName) function
- Include table descriptions for UI display

### Backend - API Endpoints {#backend-api-endpoints}
**Assigned**: Developer
**File**: `cli/src/internal/dashboard/azure_logs.go`

Added new API endpoints:
- GET /api/azure/tables - List available tables (calls ListAvailableTables)
- GET /api/azure/logs/config?service=X - Get service log config
- PUT /api/azure/logs/config - Save service log config
- Add request/response types for config endpoints

### Backend - Query Builder {#backend-query-builder}
**Assigned**: Developer
**File**: `cli/src/internal/azure/query_builder.go` (new)

Created query builder for table selections:
- BuildQueryFromTables(tables, serviceName, timespan) function
- Handle single table vs union of multiple tables
- Apply service name filter using appropriate column per table
- Support {serviceName} and {timespan} placeholders

### Backend - Config Types {#backend-config-types}
**Assigned**: Developer
**File**: `cli/src/internal/service/azure_logs_config.go`

Added new types to AzureLogsConfig:
- Mode field: "tables" | "custom"
- Tables field: []string for selected table names
- Updated yaml tags for serialization

### Schema Update {#schema-update}
**Assigned**: Developer
**File**: `schemas/v1.1/azure.yaml.json`

Updated JSON schema:
- Add mode enum to azureLogsConfig
- Add tables array property
- Updated examples with table selection

### Frontend - TableSelector Component {#frontend-table-selector}
**Assigned**: Developer
**File**: `cli/dashboard/src/components/TableSelector.tsx` (new)

Created TableSelector component:
- Multi-select checkbox list grouped by category
- Search/filter input for table names
- Category headers (Container Apps, App Service, etc.)
- Recommended tables badge/highlight
- Select All / Clear All actions
- Column preview on hover

### Frontend - LogConfigPanel Component {#frontend-log-config-panel}
**Assigned**: Developer
**File**: `cli/dashboard/src/components/LogConfigPanel.tsx` (new)

Created LogConfigPanel modal:
- Tabs/toggle for Tables vs Custom KQL mode
- TableSelector for tables mode
- KQL textarea (reuse KqlQueryInput) for custom mode
- Save/Cancel actions
- Loading state during save

### Frontend - API Hooks {#frontend-api-hooks}
**Assigned**: Developer
**File**: `cli/dashboard/src/hooks/useLogConfig.ts` (new)

Created hooks for log configuration:
- useAvailableTables() - fetch /api/azure/tables
- useLogConfig(serviceName) - fetch /api/azure/logs/config
- useSaveLogConfig() - PUT /api/azure/logs/config mutation
- Handle loading/error states

### Frontend - Integration {#frontend-integration}
**Assigned**: Developer
**File**: `cli/dashboard/src/components/HistoricalLogPanel.tsx`

Integrated config panel:
- Add "Configure" button/icon to panel header
- Open LogConfigPanel on click
- Refresh logs after config save
- Show current mode indicator (Tables/Custom)

---

## TODO: Documentation {#documentation}
**Assigned**: Developer
**File**: `cli/docs/features/azure-logs.md`

Update documentation:
- Add table selection feature description
- Document azure.yaml configuration options
- Add screenshots of UI
- Include examples for different resource types

## TODO: Backend Tests {#backend-tests}
**Assigned**: Developer
**Files**: `cli/src/internal/azure/*_test.go`, `cli/src/internal/dashboard/*_test.go`

Write unit tests:
- ListAvailableTables parsing and error handling
- BuildQueryFromTables output validation
- Config API endpoint responses
- azure.yaml config loading/merging logic

## TODO: Frontend Tests {#frontend-tests}
**Assigned**: Developer
**Files**: `cli/dashboard/src/components/*.test.tsx`

Write component tests:
- TableSelector selection behavior
- LogConfigPanel mode switching
- Form validation
- API hook integration

## TODO: E2E Tests {#e2e-tests}
**Assigned**: Tester
**File**: `cli/dashboard/e2e/log-config.spec.ts` (new)

Write E2E tests:
- Open config panel from historical logs
- Select tables and save
- Switch to custom mode and save
- Verify logs refresh with new config
