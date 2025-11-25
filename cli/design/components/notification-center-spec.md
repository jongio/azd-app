# NotificationCenter Component Specification

## Overview
Collapsible panel displaying notification history with filtering, search, grouping, and management capabilities.

## Component API

### Props
```typescript
interface NotificationCenterProps {
  notifications: NotificationHistoryItem[]
  onMarkAsRead: (id: string) => void
  onMarkAllAsRead: () => void
  onClearAll: () => void
  onNotificationClick: (id: string) => void
  onClose?: () => void
  isOpen: boolean
  className?: string
}

interface NotificationHistoryItem {
  id: string
  serviceName: string
  message: string
  severity: 'critical' | 'warning' | 'info'
  timestamp: Date
  read: boolean
  acknowledged: boolean
}
```

### State
```typescript
interface NotificationCenterState {
  searchQuery: string
  severityFilter: 'all' | 'critical' | 'warning' | 'info'
  groupBy: 'service' | 'severity' | 'time'
  sortOrder: 'newest' | 'oldest'
  expandedGroups: Set<string>
}
```

## Visual Design

### Layout Structure
```
┌────────────────────────────────────────┐
│ Notifications           [X]            │ ← Header
├────────────────────────────────────────┤
│ [Search] [Filter ▾] [Clear All]       │ ← Toolbar
├────────────────────────────────────────┤
│ ▼ api-service (3)                      │ ← Group Header
│   ● Process crashed - Exit code 1     │
│     2 minutes ago                      │
│   ○ High latency detected             │
│     15 minutes ago                     │
├────────────────────────────────────────┤
│ ▼ database-service (1)                │
│   ● Connection lost                    │
│     1 hour ago                         │
├────────────────────────────────────────┤
│ [Load More]                            │ ← Footer
└────────────────────────────────────────┘
```

### Dimensions
- **Width**: 400px (desktop), 100vw (mobile)
- **Max Height**: 600px (desktop), 100vh (mobile)
- **Position**: Slide-in panel from right edge
- **Padding**: 16px

### Colors (Light Mode)
- **Background**: `hsl(0, 0%, 100%)` (white)
- **Border**: `hsl(0, 0%, 90%)` (light gray)
- **Header BG**: `hsl(0, 0%, 97%)` (off-white)
- **Read Item BG**: `hsl(0, 0%, 98%)` (very light gray)
- **Unread Item BG**: `hsl(0, 0%, 100%)` (white)
- **Hover**: `hsl(0, 0%, 95%)` (light gray)

### Colors (Dark Mode)
- **Background**: `hsl(0, 0%, 12%)` (dark gray)
- **Border**: `hsl(0, 0%, 20%)` (gray)
- **Header BG**: `hsl(0, 0%, 10%)` (darker gray)
- **Read Item BG**: `hsl(0, 0%, 14%)` (slightly lighter)
- **Unread Item BG**: `hsl(0, 0%, 16%)` (lighter)
- **Hover**: `hsl(0, 0%, 18%)` (light gray)

### Typography
- **Header**: 18px, font-weight: 600
- **Group Title**: 14px, font-weight: 600
- **Notification**: 13px, font-weight: 400
- **Timestamp**: 11px, font-weight: 400, opacity: 0.7

## Visual States

### Closed (Hidden)
- Transform: translateX(100%)
- Opacity: 0
- Pointer-events: none

### Open (Visible)
- Transform: translateX(0)
- Opacity: 1
- Shadow: `-4px 0 12px rgba(0, 0, 0, 0.1)`

### Notification Item States

#### Unread
- Background: unread color
- Bold title
- Blue dot indicator (6px circle)
- Border-left: 3px solid severity color

#### Read
- Background: read color
- Normal title
- No dot indicator
- Border-left: 3px solid transparent

#### Hover
- Background: hover color
- Cursor: pointer
- Scale: 1.01

#### Active (Pressed)
- Scale: 0.99
- Background: slightly darker than hover

## Components Breakdown

### Header
```typescript
<div className="header">
  <h2>Notifications</h2>
  <Badge count={unreadCount} />
  <button onClick={onClose} aria-label="Close">
    <X size={20} />
  </button>
</div>
```

### Toolbar
```typescript
<div className="toolbar">
  <SearchInput
    placeholder="Search notifications..."
    value={searchQuery}
    onChange={setSearchQuery}
  />
  <SeverityFilter
    value={severityFilter}
    onChange={setSeverityFilter}
    options={['all', 'critical', 'warning', 'info']}
  />
  <button onClick={onClearAll}>Clear All</button>
  <button onClick={onMarkAllAsRead}>Mark All Read</button>
</div>
```

### Group Header
```typescript
<button className="group-header" onClick={toggleExpanded}>
  <ChevronDown className={expanded ? 'rotate-0' : 'rotate-180'} />
  <span>{groupName}</span>
  <Badge count={groupCount} />
</button>
```

### Notification Item
```typescript
<div className="notification-item" onClick={handleClick}>
  {!read && <div className="unread-indicator" />}
  <div className="severity-badge" data-severity={severity}>
    <Icon />
  </div>
  <div className="content">
    <div className="title">{serviceName}</div>
    <div className="message">{message}</div>
    <div className="timestamp">{relativeTime}</div>
  </div>
  <button onClick={handleMarkRead} aria-label="Mark as read">
    <Check size={16} />
  </button>
</div>
```

## Interactions

