# Service Dependencies Component Specification

## Document Info
- **Component**: views/ServiceDependencies
- **Status**: Design Specification
- **Created**: 2024-12-01
- **Author**: UX Design

---

## 1. Overview

The ServiceDependencies component visualizes the service architecture by grouping services by their language/technology stack. It provides a quick overview of the technology composition and status of all services.

### Use Cases
- **Tech Stack Overview**: See which languages/frameworks are used across services
- **Service Discovery**: Quickly find services by technology
- **Architecture Visualization**: Understand service groupings and relationships

---

## 2. Component Breakdown

### 2.1 Visual Structure

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│  SERVICE DEPENDENCIES                                                           │
│                                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────────────┐│
│  │  TypeScript / Node.js (3 services)                                         ││
│  │  ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐            ││
│  │  │ ● api            │ │ ● web            │ │ ● worker         │            ││
│  │  │   Express        │ │   Next.js        │ │   Node.js        │            ││
│  │  │   :3100          │ │   :3000          │ │   -              │            ││
│  │  │   4 env vars     │ │   2 env vars     │ │   1 env var      │            ││
│  │  └──────────────────┘ └──────────────────┘ └──────────────────┘            ││
│  └─────────────────────────────────────────────────────────────────────────────┘│
│                                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────────────┐│
│  │  Python (1 service)                                                        ││
│  │  ┌──────────────────┐                                                      ││
│  │  │ ○ ml-service     │                                                      ││
│  │  │   FastAPI        │                                                      ││
│  │  │   :8000          │                                                      ││
│  │  │   3 env vars     │                                                      ││
│  │  └──────────────────┘                                                      ││
│  └─────────────────────────────────────────────────────────────────────────────┘│
│                                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────────────┐│
│  │  Go (1 service)                                                            ││
│  │  ┌──────────────────┐                                                      ││
│  │  │ ● cache          │                                                      ││
│  │  │   Gin            │                                                      ││
│  │  │   :6379          │                                                      ││
│  │  │   1 env var      │                                                      ││
│  │  └──────────────────┘                                                      ││
│  └─────────────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Sub-Components

| Component | Description | Required |
|-----------|-------------|----------|
| **ServiceDependencies** | Main container component | Yes |
| **LanguageGroup** | Group of services by language | Yes |
| **ServiceDependencyCard** | Individual service card | Yes |

---

## 3. Props and Interfaces

### 3.1 Core Types

```typescript
/** Props for the main ServiceDependencies component */
interface ServiceDependenciesProps {
  /** Services data for grouping and display */
  services: Service[]
  /** Callback when a service is clicked */
  onServiceClick?: (service: Service) => void
  /** Additional class names */
  className?: string
  /** Data test ID for testing */
  'data-testid'?: string
}

/** Props for LanguageGroup */
interface LanguageGroupProps {
  /** Language name */
  language: string
  /** Services in this group */
  services: Service[]
  /** Callback when a service is clicked */
  onServiceClick?: (service: Service) => void
}

/** Props for ServiceDependencyCard */
interface ServiceDependencyCardProps {
  /** Service to display */
  service: Service
  /** Click handler */
  onClick?: () => void
}

/** Grouped services by language */
interface GroupedServices {
  [language: string]: Service[]
}
```

---

## 4. Language Grouping

### 4.1 Grouping Logic

```typescript
function groupServicesByLanguage(services: Service[]): GroupedServices {
  return services.reduce((groups, service) => {
    const language = service.language || 'Other'
    if (!groups[language]) {
      groups[language] = []
    }
    groups[language].push(service)
    return groups
  }, {} as GroupedServices)
}
```

### 4.2 Language Display Names

| Internal Value | Display Name | Icon Color |
|----------------|--------------|------------|
| typescript | TypeScript | `text-blue-500` |
| javascript | JavaScript | `text-yellow-500` |
| python | Python | `text-green-500` |
| go | Go | `text-cyan-500` |
| rust | Rust | `text-orange-500` |
| java | Java | `text-red-500` |
| csharp / c# | C# | `text-purple-500` |
| Other | Other | `text-gray-500` |

### 4.3 Group Sorting

Groups are sorted by:
1. Number of services (descending)
2. Language name (alphabetical)

---

## 5. LanguageGroup Component

### 5.1 Visual Layout

