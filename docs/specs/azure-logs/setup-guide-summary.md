# Azure Logs Setup Guide - Project Summary

## Overview

Complete redesign of how users configure Azure log streaming in the azd-app dashboard. Replaces the current "button does nothing" experience with an integrated, step-by-step setup wizard that guides users from zero configuration to streaming logs.

## Problem

**Current State** (as of 2025-12-25):
- Azure mode button is clickable but nothing happens when Azure isn't configured
- Tooltip says "Azure logging not configured. Click to diagnose and fix setup"
- No actionable guidance for users to actually set up Azure logs
- Users must piece together docs across multiple sources
- High friction = users give up or file support tickets

**User Pain Points**:
1. Don't know what's missing (workspace? credentials? diagnostic settings?)
2. Don't know how to configure azure.yaml
3. Don't understand bicep requirements
4. Confused about workspace ID vs GUID
5. No way to verify setup is working

## Solution

**Azure Logs Setup Guide** - An integrated 4-step wizard that:
1. Detects current configuration state
2. Shows exactly what's missing
3. Provides copy/paste examples for each fix
4. Validates configuration in real-time
5. Confirms logs are flowing

### User Experience

**Entry Point**: Click Azure mode button when not configured
**Flow**: 4 progressive steps with auto-detection
**Result**: Logs streaming in dashboard, setup persisted

## Architecture

### Frontend (React/TypeScript)

**New Components** (7 total):
1. `AzureSetupGuide.tsx` - Main wizard shell with stepper
2. `WorkspaceSetupStep.tsx` - Step 1: Log Analytics workspace
3. `AuthSetupStep.tsx` - Step 2: Authentication & permissions
4. `DiagnosticSettingsStep.tsx` - Step 3: Service diagnostic settings
5. `SetupVerification.tsx` - Step 4: Verify logs flowing
6. `CodeSnippet.tsx` - Reusable code block with copy
7. `StatusBadge.tsx` - Reusable status indicator

**Modified Components** (4 total):
1. `ModeToggle.tsx` - Opens setup guide when Azure disabled
2. `ConsoleView.tsx` - Hosts setup guide modal
3. `DiagnosticsModal.tsx` - "Fix Setup" button links to guide
4. `AzureErrorDisplay.tsx` - "Setup Guide" button in errors

**New Hooks**:
1. `useSetupProgress.ts` - localStorage-based progress tracking

### Backend (Go)

**New Endpoints**:
1. `GET /api/azure/logs/setup-state` - Detect configuration state
2. `POST /api/azure/logs/verify` - Test log connectivity per service

**New Files**:
1. `cli/src/internal/dashboard/azure_setup.go` - Setup state detection
2. `cli/src/internal/dashboard/azure_setup_test.go` - Unit tests

## The 4 Steps

### Step 1: Log Analytics Workspace
**What it checks**:
- `AZURE_LOG_ANALYTICS_WORKSPACE_ID` env var
- `logs.analytics.workspace` in azure.yaml
- Workspace exists in Azure

**What it provides**:
- Explanation of Log Analytics
- Bicep example for creating workspace
- Bicep output examples
- azure.yaml config snippet
- Auto-detection every 5s

**Status**: ✓ Configured | ⚠ Missing | ○ Not deployed

---

### Step 2: Authentication
**What it checks**:
- User is signed in (`azd auth login`)
- Credential has Log Analytics API scope
- User has `Log Analytics Reader` role

**What it provides**:
- "Sign In" button (triggers auth flow)
- Current user display
- Permission checklist
- Role assignment command (copy/paste)
- Retest button

**Status**: ✓ Authenticated | ⚠ Missing permissions | ✗ Not signed in

---

### Step 3: Diagnostic Settings
**What it checks**:
- Discovers all deployed services
- Checks diagnostic settings per service
- Verifies settings point to correct workspace

**What it provides**:
- Service-by-service status table
- Resource type detection
- Bicep examples per service type
- Bulk status indicator
- Auto-detection of changes

**Status per service**: ✓ Configured | ✗ Missing | ○ Not deployed

---

