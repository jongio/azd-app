# Log Analytics Table Selector Feature Spec

## Overview

Enable users to select which Log Analytics tables to query per service, providing a simple table selection mode and an advanced custom KQL mode. Configuration persists to azure.yaml.

**Zero-config by default:** The feature works automatically without any configuration. Tables are auto-selected based on resource type, and workspace is auto-detected from Azure environment.

## User Stories

1. **Zero Config**: Logs just work without any configuration - defaults based on resource type
2. **Table Selection**: User selects specific tables to query for a service  
3. **Custom KQL**: User enters a custom KQL query for advanced filtering
4. **Table Discovery**: System fetches available tables from the Log Analytics workspace
5. **Configuration Persistence**: Table/query selections save to azure.yaml per service

## Architecture

### Data Flow

```
┌─────────────────┐      ┌──────────────────┐      ┌─────────────────────┐
│   TablePicker   │─────►│ /api/azure/tables│─────►│ LogAnalyticsClient  │
│   Component     │      │   GET endpoint   │      │  .ListTables()      │
└─────────────────┘      └──────────────────┘      └─────────────────────┘
        │                                                     │
        │                                                     ▼
        │                                          ┌─────────────────────┐
        ▼                                          │  Azure Log         │
┌─────────────────┐      ┌──────────────────┐      │  Analytics API     │
│   User saves    │─────►│ PUT /api/azure/  │      └─────────────────────┘
│   selection     │      │   logs/config    │
└─────────────────┘      └──────────────────┘
                                  │
                                  ▼
                         ┌──────────────────┐
                         │   azure.yaml     │
                         │   logs.analytics │
                         │   tables/query   │
                         └──────────────────┘
```

## API Design

### GET /api/azure/tables

Fetches available Log Analytics tables for the workspace.

**Query Parameters:**
- `filter` (optional): Filter tables by prefix/category (e.g., "ContainerApp", "Function")

**Response:**
```json
{
  "tables": [
    {
      "name": "ContainerAppConsoleLogs_CL",
      "category": "containerapp",
      "description": "Console logs from Container Apps",
      "columns": ["TimeGenerated", "Log_s", "ContainerAppName_s", "Stream_s"]
    },
    {
      "name": "AppServiceConsoleLogs",
      "category": "appservice",
      "description": "Console logs from App Service",
      "columns": ["TimeGenerated", "ResultDescription", "Level"]
    }
  ],
  "recommended": ["ContainerAppConsoleLogs_CL"],
  "workspace": "workspace-id-truncated"
}
```

### GET /api/azure/logs/config

Gets the current log configuration for a service.

**Query Parameters:**
- `service` (required): Service name

**Response:**
```json
{
  "service": "api",
  "mode": "tables",
  "tables": ["ContainerAppConsoleLogs_CL", "ContainerAppSystemLogs_CL"],
  "query": null,
  "resourceType": "containerapp"
}
```

### PUT /api/azure/logs/config

Saves log configuration for a service.

**Request Body (Simple Mode):**
```json
{
  "service": "api",
  "mode": "tables",
  "tables": ["ContainerAppConsoleLogs_CL", "ContainerAppSystemLogs_CL"]
}
```

**Request Body (Custom Mode):**
```json
{
  "service": "api",
  "mode": "custom",
  "query": "ContainerAppConsoleLogs_CL | where ContainerAppName_s == 'api' | project TimeGenerated, Log_s"
}
```

**Response:**
```json
{
  "success": true,
  "service": "api",
  "mode": "tables",
  "tables": ["ContainerAppConsoleLogs_CL"]
}
```

## azure.yaml Schema

**Zero-config is the default.** All fields are optional - everything is auto-detected.

### Project Level (optional)
```yaml
logs:
  # Suppress noisy patterns from stdout/stderr
  filters:
    exclude: ["npm warn", "Debugger listening"]
  
  # Override log levels based on text matches
  classifications:
    - text: "DEPRECATED"
      level: "warning"
    - text: "fatal"
      level: "error"
  
  # Log Analytics settings (all optional)
  analytics:
    workspace: "/subscriptions/..."   # auto-detected from Azure env
    pollingInterval: "10s"            # default: 10s
    defaultTimespan: "30m"            # default: 30m
```

