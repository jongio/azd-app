# Service Detail Panel Component Specification

## Document Info
- **Component**: panels/ServiceDetailPanel
- **Status**: Design Specification
- **Created**: 2024-12-01
- **Author**: UX Design

---

## 1. Overview

The ServiceDetailPanel is a slide-in panel that displays comprehensive details about a selected service. It appears from the right side of the screen with a backdrop blur and provides tabbed navigation between different information categories.

### Use Cases
- **Quick Inspection**: View all details about a specific service
- **Debugging**: Access local development info (port, PID, status)
- **Azure Resources**: View and link to Azure deployment details
- **Environment**: Copy service-specific environment variables

---

## 2. Component Breakdown

### 2.1 Visual Structure

```
┌────────────────────────────────────────────────────────────────────────────────┐
│                                  MAIN APP                                      │
│                                                                                │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │ BACKDROP (blur, click to close)                                         │   │
│  │                                                                         │   │
│  │                                                      ┌────────────────┐ │   │
│  │                                                      │ SERVICE DETAIL │ │   │
│  │                                                      │    PANEL       │ │   │
│  │                                                      │                │ │   │
│  │                                                      │  ┌──────────┐  │ │   │
│  │                                                      │  │ Header   │  │ │   │
│  │                                                      │  │ + Close  │  │ │   │
│  │                                                      │  └──────────┘  │ │   │
│  │                                                      │                │ │   │
│  │                                                      │  ┌──────────┐  │ │   │
│  │                                                      │  │  Tabs    │  │ │   │
│  │                                                      │  └──────────┘  │ │   │
│  │                                                      │                │ │   │
│  │                                                      │  ┌──────────┐  │ │   │
│  │                                                      │  │ Content  │  │ │   │
│  │                                                      │  │          │  │ │   │
│  │                                                      │  │          │  │ │   │
│  │                                                      │  └──────────┘  │ │   │
│  │                                                      │                │ │   │
│  │                                                      └────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────────────────────────────┘

PANEL DIMENSIONS:
- Width: 500px (fixed)
- Height: 100vh
- Position: fixed, right: 0
- Z-index: 50
```

### 2.2 Panel Header

```
┌────────────────────────────────────────────────────────────────┐
│                                                                │
│  ┌────┐                                                   [X]  │
│  │ ●  │  api                                                   │
│  └────┘  Express • Running • Healthy                           │
│                                                                │
└────────────────────────────────────────────────────────────────┘

Elements:
- Status indicator (colored dot)
- Service name (h2)
- Subtitle: Framework • Status • Health
- Close button (X icon)
```

### 2.3 Tab Navigation

```
┌─────────────────────────────────────────────────────────────────┐
│  [ Overview ] [ Local ] [ Azure ] [ Environment ]               │
└─────────────────────────────────────────────────────────────────┘

Tab States:
- Active: text-foreground, bottom border
- Inactive: text-muted-foreground
- Hover: text-foreground/80
- Focus: ring-2 ring-primary
```

### 2.4 Sub-Components

| Component | Description | Required |
|-----------|-------------|----------|
| **ServiceDetailPanel** | Main container with backdrop | Yes |
| **PanelHeader** | Service name, status, close button | Yes |
| **TabList** | Tab navigation | Yes |
| **OverviewTab** | Summary of local + Azure info | Yes |
| **LocalTab** | Full local development details | Yes |
| **AzureTab** | Azure resource information | Yes |
| **EnvironmentTab** | Service env vars with copy | Yes |

---

## 3. Props and Interfaces

### 3.1 Core Types

```typescript
/** Props for the main ServiceDetailPanel component */
interface ServiceDetailPanelProps {
  /** Service to display details for */
  service: Service | null
  /** Whether the panel is open */
  isOpen: boolean
  /** Callback when panel should close */
  onClose: () => void
  /** Health check result for real-time status */
  healthStatus?: HealthCheckResult
  /** Additional class names */
  className?: string
  /** Data test ID for testing */
  'data-testid'?: string
}

/** Tab identifier */
type DetailTab = 'overview' | 'local' | 'azure' | 'environment'
```

---

## 4. Tab Content Specifications

### 4.1 Overview Tab

