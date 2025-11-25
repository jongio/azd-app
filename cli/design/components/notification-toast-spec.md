# NotificationToast Component Specification

## Overview
Individual toast notification component that appears in the top-right corner of the dashboard to display real-time service state changes.

## Component API

### Props
```typescript
interface NotificationToastProps {
  id: string                    // Unique notification ID
  title: string                 // Service name
  message: string               // State description
  severity: 'critical' | 'warning' | 'info'
  timestamp: Date               // When notification occurred
  onDismiss: (id: string) => void
  onClick?: (id: string) => void
  autoDismiss?: boolean         // Default: true
  dismissTimeout?: number       // Default: 5000 (warning/info), 10000 (critical)
  className?: string
}
```

### State
```typescript
interface NotificationToastState {
  isVisible: boolean            // Controls enter/exit animation
  isPaused: boolean             // Pauses auto-dismiss on hover
  timeRemaining: number         // For progress bar
}
```

## Visual Design

### Layout Structure
```
┌─────────────────────────────────────┐
│ [Icon] Title              [X]       │
│        Message                      │
│        "2 minutes ago"              │
│ ──────────────────────── (progress) │
└─────────────────────────────────────┘
```

### Dimensions
- **Width**: 360px (desktop), 100% - 32px (mobile)
- **Min Height**: 80px
- **Max Width**: 420px
- **Padding**: 16px
- **Border Radius**: 8px
- **Gap**: 8px (between elements)

### Colors (Light Mode)
**Critical**:
- Background: `hsl(0, 84%, 95%)` (light red)
- Border: `hsl(0, 84%, 60%)` (red)
- Icon: `hsl(0, 84%, 40%)` (dark red)
- Text: `hsl(0, 10%, 15%)` (near black)

**Warning**:
- Background: `hsl(45, 100%, 95%)` (light yellow)
- Border: `hsl(45, 100%, 50%)` (yellow)
- Icon: `hsl(45, 100%, 35%)` (dark yellow)
- Text: `hsl(45, 10%, 15%)` (near black)

**Info**:
- Background: `hsl(210, 100%, 95%)` (light blue)
- Border: `hsl(210, 100%, 50%)` (blue)
- Icon: `hsl(210, 100%, 35%)` (dark blue)
- Text: `hsl(210, 10%, 15%)` (near black)

### Colors (Dark Mode)
**Critical**:
- Background: `hsl(0, 50%, 15%)` (dark red)
- Border: `hsl(0, 60%, 50%)` (red)
- Icon: `hsl(0, 70%, 60%)` (light red)
- Text: `hsl(0, 5%, 95%)` (near white)

**Warning**:
- Background: `hsl(45, 50%, 15%)` (dark yellow)
- Border: `hsl(45, 80%, 50%)` (yellow)
- Icon: `hsl(45, 90%, 60%)` (light yellow)
- Text: `hsl(45, 5%, 95%)` (near white)

**Info**:
- Background: `hsl(210, 50%, 15%)` (dark blue)
- Border: `hsl(210, 80%, 50%)` (blue)
- Icon: `hsl(210, 90%, 60%)` (light blue)
- Text: `hsl(210, 5%, 95%)` (near white)

### Typography
- **Title**: 14px, font-weight: 600, line-height: 1.4
- **Message**: 13px, font-weight: 400, line-height: 1.5
- **Timestamp**: 11px, font-weight: 400, opacity: 0.7

### Icons
- **Critical**: AlertCircle (lucide-react)
- **Warning**: AlertTriangle (lucide-react)
- **Info**: Info (lucide-react)
- **Close**: X (lucide-react)
- **Size**: 18px

## Visual States

### Default
- Border: 1px solid (severity color)
- Shadow: `0 4px 12px rgba(0, 0, 0, 0.1)`
- Opacity: 1
- Transform: translateX(0)

### Hover
- Shadow: `0 6px 16px rgba(0, 0, 0, 0.15)`
- Close button opacity: 1 (from 0.7)
- Cursor: pointer (entire card clickable)
- Auto-dismiss pauses

### Active (Pressed)
- Transform: scale(0.98)
- Shadow: `0 2px 8px rgba(0, 0, 0, 0.1)`