### Service Level (optional)
```yaml
services:
  api:
    logs:
      analytics:
        tables:                        # override: query these specific tables
          - ContainerAppConsoleLogs_CL
          - ContainerAppSystemLogs_CL
  
  worker:
    logs:
      analytics:
        query: |                       # override: use custom KQL
          FunctionAppLogs
          | where FunctionName == '{serviceName}'
          | where TimeGenerated > ago({timespan})
```

### Query Resolution Logic
```
1. Service has `query`  → use it
2. Service has `tables` → build union query from tables
3. Neither             → use defaults for resource type (auto)
```

## Frontend Components

### TableSelector Component

New component for selecting Log Analytics tables.

**Props:**
```typescript
interface TableSelectorProps {
  serviceName: string
  selectedTables: string[]
  onTablesChange: (tables: string[]) => void
  availableTables: TableInfo[]
  isLoading: boolean
  disabled?: boolean
}

interface TableInfo {
  name: string
  category: string
  description: string
  columns: string[]
}
```

**Features:**
- Multi-select checkbox list grouped by category
- Search/filter tables by name
- Shows column preview on hover
- Recommended tables highlighted
- "Select All" / "Clear All" actions

### LogConfigPanel Component

Modal or panel for configuring log source per service.

**Props:**
```typescript
interface LogConfigPanelProps {
  serviceName: string
  isOpen: boolean
  onClose: () => void
  onSave: (config: LogConfig) => void
  initialConfig?: LogConfig
}

interface LogConfig {
  mode: 'tables' | 'custom'
  tables?: string[]
  query?: string
}
```

**UI Layout:**
```
┌─────────────────────────────────────────┐
│  Configure Logs: api                  X │
├─────────────────────────────────────────┤
│  Mode:  [● Tables] [ Custom KQL ]       │
├─────────────────────────────────────────┤
│  Available Tables:              [🔍]    │
│  ┌─────────────────────────────────┐    │
│  │ ☑ ContainerAppConsoleLogs_CL   │    │
│  │ ☑ ContainerAppSystemLogs_CL    │    │
│  │ ☐ FunctionAppLogs              │    │
│  │ ☐ AppServiceConsoleLogs        │    │
│  └─────────────────────────────────┘    │
│                                         │
│  Selected: 2 tables                     │
├─────────────────────────────────────────┤
│             [Cancel]  [Save]            │
└─────────────────────────────────────────┘
```

**Custom KQL Mode:**
```
┌─────────────────────────────────────────┐
│  Configure Logs: api                  X │
├─────────────────────────────────────────┤
│  Mode:  [ Tables] [● Custom KQL ]       │
├─────────────────────────────────────────┤
│  KQL Query:                             │
│  ┌─────────────────────────────────┐    │
│  │ ContainerAppConsoleLogs_CL     │    │
│  │ | where ContainerAppName_s ==  │    │
│  │   '{serviceName}'              │    │
│  │ | project TimeGenerated, Log_s │    │
│  └─────────────────────────────────┘    │
│                                         │
│  Variables: {serviceName}, {timespan}   │
│                                         │
│  [Test Query]                           │
├─────────────────────────────────────────┤
│             [Cancel]  [Save]            │
└─────────────────────────────────────────┘
```

## Backend Implementation

### Go SDK Table Discovery

Use Azure Monitor Query SDK to list tables:

```go
// ListAvailableTables fetches tables from Log Analytics workspace
func (c *LogAnalyticsClient) ListAvailableTables(ctx context.Context) ([]TableInfo, error) {
    // Query the schema to get available tables
    query := `
    search *
    | distinct $table
    | order by $table asc
    `
    // Or use Azure Resource Manager API:
    // GET /subscriptions/{sub}/resourceGroups/{rg}/providers/
    //     Microsoft.OperationalInsights/workspaces/{ws}/tables
}
```

### Query Builder

Generate KQL from selected tables:

