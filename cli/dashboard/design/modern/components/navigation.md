# Modern Navigation Specification

> A flexible, pill-style navigation system that can adapt to different screen sizes and contexts.

## Navigation Philosophy

The Modern navigation moves away from the traditional fixed sidebar to a **contextual navigation system** that:

- Integrates into the header on desktop
- Transforms into a bottom bar on mobile
- Can expand into a command palette for power users
- Provides visual feedback through smooth animations

---

## Navigation Patterns

### Pattern 1: Inline Tabs (Default - Desktop)

Tabs embedded in the header with pill-style indicators.

```
[ Resources ] [ Console • ] [ Environment ] [ Metrics ]
              ↑ Active indicator
```

### Pattern 2: Bottom Bar (Mobile)

iOS-style bottom navigation for touch devices.

```
┌─────────────────────────────────────────────────────┐
│                    Content Area                      │
│                                                     │
└─────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────┐
│   📊        📜        ⚙️        📈                  │
│ Resources  Console  Environ  Metrics                │
└─────────────────────────────────────────────────────┘
```

### Pattern 3: Command Palette (Power User)

Keyboard-driven navigation via `Cmd+K` / `Ctrl+K`.

---

## Visual Design

### Nav Container (Desktop)

```css
.modern-nav {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1);
  background: var(--modern-bg-tertiary);
  border-radius: var(--radius-xl);
  border: 1px solid var(--modern-border-subtle);
}
```

### Nav Item Base

```css
.modern-nav-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-4);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--foreground-tertiary);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
  white-space: nowrap;
  user-select: none;
}

.modern-nav-item:hover {
  color: var(--foreground-secondary);
  background: rgba(var(--modern-primary-rgb), 0.05);
}

.modern-nav-item:active {
  transform: scale(0.98);
}
```

### Active State

```css
.modern-nav-item--active {
  color: var(--foreground);
  background: var(--modern-bg-primary);
  box-shadow: var(--shadow-sm);
}

/* Optional: Sliding indicator animation */
.modern-nav-indicator {
  position: absolute;
  bottom: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 24px;
  height: 3px;
  background: var(--primary);
  border-radius: var(--radius-full);
  transition: all var(--duration-normal) var(--ease-spring);
}
```

---

## Nav Item Variants

### Icon + Label

```jsx
<button className="modern-nav-item">
  <Activity className="modern-nav-icon" />
  <span className="modern-nav-label">Resources</span>
</button>
```

```css
.modern-nav-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.modern-nav-label {
  /* Hidden on smaller viewports */
}

@media (max-width: 900px) {
  .modern-nav-label {
    display: none;
  }
  
  .modern-nav-item {
    padding: var(--space-2);
  }
}
```

### With Badge

```jsx
<button className="modern-nav-item">
  <Terminal className="modern-nav-icon" />
  <span className="modern-nav-label">Console</span>
  <span className="modern-nav-badge">3</span>
</button>
```

```css
.modern-nav-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 6px;
  font-size: 11px;
  font-weight: var(--font-semibold);
  color: white;
  background: var(--modern-error-light);
  border-radius: var(--radius-full);
}

/* Dot variant for status-only */
.modern-nav-badge--dot {
  width: 8px;
  height: 8px;
  min-width: 8px;
  padding: 0;
}

/* Status colors */
.modern-nav-badge--success {
  background: var(--modern-success-light);
}

.modern-nav-badge--warning {
  background: var(--modern-warning-light);
}
```

### Keyboard Shortcut Hint

```css
.modern-nav-shortcut {
  margin-left: var(--space-2);
  padding: var(--space-0.5) var(--space-1);
  font-size: 10px;
  font-weight: var(--font-medium);
  color: var(--foreground-muted);
  background: var(--modern-bg-tertiary);
  border-radius: var(--radius-sm);
  opacity: 0;
  transition: opacity var(--duration-fast);
}

.modern-nav-item:hover .modern-nav-shortcut {
  opacity: 1;
}

/* Show on keyboard focus */
.modern-nav-item:focus-visible .modern-nav-shortcut {
  opacity: 1;
}
```

---

## Bottom Navigation (Mobile)

### Container