### Dismissed (Exiting)
- Opacity: 0
- Transform: translateX(100%)
- Transition: 300ms ease-out

### Progress Bar
- Height: 2px
- Position: bottom of toast
- Color: severity border color
- Opacity: 0.6
- Animates from 100% to 0% over dismissTimeout

## Interactions

### Click Behavior
- **Toast body**: Navigate to service details, dismiss toast
- **Close button**: Dismiss toast immediately
- **Hover**: Pause auto-dismiss timer

### Keyboard
- **Tab**: Focus close button
- **Enter/Space**: Activate close button
- **Escape**: Dismiss focused toast

### Auto-Dismiss
- **Warning/Info**: 5 seconds
- **Critical**: 10 seconds
- **Paused on hover**: Timer resumes on mouse leave
- **Progress bar**: Visual indicator of time remaining

## Animations

### Enter (Slide In)
```css
@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateX(100%);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}
/* Duration: 300ms, easing: ease-out */
```

### Exit (Slide Out)
```css
@keyframes slideOut {
  from {
    opacity: 1;
    transform: translateX(0);
  }
  to {
    opacity: 0;
    transform: translateX(100%);
  }
}
/* Duration: 300ms, easing: ease-in */
```

### Progress Bar
```css
@keyframes progress {
  from { width: 100%; }
  to { width: 0%; }
}
/* Duration: dismissTimeout, easing: linear */
```

## Accessibility

### ARIA Attributes
```html
<div
  role="alert"
  aria-live="assertive"
  aria-atomic="true"
  aria-labelledby="toast-title-{id}"
  aria-describedby="toast-message-{id}"
>
```

### Screen Reader Announcements
- **Critical**: "Critical notification: {title}. {message}"
- **Warning**: "Warning: {title}. {message}"
- **Info**: "{title}. {message}"

### Focus Management
- Close button is focusable
- Focus trap not required (toast is informational)
- Focus returns to previous element after dismiss

### Keyboard Navigation
- **Tab**: Move to close button
- **Shift+Tab**: Move to previous focusable element
- **Enter/Space**: Dismiss (when close button focused)
- **Escape**: Dismiss toast

## Responsive Design

### Desktop (≥1024px)
- Width: 360px
- Position: fixed top-right
- Margin: 16px from edges

### Tablet (768px - 1023px)
- Width: 340px
- Position: fixed top-right
- Margin: 12px from edges

### Mobile (<768px)
- Width: calc(100vw - 32px)
- Position: fixed top
- Margin: 16px horizontal, 8px vertical
- Centered horizontally

## Integration Points

### WebSocket Events
```typescript
// Listen for notification events
socket.on('notification', (data: NotificationData) => {
  showToast({
    id: data.id,
    title: data.serviceName,
    message: data.description,
    severity: data.severity,
    timestamp: new Date(data.timestamp)
  })
})
```

### Navigation
```typescript
// Navigate to service on click
const handleClick = (id: string) => {
  const notification = getNotificationById(id)
  navigate(`/service/${notification.serviceName}`)
  onDismiss(id)
}
```

### State Persistence
```typescript
// Store dismissed notifications
const handleDismiss = (id: string) => {
  markNotificationAsRead(id)
  onDismiss(id)
}
```

## Example Usage

```tsx
<NotificationToast
  id="notif-123"
  title="api-service"
  message="Process crashed - Exit code 1"
  severity="critical"
  timestamp={new Date()}
  onDismiss={handleDismiss}
  onClick={handleNavigate}
  autoDismiss={true}
  dismissTimeout={10000}
/>
```

## Design Tokens

```typescript
const tokens = {
  toast: {
    width: {
      desktop: '360px',
      tablet: '340px',
      mobile: 'calc(100vw - 32px)'
    },
    padding: '16px',
    borderRadius: '8px',
    gap: '8px',
    shadow: {
      default: '0 4px 12px rgba(0, 0, 0, 0.1)',
      hover: '0 6px 16px rgba(0, 0, 0, 0.15)',
      active: '0 2px 8px rgba(0, 0, 0, 0.1)'
    },
    animation: {
      duration: '300ms',
      easing: {
        enter: 'ease-out',
        exit: 'ease-in'
      }
    }
  }
}
```
