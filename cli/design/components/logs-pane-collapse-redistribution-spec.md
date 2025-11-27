# LogsPane Collapse/Expand Space Redistribution Specification

## Overview
Enhancement to the LogsPaneGrid layout system to dynamically redistribute vertical space when panes are collapsed, ensuring expanded panes fill the available space equally.

## Current Implementation

### LogsPaneGrid.tsx
```tsx
const paneHeight = `minmax(${paneMinHeight}, calc((100% - ${(rows - 1) * 16}px) / ${rows}))`

<div
  style={{
    gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
    gridAutoRows: paneHeight,
    alignItems: 'start'
  }}
>
```

### LogsPane.tsx (Collapse Handling)
```tsx
const [isCollapsed, setIsCollapsed] = useState<boolean>(() => {
  const saved = localStorage.getItem(`logs-pane-collapsed-${serviceName}`)
  return saved === 'true'
})

<div 
  className={cn("flex flex-col border-4 rounded-lg overflow-hidden", borderClass)}
  style={{ height: isCollapsed ? 'fit-content' : '100%' }}
>
```

### Current Problem
- Grid uses `gridAutoRows` with fixed row heights
- Collapsed panes use `fit-content` but other panes don't expand
- Grid doesn't know about collapsed state of individual panes

---

## Design Requirements

### Functional Requirements
1. Collapsed panes: header-only height (fit-content, ~44px)
2. Expanded panes: fill available vertical space equally
3. Works with any number of columns (1-6)
4. Works with any number of services (1-10+)
5. Grid maintains column structure (only row behavior changes)

### Transition Requirements
1. Smooth CSS transition on collapse/expand
2. Duration: 200ms (consistent with theme transitions)
3. Easing: ease-out for expansion, ease-in for collapse
4. Respect reduced motion preferences

### Performance Requirements
1. No layout thrashing (single reflow per state change)
2. Use CSS-only approach where possible
3. Avoid JavaScript layout calculations on each frame

---

## Layout Algorithm

### Architecture Decision: Shared State Pattern

The grid needs to know the collapsed state of each pane to calculate proper row heights. Two approaches:

**Option A: Lift State Up (Recommended)**
- Parent component maintains collapsed state for all panes
- Passes `isCollapsed` and `onToggleCollapse` to each pane
- Grid receives collapsed state map to calculate layout

**Option B: CSS Subgrid + Container Queries**
- More complex, limited browser support
- Not recommended for this use case

### Proposed Component Architecture

```tsx
// Parent component (Dashboard.tsx or similar)
interface PaneCollapseState {
  [serviceName: string]: boolean
}

const [collapsedPanes, setCollapsedPanes] = useState<PaneCollapseState>({})

const togglePaneCollapse = (serviceName: string) => {
  setCollapsedPanes(prev => ({
    ...prev,
    [serviceName]: !prev[serviceName]
  }))
}

<LogsPaneGrid 
  columns={columns}
  collapsedPanes={collapsedPanes}
>
  {services.map(service => (
    <LogsPane
      key={service.name}
      serviceName={service.name}
      isCollapsed={collapsedPanes[service.name] ?? false}
      onToggleCollapse={() => togglePaneCollapse(service.name)}
      // ... other props
    />
  ))}
</LogsPaneGrid>
```

---

## Grid Layout Implementation

### Updated LogsPaneGrid Component

```tsx
interface LogsPaneGridProps {
  children: ReactNode
  columns: number
  collapsedPanes?: Record<string, boolean>
}

export function LogsPaneGrid({ children, columns, collapsedPanes = {} }: LogsPaneGridProps) {
  const childArray = Children.toArray(children)
  const childCount = childArray.length
  const rows = Math.ceil(childCount / columns)
  
  // Calculate which rows have all panes collapsed
  const rowStates = useMemo(() => {
    const states: Array<'collapsed' | 'expanded' | 'mixed'> = []
    
    for (let row = 0; row < rows; row++) {
      const startIdx = row * columns
      const endIdx = Math.min(startIdx + columns, childCount)
      const rowChildren = childArray.slice(startIdx, endIdx)
      
      const collapsedCount = rowChildren.filter((child) => {
        if (isValidElement(child)) {
          const serviceName = child.props.serviceName
          return collapsedPanes[serviceName]
        }
        return false
      }).length
      
      if (collapsedCount === 0) {
        states.push('expanded')
      } else if (collapsedCount === rowChildren.length) {
        states.push('collapsed')
      } else {
        states.push('mixed')
      }
    }
    
    return states
  }, [childArray, childCount, rows, columns, collapsedPanes])
  
  // Count expanded rows for height calculation
  const expandedRowCount = rowStates.filter(s => s !== 'collapsed').length
  
  // Generate gridTemplateRows dynamically
  const gridTemplateRows = rowStates.map(state => {
    if (state === 'collapsed') {
      return 'auto' // Collapsed rows use auto (fit-content from child)
    }
    // Expanded/mixed rows share available space equally
    return '1fr'
  }).join(' ')
  
  return (
    <div
      className="grid gap-4 w-full h-full p-4 overflow-auto box-border"
      style={{
        gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
        gridTemplateRows: gridTemplateRows,
        alignItems: 'stretch'
      }}
    >
      {children}
    </div>
  )
}
```

