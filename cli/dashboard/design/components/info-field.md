# InfoField Component Specification

## Document Info
- **Component**: ui/InfoField
- **Status**: Design Specification
- **Created**: 2024-12-01
- **Author**: UX Design

---

## 1. Overview

The InfoField component is a reusable UI primitive for displaying label/value pairs throughout the dashboard. It provides consistent styling for information display with optional copy-to-clipboard functionality.

### Use Cases
- **ServiceDetailPanel**: Display service properties (Status, Port, URL, PID, etc.)
- **EnvironmentPanel**: Display environment variable key/value pairs
- **Azure Tab**: Display resource information (Resource Group, Location, Subscription ID)

---

## 2. Component Breakdown

### 2.1 Visual Structure

```
┌─────────────────────────────────────────────────────────────┐
│  ┌─────────────────────────────────────────────────────────┐│
│  │ [Label]                                                 ││
│  │ [Value]                                    [Copy Icon]  ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘

Detailed Anatomy:
┌─────────────────────────────────────────────────────────────┐
│  Label (muted, smaller text)                               │
│  ┌─────────────────────────────────────┐ ┌───────────────┐ │
│  │ Value (primary text, selectable)    │ │ [📋] / [✓]   │ │
│  └─────────────────────────────────────┘ └───────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Sub-Elements

| Element | Description | Required |
|---------|-------------|----------|
| **Container** | Wrapper element with consistent padding and spacing | Yes |
| **Label** | Text label describing the value | Yes |
| **Value** | The displayed content (text or ReactNode) | Yes |
| **ValueWrapper** | Flex container for value + copy button | Yes |
| **CopyButton** | Icon button for copy action | No (conditional) |
| **CopyIcon** | Visual icon state (Copy → Check) | No (conditional) |

---

## 3. Props and Interface

```typescript
interface InfoFieldProps {
  /** Label text describing the value */
  label: string
  
  /** Value to display - can be text or custom ReactNode */
  value: string | React.ReactNode
  
  /** Enable copy-to-clipboard functionality */
  copyable?: boolean
  
  /** Callback fired when copy is triggered */
  onCopy?: () => void
  
  /** 
   * Value to copy (if different from displayed value)
   * Used when value is a ReactNode but you need to copy specific text
   */
  copyValue?: string
  
  /** Additional class names for the container */
  className?: string
  
  /** Data test ID for testing */
  'data-testid'?: string
}
```

### 3.1 Default Values

| Prop | Default |
|------|---------|
| `copyable` | `false` |
| `onCopy` | `undefined` |
| `copyValue` | `undefined` (uses `value` if string) |

---

## 4. Visual States

### 4.1 State Matrix

| State | Label | Value | Copy Button | Description |
|-------|-------|-------|-------------|-------------|
| **Default** | `text-muted-foreground` | `text-foreground` | `text-muted-foreground` | Resting state |
| **Hover (Row)** | unchanged | unchanged | visible (if copyable) | Subtle background hint |
| **Hover (Copy)** | unchanged | unchanged | `text-foreground` | Copy button highlighted |
| **Focus (Copy)** | unchanged | unchanged | focus ring | Keyboard navigation |
| **Active (Copy)** | unchanged | unchanged | `scale(0.95)` | Button press feedback |
| **Copied** | unchanged | unchanged | `text-success` + checkmark | Success state (2s timeout) |
| **Disabled** | `opacity-50` | `opacity-50` | hidden | Inactive state |

### 4.2 State Visuals

```
DEFAULT STATE:
┌─────────────────────────────────────────┐
│  Port                                   │   Label: text-muted-foreground
│  3000                           [📋]   │   Value: text-foreground
└─────────────────────────────────────────┘   Button: text-muted-foreground, opacity-0 → opacity-100 on hover

HOVER STATE (Row):
┌─────────────────────────────────────────┐
│  Port                                   │   bg-secondary/30 (subtle highlight)
│  3000                           [📋]   │   Button: opacity-100
└─────────────────────────────────────────┘

HOVER STATE (Copy Button):
┌─────────────────────────────────────────┐
│  Port                                   │
│  3000                           [📋]   │   Button: text-foreground, bg-secondary
└─────────────────────────────────────────┘

FOCUS STATE (Copy Button):
┌─────────────────────────────────────────┐
│  Port                                   │
│  3000                          [[📋]]  │   ring-2 ring-ring ring-offset-2
└─────────────────────────────────────────┘

