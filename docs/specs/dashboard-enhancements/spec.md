# Dashboard Enhancements Specification

## Overview

This specification details new features to implement on top of the existing dashboard in main branch. The dashboard already provides real-time service monitoring, logs streaming, health monitoring, and service operations. This spec covers **additional enhancements** to build.

**Starting Point:** Branch from `main` and implement features below.

---

## Features to Implement

### New Components to Create
- `EnvironmentPanel` - Environment variable viewer with show/hide, copy, filtering
- `QuickActions` - Stats cards + action buttons  
- `PerformanceMetrics` - Aggregate metrics view
- `ServiceDependencies` - Service grouping/visualization
- `ServiceDetailPanel` - Slide-in detail panel with tabs
- `KeyboardShortcuts` - Shortcut modal dialog
- `ErrorBoundary` - Error handling wrapper
- `ui/InfoField` - Reusable info display component

### New Hooks to Create
- `useClipboard` - Copy to clipboard with feedback
- `useEscapeKey` - Escape key handler for modals

### New Sidebar Views (add to existing Resources, Console)
- **Metrics** view - Performance metrics dashboard
- **Environment** view - Environment variables panel
- **Actions** view - Quick actions dashboard
- **Dependencies** view - Service dependency visualization

### New Features
- **Keyboard shortcuts** (1-6 navigation, T toggle view, ? help, etc.)
- **Service detail slide-in panel** with Overview/Local/Azure/Environment tabs
- **Extended Azure info** in types (resourceType, resourceGroup, location, subscriptionId, logAnalyticsId, containerAppEnvId)

---

## What Already Exists in Main (DO NOT rebuild)

**Views & Navigation:**
- Resources view with table/grid toggle
- Console view with full logs functionality
- Sidebar (2 views: Resources, Console)
- View preference persistence (localStorage)

**Service Components:**
- `ServiceCard` - Full card with health display, ServiceActions
- `ServiceTable` + `ServiceTableRow` - Table view with all columns
- `ServiceActions` - Start/Stop/Restart/Browse/Logs buttons
- `ServiceStatusCard` - Status card component
- `StatusCell`, `URLCell` - Table cell components

**Logs System:**
- `LogsView` - Single pane log viewer
- `LogsMultiPaneView` - Multi-pane grid layout
- `LogsPane` + `LogsPaneGrid` - Individual panes
- `LogsToolbar` - Pause/Clear/Export/Fullscreen
- `useLogStream` - WebSocket log streaming with reconnection
- Service filtering, search, ANSI color rendering
- Auto-scroll, pause on scroll, jump to bottom
- Log level filtering (info/warning/error), health status filtering

**Real-Time Updates:**
- `useServices` - WebSocket service updates (`/api/ws`)
- `useHealthStream` - SSE health streaming (`/api/health/stream`)
- Automatic reconnection with exponential backoff
- Mock data fallback for development

**Service Operations:**
- `useServiceOperations` - Start/Stop/Restart individual + bulk
- API endpoints: `/api/services/:name/start|stop|restart`
- Bulk operations: `/api/services/start-all|stop-all|restart-all`
- Toast notifications for operation results

**Health Monitoring:**
- `useHealthStream` - Full health streaming hook
- Health summary (healthy/degraded/unhealthy/unknown counts)
- Health change events with recovery detection
- Per-service health status in UI

**Notification System:**
- `NotificationBadge` - Unread count badge
- `NotificationCenter` - Full history with search/filter/group
- `NotificationStack` + `NotificationToast` - Toast notifications
- `useNotifications` - Full notification management
- LocalStorage persistence of notification history

**Preferences:**
- `usePreferences` - Load/save user preferences
- API: `/api/logs/preferences`
- Grid columns, view mode, auto-scroll settings
- Copy format preferences

**Settings & Theme:**
- `ThemeToggle` - Theme switcher
- `SettingsModal` - Settings configuration
- `useTheme` - Theme management

