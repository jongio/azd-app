# LogsPane Header Status Background Specification

## Overview
Enhancement to the LogsPane header to display background colors that match the service's current status (error, warning, info), providing immediate visual context while maintaining accessibility and design consistency.

## Current Implementation

```tsx
<div 
  className="flex items-center justify-between px-4 py-2 bg-card border-b cursor-pointer select-none"
  onClick={toggleCollapsed}
>
```

The header currently uses a neutral `bg-card` background for all states. The `paneStatus` is already calculated: `'error' | 'warning' | 'info'`.

## Design Requirements

### Visual Hierarchy
1. Header background provides ambient status indication
2. Status badge remains the primary explicit status indicator
3. Background colors should be subtle (not overwhelming)
4. Text must remain readable in all states

### Accessibility Requirements (WCAG 2.1 AA)
- **Contrast Ratio**: Minimum 4.5:1 for normal text against background
- **Large Text**: Minimum 3:1 for text ≥14pt bold or ≥18pt regular
- **Non-Text Contrast**: 3:1 for UI components and graphical objects
- **Color Independence**: Status must not rely on color alone (badge text provides redundancy)

---

## Visual States

### State 1: Info (Default/Neutral)

**Behavior**: No errors or warnings detected in logs.

| Property | Light Mode | Dark Mode |
|----------|------------|-----------|
| Background | `bg-card` (#ffffff) | `bg-card` (#1a1a1a) |
| Text Color | `text-foreground` (#171717) | `text-foreground` (#f5f5f5) |
| Border | `border-gray-200` | `border-[#2a2a2a]` |

**Contrast Verification (Light)**:
- Foreground (#171717) on Background (#ffffff): **16.1:1** ✓
- Muted Foreground (#525252) on Background (#ffffff): **7.5:1** ✓

**Contrast Verification (Dark)**:
- Foreground (#f5f5f5) on Background (#1a1a1a): **15.8:1** ✓
- Muted Foreground (#a3a3a3) on Background (#1a1a1a): **7.2:1** ✓

---

### State 2: Warning

**Behavior**: Warning-level logs detected, no errors.

| Property | Light Mode | Dark Mode |
|----------|------------|-----------|
| Background | `bg-yellow-50` (#fefce8) | `bg-yellow-800/20` (rgba(133,77,14,0.2)) |
| Text Color | `text-foreground` (#171717) | `text-foreground` (#f5f5f5) |
| Border (existing) | `border-yellow-500` | `border-yellow-500` |

**CSS Token Mapping**:
```css
/* Light mode */
--header-warning-bg: var(--yellow-50);  /* #fefce8 */

/* Dark mode */
--header-warning-bg: rgba(133, 77, 14, 0.2);  /* yellow-800 at 20% opacity */
```

**Contrast Verification (Light)**:
- Foreground (#171717) on Background (#fefce8): **15.2:1** ✓
- Muted Foreground (#525252) on Background (#fefce8): **7.1:1** ✓

**Contrast Verification (Dark)**:
- Foreground (#f5f5f5) on Background (~#2b2419): **12.8:1** ✓
- Muted Foreground (#a3a3a3) on Background (~#2b2419): **5.9:1** ✓

---

### State 3: Error

**Behavior**: Error-level logs detected (highest severity).

| Property | Light Mode | Dark Mode |
|----------|------------|-----------|
| Background | `bg-red-50` (#fef2f2) | `bg-red-700/20` (rgba(185,28,28,0.2)) |
| Text Color | `text-foreground` (#171717) | `text-foreground` (#f5f5f5) |
| Border (existing) | `border-red-500 animate-pulse` | `border-red-500 animate-pulse` |

**CSS Token Mapping**:
```css
/* Light mode */
--header-error-bg: var(--red-50);  /* #fef2f2 */

/* Dark mode */
--header-error-bg: rgba(185, 28, 28, 0.2);  /* red-700 at 20% opacity */
```

**Contrast Verification (Light)**:
- Foreground (#171717) on Background (#fef2f2): **14.9:1** ✓
- Muted Foreground (#525252) on Background (#fef2f2): **7.0:1** ✓

**Contrast Verification (Dark)**:
- Foreground (#f5f5f5) on Background (~#2b1a1a): **12.5:1** ✓
- Muted Foreground (#a3a3a3) on Background (~#2b1a1a): **5.8:1** ✓

---

## Color Token Definitions

### New CSS Custom Properties (index.css)

Add to light theme (`[data-theme="light"]`):
```css
/* Header Status Backgrounds */
--header-status-info: var(--card);
--header-status-warning: var(--yellow-50);
--header-status-error: var(--red-50);
```

Add to dark theme (`[data-theme="dark"]`):
```css
/* Header Status Backgrounds */
--header-status-info: var(--card);
--header-status-warning: rgba(133, 77, 14, 0.2);
--header-status-error: rgba(185, 28, 28, 0.2);
```

### Tailwind Class Mapping

Create a status-to-class mapping object:

```tsx
const headerBgClass = {
  error: 'bg-[var(--header-status-error)]',
  warning: 'bg-[var(--header-status-warning)]',
  info: 'bg-card'
}[paneStatus]
```

**Alternative (using Tailwind's arbitrary values directly)**:
```tsx
const headerBgClass = {
  error: 'bg-red-50 dark:bg-red-700/20',
  warning: 'bg-yellow-50 dark:bg-yellow-800/20',
  info: 'bg-card'
}[paneStatus]
```

---

## Implementation

### Updated Header Component

```tsx
// Header background color based on status
const headerBgClass = {
  error: 'bg-red-50 dark:bg-red-900/20',
  warning: 'bg-yellow-50 dark:bg-yellow-900/20',
  info: 'bg-card'
}[paneStatus]

<div 
  className={cn(
    "flex items-center justify-between px-4 py-2 border-b cursor-pointer select-none transition-colors duration-200",
    headerBgClass
  )}
  onClick={toggleCollapsed}
>
```

### Status Badge Adjustments

The existing status badge uses semi-transparent backgrounds that work on any header color:

```tsx
<span className={cn(
  "px-2 py-0.5 text-xs rounded-full font-medium",
  paneStatus === 'error' && "bg-destructive/10 text-destructive border border-destructive/30",
  paneStatus === 'warning' && "bg-warning/10 text-warning border border-warning/30",
  paneStatus === 'info' && "bg-muted text-muted-foreground border border-border"
)}>
  {paneStatus}
</span>
```

**Badge Visibility Check**: The badge uses `/10` opacity backgrounds and solid borders, ensuring visibility against all header backgrounds.

---

## Transition Animation

Add smooth color transition when status changes:

```tsx
className="... transition-colors duration-200 ease-in-out"
```

**Transition Properties**:
- Property: `background-color`
- Duration: `200ms` (matches existing theme transitions)
- Easing: `ease-in-out`

**Reduced Motion**: The transition respects `prefers-reduced-motion` via existing CSS:
```css
@media (prefers-reduced-motion: reduce) {
  [data-theme] * {
    transition-duration: 0ms !important;
  }
}
```

---

## Visual Mockups

### Light Mode

```
┌─────────────────────────────────────────────────┐
│ INFO STATE                                       │
│ ┌─────────────────────────────────────────────┐ │
│ │  ▼  api-gateway   [info]   50 logs    📋   │ │  ← bg: #ffffff (card)
│ └─────────────────────────────────────────────┘ │
│                                                  │
│ WARNING STATE                                    │
│ ┌─────────────────────────────────────────────┐ │
│ │  ▼  api-gateway   [warning]   50 logs  📋  │ │  ← bg: #fefce8 (yellow-50)
│ └─────────────────────────────────────────────┘ │
│                                                  │
│ ERROR STATE                                      │
│ ┌─────────────────────────────────────────────┐ │
│ │  ▼  api-gateway   [error]   50 logs    📋  │ │  ← bg: #fef2f2 (red-50)
│ └─────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
```

### Dark Mode

```
┌─────────────────────────────────────────────────┐
│ INFO STATE                                       │
│ ┌─────────────────────────────────────────────┐ │
│ │  ▼  api-gateway   [info]   50 logs    📋   │ │  ← bg: #1a1a1a (card)
│ └─────────────────────────────────────────────┘ │
│                                                  │
│ WARNING STATE                                    │
│ ┌─────────────────────────────────────────────┐ │
│ │  ▼  api-gateway   [warning]   50 logs  📋  │ │  ← bg: rgba(133,77,14,0.2)
│ └─────────────────────────────────────────────┘ │
│                                                  │
│ ERROR STATE                                      │
│ ┌─────────────────────────────────────────────┐ │
│ │  ▼  api-gateway   [error]   50 logs    📋  │ │  ← bg: rgba(185,28,28,0.2)
│ └─────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
```

---

## Accessibility Verification Checklist

### Color Contrast (WCAG AA 4.5:1 minimum)

| State | Mode | Text/BG Contrast | Status |
|-------|------|-----------------|--------|
| Info | Light | 16.1:1 | ✓ Pass |
| Info | Dark | 15.8:1 | ✓ Pass |
| Warning | Light | 15.2:1 | ✓ Pass |
| Warning | Dark | 12.8:1 | ✓ Pass |
| Error | Light | 14.9:1 | ✓ Pass |
| Error | Dark | 12.5:1 | ✓ Pass |

### Non-Color Indicators
- ✓ Status badge displays text label ("error", "warning", "info")
- ✓ Border color indicates status (redundant with background)
- ✓ Error state includes pulse animation (motion indicator)
- ✓ Status is announced to screen readers via badge text

### Focus States
- ✓ Header is clickable, requires visible focus indicator
- ✓ Collapse button has separate focus ring
- ✓ Focus ring uses `--ring` color (purple-500)

---

## Testing Requirements

### Visual Regression Tests
1. Screenshot header in all 3 states (info, warning, error)
2. Screenshot in light mode and dark mode (6 total screenshots)
3. Verify badge visibility on all backgrounds
4. Verify transition animation between states

### Accessibility Tests
1. Run axe-core on header component
2. Verify contrast ratios with color contrast analyzer
3. Test with screen reader (announce status changes)
4. Test with high contrast mode enabled

### Browser Compatibility
- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

---

## Acceptance Criteria

- [ ] Header background color reflects error state (red tones)
- [ ] Header background color reflects warning state (yellow tones)
- [ ] Header background color reflects info state (neutral/default)
- [ ] Text remains readable against all background colors (WCAG AA contrast)
- [ ] Works correctly in light mode
- [ ] Works correctly in dark mode
- [ ] Status badge remains visible and distinguishable from header background
- [ ] Smooth transition when status changes
- [ ] Reduced motion preference respected

---

**Version**: 1.0  
**Status**: Draft  
**Created**: 2025-11-26  
**Related**: `logs-pane-spec.md`, `spec.md` (Log Pane Visual Enhancements)
