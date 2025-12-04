# Mobile Menu Component Specification

## Overview

The Mobile Menu component provides full-screen navigation for mobile and tablet devices. It slides in from the right side, provides access to all navigation items, and includes the theme toggle. This component is used when the viewport width is below the desktop breakpoint (1024px).

---

## 1. Component Hierarchy

```
MobileMenu (organism)
├── MobileMenuBackdrop (atom)
├── MobileMenuContainer (molecule)
│   ├── MobileMenuHeader (molecule)
│   │   ├── Logo (atom)
│   │   └── CloseButton (atom)
│   ├── MobileNav (molecule)
│   │   ├── MobileNavItem (atom) × 5
│   │   └── MCPBadge (atom) [on MCP Server item]
│   ├── MobileMenuDivider (atom)
│   ├── ThemeToggle (atom)
│   └── MobileMenuFooter (molecule) [optional]
│       └── SocialLinks (atom)
└── FocusTrap (utility)
```

---

## 2. Props Interface

### MobileMenu

```typescript
interface MobileMenuProps {
  /** Whether the menu is open */
  isOpen: boolean;
  /** Close handler */
  onClose: () => void;
  /** Navigation items */
  navigation: NavigationItem[];
  /** Current active path */
  currentPath: string;
  /** Theme configuration */
  theme: Theme;
  /** Theme change handler */
  onThemeChange: (theme: Theme) => void;
  /** Optional social links */
  socialLinks?: SocialLink[];
  /** Custom class name */
  className?: string;
}

interface NavigationItem {
  /** Display label */
  label: string;
  /** Link destination */
  href: string;
  /** Optional icon */
  icon?: string;
  /** Whether this is prominent (MCP Server) */
  isProminent?: boolean;
  /** Badge (e.g., "AI") */
  badge?: string;
}
```

### MobileMenuBackdrop

```typescript
interface MobileMenuBackdropProps {
  /** Whether backdrop is visible */
  isVisible: boolean;
  /** Click handler (closes menu) */
  onClick: () => void;
}
```

### MobileNavItem

```typescript
interface MobileNavItemProps {
  /** Display label */
  label: string;
  /** Link destination */
  href: string;
  /** Whether item is active */
  isActive: boolean;
  /** Whether item is prominent */
  isProminent?: boolean;
  /** Optional icon */
  icon?: string;
  /** Badge text */
  badge?: string;
  /** Click handler */
  onClick: () => void;
}
```

### CloseButton

```typescript
interface CloseButtonProps {
  /** Click handler */
  onClick: () => void;
  /** Accessible label */
  ariaLabel?: string;
}
```

---

## 3. States

### Menu States

| State      | Trigger            | Visual Changes                        |
|------------|-------------------|---------------------------------------|
| Closed     | Default/onClose    | Menu hidden, backdrop invisible       |
| Opening    | isOpen → true      | Slide animation, backdrop fades in    |
| Open       | Animation complete | Fully visible, focus trapped          |
| Closing    | onClose called     | Slide out, backdrop fades out         |

### NavItem States

| State    | Trigger            | Visual Changes                        |
|----------|-------------------|---------------------------------------|
| Default  | Initial render     | Normal styling                        |
| Hover    | Touch highlight    | Background highlight (touch feedback) |
| Active   | Current page       | Primary color, bold, left border      |
| Focus    | Keyboard focus     | Focus ring visible                    |
| Pressed  | Touch active       | Scale down slightly                   |

### MCP Server NavItem States

| State    | Visual Changes                                           |
|----------|----------------------------------------------------------|
| Default  | Gradient border, badge visible, star icon                |
| Hover    | Enhanced glow effect                                     |
| Active   | Filled background, enhanced prominence                   |
| Focus    | Strong focus ring with brand color                       |

---

## 4. Visual Specifications

### Layout

