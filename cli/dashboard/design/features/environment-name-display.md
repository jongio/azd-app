# Environment Name Display - UI Specification

**Feature:** Display the current Azure environment name (from `AZURE_ENV_NAME`) in the dashboard header.

**Status:** Specification  
**Designer:** GitHub Copilot  
**Date:** 2026-01-23  
**Target Component:** `Header.tsx`

---

## 1. Overview

The dashboard header should display the active Azure environment name to provide immediate context about which environment (dev, staging, prod, etc.) the user is working with. This is critical for preventing accidental operations in the wrong environment.

**Current Context:**
- Environment variable `AZURE_ENV_NAME` is available via azd runtime
- Backend `/api/environment` endpoint exists but currently only returns Codespace info
- Header already displays: Project name, navigation pills, connection status, service health, utilities

---

## 2. Placement Recommendation

### Option A: **Brand Zone - After Project Name** ✅ **RECOMMENDED**

**Rationale:**
- Keeps environment context near the primary identifier (project name)
- Follows natural reading order: "Project Name | Environment"
- Doesn't clutter utility zone (right side)
- Provides immediate context without competing for attention

**Visual layout:**
```
┌────────────────────────────────────────────────────────────────────┐
│ 🚀 MyProject · dev    [Console] [Services] [Environment]    ●●● ⚙  │
└────────────────────────────────────────────────────────────────────┘
   ↑            ↑                    ↑                         ↑
 Brand      Env Name            Navigation                  Utilities
```

### Option B: Utility Zone - Before Service Status (Not Recommended)

**Issues:**
- Right side getting crowded
- Less prominent - environment is critical context
- Breaks visual flow of status → settings utilities

---

## 3. Visual Design

### 3.1. Component Style: **Subtle Badge**

**Design pattern:** Follows existing badge/pill patterns from `StatusIndicator.tsx`

```tsx
// Visual appearance
┌──────────┐
│ · dev    │  ← Dot + Text in muted pill
└──────────┘
```

**Styling Details:**

| Property | Value | Rationale |
|----------|-------|-----------|
| **Background** | `bg-slate-100 dark:bg-slate-800/50` | Muted, non-competing |
| **Border** | `border border-slate-200/60 dark:border-slate-700/60` | Subtle definition |
| **Text Color** | `text-slate-600 dark:text-slate-300` | Readable but secondary |
| **Font** | `text-xs font-medium` | Compact, consistent with status pills |
| **Padding** | `px-2.5 py-1` | Balanced spacing |
| **Border Radius** | `rounded-lg` | Matches nav pills (8px) |
| **Dot Size** | `w-1.5 h-1.5` | StatusDot size 'sm' |
| **Dot Color** | `bg-cyan-500 dark:bg-cyan-400` | Brand color, matches active nav |

### 3.2. Typography

- **Font family:** Inherits from system (`Inter` if loaded)
- **Font weight:** `500` (medium) for environment name
- **Letter spacing:** `-0.01em` (slightly tighter for compact appearance)
- **Text transform:** `none` (preserve original case)

### 3.3. Spacing

```
Project Name  [·12px·]  Env Badge  [·16px·]  Connection Dot
```

- **Left margin from project name:** `ml-3` (12px)
- **Right margin to connection:** `mr-4` (16px)
- **Internal gap (dot to text):** `gap-1.5` (6px)

---

## 4. Component Structure

### 4.1. TypeScript Interface Updates

```typescript
// Update HeaderProps in Header.tsx
export interface HeaderProps {
  /** Project name to display */
  projectName: string
  
  /** Azure environment name (e.g., 'dev', 'staging', 'prod') */
  environmentName?: string | null  // ← NEW
  
  /** Currently active view */
  activeView: View
  
  // ... rest unchanged
}
```

### 4.2. Component Implementation

```tsx
// New sub-component in Header.tsx
interface EnvironmentBadgeProps {
  name: string
  className?: string
}

function EnvironmentBadge({ name, className }: EnvironmentBadgeProps) {
  return (
    <div
      className={cn(
        'inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg',
        'bg-slate-100 dark:bg-slate-800/50',
        'border border-slate-200/60 dark:border-slate-700/60',
        'text-xs font-medium tracking-tight',
        'text-slate-600 dark:text-slate-300',
        'transition-colors duration-150',
        className
      )}
      role="status"
      aria-label={`Environment: ${name}`}
    >
      <span 
        className="inline-block w-1.5 h-1.5 rounded-full bg-cyan-500 dark:bg-cyan-400" 
        aria-hidden="true"
      />
      <span className="whitespace-nowrap">{name}</span>
    </div>
  )
}
```

### 4.3. Integration in Header

