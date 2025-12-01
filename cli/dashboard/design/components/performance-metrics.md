# Performance Metrics Component Specification

## Document Info
- **Component**: views/PerformanceMetrics
- **Status**: Design Specification
- **Created**: 2024-12-01
- **Author**: UX Design

---

## 1. Overview

The PerformanceMetrics component provides a metrics dashboard showing aggregate service statistics and per-service performance data. It gives users insight into system health, resource usage, and service uptime.

### Use Cases
- **Health Overview**: Quick view of system health score and active services
- **Service Monitoring**: Per-service status, uptime, and health tracking
- **Resource Visibility**: Port usage and running processes

---

## 2. Component Breakdown

### 2.1 Visual Structure

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│  AGGREGATE METRICS SECTION                                                      │
│  ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐ ┌──────────────┐│
│  │  📊 Active       │ │  🔌 Active       │ │  ⏱️ Average      │ │  💚 Health    ││
│  │  Services        │ │  Ports           │ │  Uptime          │ │  Score        ││
│  │                  │ │                  │ │                  │ │               ││
│  │   3/5            │ │      8           │ │   2h 15m         │ │    85%        ││
│  │   ↑ +1           │ │   ↔ stable       │ │   ↑ +30m         │ │   ↑ +5%       ││
│  └──────────────────┘ └──────────────────┘ └──────────────────┘ └──────────────┘│
│                                                                                 │
│  SERVICE METRICS TABLE                                                          │
│  ┌─────────────────────────────────────────────────────────────────────────────┐│
│  │  Service      │ Status     │ Uptime      │ Port    │ Health    │ Response   ││
│  │───────────────│────────────│─────────────│─────────│───────────│────────────││
│  │  api          │ ● running  │ 3h 45m      │ 3100    │ healthy   │ 45ms       ││
│  │  web          │ ● running  │ 3h 45m      │ 3000    │ healthy   │ 12ms       ││
│  │  worker       │ ● running  │ 2h 30m      │ -       │ degraded  │ -          ││
│  │  db           │ ○ stopped  │ -           │ -       │ unknown   │ -          ││
│  │  cache        │ ⚠ error    │ -           │ 6379    │ unhealthy │ timeout    ││
│  └─────────────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Sub-Components

| Component | Description | Required |
|-----------|-------------|----------|
| **PerformanceMetrics** | Main container component | Yes |
| **MetricsSection** | Grid container for metric cards | Yes |
| **MetricCard** | Individual metric display | Yes |
| **ServiceMetricsTable** | Table of per-service metrics | Yes |

---

## 3. Props and Interfaces

### 3.1 Core Types

```typescript
/** Props for the main PerformanceMetrics component */
interface PerformanceMetricsProps {
  /** Services data for computing metrics */
  services: Service[]
  /** Health report from health stream (optional) */
  healthReport?: HealthReportEvent
  /** Additional class names */
  className?: string
  /** Data test ID for testing */
  'data-testid'?: string
}

/** Props for MetricCard */
interface MetricCardProps {
  /** Card title */
  title: string
  /** Primary display value */
  value: string | number
  /** Unit or secondary text */
  unit?: string
  /** Card icon */
  icon: React.ComponentType<{ className?: string }>
  /** Card color variant */
  variant: 'primary' | 'info' | 'warning' | 'success'
  /** Trend direction */
  trend?: 'up' | 'down' | 'stable' | null
  /** Trend value (e.g., "+5%") */
  trendValue?: string
}

/** Computed metrics for display */
interface AggregateMetrics {
  activeServices: number
  totalServices: number
  activePorts: number
  averageUptime: number  // in seconds
  healthScore: number    // 0-100
}

/** Per-service metrics row */
interface ServiceMetricsRow {
  name: string
  status: LocalServiceInfo['status']
  uptime: number | null  // seconds
  port: number | null
  health: LocalServiceInfo['health']
  responseTime: number | null  // milliseconds
}
```

---

## 4. Aggregate Metrics Cards

