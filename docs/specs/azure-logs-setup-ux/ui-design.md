# Azure Logs Setup UX - UI Design

## Design System

**Existing Patterns** (from DiagnosticSettingsStep.tsx and dashboard components):
- Tailwind CSS utility classes
- Dark mode support via `dark:` prefixes
- Lucide icons
- cn() utility for conditional classes
- Slate color palette (slate-50 to slate-900)
- Semantic colors: cyan (primary), emerald (success), yellow/orange (warning), red (error)

**Component Structure**:
- Padding: px-6 py-6 for content areas
- Borders: border-slate-200 dark:border-slate-700
- Backgrounds: bg-white dark:bg-slate-900, bg-slate-50 dark:bg-slate-900/60
- Text: text-slate-900 dark:text-slate-100 (primary), text-slate-600 dark:text-slate-400 (secondary)
- Spacing: gap-3, gap-4 for consistency

---

## Component 1: DiagnosticSettingsStep (Redesign)

### Overview
Replace per-service expandable cards with a simple aggregated status view. Show all services in a compact list with status icons.

### Props
```typescript
interface DiagnosticSettingsStepProps {
  onValidationChange: (isValid: boolean) => void
}
```

### State Management
```typescript
interface DiagnosticCheckStatus {
  status: 'loading' | 'success' | 'error'
  services: ServiceDiagnosticStatus[]
  workspaceId?: string
  error?: string
}

interface ServiceDiagnosticStatus {
  serviceName: string
  resourceType: string
  status: 'configured' | 'not-configured' | 'error'
  resourceId?: string
  error?: string
}
```

### Layout Structure

```
┌─────────────────────────────────────────────────────────┐
│ Diagnostic Settings                                     │ ← H3 heading
│ Configure logging for your Azure services              │ ← Description
│                                                         │
│ ┌─────────────────────────────────────────────────┐   │
│ │ [Status Summary Area]                           │   │
│ │ ○ Checking diagnostic settings...              │   │ ← Loading state
│ │   OR                                            │   │
│ │ ✓ All 3 services are configured                │   │ ← Success (all)
│ │   OR                                            │   │
│ │ ⚠ 2 of 3 services need configuration           │   │ ← Partial/None
│ │   OR                                            │   │
│ │ ✗ Could not check diagnostic settings          │   │ ← Error
│ └─────────────────────────────────────────────────┘   │
│                                                         │
│ ┌─────────────────────────────────────────────────┐   │
│ │ [Service List]                                  │   │
│ │                                                 │   │
│ │ ✓ appService           Container Apps          │   │
│ │ ✗ containerApp         App Service             │   │
│ │ ⚠ function             Azure Functions         │   │
│ └─────────────────────────────────────────────────┘   │
│                                                         │
│ [Action Buttons]                                        │
│ [Show Bicep Template] [Recheck]                        │
└─────────────────────────────────────────────────────────┘
```

### Visual States

#### 1. Loading State
```
┌─────────────────────────────────────────────────────┐
│ Status Summary                                      │
│ ┌─────────────────────────────────────────────┐   │
│ │ ● Checking diagnostic settings...          │   │
│ │   [Spinner icon] Please wait...            │   │
│ └─────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```
- Icon: Loader2 (spinning)
- Background: bg-slate-50 dark:bg-slate-900/60
- Text: text-slate-600 dark:text-slate-400

#### 2. All Configured (Success)
```
┌─────────────────────────────────────────────────────┐
│ Status Summary                                      │
│ ┌─────────────────────────────────────────────┐   │
│ │ ✓ All 3 services are configured            │   │
│ │   Your diagnostic settings are ready       │   │
│ └─────────────────────────────────────────────┘   │
│                                                     │
│ Services                                            │
│ ┌─────────────────────────────────────────────┐   │
│ │ ✓ appService         Container Apps        │   │
│ │ ✓ containerApp       App Service           │   │
│ │ ✓ function           Azure Functions       │   │
│ └─────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```
- Summary icon: CheckCircle, text-emerald-600 dark:text-emerald-400
- Summary background: bg-emerald-50 dark:bg-emerald-950/30
- Service icons: CheckCircle, text-emerald-600
- No action buttons needed (auto-valid for Next button)

