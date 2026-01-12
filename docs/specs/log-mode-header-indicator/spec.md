# Log Mode Header Indicator

## Summary

Move the log mode indicator (Local/Azure) from a separate row into the log pane header as a persistent, right-aligned icon to reduce vertical space while maintaining clear visibility of which logs are being displayed.

## Motivation

Currently, each log pane has two header elements:
1. **Header row**: Service name, port, status icons, and action buttons
2. **Mode bar row**: Full-width colored bar showing "Viewing Azure Logs • Live from Azure resources" or "Viewing Local Logs • From local development server"

This two-row approach:
- Consumes valuable vertical space that could display more logs
- Duplicates information (mode is often implicit from service context)
- Creates visual clutter when viewing multiple service log panes

Users need to always know which log source they're viewing, but the current implementation is inefficient with space.

## Goals

- **Space Efficiency**: Reclaim vertical space by eliminating the separate mode bar row
- **Persistent Visibility**: Always show which log mode is active (never hidden)
- **Quick Scanning**: Enable instant visual identification of log source across multiple panes
- **Clarity**: Maintain or improve user understanding of what logs they're viewing
- **Smooth Transitions**: Preserve visual feedback during mode switching

## Non-Goals

- Changing the mode switching mechanism itself
- Modifying the global mode toggle behavior
- Altering the log content or filtering

## Design

### Current Layout

```
┌─────────────────────────────────────────────────────────────┐
│ ⌄ service-name:3000  ⚫ ✓  [Actions]                        │ ← Header
├─────────────────────────────────────────────────────────────┤
│ ☁ Viewing Azure Logs • Live from Azure resources           │ ← Mode Bar
├─────────────────────────────────────────────────────────────┤
│ [Log entries...]                                            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Proposed Layout

```
┌─────────────────────────────────────────────────────────────┐
│ ⌄ service-name:3000  ⚫ ✓  ☁ Azure  [Actions]              │ ← Header only
├─────────────────────────────────────────────────────────────┤
│ [Log entries...]                                            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Header Structure

The header will contain (left to right):
1. **Collapse chevron** (existing)
2. **Service name + port** (existing)
3. **Process status icon** (existing)
4. **Health status icon** (existing)
5. **→ Log mode badge** (NEW - positioned before action buttons)
6. **Action buttons** (existing: ServiceActions, Settings, ExternalLink, etc.)

### Log Mode Badge Design

#### Local Mode
- **Icon**: Monitor icon (current)
- **Text**: "Local"
- **Colors**: 
  - Light: `text-slate-600 bg-slate-100 border-slate-300`
  - Dark: `text-slate-300 bg-slate-800 border-slate-600`
- **Tooltip**: "Viewing local development logs"

#### Azure Mode
- **Icon**: Cloud icon (current)
- **Text**: "Azure"
- **Colors**:
  - Light: `text-azure-700 bg-azure-100 border-azure-300`
  - Dark: `text-azure-300 bg-azure-900/30 border-azure-700`
- **Tooltip**: "Viewing Azure resource logs"

#### Mode Switching State
- **Icon**: Loader2 icon (spinning)
- **Text**: "Switching..."
- **Colors**: Same as target mode (azure/local)
- **Tooltip**: "Switching to [Azure/Local] logs..."

#### Badge Component Spec

```tsx
<div className={cn(
  "inline-flex items-center gap-1.5 px-2 py-1 rounded-md",
  "text-xs font-medium border transition-all",
  logMode === 'azure'
    ? "text-azure-700 dark:text-azure-300 bg-azure-100 dark:bg-azure-900/30 border-azure-300 dark:border-azure-700"
    : "text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-slate-800 border-slate-300 dark:border-slate-600"
)}
title={tooltipText}
>
  <Icon className="w-3.5 h-3.5" />
  <span>{badgeText}</span>
</div>
```

### Visual Positioning

The badge will be positioned:
- **After**: Health status icon
- **Before**: ServiceActions divider and action buttons
- **Spacing**: 2-unit gap from health icon, maintains existing button spacing

### Accessibility

- Badge includes `title` attribute for tooltip on hover
- Mode switching state announced via loading indicator
- Sufficient color contrast (WCAG AA compliant)
- No information conveyed by color alone (icons + text labels)

## Implementation

### Files to Modify

1. **`LogsPaneHeader.tsx`**
   - Add log mode badge component between health icon and action buttons
   - Import Monitor, Cloud, Loader2 icons
   - Add logic to determine badge state (local/azure/switching)

