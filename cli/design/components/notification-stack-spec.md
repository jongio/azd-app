# NotificationStack Component Specification

## Overview
Container component that manages multiple NotificationToast components, handling stacking, positioning, and lifecycle.

## Component API

### Props
```typescript
interface NotificationStackProps {
  notifications: Notification[]
  onDismiss: (id: string) => void
  onNotificationClick?: (id: string) => void
  maxVisible?: number           // Default: 3
  position?: 'top-right' | 'top-center' | 'bottom-right' | 'bottom-center'
  className?: string
}

interface Notification {
  id: string
  title: string
  message: string
  severity: 'critical' | 'warning' | 'info'
  timestamp: Date
  dismissed?: boolean
}
```

### State
```typescript
interface NotificationStackState {
  visibleNotifications: Notification[]
  queuedNotifications: Notification[]
  animatingOut: Set<string>     // IDs currently animating out
}
```

## Visual Design

### Layout Structure
```
Top-Right Position:
┌────────────────────────────────┐
│                    [Toast 1]   │
│                    [Toast 2]   │
│                    [Toast 3]   │
│                    (+2 more)   │ ← Overflow indicator
└────────────────────────────────┘
```

### Dimensions
- **Gap Between Toasts**: 12px
- **Container Padding**: 16px (from viewport edges)
- **Z-Index**: 1000 (above most content)

### Stacking Behavior
- **Max Visible**: 3 toasts
- **Newest on top** (bottom of stack visually in top-right)
- **Overflow indicator**: Shows "+N more" when queue exists
- **Auto-shift**: When top toast dismissed, next from queue slides in

## Visual States

### Default Stack
- Toasts spaced 12px apart vertically
- Most recent at bottom (entering position)
- Older toasts above
- Overflow indicator below stack if queue > 0

### Adding Toast (Enter)
1. New toast slides in from right
2. Existing toasts shift up (if at max capacity, oldest removed)
3. Smooth transition: 300ms ease-out

### Removing Toast (Exit)
1. Toast slides out to right
2. Remaining toasts shift down to fill gap
3. Next queued toast slides in from bottom
4. Smooth transition: 300ms ease-in-out

### Overflow Indicator
- **Position**: Below bottom toast
- **Background**: `rgba(0, 0, 0, 0.05)` (light), `rgba(255, 255, 255, 0.05)` (dark)
- **Padding**: 8px 12px
- **Border Radius**: 4px
- **Font Size**: 11px
- **Font Weight**: 500
- **Opacity**: 0.8
- **Text**: "+{count} more notifications"

## Positioning

### Top-Right (Default)
```css
.notification-stack {
  position: fixed;
  top: 16px;
  right: 16px;
  z-index: 1000;
  display: flex;
  flex-direction: column-reverse; /* Newest at bottom */
  gap: 12px;
  max-width: 420px;
}
```

### Top-Center
```css
.notification-stack {
  position: fixed;
  top: 16px;
  left: 50%;
  transform: translateX(-50%);
  /* ... rest same as top-right */
}
```

### Bottom-Right
```css
.notification-stack {
  position: fixed;
  bottom: 16px;
  right: 16px;
  flex-direction: column; /* Newest at top */
  /* ... rest same as top-right */
}
```

## Animations

### Toast Entry
```css
@keyframes slideInStack {
  from {
    opacity: 0;
    transform: translateX(100%) translateY(0);
  }
  to {
    opacity: 1;
    transform: translateX(0) translateY(0);
  }
}
```

### Toast Exit
```css
@keyframes slideOutStack {
  from {
    opacity: 1;
    transform: translateX(0) translateY(0);
  }
  to {
    opacity: 0;
    transform: translateX(100%) translateY(0);
  }
}
```

### Stack Reflow (When Toast Removed)
```css
@keyframes reflow {
  from {
    transform: translateY(var(--from-y));
  }
  to {
    transform: translateY(0);
  }
}
```

## Interaction Behavior

### Adding Notifications
1. New notification arrives via WebSocket
2. If stack < maxVisible: Add to visible stack
3. If stack = maxVisible: Add to queue, update overflow count
4. Animate entry

### Dismissing Notifications
1. User dismisses toast (click X or auto-dismiss)
2. Toast animates out (slideOut)
3. Remaining toasts shift to fill gap (reflow)
4. If queue > 0: Next notification animates in
5. Update overflow count

