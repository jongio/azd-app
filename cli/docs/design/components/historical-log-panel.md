# Historical Log Query Panel

Component design spec for Azure historical log queries.

## Overview

Panel for querying historical Azure logs with time range selection, KQL query input, pagination, and export functionality. Appears in Azure mode alongside the real-time log stream.

## Component Structure

```
HistoricalLogPanel
├── TimeRangeSelector
│   ├── PresetButtons (15m, 1h, 6h, 24h)
│   └── CustomRangePicker (date/time inputs)
├── QuerySection (collapsible)
│   ├── KQLInput (monaco-editor or textarea)
│   └── RunQueryButton
├── ResultsSection
│   ├── ResultsHeader (count, timespan)
│   ├── LogEntries (virtualized list)
│   └── LoadMoreButton (pagination)
└── ExportSection
    ├── FormatSelector (JSON, Text, CSV)
    └── ExportButton
```

## Props Interface

```typescript
interface HistoricalLogPanelProps {
  /** Service name to query logs for */
  serviceName: string
  /** Whether panel is visible */
  isOpen: boolean
  /** Close panel callback */
  onClose: () => void
  /** Default time range preset */
  defaultTimeRange?: TimeRangePreset
  /** Whether Azure is connected */
  azureConnected: boolean
}

type TimeRangePreset = '15m' | '1h' | '6h' | '24h' | 'custom'

interface TimeRange {
  preset: TimeRangePreset
  start?: Date
  end?: Date
}

interface HistoricalLogQuery {
  serviceName: string
  timeRange: TimeRange
  customKql?: string
  limit: number
  offset: number
}
```

## Layout

### Panel Position
- Slide-in panel from right side (480px width)
- Or modal dialog (600px max-width) for larger queries
- Decision: **Slide-in panel** - matches existing ServiceDetailPanel pattern

### Sections

#### 1. Header
```
┌─────────────────────────────────────────────────┐
│ Historical Logs: {serviceName}          [X]     │
└─────────────────────────────────────────────────┘
```
- Service name in title
- Close button (X) top-right
- Keyboard: Escape to close

#### 2. Time Range Selector
```
┌─────────────────────────────────────────────────┐
│ Time Range                                      │
│ ┌──────┐ ┌────┐ ┌────┐ ┌─────┐ ┌────────┐     │
│ │ 15m  │ │ 1h │ │ 6h │ │ 24h │ │ Custom │     │
│ └──────┘ └────┘ └────┘ └─────┘ └────────┘     │
│                                                 │
│ [Custom picker shown when Custom selected]      │
│ Start: [Date] [Time]  End: [Date] [Time]       │
└─────────────────────────────────────────────────┘
```
- Preset buttons as segmented control (radio group)
- Default: 1h
- Custom: shows datetime pickers inline
- Max range: 7 days

#### 3. Advanced Query (Collapsible)
```
┌─────────────────────────────────────────────────┐
│ ▼ Advanced Query                                │
├─────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────┐ │
│ │ ContainerAppConsoleLogs_CL                  │ │
│ │ | where ContainerAppName_s == "api"         │ │
│ │ | where Log_s contains "error"              │ │
│ │                                             │ │
│ └─────────────────────────────────────────────┘ │
│                                                 │
│ [Run Query]                          [Reset]   │
└─────────────────────────────────────────────────┘
```
- Collapsed by default (show "▶ Advanced Query")
- Textarea with monospace font (code-like)
- Placeholder: shows default KQL for service type
- Reset button: restores default query
- Run Query: executes custom KQL

#### 4. Results Section
```
┌─────────────────────────────────────────────────┐
│ Results: 156 logs (1h)           [Export ▼]    │
├─────────────────────────────────────────────────┤
│ 10:32:15 [INFO]  Request completed 200ms       │
│ 10:32:14 [WARN]  Slow query detected           │
│ 10:32:12 [INFO]  Incoming request /api/users   │
│ ...                                             │
├─────────────────────────────────────────────────┤
│               [Load More (100)]                 │
└─────────────────────────────────────────────────┘
```
- Results count + timespan in header
- Log entries same format as LogsView
- Load more button with count indicator
- Virtualized list for performance

