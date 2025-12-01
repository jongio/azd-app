# EnvironmentPanel Component Specification

## Document Info
- **Component**: views/EnvironmentPanel
- **Status**: Design Specification
- **Created**: 2024-12-01
- **Author**: UX Design

---

## 1. Overview

The EnvironmentPanel component provides an aggregated view of all environment variables across services in the dashboard. It displays environment variables in a searchable, filterable table format with support for sensitive value masking, service association badges, and copy-to-clipboard functionality.

### Use Cases
- **DevOps Engineers**: View all environment variables across services at a glance
- **Debugging**: Quickly find specific environment variables by name or value
- **Security Review**: Identify sensitive values that are properly masked
- **Configuration Audit**: See which services share common environment variables

---

## 2. Component Breakdown

### 2.1 Visual Structure

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│  ┌─────────────────────────────────────────────────────────────────────────────┐│
│  │ Environment Variables                                           [👁 Show]  ││
│  └─────────────────────────────────────────────────────────────────────────────┘│
│  ┌─────────────────────────────────────────────────────────────────────────────┐│
│  │ [🔍 Search variables...]                    [Service ▼]  [{N} variables]   ││
│  └─────────────────────────────────────────────────────────────────────────────┘│
│  ┌─────────────────────────────────────────────────────────────────────────────┐│
│  │  Variable          │ Value                    │ Services                    ││
│  ├─────────────────────────────────────────────────────────────────────────────┤│
│  │  DATABASE_URL      │ postgres://... [📋]     │ [api] [worker]              ││
│  │  API_KEY 🔒        │ ••••••••••••• [📋]     │ [api]                       ││
│  │  NODE_ENV          │ development   [📋]     │ [api] [web] [worker]        ││
│  │  PORT              │ 3000          [📋]     │ [web]                       ││
│  │  ...                                                                        ││
│  └─────────────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────────────┘

Detailed Anatomy:
┌─────────────────────────────────────────────────────────────────────────────────┐
│ TOOLBAR                                                                         │
│ ┌─────────────────────────────────────────────────────────────────────────────┐│
│ │ Title: "Environment Variables"              [👁 Show Values] Toggle Button  ││
│ └─────────────────────────────────────────────────────────────────────────────┘│
│ ┌─────────────────────────────────────────────────────────────────────────────┐│
│ │ [🔍] Search Input (flex-1)  │ [Service Dropdown]  │  Count Badge            ││
│ └─────────────────────────────────────────────────────────────────────────────┘│
│                                                                                 │
│ TABLE                                                                           │
│ ┌───────────────────┬─────────────────────────────┬───────────────────────────┐│
│ │ Variable (sortable)│ Value                      │ Services                  ││
│ ├───────────────────┼─────────────────────────────┼───────────────────────────┤│
│ │ [🔒] NAME         │ [value/masked] [copy btn]  │ [badge] [badge]           ││
│ └───────────────────┴─────────────────────────────┴───────────────────────────┘│
│                                                                                 │
│ EMPTY STATE (when no results)                                                   │
│ ┌─────────────────────────────────────────────────────────────────────────────┐│
│ │                    No environment variables found                            ││
│ │                    Try adjusting your search or filter                       ││
│ └─────────────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Sub-Components

| Component | Description | Required |
|-----------|-------------|----------|
| **EnvironmentPanel** | Main container component | Yes |
| **EnvironmentToolbar** | Header with title and show/hide toggle | Yes |
| **EnvironmentFilters** | Search input and service filter dropdown | Yes |
| **EnvironmentTable** | Table displaying environment variables | Yes |
| **EnvironmentRow** | Individual row with variable, value, services | Yes |
| **SensitiveValueCell** | Value cell with mask/reveal and copy | Yes |
| **ServiceBadgeGroup** | Group of service badges | Yes |
| **EmptyState** | Displayed when no variables match filters | Yes |

---

## 3. Props and Interfaces

### 3.1 Core Types

```typescript
/** Aggregated environment variable with service associations */
interface AggregatedEnvVar {
  /** Environment variable name */
  name: string
  /** Environment variable value */
  value: string
  /** List of services that use this variable */
  services: string[]
  /** Whether the variable is considered sensitive */
  isSensitive: boolean
}

/** Props for the main EnvironmentPanel component */
interface EnvironmentPanelProps {
  /** Additional class names */
  className?: string
  /** Data test ID for testing */
  'data-testid'?: string
}

/** Props for EnvironmentToolbar */
interface EnvironmentToolbarProps {
  /** Whether values are currently visible */
  showValues: boolean
  /** Callback when show/hide toggle changes */
  onToggleShowValues: () => void
}

/** Props for EnvironmentFilters */
interface EnvironmentFiltersProps {
  /** Current search query */
  searchQuery: string
  /** Callback when search query changes */
  onSearchChange: (query: string) => void
  /** Currently selected service filter */
  selectedService: string | null
  /** Callback when service filter changes */
  onServiceChange: (service: string | null) => void
  /** List of available services for dropdown */
  availableServices: string[]
  /** Total count of filtered variables */
  filteredCount: number
  /** Total count of all variables */
  totalCount: number
}

/** Props for EnvironmentTable */
interface EnvironmentTableProps {
  /** List of environment variables to display */
  variables: AggregatedEnvVar[]
  /** Whether to show values or mask them */
  showValues: boolean
  /** Callback when a value is copied */
  onCopy?: (name: string, value: string) => void
}

/** Props for EnvironmentRow */
interface EnvironmentRowProps {
  /** Environment variable data */
  variable: AggregatedEnvVar
  /** Whether to show the value or mask it */
  showValue: boolean
  /** Callback when value is copied */
  onCopy?: (name: string, value: string) => void
}

/** Props for SensitiveValueCell */
interface SensitiveValueCellProps {
  /** The actual value */
  value: string
  /** Whether the value is sensitive */
  isSensitive: boolean
  /** Whether to show the value (overrides sensitive masking) */
  showValue: boolean
  /** Callback when copy button clicked */
  onCopy?: () => void
}

/** Props for ServiceBadgeGroup */
interface ServiceBadgeGroupProps {
  /** List of service names */
  services: string[]
  /** Maximum badges to show before "+N more" */
  maxVisible?: number
}
```

