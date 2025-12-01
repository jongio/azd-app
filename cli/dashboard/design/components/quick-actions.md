# QuickActions Component Specification

## Document Info
- **Component**: views/QuickActions
- **Status**: Design Specification
- **Created**: 2024-12-01
- **Author**: UX Design

---

## 1. Overview

The QuickActions component provides a dashboard view with service statistics cards and global action buttons. It gives users a quick overview of service health status and provides one-click access to common operations.

### Use Cases
- **Quick Status Check**: At-a-glance view of how many services are running, healthy, or in error state
- **Bulk Operations**: Refresh all services, clear all logs, export logs in one click
- **Developer Workflow**: Quick access to terminal for debugging

---

## 2. Component Breakdown

### 2.1 Visual Structure

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│  STATS SECTION                                                                  │
│  ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐                │
│  │  🔵 Running      │ │  💚 Healthy      │ │  🔴 Errors       │                │
│  │                  │ │                  │ │                  │                │
│  │      3           │ │      2           │ │      1           │                │
│  │   services       │ │   services       │ │   service        │                │
│  └──────────────────┘ └──────────────────┘ └──────────────────┘                │
│                                                                                 │
│  ACTIONS SECTION                                                                │
│  ┌─────────────────────────────────────────────────────────────────────────────┐│
│  │  Global Actions                                                             ││
│  │                                                                             ││
│  │  [🔄 Refresh All]  [🗑 Clear Logs]  [📥 Export Logs]  [💻 Open Terminal]  ││
│  │                                                                             ││
│  └─────────────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Sub-Components

| Component | Description | Required |
|-----------|-------------|----------|
| **QuickActions** | Main container component | Yes |
| **StatsSection** | Grid container for stat cards | Yes |
| **StatCard** | Individual statistic card | Yes |
| **ActionsSection** | Container for action buttons | Yes |
| **ActionButton** | Individual action button | Yes |

---

## 3. Props and Interfaces

### 3.1 Core Types

```typescript
/** Props for the main QuickActions component */
interface QuickActionsProps {
  /** Services data for computing stats */
  services: Service[]
  /** Callback to refresh all services */
  onRefresh?: () => void
  /** Additional class names */
  className?: string
  /** Data test ID for testing */
  'data-testid'?: string
}

/** Props for StatCard */
interface StatCardProps {
  /** Card title */
  title: string
  /** Statistic value */
  value: number
  /** Total for comparison (optional) */
  total?: number
  /** Card icon */
  icon: React.ComponentType<{ className?: string }>
  /** Card color variant */
  variant: 'primary' | 'success' | 'warning' | 'error'
  /** Subtitle text (e.g., "services") */
  subtitle?: string
}

/** Props for ActionButton */
interface ActionButtonProps {
  /** Button label */
  label: string
  /** Button icon */
  icon: React.ComponentType<{ className?: string }>
  /** Click handler */
  onClick: () => void
  /** Whether the action is in progress */
  loading?: boolean
  /** Whether the button is disabled */
  disabled?: boolean
  /** Button variant */
  variant?: 'default' | 'outline' | 'ghost'
}
```

---

## 4. Statistics Cards

### 4.1 Running Services Card

**Data Source:**
```typescript
const runningCount = services.filter(s => 
  s.local?.status === 'running' || s.local?.status === 'ready'
).length
```

**Display:**
- Icon: `Activity` (lucide)
- Color: Primary blue
- Title: "Running"
- Value: Count of running/ready services
- Subtitle: "services" (or "service" if count === 1)

### 4.2 Healthy Services Card

**Data Source:**
```typescript
const healthyCount = services.filter(s => 
  s.local?.health === 'healthy'
).length
```

**Display:**
- Icon: `CheckCircle` (lucide)
- Color: Success green
- Title: "Healthy"
- Value: Count of healthy services
- Subtitle: "services" (or "service" if count === 1)

### 4.3 Error Services Card

**Data Source:**
```typescript
const errorCount = services.filter(s => 
  s.local?.status === 'error' || s.local?.health === 'unhealthy'
).length
```

**Display:**
- Icon: `AlertCircle` (lucide)
- Color: Error red
- Title: "Errors"
- Value: Count of error/unhealthy services
- Subtitle: "services" (or "service" if count === 1)
- Animation: Pulse when count > 0

---

## 5. StatCard Visual States

### 5.1 State Matrix