### Step 4: Verification
**What it checks**:
- Workspace connected ✓
- Authenticated ✓
- Diagnostic settings ✓
- Logs actually flowing ✓

**What it provides**:
- Verification checklist
- Per-service log flow status
- Sample log preview
- "Waiting for logs" countdown (5-15 min normal)
- Success celebration
- "View Logs" CTA button

**Result**: Opens Azure mode with logs streaming

## Design Highlights

### Visual Design
- **Stepper**: Horizontal flow (1 → 2 → 3 → 4) with checkmarks
- **Status Badges**: Icon + text for accessibility
- **Code Blocks**: Syntax highlighting, one-click copy
- **Collapsible Sections**: Progressive disclosure
- **Responsive**: Desktop and mobile layouts

### Interaction Patterns
- **Auto-Detection**: Polls setup state every 5s while wizard open
- **Auto-Advance**: Moves to next step when current validates
- **Deep Linking**: Open to specific step from error states
- **Progress Persistence**: Resume where you left off (24h expiry)
- **Validation**: Can't advance until current step complete

### Accessibility (WCAG AA)
- **Keyboard Navigation**: Tab order, arrow keys, escape to close
- **Screen Readers**: Live regions, semantic markup, status announcements
- **Color Contrast**: 4.5:1 minimum ratio
- **Focus Management**: Trap in modal, restore on close
- **Status Communication**: Not color alone (icons + text)

## Implementation Plan

### Phase 1: Core Wizard (P0) - 11 days
**Backend** (2 days):
- Task 1: Setup state API
- Task 2: Verification API

**Frontend Core** (5 days):
- Task 3: Wizard shell component
- Task 4: Workspace step
- Task 5: Auth step
- Task 6: Diagnostic settings step
- Task 7: Verification step

**Integration** (2 days):
- Task 8: ModeToggle integration
- Task 9: ConsoleView integration
- Task 10: DiagnosticsModal integration
- Task 11: Error state integration

**Polish** (2 days):
- Task 12: Progress persistence
- Task 13: Code copy utilities
- Task 14: Unit tests
- Task 15: Documentation

### Phase 2: Enhancements (P1) - 4 days
- Task 16: Auto-refresh during setup
- Task 17: Bicep template generation
- Task 18: Azure Portal integration
- Task 19: E2E tests

## Success Metrics

**User Experience**:
- 90% discoverability (users find setup guide when needed)
- 70% completion rate (users who start, finish)
- <15 min median time to completion (excluding deployment)
- 50% reduction in "Azure logs not working" support tickets

**Technical**:
- 80% test coverage
- WCAG AA compliance
- <500ms response time for setup state API
- Zero breaking changes to existing local logs

## Files Changed

### New Files (11):
```
cli/src/internal/dashboard/azure_setup.go
cli/src/internal/dashboard/azure_setup_test.go
cli/dashboard/src/components/AzureSetupGuide.tsx
cli/dashboard/src/components/WorkspaceSetupStep.tsx
cli/dashboard/src/components/AuthSetupStep.tsx
cli/dashboard/src/components/DiagnosticSettingsStep.tsx
cli/dashboard/src/components/SetupVerification.tsx
cli/dashboard/src/components/CodeSnippet.tsx
cli/dashboard/src/hooks/useSetupProgress.ts
docs/specs/azure-logs/setup-guide-spec.md
docs/specs/azure-logs/setup-guide-tasks.md
```

### Modified Files (6):
```
cli/src/internal/dashboard/server.go (add routes)
cli/dashboard/src/components/ModeToggle.tsx (open guide on click)
cli/dashboard/src/components/ConsoleView.tsx (host guide modal)
cli/dashboard/src/components/DiagnosticsModal.tsx (add "Fix Setup" button)
cli/dashboard/src/components/AzureErrorDisplay.tsx (add "Setup Guide" button)
cli/docs/features/azure-logs.md (document setup guide)
```

## Documentation

### User Documentation
- **Setup Guide Walkthrough**: cli/docs/features/azure-logs-setup.md
- **Azure Logs Main Docs**: cli/docs/features/azure-logs.md (updated)
- **Troubleshooting**: Links to setup guide for common issues