### 3.2 Default Values

| Prop | Default |
|------|---------|
| `showValues` | `false` |
| `searchQuery` | `''` |
| `selectedService` | `null` (All services) |
| `maxVisible` (ServiceBadgeGroup) | `3` |

---

## 4. Component States

### 4.1 State Matrix

| State | Description | Visual Treatment |
|-------|-------------|------------------|
| **Loading** | Fetching environment data | Skeleton loader in table |
| **Empty (No Data)** | No services have environment variables | Empty state with message |
| **Empty (Filtered)** | No variables match current filters | Empty state with filter hint |
| **Default** | Variables displayed, values masked | Table with masked values |
| **Values Visible** | Variables displayed, values shown | Table with visible values |
| **Searching** | User is typing in search | Live filtering, debounced |
| **Filtered by Service** | Service filter applied | Filtered results, badge highlighted |

### 4.2 State Visuals

```
LOADING STATE:
┌─────────────────────────────────────────────────────────────┐
│  Environment Variables                          [👁 Show]   │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ [████████████████]      [████████]     [████]          ││
│  │ [████████████████]      [████████]     [████]          ││
│  │ [████████████████]      [████████]     [████]          ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘

EMPTY STATE (No Data):
┌─────────────────────────────────────────────────────────────┐
│  Environment Variables                          [👁 Show]   │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                                                         ││
│  │              📭 No Environment Variables                ││
│  │         Services haven't defined any variables          ││
│  │                                                         ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘

EMPTY STATE (Filtered):
┌─────────────────────────────────────────────────────────────┐
│  Environment Variables                          [👁 Show]   │
│  [🔍 "DATABASE"]                    [api ▼]     0 of 15    │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                                                         ││
│  │              🔍 No Results Found                        ││
│  │      No variables match "DATABASE" in api service       ││
│  │              [Clear filters]                            ││
│  │                                                         ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘

VALUES MASKED (Default):
┌─────────────────────────────────────────────────────────────┐
│  Variable          │ Value                    │ Services    │
│  API_KEY 🔒        │ •••••••••••••  [📋]     │ [api]       │
│  DATABASE_URL 🔒   │ •••••••••••••  [📋]     │ [api]       │
│  NODE_ENV          │ development    [📋]     │ [api] [web] │
└─────────────────────────────────────────────────────────────┘

VALUES VISIBLE (Toggle On):
┌─────────────────────────────────────────────────────────────┐
│  Variable          │ Value                    │ Services    │
│  API_KEY 🔒        │ sk-abc123xyz   [📋]     │ [api]       │
│  DATABASE_URL 🔒   │ postgres://... [📋]     │ [api]       │
│  NODE_ENV          │ development    [📋]     │ [api] [web] │
└─────────────────────────────────────────────────────────────┘
```

### 4.3 Toggle Button States

```
TOGGLE OFF (Default - Values Hidden):
┌─────────────────────────┐
│  [👁] Show Values       │   text-muted-foreground, bg-transparent
└─────────────────────────┘

TOGGLE ON (Values Visible):
┌─────────────────────────┐
│  [👁‍🗨] Hide Values       │   text-foreground, bg-secondary
└─────────────────────────┘

TOGGLE HOVER:
┌─────────────────────────┐
│  [👁] Show Values       │   bg-secondary/50
└─────────────────────────┘
```

---

## 5. Interactions

### 5.1 Search Interaction

| Action | Result |
|--------|--------|
| Type in search | Live filter (300ms debounce) by variable name OR value |
| Clear search (X button) | Reset to show all variables |
| Empty search + Enter | No action |

```
Search Flow:
┌───────────────────────────────────────────────────────────────┐
│  1. User types "DATA" in search box                           │
│     → Debounce 300ms                                          │
│     → Filter variables where name OR value contains "DATA"    │
│     → Update count badge: "3 of 15 variables"                 │
│                                                               │
│  2. User clears search                                        │
│     → Reset filter                                            │
│     → Show all variables                                      │
│     → Update count badge: "15 variables"                      │
└───────────────────────────────────────────────────────────────┘
```