COPIED STATE:
┌─────────────────────────────────────────┐
│  Port                                   │
│  3000                           [✓]    │   Icon: Check, color: text-success
└─────────────────────────────────────────┘   Auto-resets after 2 seconds
```

---

## 5. Interactions

### 5.1 Mouse Interactions

| Action | Target | Result |
|--------|--------|--------|
| Hover row | Container | Subtle background highlight, copy button appears |
| Hover copy | Copy button | Button background highlight |
| Click copy | Copy button | Copy to clipboard, icon → checkmark, `onCopy` callback |
| Double-click value | Value text | Text selection (native browser behavior) |

### 5.2 Keyboard Interactions

| Key | Target | Result |
|-----|--------|--------|
| `Tab` | Copy button | Focus moves to copy button (if focusable) |
| `Enter` / `Space` | Copy button (focused) | Trigger copy action |
| `Escape` | Copy button (focused) | Blur focus |

### 5.3 Interaction Flow

```
User Flow: Copy Value
┌───────────────────────────────────────────────────────────┐
│  1. User hovers over InfoField row                        │
│     → Row gets subtle highlight                           │
│     → Copy button fades in (if copyable=true)             │
│                                                           │
│  2. User clicks copy button                               │
│     → Value copied to clipboard                           │
│     → Copy icon transitions to checkmark                  │
│     → onCopy callback fired (if provided)                 │
│                                                           │
│  3. After 2 seconds                                       │
│     → Checkmark transitions back to copy icon             │
└───────────────────────────────────────────────────────────┘

User Flow: Keyboard Copy
┌───────────────────────────────────────────────────────────┐
│  1. User tabs to copy button                              │
│     → Focus ring appears                                  │
│                                                           │
│  2. User presses Enter or Space                           │
│     → Same as click action above                          │
│                                                           │
│  3. User presses Escape                                   │
│     → Focus moves away                                    │
└───────────────────────────────────────────────────────────┘
```

---

## 6. Accessibility

### 6.1 WCAG 2.1 AA Compliance

| Criterion | Implementation |
|-----------|----------------|
| **1.3.1 Info & Relationships** | Label associated with value via visual proximity and semantic grouping |
| **1.4.3 Contrast (Minimum)** | All text meets 4.5:1 ratio (muted-foreground on background) |
| **1.4.11 Non-text Contrast** | Copy button icon meets 3:1 ratio |
| **2.1.1 Keyboard** | Copy button fully keyboard accessible |
| **2.4.6 Headings and Labels** | Clear, descriptive labels |
| **2.4.7 Focus Visible** | Visible focus ring on copy button |
| **4.1.2 Name, Role, Value** | Button has accessible name |

### 6.2 ARIA Implementation

```tsx
// Container
<div
  className="info-field"
  data-testid={testId}
>
  
  // Label
  <span 
    id={`${id}-label`}
    className="info-field-label"
  >
    {label}
  </span>
  
  // Value Wrapper
  <div 
    className="info-field-value-wrapper"
    aria-labelledby={`${id}-label`}
  >
    // Value
    <span className="info-field-value">
      {value}
    </span>
    
    // Copy Button (when copyable)
    {copyable && (
      <button
        type="button"
        onClick={handleCopy}
        aria-label={copied ? `${label} copied` : `Copy ${label}`}
        aria-live="polite"
        className="info-field-copy-button"
      >
        {copied ? <Check /> : <Copy />}
      </button>
    )}
  </div>
</div>
```

### 6.3 Screen Reader Announcements

| Action | Announcement |
|--------|--------------|
| Focus copy button | "Copy [label], button" |
| Copy success | "[label] copied" (via aria-label update + aria-live) |
| Value read | "[label]: [value]" (when navigating by region) |

### 6.4 Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  .info-field-copy-button {
    transition: none;
  }
  
  .info-field-copy-button svg {
    transition: none;
  }
}
```

---

## 7. Design Tokens

### 7.1 Colors

| Token | Light Theme | Dark Theme | Usage |
|-------|-------------|------------|-------|
| `--foreground` | `var(--gray-900)` | `var(--slate-50)` | Value text |
| `--muted-foreground` | `var(--gray-600)` | `var(--slate-400)` | Label text, copy button default |
| `--secondary` | `var(--gray-100)` | `var(--slate-800)` | Hover backgrounds |
| `--success` | `var(--green-700)` | `var(--green-500)` | Copied state icon |
| `--ring` | `var(--purple-600)` | `var(--purple-400)` | Focus ring |
| `--border` | `var(--gray-200)` | `var(--slate-700)` | Optional separator |