### Key Changes from Current Implementation

| Property | Current | Proposed |
|----------|---------|----------|
| `gridAutoRows` | Fixed minmax calculation | Removed (use explicit rows) |
| `gridTemplateRows` | Not used | Dynamic based on collapse state |
| `alignItems` | `start` | `stretch` (panes fill row height) |

---

## CSS Approach

### Grid Row Sizing

```css
/* Collapsed rows: auto height (header only) */
grid-template-rows: auto 1fr 1fr;  /* Row 1 collapsed, rows 2-3 expanded */

/* All expanded: equal distribution */
grid-template-rows: 1fr 1fr 1fr;   /* All 3 rows expanded */

/* Mixed: expanded rows share space */
grid-template-rows: auto auto 1fr; /* Rows 1-2 collapsed, row 3 expanded */
```

### Transition Animation

Add transition to grid container:
```tsx
<div
  className="grid gap-4 w-full h-full p-4 overflow-auto box-border transition-[grid-template-rows] duration-200 ease-out"
  style={{ ... }}
>
```

**Note**: CSS Grid transitions for `grid-template-rows` have limited browser support. For smooth animation, use height transition on pane instead.

### Pane Height Transition (Alternative/Complementary)

```tsx
// LogsPane component
<div 
  className={cn(
    "flex flex-col border-4 rounded-lg overflow-hidden",
    "transition-[height] duration-200 ease-out",
    borderClass
  )}
  style={{ 
    height: isCollapsed ? 'fit-content' : '100%',
    minHeight: isCollapsed ? 'auto' : '150px'
  }}
>
```

---

## State Management

### Persistence Strategy

Collapsed state should persist across sessions:

```tsx
// Load from localStorage on mount
const [collapsedPanes, setCollapsedPanes] = useState<PaneCollapseState>(() => {
  try {
    const saved = localStorage.getItem('logs-pane-collapsed-states')
    return saved ? JSON.parse(saved) : {}
  } catch {
    return {}
  }
})

// Save to localStorage on change
useEffect(() => {
  localStorage.setItem('logs-pane-collapsed-states', JSON.stringify(collapsedPanes))
}, [collapsedPanes])
```

### Collapse All / Expand All Actions (Future Enhancement)

```tsx
const collapseAll = () => {
  const allCollapsed = services.reduce((acc, s) => ({ ...acc, [s.name]: true }), {})
  setCollapsedPanes(allCollapsed)
}

const expandAll = () => {
  setCollapsedPanes({})
}
```

---

## Visual Examples

### 3x2 Grid (6 Services, 2 Columns)

**All Expanded:**
```
┌─────────────────┐ ┌─────────────────┐
│ Service A       │ │ Service B       │
│                 │ │                 │  ← Row 1: 1fr
│                 │ │                 │
├─────────────────┤ ├─────────────────┤
│ Service C       │ │ Service D       │
│                 │ │                 │  ← Row 2: 1fr
│                 │ │                 │
├─────────────────┤ ├─────────────────┤
│ Service E       │ │ Service F       │
│                 │ │                 │  ← Row 3: 1fr
│                 │ │                 │
└─────────────────┘ └─────────────────┘
```

**Row 2 Collapsed (Both C and D):**
```
┌─────────────────┐ ┌─────────────────┐
│ Service A       │ │ Service B       │
│                 │ │                 │  ← Row 1: 1fr (expands)
│                 │ │                 │
│                 │ │                 │
├─────────────────┤ ├─────────────────┤
│ ▶ Service C     │ │ ▶ Service D     │  ← Row 2: auto (~44px)
├─────────────────┤ ├─────────────────┤
│ Service E       │ │ Service F       │
│                 │ │                 │  ← Row 3: 1fr (expands)
│                 │ │                 │
│                 │ │                 │
└─────────────────┘ └─────────────────┘
```

**Mixed Row (Only C Collapsed in Row 2):**
```
┌─────────────────┐ ┌─────────────────┐
│ Service A       │ │ Service B       │
│                 │ │                 │  ← Row 1: 1fr
│                 │ │                 │
├─────────────────┤ ├─────────────────┤
│ ▶ Service C     │ │ Service D       │
│                 │ │                 │  ← Row 2: 1fr (D still expanded)
│                 │ │                 │
├─────────────────┤ ├─────────────────┤
│ Service E       │ │ Service F       │
│                 │ │                 │  ← Row 3: 1fr
│                 │ │                 │
└─────────────────┘ └─────────────────┘
```

---

## Transition Specifications

### Collapse Animation

| Property | Value |
|----------|-------|
| Duration | 200ms |
| Easing | ease-out |
| Properties | height, opacity (for content fade) |

```css
/* Pane container */
.logs-pane {
  transition: height 200ms ease-out;
}

/* Log content area - fade on collapse */
.logs-pane-content {
  transition: opacity 150ms ease-out;
}

.logs-pane.collapsed .logs-pane-content {
  opacity: 0;
  pointer-events: none;
}
```