### 5.2 Service Filter Interaction

| Action | Result |
|--------|--------|
| Click dropdown | Open service list |
| Select service | Filter to show only variables used by that service |
| Select "All Services" | Remove service filter |
| Click outside | Close dropdown |

```
Service Filter Flow:
┌───────────────────────────────────────────────────────────────┐
│  1. User clicks service dropdown                              │
│     → Dropdown opens with: "All Services", "api", "web", etc. │
│                                                               │
│  2. User selects "api"                                        │
│     → Filter to variables where services includes "api"       │
│     → Dropdown closes                                         │
│     → "api" badge highlighted in matching rows                │
│                                                               │
│  3. User selects "All Services"                               │
│     → Remove service filter                                   │
│     → Show all variables                                      │
└───────────────────────────────────────────────────────────────┘
```

### 5.3 Show/Hide Values Toggle

| Action | Result |
|--------|--------|
| Click toggle (off) | Reveal all values, button shows "Hide Values" |
| Click toggle (on) | Mask sensitive values, button shows "Show Values" |

```
Toggle Flow:
┌───────────────────────────────────────────────────────────────┐
│  1. Default state: Values hidden                              │
│     → Sensitive values show "•••••••••••••"                   │
│     → Non-sensitive values show actual value                  │
│     → Button label: "Show Values"                             │
│                                                               │
│  2. User clicks "Show Values"                                 │
│     → ALL values become visible (including sensitive)         │
│     → Button label changes to "Hide Values"                   │
│     → Button gets active styling                              │
│                                                               │
│  3. User clicks "Hide Values"                                 │
│     → Return to masked state                                  │
│     → Button label: "Show Values"                             │
└───────────────────────────────────────────────────────────────┘
```

### 5.4 Copy Value Interaction

| Action | Result |
|--------|--------|
| Hover row | Copy button appears (if not already visible) |
| Click copy | Value copied, icon → checkmark, toast optional |
| After 2s | Checkmark returns to copy icon |

```
Copy Flow:
┌───────────────────────────────────────────────────────────────┐
│  1. User hovers over row                                      │
│     → Row gets subtle highlight                               │
│     → Copy button visible (opacity transition)                │
│                                                               │
│  2. User clicks copy button                                   │
│     → ACTUAL value copied (even if masked)                    │
│     → Icon changes: 📋 → ✓                                    │
│     → Icon color: text-success                                │
│     → Optional: toast notification "Copied API_KEY"           │
│                                                               │
│  3. After 2 seconds                                           │
│     → Icon returns to copy (📋)                               │
│     → Color returns to default                                │
└───────────────────────────────────────────────────────────────┘
```

### 5.5 Service Badge Interaction

| Action | Result |
|--------|--------|
| Click badge | Apply service filter for that service |
| Hover badge | Tooltip with full service name (if truncated) |
| "+N more" badge | Expand to show all services |

---

## 6. Sensitive Value Detection

### 6.1 Detection Patterns

Environment variables are classified as sensitive if their name (case-insensitive) contains any of the following patterns:

```typescript
const SENSITIVE_PATTERNS = [
  'key',
  'secret',
  'password',
  'token',
  'credential',
  'auth',
  'api_key',
  'apikey',
  'private',
  'cert',
  'connection_string',
  'connectionstring',
]

function isSensitiveVariable(name: string): boolean {
  const lowerName = name.toLowerCase()
  return SENSITIVE_PATTERNS.some(pattern => lowerName.includes(pattern))
}
```

### 6.2 Masking Rules

| Scenario | Display |
|----------|---------|
| Sensitive + showValues=false | `••••••••••••` (12 dots) |
| Sensitive + showValues=true | Actual value |
| Non-sensitive + showValues=false | Actual value |
| Non-sensitive + showValues=true | Actual value |

### 6.3 Visual Indicators

```
Sensitive Variable Row:
┌─────────────────────────────────────────────────────────────┐
│  🔒 API_KEY          │ •••••••••••••  [📋]     │ [api]     │
└─────────────────────────────────────────────────────────────┘
   ↑ Lock icon indicates sensitive variable

Non-Sensitive Variable Row:
┌─────────────────────────────────────────────────────────────┐
│     NODE_ENV         │ development    [📋]     │ [api]     │
└─────────────────────────────────────────────────────────────┘
   ↑ No lock icon
```

---

## 7. Accessibility

### 7.1 WCAG 2.1 AA Compliance

| Criterion | Implementation |
|-----------|----------------|
| **1.3.1 Info & Relationships** | Table uses proper `<table>`, `<thead>`, `<tbody>` structure |
| **1.3.2 Meaningful Sequence** | Logical reading order: toolbar → filters → table |
| **1.4.3 Contrast (Minimum)** | All text meets 4.5:1 ratio |
| **1.4.11 Non-text Contrast** | Icons and badges meet 3:1 ratio |
| **2.1.1 Keyboard** | All interactive elements keyboard accessible |
| **2.4.3 Focus Order** | Logical tab order through controls |
| **2.4.6 Headings and Labels** | Clear column headers and form labels |
| **2.4.7 Focus Visible** | Visible focus indicators on all focusable elements |
| **4.1.2 Name, Role, Value** | ARIA labels on all controls |