| State | Border | Background | Icon | Value |
|-------|--------|------------|------|-------|
| **Primary** | `border-primary/20` | `bg-primary/5` | `text-primary` | `text-foreground` |
| **Success** | `border-success/20` | `bg-success/5` | `text-success` | `text-foreground` |
| **Warning** | `border-warning/20` | `bg-warning/5` | `text-warning` | `text-foreground` |
| **Error** | `border-destructive/20` | `bg-destructive/5` | `text-destructive` | `text-foreground` |
| **Error+Active** | `border-destructive/30` | `bg-destructive/10` | `text-destructive animate-pulse` | `text-foreground` |

### 5.2 Visual Layout

```
STAT CARD:
┌─────────────────────────────────────┐
│  ┌─────┐                            │
│  │ 🔵 │  Running                    │   Icon + Title row
│  └─────┘                            │
│                                     │
│         3                           │   Large value (centered)
│                                     │
│      services                       │   Subtitle (muted)
└─────────────────────────────────────┘

Dimensions:
- Min width: 160px
- Padding: p-6
- Border radius: rounded-lg
- Icon size: h-8 w-8
- Value font: text-3xl font-bold
- Title font: text-sm font-medium
```

---

## 6. Action Buttons

### 6.1 Refresh All

**Behavior:**
- Triggers `onRefresh` callback
- Shows loading spinner while refreshing
- Disabled during refresh

**Display:**
- Icon: `RefreshCw`
- Label: "Refresh All"
- Animation: Icon rotates while loading

### 6.2 Clear Logs

**Behavior:**
- Clears all log buffers (triggers event to LogsMultiPaneView)
- Shows brief success feedback

**Display:**
- Icon: `Trash2`
- Label: "Clear Logs"

### 6.3 Export Logs

**Behavior:**
- Downloads all logs as file
- Triggers browser download

**Display:**
- Icon: `Download`
- Label: "Export Logs"

### 6.4 Open Terminal

**Behavior:**
- Opens system terminal (platform-dependent)
- Windows: cmd/PowerShell
- macOS/Linux: default terminal

**Display:**
- Icon: `Terminal`
- Label: "Open Terminal"

---

## 7. Action Button States

### 7.1 State Matrix

| State | Background | Text | Border | Cursor |
|-------|------------|------|--------|--------|
| **Default** | `bg-secondary` | `text-foreground` | none | pointer |
| **Hover** | `bg-secondary/80` | `text-foreground` | none | pointer |
| **Active** | `bg-secondary/60` | `text-foreground` | none | pointer |
| **Focus** | `bg-secondary` | `text-foreground` | `ring-2 ring-ring` | pointer |
| **Loading** | `bg-secondary/50` | `text-muted-foreground` | none | wait |
| **Disabled** | `bg-secondary/30` | `text-muted-foreground` | none | not-allowed |

### 7.2 Visual Layout

```
ACTION BUTTON:
┌──────────────────────────────────────┐
│  [🔄]  Refresh All                   │
└──────────────────────────────────────┘

LOADING STATE:
┌──────────────────────────────────────┐
│  [⟳]  Refreshing...                  │   Icon animates, text changes
└──────────────────────────────────────┘

Dimensions:
- Padding: px-4 py-2
- Gap between icon and text: gap-2
- Icon size: h-4 w-4
- Font: text-sm font-medium
```

---

## 8. Interactions

### 8.1 Stat Card Interactions

| Action | Result |
|--------|--------|
| Hover | Subtle background opacity increase |
| Click (optional) | Navigate to filtered view (e.g., clicking Errors navigates to Console filtered to error services) |

### 8.2 Action Button Interactions

| Action | Target | Result |
|--------|--------|--------|
| Click Refresh All | Button | Triggers refresh, shows loading state |
| Click Clear Logs | Button | Clears all logs, shows success feedback |
| Click Export Logs | Button | Downloads log file |
| Click Open Terminal | Button | Opens system terminal |
| Keyboard Enter/Space | Focused button | Triggers action |

---

## 9. Accessibility

### 9.1 WCAG 2.1 AA Compliance

| Criterion | Implementation |
|-----------|----------------|
| **1.3.1 Info & Relationships** | Semantic HTML structure, proper heading hierarchy |
| **1.4.3 Contrast (Minimum)** | All text and icons meet 4.5:1 ratio |
| **2.1.1 Keyboard** | All buttons keyboard accessible |
| **2.4.6 Headings and Labels** | Clear section headings, descriptive button labels |
| **2.4.7 Focus Visible** | Visible focus ring on all interactive elements |
| **4.1.2 Name, Role, Value** | Proper ARIA labels on all controls |

### 9.2 ARIA Implementation

