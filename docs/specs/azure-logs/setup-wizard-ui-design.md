# Azure Logs Setup Wizard - UI Design Specification

**Version**: 1.0  
**Date**: December 25, 2025  
**Designer**: GitHub Copilot  
**Status**: Ready for Developer Handoff

## Table of Contents

1. [Overview](#overview)
2. [Component Interfaces](#component-interfaces)
3. [Visual Specifications](#visual-specifications)
4. [Interaction Patterns](#interaction-patterns)
5. [State Management](#state-management)
6. [Accessibility Plan](#accessibility-plan)
7. [Implementation Checklist](#implementation-checklist)

---

## Overview

### Design Goals

1. **Progressive Disclosure**: Guide users step-by-step without overwhelming
2. **Visual Consistency**: Match existing dashboard design system
3. **Clear Progress**: Always show where users are in the setup flow
4. **Actionable Guidance**: Every issue has a clear, copyable fix
5. **Accessible**: WCAG AA compliant for all users

### Design System Reference

The wizard follows the established dashboard design patterns:
- Modal structure from [DiagnosticsModal.tsx](../../../cli/dashboard/src/components/DiagnosticsModal.tsx)
- Toggle/button patterns from [ModeToggle.tsx](../../../cli/dashboard/src/components/ModeToggle.tsx)
- Badge variants from [badge-variants.ts](../../../cli/dashboard/src/components/ui/badge-variants.ts)
- Color system: Cyan primary, Slate neutrals, Emerald success, Amber warning, Red error

---

## Component Interfaces

### 1. AzureSetupGuide (Main Container)

**Purpose**: Primary wizard modal with stepper navigation and step orchestration.

```typescript
interface AzureSetupGuideProps {
  /** Whether the wizard modal is open */
  isOpen: boolean
  
  /** Callback when wizard is closed (X button, escape, or backdrop click) */
  onClose: () => void
  
  /** Callback when setup completes successfully */
  onComplete?: () => void
  
  /** Optional: Deep link to specific step (from error states) */
  initialStep?: SetupStep
  
  /** Additional className for container customization */
  className?: string
}

type SetupStep = 'workspace' | 'auth' | 'diagnostic-settings' | 'verification'

interface SetupGuideState {
  currentStep: SetupStep
  completedSteps: Set<SetupStep>
  canAdvance: boolean
  isDirty: boolean // Has user made changes?
}
```

**Component Structure**:
```tsx
<AzureSetupGuide>
  <Modal>
    <Header>
      <Title>Azure Logs Setup</Title>
      <CloseButton />
    </Header>
    
    <StepperNavigation />
    
    <Content>
      {/* Lazy-loaded step components */}
      <WorkspaceSetupStep />
      <AuthSetupStep />
      <DiagnosticSettingsStep />
      <SetupVerification />
    </Content>
    
    <Footer>
      <ProgressIndicator />
      <NavigationButtons />
    </Footer>
  </Modal>
</AzureSetupGuide>
```

---

### 2. WorkspaceSetupStep (Task 4)

**Purpose**: Configure Log Analytics workspace (Step 1 of 4).

```typescript
interface WorkspaceSetupStepProps {
  /** Callback when workspace validation passes */
  onValidated: (workspaceId: string) => void
  
  /** Whether this step is currently active */
  isActive: boolean
  
  /** Pre-populated workspace ID if detected */
  detectedWorkspaceId?: string
}

interface WorkspaceState {
  status: 'not-configured' | 'checking' | 'configured' | 'error'
  workspaceId?: string
  workspaceName?: string
  workspaceGuid?: string
  errorMessage?: string
}
```

**Component Structure**:
```tsx
<WorkspaceSetupStep>
  <StatusBadge status={state.status} />
  
  <CollapsibleSection title="What is Log Analytics?">
    <ExplanationText />
  </CollapsibleSection>
  
  <CollapsibleSection title="Create Workspace">
    <AzureCliInstructions />
    <PortalInstructions />
  </CollapsibleSection>
  
  <CollapsibleSection title="Bicep Example" defaultExpanded>
    <CodeSnippet language="bicep" code={bicepTemplate} />
  </CollapsibleSection>
  
  <CollapsibleSection title="azure.yaml Configuration" defaultExpanded>
    <CodeSnippet language="yaml" code={azureYamlConfig} />
  </CollapsibleSection>
  
  <Actions>
    <RecheckButton />
    <DetectionStatus />
  </Actions>
</WorkspaceSetupStep>
```

---

### 3. AuthSetupStep (Task 5)

**Purpose**: Verify authentication and permissions (Step 2 of 4).

```typescript
interface AuthSetupStepProps {
  /** Callback when auth validation passes */
  onValidated: (principal: string) => void
  
  /** Whether this step is currently active */
  isActive: boolean
}

interface AuthState {
  status: 'not-authenticated' | 'checking' | 'authenticated' | 'missing-permissions'
  principal?: string
  hasLogAnalyticsReader: boolean
  subscriptionId?: string
  resourceGroupName?: string
  workspaceName?: string
}
```

**Component Structure**:
```tsx
<AuthSetupStep>
  <StatusBadge status={state.status} />
  
  <AuthenticationCheck>
    <CheckItem 
      label="Signed in" 
      status={state.principal ? 'pass' : 'fail'}
      value={state.principal}
    />
    <SignInButton />
  </AuthenticationCheck>
  
  <PermissionsCheck>
    <CheckItem
      label="Log Analytics Reader role"
      status={state.hasLogAnalyticsReader ? 'pass' : 'fail'}
    />
    {!state.hasLogAnalyticsReader && (
      <CollapsibleSection title="How to assign role">
        <PortalInstructions />
        <CodeSnippet language="bash" code={roleAssignmentCommand} />
      </CollapsibleSection>
    )}
  </PermissionsCheck>
  
  <Actions>
    <RetestButton />
  </Actions>
</AuthSetupStep>
```

---

### 4. DiagnosticSettingsStep (Task 6)

**Purpose**: Configure diagnostic settings for each service (Step 3 of 4).

```typescript
interface DiagnosticSettingsStepProps {
  /** Callback when all services are configured */
  onValidated: () => void
  
  /** Whether this step is currently active */
  isActive: boolean
}

interface DiagnosticSettingsState {
  status: 'loading' | 'ready' | 'error'
  services: ServiceDiagnosticStatus[]
  totalServices: number
  configuredServices: number
}

interface ServiceDiagnosticStatus {
  name: string
  resourceType: string
  deployed: boolean
  diagnosticSettings: boolean
  bicepExample?: string
}
```

**Component Structure**:
```tsx
<DiagnosticSettingsStep>
  <ProgressSummary>
    {configuredServices} of {totalServices} services configured
  </ProgressSummary>
  
  <FilterControls>
    <ShowAllButton />
    <ShowIncompleteButton />
  </FilterControls>
  
  <ServiceTable>
    {services.map(service => (
      <ServiceRow key={service.name}>
        <ServiceName>{service.name}</ServiceName>
        <ResourceType badge>{service.resourceType}</ResourceType>
        <StatusIndicator status={service.diagnosticSettings} />
        <ExpandBicepButton service={service} />
      </ServiceRow>
    ))}
  </ServiceTable>
  
  <ExpandedBicepExamples>
    {expandedServices.map(service => (
      <CodeSnippet 
        key={service.name}
        language="bicep" 
        code={service.bicepExample}
        title={`${service.name} - Diagnostic Settings`}
      />
    ))}
  </ExpandedBicepExamples>
  
  <Actions>
    <ShowAllBicepButton />
    <RecheckButton />
  </Actions>
</DiagnosticSettingsStep>
```

---

### 5. SetupVerification (Task 7)

**Purpose**: Verify log flow and celebrate success (Step 4 of 4).

```typescript
interface SetupVerificationProps {
  /** Callback when verification completes successfully */
  onComplete: () => void
  
  /** Whether this step is currently active */
  isActive: boolean
}

interface VerificationState {
  status: 'checking' | 'waiting-for-logs' | 'success' | 'error'
  checks: VerificationCheck[]
  services: ServiceVerificationStatus[]
}

interface VerificationCheck {
  name: 'workspace' | 'authentication' | 'diagnostic-settings' | 'log-flow'
  status: 'pass' | 'pending' | 'fail'
  message?: string
}

interface ServiceVerificationStatus {
  name: string
  logsFlowing: boolean
  lastLogTimestamp?: string
  sampleLog?: string
  waitingMinutes?: number
}
```

**Component Structure**:
```tsx
<SetupVerification>
  <ChecklistProgress>
    {checks.map(check => (
      <ChecklistItem
        key={check.name}
        label={check.name}
        status={check.status}
        message={check.message}
      />
    ))}
  </ChecklistProgress>
  
  <ServiceVerificationList>
    {services.map(service => (
      <ServiceVerification key={service.name}>
        <ServiceName>{service.name}</ServiceName>
        {service.logsFlowing ? (
          <LogsFlowingIndicator timestamp={service.lastLogTimestamp} />
        ) : (
          <WaitingIndicator minutes={service.waitingMinutes} />
        )}
        {service.sampleLog && (
          <SampleLogPreview log={service.sampleLog} />
        )}
      </ServiceVerification>
    ))}
  </ServiceVerificationList>
  
  {status === 'success' && (
    <SuccessState>
      <CelebrationIcon />
      <SuccessMessage>All set! Your Azure logs are ready.</SuccessMessage>
      <ViewLogsButton onClick={onComplete} />
    </SuccessState>
  )}
  
  {status === 'waiting-for-logs' && (
    <WaitingState>
      <InfoMessage>
        Logs may take 5-15 minutes to appear after deployment.
        You can continue using the dashboard and check back later.
      </InfoMessage>
      <ContinueAnywayButton onClick={onComplete} />
    </WaitingState>
  )}
  
  <Actions>
    <RefreshButton />
    <AdvancedConfigLink />
  </Actions>
</SetupVerification>
```

---

### 6. Shared Components

#### CodeSnippet

```typescript
interface CodeSnippetProps {
  /** Code content to display */
  code: string
  
  /** Language for syntax highlighting */
  language: 'bicep' | 'yaml' | 'bash' | 'powershell' | 'json'
  
  /** Optional title above code block */
  title?: string
  
  /** Whether snippet can collapse/expand */
  collapsible?: boolean
  
  /** Default collapsed state (if collapsible) */
  defaultCollapsed?: boolean
  
  /** Maximum height before scrolling */
  maxHeight?: number
  
  /** Additional className */
  className?: string
}
```

#### StatusBadge

```typescript
interface StatusBadgeProps {
  /** Current status */
  status: 'configured' | 'not-configured' | 'checking' | 'error' | 'warning'
  
  /** Optional custom label (defaults to status) */
  label?: string
  
  /** Show icon */
  showIcon?: boolean
  
  /** Size variant */
  size?: 'sm' | 'md' | 'lg'
}
```

#### CollapsibleSection

```typescript
interface CollapsibleSectionProps {
  /** Section title (always visible) */
  title: string
  
  /** Content to show/hide */
  children: React.ReactNode
  
  /** Start expanded */
  defaultExpanded?: boolean
  
  /** Optional badge in header */
  badge?: React.ReactNode
  
  /** Optional icon in header */
  icon?: React.ReactNode
}
```

---

## Visual Specifications

### Color Palette

```typescript
const colors = {
  // Status colors
  success: {
    bg: 'bg-emerald-50 dark:bg-emerald-950/30',
    border: 'border-emerald-200 dark:border-emerald-800',
    text: 'text-emerald-700 dark:text-emerald-200',
    icon: 'text-emerald-500',
  },
  warning: {
    bg: 'bg-amber-50 dark:bg-amber-950/30',
    border: 'border-amber-200 dark:border-amber-800',
    text: 'text-amber-700 dark:text-amber-200',
    icon: 'text-amber-500',
  },
  error: {
    bg: 'bg-red-50 dark:bg-red-950/30',
    border: 'border-red-200 dark:border-red-800',
    text: 'text-red-700 dark:text-red-200',
    icon: 'text-red-500',
  },
  info: {
    bg: 'bg-cyan-50 dark:bg-cyan-950/30',
    border: 'border-cyan-200 dark:border-cyan-800',
    text: 'text-cyan-700 dark:text-cyan-200',
    icon: 'text-cyan-500',
  },
  neutral: {
    bg: 'bg-slate-50 dark:bg-slate-800/50',
    border: 'border-slate-200 dark:border-slate-700',
    text: 'text-slate-700 dark:text-slate-300',
    icon: 'text-slate-500 dark:text-slate-400',
  },
}
```

### Stepper Navigation

**Visual Design**:
```
┌─────────────────────────────────────────────────────────────┐
│  1. Workspace  →  2. Auth  →  3. Diagnostics  →  4. Verify  │
│      ✓              ✓              ○                ○        │
└─────────────────────────────────────────────────────────────┘
```

**States**:
- **Completed**: ✓ Checkmark, emerald color, solid border
- **Current**: Circle with number, cyan color, pulsing border
- **Upcoming**: Empty circle, slate color, dashed border
- **Error**: ✗ X mark, red color, solid border

**Implementation**:
```typescript
interface StepperProps {
  steps: StepDefinition[]
  currentStep: number
  completedSteps: Set<number>
}

interface StepDefinition {
  id: SetupStep
  label: string
  shortLabel?: string // For mobile
  icon?: React.ReactNode
}

const stepperStates = {
  completed: {
    icon: '✓',
    color: 'text-emerald-500',
    border: 'border-emerald-500',
    bg: 'bg-emerald-50 dark:bg-emerald-950/30',
  },
  current: {
    icon: '○',
    color: 'text-cyan-600 dark:text-cyan-400',
    border: 'border-cyan-600 dark:border-cyan-400 border-2',
    bg: 'bg-cyan-50 dark:bg-cyan-950/30',
    animation: 'animate-pulse',
  },
  upcoming: {
    icon: '○',
    color: 'text-slate-400 dark:text-slate-600',
    border: 'border-slate-300 dark:border-slate-700 border-dashed',
    bg: 'bg-slate-50 dark:bg-slate-900/30',
  },
  error: {
    icon: '✗',
    color: 'text-red-500',
    border: 'border-red-500',
    bg: 'bg-red-50 dark:bg-red-950/30',
  },
}
```

**Responsive Behavior**:
- Desktop (≥768px): Full labels, horizontal layout
- Mobile (<768px): Short labels or icons only, compact spacing

### Status Badges

**Visual Design**:
```
┌─────────────────────────┐
│  ✓  Configured          │  Success
├─────────────────────────┤
│  ⚠  Needs Attention     │  Warning
├─────────────────────────┤
│  ○  Not Configured      │  Neutral
├─────────────────────────┤
│  ⏱  Checking...         │  Loading
├─────────────────────────┤
│  ✗  Error               │  Error
└─────────────────────────┘
```

**Size Variants**:
```typescript
const badgeSizes = {
  sm: 'px-2 py-0.5 text-xs',
  md: 'px-2.5 py-1 text-sm',
  lg: 'px-3 py-1.5 text-base',
}
```

**Implementation**:
```tsx
<StatusBadge status="configured" size="md" />
// Renders:
<div class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-sm font-semibold bg-emerald-50 border-emerald-200 text-emerald-700">
  <CheckCircle class="w-3.5 h-3.5" />
  Configured
</div>
```

### Code Blocks

**Visual Design**:
```
┌─────────────────────────────────────────────┐
│  Bicep Example                        Copy  │
├─────────────────────────────────────────────┤
│  output AZURE_LOG_ANALYTICS_WORKSPACE_ID    │
│    string = monitoring.outputs.workspaceId  │
│  output AZURE_LOG_ANALYTICS_WORKSPACE_NAME  │
│    string = monitoring.outputs.workspaceName│
│                                             │
│  [Expand for full example ▼]                │
└─────────────────────────────────────────────┘
```

**Features**:
- Syntax highlighting (use existing highlight.js or prism)
- Line numbers for long snippets (>10 lines)
- Copy button with feedback ("Copied!" for 2 seconds)
- Expand/collapse for snippets >15 lines
- Max height 400px with scroll

**Implementation**:
```tsx
<CodeSnippet
  language="bicep"
  code={bicepTemplate}
  title="Bicep Example"
  collapsible
  maxHeight={400}
/>
```

### Buttons

**Primary Action** (Next, Continue, View Logs):
```css
bg-cyan-600 hover:bg-cyan-700 text-white
shadow-sm
px-4 py-2 rounded-lg
focus:ring-2 focus:ring-cyan-500
```

**Secondary Action** (Back, Recheck):
```css
border border-slate-200 dark:border-slate-700
bg-white dark:bg-slate-900
text-slate-800 dark:text-slate-100
hover:bg-slate-100 dark:hover:bg-slate-800
px-4 py-2 rounded-lg
```

**Ghost Action** (Skip, Cancel):
```css
text-slate-700 dark:text-slate-200
hover:bg-slate-200/70 dark:hover:bg-slate-800
px-3 py-2 rounded-lg
```

**Destructive** (unused in wizard):
```css
bg-red-600 hover:bg-red-700 text-white
```

### Modal Structure

**Dimensions**:
- Width: `max-w-4xl` (896px)
- Height: `max-h-[90vh]` (90% viewport height)
- Border radius: `rounded-2xl`

**Layout**:
```
┌────────────────────────────────────────────┐
│  Header (sticky)                      [X]  │ 72px
├────────────────────────────────────────────┤
│  Stepper Navigation (sticky)               │ 80px
├────────────────────────────────────────────┤
│                                            │
│  Scrollable Content                        │ flex-1
│  (Step components)                         │
│                                            │
├────────────────────────────────────────────┤
│  Footer (sticky)                           │ 72px
│  [Progress] [Back] [Next]                  │
└────────────────────────────────────────────┘
```

**Z-index Layering**:
- Backdrop: `z-40`
- Modal: `z-50`
- Tooltips: `z-60`

---

## Interaction Patterns

### 1. Wizard Navigation

**Forward Navigation**:
```
Step 1 (Workspace)
  ↓
  Validates workspace configured
  ↓
  "Next" button enabled
  ↓
  Click "Next" → Advance to Step 2
  ↓
Step 2 (Auth)
  ...
```

**Backward Navigation**:
- "Back" button always enabled (except on Step 1)
- Returns to previous step
- Does NOT re-validate previous step
- Preserves form state

**Skip/Jump Navigation**:
- Cannot skip forward (must complete in order)
- Can jump backward using stepper dots
- Clicking completed step in stepper jumps to that step

**Validation Rules**:
```typescript
const canAdvanceToNextStep = {
  workspace: () => workspaceState.status === 'configured',
  auth: () => authState.status === 'authenticated' && authState.hasLogAnalyticsReader,
  'diagnostic-settings': () => diagnosticState.configuredServices === diagnosticState.totalServices,
  verification: () => false, // No next step
}
```

### 2. Auto-Detection & Polling

**Polling Strategy**:
```typescript
const pollingConfig = {
  // Poll setup state while guide is open and step is active
  interval: 5000, // 5 seconds
  enabled: (isOpen: boolean, isActive: boolean) => isOpen && isActive,
  
  // Back off if no changes detected
  backoff: {
    maxInterval: 15000, // 15 seconds
    multiplier: 1.5,
  },
}
```

**Auto-Advance**:
- When validation passes during polling, show toast notification
- "Next" button pulses to draw attention
- Do NOT auto-advance (user may be reading)
- Toast example: "✓ Workspace detected! You can now continue."

### 3. Copy Interactions

**Copy Button States**:
1. **Default**: "Copy" icon + label
2. **Hover**: Highlight background
3. **Click**: 
   - Immediately copy to clipboard
   - Show "Copied!" with checkmark
   - Green color for 2 seconds
   - Return to default state

**Keyboard Shortcut**:
- Focus on code block + `Ctrl+C` / `Cmd+C` copies entire snippet

### 4. Collapsible Sections

**Behavior**:
```tsx
// Default state: All collapsed except critical sections
defaultExpanded = {
  'Bicep Example': true,
  'azure.yaml Configuration': true,
  'What is Log Analytics?': false,
  'Create Workspace': false,
}

// Click anywhere in header to toggle
// Smooth height animation (150ms ease-out)
// Save expanded state to localStorage
```

**Animation**:
```css
transition: height 150ms ease-out;
overflow: hidden;
```

**Indicator**:
- Chevron icon rotates 180deg when expanded
- `▼` (down) = Collapsed
- `▲` (up) = Expanded

### 5. Error Handling

**Inline Errors** (validation failures):
```tsx
<div className="mt-2 p-3 rounded-lg bg-red-50 border border-red-200">
  <div className="flex items-start gap-2">
    <AlertCircle className="w-4 h-4 text-red-500 mt-0.5" />
    <div className="text-sm text-red-700">
      {errorMessage}
    </div>
  </div>
</div>
```

**API Errors** (network failures):
```tsx
<div className="flex flex-col items-center py-8">
  <XCircle className="w-12 h-12 text-red-500 mb-3" />
  <h3 className="text-lg font-medium mb-2">Failed to check status</h3>
  <p className="text-sm text-slate-600 mb-4">{error.message}</p>
  <Button onClick={retry}>Try Again</Button>
</div>
```

### 6. Loading States

**Skeleton Loaders**:
```tsx
// While fetching setup state
<div className="space-y-3">
  <div className="h-8 bg-slate-200 rounded animate-pulse" />
  <div className="h-20 bg-slate-200 rounded animate-pulse" />
  <div className="h-20 bg-slate-200 rounded animate-pulse" />
</div>
```

**Spinner for Actions**:
```tsx
<Button disabled={loading}>
  {loading ? (
    <>
      <Loader2 className="w-4 h-4 animate-spin mr-2" />
      Checking...
    </>
  ) : (
    <>
      <RefreshCw className="w-4 h-4 mr-2" />
      Recheck
    </>
  )}
</Button>
```

### 7. Success Celebration (Step 4)

**Visual Treatment**:
```tsx
<div className="flex flex-col items-center py-12">
  {/* Animated success icon */}
  <div className="relative">
    <div className="absolute inset-0 bg-emerald-200 rounded-full animate-ping opacity-75" />
    <div className="relative w-16 h-16 bg-emerald-100 rounded-full flex items-center justify-center">
      <CheckCircle className="w-10 h-10 text-emerald-600" />
    </div>
  </div>
  
  {/* Success message */}
  <h3 className="text-2xl font-bold text-slate-900 mt-6 mb-2">
    All set! 🎉
  </h3>
  <p className="text-slate-600 text-center max-w-md mb-6">
    Your Azure logs are configured and ready to stream.
  </p>
  
  {/* Primary CTA */}
  <Button size="lg" onClick={onComplete}>
    View Logs
  </Button>
</div>
```

---

## State Management

### 1. State Architecture

```typescript
// Global wizard state (in AzureSetupGuide)
interface WizardState {
  // Navigation
  currentStepIndex: number
  completedSteps: Set<SetupStep>
  canGoBack: boolean
  canGoForward: boolean
  
  // Data
  setupState: SetupStateResponse | null
  lastFetched: Date | null
  
  // UI
  isPolling: boolean
  isSaving: boolean
  error: string | null
}

// Per-step state (in individual step components)
interface StepState<T> {
  status: 'idle' | 'loading' | 'success' | 'error'
  data: T | null
  error: string | null
  isValidated: boolean
}
```

### 2. State Persistence

**localStorage Schema**:
```typescript
interface PersistedWizardProgress {
  version: '1.0'
  timestamp: string
  currentStep: SetupStep
  completedSteps: SetupStep[]
  detectedWorkspaceId?: string
  expandedSections: Record<string, boolean>
}

const STORAGE_KEY = 'azd-app:azure-setup-progress'
const EXPIRY_HOURS = 24

// Save on each step completion
const saveProgress = (state: WizardState) => {
  const progress: PersistedWizardProgress = {
    version: '1.0',
    timestamp: new Date().toISOString(),
    currentStep: STEPS[state.currentStepIndex],
    completedSteps: Array.from(state.completedSteps),
    detectedWorkspaceId: state.setupState?.workspace.workspaceId,
    expandedSections: getExpandedSections(),
  }
  localStorage.setItem(STORAGE_KEY, JSON.stringify(progress))
}

// Restore on mount
const restoreProgress = (): WizardState | null => {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (!stored) return null
  
  const progress = JSON.parse(stored) as PersistedWizardProgress
  
  // Check expiry
  const age = Date.now() - new Date(progress.timestamp).getTime()
  if (age > EXPIRY_HOURS * 60 * 60 * 1000) {
    localStorage.removeItem(STORAGE_KEY)
    return null
  }
  
  return {
    currentStepIndex: STEPS.indexOf(progress.currentStep),
    completedSteps: new Set(progress.completedSteps),
    // ... restore other fields
  }
}

// Clear on successful completion
const clearProgress = () => {
  localStorage.removeItem(STORAGE_KEY)
}
```

### 3. API Integration

**Polling Hook**:
```typescript
function useSetupStatePolling(isActive: boolean) {
  const [state, setState] = useState<SetupStateResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const intervalRef = useRef<NodeJS.Timeout | null>(null)
  
  const fetchSetupState = useCallback(async () => {
    try {
      const response = await fetch('/api/azure/logs/setup-state')
      if (!response.ok) throw new Error(response.statusText)
      const data = await response.json()
      setState(data)
      setError(null)
    } catch (err) {
      setError(err.message)
    }
  }, [])
  
  useEffect(() => {
    if (!isActive) {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
      return
    }
    
    // Immediate fetch
    fetchSetupState()
    
    // Poll every 5 seconds
    intervalRef.current = setInterval(fetchSetupState, 5000)
    
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
      }
    }
  }, [isActive, fetchSetupState])
  
  return { state, error, refetch: fetchSetupState }
}
```

### 4. Optimistic Updates

**Example: Sign-in flow**:
```typescript
const handleSignIn = async () => {
  // Optimistic update
  setAuthState(prev => ({ ...prev, status: 'checking' }))
  
  try {
    const response = await fetch('/api/azure/auth/login', { method: 'POST' })
    if (!response.ok) throw new Error('Sign-in failed')
    
    // Refetch setup state to get actual auth status
    await refetchSetupState()
  } catch (err) {
    // Revert on error
    setAuthState(prev => ({ ...prev, status: 'not-authenticated', error: err.message }))
  }
}
```

---

## Accessibility Plan

### 1. WCAG AA Compliance Checklist

#### ✓ Perceivable

**1.1 Text Alternatives**:
- [ ] All status icons have `aria-label`
- [ ] Code copy buttons have `aria-label="Copy command"`
- [ ] Stepper step indicators have `aria-label="Step 1: Workspace"`

**1.3 Adaptable**:
- [ ] Stepper uses semantic `<nav role="navigation">`
- [ ] Status badges use appropriate ARIA roles
- [ ] Collapsible sections use `aria-expanded`

**1.4 Distinguishable**:
- [ ] Color contrast ≥4.5:1 for text (WCAG AA)
- [ ] Status conveyed with icons + text (not color alone)
- [ ] Focus indicators visible on all interactive elements

#### ✓ Operable

**2.1 Keyboard Accessible**:
- [ ] Tab order follows visual flow
- [ ] Escape key closes modal
- [ ] Enter key submits forms / advances steps
- [ ] Arrow keys navigate stepper (optional)

**2.4 Navigable**:
- [ ] Skip links provided (skip to content)
- [ ] Modal traps focus correctly
- [ ] Clear heading hierarchy (h2 for modal title, h3 for step titles)

**2.5 Input Modalities**:
- [ ] Click targets ≥44x44px (mobile)
- [ ] Hover/focus states on all buttons

#### ✓ Understandable

**3.1 Readable**:
- [ ] Language attribute set (`lang="en"`)
- [ ] Plain language in error messages
- [ ] Technical terms explained on first use

**3.2 Predictable**:
- [ ] Navigation consistent across steps
- [ ] No automatic changes on focus
- [ ] Changes announced to screen readers

**3.3 Input Assistance**:
- [ ] Error messages clearly identify problem
- [ ] Success states announced
- [ ] Context-sensitive help available

#### ✓ Robust

**4.1 Compatible**:
- [ ] Valid HTML5 markup
- [ ] ARIA roles used correctly
- [ ] Tested with screen readers (NVDA, JAWS, VoiceOver)

### 2. Screen Reader Support

**Live Regions**:
```tsx
// Announce step changes
<div role="status" aria-live="polite" aria-atomic="true" className="sr-only">
  {`Step ${currentStepIndex + 1} of ${totalSteps}: ${currentStepTitle}`}
</div>

// Announce validation results
<div role="alert" aria-live="assertive" className="sr-only">
  {validationMessage}
</div>

// Announce polling updates
<div role="status" aria-live="polite" className="sr-only">
  {autoDetectionMessage}
</div>
```

**Semantic Stepper**:
```tsx
<nav aria-label="Setup progress">
  <ol className="flex items-center">
    {steps.map((step, index) => (
      <li key={step.id} aria-current={index === currentStepIndex ? 'step' : undefined}>
        <button
          aria-label={`Step ${index + 1}: ${step.label}. ${
            completedSteps.has(step.id) ? 'Completed' : 
            index === currentStepIndex ? 'Current' : 
            'Not started'
          }`}
          aria-disabled={index > currentStepIndex}
          onClick={() => jumpToStep(index)}
        >
          {step.label}
        </button>
      </li>
    ))}
  </ol>
</nav>
```

**Focus Management**:
```typescript
// On step change, focus step heading
useEffect(() => {
  if (stepHeadingRef.current) {
    stepHeadingRef.current.focus()
  }
}, [currentStepIndex])

// On modal open, focus close button
useEffect(() => {
  if (isOpen && closeButtonRef.current) {
    closeButtonRef.current.focus()
  }
}, [isOpen])

// On modal close, return focus to trigger
const returnFocusRef = useRef<HTMLElement | null>(null)

const openModal = (triggerElement: HTMLElement) => {
  returnFocusRef.current = triggerElement
  setIsOpen(true)
}

const closeModal = () => {
  setIsOpen(false)
  returnFocusRef.current?.focus()
}
```

### 3. Keyboard Navigation

**Key Bindings**:
```typescript
const handleKeyDown = (e: KeyboardEvent) => {
  switch (e.key) {
    case 'Escape':
      // Close modal (with confirmation if dirty)
      handleClose()
      break
      
    case 'ArrowLeft':
      // Go to previous step (if available)
      if (canGoBack) goToPreviousStep()
      break
      
    case 'ArrowRight':
      // Go to next step (if validated)
      if (canGoForward) goToNextStep()
      break
      
    case 'Tab':
      // Trap focus within modal
      trapFocus(e)
      break
  }
}
```

**Focus Trap**:
```typescript
function useFocusTrap(isActive: boolean) {
  const containerRef = useRef<HTMLElement>(null)
  
  useEffect(() => {
    if (!isActive || !containerRef.current) return
    
    const focusableElements = containerRef.current.querySelectorAll(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    )
    
    const firstElement = focusableElements[0] as HTMLElement
    const lastElement = focusableElements[focusableElements.length - 1] as HTMLElement
    
    const handleTab = (e: KeyboardEvent) => {
      if (e.key !== 'Tab') return
      
      if (e.shiftKey && document.activeElement === firstElement) {
        e.preventDefault()
        lastElement.focus()
      } else if (!e.shiftKey && document.activeElement === lastElement) {
        e.preventDefault()
        firstElement.focus()
      }
    }
    
    document.addEventListener('keydown', handleTab)
    return () => document.removeEventListener('keydown', handleTab)
  }, [isActive])
  
  return containerRef
}
```

### 4. Testing Plan

**Manual Testing**:
- [ ] Navigate entire wizard with keyboard only
- [ ] Test with NVDA on Windows
- [ ] Test with JAWS on Windows
- [ ] Test with VoiceOver on macOS
- [ ] Test with VoiceOver on iOS
- [ ] Test with TalkBack on Android
- [ ] Verify color contrast in dev tools
- [ ] Test with high contrast mode

**Automated Testing**:
```typescript
// Using @axe-core/react or jest-axe
import { axe } from 'jest-axe'

it('should have no accessibility violations', async () => {
  const { container } = render(
    <AzureSetupGuide isOpen={true} onClose={jest.fn()} />
  )
  const results = await axe(container)
  expect(results).toHaveNoViolations()
})
```

**User Testing**:
- [ ] Test with users who rely on screen readers
- [ ] Test with users with motor impairments (keyboard only)
- [ ] Test with users with low vision (high contrast, zoom)

---

## Implementation Checklist

### Phase 1: Core Components (Tasks 3-7)

#### Task 3: Setup Guide Shell
- [ ] Create `AzureSetupGuide.tsx` with modal structure
- [ ] Implement stepper navigation component
- [ ] Add progress persistence (localStorage)
- [ ] Implement focus trap and escape key handler
- [ ] Add backdrop and overlay styles
- [ ] Create step registry and navigation logic
- [ ] Test: Modal opens/closes, stepper navigation works

#### Task 4: Workspace Step
- [ ] Create `WorkspaceSetupStep.tsx`
- [ ] Implement status badge with auto-detection
- [ ] Add collapsible sections (What is Log Analytics, etc.)
- [ ] Create Bicep code snippet with copy button
- [ ] Create azure.yaml snippet with copy button
- [ ] Add recheck button with loading state
- [ ] Implement validation logic
- [ ] Test: Status updates, snippets copy, validation works

#### Task 5: Auth Step
- [ ] Create `AuthSetupStep.tsx`
- [ ] Implement auth status checklist
- [ ] Add sign-in button (integrate with backend)
- [ ] Add permission check with role assignment command
- [ ] Create collapsible "How to assign role" section
- [ ] Add retest button
- [ ] Test: Auth flow, permission check, role assignment snippet

#### Task 6: Diagnostic Settings Step
- [ ] Create `DiagnosticSettingsStep.tsx`
- [ ] Implement service status table
- [ ] Add per-service Bicep examples
- [ ] Create expandable Bicep sections
- [ ] Add "Show All Bicep" toggle
- [ ] Implement progress summary (X of Y configured)
- [ ] Add filter controls (Show All / Show Incomplete)
- [ ] Test: Table rendering, expand/collapse, filtering

#### Task 7: Verification Step
- [ ] Create `SetupVerification.tsx`
- [ ] Implement verification checklist
- [ ] Add per-service log flow status
- [ ] Create sample log preview
- [ ] Implement waiting state (5-15 min delay)
- [ ] Create success celebration UI
- [ ] Add "View Logs" completion button
- [ ] Add "Continue Anyway" for waiting state
- [ ] Test: Verification flow, success state, waiting state

#### Shared Components
- [ ] Create `CodeSnippet.tsx` with syntax highlighting
- [ ] Implement copy button with feedback
- [ ] Create `StatusBadge.tsx` with all variants
- [ ] Create `CollapsibleSection.tsx` with animation
- [ ] Create `ChecklistItem.tsx` for verification
- [ ] Test: Code copy, badge rendering, collapsing

### Phase 2: Integration (Tasks 8-11)

- [ ] Update `ModeToggle.tsx` to open setup guide
- [ ] Update `ConsoleView.tsx` to manage guide state
- [ ] Update `DiagnosticsModal.tsx` to deep link to guide
- [ ] Update `AzureErrorDisplay.tsx` to link to guide
- [ ] Test: Deep linking, error state navigation

### Phase 3: Polish

- [ ] Implement auto-refresh during setup
- [ ] Add toast notifications for auto-detection
- [ ] Polish animations and transitions
- [ ] Responsive design for mobile
- [ ] Dark mode verification
- [ ] Accessibility audit (manual + automated)
- [ ] Performance optimization (lazy loading)
- [ ] Test: Full E2E flow, mobile, accessibility

### Phase 4: Documentation

- [ ] Write JSDoc comments for all components
- [ ] Create Storybook stories (optional)
- [ ] Update main Azure logs documentation
- [ ] Create troubleshooting guide
- [ ] Add setup guide screenshots
- [ ] Test: Documentation accuracy

---

## Design Handoff Notes

### For Developers

1. **Start with Shared Components**: Build `CodeSnippet`, `StatusBadge`, `CollapsibleSection` first
2. **Use Existing Patterns**: Follow `DiagnosticsModal.tsx` for modal structure
3. **Match Color System**: Use existing Tailwind color classes from `ModeToggle.tsx`
4. **Accessibility First**: Implement ARIA from the start, don't add later
5. **Test Incrementally**: Test each component in isolation before integration

### Design Decisions

1. **Why 4 steps?**: Matches natural Azure setup flow (resource → auth → config → verify)
2. **Why stepper?**: Clear progress indicator, allows jumping back
3. **Why collapsible sections?**: Progressive disclosure, reduces initial overwhelm
4. **Why auto-detection?**: Reduces manual recheck burden, faster feedback
5. **Why localStorage?**: Preserve progress across page reloads, better UX

### Known Challenges

1. **Polling Performance**: May impact battery on mobile. Consider visibility API.
2. **Long Code Snippets**: May need virtualization for large Bicep templates.
3. **Mobile Stepper**: May need to redesign for narrow screens.
4. **Dark Mode**: Ensure all status colors have dark variants.
5. **Network Failures**: Need comprehensive error handling for all API calls.

### Open Questions for PM/Engineering

1. Should we allow skipping steps for advanced users?
2. Should we persist progress across devices (backend storage)?
3. Should we track analytics (step completion rates, drop-off points)?
4. Should we support custom resource types beyond Container Apps?
5. Should we auto-generate Bicep files (Phase 2 feature)?

---

## Appendix: Visual Mockups

### Stepper States

```
┌─────────────────────────────────────────────────────────────┐
│                     Azure Logs Setup                    [X] │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────┐         ┌──────┐         ┌──────┐         ┌──────┐ │
│  │  ✓   │────────▶│  ○   │────────▶│  ○   │────────▶│  ○   │ │
│  └──────┘  Workspace   Auth    Diagnostics   Verify         │
│  Complete                                                    │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│  [Step Content Here]                                        │
└─────────────────────────────────────────────────────────────┘
```

### Code Snippet Example

```
┌─────────────────────────────────────────────────────────────┐
│  Bicep Example - Log Analytics Workspace            Copy   │
├─────────────────────────────────────────────────────────────┤
│  1  output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = \      │
│  2    monitoring.outputs.logAnalyticsWorkspaceId            │
│  3  output AZURE_LOG_ANALYTICS_WORKSPACE_NAME string = \    │
│  4    monitoring.outputs.logAnalyticsWorkspaceName          │
│  5  output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string = \    │
│  6    monitoring.outputs.logAnalyticsWorkspaceGuid          │
│                                                             │
│  [Show full monitoring module ▼]                            │
└─────────────────────────────────────────────────────────────┘
```

### Service Table Example

```
┌─────────────────────────────────────────────────────────────┐
│  Diagnostic Settings                         2 of 3 ✓       │
├─────────────────────────────────────────────────────────────┤
│  Service    Type              Status         Action         │
├─────────────────────────────────────────────────────────────┤
│  api        Container App     ✓ Configured   [Show Bicep]  │
│  web        Container App     ✓ Configured   [Show Bicep]  │
│  database   PostgreSQL        ○ Missing      [Show Bicep]  │
└─────────────────────────────────────────────────────────────┘
```

### Success State

```
┌─────────────────────────────────────────────────────────────┐
│                          ┌────┐                             │
│                          │ ✓  │ (animated)                   │
│                          └────┘                             │
│                                                             │
│                      All set! 🎉                            │
│                                                             │
│          Your Azure logs are configured and ready.          │
│                                                             │
│                  [View Logs →]                              │
└─────────────────────────────────────────────────────────────┘
```

---

**End of Design Specification**

Ready for developer handoff to implement Tasks 3-7.

For questions or clarifications, refer to:
- [setup-guide-spec.md](setup-guide-spec.md) - Full feature specification
- [setup-guide-tasks.md](setup-guide-tasks.md) - Task breakdown
- [DiagnosticsModal.tsx](../../../cli/dashboard/src/components/DiagnosticsModal.tsx) - Reference implementation