```
┌─────────────────────────────────────────────────────────────────┐
│  LOCAL DEVELOPMENT                                              │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Status          Running                                 │    │
│  │ Health          Healthy                                 │    │
│  │ URL             http://localhost:3100 →                 │    │
│  │ Port            3100                                    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  AZURE DEPLOYMENT                                               │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Status          Deployed                                │    │
│  │ Resource        api-containerapp                        │    │
│  │ Endpoint        https://api.azurecontainers.io →        │    │
│  │                                                         │    │
│  │ [Open in Azure Portal]                                  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  (If no Azure deployment)                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Not deployed to Azure                                   │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 Local Tab

```
┌─────────────────────────────────────────────────────────────────┐
│  SERVICE DETAILS                                                │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Name            api                              [📋]   │    │
│  │ Language        TypeScript                       [📋]   │    │
│  │ Framework       Express                          [📋]   │    │
│  │ Project         ./services/api                   [📋]   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  RUNTIME                                                        │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Status          ● Running                               │    │
│  │ Health          ● Healthy                               │    │
│  │ PID             12345                            [📋]   │    │
│  │ Port            3100                             [📋]   │    │
│  │ URL             http://localhost:3100 →          [📋]   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  TIMING                                                         │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Started         2024-12-01 10:30:45              [📋]   │    │
│  │ Uptime          2h 15m 30s                              │    │
│  │ Last Checked    2024-12-01 12:45:00              [📋]   │    │
│  │ Response Time   45ms                                    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  HEALTH DETAILS                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Check Type      HTTP                                    │    │
│  │ Endpoint        /health                          [📋]   │    │
│  │ Status Code     200                                     │    │
│  │ Failures        0                                       │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### 4.3 Azure Tab

```
┌─────────────────────────────────────────────────────────────────┐
│  RESOURCE                                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Resource Name   api-containerapp                 [📋]   │    │
│  │ Resource Type   Container App                           │    │
│  │ Image           myregistry.azurecr.io/api:latest [📋]  │    │
│  │ Endpoint        https://api.azurecontainers.io → [📋]  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  AZURE METADATA                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Subscription    xxxxxxxx-xxxx-xxxx-xxxx-xxxx     [📋]   │    │
│  │ Resource Group  rg-myapp-prod                    [📋]   │    │
│  │ Location        East US                                 │    │
│  │ Environment ID  /subscriptions/.../containerApp  [📋]   │    │
│  │ Log Analytics   /subscriptions/.../workspace     [📋]   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ACTIONS                                                        │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ [Open in Azure Portal]  [View Logs]  [View Metrics]     │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  (If not deployed)                                              │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ ⚠ Not Deployed to Azure                                 │    │
│  │                                                         │    │
│  │ This service hasn't been deployed to Azure yet.         │    │
│  │ Run `azd deploy` to deploy your services.               │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### 4.4 Environment Tab

```
┌─────────────────────────────────────────────────────────────────┐
│  ENVIRONMENT VARIABLES (4)                         [Show All]   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ NODE_ENV         production                      [📋]   │    │
│  │ API_KEY          •••••••••••••                  [👁][📋]│    │
│  │ DATABASE_URL     postgres://localhost/db         [📋]   │    │
│  │ PORT             3100                            [📋]   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  (If no environment variables)                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ No environment variables configured                     │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘

