# Keyboard Shortcuts Modal - Component Specification

## Overview

Modal dialog displaying all keyboard shortcuts grouped by category. Triggered by pressing `?` key or clicking the Help icon.

## Visual Structure

```
┌──────────────────────────────────────────────────────┐
│ Keyboard Shortcuts                              [X]  │
├──────────────────────────────────────────────────────┤
│                                                      │
│  NAVIGATION                                          │
│  ┌────┐ Resources view                               │
│  │ 1  │                                              │
│  └────┘                                              │
│  ┌────┐ Console view                                 │
│  │ 2  │                                              │
│  └────┘                                              │
│  ┌────┐ Metrics view                                 │
│  │ 3  │                                              │
│  └────┘                                              │
│  ┌────┐ Environment view                             │
│  │ 4  │                                              │
│  └────┘                                              │
│  ┌────┐ Actions view                                 │
│  │ 5  │                                              │
│  └────┘                                              │
│  ┌────┐ Dependencies view                            │
│  │ 6  │                                              │
│  └────┘                                              │
│                                                      │
│  ACTIONS                                             │
│  ┌────┐ Refresh all services                         │
│  │ R  │                                              │
│  └────┘                                              │
│  ┌────┐ Clear console logs                           │
│  │ C  │                                              │
│  └────┘                                              │
│  ┌────┐ Export logs                                  │
│  │ E  │                                              │
│  └────┘                                              │
│  ┌──────────┐ Focus search input                     │
│  │ / or ⌘F │                                         │
│  └──────────┘                                        │
│                                                      │
│  VIEWS                                               │
│  ┌────┐ Toggle table/grid view                       │
│  │ T  │                                              │
│  └────┘                                              │
│  ┌────┐ Show this modal                              │
│  │ ?  │                                              │
│  └────┘                                              │
│  ┌──────┐ Close dialogs/modals                       │
│  │ Esc  │                                            │
│  └──────┘                                            │
│                                                      │
└──────────────────────────────────────────────────────┘
```

## Component Structure

```tsx
interface KeyboardShortcutsProps {
  isOpen: boolean
  onClose: () => void
}

interface Shortcut {
  key: string | string[]  // Single key or array for alternatives
  description: string
  category: 'navigation' | 'actions' | 'views'
}
```

## Sub-components

### KeyBadge
Visual representation of a keyboard key.

```tsx
interface KeyBadgeProps {
  keys: string | string[]  // Single key or alternatives
}
```

**Visual States:**
- Default: `bg-muted border border-border rounded px-2 py-1 font-mono text-xs`
- Multiple keys: Show with "or" separator

### ShortcutRow
Single shortcut entry with key badge and description.

```tsx
interface ShortcutRowProps {
  shortcut: Shortcut
}
```

### ShortcutGroup
Category group header with list of shortcuts.

```tsx
interface ShortcutGroupProps {
  title: string
  shortcuts: Shortcut[]
}
```

## Shortcuts Data

```typescript
const shortcuts: Shortcut[] = [
  // Navigation
  { key: '1', description: 'Resources view', category: 'navigation' },
  { key: '2', description: 'Console view', category: 'navigation' },
  { key: '3', description: 'Metrics view', category: 'navigation' },
  { key: '4', description: 'Environment view', category: 'navigation' },
  { key: '5', description: 'Actions view', category: 'navigation' },
  { key: '6', description: 'Dependencies view', category: 'navigation' },
  
  // Actions
  { key: 'R', description: 'Refresh all services', category: 'actions' },
  { key: 'C', description: 'Clear console logs', category: 'actions' },
  { key: 'E', description: 'Export logs', category: 'actions' },
  { key: ['/', 'Ctrl+F'], description: 'Focus search input', category: 'actions' },
  
  // Views
  { key: 'T', description: 'Toggle table/grid view', category: 'views' },
  { key: '?', description: 'Show this modal', category: 'views' },
  { key: 'Esc', description: 'Close dialogs/modals', category: 'views' },
]
```

## Accessibility

**Dialog Accessibility:**
- `role="dialog"`
- `aria-modal="true"`
- `aria-labelledby` pointing to modal title
- Focus trapped within modal when open
- Close on Escape key

**Keyboard Navigation:**
- Tab navigates through close button
- Enter/Space activates close button
- Escape closes modal

**Screen Reader:**
- Modal title announced on open
- Shortcut descriptions readable
- Key badges have proper text alternatives

## Responsive Design

**Desktop (≥768px):**
- Modal width: 480px max
- Three columns of shortcut groups

**Tablet/Mobile (<768px):**
- Modal width: full width with margins
- Single column layout
- Larger touch targets for close button

## Visual Design

**Modal Container:**
- Background: `bg-background`
- Border: `border border-border`
- Shadow: `shadow-xl`
- Border radius: `rounded-lg`
- Max height: `max-h-[80vh]` with overflow scroll

**Header:**
- Title: `text-lg font-semibold`
- Close button: `X` icon, hover state
- Border bottom: `border-b border-border`
- Padding: `p-4`

**Content:**
- Padding: `p-4`
- Gap between groups: `gap-6`

**Group Header:**
- Text: `text-xs font-semibold text-muted-foreground uppercase tracking-wider`
- Margin bottom: `mb-2`

**Shortcut Row:**
- Flex layout: `flex items-center gap-3`
- Padding: `py-1`

**Key Badge:**
- Background: `bg-muted`
- Border: `border border-border`
- Radius: `rounded`
- Padding: `px-2 py-0.5`
- Font: `font-mono text-xs`
- Min width: `min-w-[2rem]` for single keys
- Text align: center

## Animation

**Modal Open:**
- Fade in backdrop: `animate-fade-in` (200ms)
- Scale up modal: `animate-scale-in` (200ms)

**Modal Close:**
- Fade out backdrop
- Scale down modal

## Platform-specific Keys

Display platform-appropriate key names:
- macOS: ⌘ (Command), ⌥ (Option), ⌃ (Control)
- Windows/Linux: Ctrl, Alt

```typescript
const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0

const formatKey = (key: string): string => {
  if (key === 'Ctrl+F') {
    return isMac ? '⌘F' : 'Ctrl+F'
  }
  return key
}
```

## Integration Points

**App.tsx:**
- State for modal visibility: `useState<boolean>`
- Global keydown listener for `?` key
- Render KeyboardShortcuts component

**Header:**
- Help icon button (HelpCircle) triggers modal
- Tooltip: "Keyboard shortcuts (?)"

## Test Cases

1. Modal opens on `?` key press
2. Modal opens on Help icon click
3. Modal closes on Escape key
4. Modal closes on backdrop click
5. Modal closes on X button click
6. All shortcuts displayed correctly
7. Correct platform key names shown
8. Focus trapped within modal
9. Screen reader announces modal
10. Responsive layout on mobile
