# NotificationBadge Component Specification

## Overview
Small badge component displaying notification count, used in header and on service cards to indicate unacknowledged notifications.

## Component API

### Props
```typescript
interface NotificationBadgeProps {
  count: number                 // Number to display
  variant?: 'default' | 'critical' | 'warning'
  size?: 'sm' | 'md' | 'lg'
  max?: number                  // Max count to display (default: 99)
  showZero?: boolean            // Show badge when count is 0 (default: false)
  pulse?: boolean               // Animate on new notification (default: false)
  className?: string
}
```

## Visual Design

### Layout Structure
```
Small:  [3]
Medium: [12]
Large:  [99+]
```

### Dimensions

#### Small (sm)
- **Height**: 16px
- **Min Width**: 16px
- **Padding**: 0 4px
- **Font Size**: 10px
- **Border Radius**: 8px

#### Medium (md) - Default
- **Height**: 20px
- **Min Width**: 20px
- **Padding**: 0 6px
- **Font Size**: 11px
- **Border Radius**: 10px

#### Large (lg)
- **Height**: 24px
- **Min Width**: 24px
- **Padding**: 0 8px
- **Font Size**: 12px
- **Border Radius**: 12px

### Colors

#### Default Variant (Light Mode)
- **Background**: `hsl(210, 100%, 50%)` (blue)
- **Text**: `hsl(0, 0%, 100%)` (white)
- **Border**: none

#### Default Variant (Dark Mode)
- **Background**: `hsl(210, 90%, 55%)` (lighter blue)
- **Text**: `hsl(0, 0%, 100%)` (white)
- **Border**: none

#### Critical Variant (Light Mode)
- **Background**: `hsl(0, 84%, 60%)` (red)
- **Text**: `hsl(0, 0%, 100%)` (white)
- **Border**: none

#### Critical Variant (Dark Mode)
- **Background**: `hsl(0, 70%, 60%)` (lighter red)
- **Text**: `hsl(0, 0%, 100%)` (white)
- **Border**: none

#### Warning Variant (Light Mode)
- **Background**: `hsl(45, 100%, 50%)` (yellow)
- **Text**: `hsl(45, 10%, 15%)` (dark text for contrast)
- **Border**: none

#### Warning Variant (Dark Mode)
- **Background**: `hsl(45, 90%, 55%)` (lighter yellow)
- **Text**: `hsl(45, 5%, 10%)` (very dark text)
- **Border**: none

### Typography
- **Font Weight**: 600 (semi-bold)
- **Line Height**: 1
- **Letter Spacing**: -0.01em
- **Font Family**: Inherit from system

## Visual States

### Default
- Scale: 1
- Opacity: 1

### Pulse (New Notification)
- Animation: pulse effect
- Duration: 600ms
- Iterations: 2

### Hidden (count === 0 && !showZero)
- Display: none

### Overflow (count > max)
- Display: "{max}+"
- Example: "99+" for count = 150

## Positioning

### Header Badge (Absolute)
```css
.notification-badge {
  position: absolute;
  top: -8px;
  right: -8px;
  z-index: 1;
}
```

### Inline Badge (Relative)
```css
.notification-badge {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
```

### Service Card Badge
```css
.service-card .notification-badge {
  position: absolute;
  top: 8px;
  right: 8px;
}
```

## Animations

### Pulse Animation
```css
@keyframes pulse {
  0% {
    transform: scale(1);
    box-shadow: 0 0 0 0 rgba(var(--badge-color), 0.7);
  }
  50% {
    transform: scale(1.15);
    box-shadow: 0 0 0 4px rgba(var(--badge-color), 0);
  }
  100% {
    transform: scale(1);
    box-shadow: 0 0 0 0 rgba(var(--badge-color), 0);
  }
}
/* Duration: 600ms, iterations: 2 */
```

### Count Update (Number Change)
```css
@keyframes countUpdate {
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
/* Duration: 200ms, easing: ease-out */
```

## Accessibility

### ARIA Attributes
```html
<span
  className="notification-badge"
  role="status"
  aria-label="{count} unread notifications"
  aria-live="polite"
  aria-atomic="true"
>
  {displayCount}
</span>
```

### Screen Reader
- Announce count changes: "3 unread notifications"
- Don't over-announce: Debounce by 2 seconds
- Critical variant: Add "critical" to label

### Visual Only
- Badge is decorative when paired with accessible button
- Parent element should have proper label

## Responsive Design

### Desktop (≥1024px)
- Size: md
- Always visible

### Tablet (768px - 1023px)
- Size: md
- Always visible

### Mobile (<768px)
- Size: sm (to save space)
- Consider hiding if count is 0

## Variants