### 4.1 Active Services Card

**Data Source:**
```typescript
const activeCount = services.filter(s => 
  s.local?.status === 'running' || s.local?.status === 'ready'
).length
const totalCount = services.length
```

**Display:**
- Icon: `Activity` (lucide)
- Color: Primary
- Title: "Active Services"
- Value: `{active}/{total}`
- Trend: Compare to previous (if tracking history)

### 4.2 Active Ports Card

**Data Source:**
```typescript
const activePorts = services
  .filter(s => s.local?.port != null)
  .map(s => s.local!.port!)
const uniquePorts = new Set(activePorts).size
```

**Display:**
- Icon: `Network` (lucide)
- Color: Info blue
- Title: "Active Ports"
- Value: Count of unique ports
- Trend: null (no trend tracking)

### 4.3 Average Uptime Card

**Data Source:**
```typescript
const uptimes = services
  .filter(s => s.local?.startTime)
  .map(s => Date.now() - new Date(s.local!.startTime!).getTime())
const averageUptime = uptimes.length > 0 
  ? uptimes.reduce((a, b) => a + b, 0) / uptimes.length 
  : 0
```

**Display:**
- Icon: `Clock` (lucide)
- Color: Info
- Title: "Average Uptime"
- Value: Formatted duration (e.g., "2h 15m", "45m", "5d 3h")
- Trend: up/down based on recent change

### 4.4 Health Score Card

**Data Source:**
```typescript
// From healthReport if available, otherwise calculate from services
const healthScore = healthReport 
  ? (healthReport.summary.healthy / healthReport.summary.total) * 100
  : calculateHealthScore(services)

function calculateHealthScore(services: Service[]): number {
  const total = services.length
  if (total === 0) return 100
  const healthy = services.filter(s => s.local?.health === 'healthy').length
  return Math.round((healthy / total) * 100)
}
```

**Display:**
- Icon: `Heart` (lucide)
- Color: Success (green) if ≥80%, Warning (yellow) if 50-79%, Error (red) if <50%
- Title: "Health Score"
- Value: Percentage with % suffix
- Trend: Compare to previous reading

---

## 5. MetricCard Visual States

### 5.1 State Matrix

| Variant | Border | Background | Icon Color |
|---------|--------|------------|------------|
| **primary** | `border-primary/20` | `bg-primary/5` | `text-primary` |
| **info** | `border-blue-500/20` | `bg-blue-500/5` | `text-blue-500` |
| **success** | `border-green-500/20` | `bg-green-500/5` | `text-green-500` |
| **warning** | `border-amber-500/20` | `bg-amber-500/5` | `text-amber-500` |
| **error** | `border-destructive/20` | `bg-destructive/5` | `text-destructive` |

### 5.2 Trend Indicators

| Trend | Icon | Color | Animation |
|-------|------|-------|-----------|
| up | `TrendingUp` | `text-green-500` | none |
| down | `TrendingDown` | `text-red-500` | none |
| stable | `Minus` | `text-muted-foreground` | none |
| null | - | - | hidden |

### 5.3 Visual Layout

```
METRIC CARD:
┌─────────────────────────────────────┐
│  ┌─────┐                            │
│  │ 📊 │  Active Services            │   Icon + Title row
│  └─────┘                            │
│                                     │
│         3/5                         │   Large value (centered)
│                                     │
│        ↑ +1                         │   Trend indicator
└─────────────────────────────────────┘

Dimensions:
- Min width: 160px
- Padding: p-6
- Border radius: rounded-lg
- Icon size: h-6 w-6
- Value font: text-3xl font-bold
- Title font: text-sm font-medium
- Trend font: text-sm
```

---

## 6. Service Metrics Table

### 6.1 Table Structure

| Column | Type | Sortable | Width |
|--------|------|----------|-------|
| Service | string | yes | flex |
| Status | badge | yes | 100px |
| Uptime | duration | yes | 120px |
| Port | number | yes | 80px |
| Health | badge | yes | 100px |
| Response | duration | yes | 100px |