```
┌─────────────────────────────────┐
│                                 │
│  [Logo]              [X]        │  ← Header (56px)
│                                 │
├─────────────────────────────────┤
│                                 │
│    🏠  Home                     │  ← NavItem (56px each)
│                                 │
│    🚀  Quick Start              │
│                                 │
│    ★   MCP Server  [AI]         │  ← Prominent item
│                                 │
│    📖  Guided Tour              │
│                                 │
│    📚  Reference                │
│                                 │
├─────────────────────────────────┤
│                                 │
│    🌙  Dark mode                │  ← Theme toggle
│                                 │
├─────────────────────────────────┤
│                                 │
│  [GitHub] [Discord] [Twitter]   │  ← Social (optional)
│                                 │
└─────────────────────────────────┘
```

### Dimensions

| Property              | Value               |
|-----------------------|---------------------|
| Menu width            | 100% (max 320px)    |
| Menu min-width        | 280px               |
| Header height         | 56px                |
| Nav item height       | 56px                |
| Nav item padding      | 0 24px              |
| Icon size             | 24px                |
| Icon-text gap         | 16px                |
| Badge height          | 20px                |
| Badge padding         | 4px 8px             |
| Close button size     | 44px                |
| Divider height        | 1px                 |
| Divider margin        | 8px 24px            |

### Typography

| Element          | Font Size | Weight    | Line Height |
|------------------|-----------|-----------|-------------|
| Logo text        | 20px      | Bold      | 1.25        |
| Nav item         | 18px      | Medium    | 1.5         |
| Nav item (MCP)   | 18px      | Semibold  | 1.5         |
| Badge            | 11px      | Bold      | 1           |
| Theme label      | 16px      | Normal    | 1.5         |

### Colors

#### Light Theme

| Element                | Property    | Value               |
|------------------------|-------------|---------------------|
| Backdrop               | background  | rgba(0, 0, 0, 0.5)  |
| Menu bg                | background  | white               |
| Header border          | border-bottom | 1px solid #e5e7eb |
| Nav item text          | color       | #374151             |
| Nav item active text   | color       | #2563eb             |
| Nav item active bg     | background  | #eff6ff             |
| Nav item active border | border-left | 4px solid #2563eb   |
| Nav item hover bg      | background  | #f3f4f6             |
| MCP item border        | border      | 1px solid #f59e0b   |
| MCP item bg            | background  | #fffbeb             |
| MCP badge bg           | background  | #fef3c7             |
| MCP badge text         | color       | #92400e             |
| Divider                | background  | #e5e7eb             |
| Close icon             | color       | #6b7280             |

#### Dark Theme

| Element                | Property    | Value               |
|------------------------|-------------|---------------------|
| Backdrop               | background  | rgba(0, 0, 0, 0.7)  |
| Menu bg                | background  | #1e293b             |
| Header border          | border-bottom | 1px solid #334155 |
| Nav item text          | color       | #e2e8f0             |
| Nav item active text   | color       | #60a5fa             |
| Nav item active bg     | background  | #1e3a5f             |
| Nav item active border | border-left | 4px solid #60a5fa   |
| Nav item hover bg      | background  | #334155             |
| MCP item border        | border      | 1px solid #f59e0b   |
| MCP item bg            | background  | #422006             |
| MCP badge bg           | background  | #451a03             |
| MCP badge text         | color       | #fcd34d             |
| Divider                | background  | #334155             |
| Close icon             | color       | #94a3b8             |

---

## 5. Interactions

### Gestures

| Gesture             | Behavior                                   |
|---------------------|-------------------------------------------|
| Tap backdrop        | Close menu                                 |
| Tap close button    | Close menu                                 |
| Tap nav item        | Navigate and close menu                    |
| Swipe left on menu  | Close menu                                 |
| Swipe right on edge | Open menu (from edge gesture)              |

### Touch Feedback

```css
.mobile-nav-item {
  -webkit-tap-highlight-color: transparent;
  transition: background-color 100ms ease-out;
}

.mobile-nav-item:active {
  background-color: var(--color-bg-tertiary);
}
```

### Keyboard

| Key         | Behavior                                        |
|-------------|------------------------------------------------|
| Tab         | Move focus to next item                         |
| Shift+Tab   | Move focus to previous item                     |
| Enter       | Activate focused item                           |
| Escape      | Close menu, return focus to hamburger           |
| Arrow Down  | Move focus to next nav item                     |
| Arrow Up    | Move focus to previous nav item                 |
| Home        | Move focus to first nav item                    |
| End         | Move focus to last nav item                     |

