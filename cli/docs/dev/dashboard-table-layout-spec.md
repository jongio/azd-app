# Dashboard Table Layout Specification

## Overview
Add a table layout view to the dashboard that mirrors the Aspire Dashboard's table view, allowing users to toggle between the current card layout and a compact table layout.

## User Interface

### View Toggle
**Location**: Below the "Services/Logs" tabs, above the services content area

**Design**:
- Two toggle buttons: "Cards" and "Table"
- Styled similarly to the main tabs but smaller and right-aligned
- Active state uses the same gradient as main tabs
- Persisted in localStorage as `dashboard-view-preference`

```tsx
<div className="flex items-center justify-end mb-6 gap-2">
  <Button 
    variant={viewMode === 'cards' ? 'default' : 'ghost'} 
    size="sm"
    onClick={() => setViewMode('cards')}
  >
    <LayoutGrid className="w-4 h-4 mr-2" />
    Cards
  </Button>
  <Button 
    variant={viewMode === 'table' ? 'default' : 'ghost'} 
    size="sm"
    onClick={() => setViewMode('table')}
  >
    <Table className="w-4 h-4 mr-2" />
    Table
  </Button>
</div>
```

### Table Layout Structure

#### Table Columns (left to right):
1. **Name** (200px min-width)
   - Icon: Service type icon (Server icon with color based on status)
   - Service name in bold
   
2. **State** (120px)
   - Green dot indicator for running/healthy
   - Yellow dot for starting
   - Red dot for error/unhealthy
   - Gray dot for stopped
   - Status text: "Running", "Starting", "Stopped", "Error"

3. **Start Time** (140px)
   - Display local time format: "10:57:36 AM"
   - Use `service.local?.startTime` or `service.startTime`
   - Format: `new Date(startTime).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', second: '2-digit' })`

4. **Source** (flexible, min-width 250px)
   - Display `service.project` (project file path)
   - Truncate with ellipsis if too long
   - Show full path in tooltip on hover

5. **URLs** (flexible, min-width 300px)
   - Primary URL as clickable link (local URL if available)
   - Badge showing "+N" if Azure URL also exists
   - Clicking badge or main link opens URL in new tab
   - Azure URL differentiated with blue styling

6. **Actions** (100px, right-aligned)
   - Icon buttons (no text labels for compactness):
     - Stop/Restart (currently not implemented, show disabled state)
     - View Logs (opens logs tab filtered to this service)
     - More options menu (three dots)

#### Table Styling
- **Dark theme** matching current dashboard aesthetic
- **Row hover**: Subtle background color change with smooth transition
- **Borders**: Subtle borders between rows (border-white/5)
- **Header**: Sticky header with glass effect and backdrop blur
- **Cell padding**: Consistent 12px vertical, 16px horizontal
- **Font sizes**:
  - Header: text-sm font-semibold
  - Cell content: text-sm
  - Secondary info: text-xs text-muted-foreground
- **Alternating rows**: Optional subtle background difference

#### Special Features

**Multi-URL Display**:
```tsx
// If both local and azure URLs exist
<div className="flex items-center gap-2">
  <a href={service.local.url} className="text-primary hover:underline flex items-center gap-1">
    {truncateUrl(service.local.url)}
    <ExternalLink className="w-3 h-3" />
  </a>
  {service.azure?.url && (
    <Badge variant="secondary" className="bg-blue-500/20 text-blue-300">
      +1
    </Badge>
  )}
</div>

// Clicking the badge shows a popover with all URLs
```

**Status Indicator Component**:
```tsx
<div className="flex items-center gap-2">
  <div className={`w-2 h-2 rounded-full ${statusColorClass}`}></div>
  <span className="font-medium">{statusText}</span>
</div>
```

**Health Integration**:
- Show status as "Running" only if both status is running/ready AND health is healthy
- Show "Unhealthy" if status is running but health is unhealthy
- Color coding:
  - Green: running + healthy
  - Yellow: starting
  - Red: error or unhealthy
  - Gray: stopped/not-running

## Data Mapping

### Service to Table Row Mapping

| Table Column | Service Data Path | Fallback | Notes |
|--------------|------------------|----------|-------|
| Name | `service.name` | - | Always present |
| State | `service.local?.status` or `service.status` | 'not-running' | Combine with health |
| Health | `service.local?.health` or `service.health` | 'unknown' | Affects state color |
| Start Time | `service.local?.startTime` or `service.startTime` | '-' | Format to time |
| Source | `service.project` | `service.framework` or '-' | Show framework if no project |
| Local URL | `service.local?.url` | - | Primary URL |
| Azure URL | `service.azure?.url` | - | Secondary URL |
| Port | `service.local?.port` | - | Show in expanded view or tooltip |

## Component Structure

### New Components to Create

1. **`ServiceTable.tsx`** - Main table component
   - Receives `services: Service[]` prop
   - Renders table with all columns
   - Handles sorting (optional for v1)
   - Handles row selection/interaction

2. **`ServiceTableRow.tsx`** - Individual table row
   - Receives `service: Service` prop
   - Renders all cells for one service
   - Handles hover states
   - Handles action button clicks

3. **`StatusCell.tsx`** - Reusable status indicator
   - Receives `status: string, health: string`
   - Returns colored dot + text
   - Matches Aspire's status display