### 6.2 Status Badge Values

| Status | Color | Icon |
|--------|-------|------|
| running | `bg-green-500/10 text-green-500` | ● solid circle |
| ready | `bg-green-500/10 text-green-500` | ● solid circle |
| starting | `bg-yellow-500/10 text-yellow-500` | ◐ half circle |
| stopping | `bg-yellow-500/10 text-yellow-500` | ◑ half circle |
| stopped | `bg-gray-500/10 text-gray-500` | ○ empty circle |
| error | `bg-red-500/10 text-red-500` | ⚠ warning |
| not-running | `bg-gray-500/10 text-gray-500` | ○ empty circle |

### 6.3 Health Badge Values

| Health | Color | Text |
|--------|-------|------|
| healthy | `bg-green-500/10 text-green-500` | Healthy |
| degraded | `bg-yellow-500/10 text-yellow-500` | Degraded |
| unhealthy | `bg-red-500/10 text-red-500` | Unhealthy |
| unknown | `bg-gray-500/10 text-gray-500` | Unknown |

### 6.4 Duration Formatting

```typescript
function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  if (seconds < 86400) {
    const hours = Math.floor(seconds / 3600)
    const mins = Math.floor((seconds % 3600) / 60)
    return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`
  }
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  return hours > 0 ? `${days}d ${hours}h` : `${days}d`
}
```

### 6.5 Response Time Display

- `< 100ms`: Green text
- `100-500ms`: Yellow text
- `> 500ms` or "timeout": Red text
- null: "-" in muted text

---

## 7. Interactions

### 7.1 Metric Card Interactions

| Action | Result |
|--------|--------|
| Hover | Subtle background opacity increase |
| Click (optional) | Navigate to detailed view (future) |

### 7.2 Table Interactions

| Action | Target | Result |
|--------|--------|--------|
| Click header | Column | Sort ascending → descending → default |
| Hover row | Table row | Background highlight |
| Click row | Table row | (Optional) Open service details |

---

## 8. Accessibility

### 8.1 WCAG 2.1 AA Compliance

| Criterion | Implementation |
|-----------|----------------|
| **1.3.1 Info & Relationships** | Semantic table with proper headers |
| **1.4.1 Use of Color** | Status text + icons, not color alone |
| **1.4.3 Contrast (Minimum)** | All text meets 4.5:1 ratio |
| **2.1.1 Keyboard** | Table rows focusable, sortable via keyboard |
| **2.4.6 Headings and Labels** | Section headings, table caption |
| **4.1.2 Name, Role, Value** | ARIA labels on sort buttons |

### 8.2 ARIA Implementation

```tsx
// Metric cards
<div
  role="region"
  aria-labelledby="metrics-title"
>
  <h3 id="metrics-title" className="sr-only">Aggregate Metrics</h3>
  <div role="status" aria-label="3 of 5 services active">
    {/* Card content */}
  </div>
</div>

// Table
<table
  role="grid"
  aria-label="Service performance metrics"
>
  <thead>
    <tr>
      <th scope="col" aria-sort="ascending">Service</th>
      {/* ... */}
    </tr>
  </thead>
  <tbody>
    <tr role="row">
      <td role="gridcell">{service.name}</td>
      {/* ... */}
    </tr>
  </tbody>
</table>
```

### 8.3 Keyboard Navigation

| Key | Action |
|-----|--------|
| `Tab` | Move focus between sections/interactive elements |
| `Enter` | Activate sort on focused column header |
| `Arrow Keys` | Navigate table cells (when focused) |

---

## 9. Design Tokens

### 9.1 Colors

| Token | Light Theme | Dark Theme | Usage |
|-------|-------------|------------|-------|
| `--primary` | `#7c3aed` | `#a78bfa` | Primary metric card |
| `--info` | `#3b82f6` | `#60a5fa` | Info metric cards |
| `--success` | `#16a34a` | `#22c55e` | Health/running states |
| `--warning` | `#ea580c` | `#f97316` | Degraded/starting |
| `--destructive` | `#dc2626` | `#ef4444` | Error/unhealthy |
| `--muted-foreground` | `#64748b` | `#94a3b8` | Secondary text |

