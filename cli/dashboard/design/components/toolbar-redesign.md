# Logs Dashboard Toolbar Redesign Specification

## Document Info
- **Component**: LogsMultiPaneView Toolbar
- **Status**: Design Specification
- **Created**: 2025-11-29
- **Author**: UX Design

---

## 1. Current State Analysis

### Current Layout
```
┌──────────────────────────────────────────────────────────────────────────────────────┐
│ [Grid][Unified] │ [🔍 Search all logs...] │ [⏸] │ [▶ Start][↻ Restart][■ Stop] │ [⬇] │ [🗑] │ [⚙] │ [📥] │ [⛶] │
└──────────────────────────────────────────────────────────────────────────────────────┘
```

### Identified Problems

| Issue | Description | Impact |
|-------|-------------|--------|
| **Visual Overload** | 12+ controls displayed horizontally in a single row | Cognitive overwhelm, difficulty finding controls |
| **No Hierarchy** | All controls appear equally important | Users can't quickly identify primary actions |
| **Poor Grouping** | Related functions scattered (e.g., lifecycle controls mixed with log utilities) | Increased task completion time |
| **Inconsistent Icons** | Some buttons have text labels, others are icon-only | Inconsistent affordance and discoverability |
| **Responsive Issues** | `flex-wrap` causes unpredictable reflow on smaller screens | Layout breaks at intermediate widths |
| **No Progressive Disclosure** | Rarely-used actions compete for attention with frequent ones | Cluttered interface |

### Current Control Inventory

| Control | Type | Usage Frequency | Category |
|---------|------|-----------------|----------|
| View Mode Toggle | Segmented Control | Medium | View Configuration |
| Search Input | Text Input | High | Search & Filter |
| Pause/Resume | Icon Button | High | Log Stream Control |
| Start All | Button w/ Icon | Low | Service Lifecycle |
| Restart All | Button w/ Icon | Low | Service Lifecycle |
| Stop All | Button w/ Icon | Low | Service Lifecycle |
| Auto-Scroll | Icon Button | Medium | Log Stream Control |
| Clear All | Icon Button | Low | Log Management |
| Settings | Icon Button | Low | Utilities |
| Export All | Icon Button | Low | Utilities |
| Fullscreen | Icon Button | Low | View Configuration |

---

## 2. Proposed Grouping Strategy

### Control Categories

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  PRIMARY ZONE                    │  SECONDARY ZONE                          │
│  (Always Visible)                │  (Progressive Disclosure)                │
├─────────────────────────────────────────────────────────────────────────────┤
│  • Search                        │  • Service Lifecycle (dropdown)          │
│  • Pause/Resume                  │  • Settings                              │
│  • Auto-Scroll                   │  • Export                                │
│                                  │  • Clear All                             │
│                                  │  • Fullscreen                            │
├─────────────────────────────────────────────────────────────────────────────┤
│  VIEW CONFIGURATION (Contextual) │                                          │
│  • View Mode Toggle (Grid/Unified)                                          │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Hierarchy Principles

1. **Primary Actions** (always visible): Actions used during active log monitoring
2. **Secondary Actions** (grouped/collapsed): Actions used to configure or manage
3. **Contextual Actions** (conditional): Actions that depend on current state

---

## 3. Wireframe / Layout Description

### Desktop Layout (≥1024px)

```
┌────────────────────────────────────────────────────────────────────────────────────┐
│ ┌─────────────────┐  ┌──────────────────────────────┐  ┌─────────────────────────┐ │
│ │ VIEW MODE       │  │ PRIMARY CONTROLS             │  │ SECONDARY CONTROLS      │ │
│ │ [Grid][Unified] │  │ [🔍 Search logs...        ]  │  │ [Services ▾] [•••]      │ │
│ │                 │  │ [⏸ Pause] [⬇ Auto-scroll]   │  │                         │ │
│ └─────────────────┘  └──────────────────────────────┘  └─────────────────────────┘ │
└────────────────────────────────────────────────────────────────────────────────────┘
```

### Tablet Layout (768px - 1023px)

```
┌──────────────────────────────────────────────────────────────┐
│ [Grid][Unified]  │  [🔍 Search...        ]  │ [Services ▾] [•••] │
│                  │  [⏸] [⬇]               │                     │
└──────────────────────────────────────────────────────────────┘
```

### Mobile Layout (<768px)

