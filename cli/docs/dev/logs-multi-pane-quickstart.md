# Multi-Pane Logs Dashboard - Developer Quick Start

## What You're Building

A new logs view where each running service (process) gets its own isolated log pane in a grid layout. Each pane shows the service's status with color-coded borders:
- 🔴 **Red + Blinking** = Service has errors
- 🟡 **Yellow** = Service has warnings  
- ⚪ **White/Gray** = Service is running normally

## Key Features

1. **Multi-Pane Grid**: 2-4 columns depending on screen size, each pane shows one service
2. **Status Indicators**: Pane borders change color & blink based on log analysis
3. **False Positive Markers**: Users can click to mark "error" strings that aren't real errors (e.g., "0 errors found")
4. **False Negative Markers**: Users can force-flag lines that should be treated as errors
5. **Per-Pane Controls**: Search, filter, clear, export work independently per pane
6. **Independent Scrolling**: Each pane scrolls independently, pauses when user scrolls up

## Phase Breakdown (Build in Order)

### Phase 1: Core Layout (Essential First)
- Build `LogsPaneContainer` (grid wrapper)
- Build `LogsPane` (single service pane component)
- Get each pane showing logs independently
- Implement per-pane auto-scroll

### Phase 2: Status Indicators (Core Feature)
- Detect errors in logs (regex patterns)
- Detect warnings in logs
- Update pane border color + blinking based on status
- Make sure errors = red + blink, warnings = yellow, info = white

### Phase 3: False Positive/Negative Markers (UI Enhancement)
- Add click-to-mark buttons on each log line
- De-emphasize false positives (strikethrough/dim)
- Highlight false negatives (red)
- Exclude false positives from status calculation

### Phase 4: Per-Pane Controls (Polish)
- Search box per pane
- Clear/export per pane
- Log level filter per pane

### Phase 5: Global Controls (Nice to Have)
- Service selector (show/hide panes)
- Global pause button
- Multi-pane export

### Phase 6: Testing & Accessibility (Ship Quality)
- Full unit test coverage (≥80%)
- E2E tests for all workflows
- Accessibility audit (ARIA labels, keyboard nav)

## Files You'll Create/Modify

**New Components:**
- `LogsPaneContainer.tsx` (wrapper grid)
- `LogsPane.tsx` (single service pane)
- `PaneHeader.tsx` (service name + status badge)
- `LogLineMarker.tsx` (false pos/neg buttons)

**Modified Files:**
- `LogsView.tsx` (refactor to use new components)
- `App.tsx` (add route to new multi-pane view or toggle between single/multi)

**Styles:**
- Add `pane-error-blink` keyframe animation in global CSS

**Tests:**
- `LogsPane.test.tsx`
- `LogsPaneContainer.test.tsx`
- `e2e/logs-multi-pane.spec.ts`

## Key Patterns in Existing Code

### Log Entry Structure
```typescript
interface LogEntry {
  service: string
  message: string
  level: number  // 1=info, 2=warning, 3=error
  timestamp: string
  isStderr: boolean
}
```

### Status Detection Example (from current code)
```typescript
const isErrorLine = (message: string) => {
  const errorPattern = /\b(error|failed|failure|exception|fatal|panic|critical)\b/i
  return errorPattern.test(message)
}
```

### WebSocket Pattern
- Connect to `/api/logs/stream?service={serviceName}`
- Listen for JSON log entries
- Add to state, keep max 1000 recent logs

## Acceptance Criteria Checklist

When handing off to QA, ensure:
- [ ] All 3-4 services show in separate panes
- [ ] Error status shows red + blinking
- [ ] Warning status shows yellow (no blink)
- [ ] Info status shows white
- [ ] Can mark false positives (they turn dim)
- [ ] Can mark false negatives (they turn red)
- [ ] Marked lines excluded from status (errors disappear if marked false positive)
- [ ] Per-pane search works
- [ ] Per-pane export works
- [ ] All tests pass (unit + e2e)
- [ ] No TypeScript errors
- [ ] Accessible (ARIA labels, keyboard nav)

## Common Gotchas

1. **Cross-pane contamination**: Each pane must have independent state (logs, search, scroll position, marked lines)
2. **Status calculation**: Must exclude marked false positives from error detection
3. **WebSocket cleanup**: Close connections when service is hidden or pane destroyed
4. **Performance**: With 10+ services, consider lazy rendering or virtual scrolling
5. **Animation**: Error blink should be subtle (~500ms pulse), not distracting

## Running Tests

```bash
cd cli/dashboard

# Unit tests
npm test

# E2E tests
npm run test:e2e

# Coverage
npm run test:coverage
```

## Deployment Note

This is a new view. Plan to:
- Keep existing single-pane `LogsView` as fallback or legacy
- Add toggle to switch between single/multi-pane views
- Or replace single-pane with multi-pane (but handle single service gracefully)

---

**Spec**: See `docs/features/logs-multi-pane-view.md` for full details  
**Tasks**: See `docs/dev/logs-multi-pane-tasks.md` for detailed task breakdown
