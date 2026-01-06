<!-- NEXT: 1 -->
# Service Health Diagnostics Enhancement - Tasks

## Phase 1: Backend Enhanced Error Details (P0)

### 1. Extend HealthCheckResult Type
**Assigned**: Developer  
**Status**: TODO

Add new fields to `HealthCheckResult` for enhanced diagnostic information.

**Actions**:
1. Modify `cli/src/internal/healthcheck/types.go`
2. Add `ErrorDetails string` field for extended error information
3. Add `ConsecutiveFailures int` field for failure tracking
4. Add `LastSuccessTime *time.Time` field for last successful check
5. Update JSON tags for new fields

**Acceptance Criteria**:
- New fields added to struct with proper JSON serialization
- Backward compatible with existing consumers
- Fields are optional (omitempty)

---

### 2. Implement Consecutive Failure Tracking
**Assigned**: Developer  
**Status**: TODO

Track consecutive failures per service to help diagnose persistent issues.

**Actions**:
1. Add `failureCount map[string]int` to `HealthMonitor` struct
2. Add `mu sync.RWMutex` for thread-safe access
3. Implement `trackFailure()` method to increment/reset counters
4. Call tracking in check loop after each health check
5. Include count in `HealthCheckResult`

**Files**:
- `cli/src/internal/healthcheck/monitor.go`

**Acceptance Criteria**:
- Counters increment on consecutive unhealthy status
- Counters reset to 0 on healthy status
- Thread-safe implementation (no race conditions)
- Count included in SSE health events

---

### 3. Enhanced HTTP Error Messages
**Assigned**: Developer  
**Status**: TODO

Provide structured, actionable error messages for HTTP health check failures.

**Actions**:
1. Create `suggestHTTPErrorAction(statusCode int) string` helper
2. Map common status codes to actionable suggestions:
   - 503 → "Service temporarily unavailable. Check dependencies."
   - 500-599 → "Server error. Check application logs."
   - 404 → "Health endpoint not found. Verify configuration."
   - 401/403 → "Authentication failed. Check credentials."
3. Add suggestion to `result.Details["suggestion"]`
4. Parse response body for additional error details (if JSON)
5. Populate `ErrorDetails` field with parsed error

**Files**:
- `cli/src/internal/healthcheck/checker.go`

**Acceptance Criteria**:
- All HTTP status codes have meaningful suggestions
- Response body parsed when available (JSON health responses)
- Error messages are concise and actionable
- No sensitive data leaked in error messages

---

### 4. Enhanced TCP/Process Error Messages
**Assigned**: Developer  
**Status**: TODO

Improve error messages for TCP and process health checks.

**Actions**:
1. Add suggestions for TCP errors:
   - Connection refused → "Verify service is running and port is correct"
   - Timeout → "Check network connectivity and firewall rules"
2. Add suggestions for process errors:
   - Not running → "Check service logs, verify start command"
   - Pattern not matched → "Check output pattern, may still be starting"
3. Include PID/port in error details for debugging

**Files**:
- `cli/src/internal/healthcheck/checker.go`

**Acceptance Criteria**:
- TCP and process checks have actionable error messages
- Error messages include relevant context (PID, port, pattern)
- Consistent error format across all check types

---

## Phase 2: Frontend Tooltip UI (P0)

### 5. Create Health Diagnostic Types
**Assigned**: Developer  
**Status**: TODO

Define TypeScript types for health diagnostics and tooltip data.

**Actions**:
1. Update `cli/dashboard/src/types.ts`
2. Add optional fields to `HealthCheckResult`:
   - `errorDetails?: string`
   - `consecutiveFailures?: number`
   - `lastSuccessTime?: string`
3. Create `HealthDiagnostic` interface
4. Create `HealthAction` interface for suggested actions

**Acceptance Criteria**:
- Types match backend Go structs
- Types exported and documented
- No breaking changes to existing types

---

### 6. Build Health Diagnostic Helpers
**Assigned**: Developer  
**Status**: TODO

Create utility functions to build diagnostic information from health check results.

**Actions**:
1. Create `cli/dashboard/src/lib/health-diagnostics.ts`
2. Implement `buildHealthDiagnostic()` - main builder function
3. Implement `getSuggestedActions()` - generate action list based on status/error
4. Implement `getHTTPSpecificActions()` - HTTP status code specific actions
5. Implement `formatDiagnosticReport()` - generate markdown report for copy
6. Add action suggestions for common scenarios (503, 500, 404, connection errors)

**Files**:
- `cli/dashboard/src/lib/health-diagnostics.ts` (new)

**Acceptance Criteria**:
- Functions are pure and testable
- Markdown report is well-formatted and readable
- Actions are relevant to error type and status code
- Include CLI commands in actions (e.g., `azd app logs --service api`)

---

### 7. Create HealthTooltip Component
**Assigned**: Designer → Developer  
**Status**: TODO

Build tooltip component using Radix UI to display health diagnostics.

**Design Requirements**:
- Use existing Radix UI Tooltip from design system
- Max width: 400px
- Position: auto (prefer top/right)
- Delay: 400ms
- Smooth fade in/out animations
- Shadow and border for depth