**Log Classifications:**
- `ClassificationsManager` - Custom log level overrides
- `useLogClassifications` - Classification hook

**UI Components:**
- Badge (with variants), Button, Input, Select
- Table (header/body/row/cell), Tabs
- Dropdown menu, Toast, Slider

**Types:**
- Full Service, LocalServiceInfo, AzureServiceInfo
- HealthCheckResult, HealthSummary, HealthReportEvent
- HealthChangeEvent, HeartbeatEvent
- LogEntry structure

**Utilities:**
- `log-utils.ts` - ANSI conversion, log levels, constants
- `service-utils.ts` - Status display, formatting, validation
- `storage-utils.ts` - Type-safe localStorage helpers

---

## Implementation Guide

### 1. Environment Variables Panel

**Component:** `EnvironmentPanel.tsx`

**Display:**
- Aggregated view of all environment variables across services
- Table format with: Variable name, Value, Associated services
- Service badges showing which services use each variable

**Features:**
- Search/filter by variable name or value
- Filter by service
- Show/Hide values toggle (masks sensitive values by default)
- Copy value to clipboard with visual feedback
- Sensitive value detection (keys containing: key, secret, password, token)

**Integration:**
- Add "Environment" view to Sidebar (icon: `Settings` or `Variable`)
- Wire to App.tsx with `activeView === 'environment'`

---

### 2. azd Environment Management (Future Enhancement)

Extend the Environment panel to provide full management of `azd` environments.

**Environment Selector:**
- Dropdown in panel header showing current azd environment
- List all environments from `azd env list`
- Switch between environments
- Visual indicator for currently active environment

**View azd Environment Variables:**
- Display all variables from `azd env get-values`
- Show environment metadata: name, subscription, location, resource group
- Distinguish between user-set vars and Azure-provisioned outputs
- Read-only view of Azure resource connection strings

**Edit Environment Variables:**
- Inline edit mode for existing variables
- Add new variable button with key/value input
- Save triggers `azd env set <key> <value>`
- Cancel discards changes
- Validation for key format (no spaces, valid characters)

**Delete Variables:**
- Delete button per variable (with confirmation)
- Executes `azd env set <key> ""` to clear

**Environment Lifecycle:**
- Create new environment: `azd env new <name>`
- Delete environment: `azd env delete <name>` (with confirmation)
- Refresh to sync with CLI state

**API Endpoints Required:**
```
GET  /api/azd/env              - List all environments
GET  /api/azd/env/current      - Get current environment name
POST /api/azd/env/select       - Switch active environment
GET  /api/azd/env/:name/values - Get environment values
PUT  /api/azd/env/:name/values - Set/update values
POST /api/azd/env              - Create new environment
DELETE /api/azd/env/:name      - Delete environment
```

**UI Components:**
- Environment selector dropdown
- Editable table rows with save/cancel
- Add variable form/modal
- Create environment modal
- Confirmation dialogs for destructive actions

**Security Considerations:**
- Mask sensitive values by default (secrets, keys, passwords, tokens)
- Require confirmation for delete operations
- Validate inputs to prevent command injection
- Don't expose subscription IDs in logs

---

### 3. Quick Actions Panel

**Component:** `QuickActions.tsx`

**Dashboard Stats:**
- Running services count
- Healthy services count
- Error services count
- Visual cards with icons and color coding

**Global Actions:**
- Refresh All - Refresh all service statuses
- Clear Logs - Clear all log buffers
- Export Logs - Download logs as file
- Open Terminal - Open system terminal

> **Note:** Individual service Start/Stop/Restart already exists in main via `ServiceActions`.

**Integration:**
- Add "Actions" view to Sidebar (icon: `Zap` or `Play`)
- Wire to App.tsx with `activeView === 'actions'`

---

### 4. Performance Metrics View

**Component:** `PerformanceMetrics.tsx`

**Aggregate Metrics:**
- Active Services (running/total)
- Active Ports count
- Average Uptime
- Health Score percentage