### 7.2 ARIA Implementation

```tsx
// Main Panel
<section
  aria-labelledby="env-panel-title"
  className="environment-panel"
>
  <h2 id="env-panel-title" className="sr-only">
    Environment Variables
  </h2>
  
  // Search Input
  <label htmlFor="env-search" className="sr-only">
    Search environment variables
  </label>
  <input
    id="env-search"
    type="search"
    role="searchbox"
    aria-describedby="env-search-hint"
    placeholder="Search variables..."
  />
  <span id="env-search-hint" className="sr-only">
    Search by variable name or value
  </span>
  
  // Service Filter
  <label htmlFor="env-service-filter" className="sr-only">
    Filter by service
  </label>
  <select
    id="env-service-filter"
    aria-label="Filter variables by service"
  >
    <option value="">All Services</option>
    {/* services */}
  </select>
  
  // Show/Hide Toggle
  <button
    aria-pressed={showValues}
    aria-label={showValues ? 'Hide sensitive values' : 'Show sensitive values'}
  >
    {showValues ? 'Hide Values' : 'Show Values'}
  </button>
  
  // Results Count (Live Region)
  <div aria-live="polite" aria-atomic="true">
    {filteredCount} of {totalCount} variables
  </div>
  
  // Table
  <table aria-label="Environment variables">
    <thead>
      <tr>
        <th scope="col">Variable</th>
        <th scope="col">Value</th>
        <th scope="col">Services</th>
      </tr>
    </thead>
    <tbody>
      {/* rows */}
    </tbody>
  </table>
  
  // Sensitive Indicator
  <span aria-label="Sensitive value" title="Sensitive value">
    <Lock className="h-3 w-3" aria-hidden="true" />
  </span>
  
  // Copy Button
  <button
    aria-label={copied ? `${name} copied` : `Copy ${name} value`}
    aria-live="polite"
  >
    {copied ? <Check /> : <Copy />}
  </button>
</section>
```

### 7.3 Keyboard Navigation

| Key | Element | Action |
|-----|---------|--------|
| `Tab` | All focusable | Move to next focusable element |
| `Shift+Tab` | All focusable | Move to previous focusable element |
| `Enter` | Toggle button | Toggle show/hide values |
| `Enter` | Copy button | Copy value to clipboard |
| `Space` | Toggle button | Toggle show/hide values |
| `Space` | Copy button | Copy value to clipboard |
| `Escape` | Search input | Clear search |
| `Escape` | Dropdown | Close dropdown |
| `↑/↓` | Dropdown | Navigate options |
| `Enter` | Dropdown option | Select option |

### 7.4 Screen Reader Announcements

| Action | Announcement |
|--------|--------------|
| Toggle show values | "Showing/Hiding sensitive values" |
| Copy success | "[Variable name] copied to clipboard" |
| Filter applied | "Showing X of Y variables" |
| No results | "No environment variables found matching [criteria]" |

### 7.5 Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  .environment-panel,
  .environment-panel * {
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
  }
}
```

---

## 8. Design Tokens

### 8.1 Colors

| Token | Light Theme | Dark Theme | Usage |
|-------|-------------|------------|-------|
| `--foreground` | `var(--gray-900)` | `var(--slate-50)` | Variable names, values |
| `--muted-foreground` | `var(--gray-600)` | `var(--slate-400)` | Column headers, hints |
| `--secondary` | `var(--gray-100)` | `var(--slate-800)` | Hover backgrounds, badges |
| `--primary` | `var(--purple-600)` | `var(--purple-500)` | Active filter indicator |
| `--success` | `var(--green-600)` | `var(--green-500)` | Copy success icon |
| `--border` | `var(--gray-200)` | `var(--slate-700)` | Table borders, separators |
| `--card` | `var(--white)` | `var(--slate-900)` | Panel background |
| `--card-border` | `var(--gray-200)` | `var(--slate-800)` | Panel border |
| `--sensitive-indicator` | `var(--amber-500)` | `var(--amber-400)` | Lock icon color |

### 8.2 Typography

| Element | Font Size | Font Weight | Line Height |
|---------|-----------|-------------|-------------|
| Panel title | `text-xl` (20px) | `font-semibold` (600) | `leading-normal` |
| Column headers | `text-sm` (14px) | `font-semibold` (600) | `leading-normal` |
| Variable name | `text-sm` (14px) | `font-medium` (500) | `leading-normal` |
| Variable value | `text-sm` (14px) | `font-mono` | `leading-normal` |
| Service badge | `text-xs` (12px) | `font-semibold` (600) | `leading-tight` |
| Count badge | `text-xs` (12px) | `font-medium` (500) | `leading-tight` |

### 8.3 Spacing

| Property | Value | Token |
|----------|-------|-------|
| Panel padding | `p-6` | 24px |
| Toolbar margin-bottom | `mb-4` | 16px |
| Filters margin-bottom | `mb-4` | 16px |
| Search input height | `h-10` | 40px |
| Table cell padding | `px-4 py-3` | 16px × 12px |
| Badge padding | `px-2.5 py-0.5` | 10px × 2px |
| Badge gap | `gap-1.5` | 6px |
| Icon size (lock) | `h-3.5 w-3.5` | 14px |
| Icon size (copy) | `h-4 w-4` | 16px |
| Copy button size | `h-7 w-7` | 28px |

### 8.4 Effects

| Property | Value | Usage |
|----------|-------|-------|
| Transition duration | `200ms` | All interactive states |
| Transition timing | `ease` | Smooth state changes |
| Focus ring width | `2px` | Keyboard focus indicator |
| Focus ring offset | `2px` | Ring offset from element |
| Row hover bg | `bg-secondary/50` | Table row hover |
| Skeleton animation | `animate-pulse` | Loading state |
| Border radius (panel) | `rounded-lg` | 8px |
| Border radius (badges) | `rounded-full` | pill shape |

---

## 9. Responsive Behavior

### 9.1 Breakpoint Adaptations

```
DESKTOP (≥1024px):
┌─────────────────────────────────────────────────────────────────────────────────┐
│  Environment Variables                                         [👁 Show Values] │
│  [🔍 Search variables...]                    [Service ▼]       15 variables     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐│
│  │ Variable          │ Value                    │ Services                     ││
│  │ DATABASE_URL 🔒   │ •••••••••••••  [📋]     │ [api] [worker] [scheduler]   ││
│  └─────────────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────────────┘

