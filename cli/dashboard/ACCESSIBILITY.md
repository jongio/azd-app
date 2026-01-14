# Accessibility Implementation for Azure YAML Editor

## Overview

This document provides guidance and best practices for maintaining WCAG AA compliance in the Azure YAML Editor components.

## WCAG AA Compliance Checklist

### ✅ Keyboard Navigation

- **Tab Order**: Logical sequence through all interactive elements
- **Skip Links**: Provided at top of page to skip to main content areas
- **Modal Trapping**: Tab stays within open modals (using focus-trap.ts utility)
- **Keyboard Shortcuts**: All shortcuts documented in KeyboardShortcutsReference component
- **No Keyboard Traps**: All modals and dialogs can be closed with Escape

#### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Cmd/Ctrl+K` | Open command palette |
| `Cmd/Ctrl+S` | Save configuration |
| `Cmd/Ctrl+P` | Toggle preview pane |
| `Cmd/Ctrl+B` | Toggle navigation sidebar |
| `Cmd/Ctrl+F` | Search in navigation |
| `Cmd/Ctrl+N` | Add new service |
| `Escape` | Close modal/dialog/dropdown |
| `Enter` | Submit form/execute action |
| `Tab` | Next field |
| `Shift+Tab` | Previous field |
| `Arrow Keys` | Navigate lists/menus/tree |

### ✅ Screen Reader Support

#### ARIA Labels

All interactive elements have descriptive ARIA labels:

```tsx
// ✅ Good - Button has aria-label
<button aria-label="Close dialog">
  <X className="w-4 h-4" />
</button>

// ❌ Bad - Icon button without label
<button>
  <X className="w-4 h-4" />
</button>
```

#### Form Labels

All form fields have associated labels (not just placeholders):

```tsx
// ✅ Good - Proper label association
<label htmlFor="service-name">Service Name</label>
<input id="service-name" name="name" />

// ❌ Bad - Placeholder only
<input placeholder="Service Name" />
```

#### Live Regions

Dynamic content is announced via `aria-live`:

```tsx
// Success/error messages
<div role="status" aria-live="polite">
  Configuration saved successfully
</div>

// Critical errors
<div role="alert" aria-live="assertive">
  Failed to save: Permission denied
</div>
```

Use the `announcer` utility:

```tsx
import { useAnnouncer } from '@/lib/accessibility'

function MyComponent() {
  const announcer = useAnnouncer()
  
  const handleSave = async () => {
    try {
      await saveConfig()
      announcer.announceSuccess('Configuration saved')
    } catch (error) {
      announcer.announceError('Failed to save configuration')
    }
  }
}
```

### ✅ Visual Accessibility

#### Color Contrast

All text meets **4.5:1 minimum contrast ratio** (WCAG AA):

- **Body text**: `--foreground` on `--background` (16.8:1 in light mode, 15.4:1 in dark mode)
- **Secondary text**: `--foreground-secondary` (9.5:1 in light, 9.8:1 in dark)
- **Error text**: `--destructive` (4.5:1 minimum)
- **Success text**: `--success` (4.9:1 in light, 6.8:1 in dark)
- **Warning text**: Dark text on yellow background for contrast

Use design system colors to ensure compliance:

```tsx
// ✅ Good - Using design system colors
<div className="text-foreground bg-background">
  Main content
</div>

<div className="text-destructive bg-background">
  Error message
</div>

// ❌ Bad - Custom colors without contrast check
<div style={{ color: '#aaa', background: '#fff' }}>
  Low contrast text
</div>
```

#### Focus Indicators

All interactive elements have **2px outline with 3:1 minimum contrast**:

```css
*:focus-visible {
  outline: 2px solid var(--ring);
  outline-offset: 2px;
}
```

Never remove focus styles:

```tsx
// ❌ Bad - Removing focus outline
<button className="focus:outline-none">
  Click me
</button>

// ✅ Good - Custom focus with sufficient contrast
<button className="focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2">
  Click me
</button>
```

#### No Color-Only Indicators

Errors, warnings, and success states use **icons + text + color**:

```tsx
// ✅ Good - Icon + text + color
<div className="flex items-center gap-2 text-destructive">
  <AlertCircle className="w-4 h-4" />
  <span>Error: Name is required</span>
</div>

// ❌ Bad - Color only
<div className="text-red-500">
  Name is required
</div>
```

### ✅ Testing Requirements

#### Automated Testing with axe-core

All components should pass axe-core tests:

```tsx
import { runWCAGAA, assertNoViolations } from '@/lib/accessibility/axe-testing'

describe('MyComponent', () => {
  it('should have no accessibility violations', async () => {
    const { container } = render(<MyComponent />)
    const result = await runWCAGAA(container)
    assertNoViolations(result)
  })
})
```

#### Manual Testing Checklist

1. **Keyboard Navigation**
   - [ ] Can reach all interactive elements with Tab
   - [ ] Tab order is logical
   - [ ] Can close modals with Escape
   - [ ] Can submit forms with Enter
   - [ ] No keyboard traps