```css
.modern-bottom-nav {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: var(--z-sticky);
  display: none;
  background: var(--modern-bg-primary);
  border-top: 1px solid var(--modern-border-default);
  padding: var(--space-2) var(--space-4);
  padding-bottom: calc(var(--space-2) + env(safe-area-inset-bottom));
}

@media (max-width: 768px) {
  .modern-bottom-nav {
    display: flex;
    justify-content: space-around;
    align-items: center;
  }
  
  /* Add padding to main content */
  .modern-main {
    padding-bottom: calc(70px + env(safe-area-inset-bottom));
  }
}
```

### Bottom Nav Item

```css
.modern-bottom-nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-3);
  color: var(--foreground-muted);
  border-radius: var(--radius-lg);
  transition: all var(--duration-fast) var(--ease-out);
}

.modern-bottom-nav-item:active {
  transform: scale(0.95);
}

.modern-bottom-nav-item--active {
  color: var(--primary);
}

.modern-bottom-nav-icon {
  width: 22px;
  height: 22px;
}

.modern-bottom-nav-label {
  font-size: 11px;
  font-weight: var(--font-medium);
}

/* Active indicator */
.modern-bottom-nav-item--active::before {
  content: '';
  position: absolute;
  top: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 40px;
  height: 3px;
  background: var(--primary);
  border-radius: 0 0 var(--radius-full) var(--radius-full);
}
```

---

## Command Palette

Accessible via `Cmd+K` / `Ctrl+K`.

### Overlay

```css
.modern-command-overlay {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 15vh;
  background: var(--bg-overlay);
  backdrop-filter: blur(4px);
  animation: modern-fade-in var(--duration-fast);
}
```

### Command Dialog

```css
.modern-command-dialog {
  width: 100%;
  max-width: 540px;
  background: var(--modern-bg-primary);
  border: 1px solid var(--modern-border-default);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-2xl);
  overflow: hidden;
  animation: modern-scale-fade var(--duration-normal) var(--ease-spring);
}
```

### Command Input

```css
.modern-command-input-wrapper {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-4);
  border-bottom: 1px solid var(--modern-border-default);
}

.modern-command-input-icon {
  width: 20px;
  height: 20px;
  color: var(--foreground-muted);
  flex-shrink: 0;
}

.modern-command-input {
  flex: 1;
  font-size: var(--text-base);
  color: var(--foreground);
  background: transparent;
  border: none;
  outline: none;
}

.modern-command-input::placeholder {
  color: var(--foreground-placeholder);
}
```

### Command Results

```css
.modern-command-results {
  max-height: 320px;
  overflow-y: auto;
  padding: var(--space-2);
}

.modern-command-group {
  margin-bottom: var(--space-2);
}

.modern-command-group-label {
  padding: var(--space-2) var(--space-3);
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  color: var(--foreground-muted);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wider);
}

.modern-command-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: background var(--duration-fast);
}

.modern-command-item:hover,
.modern-command-item--selected {
  background: var(--modern-bg-tertiary);
}

.modern-command-item-icon {
  width: 18px;
  height: 18px;
  color: var(--foreground-tertiary);
}

.modern-command-item-label {
  flex: 1;
  font-size: var(--text-sm);
  color: var(--foreground);
}

.modern-command-item-shortcut {
  font-size: var(--text-xs);
  color: var(--foreground-muted);
}
```

---

## Keyboard Navigation

### Shortcut Reference

| Key | Action |
|-----|--------|
| `1-4` | Jump to view (Resources, Console, etc.) |
| `Cmd+K` | Open command palette |
| `←` / `→` | Navigate between tabs |
| `Enter` / `Space` | Activate tab |
| `Escape` | Close command palette |
| `?` | Show keyboard shortcuts |

### Implementation

```tsx
useEffect(() => {
  const handleKeyDown = (e: KeyboardEvent) => {
    // Don't handle if in input
    if (shouldIgnoreShortcut(e)) return;
    
    // Number keys for direct navigation
    const viewMap: Record<string, string> = {
      '1': 'resources',
      '2': 'console', 
      '3': 'environment',
      '4': 'metrics'
    };
    
    if (viewMap[e.key]) {
      e.preventDefault();
      onViewChange(viewMap[e.key]);
    }
    
    // Command palette
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
      e.preventDefault();
      setCommandPaletteOpen(true);
    }
  };
  
  window.addEventListener('keydown', handleKeyDown);
  return () => window.removeEventListener('keydown', handleKeyDown);
}, [onViewChange]);
```

---

## Transitions & Animations