**Metric Cards:**
- Icon-based display
- Value with unit
- Trend indicator (up/down/stable)
- Color coding by metric type

**Service-Level Metrics Table:**
- Per-service: Status, Uptime, Port, Health
- Visual status indicators
- Sortable/scrollable table

**Integration:**
- Add "Metrics" view to Sidebar (icon: `BarChart` or `Activity`)
- Wire to App.tsx with `activeView === 'metrics'`

---

### 5. Service Dependencies View

**Component:** `ServiceDependencies.tsx`

**Visualization:**
- Services grouped by language/technology
- Visual representation of service architecture
- Connection flow diagram for multi-service setups

**Per-Service Display:**
- Service name with status indicator
- Framework and port information
- Local URL link
- Environment variable count
- Status animation for running services

**Integration:**
- Add "Dependencies" view to Sidebar (icon: `GitBranch` or `Network`)
- Wire to App.tsx with `activeView === 'dependencies'`

---

### 6. Service Detail Panel

**Component:** `ServiceDetailPanel.tsx`

**Slide-In Panel:**
- Right-side slide-in panel (500px width)
- Backdrop blur when open
- Close on Escape key or backdrop click (use `useEscapeKey` hook)
- Tabbed interface: Overview, Local, Azure, Environment

**Overview Tab:**
- Local development status summary
- Azure deployment summary (if deployed)
- Condensed view of key information

**Local Tab:**
- Full local service details
- Status, Health, URL, Port, PID
- Start time, Last checked timestamps

**Azure Tab:**
- Azure resource information
- Resource name, type, group, location
- Subscription ID, endpoint URL
- Container image, environment ID
- "Open Azure Service" external link

**Environment Tab:**
- Service-specific environment variables
- Copyable values with visual feedback (use `useClipboard` hook)

**Integration:**
- Triggered from ServiceCard or ServiceTableRow click
- Pass selected service as prop
- Manage open/close state in App.tsx

---

### 7. Keyboard Shortcuts

**Component:** `KeyboardShortcuts.tsx`

**Navigation Shortcuts:**
- `1` - Resources view
- `2` - Console view
- `3` - Metrics view
- `4` - Environment view
- `5` - Actions view
- `6` - Dependencies view

**Action Shortcuts:**
- `R` - Refresh all services
- `C` - Clear console logs
- `E` - Export logs
- `/` or `Ctrl+F` - Focus search input

**View Shortcuts:**
- `T` - Toggle table/grid view
- `?` - Show keyboard shortcuts modal
- `Escape` - Close dialogs/modals

**Shortcuts Modal:**
- Modal dialog showing all shortcuts
- Grouped by category (Navigation, Actions, Views)
- Visual key badges

**Integration:**
- Add global keydown listener in App.tsx
- Modal triggered by `?` key or Help icon in header

---

### 8. Supporting Hooks

**`useClipboard.ts`:**
```typescript
export function useClipboard() {
  const [copiedField, setCopiedField] = useState<string | null>(null)
  
  const copyToClipboard = async (text: string, fieldName: string) => {
    await navigator.clipboard.writeText(text)
    setCopiedField(fieldName)
    setTimeout(() => setCopiedField(null), 2000)
  }
  
  return { copiedField, copyToClipboard }
}
```

**`useEscapeKey.ts`:**
```typescript
export function useEscapeKey(onEscape: () => void, isActive = true) {
  useEffect(() => {
    if (!isActive) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onEscape()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onEscape, isActive])
}
```

---

### 9. Extended Azure Info in Types

Add these fields to AzureServiceInfo (main only has url, resourceName, imageName):
```typescript
interface AzureServiceInfo {
  url?: string           // EXISTS in main
  resourceName?: string  // EXISTS in main
  imageName?: string     // EXISTS in main
  // NEW fields:
  resourceType?: string  // containerapp, appservice, function, etc.
  resourceGroup?: string
  location?: string
  subscriptionId?: string
  logAnalyticsId?: string
  containerAppEnvId?: string
}
```