```
┌────────────────────────────────────────┐
│ [🔍 Search logs...                   ] │
│ [⏸] [⬇] [Grid▾]  [Services▾]  [•••]   │
└────────────────────────────────────────┘
```

---

## 4. Component Breakdown

### 4.1 Primary Controls Group

```tsx
<PrimaryControlsGroup>
  <SearchInput 
    placeholder="Search all logs..."
    value={globalSearchTerm}
    onChange={setGlobalSearchTerm}
    aria-label="Search logs"
  />
  
  <ToggleButton
    pressed={isPaused}
    onPressedChange={setIsPaused}
    aria-label={isPaused ? "Resume log stream" : "Pause log stream"}
  >
    {isPaused ? <Play /> : <Pause />}
    <span className="sr-only md:not-sr-only">
      {isPaused ? "Resume" : "Pause"}
    </span>
  </ToggleButton>
  
  <ToggleButton
    pressed={autoScrollEnabled}
    onPressedChange={setAutoScrollEnabled}
    aria-label={autoScrollEnabled ? "Disable auto-scroll" : "Enable auto-scroll"}
  >
    <ChevronsDown />
    <span className="sr-only md:not-sr-only">Auto-scroll</span>
  </ToggleButton>
</PrimaryControlsGroup>
```

### 4.2 Services Dropdown Menu

Replace the 3 separate lifecycle buttons with a single dropdown:

```tsx
<DropdownMenu>
  <DropdownMenuTrigger asChild>
    <Button variant="outline" size="sm">
      <Server className="w-4 h-4 mr-2" />
      Services
      <ChevronDown className="w-4 h-4 ml-2" />
    </Button>
  </DropdownMenuTrigger>
  
  <DropdownMenuContent align="end">
    <DropdownMenuLabel>Service Operations</DropdownMenuLabel>
    <DropdownMenuSeparator />
    
    <DropdownMenuItem 
      onClick={startAll}
      disabled={isBulkOperationInProgress()}
    >
      <Play className="w-4 h-4 mr-2 text-success" />
      Start All Services
      <DropdownMenuShortcut>Ctrl+Shift+S</DropdownMenuShortcut>
    </DropdownMenuItem>
    
    <DropdownMenuItem 
      onClick={restartAll}
      disabled={isBulkOperationInProgress()}
    >
      <RotateCw className="w-4 h-4 mr-2 text-warning" />
      Restart All Services
      <DropdownMenuShortcut>Ctrl+Shift+R</DropdownMenuShortcut>
    </DropdownMenuItem>
    
    <DropdownMenuItem 
      onClick={stopAll}
      disabled={isBulkOperationInProgress()}
      className="text-destructive focus:text-destructive"
    >
      <Square className="w-4 h-4 mr-2" />
      Stop All Services
      <DropdownMenuShortcut>Ctrl+Shift+X</DropdownMenuShortcut>
    </DropdownMenuItem>
  </DropdownMenuContent>
</DropdownMenu>
```

### 4.3 Overflow Menu (More Actions)

Group infrequently used utilities in an overflow menu:

```tsx
<DropdownMenu>
  <DropdownMenuTrigger asChild>
    <Button variant="ghost" size="sm" aria-label="More actions">
      <MoreHorizontal className="w-4 h-4" />
    </Button>
  </DropdownMenuTrigger>
  
  <DropdownMenuContent align="end">
    <DropdownMenuGroup>
      <DropdownMenuLabel>View</DropdownMenuLabel>
      <DropdownMenuItem onClick={toggleFullscreen}>
        {isFullscreen ? <Minimize2 /> : <Maximize2 />}
        {isFullscreen ? "Exit Fullscreen" : "Enter Fullscreen"}
        <DropdownMenuShortcut>F11</DropdownMenuShortcut>
      </DropdownMenuItem>
    </DropdownMenuGroup>
    
    <DropdownMenuSeparator />
    
    <DropdownMenuGroup>
      <DropdownMenuLabel>Log Actions</DropdownMenuLabel>
      <DropdownMenuItem onClick={handleClearAll}>
        <Trash2 className="w-4 h-4 mr-2" />
        Clear All Logs
      </DropdownMenuItem>
      <DropdownMenuItem onClick={handleExportAll}>
        <Download className="w-4 h-4 mr-2" />
        Export All Logs
      </DropdownMenuItem>
    </DropdownMenuGroup>
    
    <DropdownMenuSeparator />
    
    <DropdownMenuItem onClick={() => setIsSettingsOpen(true)}>
      <Settings className="w-4 h-4 mr-2" />
      Settings
      <DropdownMenuShortcut>Ctrl+,</DropdownMenuShortcut>
    </DropdownMenuItem>
  </DropdownMenuContent>
</DropdownMenu>
```