### 7.2 Typography

| Element | Font Size | Font Weight | Line Height |
|---------|-----------|-------------|-------------|
| Label | `text-xs` (12px) | `font-medium` (500) | `leading-none` |
| Value | `text-sm` (14px) | `font-normal` (400) | `leading-normal` |

### 7.3 Spacing

| Property | Value | Token |
|----------|-------|-------|
| Container padding (vertical) | `py-2` | 8px |
| Container padding (horizontal) | `px-0` | 0 (content determines width) |
| Label-to-value gap | `gap-1` | 4px |
| Value-to-button gap | `gap-2` | 8px |
| Copy button size | `h-7 w-7` | 28px × 28px |
| Copy icon size | `h-4 w-4` | 16px × 16px |

### 7.4 Effects

| Property | Value | Usage |
|----------|-------|-------|
| Transition duration | `200ms` | All interactive states |
| Transition timing | `ease` | Smooth state changes |
| Focus ring width | `2px` | Keyboard focus indicator |
| Focus ring offset | `2px` | Ring offset from element |
| Copy button hover bg | `bg-secondary` | Hover state background |
| Row hover bg | `bg-secondary/30` | Subtle row highlight |

---

## 8. Component Variants

### 8.1 Default Variant
Standard label/value display without copy functionality.

```tsx
<InfoField
  label="Status"
  value="Running"
/>
```

### 8.2 Copyable Variant
Includes copy button with clipboard functionality.

```tsx
<InfoField
  label="Port"
  value="3000"
  copyable
/>
```

### 8.3 With Custom Value
ReactNode value for complex displays (badges, links, etc.).

```tsx
<InfoField
  label="Health"
  value={<Badge variant="success">Healthy</Badge>}
/>
```

### 8.4 Copyable with Custom Copy Value
When displayed value differs from what should be copied.

```tsx
<InfoField
  label="URL"
  value={<a href={url}>{url}</a>}
  copyable
  copyValue={url}
/>
```

### 8.5 With Copy Callback
Custom action on copy (e.g., toast notification).

```tsx
<InfoField
  label="Subscription ID"
  value="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  copyable
  onCopy={() => toast('Subscription ID copied!')}
/>
```

---

## 9. Responsive Behavior

### 9.1 Layout Adaptation

The InfoField component uses a flexible layout that adapts to container width:

```
WIDE CONTAINER (default):
┌─────────────────────────────────────────────────────────────┐
│  Port                                                       │
│  3000                                               [📋]   │
└─────────────────────────────────────────────────────────────┘

NARROW CONTAINER (value wraps):
┌─────────────────────────────────┐
│  URL                            │
│  https://example.azurewebsite   │
│  s.net/api/v1                   │
│                         [📋]   │
└─────────────────────────────────┘
```

### 9.2 Breakpoint Behavior

| Breakpoint | Behavior |
|------------|----------|
| Mobile (<640px) | Copy button always visible, larger touch target (min 44px) |
| Tablet (640-1023px) | Copy button appears on hover/focus |
| Desktop (≥1024px) | Default behavior |

---

## 10. Usage Guidelines

### 10.1 Do's

- ✅ Use for key/value information display
- ✅ Enable `copyable` for user-actionable values (URLs, IDs, commands)
- ✅ Provide descriptive labels
- ✅ Use `copyValue` when displaying formatted/linked values
- ✅ Combine with `useClipboard` hook for copy state management

### 10.2 Don'ts

- ❌ Don't use for interactive form inputs (use Input component)
- ❌ Don't make everything copyable (only useful values)
- ❌ Don't use overly long labels (keep concise)
- ❌ Don't nest complex interactive elements in value

### 10.3 Content Guidelines

| Element | Guidelines |
|---------|------------|
| **Label** | 1-3 words, sentence case, no punctuation |
| **Value** | Concise, may truncate with ellipsis if needed |

---

## 11. Implementation Reference

### 11.1 Basic Implementation