```tsx
{/* Brand Zone - Updated */}
<div className="flex items-center gap-3 min-w-0">
  <button
    type="button"
    onClick={() => onViewChange('console')}
    className="flex items-center gap-3 min-w-0 hover:opacity-80 transition-opacity"
  >
    <div className="w-7 h-7 rounded-lg bg-gradient-to-br from-cyan-500 to-cyan-600 flex items-center justify-center shrink-0">
      <Rocket className="w-4 h-4 text-white" />
    </div>
    <h1 className="text-lg font-semibold text-slate-900 dark:text-slate-100 truncate tracking-tight">
      {projectName || 'Dashboard'}
    </h1>
  </button>
  
  {/* NEW: Environment Badge */}
  {environmentName && (
    <EnvironmentBadge 
      name={environmentName} 
      className="hidden sm:inline-flex"  // Hide on mobile
    />
  )}
  
  {/* Connection Indicator */}
  {connected && (
    <span className="relative flex h-2 w-2" aria-hidden="true">
      <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
      <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
    </span>
  )}
</div>
```

---

## 5. Responsive Behavior

### Desktop (≥768px - `md:` breakpoint)
- Full display with dot + text
- Positioned after project name with proper spacing
- No truncation needed (environment names are typically short)

### Tablet (640-767px - `sm:` breakpoint)
- Same as desktop
- Use `hidden sm:inline-flex` to show on small screens and up

### Mobile (<640px)
- **Hidden** to preserve header real estate
- Environment info available in Environment tab
- Critical space reserved for navigation pills

**Breakpoint strategy:**
```tsx
className="hidden sm:inline-flex"  // Show on ≥640px (sm+)
```

---

## 6. States & Edge Cases

### 6.1. State Matrix

| State | Display | Behavior |
|-------|---------|----------|
| **Environment name loaded** | `· dev` | Show badge normally |
| **Environment name is empty string** | (hidden) | Don't render badge |
| **Environment name is null/undefined** | (hidden) | Don't render badge |
| **Environment name loading** | (hidden) | Don't show loading state (not critical) |
| **Very long name (>20 chars)** | `· very-long-envir...` | Truncate with ellipsis (rare) |

### 6.2. Handling States in Code

```tsx
{/* Conditional rendering - only show if we have a name */}
{environmentName && environmentName.trim() !== '' && (
  <EnvironmentBadge 
    name={environmentName} 
    className="hidden sm:inline-flex max-w-[140px] truncate"
  />
)}
```

### 6.3. Edge Cases

**Long environment names:**
- Add `max-w-[140px]` constraint
- Apply `truncate` class for ellipsis
- Use `title` attribute for full name on hover

```tsx
<span 
  className="whitespace-nowrap truncate max-w-[100px]" 
  title={name}
>
  {name}
</span>
```

**Special characters:**
- Environment names follow azd naming rules (alphanumeric + hyphen)
- No sanitization needed (azd validates)

**Missing environment:**
- Common during initial setup before `azd init`
- Acceptable to not display anything (graceful degradation)

---

## 7. Icon Selection

### Primary: **Dot (circle)** ✅ **SELECTED**

**Rationale:**
- Minimal, doesn't compete for attention
- Consistent with existing StatusDot pattern
- Brand cyan color reinforces active environment

### Alternative Options (Not Recommended)

| Icon | Reason Against |
|------|---------------|
| `Tag` | Too prominent, suggests interactivity |
| `Cloud` | Implies cloud vs environment distinction |
| `Database` | Environment ≠ database |
| `Server` | Too technical, doesn't convey multi-env |
| `Layers` | Not intuitive for environment concept |

---

## 8. Color Palette

### Environment Badge Colors

**Default theme (all environments):**
```tsx
// Background
light: 'bg-slate-100'
dark:  'bg-slate-800/50'

// Border
light: 'border-slate-200/60'
dark:  'border-slate-700/60'

// Text
light: 'text-slate-600'
dark:  'text-slate-300'

// Dot (brand accent)
light: 'bg-cyan-500'
dark:  'bg-cyan-400'
```

**Future enhancement (environment-specific colors):**
If needed in future, could differentiate by environment:

```tsx
const ENV_COLORS = {
  prod: {
    dot: 'bg-rose-500 dark:bg-rose-400',
    text: 'text-rose-700 dark:text-rose-300'
  },
  staging: {
    dot: 'bg-amber-500 dark:bg-amber-400', 
    text: 'text-amber-700 dark:text-amber-300'
  },
  dev: {
    dot: 'bg-cyan-500 dark:bg-cyan-400',
    text: 'text-cyan-700 dark:text-cyan-300'
  }
}
```

**Recommendation:** Start with neutral styling for MVP. Add color coding only if user research shows value.

---

## 9. Accessibility

### 9.1. ARIA Attributes