```tsx
// Main container
<section
  aria-labelledby="quick-actions-title"
  className="quick-actions"
>
  <h2 id="quick-actions-title" className="sr-only">
    Quick Actions Dashboard
  </h2>
  
  // Stats Section
  <section aria-labelledby="stats-title">
    <h3 id="stats-title" className="text-lg font-semibold">
      Service Statistics
    </h3>
    
    // Stat cards use role="status" for live regions
    <div 
      role="status" 
      aria-live="polite"
      aria-label="3 running services"
    >
      {/* StatCard content */}
    </div>
  </section>
  
  // Actions Section
  <section aria-labelledby="actions-title">
    <h3 id="actions-title" className="text-lg font-semibold">
      Global Actions
    </h3>
    
    <div role="group" aria-label="Action buttons">
      <button aria-label="Refresh all services">
        <RefreshCw /> Refresh All
      </button>
      {/* ... more buttons */}
    </div>
  </section>
</section>
```

### 9.3 Keyboard Navigation

| Key | Action |
|-----|--------|
| `Tab` | Move focus between interactive elements |
| `Enter` | Activate focused button |
| `Space` | Activate focused button |

---

## 10. Design Tokens

### 10.1 Colors

| Token | Light Theme | Dark Theme | Usage |
|-------|-------------|------------|-------|
| `--primary` | `#7c3aed` | `#a78bfa` | Running stat card |
| `--success` | `#16a34a` | `#22c55e` | Healthy stat card |
| `--warning` | `#ea580c` | `#f97316` | Warning states |
| `--destructive` | `#dc2626` | `#ef4444` | Error stat card |
| `--secondary` | `#f1f5f9` | `#1e293b` | Action button background |
| `--foreground` | `#0f172a` | `#f8fafc` | Primary text |
| `--muted-foreground` | `#64748b` | `#94a3b8` | Subtitle text |

### 10.2 Typography

| Element | Font Size | Font Weight | Line Height |
|---------|-----------|-------------|-------------|
| Section title | `text-lg` (18px) | `font-semibold` (600) | `leading-normal` |
| Card title | `text-sm` (14px) | `font-medium` (500) | `leading-normal` |
| Card value | `text-3xl` (30px) | `font-bold` (700) | `leading-tight` |
| Card subtitle | `text-sm` (14px) | `font-normal` (400) | `leading-normal` |
| Button label | `text-sm` (14px) | `font-medium` (500) | `leading-normal` |

### 10.3 Spacing

| Property | Value | Token |
|----------|-------|-------|
| Section padding | `p-6` | 24px |
| Stats grid gap | `gap-4` | 16px |
| Card padding | `p-6` | 24px |
| Button padding | `px-4 py-2` | 16px × 8px |
| Icon-to-text gap | `gap-2` | 8px |
| Section gap | `gap-6` | 24px |

---

## 11. Responsive Behavior

### 11.1 Breakpoint Adaptations

```
DESKTOP (≥1024px):
┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐
│ Running: 3       │ │ Healthy: 2       │ │ Errors: 1        │
└──────────────────┘ └──────────────────┘ └──────────────────┘

[Refresh All]  [Clear Logs]  [Export Logs]  [Open Terminal]


TABLET (640-1023px):
┌──────────────────┐ ┌──────────────────┐
│ Running: 3       │ │ Healthy: 2       │
└──────────────────┘ └──────────────────┘
┌──────────────────┐
│ Errors: 1        │
└──────────────────┘

[Refresh All]  [Clear Logs]
[Export Logs]  [Open Terminal]


MOBILE (<640px):
┌────────────────────────────────────┐
│ Running: 3                         │
└────────────────────────────────────┘
┌────────────────────────────────────┐
│ Healthy: 2                         │
└────────────────────────────────────┘
┌────────────────────────────────────┐
│ Errors: 1                          │
└────────────────────────────────────┘

[Refresh All]
[Clear Logs]
[Export Logs]
[Open Terminal]
```

### 11.2 Grid Configuration

```typescript
// Stats grid
const statsGridClass = cn(
  'grid gap-4',
  'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3'
)

// Actions grid
const actionsGridClass = cn(
  'flex flex-wrap gap-3',
  'flex-col sm:flex-row'
)
```

---

## 12. Integration with App.tsx

### 12.1 Sidebar Update

```tsx
// In Sidebar.tsx - add to navItems array
import { Zap } from 'lucide-react'

const navItems = [
  { id: 'resources', label: 'Resources', icon: Activity },
  { id: 'console', label: 'Console', icon: Terminal },
  { id: 'environment', label: 'Environment', icon: Settings2 },
  { id: 'actions', label: 'Actions', icon: Zap },  // NEW
]
```