Legend:
[📋] = Copy to clipboard button
[👁] = Show/hide sensitive value toggle
```

---

## 5. Interactions

### 5.1 Panel Interactions

| Action | Result |
|--------|--------|
| Open (service click) | Panel slides in from right with backdrop |
| Close (X button) | Panel slides out, backdrop fades |
| Close (Escape key) | Same as X button (use `useEscapeKey` hook) |
| Close (Backdrop click) | Same as X button |
| Tab click | Switch to selected tab content |

### 5.2 Content Interactions

| Action | Result |
|--------|--------|
| Copy button click | Copy value, show feedback via `useClipboard` |
| URL click | Open in new tab |
| Azure Portal click | Open resource in Azure Portal |
| Show/Hide toggle | Reveal/mask sensitive values |

### 5.3 Keyboard Navigation

| Key | Action |
|-----|--------|
| `Escape` | Close panel |
| `Tab` | Navigate between interactive elements |
| `Arrow Left/Right` | Navigate between tabs (when focused) |
| `Enter/Space` | Activate focused element |

---

## 6. Animation Specifications

### 6.1 Panel Enter Animation

```css
/* Panel slides in from right */
@keyframes slideIn {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

.panel-enter {
  animation: slideIn 200ms ease-out;
}
```

### 6.2 Panel Exit Animation

```css
@keyframes slideOut {
  from {
    transform: translateX(0);
    opacity: 1;
  }
  to {
    transform: translateX(100%);
    opacity: 0;
  }
}

.panel-exit {
  animation: slideOut 150ms ease-in;
}
```

### 6.3 Backdrop Animation

```css
/* Backdrop fades in */
@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.backdrop-enter {
  animation: fadeIn 200ms ease-out;
  backdrop-filter: blur(4px);
}
```

---

## 7. Accessibility

### 7.1 WCAG 2.1 AA Compliance

| Criterion | Implementation |
|-----------|----------------|
| **1.3.1 Info & Relationships** | Tabs use proper role="tablist" |
| **1.4.3 Contrast (Minimum)** | All text meets 4.5:1 ratio |
| **2.1.1 Keyboard** | All interactive elements keyboard accessible |
| **2.1.2 No Keyboard Trap** | Escape key closes panel |
| **2.4.3 Focus Order** | Focus trapped within panel when open |
| **4.1.2 Name, Role, Value** | Proper ARIA labels and roles |

### 7.2 ARIA Implementation

```tsx
// Panel container
<div
  role="dialog"
  aria-modal="true"
  aria-labelledby="panel-title"
  aria-describedby="panel-description"
>
  <h2 id="panel-title">{service.name}</h2>
  
  // Tab list
  <div role="tablist" aria-label="Service details tabs">
    <button
      role="tab"
      aria-selected={activeTab === 'overview'}
      aria-controls="panel-overview"
      id="tab-overview"
    >
      Overview
    </button>
    // ... more tabs
  </div>
  
  // Tab panel
  <div
    role="tabpanel"
    id="panel-overview"
    aria-labelledby="tab-overview"
    tabIndex={0}
  >
    // Content
  </div>
</div>
```

### 7.3 Focus Management

```typescript
// Focus trap implementation
useEffect(() => {
  if (isOpen) {
    // Save previously focused element
    const previousFocus = document.activeElement as HTMLElement
    
    // Focus first focusable element in panel
    panelRef.current?.querySelector<HTMLElement>('[tabindex="0"]')?.focus()
    
    // Restore focus on close
    return () => previousFocus?.focus()
  }
}, [isOpen])
```

---

## 8. Design Tokens

### 8.1 Panel Dimensions

| Property | Value | Token |
|----------|-------|-------|
| Panel width | `500px` | Fixed |
| Panel height | `100vh` | Full viewport |
| Header height | `80px` | Fixed |
| Tab bar height | `48px` | Fixed |
| Content padding | `p-6` | 24px |

### 8.2 Typography

| Element | Font Size | Font Weight | Color |
|---------|-----------|-------------|-------|
| Service name | `text-xl` (20px) | `font-semibold` (600) | `text-foreground` |
| Subtitle | `text-sm` (14px) | `font-normal` (400) | `text-muted-foreground` |
| Section heading | `text-sm` (14px) | `font-medium` (500) | `text-muted-foreground` |
| Field label | `text-sm` (14px) | `font-normal` (400) | `text-muted-foreground` |
| Field value | `text-sm` (14px) | `font-medium` (500) | `text-foreground` |

### 8.3 Colors

| Element | Light Mode | Dark Mode |
|---------|------------|-----------|
| Panel background | `bg-background` | `bg-background` |
| Backdrop | `bg-black/50` | `bg-black/50` |
| Section card | `bg-card` | `bg-card` |
| Tab active | `border-primary` | `border-primary` |
| Close button hover | `bg-secondary` | `bg-secondary` |

### 8.4 Z-Index

| Element | Z-Index |
|---------|---------|
| Backdrop | `z-40` |
| Panel | `z-50` |

---

## 9. Responsive Behavior

### 9.1 Mobile (< 640px)

```
- Panel width: 100vw (full width)
- Tabs: horizontal scroll if needed
- Content: single column layout
```

### 9.2 Tablet (640-1023px)

```
- Panel width: 450px
- Same tab layout
- Same content layout
```

### 9.3 Desktop (≥ 1024px)

```
- Panel width: 500px
- Full tab display
- Full content layout
```

---

## 10. Implementation Reference

```tsx
import * as React from 'react'
import { X, ExternalLink } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useEscapeKey } from '@/hooks/useEscapeKey'
import { useClipboard } from '@/hooks/useClipboard'
import { InfoField } from '@/components/ui/InfoField'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import type { Service, HealthCheckResult } from '@/types'

// Helper: Format uptime from timestamp
export function formatUptime(startTime?: string): string {
  if (!startTime) return 'N/A'
  const start = new Date(startTime)
  const now = new Date()
  const diff = now.getTime() - start.getTime()
  
  const hours = Math.floor(diff / (1000 * 60 * 60))
  const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))
  const seconds = Math.floor((diff % (1000 * 60)) / 1000)
  
  if (hours > 0) return `${hours}h ${minutes}m ${seconds}s`
  if (minutes > 0) return `${minutes}m ${seconds}s`
  return `${seconds}s`
}

// Helper: Get status color
export function getStatusColor(status?: string): string {
  const colors: Record<string, string> = {
    running: 'text-green-500',
    ready: 'text-green-500',
    starting: 'text-yellow-500',
    stopping: 'text-yellow-500',
    stopped: 'text-gray-500',
    error: 'text-red-500',
    'not-running': 'text-gray-500',
  }
  return colors[status || 'not-running'] || 'text-gray-500'
}