---

### 10. ErrorBoundary Component

**Component:** `ErrorBoundary.tsx`

```typescript
interface ErrorBoundaryState {
  hasError: boolean
  error?: Error
}

class ErrorBoundary extends React.Component<Props, ErrorBoundaryState> {
  // Catches errors in child components
  // Displays fallback UI instead of crashing
  // Logs errors for debugging
}
```

---

### 11. InfoField UI Component

**Component:** `ui/InfoField.tsx`

Reusable component for displaying label/value pairs:
```typescript
interface InfoFieldProps {
  label: string
  value: string | React.ReactNode
  copyable?: boolean
  onCopy?: () => void
}
```

---

## Sidebar Updates

Update `Sidebar.tsx` to add new views:

```typescript
const navItems = [
  { id: 'resources', label: 'Resources', icon: Activity },
  { id: 'console', label: 'Console', icon: Terminal },
  // NEW views:
  { id: 'metrics', label: 'Metrics', icon: BarChart2 },
  { id: 'environment', label: 'Environment', icon: Settings },
  { id: 'actions', label: 'Actions', icon: Zap },
  { id: 'dependencies', label: 'Dependencies', icon: GitBranch },
]
```

---

## App.tsx Updates

Add rendering for new views:

```typescript
const renderContent = () => {
  if (activeView === 'resources') { /* existing */ }
  if (activeView === 'console') { /* existing */ }
  
  // NEW views:
  if (activeView === 'metrics') {
    return <PerformanceMetrics services={services} />
  }
  if (activeView === 'environment') {
    return <EnvironmentPanel services={services} />
  }
  if (activeView === 'actions') {
    return <QuickActions services={services} onRefresh={refetch} />
  }
  if (activeView === 'dependencies') {
    return <ServiceDependencies services={services} />
  }
}
```

---

## Testing Requirements

- Unit tests for all new components
- Unit tests for new hooks (`useClipboard`, `useEscapeKey`)
- Unit tests for utility functions
- E2E tests for critical user flows
- Test coverage ≥80%

---

## Future Enhancement Ideas

These are ideas for AFTER implementing the core features above.

### Priority Matrix

| Priority | Feature | Impact | Effort |
|----------|---------|--------|--------|
| **P0** | azd Environment Management | High | Medium |
| **P1** | Azure Deployment Integration | High | Medium |
| **P1** | Enhanced Logging (regex, time range) | Medium | Low |
| **P2** | Command Palette | Medium | Medium |
| **P2** | Health History/Graphs | Medium | Medium |
| **P3** | azure.yaml Editor | Medium | High |
| **P3** | Azure Resource Explorer | Medium | High |
| **P3** | Git Integration | Low | Medium |
| **P4** | Cost Tracking | Low | High |
| **P4** | Collaboration Features | Low | High |
| **P4** | Widget Customization | Low | Medium |

---

### P0 - Critical (Immediate)

#### A. azd Environment Management

(See section 2 for full details)

---

### P1 - High Priority (Near Term)

#### A. Azure Deployment Integration

**Deploy from Dashboard:**
- "Deploy to Azure" button per service or global
- Trigger `azd deploy` with progress indicator
- Show deployment logs in Console view
- Display deployment status (in-progress, succeeded, failed)

**Provision Resources:**
- "Provision" button to run `azd provision`
- Infrastructure diff preview before provisioning
- Cost estimation display (if available from Azure)

**Azure Portal Links:**
- Direct links to Azure Portal for each resource
- Link to resource group overview
- Link to Log Analytics / Application Insights

#### B. Enhanced Logging Features

> **Note:** Main has: service filtering, log level filtering, text search, pause/resume. These are ADDITIONS:

**Advanced Filtering:**
- Regex search support
- Time range filtering

**Log Analysis:**
- Error pattern detection
- Duplicate log grouping
- Frequency analysis (errors per minute)