**Actions**:
1. Create `cli/dashboard/src/components/HealthTooltip.tsx`
2. Implement `HealthTooltip` wrapper using Radix `<Tooltip>`
3. Create `HealthTooltipContent` for layout
4. Add sections:
   - Status header with icon
   - Check details (type, endpoint, timing)
   - Error section (if applicable)
   - Service info (uptime, port, pid)
   - Suggested actions list
5. Add copy button with clipboard API
6. Style with Tailwind (match dashboard theme)

**Files**:
- `cli/dashboard/src/components/HealthTooltip.tsx` (new)
- `cli/dashboard/src/components/HealthTooltipContent.tsx` (new)

**Acceptance Criteria**:
- Tooltip renders correctly for all health statuses
- Positioning avoids viewport overflow
- Copy button works and shows success feedback
- Keyboard accessible (Tab, Enter, Escape)
- Respects `prefers-reduced-motion`

---

### 8. Integrate Tooltip with Status Badges
**Assigned**: Developer  
**Status**: TODO

Wrap health status icons with tooltip to enable diagnostic on hover.

**Actions**:
1. Modify `cli/dashboard/src/components/StatusIndicator.tsx`
2. Update `DualStatusBadge` to accept `service` and `healthStatus` props
3. Wrap health icon (second icon) in `<HealthTooltip>`
4. Pass full `healthStatus` and `service` data to tooltip
5. Update `ServiceCard` to pass required props to `DualStatusBadge`

**Files**:
- `cli/dashboard/src/components/StatusIndicator.tsx`
- `cli/dashboard/src/components/ServiceCard.tsx`

**Acceptance Criteria**:
- Tooltip appears on hover over health icon (not process icon)
- Tooltip shows correct data for each service
- No performance regression (tooltip lazy loads content)
- Works in both grid and table views

---

### 9. Style and Polish Tooltip UI
**Assigned**: Designer → Developer  
**Status**: TODO

Apply visual design and polish to tooltip for professional appearance.

**Design Elements**:
- Status-specific color theming (emerald/amber/rose/slate)
- Icons for each section (check type, timing, error, actions)
- Monospace font for endpoints and commands
- Interactive hover states on action buttons
- Copy success animation/toast notification

**Actions**:
1. Create status-specific color variants
2. Add icons from lucide-react
3. Style action buttons with hover states
4. Add copy success feedback (toast or icon change)
5. Test with long error messages (ensure truncation/scroll)
6. Test with missing data (graceful degradation)

**Files**:
- `cli/dashboard/src/components/HealthTooltip.tsx`
- `cli/dashboard/src/components/HealthTooltipContent.tsx`

**Acceptance Criteria**:
- Tooltip is visually polished and professional
- Color themes match service status
- Copy feedback is clear and immediate
- Long content is scrollable/truncated gracefully
- Dark mode support

---

## Phase 3: Testing & Documentation (P1)

### 10. Unit Tests for Diagnostic Logic
**Assigned**: Tester  
**Status**: TODO

Write comprehensive unit tests for health diagnostic helpers.

**Test Coverage**:
- `buildHealthDiagnostic()` with various health statuses
- `getSuggestedActions()` for different error scenarios
- `formatDiagnosticReport()` output formatting
- Action generation for HTTP status codes (503, 500, 404, etc.)
- Handling of missing/incomplete data

**Files**:
- `cli/dashboard/src/lib/health-diagnostics.test.ts` (new)

**Acceptance Criteria**:
- 100% code coverage for diagnostic helpers
- Tests cover edge cases (missing data, null values)
- Tests verify correct action suggestions

---

### 11. Component Tests for HealthTooltip
**Assigned**: Tester  
**Status**: TODO

Test tooltip component rendering and interactions.

**Test Cases**:
- Renders for all health statuses (healthy, degraded, unhealthy, unknown)
- Shows correct data from `HealthCheckResult`
- Copy button copies to clipboard
- Tooltip opens on hover with delay
- Tooltip closes on mouse leave / Escape key
- Keyboard navigation works (Tab, Enter)
- Respects `prefers-reduced-motion`

**Files**:
- `cli/dashboard/src/components/HealthTooltip.test.tsx` (new)

**Acceptance Criteria**:
- Component tests pass with >=80% coverage
- All interactive elements tested
- Clipboard API mocked and verified

---

### 12. E2E Tests for Tooltip Flow
**Assigned**: Tester  
**Status**: TODO

End-to-end tests for complete tooltip user flow.

**Test Scenarios**:
1. Hover on unhealthy service health icon → tooltip appears
2. Tooltip shows error details and suggested actions
3. Click copy button → success notification → clipboard has markdown
4. Click view logs action → log panel opens filtered to service
5. Test with all health statuses (healthy, degraded, unhealthy, unknown)
6. Test tooltip positioning near viewport edges

**Files**:
- `cli/dashboard/e2e/health-tooltip.spec.ts` (new)

**Acceptance Criteria**:
- All E2E scenarios pass
- Tooltip positioning tested in all viewport locations
- Copy-to-clipboard verified
- Log panel integration verified

