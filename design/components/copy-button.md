````markdown
# Copy Button Component Specification

## Overview

The Copy Button is a reusable atom component that copies text to the clipboard and provides visual feedback. It's used within CodeBlock and Terminal components, as well as standalone for copy-able text elements.

---

## 1. Component Hierarchy

```
CopyButton (atom)
├── CopyIcon (atom)
├── CheckIcon (atom) [success state]
├── ErrorIcon (atom) [error state]
├── ButtonLabel (atom) [optional]
└── ScreenReaderAnnouncer (atom)
```

---

## 2. Props Interface

```typescript
interface CopyButtonProps {
  /** Text to copy to clipboard */
  text: string;
  /** Size variant */
  size?: 'sm' | 'md' | 'lg';
  /** Visual variant */
  variant?: 'icon' | 'button' | 'text';
  /** Position (affects styling) */
  position?: 'inline' | 'floating' | 'header';
  /** Custom label text */
  label?: string;
  /** Success message for screen readers */
  successMessage?: string;
  /** Error message for screen readers */
  errorMessage?: string;
  /** Duration to show feedback (ms) */
  feedbackDuration?: number;
  /** Callback on successful copy */
  onCopy?: () => void;
  /** Callback on copy error */
  onError?: (error: Error) => void;
  /** Custom class name */
  className?: string;
  /** Disabled state */
  disabled?: boolean;
}
```

---

## 3. States

### Button States

| State    | Trigger               | Visual Changes                        | Duration |
|----------|----------------------|---------------------------------------|----------|
| Idle     | Default              | Copy icon visible                     | -        |
| Hover    | Mouse enter          | Background highlight                  | -        |
| Focus    | Keyboard focus       | Focus ring visible                    | -        |
| Active   | Mouse down           | Scale down, darker background         | -        |
| Copied   | Successful copy      | Check icon, green, "Copied!" text     | 2000ms   |
| Error    | Copy failed          | X icon, red, "Failed" text            | 2000ms   |
| Disabled | disabled={true}      | Muted, cursor not-allowed             | -        |

### State Transitions

```
       ┌─────────────────────────────────┐
       ▼                                 │
    ┌──────┐    click    ┌────────┐      │ 2s timeout
    │ Idle │ ──────────> │ Copied │ ─────┘
    └──────┘             └────────┘
       │                      │
       │    click (fail)      │
       │         │            │
       │         ▼            │
       │    ┌───────┐         │
       └───>│ Error │─────────┘
            └───────┘
                │ 2s timeout
                ▼
            ┌──────┐
            │ Idle │
            └──────┘
```

---

## 4. Visual Specifications

### Variants

```
Icon Only (default):
┌────┐
│ 📋 │
└────┘

Button with Label:
┌──────────────┐
│  📋  Copy    │
└──────────────┘

Text Link:
Copy code

Copied State:
┌──────────────┐
│  ✓  Copied!  │
└──────────────┘

Error State:
┌──────────────┐
│  ✗  Failed   │
└──────────────┘
```

### Dimensions

| Property         | sm      | md (default) | lg      |
|------------------|---------|--------------|---------|
| Button height    | 28px    | 32px         | 40px    |
| Button min-width | 28px    | 32px         | 40px    |
| Icon size        | 14px    | 16px         | 20px    |
| Border radius    | 4px     | 6px          | 8px     |
| Padding (icon)   | 6px     | 8px          | 10px    |
| Padding (button) | 6px 10px| 8px 12px     | 10px 16px|
| Font size        | 12px    | 13px         | 14px    |

### Position Variants

| Position  | Background      | Border          | Shadow          |
|-----------|-----------------|-----------------|-----------------|
| inline    | transparent     | none            | none            |
| floating  | solid bg        | subtle          | shadow-sm       |
| header    | transparent     | none            | none            |

### Colors

#### Light Theme

| Element                | State    | Value               |
|------------------------|----------|---------------------|
| Icon color             | idle     | #64748b             |
| Icon color             | hover    | #334155             |
| Background             | idle     | transparent         |
| Background             | hover    | #f1f5f9             |
| Background             | active   | #e2e8f0             |
| Border                 | focus    | #3b82f6             |
| Icon color             | copied   | #059669             |
| Background             | copied   | #ecfdf5             |
| Text color             | copied   | #059669             |
| Icon color             | error    | #dc2626             |
| Background             | error    | #fef2f2             |
| Text color             | error    | #dc2626             |
| Disabled opacity       | disabled | 0.5                 |

#### Dark Theme

| Element                | State    | Value               |
|------------------------|----------|---------------------|
| Icon color             | idle     | #94a3b8             |
| Icon color             | hover    | #e2e8f0             |
| Background             | idle     | transparent         |
| Background             | hover    | #334155             |
| Background             | active   | #475569             |
| Border                 | focus    | #60a5fa             |
| Icon color             | copied   | #34d399             |
| Background             | copied   | #064e3b             |
| Text color             | copied   | #34d399             |
| Icon color             | error    | #f87171             |
| Background             | error    | #450a0a             |
| Text color             | error    | #f87171             |