### 9.2 Typography

| Element | Font Size | Font Weight | Line Height |
|---------|-----------|-------------|-------------|
| Section title | `text-lg` (18px) | `font-semibold` (600) | `leading-normal` |
| Card title | `text-sm` (14px) | `font-medium` (500) | `leading-normal` |
| Card value | `text-3xl` (30px) | `font-bold` (700) | `leading-tight` |
| Trend text | `text-sm` (14px) | `font-normal` (400) | `leading-normal` |
| Table header | `text-xs` (12px) | `font-medium` (500) | `leading-normal` |
| Table cell | `text-sm` (14px) | `font-normal` (400) | `leading-normal` |

### 9.3 Spacing

| Property | Value | Token |
|----------|-------|-------|
| Section padding | `p-6` | 24px |
| Metrics grid gap | `gap-4` | 16px |
| Card padding | `p-6` | 24px |
| Table padding | `px-4 py-3` | 16px × 12px |
| Section gap | `gap-6` | 24px |

---

## 10. Responsive Behavior

### 10.1 Breakpoint Adaptations

```
DESKTOP (≥1024px):
┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐
│ Active    │ │ Ports     │ │ Uptime    │ │ Health    │
└───────────┘ └───────────┘ └───────────┘ └───────────┘

Full table with all columns


TABLET (640-1023px):
┌───────────┐ ┌───────────┐
│ Active    │ │ Ports     │
└───────────┘ └───────────┘
┌───────────┐ ┌───────────┐
│ Uptime    │ │ Health    │
└───────────┘ └───────────┘

Table with horizontal scroll


MOBILE (<640px):
┌──────────────────────────┐
│ Active Services          │
└──────────────────────────┘
┌──────────────────────────┘
│ Active Ports             │
└──────────────────────────┘
... (stacked cards)

Table with horizontal scroll, reduced columns
```

### 10.2 Grid Configuration

```typescript
// Metrics grid
const metricsGridClass = cn(
  'grid gap-4',
  'grid-cols-1 sm:grid-cols-2 lg:grid-cols-4'
)
```

---

## 11. Integration with App.tsx

### 11.1 Sidebar Update

```tsx
// In Sidebar.tsx - add to navItems array
import { BarChart3 } from 'lucide-react'

const navItems = [
  { id: 'resources', label: 'Resources', icon: Activity },
  { id: 'console', label: 'Console', icon: Terminal },
  { id: 'environment', label: 'Environment', icon: Settings2 },
  { id: 'actions', label: 'Actions', icon: Zap },
  { id: 'metrics', label: 'Metrics', icon: BarChart3 },  // NEW
]
```

### 11.2 App.tsx View Rendering

```tsx
// In App.tsx - add to renderContent function
if (activeView === 'metrics') {
  return (
    <>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-foreground">Metrics</h2>
      </div>
      <PerformanceMetrics 
        services={services} 
        healthReport={healthReport}
      />
    </>
  )
}
```

---

## 12. Implementation Reference

```tsx
import * as React from 'react'
import {
  Activity,
  Network,
  Clock,
  Heart,
  TrendingUp,
  TrendingDown,
  Minus
} from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import type { Service, HealthReportEvent } from '@/types'

// Helper functions
export function countActiveServices(services: Service[]): number {
  return services.filter(s => 
    s.local?.status === 'running' || s.local?.status === 'ready'
  ).length
}

export function countActivePorts(services: Service[]): number {
  const ports = services
    .filter(s => s.local?.port != null)
    .map(s => s.local!.port!)
  return new Set(ports).size
}

export function calculateAverageUptime(services: Service[]): number {
  const now = Date.now()
  const uptimes = services
    .filter(s => s.local?.startTime)
    .map(s => now - new Date(s.local!.startTime!).getTime())
  
  if (uptimes.length === 0) return 0
  return Math.floor(uptimes.reduce((a, b) => a + b, 0) / uptimes.length / 1000)
}

export function calculateHealthScore(services: Service[]): number {
  if (services.length === 0) return 100
  const healthy = services.filter(s => s.local?.health === 'healthy').length
  return Math.round((healthy / services.length) * 100)
}

export function formatDuration(seconds: number): string {
  if (seconds === 0) return '-'
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  if (seconds < 86400) {
    const hours = Math.floor(seconds / 3600)
    const mins = Math.floor((seconds % 3600) / 60)
    return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`
  }
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  return hours > 0 ? `${days}d ${hours}h` : `${days}d`
}