#### 5. Export Dropdown
```
┌──────────────┐
│ Export as... │
├──────────────┤
│ JSON         │
│ Plain Text   │
│ CSV          │
└──────────────┘
```
- Dropdown menu with format options
- Downloads file on selection

## States

### Loading
- Spinner in results area
- "Querying Azure Log Analytics..." text
- Time range and query inputs disabled during load

### Empty Results
```
┌─────────────────────────────────────────────────┐
│         📭 No logs found                        │
│                                                 │
│   No logs matching your query in the           │
│   selected time range.                         │
│                                                 │
│   Try expanding the time range or              │
│   adjusting your query filters.                │
└─────────────────────────────────────────────────┘
```

### Error
```
┌─────────────────────────────────────────────────┐
│         ⚠️ Query failed                         │
│                                                 │
│   {error message}                              │
│                                                 │
│   [Retry]                                      │
└─────────────────────────────────────────────────┘
```

### Azure Not Connected
- Panel disabled with overlay message
- "Connect to Azure to view historical logs"
- Link to Azure connection status

## Interactions

### Time Range Selection
1. Click preset button → immediate query execution
2. Click Custom → show datetime pickers
3. Select custom range → click "Apply" to query

### Query Execution
1. Changing time range auto-runs default query
2. Custom KQL requires explicit "Run Query" click
3. Debounce rapid changes (300ms)

### Pagination
- Initial load: 100 entries
- Load More: +100 entries per click
- Server returns `hasMore` flag
- Stop showing button when no more results

### Export
1. Click Export dropdown
2. Select format
3. Downloads file: `{serviceName}-logs-{timestamp}.{ext}`
4. Toast notification on success

## Keyboard Navigation

| Key | Action |
|-----|--------|
| Escape | Close panel |
| Enter (in KQL input) | Run query (with Cmd/Ctrl) |
| Tab | Navigate through controls |
| Arrow keys | Navigate time presets |

## Accessibility

- ARIA labels on all interactive elements
- Focus trapped within panel when open
- Announce query results to screen readers
- Time inputs use native datetime-local for mobile support

## Responsive Behavior

### Desktop (>768px)
- Full slide-in panel (480px width)
- All sections visible

### Tablet (768px)
- Panel width 100%, max 480px
- Slightly reduced padding

### Mobile (<640px)
- Full-screen modal instead of slide-in
- Stacked time range buttons (2x2 grid + Custom)
- Collapsible sections default closed

## Visual Design

### Colors (following existing theme)
- Background: slate-50 dark:slate-900
- Border: slate-200 dark:slate-700
- Selected preset: cyan-500 bg
- Query textarea: slate-100 dark:slate-800
- Results: same as LogsView entries

### Typography
- Header: text-lg font-semibold
- Labels: text-xs font-medium text-slate-500
- Log entries: text-sm font-mono
- Buttons: text-sm font-medium

### Spacing
- Panel padding: p-4
- Section gaps: space-y-4
- Button group gaps: gap-1

## API Integration

### Endpoint
```
POST /api/azure/logs/query
{
  "service": "api",
  "timespan": "PT1H",  // ISO 8601 duration
  "query": "ContainerAppConsoleLogs_CL | ...",
  "limit": 100,
  "offset": 0
}
```

### Response
```
{
  "logs": [...],
  "total": 156,
  "hasMore": true,
  "executionTime": 1234  // ms
}
```

## Component Files

```
cli/dashboard/src/components/
├── HistoricalLogPanel.tsx      # Main panel component
├── TimeRangeSelector.tsx       # Time range picker
├── KqlQueryInput.tsx           # KQL textarea with reset
└── hooks/
    └── useHistoricalLogs.ts    # Query execution hook
```

## Dependencies

- Existing: cn, LogEntry type, LogsView styling
- New: datetime-local inputs (native HTML5)
- Optional: monaco-editor for KQL syntax highlighting (defer to Phase 3)

## Test Scenarios

1. Select preset → query executes → results display
2. Select Custom → pick range → apply → query executes
3. Expand Advanced → modify KQL → run → results display
4. Click Load More → additional results append
5. Export JSON → file downloads
6. Query fails → error state shows → retry works
7. No results → empty state displays
8. Panel keyboard navigation works
9. Screen reader announces state changes