---

## 6. Accessibility

### ARIA Attributes

```html
<!-- Hamburger trigger (in header) -->
<button
  type="button"
  class="hamburger"
  aria-label="Open main menu"
  aria-expanded="false"
  aria-controls="mobile-menu"
>
  <span class="hamburger-line"></span>
  <span class="hamburger-line"></span>
  <span class="hamburger-line"></span>
</button>

<!-- Backdrop -->
<div 
  class="mobile-menu-backdrop"
  aria-hidden="true"
  tabindex="-1"
></div>

<!-- Menu -->
<div
  id="mobile-menu"
  class="mobile-menu"
  role="dialog"
  aria-modal="true"
  aria-label="Main menu"
  hidden
>
  <div class="mobile-menu-header">
    <a href="/" aria-label="azd-app home">
      <span>azd-app</span>
    </a>
    <button
      type="button"
      class="close-button"
      aria-label="Close menu"
    >
      <svg aria-hidden="true"><!-- X icon --></svg>
    </button>
  </div>
  
  <nav aria-label="Main navigation">
    <ul role="list">
      <li>
        <a href="/" aria-current="page">
          <svg aria-hidden="true"><!-- home icon --></svg>
          Home
        </a>
      </li>
      <li>
        <a href="/mcp-server" aria-describedby="mcp-description">
          <svg aria-hidden="true"><!-- star icon --></svg>
          MCP Server
          <span class="badge" aria-label="AI feature">AI</span>
        </a>
        <span id="mcp-description" class="sr-only">
          AI-powered development assistant
        </span>
      </li>
      <!-- Additional nav items -->
    </ul>
  </nav>
  
  <div class="mobile-menu-divider" role="separator"></div>
  
  <div class="mobile-menu-theme">
    <!-- Theme toggle component -->
  </div>
</div>
```

### Focus Management

```typescript
// When menu opens
function onMenuOpen(): void {
  // Save trigger for focus return
  previousFocus = document.activeElement;
  
  // Move focus to first focusable element (close button)
  const closeButton = menu.querySelector('.close-button');
  closeButton?.focus();
  
  // Trap focus within menu
  activateFocusTrap();
  
  // Prevent body scroll
  document.body.style.overflow = 'hidden';
}

// When menu closes
function onMenuClose(): void {
  // Deactivate focus trap
  deactivateFocusTrap();
  
  // Return focus to trigger
  previousFocus?.focus();
  
  // Restore body scroll
  document.body.style.overflow = '';
}
```

### Screen Reader Announcements

- Menu open: "Main menu dialog opened"
- Menu close: "Main menu closed"
- Navigation: "Current page: Home"

---

## 7. Animation Specifications

### Menu Slide In

```css
.mobile-menu {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  width: min(100vw, 320px);
  transform: translateX(100%);
  transition: transform 250ms cubic-bezier(0.4, 0, 0.2, 1);
  z-index: 50;
}

.mobile-menu[data-open="true"] {
  transform: translateX(0);
}
```

### Backdrop Fade

```css
.mobile-menu-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  opacity: 0;
  visibility: hidden;
  transition: opacity 200ms ease-out, visibility 0s 200ms;
  z-index: 40;
}

.mobile-menu-backdrop[data-visible="true"] {
  opacity: 1;
  visibility: visible;
  transition: opacity 200ms ease-out;
}

/* Backdrop blur (when supported) */
@supports (backdrop-filter: blur(4px)) {
  .mobile-menu-backdrop {
    backdrop-filter: blur(4px);
  }
}
```

### Close Button X Animation

```css
.close-button svg {
  transition: transform 150ms ease-out;
}

.close-button:hover svg {
  transform: rotate(90deg);
}

.close-button:active svg {
  transform: rotate(90deg) scale(0.9);
}
```

### Nav Item Stagger (Optional)

