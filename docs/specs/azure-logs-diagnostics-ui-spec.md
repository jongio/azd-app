# Azure Logs Diagnostics UI Specification

**Version:** 1.0  
**Date:** December 29, 2025  
**Status:** Complete Specification  
**Target:** React/TypeScript Dashboard

---

## Overview

This specification defines the user interface components for the Azure logs diagnostic system. The system helps users troubleshoot why Azure logs aren't appearing by displaying service health status, configuration requirements, and setup guides.

### Design Philosophy

- **Zero-touch**: Logs should stream automatically when Azure is configured correctly
- **Diagnostic-first**: When logs aren't appearing, provide actionable diagnostics
- **Progressive disclosure**: Show summary first, details on demand
- **Accessibility**: WCAG 2.1 AA compliant throughout

---

## Component Architecture

```
LogsPane
├── LogsPaneHeader
│   ├── Diagnostic Button ──────► Opens DiagnosticModal
│   └── [Azure] Tab
├── LogsPaneContent
│   └── NoLogsPrompt ───────────► Links to DiagnosticModal
└── DiagnosticModal (overlay)
    ├── Header
    ├── Service Cards (expandable)
    └── Footer Actions
```

---

## 1. DiagnosticModal Component

### 1.1 Component Structure

```typescript
/**
 * DiagnosticModal - Full-screen modal for Azure logs diagnostics
 * Displays real-time health checks and troubleshooting guidance
 */

export interface DiagnosticModalProps {
  isOpen: boolean
  onClose: () => void
  onOpenSetupGuide?: (step: SetupStep) => void
  serviceName?: string  // Optional: focus on specific service
}

export interface DiagnosticData {
  timestamp: string
  status: 'healthy' | 'degraded' | 'error'
  services: ServiceDiagnostic[]
  overall: {
    totalServices: number
    healthyCount: number
    degradedCount: number
    errorCount: number
  }
}

export interface ServiceDiagnostic {
  serviceName: string
  hostType: 'containerapp' | 'appservice' | 'function' | 'aks' | string
  status: 'healthy' | 'degraded' | 'error'
  logCount: number
  lastLogTime?: string  // ISO timestamp
  requirements: DiagnosticRequirement[]
  setupGuide?: string  // Markdown content
}

export interface DiagnosticRequirement {
  id: string
  name: string
  status: 'pass' | 'warn' | 'fail'
  message: string
  fix?: string  // Command or action to fix
  docsUrl?: string
}
```

### 1.2 Layout Specification

