# Azure Integration Implementation Plan

## User Request Summary
From comment #3505589730:
1. Show all relevant info from Azure (resources in the app)
2. Info from resource groups and metadata about each service
3. Stream logs from each Azure service into the dashboard
4. Detail view for each running service and supporting services

## Current State
- ✅ Azure environment variables already accessible via `azd env get-values`
- ✅ Basic Azure info already in ServiceInfo (url, resourceName, imageName)
- ✅ Azure URL displayed in ServiceCard component
- ✅ Environment variables panel already implemented

## Proposed Implementation

### Phase 1: Enhanced Azure Metadata (Backend)
**File: cli/src/internal/serviceinfo/azure.go** (NEW)

Add extended Azure resource information:
```go
type AzureResourceInfo struct {
    ResourceName     string `json:"resourceName"`
    ResourceType     string `json:"resourceType"`     // "containerapp", "appservice", etc.
    ResourceGroup    string `json:"resourceGroup"`
    Location         string `json:"location"`
    SubscriptionId   string `json:"subscriptionId"`
    URL              string `json:"url"`
    ImageName        string `json:"imageName"`
    Status           string `json:"status"`            // "Running", "Stopped", etc.
    Sku              string `json:"sku,omitempty"`
    Tags             map[string]string `json:"tags,omitempty"`
}
```

Extract from environment variables:
- AZURE_RESOURCE_GROUP
- AZURE_LOCATION
- AZURE_SUBSCRIPTION_ID
- SERVICE_{NAME}_RESOURCE_TYPE
- SERVICE_{NAME}_STATUS

### Phase 2: Azure Log Streaming (Backend)
**File: cli/src/internal/azure/logs.go** (NEW)

Options:
A) Use Azure CLI wrapper (simpler, no SDK deps)
   ```bash
   az containerapp logs show --name {name} --resource-group {rg} --follow
   ```

B) Use Azure Monitor SDK (more control, requires workspace ID)
   - Query Log Analytics workspace
   - Stream logs via WebSocket

Recommendation: Start with Option A (CLI wrapper) as it's simpler and works immediately.

**File: cli/src/internal/dashboard/server.go** (UPDATE)

Add endpoints:
- `GET /api/azure/resources` - List Azure resources
- `WS /api/azure/logs/{serviceName}` - Stream Azure logs

### Phase 3: Enhanced Frontend

**File: cli/dashboard/src/types.ts** (UPDATE)
```typescript
export interface AzureServiceInfo {
  url?: string
  resourceName?: string
  resourceType?: 'containerapp' | 'appservice' | 'function' | 'unknown'
  resourceGroup?: string
  location?: string
  subscriptionId?: string
  imageName?: string
  status?: 'Running' | 'Stopped' | 'Starting' | 'Unknown'
  sku?: string
  tags?: Record<string, string>
}
```

**Component: ServiceDetailModal.tsx** (NEW)
- Tabs: Overview, Local, Azure, Logs, Environment
- Shows all local + Azure info
- Embedded log viewer with tabs for local/Azure logs
- Environment variables specific to this service

**Component: AzureResourcesPanel.tsx** (NEW)
- List all Azure resources from resource group
- Group by type (Container Apps, App Services, etc.)
- Show status, location, SKU
- Click to view details

**Component: AzureLogsView.tsx** (NEW)
- Stream Azure logs alongside local logs
- Filter by service
- Search and filtering
- Export capability

### Phase 4: Integration Points

Update existing components:
- **ServiceCard.tsx**: Add "View Details" button → opens ServiceDetailModal
- **ServiceTable.tsx**: Add detail icon → opens ServiceDetailModal
- **LogsView.tsx**: Add Azure logs tab
- **Sidebar.tsx**: Add "Azure Resources" view

## Implementation Priority (pending user confirmation)

### Option A: Metadata First
1. Extend AzureServiceInfo with resource group, location, type, status
2. Update ServiceCard to show extended Azure info
3. Create ServiceDetailModal with tabs
4. Add Azure Resources view

### Option B: Logs First  
1. Implement Azure log streaming backend
2. Create AzureLogsView component
3. Integrate into existing LogsView
4. Add service detail modal later

### Option C: Comprehensive
1. All of the above in sequence

## Questions for User
1. Target Azure services: Container Apps, App Service, or both?
2. Log source: Azure CLI or direct SDK integration?
3. Priority: Metadata display (A) vs Log streaming (B) vs Both (C)?
4. Auto-detect resources or require configuration?

## Risks & Considerations
- Azure CLI must be installed and authenticated
- Log Analytics workspace ID needed for SDK approach
- Additional Azure SDK dependencies if not using CLI
- Performance impact of streaming multiple log sources
- Rate limiting on Azure APIs

## Testing Strategy
- Mock Azure CLI responses for testing
- Create test fixtures with sample Azure metadata
- E2E tests for log streaming
- UI tests for detail modal and Azure panels