2. **Screen Reader (NVDA/VoiceOver)**
   - [ ] All buttons/links announced correctly
   - [ ] Form labels associated properly
   - [ ] Error messages announced
   - [ ] Navigation landmarks present
   - [ ] Dynamic content updates announced

3. **Visual**
   - [ ] All text readable at 200% zoom
   - [ ] Focus indicators visible
   - [ ] Error states not color-only
   - [ ] High contrast mode works
   - [ ] Reduced motion respected

## Accessibility Utilities

### Focus Trap

Use for modals and dialogs:

```tsx
import { useFocusTrap } from '@/lib/accessibility'

function MyModal({ isOpen, onClose }) {
  const ref = useRef<HTMLDivElement>(null)
  
  useFocusTrap(ref, isOpen, {
    onEscape: onClose,
  })
  
  return (
    <div ref={ref} role="dialog">
      {/* Modal content */}
    </div>
  )
}
```

### Screen Reader Announcer

Use for dynamic updates:

```tsx
import { useAnnouncer } from '@/lib/accessibility'

function MyComponent() {
  const announcer = useAnnouncer()
  
  // Success
  announcer.announceSuccess('Configuration saved')
  
  // Error
  announcer.announceError('Failed to save')
  
  // Warning
  announcer.announceWarning('Port conflict detected')
  
  // Navigation
  announcer.announceNavigation('Services')
  
  // Modal
  announcer.announceModal(true, 'Add Service')
}
```

### Keyboard Shortcuts

Register global shortcuts:

```tsx
import { useKeyboardShortcuts } from '@/lib/accessibility'

function MyComponent() {
  useKeyboardShortcuts([
    {
      id: 'save',
      key: 's',
      modifiers: { ctrl: true },
      description: 'Save configuration',
      action: handleSave,
      category: 'editor',
    },
  ])
}
```

### Skip Links

Add to top of page:

```tsx
import { SkipLinks, DEFAULT_SKIP_LINKS } from '@/lib/accessibility'

function App() {
  return (
    <>
      <SkipLinks links={DEFAULT_SKIP_LINKS} />
      {/* Rest of app */}
    </>
  )
}
```

## Common Accessibility Issues and Fixes

### Issue: Icon button without label

```tsx
// ❌ Bad
<button onClick={onClose}>
  <X className="w-4 h-4" />
</button>

// ✅ Good
<button onClick={onClose} aria-label="Close dialog">
  <X className="w-4 h-4" />
</button>
```

### Issue: Form input without label

```tsx
// ❌ Bad
<input placeholder="Service name" />

// ✅ Good
<label htmlFor="service-name">Service Name</label>
<input id="service-name" placeholder="e.g., api" />
```

### Issue: Dynamic content not announced

```tsx
// ❌ Bad
<div>{errorMessage}</div>

// ✅ Good
<div role="alert" aria-live="assertive">
  {errorMessage}
</div>

// Or use announcer utility
announcer.announceError(errorMessage)
```

### Issue: Modal not trapping focus

```tsx
// ❌ Bad
<div className="modal">
  <button>Action</button>
</div>

// ✅ Good
import { useFocusTrap } from '@/lib/accessibility'

const ref = useRef<HTMLDivElement>(null)
useFocusTrap(ref, isOpen, { onEscape: onClose })

<div ref={ref} role="dialog" aria-modal="true">
  <button>Action</button>
</div>
```

### Issue: Color-only error indicator

```tsx
// ❌ Bad
<span className="text-red-500">{error}</span>

// ✅ Good
<span className="flex items-center gap-2 text-destructive">
  <AlertCircle className="w-4 h-4" aria-hidden="true" />
  {error}
</span>
```

## Screen Reader Testing Guide

### NVDA (Windows)

1. Download from: https://www.nvaccess.org/download/
2. Start NVDA
3. Navigate to editor with Tab
4. Verify:
   - All buttons/links read correctly
   - Form fields have labels
   - Errors announced
   - Dynamic updates announced

### VoiceOver (macOS)

1. Enable: System Preferences → Accessibility → VoiceOver
2. Start: Cmd+F5
3. Navigate with: Ctrl+Option+Arrow keys
4. Verify same as NVDA

### JAWS (Windows)

1. Commercial software: https://www.freedomscientific.com/products/software/jaws/
2. Start JAWS
3. Navigate with Tab and arrow keys
4. Verify same as NVDA

## Resources

- [WCAG 2.1 Quick Reference](https://www.w3.org/WAI/WCAG21/quickref/?currentsidebar=%23col_customize&levels=aa)
- [axe DevTools Browser Extension](https://www.deque.com/axe/devtools/)
- [ARIA Authoring Practices Guide](https://www.w3.org/WAI/ARIA/apg/)
- [WebAIM Color Contrast Checker](https://webaim.org/resources/contrastchecker/)
- [NVDA Screen Reader](https://www.nvaccess.org/)
- [VoiceOver User Guide](https://support.apple.com/guide/voiceover/welcome/mac)

## Maintenance

1. Run axe-core tests on all new components
2. Test with keyboard navigation before merging
3. Verify screen reader compatibility for major changes
4. Check color contrast when adding new colors
5. Update this guide when adding new accessibility patterns