```
┌─────────────────────────────────────────────────────────────┐
│ MODAL HEADER                                                │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ 🔍 Azure Logs Diagnostics                        [X]    │ │
│ │ Last checked: Dec 29, 2025 at 2:15 PM     [↻ Refresh]  │ │
│ └─────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│ SCROLLABLE CONTENT (max-h-[80vh])                          │
│                                                             │
│ ┌── SERVICE CARD: api ──────────────────────────────────┐  │
│ │ [✅] api                    [Container App] [Details▼] │  │
│ │ • 1,247 logs • Last: 2 minutes ago                     │  │
│ │                                                         │  │
│ │ [Expanded Details]                                      │  │
│ │ Requirements Checklist:                                 │  │
│ │ ✅ Log Analytics workspace configured                   │  │
│ │ ✅ Diagnostic settings enabled                          │  │
│ │ ✅ Authentication valid                                 │  │
│ │ ✅ Resource deployed                                    │  │
│ └─────────────────────────────────────────────────────────┘  │
│                                                             │
│ ┌── SERVICE CARD: web ──────────────────────────────────┐  │
│ │ [⚠️] web                    [App Service]  [Details▼] │  │
│ │ • 0 logs • No logs found                              │  │
│ │                                                         │  │
│ │ [Expanded Details]                                      │  │
│ │ Requirements Checklist:                                 │  │
│ │ ✅ Log Analytics workspace configured                   │  │
│ │ ⚠️ Diagnostic settings partially configured            │  │
│ │   → Missing: AppServiceHTTPLogs category               │  │
│ │   Fix: azd app azure-setup --service web               │  │
│ │ ✅ Authentication valid                                 │  │
│ │ ✅ Resource deployed                                    │  │
│ │                                                         │  │
│ │ [📖 View Setup Guide]                                  │  │
│ └─────────────────────────────────────────────────────────┘  │
│                                                             │
│ ┌── SERVICE CARD: worker ────────────────────────────────┐  │
│ │ [❌] worker                 [Function App] [Details▼] │  │
│ │ • 0 logs • Authentication required                     │  │
│ │                                                         │  │
│ │ [Expanded Details]                                      │  │
│ │ Requirements Checklist:                                 │  │
│ │ ✅ Log Analytics workspace configured                   │  │
│ │ ✅ Diagnostic settings enabled                          │  │
│ │ ❌ Authentication required                              │  │
│ │   → Run: azd auth login                                │  │
│ │   [📋 Copy Command]                                    │  │
│ │ ⚠️ Resource not deployed                               │  │
│ │                                                         │  │
│ │ [🔧 Fix Setup]                                         │  │
│ └─────────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│ FOOTER                                                      │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ ● All checks passing          [📋 Copy Report] [📚 Docs] │ │
│ │ Updated 1 second ago          [🔧 Fix Setup]   [Close] │ │
│ └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### 1.3 Visual Design Tokens

```typescript
const DiagnosticModalStyles = {
  // Modal Container
  modal: {
    maxWidth: '900px',
    maxHeight: '90vh',
    borderRadius: '16px',
    background: 'bg-white dark:bg-slate-900',
    border: 'border border-slate-200 dark:border-slate-700',
    shadow: 'shadow-2xl',
  },
  
  // Header
  header: {
    padding: 'px-6 py-4',
    borderBottom: 'border-b border-slate-200 dark:border-slate-700',
    background: 'bg-slate-50 dark:bg-slate-800/50',
  },
  
  // Status Indicators
  status: {
    healthy: {
      icon: '✅', // or CheckCircle component
      color: 'text-emerald-600 dark:text-emerald-400',
      bg: 'bg-emerald-50 dark:bg-emerald-900/30',
      border: 'border-emerald-200 dark:border-emerald-800',
    },
    degraded: {
      icon: '⚠️', // or AlertCircle component
      color: 'text-amber-600 dark:text-amber-400',
      bg: 'bg-amber-50 dark:bg-amber-900/30',
      border: 'border-amber-200 dark:border-amber-800',
    },
    error: {
      icon: '❌', // or XCircle component
      color: 'text-red-600 dark:text-red-400',
      bg: 'bg-red-50 dark:bg-red-900/30',
      border: 'border-red-200 dark:border-red-800',
    },
  },
  
  // Service Cards
  serviceCard: {
    base: 'rounded-lg border p-4 transition-all duration-200',
    healthy: 'bg-slate-50 dark:bg-slate-800/50 border-slate-200 dark:border-slate-700',
    degraded: 'bg-amber-50 dark:bg-amber-900/20 border-amber-200 dark:border-amber-700',
    error: 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-700',
  },
  
  // Buttons
  button: {
    primary: 'bg-cyan-600 hover:bg-cyan-700 text-white',
    secondary: 'bg-white hover:bg-slate-100 dark:bg-slate-800 dark:hover:bg-slate-700 border',
    danger: 'bg-red-600 hover:bg-red-700 text-white',
    warning: 'bg-orange-600 hover:bg-orange-700 text-white',
  },
}
```

### 1.4 Component States

#### 1.4.1 Loading State
```typescript
{
  isLoading: true,
  data: null,
  error: null
}
```

**UI Treatment:**
- Centered spinner with "Running health checks..." message
- Disable refresh button
- Gray out previous results if any

#### 1.4.2 Error State
```typescript
{
  isLoading: false,
  data: null,
  error: "Failed to fetch diagnostics: Network timeout"
}
```

**UI Treatment:**
- Show error icon (XCircle) with message
- Display "Retry" button
- Show last successful timestamp if available

#### 1.4.3 Success State
```typescript
{
  isLoading: false,
  data: DiagnosticData,
  error: null
}
```

**UI Treatment:**
- Render service cards
- Show overall status badge in footer
- Enable all actions

### 1.5 Interactions

#### 1.5.1 Expand/Collapse Service Card
```typescript
const [expandedServices, setExpandedServices] = useState<Set<string>>(new Set())

const toggleService = (serviceName: string) => {
  setExpandedServices(prev => {
    const next = new Set(prev)
    if (next.has(serviceName)) {
      next.delete(serviceName)
    } else {
      next.add(serviceName)
    }
    return next
  })
}
```

**Behavior:**
- Click card header or chevron to toggle
- Animate height transition (200ms ease-out)
- Persist expanded state during refresh
- Auto-expand services with errors

#### 1.5.2 View Setup Guide
```typescript
const [showingGuide, setShowingGuide] = useState<string | null>(null)

