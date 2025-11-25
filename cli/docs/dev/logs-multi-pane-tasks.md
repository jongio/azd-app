# Multi-Pane Logs Dashboard - Development Tasks

**Spec**: [logs-multi-pane-view.md](../features/logs-multi-pane-view.md)

## Task Breakdown

### Phase 1: Core Multi-Pane Layout (Priority: HIGH)

- [ ] **T1.1**: Create `LogsPaneContainer.tsx` component
  - Display grid layout with responsive columns (2-4 based on screen size)
  - Accept array of services to display
  - Each service gets dedicated pane

- [ ] **T1.2**: Create `LogsPane.tsx` component
  - Isolated log display for single service
  - Header with service name + status badge
  - Scrollable body (600px height, overflow-y auto)
  - Footer with per-pane controls
  - Border styling (3-4px, color-coded by status)

- [ ] **T1.3**: Create `PaneHeader.tsx` sub-component
  - Display service name
  - Show status badge (error/warning/info)
  - Show log count

- [ ] **T1.4**: Implement independent log streaming per pane
  - Each pane maintains its own log array
  - WebSocket or filtered stream per service
  - Max 1000 logs per pane (independent from global)

- [ ] **T1.5**: Implement per-pane auto-scroll & pause
  - Detect user scroll in pane
  - Pause auto-scroll when scrolled up
  - Resume when returned to bottom
  - Scroll-up indicator in pane

### Phase 2: Status Indicators & Animations (Priority: HIGH)

- [ ] **T2.1**: Implement error detection logic
  - Analyze logs for error keywords (error, failed, exception, fatal, panic, etc.)
  - Track most recent error in pane
  - Update pane status to ERROR when detected

- [ ] **T2.2**: Implement warning detection logic
  - Analyze logs for warning keywords (warn, warning, caution, deprecated)
  - Update pane status to WARNING when detected (no error)
  - Update pane status to INFO otherwise

- [ ] **T2.3**: Implement status-driven border styling
  - ERROR = red (#EF4444) border + blinking animation
  - WARNING = yellow/amber (#FBBF24) border, no animation
  - INFO = white/gray (#E5E7EB) border, no animation
  - Use Tailwind classes + custom CSS for blinking

- [ ] **T2.4**: Add CSS keyframe animation for blinking
  - Pulse effect 500ms on/off
  - Apply only to error-status panes

### Phase 3: False Positive / False Negative Markers (Priority: MEDIUM)

- [ ] **T3.1**: Create `LogLineMarker.tsx` component
  - Clickable checkmark for marking false positive
  - Clickable exclamation for marking false negative
  - Visual state indicators

- [ ] **T3.2**: Implement false positive marking logic
  - Store marked line indices per pane
  - De-emphasize marked lines (opacity-50, text-decoration-line-through or similar)
  - Exclude marked lines from status calculation
  - Persist marked set in component state during session

- [ ] **T3.3**: Implement false negative marking logic
  - Store force-flagged line indices per pane
  - Highlight in red (#EF4444)
  - Include in error status calculation (treat as real error)
  - Show force-flag badge

- [ ] **T3.4**: Add common false positive patterns
  - Detect patterns like "0 errors", "error rate: 0%"
  - Provide one-click dismiss for pattern
  - Option to auto-mark similar patterns
  - Store patterns in localStorage with regex

- [ ] **T3.5**: Add pattern management UI
  - Show detected patterns in footer
  - Allow user to add custom patterns (regex)
  - Allow user to delete patterns
  - Persist to localStorage

### Phase 4: Per-Pane Controls (Priority: MEDIUM)

- [ ] **T4.1**: Create per-pane search box
  - Live search within single pane
  - Highlight matching lines
  - Show match count (e.g., "3 of 15 matches")

- [ ] **T4.2**: Create per-pane log level filter
  - Filter by: All, Errors only, Warnings only, Info only
  - Combine with search
  - Show filtered count

- [ ] **T4.3**: Create per-pane clear/export
  - Clear logs button (with confirmation)
  - Export single pane logs as text file

- [ ] **T4.4**: Create false positive exclude toggle
  - Toggle to hide/show false positives
  - Shows/hides marked lines

### Phase 5: Global Controls & Interactions (Priority: MEDIUM)

- [ ] **T5.1**: Add global service selector
  - Checkboxes or multi-select for services
  - Show/hide panes dynamically
  - Preserve scroll position when toggling

- [ ] **T5.2**: Add global pause/resume
  - Pauses all panes simultaneously
  - Show paused indicator at top
  - Resume button applies to all

- [ ] **T5.3**: Add global search (optional, lower priority)
  - Search across all visible panes
  - Highlight matches in all panes

- [ ] **T5.4**: Add multi-pane export
  - Export all visible panes (concatenated or separate files)
  - Include service name in header

### Phase 6: Testing & Polish (Priority: HIGH)

- [ ] **T6.1**: Unit tests for status detection
  - Test error pattern matching
  - Test warning pattern matching
  - Test false positive/negative exclusion from status

- [ ] **T6.2**: Unit tests for pane isolation
  - Verify logs don't cross panes
  - Verify search doesn't affect other panes
  - Verify marked lines only affect own pane

- [ ] **T6.3**: E2E tests for multi-pane view
  - Test pane rendering with multiple services
  - Test status indicator changes
  - Test false positive marking workflow
  - Test export functionality

- [ ] **T6.4**: Accessibility testing
  - ARIA labels on headers, panes, buttons
  - Keyboard navigation between panes (arrow keys)
  - Screen reader announces status changes
  - Color contrast check (WCAG AA)

- [ ] **T6.5**: Responsive design testing
  - Test layout on mobile (1 col)
  - Test layout on tablet (2 col)
  - Test layout on desktop (3-4 col)
  - No overflow or truncation

- [ ] **T6.6**: Performance testing
  - Stress test with 10+ services
  - Stress test with 1000+ logs per pane
  - Measure scroll performance (should be smooth)
  - Measure search/filter latency (<100ms)

### Phase 7: Optional Enhancements (Lower Priority)

- [ ] **T7.1**: Virtual scrolling for large logs
  - Only render visible log lines in viewport
  - Improves performance with 1000+ logs

- [ ] **T7.2**: Custom pane sizing
  - Allow user to resize panes (drag divider)
  - Persist sizes in localStorage

- [ ] **T7.3**: Pane reordering
  - Drag-and-drop to reorder panes
  - Persist order in localStorage

- [ ] **T7.4**: Session persistence
  - Save marked patterns to localStorage
  - Save service selection to localStorage

## Dependencies

- React 18+
- Tailwind CSS (already in project)
- lucide-react (already in project for icons)
- Vitest + React Testing Library (already in project)
- Playwright (already in project for e2e)

## Completion Criteria

All Phase 1-6 tasks must be complete with:
- All tests passing (target: ≥80% coverage)
- All e2e tests passing
- Accessibility audit passing (WCAG AA)
- No console errors or warnings
- No TypeScript errors (`any` types forbidden)
- Performance: search/filter <100ms, scroll smooth at 60fps

## Success Metrics

- Users can see 2-4 services simultaneously in separate panes
- Status indicators accurately reflect service health
- False positive marking reduces noise by ≥50% in typical logs
- Export functionality works for single and multi-pane
- All tests passing, ≥80% code coverage