```tsx
import * as React from 'react'
import { Copy, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

interface InfoFieldProps {
  label: string
  value: string | React.ReactNode
  copyable?: boolean
  onCopy?: () => void
  copyValue?: string
  className?: string
  'data-testid'?: string
}

export function InfoField({
  label,
  value,
  copyable = false,
  onCopy,
  copyValue,
  className,
  'data-testid': testId,
}: InfoFieldProps) {
  const [copied, setCopied] = React.useState(false)
  const id = React.useId()

  const handleCopy = async () => {
    const textToCopy = copyValue ?? (typeof value === 'string' ? value : '')
    if (!textToCopy) return

    try {
      await navigator.clipboard.writeText(textToCopy)
      setCopied(true)
      onCopy?.()
      setTimeout(() => setCopied(false), 2000)
    } catch (error) {
      console.error('Failed to copy:', error)
    }
  }

  return (
    <div
      className={cn(
        'group py-2',
        'transition-colors duration-200',
        'hover:bg-secondary/30 rounded-md -mx-2 px-2',
        className
      )}
      data-testid={testId}
    >
      <span
        id={`${id}-label`}
        className="block text-xs font-medium text-muted-foreground mb-1"
      >
        {label}
      </span>
      
      <div 
        className="flex items-center justify-between gap-2"
        aria-labelledby={`${id}-label`}
      >
        <span className="text-sm text-foreground break-all">
          {value}
        </span>
        
        {copyable && (typeof value === 'string' || copyValue) && (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={handleCopy}
            aria-label={copied ? `${label} copied` : `Copy ${label}`}
            aria-live="polite"
            className={cn(
              'h-7 w-7 shrink-0',
              'opacity-0 group-hover:opacity-100 focus:opacity-100',
              'transition-all duration-200',
              copied && 'text-success'
            )}
          >
            {copied ? (
              <Check className="h-4 w-4" />
            ) : (
              <Copy className="h-4 w-4" />
            )}
          </Button>
        )}
      </div>
    </div>
  )
}
```

### 11.2 Integration with useClipboard Hook

```tsx
// For managing copy state across multiple InfoFields
import { useClipboard } from '@/hooks/useClipboard'

function ServiceDetails({ service }) {
  const { copiedField, copyToClipboard } = useClipboard()

  return (
    <div>
      <InfoField
        label="Port"
        value={service.port}
        copyable
        onCopy={() => copyToClipboard(service.port, 'port')}
      />
      <InfoField
        label="URL"
        value={service.url}
        copyable
        onCopy={() => copyToClipboard(service.url, 'url')}
      />
    </div>
  )
}
```

---

## 12. Testing Checklist

### 12.1 Unit Tests

- [ ] Renders label and value correctly
- [ ] Copy button hidden when `copyable=false`
- [ ] Copy button visible when `copyable=true`
- [ ] Clicking copy button copies value to clipboard
- [ ] Icon changes from Copy to Check on success
- [ ] Icon resets after 2 second timeout
- [ ] `onCopy` callback fires on copy
- [ ] `copyValue` used when provided (instead of value)
- [ ] Custom className applied correctly

### 12.2 Accessibility Tests

- [ ] Copy button has accessible name
- [ ] Focus ring visible on keyboard navigation
- [ ] Enter/Space triggers copy action
- [ ] Screen reader announces copied state
- [ ] Color contrast meets WCAG AA

### 12.3 Integration Tests

- [ ] Works within ServiceDetailPanel
- [ ] Works within EnvironmentPanel
- [ ] Works with useClipboard hook
- [ ] Responsive behavior at all breakpoints

---

## 13. Related Components

| Component | Relationship |
|-----------|--------------|
| **Button** | Copy button uses ghost variant |
| **Badge** | May be used as value content |
| **Input** | For editable values (not InfoField) |
| **Table** | InfoField for vertical layout; Table for tabular data |
| **useClipboard** | Hook for managing copy state |

---

## Appendix A: Token Quick Reference

```css
/* InfoField Tokens */
.info-field {
  /* Container */
  padding: theme('spacing.2') 0;          /* py-2 */
  border-radius: theme('borderRadius.md'); /* rounded-md */
  
  /* Hover state */
  --hover-bg: theme('colors.secondary / 30%');
}

.info-field-label {
  font-size: theme('fontSize.xs');        /* 12px */
  font-weight: theme('fontWeight.medium'); /* 500 */
  color: var(--muted-foreground);
  margin-bottom: theme('spacing.1');      /* 4px */
}

.info-field-value {
  font-size: theme('fontSize.sm');        /* 14px */
  color: var(--foreground);
}

.info-field-copy-button {
  height: theme('spacing.7');             /* 28px */
  width: theme('spacing.7');              /* 28px */
  transition-duration: 200ms;
}

.info-field-copy-button:focus-visible {
  ring-width: 2px;
  ring-color: var(--ring);
  ring-offset: 2px;
}

.info-field-copy-button--copied {
  color: var(--success);
}
```