const handleViewGuide = (serviceName: string) => {
  setShowingGuide(serviceName)
  // Render markdown in expandable section
}
```

**Behavior:**
- Appears inline within service card
- Markdown rendered with syntax highlighting
- Collapsible with smooth transition
- "Copy Guide" button in header

#### 1.5.3 Copy Fix Command
```typescript
const handleCopyFix = async (command: string) => {
  await navigator.clipboard.writeText(command)
  // Show toast: "Command copied to clipboard"
}
```

**Behavior:**
- Copy button shows checkmark for 2 seconds
- Toast notification appears
- Button has accessible label

#### 1.5.4 Fix Setup Flow
```typescript
const handleFixSetup = (serviceName: string, failingRequirements: DiagnosticRequirement[]) => {
  // Determine which setup step failed
  const step = determineFailingStep(failingRequirements)
  
  // Close diagnostic modal
  onClose()
  
  // Open setup guide to specific step
  onOpenSetupGuide?.(step)
}
```

**Behavior:**
- Only shown when `onOpenSetupGuide` provided
- Closes diagnostic modal
- Opens setup guide modal to specific step
- Deep links to failing requirement

### 1.6 Accessibility

#### Keyboard Navigation
```typescript
// Focus trap within modal
const dialogRef = useRef<HTMLDialogElement>(null)
useFocusTrap(dialogRef, isOpen)

// Escape key to close
useEscapeKey(onClose, isOpen)

// Tab order:
// 1. Close button
// 2. Refresh button
// 3. Service card headers (focusable)
// 4. Expand buttons
// 5. Action buttons (Copy, View Guide, Fix Setup)
// 6. Footer buttons
```

#### ARIA Labels
```typescript
<dialog
  ref={dialogRef}
  aria-labelledby="diagnostics-title"
  aria-describedby="diagnostics-description"
  role="dialog"
  aria-modal="true"
>
  <h2 id="diagnostics-title">Azure Logs Diagnostics</h2>
  <p id="diagnostics-description" className="sr-only">
    Health check results and troubleshooting guidance for Azure logs
  </p>

  {services.map(service => (
    <section
      key={service.serviceName}
      aria-labelledby={`service-${service.serviceName}-title`}
    >
      <button
        aria-expanded={isExpanded}
        aria-controls={`service-${service.serviceName}-details`}
      >
        <h3 id={`service-${service.serviceName}-title`}>
          {service.serviceName}
        </h3>
      </button>
      
      <div
        id={`service-${service.serviceName}-details`}
        aria-hidden={!isExpanded}
      >
        {/* Requirements checklist */}
      </div>
    </section>
  ))}
</dialog>
```

#### Screen Reader Support
- Announce modal open: "Azure Logs Diagnostics dialog opened"
- Announce status changes: "Service api status changed from degraded to healthy"
- Announce refresh completion: "Diagnostics refreshed, 3 services checked, 1 warning found"
- Provide text alternatives for status icons

#### Color Contrast
- All text meets WCAG AA (4.5:1 for normal text)
- Status indicators use both color AND icons
- Focus indicators visible in both light/dark modes

---

## 2. NoLogsPrompt Component

### 2.1 Component Structure

```typescript
/**
 * NoLogsPrompt - Contextual prompt when no logs are available
 * Appears in log pane content area when service has 0 logs
 */

export interface NoLogsPromptProps {
  serviceName: string
  logMode: 'local' | 'azure'
  onOpenDiagnostics: () => void
  azureDeployed?: boolean  // Is service deployed to Azure?
}