export function formatResponseTime(ms: number | null | undefined): string {
  if (ms == null) return '-'
  if (ms < 1000) return `${Math.round(ms)}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

export function getResponseTimeVariant(ms: number | null | undefined): 'success' | 'warning' | 'error' | 'default' {
  if (ms == null) return 'default'
  if (ms < 100) return 'success'
  if (ms < 500) return 'warning'
  return 'error'
}
```

---

## 13. Testing Checklist

### 13.1 Unit Tests

**Helper Functions**
- [ ] countActiveServices - counts running + ready
- [ ] countActivePorts - counts unique ports
- [ ] calculateAverageUptime - handles empty, single, multiple
- [ ] calculateHealthScore - returns 0-100
- [ ] formatDuration - seconds, minutes, hours, days
- [ ] formatResponseTime - ms, seconds, null
- [ ] getResponseTimeVariant - thresholds

**MetricCard**
- [ ] Renders title, value, icon
- [ ] Applies correct variant styles
- [ ] Shows trend indicator when provided
- [ ] Hides trend when null

**PerformanceMetrics**
- [ ] Renders all metric cards
- [ ] Computes correct statistics
- [ ] Handles empty services array
- [ ] Handles all services stopped

**Service Table**
- [ ] Renders all columns
- [ ] Displays correct status badges
- [ ] Displays correct health badges
- [ ] Formats uptime correctly
- [ ] Shows "-" for null values
- [ ] Sorts by column when clicked

### 13.2 Accessibility Tests

- [ ] Metric cards have aria-label
- [ ] Table has proper headers and scope
- [ ] Sort buttons have aria-sort
- [ ] Focus indicators visible
- [ ] Keyboard navigation works

### 13.3 Integration Tests

- [ ] Sidebar navigation to Metrics view works
- [ ] Metrics view renders in App.tsx
- [ ] Services data flows correctly
- [ ] Health report integration works

---

## 14. Related Components

| Component | Relationship |
|-----------|--------------|
| **QuickActions** | Similar stat cards pattern |
| **ServiceTable** | Similar table structure |
| **Badge** | Used for status/health badges |
| **Table** | Uses Table UI component |

---

## Appendix A: Token Quick Reference

```css
/* PerformanceMetrics Tokens */
.performance-metrics {
  gap: theme('spacing.6');               /* 24px */
}

.metric-card {
  padding: theme('spacing.6');           /* 24px */
  border-radius: theme('borderRadius.lg'); /* 8px */
}

.metric-card-value {
  font-size: theme('fontSize.3xl');      /* 30px */
  font-weight: theme('fontWeight.bold'); /* 700 */
}

.metric-card-title {
  font-size: theme('fontSize.sm');       /* 14px */
  font-weight: theme('fontWeight.medium'); /* 500 */
  color: var(--muted-foreground);
}

.metric-card-icon {
  height: theme('spacing.6');            /* 24px */
  width: theme('spacing.6');             /* 24px */
}

.table-header {
  font-size: theme('fontSize.xs');       /* 12px */
  font-weight: theme('fontWeight.medium'); /* 500 */
}

.table-cell {
  font-size: theme('fontSize.sm');       /* 14px */
  padding: theme('spacing.3') theme('spacing.4'); /* 12px 16px */
}

/* Variant styles same as QuickActions */
```