TABLET (640-1023px):
┌─────────────────────────────────────────────────────────────┐
│  Environment Variables                      [👁 Show Values] │
│  [🔍 Search...]            [Service ▼]      15 vars          │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ Variable       │ Value           │ Services              ││
│  │ DATABASE_URL 🔒│ •••••••• [📋]  │ [api] [+2]            ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘

MOBILE (<640px):
┌───────────────────────────────────────────┐
│  Environment Variables        [👁 Show]   │
│  [🔍 Search...]                           │
│  [Service ▼]                   15 vars    │
│  ┌───────────────────────────────────────┐│
│  │ DATABASE_URL 🔒             [📋]      ││
│  │ ••••••••••••••                        ││
│  │ [api] [worker] [+1]                   ││
│  ├───────────────────────────────────────┤│
│  │ NODE_ENV                    [📋]      ││
│  │ development                            ││
│  │ [api] [web] [worker]                  ││
│  └───────────────────────────────────────┘│
└───────────────────────────────────────────┘
```

### 9.2 Responsive Rules

| Breakpoint | Behavior |
|------------|----------|
| **Desktop** (≥1024px) | Full table layout, all badges visible |
| **Tablet** (640-1023px) | Compact table, max 2 badges + overflow |
| **Mobile** (<640px) | Stacked card layout, search/filter on separate lines |

---

## 10. Data Aggregation Logic

### 10.1 Aggregation Algorithm

```typescript
/**
 * Aggregates environment variables from all services
 * Groups by variable name, collects all services that use each variable
 */
function aggregateEnvironmentVariables(services: Service[]): AggregatedEnvVar[] {
  const envMap = new Map<string, AggregatedEnvVar>()
  
  for (const service of services) {
    const envVars = service.environmentVariables ?? {}
    
    for (const [name, value] of Object.entries(envVars)) {
      const existing = envMap.get(name)
      
      if (existing) {
        // Variable exists - add service to list
        if (!existing.services.includes(service.name)) {
          existing.services.push(service.name)
        }
        // Note: If values differ between services, use first encountered
        // Could add conflict indicator in future
      } else {
        // New variable
        envMap.set(name, {
          name,
          value,
          services: [service.name],
          isSensitive: isSensitiveVariable(name),
        })
      }
    }
  }
  
  // Sort alphabetically by variable name
  return Array.from(envMap.values()).sort((a, b) => 
    a.name.localeCompare(b.name)
  )
}
```

### 10.2 Filtering Logic

```typescript
function filterEnvironmentVariables(
  variables: AggregatedEnvVar[],
  searchQuery: string,
  selectedService: string | null
): AggregatedEnvVar[] {
  return variables.filter(envVar => {
    // Service filter
    if (selectedService && !envVar.services.includes(selectedService)) {
      return false
    }
    
    // Search filter (name OR value)
    if (searchQuery) {
      const query = searchQuery.toLowerCase()
      const matchesName = envVar.name.toLowerCase().includes(query)
      const matchesValue = envVar.value.toLowerCase().includes(query)
      if (!matchesName && !matchesValue) {
        return false
      }
    }
    
    return true
  })
}
```

---

## 11. Integration with App.tsx

### 11.1 Sidebar Update

Add new navigation item to Sidebar component:

```tsx
// In Sidebar.tsx - add to navItems array
import { Activity, Terminal, Settings2 } from 'lucide-react'