interface NoLogsReason {
  title: string
  description: string
  icon: ReactNode
  severity: 'info' | 'warning' | 'error'
}
```

### 2.2 Layout Specification

```
┌─────────────────────────────────────────────┐
│         LOG PANE (when empty)               │
│                                             │
│     ╔═══════════════════════════════╗      │
│     ║  ⚠️  No Logs Available        ║      │
│     ║                               ║      │
│     ║  Service: api                 ║      │
│     ║                               ║      │
│     ║  This could mean:             ║      │
│     ║  • Not configured for Azure   ║      │
│     ║  • Propagation delay (5-10min)║      │
│     ║  • No recent activity         ║      │
│     ║                               ║      │
│     ║  [🔍 View Diagnostic Details] ║      │
│     ╚═══════════════════════════════╝      │
│                                             │
└─────────────────────────────────────────────┘
```

### 2.3 Visual Design

```typescript
const NoLogsPromptStyles = {
  container: {
    base: 'flex flex-col items-center justify-center py-12 px-6 text-center',
    minHeight: 'min-h-[300px]',
  },
  
  card: {
    base: 'max-w-md rounded-lg border p-6',
    info: 'bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800',
    warning: 'bg-amber-50 dark:bg-amber-900/20 border-amber-200 dark:border-amber-800',
    error: 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800',
  },
  
  icon: {
    container: 'w-12 h-12 mx-auto mb-4 rounded-full flex items-center justify-center',
    info: 'bg-blue-100 dark:bg-blue-900/40 text-blue-600 dark:text-blue-400',
    warning: 'bg-amber-100 dark:bg-amber-900/40 text-amber-600 dark:text-amber-400',
    error: 'bg-red-100 dark:bg-red-900/40 text-red-600 dark:text-red-400',
  },
}
```

### 2.4 Logic for Determining Reason

```typescript
function determineNoLogsReason(
  serviceName: string,
  azureDeployed: boolean,
  timeRange: AzureTimeRange
): NoLogsReason {
  // Not deployed to Azure yet
  if (!azureDeployed) {
    return {
      title: 'Service Not Deployed',
      description: 'This service hasn\'t been deployed to Azure yet. Deploy with "azd up" to see Azure logs.',
      icon: <CloudOff />,
      severity: 'info',
    }
  }
  
  // Recently deployed (< 10 minutes)
  const deployAge = getDeploymentAge(serviceName)
  if (deployAge && deployAge < 600000) { // 10 minutes in ms
    return {
      title: 'Propagation Delay',
      description: 'Logs can take 5-10 minutes to appear after deployment. Check back soon.',
      icon: <Clock />,
      severity: 'info',
    }
  }
  
  // Deployed but no logs
  return {
    title: 'No Logs Found',
    description: 'The service is deployed but has no logs in the selected time range. It may not be configured correctly or hasn\'t received traffic.',
    icon: <AlertTriangle />,
    severity: 'warning',
  }
}
```

### 2.5 Component States

#### Default State (Warning)
```typescript
<NoLogsPrompt
  serviceName="api"
  logMode="azure"
  onOpenDiagnostics={handleOpenDiagnostics}
  azureDeployed={true}
/>
```

Renders warning variant with diagnostic link.

#### Info State (Not Deployed)
```typescript
<NoLogsPrompt
  serviceName="web"
  logMode="azure"
  onOpenDiagnostics={handleOpenDiagnostics}
  azureDeployed={false}
/>
```

Renders info variant with deployment guidance.

### 2.6 Accessibility

```typescript
<div
  role="status"
  aria-live="polite"
  aria-label={`No logs available for ${serviceName}`}
>
  <div className={iconContainerClass}>
    <WarningIcon aria-hidden="true" />
  </div>
  
  <h3 className="font-semibold mb-2">
    No Logs Available
  </h3>
  
  <p className="text-sm text-muted-foreground mb-1">
    Service: <strong>{serviceName}</strong>
  </p>
  
  <ul className="text-sm text-muted-foreground mb-4 space-y-1">
    <li>Not configured for Azure</li>
    <li>Propagation delay (5-10 minutes)</li>
    <li>No recent activity</li>
  </ul>
  
  <button
    onClick={onOpenDiagnostics}
    aria-label={`Open diagnostic details for ${serviceName}`}
    className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-cyan-600 text-white hover:bg-cyan-700"
  >
    <Search className="w-4 h-4" aria-hidden="true" />
    View Diagnostic Details
  </button>
</div>
```

---

## 3. Diagnostic Button

### 3.1 Component Structure

```typescript
/**
 * DiagnosticButton - Opens diagnostic modal
 * Positioned in LogsPaneHeader after [Azure] tab
 */