### 4.4 View Mode Toggle

Keep as segmented control but make responsive:

```tsx
<ViewModeToggle>
  {/* Desktop: Full segmented control */}
  <div className="hidden md:flex gap-1 border rounded-lg p-1">
    <Button variant={viewMode === 'grid' ? 'default' : 'ghost'} size="sm">
      <LayoutGrid className="w-4 h-4 mr-2" />
      Grid
    </Button>
    <Button variant={viewMode === 'unified' ? 'default' : 'ghost'} size="sm">
      <List className="w-4 h-4 mr-2" />
      Unified
    </Button>
  </div>
  
  {/* Mobile: Dropdown */}
  <div className="md:hidden">
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm">
          {viewMode === 'grid' ? <LayoutGrid /> : <List />}
          <ChevronDown className="w-4 h-4 ml-1" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem onClick={() => setViewMode('grid')}>
          <LayoutGrid className="w-4 h-4 mr-2" />
          Grid View
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setViewMode('unified')}>
          <List className="w-4 h-4 mr-2" />
          Unified View
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  </div>
</ViewModeToggle>
```

---

## 5. Interaction Patterns

### 5.1 Keyboard Navigation

| Key | Action | Scope |
|-----|--------|-------|
| `Tab` | Navigate between control groups | Toolbar |
| `Arrow Left/Right` | Navigate within segmented controls | View Mode Toggle |
| `Space/Enter` | Activate focused control | All buttons |
| `Escape` | Close open dropdown, exit fullscreen | Dropdowns, Fullscreen |
| `Ctrl+Shift+L` | Toggle view mode | Global |
| `Space` | Toggle pause (when not in input) | Global |
| `F11` | Toggle fullscreen | Global |

### 5.2 Focus Management

```
Tab Order:
1. View Mode Toggle (segmented control or dropdown trigger)
2. Search Input
3. Pause/Resume Button
4. Auto-Scroll Button
5. Services Dropdown Trigger
6. More Actions Dropdown Trigger
```

### 5.3 State Indicators

| State | Visual Indicator | Location |
|-------|------------------|----------|
| Paused | Yellow badge "⏸ Paused" | Right side of toolbar |
| Auto-scroll Off | Blue badge "🛑 Auto-scroll stopped" | Right side of toolbar |
| Bulk Operation In Progress | Spinner on dropdown trigger + disabled items | Services dropdown |
| Active View Mode | Filled/highlighted button | View Mode Toggle |

### 5.4 Dropdown Behavior

- **Trigger**: Click or `Enter`/`Space` on trigger button
- **Close**: Click outside, `Escape`, or selecting an item
- **Focus**: First focusable item when opened
- **Scroll**: If content overflows, allow internal scroll with visible scroll indicators

---

## 6. Accessibility Considerations

### 6.1 ARIA Attributes

```tsx
// Toolbar container
<div role="toolbar" aria-label="Log viewer controls">

// View mode toggle (segmented control)
<div role="group" aria-label="View mode">
  <Button aria-pressed={viewMode === 'grid'}>Grid</Button>
  <Button aria-pressed={viewMode === 'unified'}>Unified</Button>
</div>

// Dropdown menus
<DropdownMenuTrigger aria-haspopup="menu" aria-expanded={isOpen}>
<DropdownMenuContent role="menu">
<DropdownMenuItem role="menuitem">

// Toggle buttons
<Button aria-pressed={isPaused} aria-label="Pause log stream">
```

### 6.2 Screen Reader Announcements

| Action | Announcement |
|--------|--------------|
| Pause logs | "Log stream paused" |
| Resume logs | "Log stream resumed" |
| Clear all logs | "All logs cleared" |
| Start all services | "Starting all services" |
| Service operation complete | "[Operation] complete: [X] services affected" |

### 6.3 Color & Contrast

- All icon buttons must have minimum 4.5:1 contrast ratio
- State changes (paused, auto-scroll off) use color + icon/text for non-color-only indication
- Focus rings visible in both light and dark themes