### 12.2 App.tsx View Rendering

```tsx
// In App.tsx - add to renderContent function
if (activeView === 'actions') {
  return (
    <>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-foreground">Actions</h2>
      </div>
      <QuickActions 
        services={services} 
        onRefresh={() => {
          // Trigger service refresh
        }}
      />
    </>
  )
}
```

---

## 13. Implementation Reference

```tsx
import * as React from 'react'
import { 
  Activity, 
  CheckCircle, 
  AlertCircle, 
  RefreshCw, 
  Trash2, 
  Download, 
  Terminal 
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import type { Service } from '@/types'

interface QuickActionsProps {
  services: Service[]
  onRefresh?: () => void
  className?: string
  'data-testid'?: string
}

interface StatCardProps {
  title: string
  value: number
  icon: React.ComponentType<{ className?: string }>
  variant: 'primary' | 'success' | 'error'
  subtitle?: string
}

function StatCard({ title, value, icon: Icon, variant, subtitle }: StatCardProps) {
  const variantStyles = {
    primary: 'border-primary/20 bg-primary/5',
    success: 'border-success/20 bg-success/5',
    error: cn(
      'border-destructive/20 bg-destructive/5',
      value > 0 && 'border-destructive/30 bg-destructive/10'
    ),
  }
  
  const iconStyles = {
    primary: 'text-primary',
    success: 'text-success',
    error: cn('text-destructive', value > 0 && 'animate-pulse'),
  }
  
  return (
    <div
      className={cn(
        'rounded-lg border p-6',
        variantStyles[variant]
      )}
      role="status"
      aria-label={`${value} ${title.toLowerCase()} ${value === 1 ? 'service' : 'services'}`}
    >
      <div className="flex items-center gap-3 mb-4">
        <div className={cn('p-2 rounded-lg', variantStyles[variant])}>
          <Icon className={cn('h-6 w-6', iconStyles[variant])} aria-hidden="true" />
        </div>
        <span className="text-sm font-medium text-muted-foreground">{title}</span>
      </div>
      <div className="text-3xl font-bold text-foreground">{value}</div>
      {subtitle && (
        <div className="text-sm text-muted-foreground mt-1">{subtitle}</div>
      )}
    </div>
  )
}

export function QuickActions({
  services,
  onRefresh,
  className,
  'data-testid': testId,
}: QuickActionsProps) {
  const [isRefreshing, setIsRefreshing] = React.useState(false)
  
  // Compute statistics
  const runningCount = services.filter(s => 
    s.local?.status === 'running' || s.local?.status === 'ready'
  ).length
  
  const healthyCount = services.filter(s => 
    s.local?.health === 'healthy'
  ).length
  
  const errorCount = services.filter(s => 
    s.local?.status === 'error' || s.local?.health === 'unhealthy'
  ).length
  
  const handleRefresh = async () => {
    setIsRefreshing(true)
    try {
      onRefresh?.()
    } finally {
      // Show loading state briefly
      setTimeout(() => setIsRefreshing(false), 500)
    }
  }
  
  const handleClearLogs = () => {
    // Dispatch event for LogsMultiPaneView to handle
    window.dispatchEvent(new CustomEvent('clear-all-logs'))
  }
  
  const handleExportLogs = () => {
    // Dispatch event for LogsMultiPaneView to handle
    window.dispatchEvent(new CustomEvent('export-all-logs'))
  }
  
  const handleOpenTerminal = () => {
    // This would need backend support
    fetch('/api/terminal/open', { method: 'POST' }).catch(console.error)
  }
  
  return (
    <section
      aria-labelledby="quick-actions-title"
      className={cn('space-y-6', className)}
      data-testid={testId}
    >
      <h2 id="quick-actions-title" className="sr-only">
        Quick Actions Dashboard
      </h2>
      
      {/* Stats Section */}
      <section aria-labelledby="stats-title">
        <h3 id="stats-title" className="text-lg font-semibold text-foreground mb-4">
          Service Statistics
        </h3>
        <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
          <StatCard
            title="Running"
            value={runningCount}
            icon={Activity}
            variant="primary"
            subtitle={runningCount === 1 ? 'service' : 'services'}
          />
          <StatCard
            title="Healthy"
            value={healthyCount}
            icon={CheckCircle}
            variant="success"
            subtitle={healthyCount === 1 ? 'service' : 'services'}
          />
          <StatCard
            title="Errors"
            value={errorCount}
            icon={AlertCircle}
            variant="error"
            subtitle={errorCount === 1 ? 'service' : 'services'}
          />
        </div>
      </section>
      
      {/* Actions Section */}
      <section aria-labelledby="actions-title">
        <h3 id="actions-title" className="text-lg font-semibold text-foreground mb-4">
          Global Actions
        </h3>
        <div 
          role="group" 
          aria-label="Action buttons"
          className="flex flex-wrap gap-3"
        >
          <Button
            variant="secondary"
            onClick={handleRefresh}
            disabled={isRefreshing}
            className="gap-2"
          >
            <RefreshCw 
              className={cn('h-4 w-4', isRefreshing && 'animate-spin')} 
              aria-hidden="true" 
            />
            {isRefreshing ? 'Refreshing...' : 'Refresh All'}
          </Button>
          
          <Button
            variant="secondary"
            onClick={handleClearLogs}
            className="gap-2"
          >
            <Trash2 className="h-4 w-4" aria-hidden="true" />
            Clear Logs
          </Button>
          
          <Button
            variant="secondary"
            onClick={handleExportLogs}
            className="gap-2"
          >
            <Download className="h-4 w-4" aria-hidden="true" />
            Export Logs
          </Button>
          
          <Button
            variant="secondary"
            onClick={handleOpenTerminal}
            className="gap-2"
          >
            <Terminal className="h-4 w-4" aria-hidden="true" />
            Open Terminal
          </Button>
        </div>
      </section>
    </section>
  )
}
```