### Tab Switch Animation

```css
/* Animate active indicator */
.modern-nav-item::after {
  content: '';
  position: absolute;
  bottom: -1px;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--primary);
  transform: scaleX(0);
  transition: transform var(--duration-normal) var(--ease-spring);
}

.modern-nav-item--active::after {
  transform: scaleX(1);
}
```

### Sliding Pill Indicator (Advanced)

```tsx
// Track active item position for sliding indicator
const [indicatorStyle, setIndicatorStyle] = useState({
  left: 0,
  width: 0
});

useLayoutEffect(() => {
  const activeEl = navRef.current?.querySelector('[data-active="true"]');
  if (activeEl) {
    setIndicatorStyle({
      left: activeEl.offsetLeft,
      width: activeEl.offsetWidth
    });
  }
}, [activeView]);

<div className="modern-nav" ref={navRef}>
  <div 
    className="modern-nav-sliding-indicator"
    style={indicatorStyle}
  />
  {items.map(item => (
    <button data-active={item.id === activeView}>
      {item.label}
    </button>
  ))}
</div>
```

```css
.modern-nav-sliding-indicator {
  position: absolute;
  top: 4px;
  bottom: 4px;
  background: var(--modern-bg-primary);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  transition: all var(--duration-normal) var(--ease-spring);
  z-index: 0;
}

.modern-nav-item {
  position: relative;
  z-index: 1;
}
```

---

## Accessibility

### ARIA Structure

```jsx
<nav role="navigation" aria-label="Main navigation">
  <div 
    className="modern-nav"
    role="tablist"
    aria-orientation="horizontal"
  >
    {items.map((item, index) => (
      <button
        key={item.id}
        role="tab"
        aria-selected={activeView === item.id}
        aria-controls={`${item.id}-panel`}
        tabIndex={activeView === item.id ? 0 : -1}
        className={cn(
          'modern-nav-item',
          activeView === item.id && 'modern-nav-item--active'
        )}
      >
        <item.icon aria-hidden="true" />
        <span>{item.label}</span>
        {item.badge && (
          <span className="modern-nav-badge" aria-label={`${item.badge} items`}>
            {item.badge}
          </span>
        )}
      </button>
    ))}
  </div>
</nav>

{/* Content panels */}
<div
  role="tabpanel"
  id={`${activeView}-panel`}
  aria-labelledby={`${activeView}-tab`}
>
  {/* View content */}
</div>
```

### Focus Management

- Use `tabindex="-1"` on inactive tabs
- Focus follows active tab
- Arrow keys navigate between tabs
- Home/End jump to first/last tab

```tsx
const handleKeyNavigation = (e: React.KeyboardEvent) => {
  const tabs = Array.from(navRef.current?.querySelectorAll('[role="tab"]') || []);
  const currentIndex = tabs.findIndex(tab => tab === document.activeElement);
  
  let nextIndex = currentIndex;
  
  switch (e.key) {
    case 'ArrowLeft':
      nextIndex = currentIndex > 0 ? currentIndex - 1 : tabs.length - 1;
      break;
    case 'ArrowRight':
      nextIndex = currentIndex < tabs.length - 1 ? currentIndex + 1 : 0;
      break;
    case 'Home':
      nextIndex = 0;
      break;
    case 'End':
      nextIndex = tabs.length - 1;
      break;
    default:
      return;
  }
  
  e.preventDefault();
  (tabs[nextIndex] as HTMLElement)?.focus();
};
```

---

## Component Structure

```tsx
interface ModernNavProps {
  items: NavItem[];
  activeView: string;
  onViewChange: (view: string) => void;
}

interface NavItem {
  id: string;
  label: string;
  icon: LucideIcon;
  badge?: number | string;
  badgeVariant?: 'default' | 'success' | 'warning' | 'error';
}

export function ModernNav({ items, activeView, onViewChange }: ModernNavProps) {
  const navRef = useRef<HTMLDivElement>(null);
  
  return (
    <nav aria-label="Main navigation">
      <div 
        ref={navRef}
        className="modern-nav" 
        role="tablist"
        onKeyDown={handleKeyNavigation}
      >
        {items.map(item => (
          <ModernNavItem
            key={item.id}
            item={item}
            isActive={activeView === item.id}
            onClick={() => onViewChange(item.id)}
          />
        ))}
      </div>
    </nav>
  );
}
```
