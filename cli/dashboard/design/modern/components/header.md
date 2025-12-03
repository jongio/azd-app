# Modern Header Specification

> A streamlined, information-rich header that serves as the command center for the dashboard.

## Design Concept

The Modern header is a **floating glass-style bar** with subtle backdrop blur, containing the project identity, primary navigation, live status indicators, and quick actions. It stays fixed at the top while content scrolls beneath it.

---

## Visual Design

### Anatomy

```
┌────────────────────────────────────────────────────────────────────────────┐
│ [Logo/Icon] Project Name     │  Resources │ Console │ Environment │ Metrics │ ··· │ [Status Pill] [Actions] │
└────────────────────────────────────────────────────────────────────────────┘
       Brand Zone                         Navigation Zone                            Utility Zone
```

### Dimensions

| Property | Value |
|----------|-------|
| Height | 64px |
| Horizontal Padding | 24px |
| Logo Size | 28px |
| Nav Item Padding | 12px 16px |
| Gap between zones | 24px |
| Border Radius | 0 (full width) |

---

## Style Tokens

### Background

```css
.modern-header {
  /* Light mode - Glass effect */
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--modern-border-subtle);
  
  /* Alternative: Solid */
  /* background: var(--modern-bg-primary); */
}

[data-theme="modern-dark"] .modern-header {
  background: rgba(12, 18, 34, 0.9);
  border-bottom-color: var(--modern-border-default);
}
```

### Shadow (Optional)

```css
.modern-header--elevated {
  box-shadow: var(--shadow-sm);
}

/* Add shadow on scroll */
.modern-header--scrolled {
  box-shadow: var(--shadow-md);
}
```

---

## Brand Zone

### Logo/Icon

```css
.modern-header-logo {
  width: 28px;
  height: 28px;
  border-radius: var(--radius-md);
  background: linear-gradient(135deg, var(--modern-primary-500), var(--modern-primary-600));
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}
```

### Project Name

```css
.modern-header-title {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--foreground);
  letter-spacing: var(--tracking-tight);
}

/* Optional: Status dot after name */
.modern-header-title::after {
  content: '';
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
  background: var(--modern-success-light);
  margin-left: var(--space-2);
  animation: modern-breathe 2s ease-in-out infinite;
}
```

---

## Navigation Zone

### Nav Container

```css
.modern-header-nav {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1);
  border-radius: var(--radius-lg);
  background: var(--modern-bg-tertiary);
}
```

### Nav Item

```css
.modern-nav-item {
  position: relative;
  padding: var(--space-2) var(--space-4);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--foreground-tertiary);
  border-radius: var(--radius-md);
  transition: all var(--duration-fast) var(--ease-out);
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.modern-nav-item:hover {
  color: var(--foreground-secondary);
  background: var(--modern-bg-secondary);
}

.modern-nav-item--active {
  color: var(--foreground);
  background: var(--modern-bg-primary);
  box-shadow: var(--shadow-sm);
}

/* Icon in nav item */
.modern-nav-item-icon {
  width: 16px;
  height: 16px;
  opacity: 0.7;
}

.modern-nav-item--active .modern-nav-item-icon {
  opacity: 1;
  color: var(--primary);
}
```

### Notification Badge

```css
.modern-nav-badge {
  position: absolute;
  top: 4px;
  right: 4px;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  font-size: 10px;
  font-weight: var(--font-semibold);
  line-height: 18px;
  text-align: center;
  color: white;
  background: var(--modern-error-light);
  border-radius: var(--radius-full);
  animation: modern-scale-fade var(--duration-normal) var(--ease-spring);
}

/* Dot variant (no number) */
.modern-nav-badge--dot {
  width: 8px;
  height: 8px;
  min-width: unset;
  padding: 0;
  top: 6px;
  right: 6px;
}
```

---

## Utility Zone

### Status Pill

A live status indicator showing overall system health.

```css
.modern-status-pill {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-3);
  font-size: var(--text-xs);
  font-weight: var(--font-medium);
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
}

/* Healthy variant */
.modern-status-pill--healthy {
  background: var(--modern-success-bg-light);
  color: var(--modern-success-light);
  border: 1px solid transparent;
}

.modern-status-pill--healthy:hover {
  border-color: var(--modern-success-light);
}

/* Warning variant */
.modern-status-pill--warning {
  background: var(--modern-warning-bg-light);
  color: var(--modern-warning-light);
}

/* Error variant */
.modern-status-pill--error {
  background: var(--modern-error-bg-light);
  color: var(--modern-error-light);
  animation: modern-pulse-glow 2s ease-in-out infinite;
}
```

### Status Pill Content

```jsx
<div className="modern-status-pill modern-status-pill--healthy">
  <span className="modern-status-dot" />
  <span>3 Running</span>
  <ChevronDown className="modern-status-chevron" />
</div>
```

```css
.modern-status-dot {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
  background: currentColor;
}

.modern-status-pill--healthy .modern-status-dot {
  animation: modern-breathe 2s ease-in-out infinite;
}

.modern-status-chevron {
  width: 14px;
  height: 14px;
  opacity: 0.6;
  transition: transform var(--duration-fast);
}

.modern-status-pill:hover .modern-status-chevron {
  opacity: 1;
}

.modern-status-pill[aria-expanded="true"] .modern-status-chevron {
  transform: rotate(180deg);
}
```

### Action Buttons

```css
.modern-header-actions {
  display: flex;
  align-items: center;
  gap: var(--space-1);
}

.modern-header-action {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  color: var(--foreground-tertiary);
  transition: all var(--duration-fast) var(--ease-out);
}

.modern-header-action:hover {
  color: var(--foreground);
  background: var(--modern-bg-tertiary);
}

.modern-header-action:active {
  background: var(--modern-bg-secondary);
  transform: scale(0.95);
}

.modern-header-action-icon {
  width: 18px;
  height: 18px;
}
```