```
LANGUAGE GROUP:
┌─────────────────────────────────────────────────────────────────────────────────┐
│  ┌────────┐                                                                     │
│  │ 🟦 TS  │  TypeScript / Node.js (3 services)                                 │  Header
│  └────────┘                                                                     │
│                                                                                 │
│  ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐                │
│  │ Service Card     │ │ Service Card     │ │ Service Card     │                │  Grid
│  └──────────────────┘ └──────────────────┘ └──────────────────┘                │
└─────────────────────────────────────────────────────────────────────────────────┘

Dimensions:
- Padding: p-4
- Border radius: rounded-lg
- Grid gap: gap-3
- Card width: min 200px
```

### 5.2 Header Content

- Language icon/badge with color
- Language name (e.g., "TypeScript / Node.js")
- Service count badge (e.g., "3 services")

---

## 6. ServiceDependencyCard Component

### 6.1 Visual Layout

```
SERVICE CARD:
┌──────────────────────────────────────┐
│  ● api                               │  Name with status indicator
│                                      │
│  Express                             │  Framework
│  :3100                               │  Port
│  4 env vars                          │  Environment variable count
│                                      │
│  http://localhost:3100  →            │  URL link (if available)
└──────────────────────────────────────┘

Dimensions:
- Min width: 200px
- Padding: p-4
- Border radius: rounded-md
- Border: 1px solid border-color
```

### 6.2 Status Indicators

| Status | Icon | Color | Animation |
|--------|------|-------|-----------|
| running | ● | `text-green-500` | `animate-pulse` (subtle) |
| ready | ● | `text-green-500` | none |
| starting | ◐ | `text-yellow-500` | `animate-spin` |
| stopping | ◑ | `text-yellow-500` | none |
| stopped | ○ | `text-gray-500` | none |
| error | ⚠ | `text-red-500` | `animate-pulse` |
| not-running | ○ | `text-gray-500` | none |

### 6.3 Card Content

| Row | Content | Style |
|-----|---------|-------|
| 1 | Status indicator + Service name | `font-medium text-foreground` |
| 2 | Framework name | `text-sm text-muted-foreground` |
| 3 | Port (prefixed with `:`) | `text-sm text-muted-foreground` |
| 4 | Env var count | `text-xs text-muted-foreground` |
| 5 | URL link (if available) | `text-xs text-primary hover:underline` |

---

## 7. Interactions

### 7.1 Card Interactions

| Action | Result |
|--------|--------|
| Hover | Background highlight, subtle scale |
| Click | Trigger onServiceClick callback |
| URL Click | Open URL in new tab (stopPropagation) |

### 7.2 Group Interactions

| Action | Result |
|--------|--------|
| Collapse/Expand (optional) | Toggle group visibility |

---

## 8. Accessibility

### 8.1 WCAG 2.1 AA Compliance

| Criterion | Implementation |
|-----------|----------------|
| **1.3.1 Info & Relationships** | Groups use heading + list structure |
| **1.4.1 Use of Color** | Status has text + icon, not color alone |
| **1.4.3 Contrast (Minimum)** | All text meets 4.5:1 ratio |
| **2.1.1 Keyboard** | Cards are focusable and activatable |
| **2.4.6 Headings and Labels** | Group headings are semantic h3 |
| **4.1.2 Name, Role, Value** | Cards have role="button" or are buttons |

### 8.2 ARIA Implementation

```tsx
// Main container
<section
  aria-labelledby="dependencies-title"
  className="service-dependencies"
>
  <h2 id="dependencies-title" className="sr-only">
    Service Dependencies by Language
  </h2>
  
  // Language group
  <section aria-labelledby="group-typescript">
    <h3 id="group-typescript">TypeScript / Node.js (3 services)</h3>
    <div role="list">
      <button
        role="listitem"
        aria-label="api service - running - Express on port 3100"
      >
        {/* Card content */}
      </button>
    </div>
  </section>
</section>
```

### 8.3 Keyboard Navigation

| Key | Action |
|-----|--------|
| `Tab` | Move between service cards |
| `Enter` / `Space` | Activate card click |
| `Arrow Keys` (optional) | Navigate within group |

---

## 9. Design Tokens

### 9.1 Language Colors

| Language | Badge Color | Icon |
|----------|-------------|------|
| TypeScript | `bg-blue-500/10 text-blue-500` | TS |
| JavaScript | `bg-yellow-500/10 text-yellow-500` | JS |
| Python | `bg-green-500/10 text-green-500` | PY |
| Go | `bg-cyan-500/10 text-cyan-500` | GO |
| Rust | `bg-orange-500/10 text-orange-500` | RS |
| Java | `bg-red-500/10 text-red-500` | JV |
| C# | `bg-purple-500/10 text-purple-500` | C# |
| Other | `bg-gray-500/10 text-gray-500` | ?? |