### 6.4 Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  .animate-spin {
    animation: none;
  }
  
  /* Instant transitions instead of animated */
  .transition-all {
    transition-duration: 0ms;
  }
}
```

---

## 7. Responsive Behavior

### 7.1 Breakpoint Strategy

| Breakpoint | Width | Behavior |
|------------|-------|----------|
| Mobile | <640px | Search full width, all controls below, view mode as dropdown |
| Tablet | 640-1023px | Search inline, some controls collapse to icons |
| Desktop | ≥1024px | Full layout with all labels visible |

### 7.2 Progressive Enhancement

```
Desktop (Full):
[Grid][Unified] | [🔍 Search...] | [⏸ Pause] [⬇ Auto-scroll] | [Services ▾] | [•••]

Tablet (Condensed):
[Grid][Unified] | [🔍 Search...] | [⏸][⬇] | [Services ▾] | [•••]

Mobile (Stacked):
[🔍 Search...                                    ]
[View ▾] [⏸][⬇] [Services ▾] [•••]
```

### 7.3 Container Queries (Future Enhancement)

```css
@container toolbar (width < 600px) {
  .view-mode-toggle {
    display: none;
  }
  .view-mode-dropdown {
    display: flex;
  }
}
```

---

## 8. Implementation Recommendations

### 8.1 New Components Required

| Component | Source | Notes |
|-----------|--------|-------|
| `DropdownMenu` | shadcn/ui | Already in project |
| `ToggleButton` | Custom or Radix | For pressed state management |
| `Toolbar` | Custom wrapper | For ARIA role and keyboard nav |

### 8.2 Migration Steps

1. **Phase 1**: Introduce Services dropdown, keep existing buttons for comparison
2. **Phase 2**: Add overflow menu for secondary actions
3. **Phase 3**: Implement responsive breakpoints
4. **Phase 4**: Remove legacy inline buttons
5. **Phase 5**: Add keyboard navigation enhancements

### 8.3 Testing Checklist

- [ ] All controls accessible via keyboard
- [ ] Screen reader announces state changes
- [ ] Dropdowns open/close correctly
- [ ] Focus returns to trigger after dropdown closes
- [ ] Responsive layout works at all breakpoints
- [ ] Keyboard shortcuts still function
- [ ] Dark/light theme contrast passes WCAG AA

---

## 9. Visual Summary

### Before
```
┌──────────────────────────────────────────────────────────────────────────────────────────┐
│ [Grid][Unified] [🔍 Search...] [⏸] [▶Start][↻Restart][■Stop] [⬇] [🗑] [⚙] [📥] [⛶]       │
└──────────────────────────────────────────────────────────────────────────────────────────┘
12 visible controls, no hierarchy, overwhelming
```

### After
```
┌────────────────────────────────────────────────────────────────────────────────┐
│ [Grid][Unified]  │  [🔍 Search logs...]  │  [⏸ Pause][⬇ Scroll]  │  [Services ▾]  [•••] │
└────────────────────────────────────────────────────────────────────────────────┘
6 visible controls, clear hierarchy, progressive disclosure
```

**Reduction**: 12 controls → 6 visible (50% reduction in visual complexity)

---

## 10. Success Metrics

| Metric | Current (Est.) | Target |
|--------|----------------|--------|
| Time to find pause button | 3-5 seconds | <1 second |
| Controls visible at once | 12 | 6 |
| Keyboard nav tab stops | 12 | 6 |
| Mobile usability score | Poor | Good |
| WCAG compliance | Partial | AA |

---

## Appendix: Component Props Reference

### ToolbarRedesign Props

```tsx
interface ToolbarProps {
  // View state
  viewMode: 'grid' | 'unified'
  onViewModeChange: (mode: 'grid' | 'unified') => void
  isFullscreen: boolean
  onFullscreenChange: (isFullscreen: boolean) => void
  
  // Log stream state
  isPaused: boolean
  onPauseChange: (isPaused: boolean) => void
  autoScrollEnabled: boolean
  onAutoScrollChange: (enabled: boolean) => void
  
  // Search
  searchTerm: string
  onSearchChange: (term: string) => void
  
  // Actions
  onClearAll: () => void
  onExportAll: () => void
  onOpenSettings: () => void
  
  // Service operations
  onStartAll: () => Promise<void>
  onStopAll: () => Promise<void>
  onRestartAll: () => Promise<void>
  isBulkOperationInProgress: () => boolean
}
```