---

## 14. Testing Checklist

### 14.1 Unit Tests

**QuickActions**
- [ ] Renders all three stat cards
- [ ] Computes running count correctly
- [ ] Computes healthy count correctly
- [ ] Computes error count correctly
- [ ] Handles empty services array
- [ ] Renders all action buttons

**StatCard**
- [ ] Renders title, value, icon
- [ ] Applies correct variant styles
- [ ] Shows singular "service" when count is 1
- [ ] Shows plural "services" when count > 1
- [ ] Applies pulse animation for error variant with count > 0

**Action Buttons**
- [ ] Refresh All triggers onRefresh callback
- [ ] Refresh All shows loading state
- [ ] Clear Logs dispatches clear-all-logs event
- [ ] Export Logs dispatches export-all-logs event
- [ ] Open Terminal calls API endpoint

### 14.2 Accessibility Tests

- [ ] All cards have aria-label with count
- [ ] Buttons have descriptive labels
- [ ] Focus indicators visible
- [ ] Keyboard navigation works
- [ ] Screen reader announces stat updates

### 14.3 Integration Tests

- [ ] Sidebar navigation to Actions view works
- [ ] Actions view renders in App.tsx
- [ ] Services data flows correctly
- [ ] Real-time service updates reflected in stats

---

## 15. Related Components

| Component | Relationship |
|-----------|--------------|
| **Button** | Uses Button for action buttons |
| **Sidebar** | Adds navigation entry |
| **ServiceStatusCard** | Similar stat display pattern |
| **LogsMultiPaneView** | Receives clear/export events |

---

## Appendix A: Token Quick Reference

```css
/* QuickActions Tokens */
.quick-actions {
  gap: theme('spacing.6');               /* 24px */
}

.stat-card {
  padding: theme('spacing.6');           /* 24px */
  border-radius: theme('borderRadius.lg'); /* 8px */
}

.stat-card-value {
  font-size: theme('fontSize.3xl');      /* 30px */
  font-weight: theme('fontWeight.bold'); /* 700 */
}

.stat-card-title {
  font-size: theme('fontSize.sm');       /* 14px */
  font-weight: theme('fontWeight.medium'); /* 500 */
  color: var(--muted-foreground);
}

.stat-card-icon {
  height: theme('spacing.6');            /* 24px */
  width: theme('spacing.6');             /* 24px */
}

.action-button {
  padding: theme('spacing.2') theme('spacing.4'); /* 8px 16px */
  gap: theme('spacing.2');               /* 8px */
}

/* Variant styles */
.stat-card--primary {
  border-color: var(--primary) / 20%;
  background: var(--primary) / 5%;
}

.stat-card--success {
  border-color: var(--success) / 20%;
  background: var(--success) / 5%;
}

.stat-card--error {
  border-color: var(--destructive) / 20%;
  background: var(--destructive) / 5%;
}

.stat-card--error.active {
  border-color: var(--destructive) / 30%;
  background: var(--destructive) / 10%;
}
```