2. **`LogsPane.tsx`**
   - Remove `<LogsPaneModeBar>` component usage
   - Keep passing `logMode` and `isModeSwitching` props to header

3. **`LogsPaneAzureControls.tsx`**
   - Mark `LogsPaneModeBar` as deprecated (keep for now in case of rollback)
   - Add comment noting replacement with header badge

### Component Changes

```tsx
// In LogsPaneHeader.tsx
export function LogsPaneHeader({
  // ... existing props
  logMode,
  isModeSwitching,
  azureUrl,
}: Readonly<LogsPaneHeaderProps>) {
  
  // Badge rendering logic
  const ModeBadge = () => {
    const Icon = isModeSwitching ? Loader2 : (logMode === 'azure' ? Cloud : Monitor)
    const text = isModeSwitching ? 'Switching...' : (logMode === 'azure' ? 'Azure' : 'Local')
    const tooltip = isModeSwitching 
      ? `Switching to ${logMode === 'azure' ? 'Azure' : 'Local'} logs...`
      : logMode === 'azure' 
        ? 'Viewing Azure resource logs'
        : 'Viewing local development logs'
    
    return (
      <div className={cn(
        "inline-flex items-center gap-1.5 px-2 py-1 rounded-md",
        "text-xs font-medium border transition-all",
        isModeSwitching && "animate-pulse",
        logMode === 'azure'
          ? "text-azure-700 dark:text-azure-300 bg-azure-100 dark:bg-azure-900/30 border-azure-300 dark:border-azure-700"
          : "text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-slate-800 border-slate-300 dark:border-slate-600"
      )}
      title={tooltip}
      >
        <Icon className={cn("w-3.5 h-3.5", isModeSwitching && "animate-spin")} />
        <span>{text}</span>
      </div>
    )
  }

  return (
    <div className={cn("flex items-center justify-between px-4 py-2 border-b", headerBgClass)}>
      {/* Left section: collapse + name + icons */}
      <button className="flex items-center gap-2 flex-1 min-w-0" onClick={toggleCollapsed}>
        {/* ... existing content ... */}
        <ModeBadge />
      </button>

      {/* Right section: action buttons */}
      <div className="flex items-center gap-2">
        {/* ... existing buttons ... */}
      </div>
    </div>
  )
}
```

### Testing Requirements

1. **Visual Testing**
   - Verify badge appears correctly in both local and azure modes
   - Confirm switching state shows spinner and "Switching..." text
   - Check color contrast in light and dark themes
   - Test with multiple panes side-by-side

2. **Functional Testing**
   - Mode switching animation works smoothly
   - Tooltip displays correct text for each state
   - Badge updates when global mode toggle is used
   - Services with `host: local` always show "Local" badge

3. **Responsive Testing**
   - Header doesn't overflow or wrap on narrow viewports
   - Badge text truncates gracefully if needed
   - Touch targets remain adequate (min 44x44px)

4. **Accessibility Testing**
   - Screen reader announces mode changes appropriately
   - Keyboard navigation works correctly
   - Color contrast meets WCAG AA standards
   - Information not conveyed by color alone

## Migration Strategy

### Phase 1: Add Badge (Low Risk)
- Add badge to header alongside existing mode bar
- Test visual layout and functionality
- Gather feedback on badge design

### Phase 2: Remove Mode Bar (Completion)
- Remove LogsPaneModeBar component usage
- Update tests to expect new layout
- Update documentation/screenshots

### Rollback Plan
If issues arise, the mode bar component remains in the codebase and can be re-enabled by:
1. Reverting LogsPane.tsx to include `<LogsPaneModeBar>`
2. Removing badge from LogsPaneHeader.tsx

## Metrics

- **Space Savings**: ~28-32px vertical height per log pane (mode bar removed)
- **Scan Time**: Time to identify log mode should remain ≤500ms (informal testing)
- **User Feedback**: Monitor for confusion or usability issues in first 2 weeks

## Open Questions

- ~~Should badge be clickable to toggle mode?~~ No - mode switching handled by global toggle
- ~~Should diagnostics button move to header?~~ No - diagnostics feature to be addressed separately

## References

- Current implementation: `LogsPaneAzureControls.tsx` (LogsPaneModeBar component)
- Header component: `LogsPaneHeader.tsx`
- Similar patterns: Service card badges, status indicators throughout dashboard