const navItems = [
  { id: 'resources', label: 'Resources', icon: Activity },
  { id: 'console', label: 'Console', icon: Terminal },
  { id: 'environment', label: 'Environment', icon: Settings2 },  // NEW
]
```

### 11.2 App.tsx View Rendering

```tsx
// In App.tsx - add to renderContent function
if (activeView === 'environment') {
  return (
    <>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-foreground">Environment</h2>
      </div>
      <EnvironmentPanel services={services} />
    </>
  )
}
```

---

## 12. Implementation Reference

### 12.1 Main Component Structure

```tsx
import * as React from 'react'
import { Search, Settings2, Eye, EyeOff, Lock, Copy, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { 
  Table, 
  TableHeader, 
  TableBody, 
  TableHead, 
  TableRow, 
  TableCell 
} from '@/components/ui/table'
import { 
  Select, 
  SelectContent, 
  SelectItem, 
  SelectTrigger, 
  SelectValue 
} from '@/components/ui/select'
import { useClipboard } from '@/hooks/useClipboard'
import type { Service } from '@/types'

// ... type definitions from Section 3 ...

const SENSITIVE_PATTERNS = [
  'key', 'secret', 'password', 'token', 'credential', 
  'auth', 'api_key', 'apikey', 'private', 'cert',
  'connection_string', 'connectionstring',
]

function isSensitiveVariable(name: string): boolean {
  const lowerName = name.toLowerCase()
  return SENSITIVE_PATTERNS.some(pattern => lowerName.includes(pattern))
}

export function EnvironmentPanel({ 
  services,
  className, 
  'data-testid': testId 
}: EnvironmentPanelProps & { services: Service[] }) {
  const [showValues, setShowValues] = React.useState(false)
  const [searchQuery, setSearchQuery] = React.useState('')
  const [selectedService, setSelectedService] = React.useState<string | null>(null)
  const [debouncedSearch, setDebouncedSearch] = React.useState('')
  const { copiedField, copyToClipboard } = useClipboard()
  
  // Debounce search input
  React.useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(searchQuery), 300)
    return () => clearTimeout(timer)
  }, [searchQuery])
  
  // Aggregate environment variables
  const aggregatedVars = React.useMemo(() => 
    aggregateEnvironmentVariables(services),
    [services]
  )
  
  // Get unique service names
  const availableServices = React.useMemo(() => 
    [...new Set(services.map(s => s.name))].sort(),
    [services]
  )
  
  // Apply filters
  const filteredVars = React.useMemo(() =>
    filterEnvironmentVariables(aggregatedVars, debouncedSearch, selectedService),
    [aggregatedVars, debouncedSearch, selectedService]
  )
  
  const handleCopy = async (name: string, value: string) => {
    await copyToClipboard(value, name)
  }
  
  return (
    <section
      aria-labelledby="env-panel-title"
      className={cn('bg-card rounded-lg border border-card-border', className)}
      data-testid={testId}
    >
      {/* Toolbar */}
      <div className="flex items-center justify-between p-4 border-b border-border">
        <h2 id="env-panel-title" className="text-sm font-semibold text-foreground">
          Environment Variables ({aggregatedVars.length})
        </h2>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setShowValues(!showValues)}
          aria-pressed={showValues}
          aria-label={showValues ? 'Hide sensitive values' : 'Show sensitive values'}
          className="gap-2"
        >
          {showValues ? (
            <>
              <EyeOff className="h-4 w-4" />
              Hide Values
            </>
          ) : (
            <>
              <Eye className="h-4 w-4" />
              Show Values
            </>
          )}
        </Button>
      </div>
      
      {/* Filters */}
      <div className="flex items-center gap-3 p-4 border-b border-border">
        <div className="relative flex-1">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="search"
            placeholder="Search variables..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
            aria-label="Search environment variables"
          />
        </div>
        
        <Select
          value={selectedService ?? 'all'}
          onValueChange={(v) => setSelectedService(v === 'all' ? null : v)}
        >
          <SelectTrigger className="w-[180px]" aria-label="Filter by service">
            <SelectValue placeholder="All Services" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Services</SelectItem>
            {availableServices.map(service => (
              <SelectItem key={service} value={service}>{service}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        
        <div 
          className="text-xs text-muted-foreground whitespace-nowrap"
          aria-live="polite"
        >
          {filteredVars.length === aggregatedVars.length 
            ? `${aggregatedVars.length} variables`
            : `${filteredVars.length} of ${aggregatedVars.length} variables`
          }
        </div>
      </div>
      
      {/* Table or Empty State */}
      {filteredVars.length === 0 ? (
        <EmptyState 
          hasFilters={!!debouncedSearch || !!selectedService}
          onClearFilters={() => {
            setSearchQuery('')
            setSelectedService(null)
          }}
        />
      ) : (
        <Table aria-label="Environment variables">
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead className="w-[250px]">Variable</TableHead>
              <TableHead>Value</TableHead>
              <TableHead className="w-[250px]">Services</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filteredVars.map((envVar) => (
              <EnvironmentRow
                key={envVar.name}
                variable={envVar}
                showValue={showValues}
                copied={copiedField === envVar.name}
                onCopy={() => handleCopy(envVar.name, envVar.value)}
                selectedService={selectedService}
              />
            ))}
          </TableBody>
        </Table>
      )}
    </section>
  )
}