// Helper: Get health color
export function getHealthColor(health?: string): string {
  const colors: Record<string, string> = {
    healthy: 'text-green-500',
    degraded: 'text-yellow-500',
    unhealthy: 'text-red-500',
    unknown: 'text-gray-500',
  }
  return colors[health || 'unknown'] || 'text-gray-500'
}

// Helper: Build Azure Portal URL
export function buildAzurePortalUrl(service: Service): string | null {
  const azure = service.azure
  if (!azure?.subscriptionId || !azure?.resourceGroup || !azure?.resourceName) {
    return null
  }
  return `https://portal.azure.com/#@/resource/subscriptions/${azure.subscriptionId}/resourceGroups/${azure.resourceGroup}/providers/Microsoft.App/containerApps/${azure.resourceName}`
}

// Helper: Check if value is sensitive
export function isSensitiveKey(key: string): boolean {
  const sensitivePatterns = [
    /password/i,
    /secret/i,
    /key/i,
    /token/i,
    /api[-_]?key/i,
    /auth/i,
    /credential/i,
    /private/i,
  ]
  return sensitivePatterns.some(pattern => pattern.test(key))
}
```

---

## 11. Testing Checklist

### 11.1 Unit Tests

**Panel Behavior**
- [ ] Renders when isOpen is true
- [ ] Does not render when isOpen is false
- [ ] Calls onClose when X button clicked
- [ ] Calls onClose when backdrop clicked
- [ ] Calls onClose when Escape key pressed
- [ ] Shows service name in header
- [ ] Shows status and health in header

**Tab Navigation**
- [ ] Renders all four tabs
- [ ] Defaults to Overview tab
- [ ] Switches content on tab click
- [ ] Tab has proper ARIA attributes
- [ ] Keyboard navigation works

**Overview Tab**
- [ ] Shows local development section
- [ ] Shows Azure deployment section (if deployed)
- [ ] Shows "Not deployed" message (if not deployed)

**Local Tab**
- [ ] Shows all service details
- [ ] Shows runtime information
- [ ] Shows timing information
- [ ] Shows health details (if available)
- [ ] Copy buttons work

**Azure Tab**
- [ ] Shows resource information (if deployed)
- [ ] Shows metadata (subscription, group, etc.)
- [ ] Shows action buttons
- [ ] "Not deployed" empty state

**Environment Tab**
- [ ] Shows all environment variables
- [ ] Copy buttons work
- [ ] Sensitive values masked by default
- [ ] Show/hide toggle works
- [ ] Empty state when no vars

### 11.2 Accessibility Tests

- [ ] Focus trapped when open
- [ ] Focus restored on close
- [ ] Screen reader announces dialog
- [ ] Tabs properly labeled
- [ ] All buttons have accessible names

### 11.3 Animation Tests

- [ ] Panel slides in smoothly
- [ ] Panel slides out smoothly
- [ ] Backdrop fades in/out
- [ ] No animation flicker

---

## 12. Related Components

| Component | Relationship |
|-----------|--------------|
| **ServiceCard** | Triggers panel open |
| **ServiceTable** | Triggers panel open |
| **InfoField** | Used for copyable fields |
| **Tabs** | UI component for tab navigation |
| **useEscapeKey** | Hook for escape key handling |
| **useClipboard** | Hook for copy functionality |

---

## Appendix A: Token Quick Reference

```css
/* ServiceDetailPanel Tokens */
.service-detail-panel {
  width: 500px;
  height: 100vh;
  position: fixed;
  right: 0;
  top: 0;
  z-index: 50;
  background: var(--background);
  border-left: 1px solid var(--border);
}

.panel-backdrop {
  position: fixed;
  inset: 0;
  z-index: 40;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(4px);
}

.panel-header {
  padding: theme('spacing.6');
  border-bottom: 1px solid var(--border);
  height: 80px;
}

.panel-tabs {
  padding: 0 theme('spacing.6');
  border-bottom: 1px solid var(--border);
  height: 48px;
}

.panel-content {
  padding: theme('spacing.6');
  overflow-y: auto;
  height: calc(100vh - 128px); /* header + tabs */
}

.section-card {
  padding: theme('spacing.4');
  border-radius: theme('borderRadius.lg');
  background: var(--card);
  border: 1px solid var(--border);
  margin-bottom: theme('spacing.4');
}

.section-title {
  font-size: theme('fontSize.sm');
  font-weight: theme('fontWeight.medium');
  color: var(--muted-foreground);
  margin-bottom: theme('spacing.3');
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
```