#### 3. Partial/None Configured (Warning)
```
┌─────────────────────────────────────────────────────┐
│ Status Summary                                      │
│ ┌─────────────────────────────────────────────┐   │
│ │ ⚠ 2 of 3 services need configuration       │   │
│ │   Diagnostic settings required for logs    │   │
│ └─────────────────────────────────────────────┘   │
│                                                     │
│ Services                                            │
│ ┌─────────────────────────────────────────────┐   │
│ │ ✓ appService         Container Apps        │   │
│ │ ○ containerApp       App Service           │   │
│ │ ○ function           Azure Functions       │   │
│ └─────────────────────────────────────────────┘   │
│                                                     │
│ How to fix:                                         │
│ 1. Click "Show Bicep Template" below               │
│ 2. Copy the template to your infrastructure        │
│ 3. Run `azd up` to deploy the changes              │
│                                                     │
│ [Show Bicep Template →] [Recheck]                  │
└─────────────────────────────────────────────────────┘
```
- Summary icon: AlertTriangle, text-orange-600 dark:text-orange-400
- Summary background: bg-orange-50 dark:bg-orange-950/30
- Configured services: CheckCircle, text-emerald-600
- Not configured: Circle (outline), text-slate-400
- Error services: AlertTriangle, text-red-600

#### 4. Error State
```
┌─────────────────────────────────────────────────────┐
│ Status Summary                                      │
│ ┌─────────────────────────────────────────────┐   │
│ │ ✗ Could not check diagnostic settings      │   │
│ │   InsufficientPermissions: Missing Reader  │   │
│ └─────────────────────────────────────────────┘   │
│                                                     │
│ Troubleshooting:                                    │
│ • Ensure you have Reader role on resources         │
│ • Verify azd auth login is current                 │
│ • Check network connectivity                       │
│                                                     │
│ [Retry] [Skip This Step →]                         │
└─────────────────────────────────────────────────────┘
```
- Summary icon: AlertTriangle, text-red-600 dark:text-red-400
- Summary background: bg-red-50 dark:bg-red-950/30
- Show error message from API
- Actionable troubleshooting steps

### Service List Item Design

Each service shown as a single row:
```
┌───────────────────────────────────────────────────┐
│ [Icon] serviceName       Resource Type Name      │
└───────────────────────────────────────────────────┘
```

- Icon (20x20): CheckCircle (green), Circle (gray), AlertTriangle (red)
- Service name: text-sm font-medium, text-slate-900 dark:text-slate-100
- Resource type: text-sm, text-slate-600 dark:text-slate-400
- Padding: py-2.5
- Border-bottom: last item has no border

**No expand/collapse** - all information visible at once

### Action Buttons

**Show Bicep Template** (primary):
```tsx
<button className="bg-cyan-600 text-white hover:bg-cyan-700 px-4 py-2 rounded-lg">
  Show Bicep Template →
</button>
```

**Recheck** (secondary):
```tsx
<button className="border border-slate-300 text-slate-700 hover:bg-slate-100 px-4 py-2 rounded-lg">
  <RefreshCw className="w-4 h-4 mr-2" />
  Recheck
</button>
```

**Skip This Step** (ghost):
```tsx
<button className="text-slate-600 hover:text-slate-900 hover:bg-slate-100 px-4 py-2 rounded-lg">
  Skip This Step →
</button>
```

### Responsive Behavior

- Desktop (≥768px): Service list shows name and type side-by-side
- Mobile (<768px): Stack service name over type name
- Buttons: Stack vertically on mobile, horizontal on desktop

### Accessibility

- Summary status has role="status" for screen readers
- Loading state has aria-live="polite"
- Service list is semantic `<ul>` with `<li>` items
- Icon meanings conveyed via text (not color alone)
- Focus indicators on all interactive elements
- Keyboard navigation: Tab through buttons, Enter/Space to activate

---

## Component 2: BicepTemplateModal

### Overview
Modal dialog displaying unified Bicep template for all detected services.

### Props
```typescript
interface BicepTemplateModalProps {
  isOpen: boolean
  onClose: () => void
  services: string[] // ['appService', 'containerApp', 'function']
}
```

### Layout Structure