function EnvironmentRow({ 
  variable, 
  showValue, 
  copied,
  onCopy,
  selectedService 
}: EnvironmentRowProps & { copied: boolean; selectedService: string | null }) {
  const displayValue = variable.isSensitive && !showValue 
    ? '••••••••••••' 
    : variable.value
  
  return (
    <TableRow className="group">
      <TableCell className="font-medium">
        <div className="flex items-center gap-2">
          {variable.isSensitive && (
            <Lock 
              className="h-3.5 w-3.5 text-amber-500" 
              aria-label="Sensitive value"
              title="Sensitive value"
            />
          )}
          <span className="font-mono text-sm">{variable.name}</span>
        </div>
      </TableCell>
      
      <TableCell>
        <div className="flex items-center gap-2">
          <code className="text-sm font-mono text-muted-foreground truncate max-w-md">
            {displayValue}
          </code>
          <Button
            variant="ghost"
            size="icon"
            onClick={onCopy}
            aria-label={copied ? `${variable.name} copied` : `Copy ${variable.name} value`}
            className={cn(
              'h-7 w-7 shrink-0',
              'opacity-0 group-hover:opacity-100 focus:opacity-100',
              'transition-opacity',
              copied && 'text-green-600 dark:text-green-500'
            )}
          >
            {copied ? (
              <Check className="h-4 w-4" aria-hidden="true" />
            ) : (
              <Copy className="h-4 w-4" aria-hidden="true" />
            )}
          </Button>
        </div>
      </TableCell>
      
      <TableCell>
        <ServiceBadgeGroup 
          services={variable.services} 
          highlightedService={selectedService}
        />
      </TableCell>
    </TableRow>
  )
}

function ServiceBadgeGroup({ 
  services, 
  maxVisible = 3,
  highlightedService 
}: ServiceBadgeGroupProps & { highlightedService?: string | null }) {
  const visible = services.slice(0, maxVisible)
  const overflow = services.length - maxVisible
  
  return (
    <div className="flex flex-wrap gap-1.5">
      {visible.map(service => (
        <Badge
          key={service}
          variant={service === highlightedService ? 'default' : 'secondary'}
          className="text-xs"
        >
          {service}
        </Badge>
      ))}
      {overflow > 0 && (
        <Badge variant="outline" className="text-xs">
          +{overflow} more
        </Badge>
      )}
    </div>
  )
}

function EmptyState({ 
  hasFilters, 
  onClearFilters 
}: { 
  hasFilters: boolean
  onClearFilters: () => void 
}) {
  return (
    <div className="flex flex-col items-center justify-center py-12 px-4 text-center">
      <Settings2 className="h-12 w-12 text-muted-foreground/50 mb-4" />
      <h3 className="text-lg font-medium text-foreground mb-2">
        {hasFilters ? 'No Results Found' : 'No Environment Variables'}
      </h3>
      <p className="text-sm text-muted-foreground mb-4">
        {hasFilters 
          ? 'No variables match your current search or filter criteria.'
          : 'Services haven\'t defined any environment variables.'
        }
      </p>
      {hasFilters && (
        <Button variant="outline" size="sm" onClick={onClearFilters}>
          Clear Filters
        </Button>
      )}
    </div>
  )
}
```

---

## 13. Testing Checklist

### 13.1 Unit Tests

**EnvironmentPanel**
- [ ] Renders with services data
- [ ] Aggregates environment variables correctly
- [ ] Groups variables by name across services
- [ ] Detects sensitive variables correctly
- [ ] Shows correct variable count

**Filtering**
- [ ] Search filters by variable name
- [ ] Search filters by variable value
- [ ] Search is case-insensitive
- [ ] Search debounces input (300ms)
- [ ] Service filter shows only selected service variables
- [ ] Combined search + service filter works
- [ ] Clear filters resets both search and service

**Show/Hide Toggle**
- [ ] Default state hides sensitive values
- [ ] Toggle reveals all values
- [ ] Toggle hides values again
- [ ] Non-sensitive values always visible
- [ ] Button label updates correctly
- [ ] aria-pressed updates correctly

**Copy Functionality**
- [ ] Copy button copies actual value (even when masked)
- [ ] Copy button shows checkmark on success
- [ ] Checkmark reverts after 2 seconds
- [ ] onCopy callback fires
- [ ] Works with useClipboard hook

**Service Badges**
- [ ] Shows all services for variable
- [ ] Truncates with "+N more" when > maxVisible
- [ ] Highlighted badge when service filter active

**Empty States**
- [ ] Shows "No Environment Variables" when no data
- [ ] Shows "No Results Found" when filters match nothing
- [ ] "Clear Filters" button works

### 13.2 Accessibility Tests

- [ ] Panel has proper heading structure
- [ ] Search input has accessible label
- [ ] Service dropdown has accessible label
- [ ] Toggle button has aria-pressed
- [ ] Copy buttons have aria-label
- [ ] Results count announced to screen readers
- [ ] Lock icon has accessible name
- [ ] Keyboard navigation works throughout
- [ ] Focus indicators visible
- [ ] Reduced motion respected

### 13.3 Integration Tests

- [ ] Sidebar navigation to Environment view works
- [ ] Environment view renders in App.tsx
- [ ] Services data flows to panel correctly
- [ ] Real-time service updates reflected
- [ ] Works alongside Resources and Console views

---

## 14. Related Components

| Component | Relationship |
|-----------|--------------|
| **Table** | Uses Table components for data display |
| **Badge** | Uses Badge for service indicators |
| **Input** | Uses Input for search field |
| **Select** | Uses Select for service filter dropdown |
| **Button** | Uses Button for toggle and copy actions |
| **InfoField** | Pattern reference for copy functionality |
| **useClipboard** | Hook for managing copy state |
| **Sidebar** | Adds navigation entry for Environment view |

---

## Appendix A: Token Quick Reference

```css
/* EnvironmentPanel Tokens */
.environment-panel {
  background: var(--card);
  border: 1px solid var(--card-border);
  border-radius: theme('borderRadius.lg');  /* 8px */
}