```tsx
<div
  role="status"
  aria-label={`Environment: ${name}`}
>
  <span aria-hidden="true">{/* dot */}</span>
  <span>{name}</span>
</div>
```

### 9.2. Screen Readers

- Badge announces as: "Environment: dev"
- Decorative dot hidden with `aria-hidden="true"`
- No interactive elements (not a button)

### 9.3. Keyboard Navigation

- Not focusable (informational only)
- No keyboard interactions required

### 9.4. Color Contrast

**WCAG AA compliance:**
- Text: `slate-600` on `slate-100` → 4.8:1 ✅
- Text (dark): `slate-300` on `slate-800` → 9.2:1 ✅
- Dot serves as visual accent, not sole indicator

---

## 10. Backend Integration

### 10.1. API Endpoint Extension

**Extend `/api/environment` to include AZURE_ENV_NAME:**

```go
// In handleGetEnvironment (server_handlers.go)
func (s *Server) handleGetEnvironment(w http.ResponseWriter, r *http.Request) {
	// Existing Codespace detection...
	
	// NEW: Add Azure environment name
	azureEnvName := os.Getenv("AZURE_ENV_NAME")
	
	response := map[string]interface{}{
		"codespace": map[string]interface{}{
			"enabled":         codespaceName != "",
			"name":            codespaceName,
			"domain":          codespacePortDomain,
			"isVsCodeDesktop": isVsCodeDesktop,
		},
		"azure": map[string]interface{}{
			"environmentName": azureEnvName,  // NEW
		},
	}
	
	WriteJSONSuccess(w, response)
}
```

### 10.2. TypeScript Type Update

```typescript
// In codespace-utils.ts or new file azure-env-utils.ts
export interface AzureEnvironment {
  environmentName: string
}

export interface EnvironmentInfo {
  codespace: CodespaceConfig
  azure: AzureEnvironment  // NEW
}
```

### 10.3. Data Fetching

```tsx
// In App.tsx or new useAzureEnv hook
const [environmentName, setEnvironmentName] = useState<string | null>(null)

useEffect(() => {
  async function fetchEnvironment() {
    try {
      const response = await fetch('/api/environment')
      const data = await response.json() as EnvironmentInfo
      setEnvironmentName(data.azure?.environmentName || null)
    } catch (err) {
      console.warn('Failed to fetch environment:', err)
      setEnvironmentName(null)
    }
  }
  fetchEnvironment()
}, [])
```

### 10.4. Caching Strategy

- **Cache in sessionStorage** alongside Codespace config
- **TTL:** 5 minutes (environment rarely changes during session)
- **Invalidation:** On manual refresh or app reload

---

## 11. Testing Scenarios

### 11.1. Unit Tests

```typescript
describe('EnvironmentBadge', () => {
  it('renders with environment name', () => {
    render(<EnvironmentBadge name="dev" />)
    expect(screen.getByText('dev')).toBeInTheDocument()
  })
  
  it('applies correct ARIA label', () => {
    render(<EnvironmentBadge name="staging" />)
    expect(screen.getByRole('status')).toHaveAccessibleName('Environment: staging')
  })
  
  it('truncates very long names', () => {
    const longName = 'very-long-environment-name-exceeding-limits'
    render(<EnvironmentBadge name={longName} />)
    // Check max-width constraint is applied
  })
})
```

### 11.2. Integration Tests (E2E)

```typescript
test('displays environment name in header', async ({ page }) => {
  await mockEnvironmentAPI(page, { 
    azure: { environmentName: 'dev' } 
  })
  
  await page.goto('http://localhost:3000')
  
  const envBadge = page.locator('[role="status"][aria-label*="Environment"]')
  await expect(envBadge).toBeVisible()
  await expect(envBadge).toContainText('dev')
})

test('hides environment badge when name is missing', async ({ page }) => {
  await mockEnvironmentAPI(page, { 
    azure: { environmentName: null } 
  })
  
  await page.goto('http://localhost:3000')
  
  const envBadge = page.locator('[role="status"][aria-label*="Environment"]')
  await expect(envBadge).not.toBeVisible()
})
```

### 11.3. Visual Regression Tests

**Chromatic snapshots:**
- Header with environment name (light mode)
- Header with environment name (dark mode)
- Header without environment name
- Mobile view (environment hidden)

---

## 12. Implementation Checklist

### Backend Changes
- [ ] Extend `/api/environment` endpoint to include `AZURE_ENV_NAME`
- [ ] Update API response type to include `azure.environmentName`
- [ ] Test endpoint returns correct value from environment variable

### Frontend Changes
- [ ] Update `EnvironmentInfo` TypeScript interface
- [ ] Create `EnvironmentBadge` component
- [ ] Update `HeaderProps` to include `environmentName?: string | null`
- [ ] Integrate badge in Header brand zone
- [ ] Add responsive hiding (`hidden sm:inline-flex`)
- [ ] Update `useCodespaceEnv` or create new `useEnvironment` hook
- [ ] Add sessionStorage caching for environment data

