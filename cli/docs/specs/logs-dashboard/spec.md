# Multi-Pane Logs Dashboard - Enhancement Specification

## Overview

Enhancement to the existing multi-pane logs dashboard to improve usability and testing capabilities.

## Current State

The multi-pane logs dashboard is fully implemented with:
- Multi-pane grid view (1-6 columns)
- Pattern-based error/warning detection
- Two-level configuration (user + app)
- Status indicators and animations
- Export functionality
- Keyboard shortcuts

## Enhancement Requirements

### 1. Auto-Scroll Control Enhancement

**Problem**: LogsPane currently auto-scrolls based only on pause state and user scrolling detection. Users cannot explicitly control auto-scroll behavior per pane.

**Solution**: Add per-pane auto-scroll toggle control similar to LogsView implementation.

**Requirements**:
- Add auto-scroll toggle button to each LogsPane footer
- Visual indicator showing auto-scroll state (enabled/disabled)
- Auto-scroll should:
  - Default to ON when pane loads
  - Automatically turn OFF when user scrolls up
  - Automatically turn ON when user scrolls to bottom
  - Can be manually toggled by user regardless of scroll position
- When auto-scroll is ON: new logs scroll pane to bottom
- When auto-scroll is OFF: new logs don't move scroll position
- State should be independent per pane
- Tooltip explaining behavior

**User Interaction**:
- Click auto-scroll icon to toggle ON/OFF
- Visual feedback (icon changes, color change, or label)
- Works independently from global pause/resume

### 2. Test Project Enhancement

**Problem**: Current TestAppHost only has one service, making it difficult to test multi-pane layout with realistic data.

**Solution**: Expand TestAppHost to have 5 distinct services.

**Requirements**:
- Add 4 new services to TestAppHost.cs
- Each service should:
  - Have distinct name (api, web, worker, cache, database)
  - Generate realistic logs (mix of info, warnings, errors)
  - Run continuously (not just startup logs)
  - Have different log patterns to test detection
- Services should simulate:
  - api: HTTP request logs with occasional errors
  - web: Frontend build/serve logs
  - worker: Background job processing
  - cache: Redis-style cache operations
  - database: SQL query logs
- Use .NET Aspire patterns where applicable
- Ensure all services start successfully

### 3. UI Refresh Consultation

**Objective**: Review current UI/UX with Designer to identify improvements.

**Areas for Review**:
- LogsPane layout and information density
- Status indicator visibility and clarity
- Controls placement (pause, auto-scroll, search, copy, clear)
- Grid responsiveness at different column counts
- Color scheme for status borders (error red, warning yellow)
- Animation timing and smoothness
- Accessibility (keyboard nav, ARIA labels, focus states)
- Mobile/tablet responsive behavior
- Dark mode support

**Designer Deliverables**:
- Component specification updates
- Visual state improvements
- Interaction refinements
- Accessibility enhancements
- Responsive design recommendations

## Acceptance Criteria

### Auto-Scroll Control
- [ ] Auto-scroll toggle button visible in LogsPane footer
- [ ] Button shows current state (ON/OFF)
- [ ] Clicking button toggles auto-scroll
- [ ] Auto-scroll OFF prevents scroll on new logs
- [ ] Auto-scroll ON scrolls to bottom on new logs
- [ ] User scrolling up disables auto-scroll automatically
- [ ] User scrolling to bottom enables auto-scroll automatically
- [ ] State independent per pane
- [ ] Tooltip provides clear explanation
- [ ] Works with global pause (pause stops logs, auto-scroll controls scroll behavior)

### Test Project
- [ ] TestAppHost has 5 services total
- [ ] All services start without errors
- [ ] Each service generates logs continuously
- [ ] Logs include mix of info/warning/error messages
- [ ] Services have realistic names and behaviors
- [ ] Multi-pane view displays all 5 services correctly
- [ ] Grid layout works well with 5 panes (test 2-3 column layouts)

### UI Refresh
- [ ] Designer reviews current implementation
- [ ] Improvement areas identified
- [ ] Component specs updated
- [ ] Accessibility issues addressed
- [ ] Responsive design verified

## Technical Approach

### Auto-Scroll Implementation
1. Add state variable: `const [autoScrollEnabled, setAutoScrollEnabled] = useState(true)`
2. Modify scroll handler to update autoScrollEnabled based on scroll position
3. Update auto-scroll effect to check both isPaused and autoScrollEnabled
4. Add toggle button to footer with icon (ChevronsDown or similar)
5. Update LogsMultiPaneView to pass global isPaused separately

### Test Project Implementation
1. Define 5 service builders in AppHost.cs
2. Use .NET projects or container images for each service
3. Configure logging for each service
4. Add realistic log generation logic
5. Ensure services remain running (not exit after startup)

## Dependencies

- No new npm packages required
- No new .NET packages required
- Uses existing components and utilities

## Testing

### Manual Testing
- Test auto-scroll toggle on individual panes
- Test with 1-6 column grid layouts
- Test global pause + per-pane auto-scroll combinations
- Test all 5 services running simultaneously
- Test responsive behavior with 5 panes
- Test dark mode
- Test keyboard navigation

### Automated Testing (Future)
- E2E test: auto-scroll behavior
- E2E test: 5 services rendering
- Unit test: LogsPane auto-scroll logic
- Accessibility scan

## Timeline

This is an enhancement to existing feature, not a new feature.

## Success Metrics

- Auto-scroll control improves user experience (qualitative)
- Test project provides realistic multi-service scenario
- UI refresh identifies and addresses usability issues
- No regressions to existing functionality