---

## 5. Icons

### Copy Icon

```svg
<svg 
  width="16" 
  height="16" 
  viewBox="0 0 24 24" 
  fill="none"
  stroke="currentColor" 
  stroke-width="2"
  stroke-linecap="round" 
  stroke-linejoin="round"
  aria-hidden="true"
>
  <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
  <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
</svg>
```

### Check Icon (Success)

```svg
<svg 
  width="16" 
  height="16" 
  viewBox="0 0 24 24" 
  fill="none"
  stroke="currentColor" 
  stroke-width="2"
  stroke-linecap="round" 
  stroke-linejoin="round"
  aria-hidden="true"
>
  <polyline points="20 6 9 17 4 12"></polyline>
</svg>
```

### X Icon (Error)

```svg
<svg 
  width="16" 
  height="16" 
  viewBox="0 0 24 24" 
  fill="none"
  stroke="currentColor" 
  stroke-width="2"
  stroke-linecap="round" 
  stroke-linejoin="round"
  aria-hidden="true"
>
  <line x1="18" y1="6" x2="6" y2="18"></line>
  <line x1="6" y1="6" x2="18" y2="18"></line>
</svg>
```

---

## 6. Interactions

### Click/Touch

| Action         | Behavior                                      |
|----------------|-----------------------------------------------|
| Click          | Attempt to copy, show feedback                |
| Touch          | Same as click                                 |
| Hold           | No special behavior                           |

### Keyboard

| Key            | Behavior                                      |
|----------------|-----------------------------------------------|
| Tab            | Focus the button                              |
| Enter          | Activate copy                                 |
| Space          | Activate copy                                 |

### Feedback Timing

```typescript
const FEEDBACK_DURATION = 2000; // 2 seconds

async function handleCopy(): Promise<void> {
  if (status !== 'idle') return; // Prevent double-click
  
  try {
    await navigator.clipboard.writeText(text);
    setStatus('copied');
    onCopy?.();
    announceToSR(successMessage || 'Copied to clipboard');
  } catch (error) {
    setStatus('error');
    onError?.(error);
    announceToSR(errorMessage || 'Failed to copy');
  }
  
  setTimeout(() => setStatus('idle'), FEEDBACK_DURATION);
}
```

---

## 7. Accessibility

### HTML Structure

```html
<button
  type="button"
  class="copy-button"
  aria-label="Copy code to clipboard"
  aria-live="polite"
  aria-disabled="false"
>
  <svg class="copy-button-icon" aria-hidden="true">
    <!-- Icon based on state -->
  </svg>
  <span class="copy-button-label">Copy</span>
</button>

<!-- Screen reader announcer (visually hidden) -->
<span 
  id="copy-status-announcer" 
  class="sr-only" 
  role="status" 
  aria-live="polite"
>
  <!-- Announcements injected here -->
</span>
```

### ARIA Attributes

| Attribute        | Value                     | Purpose                    |
|------------------|--------------------------|----------------------------|
| type             | "button"                  | Prevent form submission    |
| aria-label       | "Copy code to clipboard"  | Accessible name            |
| aria-live        | "polite"                  | Announce state changes     |
| aria-disabled    | "true"/"false"            | Disabled state             |

### Screen Reader Announcements

| State   | Announcement                                   |
|---------|------------------------------------------------|
| Copied  | "Copied to clipboard" or custom message        |
| Error   | "Failed to copy" or custom message             |

### Focus Behavior

```css
.copy-button:focus-visible {
  outline: 2px solid var(--color-border-focus);
  outline-offset: 2px;
}

/* Remove outline for mouse clicks */
.copy-button:focus:not(:focus-visible) {
  outline: none;
}
```

---

## 8. Responsive Design

### Touch Targets

```css
/* Ensure adequate touch target on mobile */
@media (pointer: coarse) {
  .copy-button {
    min-width: 44px;
    min-height: 44px;
  }
}
```

### Breakpoint Adjustments

| Breakpoint      | Changes                                       |
|-----------------|-----------------------------------------------|
| Mobile (<640px) | Icon only, larger touch target                |
| Tablet+         | Can show label if variant="button"            |

---

## 9. Animation Specifications

### Icon Transition

```css
.copy-button-icon {
  transition: transform 0.15s ease-out, opacity 0.15s ease-out;
}

.copy-button:active .copy-button-icon {
  transform: scale(0.9);
}
```

### Success Animation

```css
.copy-button--copied .copy-button-icon {
  animation: copy-success 0.3s ease-out;
}

@keyframes copy-success {
  0% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.2);
  }
  100% {
    transform: scale(1);
  }
}
```

### Icon Swap