export interface DiagnosticButtonProps {
  onClick: () => void
  hasIssues?: boolean  // Show indicator badge if diagnostics found issues
  disabled?: boolean
}
```

### 3.2 Placement

```
LogsPaneHeader Layout:
┌────────────────────────────────────────────────────┐
│ [Local] [Azure]  [🔍 Diagnostics]  │  [⚙️] [↻]  │
│         ^^^^^^    ^^^^^^^^^^^^^^^                  │
│       Active Tab   NEW BUTTON                      │
└────────────────────────────────────────────────────┘
```

### 3.3 Visual Design

```typescript
const DiagnosticButtonStyles = {
  base: cn(
    'inline-flex items-center gap-2 px-3 py-1.5 rounded-lg',
    'text-sm font-medium transition-colors',
    'focus:outline-none focus:ring-2 focus:ring-cyan-500',
  ),
  
  default: cn(
    'bg-white dark:bg-slate-800',
    'border border-slate-200 dark:border-slate-700',
    'text-slate-700 dark:text-slate-300',
    'hover:bg-slate-50 dark:hover:bg-slate-700',
  ),
  
  hasIssues: cn(
    'bg-amber-50 dark:bg-amber-900/30',
    'border border-amber-200 dark:border-amber-700',
    'text-amber-700 dark:text-amber-300',
    'hover:bg-amber-100 dark:hover:bg-amber-900/40',
  ),
  
  disabled: 'opacity-50 cursor-not-allowed',
}
```

### 3.4 Icon Options

**Option A: Stethoscope (Medical)**
```typescript
import { Stethoscope } from 'lucide-react'
<Stethoscope className="w-4 h-4" />
```

**Option B: Search (Investigation)**
```typescript
import { Search } from 'lucide-react'
<Search className="w-4 h-4" />
```

**Option C: Activity (Health Monitoring)**
```typescript
import { Activity } from 'lucide-react'
<Activity className="w-4 h-4" />
```

**Recommendation:** Use `Activity` for semantic clarity (health monitoring).

### 3.5 Badge Indicator

When diagnostics detect issues, show a badge:

```typescript
<button className={buttonClass}>
  <Activity className="w-4 h-4" />
  Diagnostics
  
  {hasIssues && (
    <span
      className="w-2 h-2 bg-amber-500 rounded-full"
      aria-label="Issues detected"
    />
  )}
</button>
```

### 3.6 Accessibility

```typescript
<button
  onClick={onClick}
  disabled={disabled}
  aria-label={hasIssues 
    ? "Open diagnostics - issues detected" 
    : "Open diagnostics"}
  className={cn(buttonClass, disabled && disabledClass)}
>
  <Activity 
    className="w-4 h-4" 
    aria-hidden="true"
  />
  <span>Diagnostics</span>
  
  {hasIssues && (
    <span
      className="w-2 h-2 bg-amber-500 rounded-full"
      role="status"
      aria-label="Issues detected"
    >
      <span className="sr-only">Warning: diagnostic issues found</span>
    </span>
  )}
</button>
```

---

## 4. Integration Points

### 4.1 API Contract

```typescript
// GET /api/azure/diagnostics
interface DiagnosticsResponse {
  timestamp: string
  status: 'healthy' | 'degraded' | 'error'
  services: ServiceDiagnostic[]
  overall: {
    totalServices: number
    healthyCount: number
    degradedCount: number
    errorCount: number
  }
}

// Backend should return:
{
  "timestamp": "2025-12-29T14:15:30Z",
  "status": "degraded",
  "services": [
    {
      "serviceName": "api",
      "hostType": "containerapp",
      "status": "healthy",
      "logCount": 1247,
      "lastLogTime": "2025-12-29T14:13:00Z",
      "requirements": [
        {
          "id": "workspace",
          "name": "Log Analytics workspace configured",
          "status": "pass",
          "message": "Workspace ID: abc123-workspace"
        },
        {
          "id": "diagnostics",
          "name": "Diagnostic settings enabled",
          "status": "pass",
          "message": "All required categories enabled"
        },
        {
          "id": "auth",
          "name": "Authentication valid",
          "status": "pass",
          "message": "Token expires in 45 minutes"
        },
        {
          "id": "deployed",
          "name": "Resource deployed",
          "status": "pass",
          "message": "Deployed to West US 2"
        }
      ]
    },
    {
      "serviceName": "web",
      "hostType": "appservice",
      "status": "degraded",
      "logCount": 0,
      "requirements": [
        {
          "id": "workspace",
          "name": "Log Analytics workspace configured",
          "status": "pass",
          "message": "Workspace ID: abc123-workspace"
        },
        {
          "id": "diagnostics",
          "name": "Diagnostic settings enabled",
          "status": "warn",
          "message": "Missing category: AppServiceHTTPLogs",
          "fix": "azd app azure-setup --service web",
          "docsUrl": "https://learn.microsoft.com/azure/app-service/troubleshoot-diagnostic-logs"
        },
        {
          "id": "auth",
          "name": "Authentication valid",
          "status": "pass",
          "message": "Token expires in 45 minutes"
        },
        {
          "id": "deployed",
          "name": "Resource deployed",
          "status": "pass",
          "message": "Deployed to West US 2"
        }
      ],
      "setupGuide": "## Configuring App Service Logs\n\n..."
    }
  ],
  "overall": {
    "totalServices": 2,
    "healthyCount": 1,
    "degradedCount": 1,
    "errorCount": 0
  }
}
```

### 4.2 State Management

```typescript
// Context for diagnostic state
interface DiagnosticContextValue {
  isOpen: boolean
  openDiagnostics: (serviceName?: string) => void
  closeDiagnostics: () => void
  diagnosticData: DiagnosticData | null
  isLoading: boolean
  error: string | null
  refresh: () => Promise<void>
}