```
┌─────────────────────────────────────────────────────────┐
│ Diagnostic Settings Template                       [X] │ ← Header
├─────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────┐   │
│ │ ► Integration Instructions                      │   │ ← Collapsible
│ └─────────────────────────────────────────────────┘   │
│                                                         │
│ Template (Bicep)                            [Copy All] │
│ ┌─────────────────────────────────────────────────┐   │
│ │ // Diagnostic settings for Azure Logs          │   │
│ │ param logAnalyticsWorkspaceId string            │   │ ← Code block
│ │                                                 │   │
│ │ resource appService 'Microsoft.Web/...' = {    │   │
│ │   ...                                           │   │
│ │ }                                               │   │
│ └─────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────┤
│ [Download .bicep]                          [Close]     │ ← Footer
└─────────────────────────────────────────────────────────┘
```

### Visual Design

**Modal Container**:
- Max width: max-w-4xl
- Max height: max-h-[85vh]
- Background: bg-white dark:bg-slate-900
- Border: border border-slate-200 dark:border-slate-700
- Shadow: shadow-2xl
- Rounded: rounded-2xl
- Overlay: bg-black/50 dark:bg-black/70

**Header**:
- Title: text-lg font-semibold, text-slate-900 dark:text-slate-100
- Close button: p-2, hover:bg-slate-100 dark:hover:bg-slate-800
- Border-bottom: border-slate-200 dark:border-slate-700

**Instructions Section** (collapsible):
```tsx
<details className="border-b border-slate-200 dark:border-slate-700">
  <summary className="cursor-pointer p-4 hover:bg-slate-50 dark:hover:bg-slate-900/60">
    <ChevronRight className="inline-block" /> Integration Instructions
  </summary>
  <div className="px-6 pb-4">
    <ol className="list-decimal list-inside space-y-2">
      <li>Save template as <code>infra/modules/diagnostic-settings.bicep</code></li>
      <li>Add workspace parameter to main.bicep</li>
      <li>Reference module for each service</li>
      <li>Run <code>azd up</code> to deploy</li>
    </ol>
  </div>
</details>
```

**Code Block**:
- Use existing CodeBlock component
- Syntax highlighting for Bicep
- Max height: max-h-96, overflow-y-auto
- Copy button in top-right corner
- Line numbers optional

**Copy All Button**:
```tsx
<button className="absolute top-2 right-2 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 px-3 py-1.5 rounded text-xs">
  {copied ? '✓ Copied' : 'Copy All'}
</button>
```
- Toast notification on copy: "Template copied to clipboard"

**Download Button**:
```tsx
<button className="border border-slate-300 hover:bg-slate-100 px-4 py-2 rounded-lg">
  Download .bicep
</button>
```
- Downloads file as `diagnostic-settings.bicep`

### Interaction Flow

1. User clicks "Show Bicep Template"
2. Modal fades in (animate-in)
3. Template loads from API (loading state shows spinner)
4. User can:
   - Expand/collapse instructions
   - Scroll through template
   - Click "Copy All" (shows toast)
   - Click "Download" (saves file)
   - Click "Close" or Esc (fade out)

### Accessibility

- Modal has role="dialog", aria-modal="true"
- Focus trap: Tab cycles within modal
- Esc key closes modal
- Focus returns to trigger button on close
- Close button has aria-label="Close template"
- Code block has aria-label="Bicep template code"

---

## Component 3: SetupVerification (Enhanced)

### Overview
Replace placeholder verification with actual workspace query results.

### Props
```typescript
interface SetupVerificationProps {
  onValidationChange: (isValid: boolean) => void
  onComplete?: () => void
}
```

### State Management
```typescript
interface VerificationStatus {
  status: 'idle' | 'verifying' | 'success' | 'partial' | 'error'
  results?: ServiceVerificationResult[]
  error?: string
}

interface ServiceVerificationResult {
  serviceName: string
  logCount: number
  lastLogTime?: string
  status: 'ok' | 'no-logs' | 'error'
  message?: string
}
```

### Layout Structure