```css
.copy-icon,
.check-icon,
.error-icon {
  position: absolute;
  transition: opacity 0.15s ease-out, transform 0.15s ease-out;
}

/* Idle: show copy icon */
.copy-button--idle .copy-icon {
  opacity: 1;
  transform: scale(1);
}
.copy-button--idle .check-icon,
.copy-button--idle .error-icon {
  opacity: 0;
  transform: scale(0.5);
}

/* Copied: show check icon */
.copy-button--copied .check-icon {
  opacity: 1;
  transform: scale(1);
}
.copy-button--copied .copy-icon {
  opacity: 0;
  transform: scale(0.5);
}

/* Error: show error icon */
.copy-button--error .error-icon {
  opacity: 1;
  transform: scale(1);
}
.copy-button--error .copy-icon {
  opacity: 0;
  transform: scale(0.5);
}
```

### Label Transition

```css
.copy-button-label {
  transition: color 0.15s ease-out;
  white-space: nowrap;
}
```

### Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  .copy-button-icon,
  .copy-icon,
  .check-icon,
  .error-icon,
  .copy-button-label {
    transition: none;
    animation: none;
  }
}
```

---

## 10. Error Handling

### Clipboard API Fallback

```typescript
async function copyToClipboard(text: string): Promise<void> {
  // Modern API
  if (navigator.clipboard?.writeText) {
    return navigator.clipboard.writeText(text);
  }
  
  // Fallback for older browsers
  const textarea = document.createElement('textarea');
  textarea.value = text;
  textarea.style.position = 'fixed';
  textarea.style.opacity = '0';
  document.body.appendChild(textarea);
  textarea.select();
  
  try {
    const success = document.execCommand('copy');
    if (!success) throw new Error('Copy command failed');
  } finally {
    document.body.removeChild(textarea);
  }
}
```

### Common Error Scenarios

| Scenario                      | Behavior                              |
|-------------------------------|---------------------------------------|
| Clipboard API not available   | Use fallback, then show error if fails|
| User denied clipboard access  | Show error state                      |
| Empty text                    | Prevent copy, no feedback             |
| HTTPS required                | Show error with explanation           |

---

## 11. Implementation Notes

### Component Implementation

```typescript
import { useState, useCallback, useRef } from 'react';

type CopyStatus = 'idle' | 'copied' | 'error';

export function CopyButton({
  text,
  size = 'md',
  variant = 'icon',
  label = 'Copy',
  successMessage = 'Copied to clipboard',
  errorMessage = 'Failed to copy',
  feedbackDuration = 2000,
  onCopy,
  onError,
  className,
  disabled = false,
}: CopyButtonProps) {
  const [status, setStatus] = useState<CopyStatus>('idle');
  const timeoutRef = useRef<number>();
  
  const handleCopy = useCallback(async () => {
    if (disabled || status !== 'idle' || !text) return;
    
    // Clear any existing timeout
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }
    
    try {
      await copyToClipboard(text);
      setStatus('copied');
      onCopy?.();
      announceToSR(successMessage);
    } catch (error) {
      setStatus('error');
      onError?.(error as Error);
      announceToSR(errorMessage);
    }
    
    timeoutRef.current = window.setTimeout(() => {
      setStatus('idle');
    }, feedbackDuration);
  }, [text, disabled, status, feedbackDuration, onCopy, onError]);
  
  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);
  
  return (
    <button
      type="button"
      className={cn(
        'copy-button',
        `copy-button--${size}`,
        `copy-button--${variant}`,
        `copy-button--${status}`,
        className
      )}
      onClick={handleCopy}
      disabled={disabled}
      aria-label={label}
      aria-live="polite"
    >
      <span className="copy-button-icons">
        <CopyIcon className="copy-icon" />
        <CheckIcon className="check-icon" />
        <ErrorIcon className="error-icon" />
      </span>
      {variant === 'button' && (
        <span className="copy-button-label">
          {status === 'idle' && label}
          {status === 'copied' && 'Copied!'}
          {status === 'error' && 'Failed'}
        </span>
      )}
    </button>
  );
}
```

### Testing Checklist

- [ ] Click copies text correctly
- [ ] Keyboard activation works
- [ ] Success state shows and resets
- [ ] Error state shows and resets
- [ ] Screen reader announces status
- [ ] Focus visible on keyboard navigation
- [ ] Touch target adequate on mobile
- [ ] Works in both themes
- [ ] Reduced motion respected
- [ ] Disabled state prevents interaction
- [ ] Rapid clicks handled correctly

---

## 12. CSS Custom Properties

```css
/* Copy Button Tokens */
--copy-button-icon-color: #64748b;
--copy-button-icon-hover: #334155;
--copy-button-bg: transparent;
--copy-button-bg-hover: #f1f5f9;
--copy-button-bg-active: #e2e8f0;
--copy-button-success-icon: #059669;
--copy-button-success-bg: #ecfdf5;
--copy-button-success-text: #059669;
--copy-button-error-icon: #dc2626;
--copy-button-error-bg: #fef2f2;
--copy-button-error-text: #dc2626;
--copy-button-focus-ring: #3b82f6;
--copy-button-transition: 0.15s ease-out;
--copy-button-feedback-duration: 2000ms;
```

````