### Queue Management
- **FIFO**: First in, first out
- **Priority**: Critical notifications bypass queue and show immediately
- **Deduplication**: Same service + severity within 30s = update existing

## Accessibility

### ARIA Live Region
```html
<div
  role="region"
  aria-label="Notifications"
  aria-live="polite"
  aria-atomic="false"
>
```

### Screen Reader Announcements
- Announce count when notifications added: "{count} notifications"
- Announce when queue grows: "{count} more notifications waiting"
- Don't over-announce: Max 1 announcement per 2 seconds

### Focus Management
- No automatic focus stealing
- Toasts are informational, not modal
- Users can tab to dismiss buttons naturally

## Responsive Design

### Desktop (≥1024px)
- Max 3 visible toasts
- Width: 360px per toast
- Position: top-right with 16px margin

### Tablet (768px - 1023px)
- Max 3 visible toasts
- Width: 340px per toast
- Position: top-right with 12px margin

### Mobile (<768px)
- Max 2 visible toasts (screen real estate)
- Width: calc(100vw - 32px)
- Position: top-center with 8px vertical gap
- Overflow indicator more prominent

## State Management

### Adding Notification
```typescript
const addNotification = (notification: Notification) => {
  setNotifications(prev => {
    const newList = [...prev, notification]
    
    // Deduplicate
    const deduped = deduplicateNotifications(newList)
    
    // Sort by priority (critical first) then timestamp
    const sorted = sortNotifications(deduped)
    
    return sorted
  })
}
```

### Removing Notification
```typescript
const removeNotification = (id: string) => {
  setAnimatingOut(prev => new Set([...prev, id]))
  
  setTimeout(() => {
    setNotifications(prev => prev.filter(n => n.id !== id))
    setAnimatingOut(prev => {
      const next = new Set(prev)
      next.delete(id)
      return next
    })
  }, 300) // Match animation duration
}
```

### Queue Logic
```typescript
const visibleNotifications = notifications.slice(0, maxVisible)
const queuedNotifications = notifications.slice(maxVisible)
const queueCount = queuedNotifications.length
```

## Integration Points

### WebSocket Integration
```typescript
useEffect(() => {
  const handleNotification = (data: NotificationData) => {
    addNotification({
      id: generateId(),
      title: data.serviceName,
      message: data.description,
      severity: data.severity,
      timestamp: new Date(data.timestamp)
    })
  }
  
  socket.on('notification', handleNotification)
  return () => socket.off('notification', handleNotification)
}, [])
```

### Persistence
```typescript
// Store notification IDs that have been shown
useEffect(() => {
  const shownIds = notifications.map(n => n.id)
  localStorage.setItem('shownNotifications', JSON.stringify(shownIds))
}, [notifications])
```

## Example Usage

```tsx
<NotificationStack
  notifications={notifications}
  onDismiss={handleDismiss}
  onNotificationClick={handleNavigate}
  maxVisible={3}
  position="top-right"
/>
```

## Edge Cases

### Rapid Notifications
- **Problem**: 10+ notifications arrive in 1 second
- **Solution**: Rate limit to 1 new toast per 500ms, queue rest

### All Critical
- **Problem**: All queued notifications are critical
- **Solution**: Show indicator "3 critical notifications waiting" in red

### Page Navigation
- **Problem**: User navigates away while toasts visible
- **Solution**: Persist to notification center, clear toasts

### Browser Tab Inactive
- **Problem**: Notifications arrive while tab inactive
- **Solution**: Queue all, show summary on tab focus: "5 new notifications"

## Performance

### Optimization
- Use `React.memo` for individual toasts
- Virtualize if queue > 20 notifications
- Debounce reflow animations
- CSS transforms over position changes
- `will-change: transform` on animating elements

### Memory
- Auto-dismiss toasts don't stay in memory
- Queue limited to 50 notifications max
- Older notifications moved to notification center

## Design Tokens

```typescript
const tokens = {
  stack: {
    gap: '12px',
    maxVisible: {
      desktop: 3,
      tablet: 3,
      mobile: 2
    },
    zIndex: 1000,
    padding: {
      desktop: '16px',
      tablet: '12px',
      mobile: '8px'
    },
    overflow: {
      background: 'rgba(0, 0, 0, 0.05)',
      padding: '8px 12px',
      borderRadius: '4px',
      fontSize: '11px'
    }
  }
}
```