---

### 13. Backend Tests for Error Details
**Assigned**: Tester  
**Status**: TODO

Test enhanced error details and failure tracking in Go health checker.

**Test Cases**:
- Consecutive failure tracking increments correctly
- Failure count resets on healthy status
- HTTP error suggestions are correct for each status code
- Error details populated from response body
- Thread safety of failure tracking (concurrent checks)

**Files**:
- `cli/src/internal/healthcheck/monitor_test.go`
- `cli/src/internal/healthcheck/checker_test.go`

**Acceptance Criteria**:
- All backend tests pass
- No race conditions detected (run with `-race` flag)
- Error messages are actionable and accurate

---

### 14. Documentation Updates
**Assigned**: Developer  
**Status**: TODO

Update documentation to explain health diagnostic tooltips.

**Updates Needed**:
1. Add section to `cli/docs/commands/health.md`:
   - Explain diagnostic tooltips in dashboard
   - Show tooltip screenshot
   - Explain copy-to-clipboard feature
2. Update `cli/docs/features/service-states.md`:
   - Document health diagnostic information
   - Explain suggested actions
3. Add troubleshooting guide:
   - Common health check errors
   - Suggested solutions
   - How to interpret diagnostic reports

**Files**:
- `cli/docs/commands/health.md`
- `cli/docs/features/service-states.md`
- `cli/docs/troubleshooting/health-checks.md` (new)

**Acceptance Criteria**:
- Documentation is clear and comprehensive
- Screenshots included
- Examples of diagnostic reports shown

---

## Phase 4: Polish & Metrics (P2)

### 15. Accessibility Audit
**Assigned**: Tester  
**Status**: TODO

Ensure tooltip meets WCAG AA accessibility standards.

**Audit Checklist**:
- Screen reader announces status changes
- Keyboard-only navigation works completely
- ARIA labels on all interactive elements
- Color contrast ratios meet WCAG AA (4.5:1 for text)
- Focus indicators visible
- Tooltip dismissible with Escape key

**Tools**:
- axe DevTools browser extension
- Manual testing with screen reader (NVDA/JAWS)
- Keyboard-only navigation testing

**Acceptance Criteria**:
- No accessibility violations found
- Manual screen reader testing passes
- Keyboard navigation smooth and intuitive

---

### 16. Add Usage Tracking
**Assigned**: Developer  
**Status**: TODO

Add analytics events to measure tooltip usage and effectiveness.

**Events to Track**:
- `health_tooltip_viewed` - when tooltip opens (status, checkType)
- `health_diagnostic_copied` - when user copies diagnostics (status, hasError)
- `health_action_clicked` - when user clicks suggested action (action, status)
- Tooltip open duration (time spent reading)

**Implementation**:
- Use existing analytics infrastructure
- Add events in `HealthTooltip.tsx`
- Include relevant context (service name, status, error type)

**Acceptance Criteria**:
- Events fire correctly and include context
- No PII or sensitive data in events
- Events can be analyzed in analytics dashboard

---

### 17. Performance Testing
**Assigned**: Tester  
**Status**: TODO

Verify tooltip does not impact dashboard performance.

**Performance Metrics**:
- Tooltip render time < 16ms (60fps)
- No extra API calls when tooltip opens
- Memory usage stable (no leaks from tooltip instances)
- Smooth animations on lower-end devices

**Test Methodology**:
- Chrome DevTools Performance profiler
- React DevTools Profiler
- Test with 20+ services in dashboard
- Test rapid hover/unhover (stress test)

**Acceptance Criteria**:
- No performance regression
- Tooltip renders within one frame (16ms)
- No memory leaks detected
- Animations remain smooth

---

### 18. User Feedback & Iteration
**Assigned**: Developer  
**Status**: TODO

Gather user feedback and iterate on tooltip design.

**Feedback Sources**:
- Internal team testing
- Early adopter program
- GitHub issues/discussions
- Usage metrics analysis

**Actions**:
1. Deploy to staging environment
2. Gather feedback over 1-2 weeks
3. Analyze usage metrics (open rate, copy rate)
4. Identify pain points or confusion
5. Iterate on design/content based on feedback

**Acceptance Criteria**:
- Feedback collected from >=10 users
- Metrics show tooltip is being used (>50% hover rate on unhealthy services)
- Issues identified and prioritized for follow-up

---

## Success Metrics

**Completion Criteria**:
- [ ] All tasks marked DONE
- [ ] All tests passing (>=80% coverage for new code)
- [ ] Documentation updated
- [ ] Accessibility audit passed
- [ ] Performance benchmarks met
- [ ] Feature deployed to production

**User Impact Metrics** (measure 2 weeks post-launch):
- Tooltip usage rate on unhealthy services: Target >=60%
- Diagnostic copy rate: Target >=40%
- Time to issue resolution: Target 30% reduction
- Support tickets about "why unhealthy?": Target 50% reduction

**Quality Gates**:
- No critical bugs
- No accessibility violations
- No performance regressions
- Positive user feedback (>=4/5 satisfaction)