### Expand Animation

| Property | Value |
|----------|-------|
| Duration | 200ms |
| Easing | ease-out |
| Properties | height, opacity (for content fade-in) |
| Delay | Content opacity delayed 50ms for height to start |

### Reduced Motion Support

```css
@media (prefers-reduced-motion: reduce) {
  .logs-pane,
  .logs-pane-content {
    transition: none;
  }
}
```

---

## Edge Cases

### 1. All Panes Collapsed
- Grid shows all headers stacked
- No content area to distribute
- Grid height shrinks to fit headers only

```
grid-template-rows: auto auto auto;  /* All rows auto */
```

### 2. Single Column Layout
- Each pane is its own row
- Collapsed panes shrink individually
- Expanded panes share remaining space

### 3. Last Row Partially Filled
- Example: 5 services in 2 columns = 3 rows, last row has 1 pane
- Empty cell in last row doesn't affect height calculation
- Lone pane in last row still gets 1fr if expanded

### 4. Dynamic Service Add/Remove
- When services change, recalculate row states
- New services default to expanded
- Removed services cleared from collapsed state

---

## Component Interface Changes

### LogsPaneGrid Props Update

```typescript
interface LogsPaneGridProps {
  children: ReactNode
  columns: number
  collapsedPanes?: Record<string, boolean>  // NEW: map of collapsed states
}
```

### LogsPane Props Update

```typescript
interface LogsPaneProps {
  serviceName: string
  patterns: LogPattern[]
  onCopy: (logs: LogEntry[]) => void
  isPaused: boolean
  globalSearchTerm?: string
  autoScrollEnabled?: boolean
  clearAllTrigger?: number
  levelFilter?: Set<'info' | 'warning' | 'error'>
  isCollapsed?: boolean              // NEW: controlled collapse state
  onToggleCollapse?: () => void      // NEW: collapse toggle callback
}
```

### Migration Path

1. Add optional `isCollapsed` and `onToggleCollapse` props to LogsPane
2. If not provided, fall back to internal state (current behavior)
3. Update parent components to manage collapse state
4. Pass `collapsedPanes` to LogsPaneGrid
5. Remove internal collapse state from LogsPane once migration complete

---

## Accessibility Considerations

### Focus Management
- When collapsing, keep focus on collapse button (header remains visible)
- When expanding, focus stays on collapse button
- Content becomes focusable only when expanded

### Screen Reader Announcements
- Announce "collapsed" / "expanded" state change
- Use `aria-expanded` on collapse button

```tsx
<button
  onClick={onToggleCollapse}
  aria-expanded={!isCollapsed}
  aria-label={isCollapsed ? "Expand pane" : "Collapse pane"}
>
```

### Keyboard Support
- Space/Enter on header toggles collapse
- Tab skips collapsed pane content (only header focusable)

---

## Performance Considerations

### CSS-Only Height Calculation
- Grid `1fr` distribution is handled by browser layout engine
- No JavaScript measurement required
- Single reflow on state change

### Avoid Layout Thrashing
```tsx
// BAD: Measure height in JavaScript
const containerHeight = containerRef.current?.clientHeight
const rowHeight = containerHeight / expandedRowCount

// GOOD: Let CSS Grid handle it
gridTemplateRows: rowStates.map(s => s === 'collapsed' ? 'auto' : '1fr').join(' ')
```

### Transition Performance
- Use `will-change: height` sparingly (only during animation)
- Prefer `transform` animations for content if possible
- Test with 10+ panes for performance

---

## Testing Requirements

### Unit Tests
1. Grid calculates correct row states for various collapse combinations
2. Row sizing updates when collapse state changes
3. Persistence saves/loads correctly

### Visual Regression Tests
1. All panes expanded (baseline)
2. Single pane collapsed in row
3. Entire row collapsed
4. All panes collapsed
5. Animation during collapse/expand

### Accessibility Tests
1. `aria-expanded` updates correctly
2. Screen reader announces state changes
3. Focus management works during collapse/expand
4. Reduced motion disables animation

### Performance Tests
1. Collapse/expand 10 panes rapidly
2. Measure layout time (should be <16ms)
3. Memory usage stable during repeated collapse/expand

---

## Acceptance Criteria

- [ ] Collapsed panes shrink to header-only height
- [ ] Remaining expanded panes grow to fill freed space
- [ ] Space distribution is equal among expanded panes
- [ ] Works with 1-6 column grid layouts
- [ ] Works with any number of services/panes
- [ ] Transition is smooth (CSS transition)
- [ ] Performance remains acceptable (no layout thrashing)
- [ ] Collapse state persists across sessions
- [ ] aria-expanded indicates current state
- [ ] Reduced motion preference respected
- [ ] Works correctly when all panes collapsed
- [ ] Works correctly with mixed collapsed/expanded rows

---

**Version**: 1.0  
**Status**: Draft  
**Created**: 2025-11-26  
**Related**: `logs-pane-spec.md`, `multi-pane-grid-spec.md`, `spec.md` (Log Pane Visual Enhancements)