### Developer Documentation
- **Spec**: docs/specs/azure-logs/setup-guide-spec.md
- **Tasks**: docs/specs/azure-logs/setup-guide-tasks.md
- **Design**: docs/specs/azure-logs/setup-guide-design.md (from Designer)
- **Component API**: JSDoc in each component file

## Dependencies

**Required for Implementation**:
- Existing Azure logs infrastructure (Tasks HF-1 through HF-5 complete)
- Backend API endpoints for health checks
- Frontend modal/dialog patterns
- Code highlighting library (highlight.js or similar)

**Blocks**:
- None - this is a net-new feature, doesn't block existing work

**Unblocks**:
- User adoption of Azure logs feature
- Reduced support burden
- Improved developer experience

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Users skip wizard, config manually | Medium | Make wizard optional, support manual config |
| Bicep examples don't match user infra | Medium | Provide multiple template variations |
| Long setup time discourages users | Medium | Clear expectations (15 min + deployment), save progress |
| Permissions too complex | Medium | Provide exact commands, link to portal |
| Auto-detection false positives | Low | Manual recheck button, clear status messages |

## Next Steps

### Immediate (Next Sprint)
1. **Developer**: Implement backend APIs (Tasks 1-2)
2. **Developer**: Build core wizard components (Tasks 3-7)
3. **Tester**: Write unit tests alongside implementation

### Short-term (1-2 weeks)
4. **Developer**: Integrate with existing components (Tasks 8-11)
5. **Developer**: Add polish (Tasks 12-13)
6. **Tester**: Integration and accessibility testing (Task 14)
7. **Developer**: Documentation (Task 15)

### Medium-term (1 month)
8. **Product Review**: Demo to stakeholders
9. **Beta Testing**: Release to subset of users
10. **Metrics Collection**: Track completion rates, drop-off points
11. **Iteration**: Improve based on user feedback

### Long-term (2-3 months)
12. **Phase 2**: Auto-refresh, bicep generation, portal links (Tasks 16-18)
13. **E2E Testing**: Full workflow coverage (Task 19)
14. **Advanced Features**: Custom queries, multi-workspace support

## Decision Log

**2025-12-25**: Initial spec created
- **Why**: Current Azure mode button UX is broken (clickable but does nothing)
- **Approach**: Integrated wizard vs. external docs
- **Rationale**: Integrated wizard reduces context switching, validates config, provides instant feedback

**Design Choices**:
- **4 steps**: Maps to natural setup progression (infra → auth → config → verify)
- **Auto-detection**: Reduces manual "check if done" clicks
- **Deep linking**: Supports error recovery workflow
- **Progress persistence**: Users can pause/resume across sessions
- **Copy buttons**: Faster than highlighting and copying

## Questions & Answers

**Q: Why not auto-configure everything?**
A: Requires elevated Azure permissions (write access to resources). Keep azd-app read-only, users control their infrastructure.

**Q: What about users with complex infra (Terraform, manual)?**
A: Wizard provides examples, but users can configure manually. Setup guide validates end-state, not how you got there.

**Q: Why 4 steps instead of 1 page?**
A: Reduces cognitive load, enables validation per step, provides clear progress indication.

**Q: How do we handle multiple workspaces?**
A: Phase 1 supports single workspace. Phase 3 adds multi-workspace support (different workspace per service).

**Q: What if logs don't show up after 15 minutes?**
A: Verification step provides troubleshooting link, diagnostic command examples, option to contact support.

## References

- [Azure Logs Main Spec](spec.md)
- [Azure Logs Tasks](tasks.md)
- [Setup Guide Spec](setup-guide-spec.md)
- [Setup Guide Tasks](setup-guide-tasks.md)
- [DiagnosticsModal Component](../../cli/dashboard/src/components/DiagnosticsModal.tsx)
- [AzureErrorDisplay Component](../../cli/dashboard/src/components/AzureErrorDisplay.tsx)

---

**Status**: Spec complete, ready for implementation
**Priority**: P0 - Critical for Azure logs adoption
**Est. Effort**: 11 days Phase 1, 4 days Phase 2
**Owner**: Manager → Designer → Developer