.environment-panel-toolbar {
  padding: theme('spacing.4');              /* 16px */
  border-bottom: 1px solid var(--border);
}

.environment-panel-filters {
  padding: theme('spacing.4');              /* 16px */
  border-bottom: 1px solid var(--border);
  gap: theme('spacing.3');                  /* 12px */
}

.environment-search {
  height: theme('spacing.10');              /* 40px */
  padding-left: theme('spacing.9');         /* 36px - for icon */
}

.environment-table-row:hover {
  background: var(--secondary) / 50%;
}

.environment-variable-name {
  font-family: theme('fontFamily.mono');
  font-size: theme('fontSize.sm');          /* 14px */
  font-weight: theme('fontWeight.medium');  /* 500 */
}

.environment-variable-value {
  font-family: theme('fontFamily.mono');
  font-size: theme('fontSize.sm');          /* 14px */
  color: var(--muted-foreground);
}

.environment-sensitive-icon {
  height: theme('spacing.3.5');             /* 14px */
  width: theme('spacing.3.5');              /* 14px */
  color: var(--amber-500);
}

.environment-service-badge {
  font-size: theme('fontSize.xs');          /* 12px */
  padding: theme('spacing.0.5') theme('spacing.2.5'); /* 2px 10px */
  border-radius: theme('borderRadius.full');
}

.environment-copy-button {
  height: theme('spacing.7');               /* 28px */
  width: theme('spacing.7');                /* 28px */
  opacity: 0;
  transition: opacity 200ms ease;
}

.environment-row:hover .environment-copy-button,
.environment-copy-button:focus {
  opacity: 1;
}

.environment-copy-button--copied {
  color: var(--success);
}

.environment-empty-state {
  padding: theme('spacing.12') theme('spacing.4'); /* 48px 16px */
  text-align: center;
}
```

---

## Appendix B: User Flow Diagrams

```
User Flow 1: Navigate to Environment View
┌──────────────────────────────────────────────────────────────────┐
│  START                                                           │
│    │                                                             │
│    ▼                                                             │
│  User clicks "Environment" in Sidebar                            │
│    │                                                             │
│    ▼                                                             │
│  App.tsx sets activeView = 'environment'                         │
│    │                                                             │
│    ▼                                                             │
│  EnvironmentPanel renders with services data                     │
│    │                                                             │
│    ▼                                                             │
│  User sees aggregated environment variables table                │
│    │                                                             │
│    ▼                                                             │
│  END                                                             │
└──────────────────────────────────────────────────────────────────┘

User Flow 2: Search and Filter Variables
┌──────────────────────────────────────────────────────────────────┐
│  START                                                           │
│    │                                                             │
│    ▼                                                             │
│  User types "DATABASE" in search                                 │
│    │                                                             │
│    ▼                                                             │
│  300ms debounce                                                  │
│    │                                                             │
│    ▼                                                             │
│  Table filters to show matching variables                        │
│    │                                                             │
│    ▼                                                             │
│  Count updates: "2 of 15 variables"                              │
│    │                                                             │
│    ▼                                                             │
│  User selects "api" from service dropdown                        │
│    │                                                             │
│    ▼                                                             │
│  Table filters further to "api" service only                     │
│    │                                                             │
│    ▼                                                             │
│  Count updates: "1 of 15 variables"                              │
│    │                                                             │
│    ▼                                                             │
│  END                                                             │
└──────────────────────────────────────────────────────────────────┘

User Flow 3: Reveal and Copy Sensitive Value
┌──────────────────────────────────────────────────────────────────┐
│  START                                                           │
│    │                                                             │
│    ▼                                                             │
│  User sees "API_KEY" with masked value "••••••••••••"            │
│    │                                                             │
│    ▼                                                             │
│  User clicks "Show Values" toggle                                │
│    │                                                             │
│    ▼                                                             │
│  All values revealed, toggle shows "Hide Values"                 │
│    │                                                             │
│    ▼                                                             │
│  User hovers over API_KEY row                                    │
│    │                                                             │
│    ▼                                                             │
│  Copy button appears                                             │
│    │                                                             │
│    ▼                                                             │
│  User clicks copy button                                         │
│    │                                                             │
│    ▼                                                             │
│  Value copied, icon changes to checkmark                         │
│    │                                                             │
│    ▼                                                             │
│  After 2s, icon reverts to copy icon                             │
│    │                                                             │
│    ▼                                                             │
│  END                                                             │
└──────────────────────────────────────────────────────────────────┘
```
