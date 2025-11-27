# LogsPane Component Specification

## Overview
Individual service log viewer pane with real-time streaming, visual status indicators, search, and scroll control for monitoring a single service within the multi-pane dashboard.

## Functional Requirements

### 1. Service Log Display
- Display real-time logs for a single service
- Show service name in pane header
- Display log count (number of entries currently displayed)
- Support ANSI color codes in log messages
- Display timestamps for each log entry
- Show log level for each entry (info, warning, error)

### 2. Visual Status Indicators
- **Pane Border**: Changes based on highest severity log detected
  - Error state: Red border with pulse animation
  - Warning state: Yellow border (no animation)
  - Info state: Gray border (no animation)
- **Status Badge**: Shows current pane status (error/warning/info)
- **Log Level Icons**: Each line shows icon indicating its level
  - Error: Red X-circle icon
  - Warning: Yellow triangle icon
  - Info: Blue info icon

### 3. Auto-Scroll Behavior
- **Default State**: Auto-scroll enabled (automatically scrolls to show newest logs)
- **Manual Control**: User can toggle auto-scroll on/off via button
- **Automatic Detection**: 
  - When user scrolls up: Auto-scroll automatically disables
  - When user scrolls to bottom: Auto-scroll automatically re-enables
- **Visual Indicator**: Button shows clear ON/OFF state
  - ON: Primary/highlighted button style
  - OFF: Muted/secondary button style
- **Tooltip**: Describes current state and action
- **Global Pause Integration**: Auto-scroll toggle remains functional even when logs are paused

### 4. Search and Filter
- **Search Input**: Filter logs by text search (case-insensitive)
- **Real-time Filter**: Results update as user types
- **Clear Search**: Escape key clears search
- **Search Scope**: Only filters displayed logs (does not search historical logs)
- **Empty State**: Show message when no logs match search

### 5. Log Management
- **Copy All Logs**: Button to copy all visible logs to clipboard
- **Copy Single Line**: Right-click or hover action to copy individual log line
- **Clear Logs**: Button to clear all logs from pane
- **Log Limit**: Maintain maximum number of logs in memory (1000 entries)
- **Format Preservation**: Copied logs include timestamp, service name, and message

### 6. Error Detection and Annotation
- **Pattern Matching**: Automatically detect errors and warnings via keyword patterns
- **User Corrections**: Allow marking false positives and false negatives
  - False Positive: User marks error as non-error (line shows strikethrough/dimmed)
  - False Negative: User marks normal log as error (line shows error highlighting)
- **Hover Actions**: On line hover, show inline buttons for:
  - Mark as false positive
  - Mark as error (false negative)
  - Copy line
- **Pattern Exclusions**: Support global patterns to exclude known false positives

### 7. Keyboard and Mouse Interactions
- **Tab Navigation**: Move focus between controls (search → auto-scroll → clear → copy)
- **Enter/Space**: Activate focused button
- **Escape**: Clear search when search input is focused
- **Right-click**: Copy line to clipboard (context menu action)
- **Scroll**: Container scrollable independently of page
- **Hover**: Show inline actions on log lines

### 8. Real-time Updates
- **WebSocket Connection**: Connect to service log stream
- **Stream Updates**: Receive and display new logs as they arrive
- **Pause Handling**: Respect global pause state (buffer logs, don't display)
- **Resume Handling**: Display buffered logs when resumed
- **Connection Error**: Show error state if stream disconnects

## Accessibility (WCAG 2.1 AA)

### Screen Reader Support
- Pane must announce its purpose: "Logs for {serviceName}"
- Log container marked as live region for new log announcements
- Auto-scroll toggle announces state changes
- All buttons have descriptive labels
- Status changes announced (error detected, logs cleared)

### Keyboard Navigation
- All interactive elements accessible via Tab key
- Logical tab order: search → auto-scroll → clear → copy
- Enter/Space activates buttons
- Escape clears search
- Focus indicators visible and high contrast

### Visual Accessibility
- Sufficient color contrast for all text (WCAG AA minimum 4.5:1)
- Error/warning states use color + icon + text (not color alone)
- Log level icons provide redundant encoding beyond color
- Focus states have visible outline/ring
- Touch targets minimum 40px on mobile

### Reduced Motion
- Respect user's motion preferences
- Pulse animation disabled if reduced motion preferred
- Smooth scroll behavior optional based on preferences

## Responsive Design

### Mobile (<600px)
- Single column layout (panes stack vertically)
- Pane height: 300px minimum
- Footer controls stack vertically
- Touch targets minimum 40px
- Search may collapse to icon with expand action

### Tablet (600-960px)
- 2-column default layout
- Pane height: 400px
- Footer controls horizontal, may wrap
- Touch targets minimum 40px

### Desktop (>960px)
- 2-6 columns (user configurable)
- Pane height: 200-800px (user configurable)
- Footer controls horizontal
- Button size: 32px × 32px

## Acceptance Criteria

1. ✓ Log pane displays service name, status badge, and log count in header
2. ✓ Real-time logs stream from WebSocket connection
3. ✓ Pane border color and animation reflect highest severity (error/warning/info)
4. ✓ Each log line shows level icon (error/warning/info)
5. ✓ Auto-scroll enabled by default, scrolls to show newest logs
6. ✓ Auto-scroll toggle button shows clear ON/OFF visual state
7. ✓ User scroll up automatically disables auto-scroll
8. ✓ User scroll to bottom automatically re-enables auto-scroll
9. ✓ Search input filters logs in real-time (case-insensitive)
10. ✓ Copy all button copies visible logs to clipboard
11. ✓ Right-click on log line copies that line to clipboard
12. ✓ Clear button removes all logs from pane
13. ✓ Hover on log line shows inline actions (copy, mark false positive/negative)
14. ✓ False positive marking shows line with strikethrough/dimmed
15. ✓ False negative marking shows line with error highlighting
16. ✓ Keyboard navigation works (Tab, Enter, Space, Escape)
17. ✓ Screen reader announces pane purpose and state changes
18. ✓ WCAG 2.1 AA color contrast maintained
19. ✓ Reduced motion preference respected (no pulse animation)
20. ✓ Responsive layout works on mobile, tablet, and desktop
21. ✓ Dark mode support with appropriate colors
22. ✓ Maximum logs in memory limit enforced (1000 entries)
23. ✓ Global pause state respected (logs buffer, not displayed)

## Success Metrics

- Users can monitor service logs without manual scrolling (auto-scroll adoption)
- Error detection reduces troubleshooting time (visual status indicators)
- False positive/negative marking improves error pattern accuracy
- Keyboard-only users can operate all pane functions
- Screen reader users understand pane state and content
- Responsive design works across all device sizes

## Future Enhancements

- Line number display
- Timestamp format customization
- Log level filtering (show only errors, warnings, etc.)
- Export logs to file
- Search with regex support
- Match highlighting in search results
- Customizable error/warning patterns per service
- Log line wrapping toggle
- Font size adjustment
- Virtual scrolling for 1000+ log lines

---

**Version**: 1.0  
**Status**: Implemented  
**Last Updated**: 2025-11-23