### Opening/Closing
- **Open**: Slide in from right (300ms ease-out)
- **Close**: Slide out to right (300ms ease-in)
- **Click outside**: Close panel (optional)
- **Escape key**: Close panel

### Filtering
- **Search**: Real-time filter as user types
- **Severity**: Dropdown filter (all, critical, warning, info)
- **Group by**: Toggle between service/severity/time
- **Clear filters**: Reset to default view

### Notification Actions
- **Click notification**: Navigate to service, mark as read
- **Mark as read**: Click checkmark icon
- **Mark all as read**: Click "Mark All Read" button
- **Clear all**: Click "Clear All" button (confirmation dialog)

### Grouping
- **Expand/Collapse**: Click group header
- **Default**: All groups expanded
- **Persist**: Remember expanded state in localStorage

## Animations

### Panel Slide In
```css
@keyframes slideIn {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}
/* Duration: 300ms, easing: ease-out */
```

### Panel Slide Out
```css
@keyframes slideOut {
  from {
    transform: translateX(0);
    opacity: 1;
  }
  to {
    transform: translateX(100%);
    opacity: 0;
  }
}
/* Duration: 300ms, easing: ease-in */
```

### Group Expand/Collapse
```css
@keyframes expand {
  from {
    max-height: 0;
    opacity: 0;
  }
  to {
    max-height: 1000px;
    opacity: 1;
  }
}
/* Duration: 200ms, easing: ease-out */
```

## Accessibility

### ARIA Attributes
```html
<aside
  role="complementary"
  aria-label="Notification Center"
  aria-hidden={!isOpen}
>
  <div role="search"><!-- Search input --></div>
  <div role="list">
    <div role="group" aria-labelledby="group-{id}">
      <h3 id="group-{id}">{groupName}</h3>
      <div role="listitem" aria-label="Notification">
        <!-- Notification item -->
      </div>
    </div>
  </div>
</aside>
```

### Keyboard Navigation
- **Tab**: Navigate through notifications and controls
- **Enter/Space**: Activate focused notification
- **Arrow Up/Down**: Navigate between notifications
- **Escape**: Close panel
- **Home**: Jump to first notification
- **End**: Jump to last notification

### Screen Reader
- Announce unread count: "{count} unread notifications"
- Announce when marked as read: "Notification marked as read"
- Announce when cleared: "All notifications cleared"

## Responsive Design

### Desktop (≥1024px)
- Width: 400px
- Max height: 600px
- Position: fixed right edge
- Overlay: backdrop-filter blur

### Tablet (768px - 1023px)
- Width: 360px
- Max height: 100vh
- Position: fixed right edge

### Mobile (<768px)
- Width: 100vw
- Height: 100vh
- Position: fixed full screen
- Header sticky at top

## Data Management

### Filtering Logic
```typescript
const filteredNotifications = notifications
  .filter(n => {
    // Search filter
    if (searchQuery && !n.message.toLowerCase().includes(searchQuery.toLowerCase())) {
      return false
    }
    // Severity filter
    if (severityFilter !== 'all' && n.severity !== severityFilter) {
      return false
    }
    return true
  })
  .sort((a, b) => {
    return sortOrder === 'newest'
      ? b.timestamp.getTime() - a.timestamp.getTime()
      : a.timestamp.getTime() - b.timestamp.getTime()
  })
```

### Grouping Logic
```typescript
const grouped = filteredNotifications.reduce((acc, notification) => {
  const key = groupBy === 'service'
    ? notification.serviceName
    : groupBy === 'severity'
    ? notification.severity
    : formatTimeGroup(notification.timestamp)
  
  if (!acc[key]) acc[key] = []
  acc[key].push(notification)
  return acc
}, {} as Record<string, NotificationHistoryItem[]>)
```

### Relative Time
```typescript
const relativeTime = (timestamp: Date): string => {
  const now = new Date()
  const diff = now.getTime() - timestamp.getTime()
  
  if (diff < 60000) return 'Just now'
  if (diff < 3600000) return `${Math.floor(diff / 60000)} minutes ago`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} hours ago`
  return `${Math.floor(diff / 86400000)} days ago`
}
```

## Integration Points

### Notification Store
```typescript
// Load notifications from localStorage
useEffect(() => {
  const stored = localStorage.getItem('notificationHistory')
  if (stored) {
    setNotifications(JSON.parse(stored))
  }
}, [])

// Save on change
useEffect(() => {
  localStorage.setItem('notificationHistory', JSON.stringify(notifications))
}, [notifications])
```

### WebSocket Updates
```typescript
useEffect(() => {
  socket.on('notification', (data) => {
    addNotification(data)
  })
  
  return () => socket.off('notification')
}, [])
```

## Example Usage

```tsx
<NotificationCenter
  notifications={notificationHistory}
  onMarkAsRead={handleMarkAsRead}
  onMarkAllAsRead={handleMarkAllAsRead}
  onClearAll={handleClearAll}
  onNotificationClick={handleNavigate}
  onClose={closePanel}
  isOpen={isPanelOpen}
/>
```

## Design Tokens

```typescript
const tokens = {
  notificationCenter: {
    width: {
      desktop: '400px',
      tablet: '360px',
      mobile: '100vw'
    },
    maxHeight: {
      desktop: '600px',
      mobile: '100vh'
    },
    padding: '16px',
    gap: '12px',
    borderRadius: '0', // Full height panel
    shadow: '-4px 0 12px rgba(0, 0, 0, 0.1)',
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