### Testing
- [ ] Unit tests for `EnvironmentBadge`
- [ ] E2E test for environment display
- [ ] E2E test for missing environment
- [ ] Visual regression tests (Chromatic)
- [ ] Manual testing: dev, staging, prod environments
- [ ] Manual testing: dark mode
- [ ] Manual testing: mobile viewport

### Documentation
- [ ] Update component documentation
- [ ] Add usage examples to Storybook (if applicable)
- [ ] Update CHANGELOG.md

---

## 13. Future Enhancements (Out of Scope)

### Phase 2 Considerations:

1. **Environment switching UI**
   - Quick switcher dropdown: dev ↓ → staging, prod
   - Requires backend support for `azd env select`

2. **Environment-specific styling**
   - Red accent for prod (danger zone)
   - Yellow for staging (caution)
   - Cyan for dev (default)

3. **Environment metadata tooltip**
   - Hover shows: subscription ID, location, last deployed
   - Requires additional API data

4. **Environment health indicator**
   - Show if environment is deployed / provisioned
   - Combine with resource health

---

## 14. Design Mockups

### Desktop View (≥768px)

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│  🚀 azd-app    · dev ●       [Console] [Services] [Environment]       2⚡ 🌙 ⚙️   │
└─────────────────────────────────────────────────────────────────────────────────┘
   ↑            ↑   ↑                   ↑                                ↑
 Icon      Project  Env             Navigation                       Utilities
                    Badge
```

### Mobile View (<640px)

```
┌──────────────────────────────────────────────┐
│  🚀 Dashboard  ●    [○][□][⚙]    2⚡🌙⚙️      │
└──────────────────────────────────────────────┘
   ↑                  ↑          ↑
 Brand           Nav Icons   Utilities
(env hidden on mobile - space constrained)
```

---

## 15. Component Specification Summary

| Property | Value |
|----------|-------|
| **Component Name** | `EnvironmentBadge` |
| **Location** | `Header.tsx` (brand zone, after project name) |
| **Visibility** | Desktop/Tablet only (`hidden sm:inline-flex`) |
| **Style** | Subtle pill with dot + text |
| **Color** | Muted slate with cyan brand accent |
| **States** | Shown (with name) / Hidden (no name) |
| **Icon** | Small cyan dot (1.5×1.5px) |
| **Typography** | `text-xs font-medium` |
| **Accessibility** | `role="status"`, proper ARIA labels |
| **Data Source** | `/api/environment` → `AZURE_ENV_NAME` |
| **Caching** | SessionStorage, 5min TTL |

---

## 16. Developer Handoff Notes

**Key files to modify:**

1. **Backend:** `cli/src/internal/dashboard/server_handlers.go`
   - Update `handleGetEnvironment()` to include `AZURE_ENV_NAME`

2. **Types:** `cli/dashboard/src/lib/codespace-utils.ts`
   - Add `AzureEnvironment` interface
   - Update `EnvironmentInfo` interface

3. **Component:** `cli/dashboard/src/components/Header.tsx`
   - Add `EnvironmentBadge` sub-component
   - Update `HeaderProps` interface
   - Integrate badge in brand zone

4. **Hook:** `cli/dashboard/src/hooks/useCodespaceEnv.tsx` or new file
   - Parse `azure.environmentName` from API response
   - Consider renaming to `useEnvironmentInfo` (handles both Codespace + Azure env)

5. **App Integration:** `cli/dashboard/src/App.tsx`
   - Fetch environment data
   - Pass `environmentName` prop to Header

6. **Tests:** `cli/dashboard/e2e/helpers/test-setup.ts`
   - Update `/api/environment` mock to include `azure.environmentName`

**Design tokens (for reference):**
- Background: `bg-slate-100` / `dark:bg-slate-800/50`
- Border: `border-slate-200/60` / `dark:border-slate-700/60`
- Text: `text-slate-600` / `dark:text-slate-300`
- Accent dot: `bg-cyan-500` / `dark:bg-cyan-400`
- Spacing: `ml-3` from project name, `gap-1.5` internal

**Priority:** Medium  
**Effort:** ~2-4 hours (backend + frontend + tests)  
**Risk:** Low (additive feature, no breaking changes)

---

## Questions for Product/UX Review

1. Should we add environment-specific color coding (e.g., red for prod)?
2. Is environment switching in-scope for this feature?
3. Should mobile view show env name differently (e.g., in a menu)?
4. Do we need a "no environment" warning/state?
5. Should the badge be clickable (navigate to Environment tab)?

---

**End of Specification**
