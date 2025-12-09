# Log Source Switcher Component Design

## Overview

A compact icon-based toggle to switch between Local and Azure log sources. The switcher provides visual clarity through distinct iconography and branding colors while maintaining a small footprint in the toolbar.

---

## 1. Component Specifications

### Visual Design

```
┌─────────────────────────────────────────────────────────────────┐
│  Compact Icon Switcher (default state - Local selected)         │
│                                                                 │
│  ┌─────────────────────────────────────────────┐                │
│  │  ┌─────────┐   ┌─────────┐                   │               │
│  │  │   💻    │   │   ☁️    │                   │               │
│  │  │ Monitor │   │  Azure  │                   │               │
│  │  └─────────┘   └─────────┘                   │               │
│  │   [active]      [inactive]                   │               │
│  └─────────────────────────────────────────────┘                │
│                                                                 │
│  Dimensions: ~72px × 32px (icon-only variant)                   │
│              ~140px × 36px (with labels)                        │
└─────────────────────────────────────────────────────────────────┘
```

### Icon Selection

| Mode   | Icon                | Source   | Rationale                        |
|--------|---------------------|----------|----------------------------------|
| Local  | `Monitor` / `Laptop`| Lucide   | Desktop/local development        |
| Azure  | Custom Azure logo   | Azure    | Brand recognition, cloud service |

**Azure Icon**: Use the official Azure cloud icon or a stylized cloud with the Azure blue gradient. For simplicity, fallback to Lucide's `Cloud` icon with Azure branding colors.

---

## 2. States

### Local Mode (Active)

```
┌─────────┐  ┌─────────┐
│  💻    │  │   ☁️    │
│  ████  │  │        │
└─────────┘  └─────────┘
 Selected     Dimmed

Background: slate-200 (light) / slate-700 (dark)
Icon Color: slate-800 (light) / slate-100 (dark)
Border: 2px solid cyan-500 (focus ring on active)
```

### Azure Mode (Active)

```
┌─────────┐  ┌─────────┐
│   💻    │  │   ☁️   │
│        │  │  ████  │
└─────────┘  └─────────┘
  Dimmed     Selected

Background: azure-100 / azure-900/30
Icon Color: azure-600 (light) / azure-400 (dark)
Border: 2px solid azure-500 (focus ring on active)
```

### Disabled State (Azure unavailable)

```
┌─────────┐  ┌─────────┐
│   💻    │  │   ☁️   │
│  ████  │  │  ░░░░  │
└─────────┘  └─────────┘
 Selected    Disabled

Azure icon: 50% opacity
Cursor: not-allowed
Tooltip: "Azure logging not configured"
```

### Loading State

```
┌─────────┐  ┌─────────┐
│   💻    │  │   ⏳   │
│        │  │  ████  │
└─────────┘  └─────────┘
            Spinner

Show loading spinner in place of icon during mode switch
```

---

## 3. Color Palette

### Local Mode Colors

```css
/* Light theme */
--local-bg: var(--color-slate-100);
--local-bg-active: var(--color-white);
--local-icon: var(--color-slate-600);
--local-icon-active: var(--color-slate-900);
--local-border-active: var(--color-cyan-500);

/* Dark theme */
--local-bg-dark: var(--color-slate-800);
--local-bg-active-dark: var(--color-slate-700);
--local-icon-dark: var(--color-slate-400);
--local-icon-active-dark: var(--color-slate-100);
```

### Azure Mode Colors

```css
/* Light theme */
--azure-bg: var(--color-azure-50);
--azure-bg-active: var(--color-azure-100);
--azure-icon: var(--color-azure-500);
--azure-icon-active: var(--color-azure-600);
--azure-border-active: var(--color-azure-500);

/* Dark theme */
--azure-bg-dark: rgba(59, 130, 246, 0.1);
--azure-bg-active-dark: rgba(59, 130, 246, 0.2);
--azure-icon-dark: var(--color-azure-400);
--azure-icon-active-dark: var(--color-azure-300);
```

### Status Indicator Colors

```css
/* Connection status dot */
--status-connected: var(--color-emerald-500);    /* Green */
--status-connecting: var(--color-amber-500);     /* Yellow/Orange */
--status-disconnected: var(--color-red-500);     /* Red */
--status-disabled: var(--color-slate-400);       /* Gray */
```

---

## 4. Sizes & Variants

### Compact (Icon-only) - Recommended for toolbar

```
Width: 72px
Height: 32px
Icon size: 16px × 16px
Padding: 8px
Gap between icons: 4px
Border radius: 8px (lg)
```

### Standard (Icons with labels)

```
Width: auto (min 140px)
Height: 36px
Icon size: 16px × 16px
Label font: 12px, medium weight
Padding: 12px horizontal, 8px vertical
Gap: 6px (icon to label), 4px (between buttons)
Border radius: 8px (lg)
```