// Usage in LogsPane
const { openDiagnostics } = useDiagnostics()

<DiagnosticButton onClick={() => openDiagnostics()} />
<NoLogsPrompt onOpenDiagnostics={() => openDiagnostics(serviceName)} />
```

### 4.3 Setup Guide Integration

```typescript
// When user clicks "Fix Setup" in DiagnosticModal
const handleFixSetup = (serviceName: string, requirements: DiagnosticRequirement[]) => {
  // Determine which step failed
  const failedStep = requirements.find(r => r.status === 'fail')
  
  let targetStep: SetupStep = 'verification'
  
  if (failedStep?.id === 'workspace') {
    targetStep = 'workspace'
  } else if (failedStep?.id === 'auth') {
    targetStep = 'auth'
  } else if (failedStep?.id === 'diagnostics') {
    targetStep = 'diagnostic-settings'
  }
  
  // Close diagnostic modal
  onClose()
  
  // Open setup guide to specific step
  onOpenSetupGuide?.(targetStep)
}
```

---

## 5. Responsive Design

### 5.1 Breakpoints

```typescript
const breakpoints = {
  sm: '640px',   // Mobile landscape
  md: '768px',   // Tablet
  lg: '1024px',  // Desktop
  xl: '1280px',  // Large desktop
}
```

### 5.2 DiagnosticModal Responsive Behavior

#### Mobile (< 640px)
- Full-screen modal (w-full h-full)
- Stack footer buttons vertically
- Reduce padding (px-4 instead of px-6)
- Smaller font sizes

#### Tablet (640px - 1024px)
- Modal width: 90vw, max-width: 700px
- Footer buttons wrap when needed
- Maintain padding

#### Desktop (> 1024px)
- Modal width: max-width: 900px
- Footer buttons inline
- Full padding

### 5.3 Responsive Grid

```typescript
<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
  {/* Service cards - 1 column on mobile, 2 on tablet+ */}
</div>
```

---

## 6. Animation Specifications

### 6.1 Modal Entrance/Exit

```typescript
// Entrance (200ms)
@keyframes modal-enter {
  from {
    opacity: 0;
    transform: translate(-50%, -48%) scale(0.96);
  }
  to {
    opacity: 1;
    transform: translate(-50%, -50%) scale(1);
  }
}

// Exit (150ms)
@keyframes modal-exit {
  from {
    opacity: 1;
    transform: translate(-50%, -50%) scale(1);
  }
  to {
    opacity: 0;
    transform: translate(-50%, -48%) scale(0.96);
  }
}
```

### 6.2 Expand/Collapse

```typescript
// Height transition for service card details
transition: 'height 200ms ease-out, opacity 150ms ease-out'

// Chevron rotation
transition: 'transform 200ms ease-out'
transform: expanded ? 'rotate(180deg)' : 'rotate(0deg)'
```

### 6.3 Status Changes

```typescript
// Pulse effect when status changes
@keyframes status-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