### Default (Info)
- Blue background
- General notifications
- Used in header

### Critical
- Red background
- Critical severity only
- Used on service cards
- Pulse on new critical

### Warning
- Yellow background
- Warning severity
- Used on service cards
- Subtle pulse

## Count Display Logic

```typescript
const getDisplayCount = (count: number, max: number = 99): string => {
  if (count === 0) return '0'
  if (count > max) return `${max}+`
  return count.toString()
}
```

## Integration Points

### Header Usage
```tsx
<button onClick={openNotificationCenter} aria-label="Notifications">
  <Bell size={20} />
  <NotificationBadge
    count={unreadCount}
    variant="default"
    size="sm"
    pulse={hasNewNotifications}
  />
</button>
```

### Service Card Usage
```tsx
<div className="service-card">
  <NotificationBadge
    count={criticalCount}
    variant="critical"
    size="md"
    showZero={false}
  />
  {/* Service card content */}
</div>
```

### Inline Usage
```tsx
<h2>
  Notifications
  <NotificationBadge
    count={totalCount}
    variant="default"
    size="lg"
  />
</h2>
```

## State Management

### Count Updates
```typescript
// Auto-pulse on count increase
useEffect(() => {
  if (count > prevCount) {
    setPulse(true)
    setTimeout(() => setPulse(false), 1200) // 2 iterations × 600ms
  }
}, [count])
```

### Debounced Announcements
```typescript
const announceCount = useDebouncedCallback((count: number) => {
  // Trigger screen reader announcement
  setAriaLive(`${count} unread notifications`)
}, 2000)
```

## Example Usage

### Basic
```tsx
<NotificationBadge count={5} />
```

### Critical with Pulse
```tsx
<NotificationBadge
  count={3}
  variant="critical"
  pulse={true}
  size="md"
/>
```

### Header Bell Icon
```tsx
<button className="notification-trigger">
  <Bell size={20} />
  {unreadCount > 0 && (
    <NotificationBadge
      count={unreadCount}
      size="sm"
      variant="default"
      pulse={hasNewCritical}
    />
  )}
</button>
```

### Service Card
```tsx
<div className="service-card-header">
  <h3>{serviceName}</h3>
  {criticalNotifications > 0 && (
    <NotificationBadge
      count={criticalNotifications}
      variant="critical"
      size="md"
    />
  )}
</div>
```

## Edge Cases

### Very Large Numbers
- **Input**: count = 9999
- **Display**: "99+"
- **Aria**: "9999 unread notifications"

### Zero Count
- **Default**: Hidden (display: none)
- **With showZero**: Show "0" badge

### Negative Count
- **Behavior**: Clamp to 0
- **Display**: Hidden or "0" based on showZero

### Rapid Updates
- **Problem**: Count changes rapidly (1→2→3→4 in 1 second)
- **Solution**: Debounce pulse, only trigger once per 600ms

## Performance

### Optimization
- Use CSS transforms for animations (GPU accelerated)
- Memoize display count calculation
- Avoid re-renders on parent state changes
- Use `React.memo` for badge component

## Design Tokens

```typescript
const tokens = {
  badge: {
    sizes: {
      sm: {
        height: '16px',
        minWidth: '16px',
        padding: '0 4px',
        fontSize: '10px',
        borderRadius: '8px'
      },
      md: {
        height: '20px',
        minWidth: '20px',
        padding: '0 6px',
        fontSize: '11px',
        borderRadius: '10px'
      },
      lg: {
        height: '24px',
        minWidth: '24px',
        padding: '0 8px',
        fontSize: '12px',
        borderRadius: '12px'
      }
    },
    variants: {
      default: {
        light: {
          bg: 'hsl(210, 100%, 50%)',
          text: 'hsl(0, 0%, 100%)'
        },
        dark: {
          bg: 'hsl(210, 90%, 55%)',
          text: 'hsl(0, 0%, 100%)'
        }
      },
      critical: {
        light: {
          bg: 'hsl(0, 84%, 60%)',
          text: 'hsl(0, 0%, 100%)'
        },
        dark: {
          bg: 'hsl(0, 70%, 60%)',
          text: 'hsl(0, 0%, 100%)'
        }
      },
      warning: {
        light: {
          bg: 'hsl(45, 100%, 50%)',
          text: 'hsl(45, 10%, 15%)'
        },
        dark: {
          bg: 'hsl(45, 90%, 55%)',
          text: 'hsl(45, 5%, 10%)'
        }
      }
    },
    animation: {
      pulse: {
        duration: '600ms',
        iterations: 2
      },
      countUpdate: {
        duration: '200ms',
        easing: 'ease-out'
      }
    }
  }
}
```