### 9.2 Typography

| Element | Font Size | Font Weight | Line Height |
|---------|-----------|-------------|-------------|
| Group heading | `text-base` (16px) | `font-semibold` (600) | `leading-normal` |
| Service name | `text-sm` (14px) | `font-medium` (500) | `leading-normal` |
| Framework | `text-sm` (14px) | `font-normal` (400) | `leading-normal` |
| Port/Env | `text-xs` (12px) | `font-normal` (400) | `leading-normal` |
| URL | `text-xs` (12px) | `font-normal` (400) | `leading-normal` |

### 9.3 Spacing

| Property | Value | Token |
|----------|-------|-------|
| Section gap | `gap-6` | 24px |
| Group padding | `p-4` | 16px |
| Card padding | `p-4` | 16px |
| Grid gap | `gap-3` | 12px |
| Card internal gap | `gap-1` | 4px |

---

## 10. Responsive Behavior

### 10.1 Breakpoint Adaptations

```
DESKTOP (≥1024px):
┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│ Service 1    │ │ Service 2    │ │ Service 3    │ │ Service 4    │
└──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘


TABLET (640-1023px):
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│ Service 1    │ │ Service 2    │ │ Service 3    │
└──────────────┘ └──────────────┘ └──────────────┘
┌──────────────┐
│ Service 4    │
└──────────────┘


MOBILE (<640px):
┌────────────────────────────────────┐
│ Service 1                          │
└────────────────────────────────────┘
┌────────────────────────────────────┘
│ Service 2                          │
└────────────────────────────────────┘
... (stacked cards)
```

### 10.2 Grid Configuration

```typescript
const cardGridClass = cn(
  'grid gap-3',
  'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4'
)
```

---

## 11. Integration with App.tsx

### 11.1 Sidebar Update

```tsx
// In Sidebar.tsx - add to navItems array
import { GitBranch } from 'lucide-react'

const navItems = [
  { id: 'resources', label: 'Resources', icon: Activity },
  { id: 'console', label: 'Console', icon: Terminal },
  { id: 'environment', label: 'Environment', icon: Settings2 },
  { id: 'actions', label: 'Actions', icon: Zap },
  { id: 'metrics', label: 'Metrics', icon: BarChart3 },
  { id: 'dependencies', label: 'Dependencies', icon: GitBranch },  // NEW
]
```

### 11.2 App.tsx View Rendering

```tsx
// In App.tsx - add to renderContent function
if (activeView === 'dependencies') {
  return (
    <>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-foreground">Dependencies</h2>
      </div>
      <ServiceDependencies 
        services={services} 
        onServiceClick={(service) => {
          // Open service detail panel
        }}
      />
    </>
  )
}
```

---

## 12. Implementation Reference