---

### P2 - Medium Priority (Future)

#### C. Developer Experience

**Command Palette:**
- `Cmd/Ctrl+K` to open command palette
- Quick access to all actions
- Fuzzy search for commands

**Session Persistence:**
- Remember active view on reload
- Restore scroll position
- Remember expanded/collapsed states

**Bookmarks/Favorites:**
- Pin frequently used services
- Quick access section in sidebar

#### D. Health Monitoring Enhancements

> **Note:** Main has: `useHealthStream` with SSE, health summary, change events. These are ADDITIONS:

**Health History:**
- Graph of health status over time
- Uptime percentage calculation
- Mean time between failures

**Alerting Enhancements:**
> Main has toast notifications. These are additions:
- Browser/OS desktop notifications (not just in-app toasts)
- Sound alerts for errors
- Configurable alert thresholds

**Health Check Configuration:**
- Custom health endpoints per service
- Configurable check intervals
- Timeout settings

#### E. Log Persistence

- Save logs to file automatically (main has manual export)
- Log rotation settings
- Configurable log buffer size

---

### P3 - Low Priority (Backlog)

#### G. azure.yaml Editor Experience

**Visual Schema Editor:**
- Form-based editing of azure.yaml
- Service configuration UI with dropdowns for host types (appservice, containerapp, function, staticwebapp)
- Language/framework auto-detection display
- Port and path configuration
- Environment variable management per service

**Validation & Assistance:**
- Real-time schema validation against official schema
- Error highlighting with helpful messages
- Auto-complete for known values (Azure regions, SKUs)
- Link to schema documentation

**Service Management:**
- Add new service wizard
- Remove service with confirmation
- Duplicate service as template
- Reorder services via drag-and-drop

**Preview & Apply:**
- Side-by-side YAML preview
- Diff view before saving
- Syntax highlighting in preview
- Format/prettify option

**Implementation Notes:**
```typescript
// Service configuration form
interface ServiceEditorProps {
  service: AzureYamlService;
  onUpdate: (service: AzureYamlService) => void;
  onDelete: () => void;
}

// azure.yaml structure
interface AzureYaml {
  name: string;
  services: Record<string, AzureYamlService>;
  metadata?: {
    template?: string;
  };
}
```

#### H. Azure Resource Explorer

**Resource Tree View:**
- Hierarchical view of Azure resources
- Subscription → Resource Group → Resources
- Expand to see resource details

**Resource Metrics:**
- Pull metrics from Azure Monitor
- CPU/Memory for Container Apps
- Request counts for App Services
- Display inline or in dedicated view

#### I. Integration with Development Tools

**VS Code Integration:**
- Open service folder in VS Code
- Debug service attachment
- Quick edit configuration files

**Git Integration:**
- Show current branch per service
- Display uncommitted changes indicator
- Link to GitHub/ADO repository

**Container Integration:**
- Docker container status
- Container logs access
- Image version information

#### J. Offline/Disconnected Mode

**Graceful Degradation:**
- Cache last known state
- Show stale data indicator
- Queue actions for reconnection

**Local Development Focus:**
- Work without Azure connectivity
- Mock Azure responses for testing
- Local-only mode toggle

---

### P4 - Future Consideration

#### K. Cost Tracking

- Show estimated costs per resource
- Budget alerts integration
- Cost trend visualization

#### L. Collaboration Features

**Share Configuration:**
- Export dashboard settings
- Import team configurations
- Sync preferences across machines

**Activity Log:**
- Track who started/stopped services
- Deployment history with user info
- Audit trail for env var changes

#### M. Customization

**Dashboard Themes:**
- Light/Dark/System theme (exists)
- Custom accent colors
- Compact/Comfortable density modes

**Widget Layout:**
- Drag-and-drop dashboard widgets
- Resizable panels
- Save custom layouts

**Notification Preferences:**
- Configure which events notify
- Quiet hours setting
- Per-service notification rules