// Apply when status changes
className={cn(
  'transition-all duration-300',
  statusChanged && 'animate-pulse'
)}
```

---

## 7. Error Handling

### 7.1 API Errors

```typescript
try {
  const response = await fetch('/api/azure/diagnostics')
  
  if (!response.ok) {
    if (response.status === 401) {
      throw new Error('Authentication required. Run "azd auth login".')
    } else if (response.status === 503) {
      throw new Error('Diagnostic service temporarily unavailable. Try again in a moment.')
    } else {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`)
    }
  }
  
  const data = await response.json()
  setDiagnosticData(data)
  
} catch (err) {
  if (err instanceof Error && err.name === 'AbortError') {
    // Request was cancelled, ignore
    return
  }
  
  setError(err instanceof Error ? err.message : 'Failed to fetch diagnostics')
}
```

### 7.2 Network Timeout

```typescript
const controller = new AbortController()
const timeoutId = setTimeout(() => controller.abort(), 30000) // 30s

try {
  const response = await fetch('/api/azure/diagnostics', {
    signal: controller.signal
  })
  // ...
} finally {
  clearTimeout(timeoutId)
}
```

### 7.3 Graceful Degradation

When diagnostics fail to load:
- Show last known status if available
- Provide manual troubleshooting link
- Allow manual refresh
- Don't block log viewing

---

## 8. Testing Specifications

### 8.1 Unit Tests

```typescript
describe('DiagnosticModal', () => {
  it('renders loading state', () => {
    render(<DiagnosticModal isOpen={true} isLoading={true} />)
    expect(screen.getByText(/running health checks/i)).toBeInTheDocument()
  })
  
  it('renders error state', () => {
    render(<DiagnosticModal isOpen={true} error="Network timeout" />)
    expect(screen.getByText(/network timeout/i)).toBeInTheDocument()
  })
  
  it('renders service cards', () => {
    const data = createMockDiagnosticData()
    render(<DiagnosticModal isOpen={true} data={data} />)
    expect(screen.getByText('api')).toBeInTheDocument()
    expect(screen.getByText('web')).toBeInTheDocument()
  })
  
  it('expands service card on click', async () => {
    const data = createMockDiagnosticData()
    render(<DiagnosticModal isOpen={true} data={data} />)
    
    const cardHeader = screen.getByRole('button', { name: /api/i })
    await userEvent.click(cardHeader)
    
    expect(screen.getByText(/log analytics workspace/i)).toBeInTheDocument()
  })
  
  it('calls onOpenSetupGuide when Fix Setup clicked', async () => {
    const onOpenSetupGuide = vi.fn()
    const data = createMockDiagnosticDataWithErrors()
    
    render(
      <DiagnosticModal 
        isOpen={true} 
        data={data} 
        onOpenSetupGuide={onOpenSetupGuide}
      />
    )
    
    const fixButton = screen.getByRole('button', { name: /fix setup/i })
    await userEvent.click(fixButton)
    
    expect(onOpenSetupGuide).toHaveBeenCalledWith('diagnostic-settings')
  })
})
```

### 8.2 Accessibility Tests

```typescript
describe('DiagnosticModal Accessibility', () => {
  it('traps focus within modal', async () => {
    render(<DiagnosticModal isOpen={true} />)
    
    const closeButton = screen.getByRole('button', { name: /close/i })
    const refreshButton = screen.getByRole('button', { name: /refresh/i })
    
    // Tab forward
    closeButton.focus()
    await userEvent.tab()
    expect(refreshButton).toHaveFocus()
    
    // Tab backward
    await userEvent.tab({ shift: true })
    expect(closeButton).toHaveFocus()
  })
  
  it('closes on Escape key', async () => {
    const onClose = vi.fn()
    render(<DiagnosticModal isOpen={true} onClose={onClose} />)
    
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })
  
  it('has proper ARIA attributes', () => {
    render(<DiagnosticModal isOpen={true} />)
    
    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAttribute('aria-labelledby', 'diagnostics-title')
  })
})
```

### 8.3 Integration Tests

```typescript
describe('Diagnostic Flow Integration', () => {
  it('opens diagnostic modal from NoLogsPrompt', async () => {
    render(<LogsPane serviceName="api" logMode="azure" />)
    
    // No logs shown
    expect(screen.getByText(/no logs available/i)).toBeInTheDocument()
    
    // Click diagnostic link
    const diagnosticLink = screen.getByRole('button', { 
      name: /view diagnostic details/i 
    })
    await userEvent.click(diagnosticLink)
    
    // Modal opens
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText(/azure logs diagnostics/i)).toBeInTheDocument()
  })
  
  it('navigates to setup guide from diagnostic modal', async () => {
    const { onOpenSetupGuide } = renderDiagnosticFlow()
    
    // Open diagnostics
    await userEvent.click(screen.getByRole('button', { name: /diagnostics/i }))
    
    // Click Fix Setup
    await userEvent.click(screen.getByRole('button', { name: /fix setup/i }))
    
    // Setup guide opens to correct step
    expect(onOpenSetupGuide).toHaveBeenCalledWith('diagnostic-settings')
  })
})
```

---

## 9. Performance Considerations

### 9.1 Lazy Loading

```typescript
// Lazy load diagnostic modal (not rendered until opened)
const DiagnosticModal = lazy(() => import('./DiagnosticsModal'))

// Usage
{isOpen && (
  <Suspense fallback={<LoadingSpinner />}>
    <DiagnosticModal isOpen={isOpen} onClose={onClose} />
  </Suspense>
)}
```

### 9.2 Memoization

```typescript
// Memoize service cards
const ServiceCard = memo(({ service }: { service: ServiceDiagnostic }) => {
  // ...
})

// Memoize requirement items
const RequirementItem = memo(({ requirement }: { requirement: DiagnosticRequirement }) => {
  // ...
})
```

### 9.3 Debounced Refresh

```typescript
const debouncedRefresh = useMemo(
  () => debounce(refresh, 1000, { leading: true, trailing: false }),
  [refresh]
)
```

---

## 10. Future Enhancements

### 10.1 Real-time Updates

```typescript
// SSE for live diagnostic updates
useEffect(() => {
  if (!isOpen) return
  
  const eventSource = new EventSource('/api/azure/diagnostics/stream')
  
  eventSource.addEventListener('diagnostic-update', (event) => {
    const update = JSON.parse(event.data)
    setDiagnosticData(update)
  })
  
  return () => eventSource.close()
}, [isOpen])
```

### 10.2 Export Diagnostics

```typescript
const handleExportDiagnostics = () => {
  const report = generateMarkdownReport(diagnosticData)
  const blob = new Blob([report], { type: 'text/markdown' })
  const url = URL.createObjectURL(blob)
  
  const a = document.createElement('a')
  a.href = url
  a.download = `azure-diagnostics-${Date.now()}.md`
  a.click()
  
  URL.revokeObjectURL(url)
}
```

### 10.3 Historical Comparison

```typescript
interface DiagnosticHistory {
  timestamp: string
  status: 'healthy' | 'degraded' | 'error'
  serviceCount: number
  issues: number
}

// Show trend: "3 issues resolved since last check"
```

---

## 11. Implementation Checklist

### Phase 1: Core Components
- [ ] Create `DiagnosticModal.tsx` base structure
- [ ] Implement API integration (`/api/azure/diagnostics`)
- [ ] Add loading/error states
- [ ] Implement service card rendering
- [ ] Add expand/collapse functionality

### Phase 2: Interactions
- [ ] Implement refresh functionality
- [ ] Add copy report feature
- [ ] Add copy fix command feature
- [ ] Implement "Fix Setup" flow
- [ ] Add setup guide deep linking

### Phase 3: Empty States
- [ ] Create `NoLogsPrompt.tsx` component
- [ ] Integrate with `LogsPaneEmptyState.tsx`
- [ ] Add diagnostic link

### Phase 4: Header Integration
- [ ] Add diagnostic button to `LogsPaneHeader.tsx`
- [ ] Implement badge indicator
- [ ] Wire up modal open

### Phase 5: Accessibility
- [ ] Add ARIA labels and roles
- [ ] Implement keyboard navigation
- [ ] Add focus trap
- [ ] Test with screen readers
- [ ] Ensure color contrast

### Phase 6: Testing
- [ ] Write unit tests
- [ ] Write accessibility tests
- [ ] Write integration tests
- [ ] Manual testing across browsers

### Phase 7: Polish
- [ ] Add animations
- [ ] Responsive design tweaks
- [ ] Dark mode refinement
- [ ] Performance optimization

---

## Appendix A: Design Mockups

*Note: Wireframes provided in specification. High-fidelity mockups to be created during implementation.*

## Appendix B: Component File Structure

```
src/components/
├── diagnostics/
│   ├── DiagnosticsModal.tsx        # Main modal component
│   ├── DiagnosticsModal.test.tsx
│   ├── ServiceCard.tsx              # Individual service card
│   ├── ServiceCard.test.tsx
│   ├── RequirementItem.tsx          # Checklist item
│   ├── RequirementItem.test.tsx
│   ├── SetupGuidePanel.tsx          # Inline setup guide
│   └── NoLogsPrompt.tsx             # Empty state prompt
├── LogsPaneHeader.tsx               # Updated with diagnostic button
└── LogsPaneEmptyState.tsx           # Updated to use NoLogsPrompt
```

## Appendix C: Theme Tokens

```typescript
// tailwind.config.js additions
module.exports = {
  theme: {
    extend: {
      colors: {
        diagnostic: {
          healthy: 'rgb(16 185 129)',     // emerald-500
          degraded: 'rgb(245 158 11)',    // amber-500
          error: 'rgb(239 68 68)',        // red-500
        },
      },
      animation: {
        'modal-enter': 'modal-enter 200ms ease-out',
        'modal-exit': 'modal-exit 150ms ease-in',
        'status-pulse': 'status-pulse 600ms ease-in-out',
      },
    },
  },
}
```

---

**END OF SPECIFICATION**

This specification provides complete implementation guidance for the Azure logs diagnostic system UI. All components follow the existing dashboard patterns and meet WCAG 2.1 AA accessibility standards.