```go
// BuildQueryFromTables generates a union query for multiple tables
func BuildQueryFromTables(tables []string, serviceName string, timespan string) string {
    if len(tables) == 0 {
        return ""
    }
    if len(tables) == 1 {
        return buildSingleTableQuery(tables[0], serviceName, timespan)
    }
    // Union multiple tables
    var parts []string
    for _, table := range tables {
        parts = append(parts, buildSingleTableQuery(table, serviceName, timespan))
    }
    return strings.Join(parts, "\n| union ")
}
```

## Configuration Loading

When fetching logs, the system checks (in order):

1. Service-level `logs.analytics.query` (custom KQL takes precedence)
2. Service-level `logs.analytics.tables` (table selection)
3. Project-level `logs.analytics.query`
4. Project-level `logs.analytics.tables`
5. Default tables for the resource type (auto)

```go
func (s *Server) getLogConfigForService(serviceName string) *LogConfig {
    azureYaml := loadAzureYaml(s.projectDir)
    
    // Check service-level config
    if svc, ok := azureYaml.Services[serviceName]; ok {
        if svc.Logs != nil && svc.Logs.Analytics != nil {
            // Query takes precedence over tables
            if svc.Logs.Analytics.Query != "" {
                return &LogConfig{Mode: "custom", Query: svc.Logs.Analytics.Query}
            }
            if len(svc.Logs.Analytics.Tables) > 0 {
                return &LogConfig{Mode: "tables", Tables: svc.Logs.Analytics.Tables}
            }
        }
    }
    
    // Check project-level config
    if azureYaml.Logs != nil && azureYaml.Logs.Analytics != nil {
        if azureYaml.Logs.Analytics.Query != "" {
            return &LogConfig{Mode: "custom", Query: azureYaml.Logs.Analytics.Query}
        }
        if len(azureYaml.Logs.Analytics.Tables) > 0 {
            return &LogConfig{Mode: "tables", Tables: azureYaml.Logs.Analytics.Tables}
        }
    }
    
    // Return default for resource type (auto-detected)
    return getDefaultLogConfig(serviceName)
}
```

## Predefined Table Categories

```go
var TableCategories = map[string][]string{
    "containerapp": {
        "ContainerAppConsoleLogs_CL",
        "ContainerAppSystemLogs_CL",
    },
    "appservice": {
        "AppServiceConsoleLogs",
        "AppServiceHTTPLogs",
        "AppServicePlatformLogs",
    },
    "function": {
        "FunctionAppLogs",
        "AppServiceConsoleLogs",
    },
    "aks": {
        "ContainerLogV2",
        "KubeEvents",
        "KubePodInventory",
    },
    "aci": {
        "ContainerInstanceLog_CL",
        "ContainerEvent_CL",
    },
}
```

## Testing Requirements

### Unit Tests
- Table list parsing from API response
- Query generation from selected tables
- Configuration merging (service vs project level)
- azure.yaml serialization/deserialization

### Integration Tests
- API endpoint responses
- Configuration persistence roundtrip
- Live table discovery (with mocked workspace)

### E2E Tests
- Table selector UI interaction
- Save and reload configuration
- Query execution with selected tables

## Implementation Tasks

1. **Backend: Table Discovery API** - Add ListTables to LogAnalyticsClient
2. **Backend: Config API** - GET/PUT /api/azure/logs/config endpoints
3. **Backend: Query Builder** - Generate KQL from table selections
4. **Schema: azure.yaml** - Add tables/mode fields to schema
5. **Frontend: TableSelector** - New component for table selection
6. **Frontend: LogConfigPanel** - Modal for log configuration
7. **Frontend: Integration** - Add config button to HistoricalLogPanel
8. **Tests: Unit** - Backend and frontend unit tests
9. **Tests: E2E** - Full workflow tests
10. **Docs: Update** - CLI reference and feature documentation

## Success Criteria

- Users can select specific tables without writing KQL
- Custom KQL mode available for advanced users
- Configuration persists in azure.yaml
- UI provides clear feedback on available/selected tables
- Works across all supported resource types (Container Apps, App Service, Functions, AKS, ACI)