```tsx
import * as React from 'react'
import { ExternalLink } from 'lucide-react'
import type { Service } from '@/types'

// Helper: Group services by language
export function groupServicesByLanguage(services: Service[]): Record<string, Service[]> {
  return services.reduce((groups, service) => {
    const language = normalizeLanguage(service.language || 'Other')
    if (!groups[language]) {
      groups[language] = []
    }
    groups[language].push(service)
    return groups
  }, {} as Record<string, Service[]>)
}

// Helper: Normalize language names
export function normalizeLanguage(language: string): string {
  const normalized = language.toLowerCase()
  const languageMap: Record<string, string> = {
    'ts': 'TypeScript',
    'typescript': 'TypeScript',
    'js': 'JavaScript',
    'javascript': 'JavaScript',
    'py': 'Python',
    'python': 'Python',
    'go': 'Go',
    'golang': 'Go',
    'rs': 'Rust',
    'rust': 'Rust',
    'java': 'Java',
    'c#': 'C#',
    'csharp': 'C#',
  }
  return languageMap[normalized] || language
}

// Helper: Get language badge style
export function getLanguageBadgeStyle(language: string): { bg: string; text: string; abbr: string } {
  const styles: Record<string, { bg: string; text: string; abbr: string }> = {
    'TypeScript': { bg: 'bg-blue-500/10', text: 'text-blue-500', abbr: 'TS' },
    'JavaScript': { bg: 'bg-yellow-500/10', text: 'text-yellow-500', abbr: 'JS' },
    'Python': { bg: 'bg-green-500/10', text: 'text-green-500', abbr: 'PY' },
    'Go': { bg: 'bg-cyan-500/10', text: 'text-cyan-500', abbr: 'GO' },
    'Rust': { bg: 'bg-orange-500/10', text: 'text-orange-500', abbr: 'RS' },
    'Java': { bg: 'bg-red-500/10', text: 'text-red-500', abbr: 'JV' },
    'C#': { bg: 'bg-purple-500/10', text: 'text-purple-500', abbr: 'C#' },
  }
  return styles[language] || { bg: 'bg-gray-500/10', text: 'text-gray-500', abbr: '??' }
}

// Helper: Get status indicator
export function getStatusIndicator(status?: string): { icon: string; color: string; animate: string } {
  const indicators: Record<string, { icon: string; color: string; animate: string }> = {
    'running': { icon: '●', color: 'text-green-500', animate: 'animate-pulse' },
    'ready': { icon: '●', color: 'text-green-500', animate: '' },
    'starting': { icon: '◐', color: 'text-yellow-500', animate: 'animate-spin' },
    'stopping': { icon: '◑', color: 'text-yellow-500', animate: '' },
    'stopped': { icon: '○', color: 'text-gray-500', animate: '' },
    'error': { icon: '⚠', color: 'text-red-500', animate: 'animate-pulse' },
    'not-running': { icon: '○', color: 'text-gray-500', animate: '' },
  }
  return indicators[status || 'not-running'] || indicators['not-running']
}

// Helper: Count environment variables
export function countEnvVars(service: Service): number {
  return Object.keys(service.environmentVariables || {}).length
}

// Helper: Sort groups by service count
export function sortGroupsBySize(groups: Record<string, Service[]>): [string, Service[]][] {
  return Object.entries(groups).sort((a, b) => {
    // Sort by count descending, then by name ascending
    if (b[1].length !== a[1].length) {
      return b[1].length - a[1].length
    }
    return a[0].localeCompare(b[0])
  })
}
```

---

## 13. Testing Checklist

### 13.1 Unit Tests

**Helper Functions**
- [ ] groupServicesByLanguage - groups correctly
- [ ] normalizeLanguage - maps variations
- [ ] getLanguageBadgeStyle - returns correct styles
- [ ] getStatusIndicator - returns correct indicators
- [ ] countEnvVars - counts correctly, handles undefined
- [ ] sortGroupsBySize - sorts by count then name

**ServiceDependencyCard**
- [ ] Renders service name
- [ ] Shows correct status indicator
- [ ] Displays framework
- [ ] Displays port when available
- [ ] Shows env var count
- [ ] Shows URL link when available
- [ ] Handles click event
- [ ] URL click doesn't bubble

**LanguageGroup**
- [ ] Renders group header with language
- [ ] Shows service count
- [ ] Renders all service cards

**ServiceDependencies**
- [ ] Renders all language groups
- [ ] Groups are sorted correctly
- [ ] Handles empty services
- [ ] Passes click events through

### 13.2 Accessibility Tests

- [ ] Groups have proper headings
- [ ] Cards are keyboard accessible
- [ ] Status has text alternative
- [ ] Focus indicators visible

### 13.3 Integration Tests

- [ ] Sidebar navigation works
- [ ] View renders in App.tsx
- [ ] Service click triggers callback

---

## 14. Related Components

| Component | Relationship |
|-----------|--------------|
| **ServiceCard** | Similar card pattern |
| **Badge** | Used for language badges |
| **PerformanceMetrics** | Similar grouping pattern |

---

## Appendix A: Token Quick Reference

```css
/* ServiceDependencies Tokens */
.service-dependencies {
  gap: theme('spacing.6');               /* 24px */
}

.language-group {
  padding: theme('spacing.4');           /* 16px */
  border-radius: theme('borderRadius.lg'); /* 8px */
  background: var(--card);
  border: 1px solid var(--border);
}

.service-dependency-card {
  padding: theme('spacing.4');           /* 16px */
  border-radius: theme('borderRadius.md'); /* 6px */
  border: 1px solid var(--border);
  min-width: 200px;
}

.service-dependency-card:hover {
  background: var(--accent);
  transform: scale(1.02);
}

.language-badge {
  padding: theme('spacing.1') theme('spacing.2'); /* 4px 8px */
  font-size: theme('fontSize.xs');       /* 12px */
  font-weight: theme('fontWeight.semibold'); /* 600 */
  border-radius: theme('borderRadius.md'); /* 6px */
}

/* Grid */
.card-grid {
  display: grid;
  gap: theme('spacing.3');               /* 12px */
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
}
```