```css
.mobile-nav-item {
  opacity: 0;
  transform: translateX(20px);
}

.mobile-menu[data-open="true"] .mobile-nav-item {
  animation: slideIn 200ms ease-out forwards;
}

.mobile-nav-item:nth-child(1) { animation-delay: 50ms; }
.mobile-nav-item:nth-child(2) { animation-delay: 100ms; }
.mobile-nav-item:nth-child(3) { animation-delay: 150ms; }
.mobile-nav-item:nth-child(4) { animation-delay: 200ms; }
.mobile-nav-item:nth-child(5) { animation-delay: 250ms; }

@keyframes slideIn {
  to {
    opacity: 1;
    transform: translateX(0);
  }
}
```

### Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  .mobile-menu {
    transition: opacity 100ms ease-out;
    transform: none !important;
  }
  
  .mobile-menu:not([data-open="true"]) {
    opacity: 0;
    visibility: hidden;
  }
  
  .mobile-menu[data-open="true"] {
    opacity: 1;
    visibility: visible;
  }
  
  .mobile-nav-item {
    animation: none !important;
    opacity: 1;
    transform: none;
  }
}
```

---

## 8. Swipe Gesture Implementation

### Swipe to Close

```typescript
let startX: number;
let currentX: number;
let isDragging = false;

function handleTouchStart(e: TouchEvent): void {
  startX = e.touches[0].clientX;
  isDragging = true;
}

function handleTouchMove(e: TouchEvent): void {
  if (!isDragging) return;
  
  currentX = e.touches[0].clientX;
  const diff = startX - currentX;
  
  // Only allow swiping left (to close)
  if (diff > 0) {
    menu.style.transform = `translateX(${Math.min(diff, menuWidth)}px)`;
  }
}

function handleTouchEnd(): void {
  if (!isDragging) return;
  isDragging = false;
  
  const diff = startX - currentX;
  const threshold = menuWidth * 0.3; // 30% of menu width
  
  if (diff > threshold) {
    closeMenu();
  } else {
    // Snap back
    menu.style.transform = '';
  }
}
```

### Edge Swipe to Open

```typescript
// Detect swipe from right edge
function handleEdgeSwipe(e: TouchEvent): void {
  const touch = e.touches[0];
  const edgeThreshold = 20; // px from right edge
  
  if (window.innerWidth - touch.clientX < edgeThreshold) {
    // Start opening animation based on swipe
    openMenu();
  }
}
```

---

## 9. Body Scroll Lock

```typescript
// Prevent background scrolling when menu is open
function lockBodyScroll(): void {
  const scrollY = window.scrollY;
  document.body.style.position = 'fixed';
  document.body.style.top = `-${scrollY}px`;
  document.body.style.width = '100%';
  document.body.dataset.scrollLock = String(scrollY);
}

function unlockBodyScroll(): void {
  const scrollY = document.body.dataset.scrollLock || '0';
  document.body.style.position = '';
  document.body.style.top = '';
  document.body.style.width = '';
  delete document.body.dataset.scrollLock;
  window.scrollTo(0, parseInt(scrollY));
}
```

---

## 10. Error States

| Scenario              | Behavior                                     |
|-----------------------|---------------------------------------------|
| JavaScript disabled   | Menu hidden, nav should be visible elsewhere|
| Touch not supported   | Keyboard navigation works                    |
| Animation fails       | Menu appears/disappears immediately          |

---

## 11. Implementation Notes

### Performance

- Use `will-change: transform` on menu
- Use CSS transitions over JavaScript animations
- Lazy load menu content until first open
- Use passive touch listeners

### SSR Considerations

- Render menu with `hidden` attribute
- Initialize with `aria-hidden="true"`
- Hydrate on client

### Z-Index Stack

```css
--z-backdrop: 40;
--z-menu: 50;
```

### Testing Checklist

- [ ] Menu opens/closes with hamburger button
- [ ] All nav items keyboard accessible
- [ ] Focus trapped within menu
- [ ] Escape closes menu
- [ ] Focus returns to hamburger on close
- [ ] Swipe left closes menu
- [ ] Tap on backdrop closes menu
- [ ] Background scroll prevented
- [ ] Touch targets ≥ 44x44px
- [ ] Color contrast ≥ 4.5:1
- [ ] Screen reader announces open/close
- [ ] Reduced motion respected
- [ ] Works on iOS Safari
- [ ] Works on Android Chrome