### Large (For settings panel)

```
Width: auto (min 180px)
Height: 44px
Icon size: 20px × 20px
Label font: 14px, medium weight
Padding: 16px horizontal, 12px vertical
```

---

## 5. Interactions

### Hover

- Background lightens/darkens slightly
- Cursor: pointer (unless disabled)
- Transform: scale(1.02) subtle grow

### Active/Click

- Transform: scale(0.98) subtle press
- Immediate visual feedback

### Focus (Keyboard)

- Focus ring: 2px solid cyan-500/azure-500
- Offset: 2px
- No outline on mouse click (`:focus-visible` only)

### Transition

```css
transition: all 150ms ease-out;
transform-origin: center;
```

---

## 6. Accessibility

### ARIA Attributes

```html
<div role="radiogroup" aria-label="Log source">
  <button
    role="radio"
    aria-checked="true|false"
    aria-label="View local logs"
    tabindex="0"
  >
    <Monitor aria-hidden="true" />
    <span class="sr-only">Local</span>
  </button>
  <button
    role="radio"
    aria-checked="true|false"
    aria-label="View Azure logs"
    aria-disabled="true|false"
    tabindex="-1|0"
  >
    <Cloud aria-hidden="true" />
    <span class="sr-only">Azure</span>
  </button>
</div>
```

### Screen Reader Announcements

- Mode change: "Switched to [Local|Azure] logs"
- Error: "Azure logging is not configured"
- Loading: "Switching log source..."

### Keyboard Navigation

- `Tab`: Move focus to/from component
- `ArrowLeft`/`ArrowRight`: Switch between options
- `Space`/`Enter`: Select focused option
- `Ctrl+Shift+M`: Global shortcut to cycle modes

---

## 7. Responsive Behavior

| Breakpoint | Behavior                           |
|------------|------------------------------------|
| < 640px    | Icon-only, labels hidden           |
| ≥ 640px    | Standard with labels (optional)    |
| ≥ 1024px   | Full width with connection status  |

---

## 8. Azure Icon Design Options

### Option A: Official Azure Logo (SVG)
Use the official Microsoft Azure logo icon (simplified cloud shape in Azure blue).

### Option B: Stylized Cloud with Azure Colors

```svg
<svg viewBox="0 0 24 24" fill="none">
  <!-- Cloud shape with Azure gradient -->
  <defs>
    <linearGradient id="azure-gradient" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#0078D4"/>
      <stop offset="100%" stop-color="#50E6FF"/>
    </linearGradient>
  </defs>
  <path
    d="M6.5 20a5 5 0 0 1-.37-9.98A7 7 0 0 1 19.73 11a4.5 4.5 0 0 1 .77 8.95"
    fill="url(#azure-gradient)"
    stroke="currentColor"
    stroke-width="1.5"
  />
</svg>
```

### Option C: Lucide Cloud with Azure Styling
Use Lucide's `Cloud` icon but apply Azure brand colors.

**Recommendation**: Option C for consistency with existing icon library, with Azure blue (`#0078D4`) as the primary color.

---

## 9. Component API

```typescript
interface LogSourceSwitcherProps {
  /** Current log source mode */
  mode: 'local' | 'azure'
  /** Whether Azure logging is enabled/available */
  azureEnabled?: boolean
  /** Azure connection status */
  azureStatus?: 'connected' | 'disconnected' | 'connecting' | 'disabled'
  /** Loading state during mode switch */
  isLoading?: boolean
  /** Size variant */
  size?: 'compact' | 'standard' | 'large'
  /** Show text labels */
  showLabels?: boolean
  /** Show connection status indicator */
  showStatus?: boolean
  /** Callback when mode changes */
  onModeChange?: (mode: 'local' | 'azure') => void
  /** Additional class names */
  className?: string
}
```

---

## 10. Placement in UI

### Primary Location: ConsoleView Toolbar

```
┌─────────────────────────────────────────────────────────────────────────┐
│  [Pause] [Clear] [Auto-scroll] │ Search...          │ [💻|☁️] [⚙] [⛶] │
│                                │                    │                    │
│  ↑ Actions                     ↑ Search             ↑ Mode + View       │
└─────────────────────────────────────────────────────────────────────────┘
```

Position the switcher in the right section of the toolbar, before the settings and fullscreen buttons, after the view mode toggle (grid/list).

### Secondary Locations

1. **Service Detail Panel**: Per-service log source override
2. **Settings Dialog**: Default mode preference
3. **Header Status Bar**: Global indicator (minimal)

---

## 11. Implementation Notes

1. Prefer the compact icon-only variant in the main toolbar
2. Show labels on hover via tooltip
3. Include connection status dot only when Azure is enabled
4. Persist user preference to session storage
5. Respect `azure.yaml` default on fresh page load
6. Animate icon transition when switching modes