### Theme Toggle

```css
.modern-theme-toggle {
  position: relative;
  width: 44px;
  height: 24px;
  border-radius: var(--radius-full);
  background: var(--modern-bg-tertiary);
  border: 1px solid var(--modern-border-default);
  cursor: pointer;
  transition: background var(--duration-normal);
}

.modern-theme-toggle-track {
  position: absolute;
  inset: 2px;
  border-radius: var(--radius-full);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 4px;
}

.modern-theme-toggle-thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 18px;
  height: 18px;
  border-radius: var(--radius-full);
  background: white;
  box-shadow: var(--shadow-sm);
  transition: transform var(--duration-normal) var(--ease-spring);
}

[data-theme="modern-dark"] .modern-theme-toggle-thumb {
  transform: translateX(20px);
}

.modern-theme-icon {
  width: 12px;
  height: 12px;
  color: var(--foreground-muted);
}
```

---

## Responsive Behavior

### Desktop (> 1024px)

Full header with all elements visible.

### Tablet (768px - 1024px)

```css
@media (max-width: 1024px) {
  .modern-header-nav {
    gap: var(--space-0.5);
  }
  
  .modern-nav-item {
    padding: var(--space-2) var(--space-3);
  }
  
  .modern-nav-item-label {
    display: none;
  }
  
  .modern-nav-item-icon {
    margin: 0;
  }
}
```

### Mobile (< 768px)

```css
@media (max-width: 768px) {
  .modern-header {
    height: 56px;
    padding: 0 var(--space-4);
  }
  
  .modern-header-nav {
    display: none;
  }
  
  .modern-header-mobile-nav-trigger {
    display: flex;
  }
  
  .modern-status-pill {
    padding: var(--space-1) var(--space-2);
  }
  
  .modern-status-pill span:not(.modern-status-dot) {
    display: none;
  }
}
```

---

## States & Interactions

### Scroll Detection

```tsx
const [isScrolled, setIsScrolled] = useState(false);

useEffect(() => {
  const handleScroll = () => {
    setIsScrolled(window.scrollY > 10);
  };
  window.addEventListener('scroll', handleScroll);
  return () => window.removeEventListener('scroll', handleScroll);
}, []);

return (
  <header className={cn(
    'modern-header',
    isScrolled && 'modern-header--scrolled'
  )}>
    {/* ... */}
  </header>
);
```

### Connection Status Indicator

```css
.modern-header-connection {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-2);
  font-size: var(--text-xs);
  color: var(--foreground-muted);
  border-radius: var(--radius-full);
}

.modern-header-connection--connected {
  color: var(--modern-success-light);
}

.modern-header-connection--reconnecting {
  color: var(--modern-warning-light);
  animation: modern-pulse-glow 1.5s ease-in-out infinite;
}

.modern-header-connection--disconnected {
  color: var(--modern-error-light);
}

.modern-header-connection-icon {
  width: 14px;
  height: 14px;
}
```

---

## Accessibility

### Keyboard Navigation

- Tab order: Brand → Nav Items → Status → Actions
- Arrow keys navigate within nav group
- Enter/Space activates nav items
- Escape closes any open menus

### ARIA Attributes

```jsx
<header role="banner" className="modern-header">
  <nav role="navigation" aria-label="Main navigation">
    <ul className="modern-header-nav" role="menubar">
      <li role="none">
        <a 
          role="menuitem"
          aria-current={isActive ? 'page' : undefined}
          tabIndex={0}
        >
          Resources
        </a>
      </li>
      {/* ... */}
    </ul>
  </nav>
  
  <button
    aria-label="System status: 3 services running"
    aria-haspopup="true"
    aria-expanded={isExpanded}
  >
    {/* Status pill */}
  </button>
</header>
```

### Focus States

```css
.modern-nav-item:focus-visible,
.modern-header-action:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}
```

---

## Component Structure

```tsx
interface ModernHeaderProps {
  projectName: string;
  activeView: string;
  onViewChange: (view: string) => void;
  healthSummary: HealthSummary | null;
  connected: boolean;
}

export function ModernHeader({
  projectName,
  activeView,
  onViewChange,
  healthSummary,
  connected
}: ModernHeaderProps) {
  return (
    <header className="modern-header">
      {/* Brand Zone */}
      <div className="modern-header-brand">
        <div className="modern-header-logo">
          <Zap className="w-4 h-4" />
        </div>
        <span className="modern-header-title">{projectName}</span>
      </div>
      
      {/* Navigation Zone */}
      <nav className="modern-header-nav">
        {navItems.map(item => (
          <button
            key={item.id}
            className={cn(
              'modern-nav-item',
              activeView === item.id && 'modern-nav-item--active'
            )}
            onClick={() => onViewChange(item.id)}
          >
            <item.icon className="modern-nav-item-icon" />
            <span className="modern-nav-item-label">{item.label}</span>
            {item.badge && <span className="modern-nav-badge">{item.badge}</span>}
          </button>
        ))}
      </nav>
      
      {/* Utility Zone */}
      <div className="modern-header-utilities">
        <StatusPill summary={healthSummary} />
        <div className="modern-header-actions">
          <ThemeToggle />
          <button className="modern-header-action" aria-label="Help">
            <HelpCircle />
          </button>
          <button className="modern-header-action" aria-label="Settings">
            <Settings />
          </button>
        </div>
      </div>
    </header>
  );
}
```
