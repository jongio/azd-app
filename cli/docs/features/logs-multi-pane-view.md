# Multi-Pane Logs Dashboard

## Overview

Enhance the logs dashboard with a multi-pane view where each process's logs are displayed in a separate, isolated pane. This allows users to monitor all services simultaneously while maintaining visual separation and status indicators.

## User Story

As a developer running multiple services, I want to see each process's logs in isolation so that I can quickly identify issues in specific services and correlate behavior across processes.

## Functional Requirements

### 1. Multi-Pane Layout

**View Configuration:**
- Display logs for all selected services in separate, side-by-side or stacked panes
- Each pane represents one service
- Support dynamic pane arrangement (grid layout, configurable columns)
- Each pane independently scrollable
- Auto-layout: services should use a 2-column grid layout by default, expanding to 3-4 columns on larger screens

**Pane Components:**
- **Header**: Service name, status badge, log count
- **Body**: Scrollable log container (max-height with overflow-y: auto)
- **Footer**: Search/filter controls (per-pane), export, clear logs
- **Border/Visual Indicator**: Dynamic color-coded border based on service status

### 2. Status Indicators

**Service Status States (based on log analysis):**

| State | Indicator | Color | Animation | Meaning |
|-------|-----------|-------|-----------|---------|
| **Error** | Solid border + blinking | Red (#EF4444 or similar) | Pulse/blink (500ms) | Service has logged an actual error |
| **Warning** | Solid border (no animation) | Yellow/Amber (#FBBF24) | None | Service has warnings but no errors |
| **Info** | Solid border (no animation) | White/Gray (#E5E7EB) | None | Service is running normally, info/debug logs only |

**Border Styling:**
- Thick border (3-4px) to make status immediately visible
- Applied to pane container
- Status determined by the most severe log level detected

### 3. False Positive / False Negative Detection

**Problem Definition:**
- False Positive: Log line contains "error" keyword but is not actually an error (e.g., "0 errors found")
- False Negative: Should flag something as error that isn't being caught

**Solution - Pattern-Based Filtering with Global Configuration:**

Allow users to mark individual log lines AND configure global patterns at app and user levels:

**Per-Line Markers (Session-Only):**
- Clickable checkmark icon on each log line to mark false positive
- Clickable exclamation icon to force-flag as error
- When marked: lines de-emphasized or highlighted accordingly
- Marked lines excluded/included from status calculation
- Persist only during current session

**Global False Positive Pattern Configuration:**

**Two-Level Storage Architecture:**

1. **User-Level Patterns** (`~/.azure/logs-dashboard/patterns.json`)
   - Applies to all projects user runs
   - User can edit, add, delete patterns via dashboard UI
   - Examples: "0 errors", "error rate: 0%", "Connection error: retrying" 

2. **App-Level Patterns** (`.azure/logs-dashboard/patterns.json`)
   - Project-specific overrides for team
   - Can be committed to repo for consistency across team
   - Higher priority than user patterns
   - Useful for known non-error strings in project logs

**Pattern File Format (JSON):**
```json
{
  "version": "1.0",
  "patterns": [
    {
      "id": "pattern-001",
      "name": "Zero Errors",
      "regex": "^.*\\b0\\s+errors?\\b.*$",
      "description": "Matches '0 errors' or '0 error' in any context",
      "enabled": true,
      "createdAt": "2025-11-23T10:00:00Z",
      "source": "user|app"
    }
  ]
}
```

**Dashboard UI for Pattern Management:**
- Settings/gear icon in logs view toolbar
- Modal with two tabs: "User Patterns" and "Project Patterns"
- Each tab shows list of patterns with:
  - Pattern name and description
  - Regex preview (human-readable explanation)
  - Toggle to enable/disable
  - Edit button (inline regex editor)
  - Delete button
  - "Add New Pattern" button
- Quick suggestion feature: when user marks line as false positive, offer to create pattern from it
- Test pattern feature: input sample log line, see if it matches
- Import/Export patterns (share with team via JSON)

### 4. Grid & Pane Layout Configuration

**Dynamic Sizing:**
- **Column Slider**: Horizontal slider control (1-6 columns) to adjust grid layout
  - 1 col: Vertical stack (mobile)
  - 2 col: Default (tablet)
  - 3-4 col: Desktop
  - 5-6 col: Ultra-wide monitors
  - Real-time preview as user adjusts
  - Persists to `~/.azure/logs-dashboard/preferences.json`

- **Row Height Slider**: Adjust individual pane heights
  - Min: 300px, Max: 800px
  - Default: 500px
  - Real-time preview
  - Applies to all panes uniformly (for now)

**Pane Size Controls:**
- Add/remove panes on the fly
- Responsive info: show current viewport width and recommended column count
- Mobile detection: auto-collapse to 1 column on screens <600px

### 5. View Switcher: Grid vs. Unified

**Two Log Display Modes:**

**A. Grid View (Current/Proposed)**
- Each service in separate pane
- Status indicators per pane
- Best for: Multi-service monitoring, quick issue isolation

**B. Unified/Single View**
- All services combined in single scrollable log stream
- Service name color-coded per line (like current LogsView)
- Global status badge (highest severity from all services)
- Timestamp-ordered across all services
- Best for: Tracing interactions between services, full timeline

**View Switcher Toggle:**
- Button in toolbar: "Grid" vs "Unified"
- Keyboard shortcut: `Ctrl+Shift+L` (toggle modes)
- Preserve scroll position when switching (best effort)
- Persist selected view to preferences.json

**Unified View Features:**
- Search applies to all services
- Global pause/resume
- Can filter by service from dropdown
- Export combines all visible services
- Single auto-scroll (not per-pane)
- Status pattern application (false pos/neg patterns apply globally)

### 6. Copy Functionality

**Copy Buttons:**
- **Per-Pane Copy** (top-right corner of pane header):
  - Copy entire pane logs (currently visible or all?)
  - Copy as plain text to clipboard
  - Show toast notification: "Copied 250 lines to clipboard"
  - Visual feedback: button highlight/animation

- **Per-Line Copy**:
  - Right-click on log line → "Copy line"
  - Or small copy icon on hover
  - Copy just that single line

- **Multi-Pane Copy** (global toolbar):
  - Copy all visible panes concatenated (with separator)
  - Option: separate files or single file with delimiters
  - Format: `\n=== SERVICE_NAME ===\n`

**Copy Formats:**
- Plain text (default)
- JSON (each line as object with service, message, timestamp, level)
- Markdown (code blocks per service)
- CSV (service, timestamp, level, message)

### 7. Log Filtering & Search

**Per-Pane Controls:**
- Search box (live filter, match highlighting)
- Log level filter (errors, warnings, info, debug)
- Exclude false positives toggle (hides matched pattern lines)
- Clear logs button

**Global Controls (above all panes):**
- Select services to display (checkboxes)
- Apply search across all visible panes simultaneously
- View mode switcher (Grid ↔ Unified)
- Column count slider (1-6 columns)
- Row height slider (300-800px)

### 8. Interactions

**Pause/Resume:**
- Global pause (pauses all panes/streams simultaneously)
- Show indicator when paused
- Keyboard shortcut: `Space` to toggle

**Export:**
- Per-pane export (single service logs)
- Multi-pane export (all visible services, format options)
- Format options: Plain text, JSON, Markdown, CSV
- Auto-naming: `logs-{service}-{timestamp}.txt`

**Auto-Scroll:**
- Each pane has independent auto-scroll behavior
- Pause when user manually scrolls in a pane
- Resume when pane returns to bottom
- Indicator when pane is scrolled up

### 9. User Preferences Persistence

**Preferences File** (`~/.azure/logs-dashboard/preferences.json`)
```json
{
  "version": "1.0",
  "ui": {
    "gridColumns": 2,
    "paneHeight": 500,
    "viewMode": "grid",
    "selectedServices": ["web", "api", "db"]
  },
  "behavior": {
    "autoScroll": true,
    "pauseOnScroll": true,
    "timestampFormat": "hh:mm:ss.sss"
  },
  "copy": {
    "defaultFormat": "plaintext",
    "includeTimestamp": true,
    "includeService": true
  }
}
```

**App-Level Config** (`.azure/logs-dashboard/config.json`)
- Team-shared settings (patterns, services to display by default)
- Can be committed to repo
- Overrides user preferences for specific keys

## Acceptance Criteria

**Core Features:**
- [ ] Multi-pane layout renders correctly for all services (1-6 columns configurable)
- [ ] Each pane displays logs independently without cross-contamination
- [ ] Error status shows red border with blinking animation
- [ ] Warning status shows yellow/amber border without animation
- [ ] Info status shows white/gray border
- [ ] Auto-scroll pauses when user scrolls and resumes at bottom
- [ ] Pane layout is responsive (auto-adjust 1-6 columns based on slider)

**View Modes:**
- [ ] Grid view shows multiple services in panes (default)
- [ ] Unified view shows all services in single scrollable stream
- [ ] View switcher toggles between modes
- [ ] View preference persists to `~/.azure/logs-dashboard/preferences.json`

**Layout Configuration:**
- [ ] Column slider works (1-6 columns, real-time preview)
- [ ] Row height slider works (300-800px, uniform across panes)
- [ ] Layout preference persists to preferences.json
- [ ] Mobile detection auto-collapses to 1 column on <600px

**Copy Functionality:**
- [ ] Per-pane copy button works (top-right of pane header)
- [ ] Per-line copy on right-click or hover icon
- [ ] Multi-pane copy exports all visible services
- [ ] Copy format options: plaintext, JSON, Markdown, CSV
- [ ] Clipboard notification shows line count copied

**False Positive/Negative Markers (Per-Line):**
- [ ] Users can mark log lines as false positives (checkmark icon)
- [ ] Users can mark log lines as false negatives (exclamation icon)
- [ ] Marked lines persist during session
- [ ] Marked false positives excluded from status calculation
- [ ] Marked false negatives included in status calculation

**Global False Positive Pattern Configuration:**
- [ ] User patterns stored in `~/.azure/logs-dashboard/patterns.json`
- [ ] App patterns stored in `.azure/logs-dashboard/patterns.json`
- [ ] Settings modal allows add/edit/delete patterns
- [ ] Patterns apply to automatic status detection
- [ ] "Quick pattern" suggestion when user marks false positive
- [ ] Pattern test feature (input log line, see if matches)
- [ ] Pattern import/export for team sharing
- [ ] Exclude false positives toggle in UI

**Per-Pane Controls:**
- [ ] Per-pane search works independently
- [ ] Per-pane log level filtering works
- [ ] Per-pane clear/export works
- [ ] Per-pane scroll position independent

**Global Controls:**
- [ ] Service selector shows all services with checkboxes
- [ ] Service selection hides/shows panes dynamically
- [ ] Global pause affects all panes
- [ ] Global search applies to all visible panes
- [ ] Preferences persists to `~/.azure/logs-dashboard/preferences.json`

**UI/UX:**
- [ ] All controls accessible via keyboard (Tab, Enter, arrow keys)
- [ ] Keyboard shortcut: Space = toggle pause
- [ ] Keyboard shortcut: Ctrl+Shift+L = toggle view mode
- [ ] ARIA labels on all interactive elements
- [ ] Screen reader announces status changes
- [ ] Color contrast meets WCAG AA
- [ ] Responsive design works on mobile/tablet/desktop

**Configuration & Persistence:**
- [ ] User preferences file has correct schema (version, ui, behavior, copy)
- [ ] App config file can override user preferences
- [ ] Settings survive application reload
- [ ] Export includes configured headers/timestamps per user preference

## Technical Implementation Notes

### Component Structure

```
LogsMultiPaneView (NEW - main container)
├── GlobalToolbar
│   ├── ViewModeToggle (Grid ↔ Unified)
│   ├── ServiceSelector (checkboxes)
│   ├── ColumnSlider (1-6 cols)
│   ├── RowHeightSlider (300-800px)
│   ├── PauseButton
│   ├── GlobalSearch
│   ├── SettingsButton (→ modal)
│   └── ExportAllButton
├── LogsGridView (conditional render)
│   ├── LogsPaneGrid (CSS Grid layout)
│   │   └── LogsPane (repeated for each service)
│   │       ├── PaneHeader (service name, status badge, copy button)
│   │       ├── LogsContainer (scrollable)
│   │       └── PaneFooter (search, filters, controls)
│   └── SettingsModal
│       ├── UserPatternsList
│       ├── AppPatternsList
│       └── PreferencesEditor
└── LogsUnifiedView (conditional render)
    ├── UnifiedHeader (global status badge)
    ├── ServiceFilter (dropdown)
    ├── UnifiedLogsContainer (single scroll)
    └── UnifiedFooter
```

### State Management

**Per-Pane:**
- Logs array (independent)
- Search term
- Is scrolled up
- Marked false positives (Set<logIndex>)
- Marked false negatives (Set<logIndex>)
- Scroll position

**Global:**
- Selected services (Set<serviceName>)
- Global pause state
- View mode (grid | unified)
- Grid columns (1-6)
- Pane height (300-800)
- Selected patterns (user + app combined)
- False positive patterns (from files)

**Persisted to Files:**
- `~/.azure/logs-dashboard/preferences.json` (user level)
- `.azure/logs-dashboard/patterns.json` (app level, optional)
- `.azure/logs-dashboard/config.json` (app level, optional)

### Styling

- **Grid Layout**: CSS Grid with dynamic columns (grid-template-columns: repeat(var(--pane-columns), 1fr))
- **Responsive Breakpoints**: 
  - Mobile (<600px): 1 column (override slider)
  - Tablet (600-1024px): 1-2 columns default
  - Desktop (>1024px): 2-6 columns
- **Pane Height**: CSS variable --pane-height-px (300-800px)
- **Status Borders**: 
  - Error: 4px solid #EF4444 + pulse animation
  - Warning: 4px solid #FBBF24
  - Info: 4px solid #E5E7EB
- **Animations**:
  - Error pulse: 500ms on/off, runs indefinitely
  - Copy button highlight: 200ms highlight flash
- **Tailwind**: Use CSS Grid, Gap, Padding utilities
- **Marked Lines**: 
  - False positive: opacity-50 line-through
  - False negative: bg-red-100 dark:bg-red-900
- **Unified View**: Single scrollable container, all logs timestamp-ordered

### WebSocket/Streaming

- Multiple concurrent WebSocket connections (one per visible service) OR
- Single aggregated stream with service filtering
- Buffer strategy: each pane maintains max 1000 logs independently

## Out of Scope

- Persistent log storage (session-only for now; could add to .azure later)
- Custom pane sizing or reordering (grid is uniform for v1)
- Log parsing plugins or custom formatters
- Advanced analytics or aggregation
- Filtering by specific log fields beyond level/service
- Dark mode (use existing app theme)
- Log rotation or archival
- Remote log streaming from cloud
- Performance profiling within dashboard

## Performance Considerations

- Lazy render panes when scrolled off-screen (if many services)
- Debounce search/filter operations
- Limit DOM size per pane (max 1000 logs visible + buffer)
- Virtual scrolling if panes have >500 logs

## Accessibility

- ARIA labels for each pane header
- Keyboard navigation between panes (arrow keys)
- Screen reader announces status changes
- High contrast for color-coded borders (maintain WCAG AA compliance)
- Marked false positives/negatives announced to screen readers