```
┌─────────────────────────────────────────────────────────┐
│ Verification                                            │
│ Test your workspace connection and log flow            │
│                                                         │
│ ┌─────────────────────────────────────────────────┐   │
│ │ [Progress/Summary Area]                         │   │
│ └─────────────────────────────────────────────────┘   │
│                                                         │
│ ┌─────────────────────────────────────────────────┐   │
│ │ [Service Results List]                          │   │
│ └─────────────────────────────────────────────────┘   │
│                                                         │
│ [Action Buttons]                                        │
└─────────────────────────────────────────────────────────┘
```

### Visual States

#### 1. Idle (Initial)
```
┌─────────────────────────────────────────────────────┐
│ Ready to verify your Azure logs setup              │
│                                                     │
│ This will query your workspace for recent logs     │
│ and verify that diagnostic settings are working.   │
│                                                     │
│ [Start Verification →]                             │
└─────────────────────────────────────────────────────┘
```

#### 2. Verifying (Loading)
```
┌─────────────────────────────────────────────────────┐
│ ● Testing connection to Log Analytics workspace... │
│   [Spinner]                                         │
│                                                     │
│ This may take a few seconds...                     │
└─────────────────────────────────────────────────────┘
```

- Animated spinner (Loader2)
- Progress text: "Connecting...", "Querying logs...", "Analyzing results..."
- Background: bg-slate-50 dark:bg-slate-900/60

#### 3. Success (All Services)
```
┌─────────────────────────────────────────────────────┐
│ ✓ All 3 services verified                          │
│   Your Azure logs are flowing correctly            │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│ Services                                            │
│                                                     │
│ ✓ appService                                       │
│   15 log entries in last 15 minutes                │
│   Last log: 2 minutes ago                          │
│                                                     │
│ ✓ containerApp                                     │
│   23 log entries in last 15 minutes                │
│   Last log: 30 seconds ago                         │
│                                                     │
│ ✓ function                                         │
│   3 log entries in last 15 minutes                 │
│   Last log: 5 minutes ago                          │
└─────────────────────────────────────────────────────┘

[View Logs →]  [Complete Setup]
```

- Summary: CheckCircle icon, emerald background
- Each service: CheckCircle icon, log count, timestamp
- "Complete Setup" button is primary (cyan)

#### 4. Success (Partial)
```
┌─────────────────────────────────────────────────────┐
│ ⚠ 2 of 3 services verified                         │
│   Some services may not have generated logs yet    │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│ Services                                            │
│                                                     │
│ ✓ appService                                       │
│   15 log entries in last 15 minutes                │
│                                                     │
│ ✓ containerApp                                     │
│   8 log entries in last 15 minutes                 │
│                                                     │
│ ⚠ function                                         │
│   No logs found in last 15 minutes                 │
│   This is normal if the function hasn't run yet.   │
│   Try triggering the function and recheck.         │
└─────────────────────────────────────────────────────┘

[Retry]  [View Logs Anyway →]  [Complete Setup]
```

- Summary: AlertTriangle icon, yellow/orange background
- Services with logs: CheckCircle (green)
- Services without logs: AlertTriangle (yellow), explanatory message
- All buttons enabled (user can proceed)

#### 5. Error (No Diagnostic Settings)
```
┌─────────────────────────────────────────────────────┐
│ ✗ No logs found for any service                    │
│   Diagnostic settings are not configured           │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│ Error Details:                                      │
│                                                     │
│ ○ appService                                       │
│   No diagnostic settings configured                │
│                                                     │
│ ○ containerApp                                     │
│   No diagnostic settings configured                │
│                                                     │
│ ○ function                                         │
│   No diagnostic settings configured                │
└─────────────────────────────────────────────────────┘

How to fix:
1. Go back to Diagnostic Settings step
2. Configure diagnostic settings for your services
3. Wait 2-5 minutes for logs to appear
4. Return here to verify

[← Back to Diagnostic Settings]  [Retry]  [Skip →]
```

- Summary: AlertTriangle icon, red background
- Services: Circle (outline), error messages
- Primary action: "Back to Diagnostic Settings" (navigates to step 3)
- Secondary: Retry, Skip

### Service Result Item Design

Each service result shown as a card:
```
┌───────────────────────────────────────────────────┐
│ [Icon] serviceName                                │
│ → log count, last log time                       │
│ → optional message                                │
└───────────────────────────────────────────────────┘
```