4. **`URLCell.tsx`** - URL display with multi-URL support
   - Receives `localUrl?: string, azureUrl?: string`
   - Shows primary URL + badge for additional URLs
   - Handles popover for multiple URLs

### Modified Components

**`App.tsx`**:
- Add `viewMode` state: `'cards' | 'table'`
- Add view toggle buttons
- Conditionally render `<ServiceTable>` or card grid
- Persist viewMode to localStorage

```tsx
const [viewMode, setViewMode] = useState<'cards' | 'table'>(() => {
  return localStorage.getItem('dashboard-view-preference') as 'cards' | 'table' || 'cards'
})

useEffect(() => {
  localStorage.setItem('dashboard-view-preference', viewMode)
}, [viewMode])
```

## Implementation Plan

### Phase 1: Basic Table Structure
1. Create `ServiceTable.tsx` component with basic table markup
2. Create `ServiceTableRow.tsx` for row rendering
3. Implement column headers
4. Add basic cell rendering for all columns
5. Apply dark theme styling

### Phase 2: View Toggle
1. Add view mode state to App.tsx
2. Create toggle buttons in UI
3. Conditionally render table vs cards
4. Add localStorage persistence

### Phase 3: Interactive Features
1. Implement status indicators with proper colors
2. Add URL cells with clickable links
3. Add multi-URL badge support
4. Implement action buttons (logs view)
5. Add hover states and transitions

### Phase 4: Polish
1. Add smooth transitions between views
2. Implement responsive design (collapse to cards on mobile?)
3. Add loading states for table
4. Add empty state for table
5. Ensure accessibility (keyboard navigation, ARIA labels)

## Styling Details

### Table Container
```css
.service-table {
  @apply glass rounded-2xl overflow-hidden border border-white/10;
}
```

### Table Header
```css
.table-header {
  @apply sticky top-0 z-10 glass backdrop-blur-xl;
  @apply border-b border-white/10;
  @apply text-sm font-semibold text-muted-foreground;
}
```

### Table Row
```css
.table-row {
  @apply border-b border-white/5 last:border-0;
  @apply transition-all-smooth hover:bg-white/5;
  @apply text-sm;
}
```

### Status Colors
```tsx
const statusColors = {
  running: 'bg-green-500',    // Green dot
  starting: 'bg-yellow-500',  // Yellow dot
  error: 'bg-red-500',        // Red dot
  stopped: 'bg-gray-500',     // Gray dot
}

const statusTextColors = {
  running: 'text-green-400',
  starting: 'text-yellow-400',
  error: 'text-red-400',
  stopped: 'text-gray-400',
}
```

## Edge Cases

1. **No services**: Show same empty state as cards view
2. **Long project paths**: Truncate with ellipsis, show full path in tooltip
3. **No URLs**: Show "-" or "N/A"
4. **No start time**: Show "-"
5. **Missing health data**: Default to 'unknown', show gray status
6. **Both local and azure URLs**: Show local as primary, azure in badge
7. **Responsive breakpoints**: 
   - < 768px: Switch to cards view automatically
   - 768-1024px: Reduce column widths, hide less important columns
   - > 1024px: Show all columns

## Accessibility

- **Keyboard navigation**: Tab through table, Enter to click links
- **ARIA labels**: 
  - Table: `aria-label="Services Table"`
  - Status cells: `aria-label="Service status: running"`
  - URL links: `aria-label="Open service URL"`
- **Screen reader support**: Proper table headers with scope attributes
- **Focus indicators**: Visible focus rings on interactive elements

## Future Enhancements (Not in Initial Scope)

1. **Column sorting**: Click headers to sort by that column
2. **Column visibility toggle**: Show/hide specific columns
3. **Row selection**: Multi-select services for bulk actions
4. **Export data**: Export table to CSV/JSON
5. **Search/filter**: Filter services by name, status, etc.
6. **Column resizing**: Drag column borders to resize
7. **Density toggle**: Compact/comfortable/spacious row heights

## Testing Checklist

- [ ] Table renders with all columns
- [ ] View toggle switches between cards and table
- [ ] View preference persists in localStorage
- [ ] Status indicators show correct colors
- [ ] URLs are clickable and open in new tab
- [ ] Multi-URL badge displays correctly
- [ ] Start time formats correctly
- [ ] Long project paths truncate properly
- [ ] Hover states work on rows
- [ ] Empty state displays correctly
- [ ] Loading state displays correctly
- [ ] Error state displays correctly
- [ ] Works on different screen sizes
- [ ] Keyboard navigation works
- [ ] Screen reader announces table properly

## Dependencies

### New Icons Needed
- `LayoutGrid` (for Cards button) - from lucide-react
- `Table` (for Table button) - from lucide-react
- Already have: `Server`, `ExternalLink`, `CheckCircle`, `XCircle`, `Clock`, `AlertCircle`, `StopCircle`

### New UI Components
- Consider creating `<Table>`, `<TableHeader>`, `<TableBody>`, `<TableRow>`, `<TableCell>` components in `components/ui/`
- Or build custom table directly in ServiceTable.tsx

### No New Backend Changes
- All data needed already exists in Service interface
- No API changes required

## Success Metrics

1. **Visual Fidelity**: Table layout closely matches Aspire dashboard aesthetic
2. **Performance**: Table renders smoothly with 10+ services
3. **Usability**: Users can easily toggle between views
4. **Persistence**: View preference survives page refresh
5. **Accessibility**: Passes keyboard navigation and screen reader testing
