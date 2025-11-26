# Log Pane Visual Enhancements - Specification

## Overview

Enhancements to the LogsPane component to improve visual status indication and optimize space utilization when panes are collapsed.

## Requirements

### 1. Header Background Color Based on Service State

The log pane header should have a background color that visually indicates the current status of that service based on the highest severity log detected.

**States:**
- **Error state**: Header background uses error/red color scheme
- **Warning state**: Header background uses warning/yellow color scheme  
- **Info state**: Header background uses default/neutral color scheme

**Behavior:**
- Color updates in real-time as logs arrive
- Maintains consistency with existing pane border status colors
- Must be accessible (sufficient contrast for text readability)
- Works in both light and dark mode

### 2. Dynamic Space Redistribution on Collapse/Expand

When a log pane is collapsed, the remaining expanded panes should automatically resize to fill the available vertical space that the collapsed pane freed up.

**Behavior:**
- Collapsed panes show only the header (minimal height)
- Expanded panes grow to fill available space equally
- Smooth transition when collapsing/expanding
- Works correctly with any number of panes in grid layout
- Maintains grid column structure (only row height changes)

## Acceptance Criteria

### Header Background Color
- [ ] Header background color reflects error state (red tones)
- [ ] Header background color reflects warning state (yellow tones)
- [ ] Header background color reflects info state (neutral/default)
- [ ] Text remains readable against all background colors (WCAG AA contrast)
- [ ] Works correctly in light mode
- [ ] Works correctly in dark mode
- [ ] Status badge remains visible and distinguishable from header background

### Collapse/Expand Space Redistribution
- [ ] Collapsed panes shrink to header-only height
- [ ] Remaining expanded panes grow to fill freed space
- [ ] Space distribution is equal among expanded panes
- [ ] Works with 1-6 column grid layouts
- [ ] Works with any number of services/panes
- [ ] Transition is smooth (CSS transition)
- [ ] Performance remains acceptable (no layout thrashing)

## Technical Considerations

- LogsPane component handles collapse state and header rendering
- LogsPaneGrid manages grid layout and pane sizing
- Need coordination between collapsed state and grid row sizing
- CSS Grid can handle dynamic sizing with proper configuration

## Dependencies

- No new packages required
- Uses existing Tailwind CSS utility classes
- Uses existing color tokens from design system

---

**Version**: 1.0  
**Status**: Draft  
**Created**: 2025-11-26