**Success (ok)**:
- Icon: CheckCircle, text-emerald-600
- Count: text-sm, text-slate-900 dark:text-slate-100
- Time: text-xs, text-slate-600 dark:text-slate-400

**Warning (no-logs)**:
- Icon: AlertTriangle, text-orange-600
- Message: text-sm, text-slate-700 dark:text-slate-300
- Background: bg-orange-50 dark:bg-orange-950/30

**Error (error)**:
- Icon: AlertTriangle, text-red-600
- Error message: text-sm, text-red-700 dark:text-red-300
- Background: bg-red-50 dark:bg-red-950/30

### Action Buttons

**Start Verification** (primary):
```tsx
<button className="bg-cyan-600 text-white hover:bg-cyan-700 px-4 py-2 rounded-lg">
  Start Verification →
</button>
```

**Retry** (secondary):
```tsx
<button className="border border-slate-300 hover:bg-slate-100 px-4 py-2 rounded-lg">
  <RefreshCw className="w-4 h-4 mr-2" />
  Retry
</button>
```

**View Logs** (primary):
```tsx
<button className="bg-cyan-600 text-white hover:bg-cyan-700 px-4 py-2 rounded-lg">
  View Logs →
</button>
```
- Navigates to dashboard logs view with Azure mode enabled

**Complete Setup** (primary):
```tsx
<button className="bg-emerald-600 text-white hover:bg-emerald-700 px-4 py-2 rounded-lg">
  Complete Setup
</button>
```

**Back to Diagnostic Settings** (secondary):
```tsx
<button className="border border-slate-300 hover:bg-slate-100 px-4 py-2 rounded-lg">
  ← Back to Diagnostic Settings
</button>
```

### Responsive Behavior

- Service cards stack vertically on all screen sizes
- Buttons wrap on mobile (flex-wrap)
- Compact spacing on mobile (reduce padding)

### Accessibility

- Verification status has role="status"
- Loading state has aria-live="polite"
- Service list is semantic `<ul>`
- Error messages are aria-live="assertive"
- Focus management: focus "View Logs" on success, "Retry" on error
- Keyboard navigation for all buttons

---

## Animation & Transitions

**Modal Entry/Exit**:
- Fade in: opacity 0 → 1, duration 200ms
- Scale in: scale 95 → 100, duration 200ms
- Backdrop fade: opacity 0 → 1, duration 150ms

**State Transitions**:
- Status changes: fade crossfade, duration 200ms
- Service list updates: stagger animation (delay 50ms per item)
- Button states: background/color transition 150ms

**Loading Spinners**:
- Continuous rotation: animate-spin
- Icon: Loader2 from lucide-react

---

## Color Palette Reference

**Status Colors**:
- Success: emerald-600 (light), emerald-400 (dark)
- Warning: orange-600 (light), orange-400 (dark)
- Error: red-600 (light), red-400 (dark)
- Info: cyan-600 (light), cyan-400 (dark)

**Backgrounds**:
- Success: bg-emerald-50 dark:bg-emerald-950/30
- Warning: bg-orange-50 dark:bg-orange-950/30
- Error: bg-red-50 dark:bg-red-950/30
- Info: bg-slate-50 dark:bg-slate-900/60

**Text**:
- Primary: text-slate-900 dark:text-slate-100
- Secondary: text-slate-600 dark:text-slate-400
- Muted: text-slate-500 dark:text-slate-500

---

## Implementation Notes

1. **API Integration**: Use existing fetch patterns from current dashboard
2. **Error Handling**: Show specific error messages from API, fallback to generic
3. **Loading States**: Always show loading feedback for async operations
4. **Optimistic UI**: Immediately show "Rechecking..." on retry button
5. **Persistence**: Consider saving last verification result in session storage
6. **Testing**: Test all states with Playwright, including error scenarios

---

## Success Criteria

✅ All states are visually distinct and clear
✅ Actions are obvious and easy to find
✅ Error recovery paths are straightforward
✅ Accessible via keyboard and screen readers
✅ Responsive on mobile and desktop
✅ Consistent with existing dashboard design system
✅ No per-service expansion/collapse (simplified UX)
✅ Single unified template (not per-service templates)
✅ Actual verification (not placeholder UI)
