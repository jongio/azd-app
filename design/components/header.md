# Header Component Specification

## Overview

The Header component provides primary navigation for the azd-app marketing website. It features a sticky behavior, prominent MCP Server link, responsive design with hamburger menu, and integrated theme toggle.

---

## 1. Component Hierarchy

```
Header (organism)
├── HeaderContainer (molecule)
│   ├── Logo (atom)
│   ├── MainNav (molecule)
│   │   ├── NavItem (atom) × 5
│   │   └── MCPBadge (atom)
│   ├── ThemeToggle (atom)
│   └── MobileMenuButton (atom)
└── MobileMenu (organism)
    ├── MobileNavItem (atom) × 5
    └── ThemeToggle (atom)
```

---

## 2. Props Interface

### Header

```typescript
interface HeaderProps {
  /** Current active route path */
  currentPath: string;
  /** Initial theme value */
  initialTheme?: 'light' | 'dark' | 'system';
  /** Callback when theme changes */
  onThemeChange?: (theme: 'light' | 'dark' | 'system') => void;
  /** Whether header is in sticky mode */
  sticky?: boolean;
  /** Custom class name */
  className?: string;
}
```

### NavItem

```typescript
interface NavItemProps {
  /** Navigation item label */
  label: string;
  /** Link destination */
  href: string;
  /** Whether this item is currently active */
  isActive: boolean;
  /** Whether this is the MCP Server (prominent) link */
  isProminent?: boolean;
  /** Icon to display (optional) */
  icon?: React.ReactNode;
  /** Badge text (e.g., "AI" for MCP Server) */
  badge?: string;
  /** onClick handler for accessibility */
  onClick?: () => void;
}
```

### MobileMenuButton

```typescript
interface MobileMenuButtonProps {
  /** Whether the mobile menu is open */
  isOpen: boolean;
  /** Toggle handler */
  onToggle: () => void;
  /** Accessible label */
  ariaLabel?: string;
}
```

---

## 3. States

### Header States

| State     | Description                                           | Visual Changes                    |
|-----------|-------------------------------------------------------|-----------------------------------|
| Default   | Normal header appearance                              | White/dark bg based on theme      |
| Scrolled  | User has scrolled past threshold                      | Adds shadow, slight bg opacity    |
| MenuOpen  | Mobile menu is expanded                               | Hamburger becomes X, menu slides  |

### NavItem States

| State    | Trigger                | Visual Changes                                       |
|----------|------------------------|------------------------------------------------------|
| Default  | Initial render         | Standard text color                                  |
| Hover    | Mouse enter            | Text color intensifies, subtle bg                    |
| Active   | Current page           | Bold weight, underline indicator, primary color      |
| Focus    | Keyboard focus         | Focus ring visible                                   |
| Disabled | `disabled={true}`      | Muted color, cursor not-allowed (rare for nav)       |

### MCP Server NavItem States (Prominent)

| State    | Visual Changes                                                    |
|----------|-------------------------------------------------------------------|
| Default  | Badge visible, slightly larger text, gradient border              |
| Hover    | Badge glows, bg highlight, scale(1.02)                            |
| Active   | Filled badge bg, strong underline                                 |
| Focus    | Enhanced focus ring with brand color                              |

---

## 4. Visual Specifications

### Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  [Logo]         [Home] [Quick Start] [MCP Server✨] [Tour] [Ref]  🌙 │
│   azd-app                                            [AI]           │
└─────────────────────────────────────────────────────────────────────┘

Mobile:
┌───────────────────────────────────────┐
│  [Logo]                         [☰] 🌙 │
└───────────────────────────────────────┘
```

### Dimensions

| Property            | Desktop         | Tablet          | Mobile          |
|---------------------|-----------------|-----------------|-----------------|
| Height              | 64px            | 64px            | 56px            |
| Logo width          | 120px           | 100px           | 100px           |
| Nav item padding    | 16px            | 12px            | N/A             |
| Container max-width | 1280px          | 100%            | 100%            |
| Container padding   | 0 24px          | 0 16px          | 0 16px          |

### Typography

| Element             | Font Size | Weight    | Line Height |
|---------------------|-----------|-----------|-------------|
| Logo text           | 24px      | Bold (700)| 1.25        |
| Nav item            | 16px      | Medium    | 1.5         |
| MCP Server item     | 16px      | Semibold  | 1.5         |
| MCP badge           | 12px      | Semibold  | 1           |

### Colors

#### Light Theme

| Element                | Property    | Value               |
|------------------------|-------------|---------------------|
| Header bg              | background  | white               |
| Header border          | border-bottom | 1px solid #e5e7eb |
| Nav item text          | color       | #4b5563             |
| Nav item hover bg      | background  | #f3f4f6             |
| Nav item active        | color       | #2563eb             |
| MCP badge bg           | background  | #fef3c7             |
| MCP badge text         | color       | #92400e             |
| Scrolled shadow        | box-shadow  | 0 4px 6px rgba(0,0,0,0.05) |

#### Dark Theme

| Element                | Property    | Value               |
|------------------------|-------------|---------------------|
| Header bg              | background  | #1e293b             |
| Header border          | border-bottom | 1px solid #334155 |
| Nav item text          | color       | #cbd5e1             |
| Nav item hover bg      | background  | #334155             |
| Nav item active        | color       | #60a5fa             |
| MCP badge bg           | background  | #422006             |
| MCP badge text         | color       | #fcd34d             |

---

## 5. Interactions

### Desktop

| Action              | Behavior                                              |
|---------------------|-------------------------------------------------------|
| Click nav item      | Navigate to page, update active state                 |
| Hover nav item      | Show hover state with 100ms delay                     |
| Click theme toggle  | Cycle through light → dark → system                   |
| Scroll past 20px    | Add scrolled class with shadow                        |

### Mobile

| Action              | Behavior                                              |
|---------------------|-------------------------------------------------------|
| Click hamburger     | Open mobile menu with slide animation                 |
| Click X             | Close mobile menu                                     |
| Click nav item      | Navigate and close menu                               |
| Click outside menu  | Close menu                                            |
| Swipe left on menu  | Close menu                                            |

### Keyboard

| Key         | Behavior                                                |
|-------------|--------------------------------------------------------|
| Tab         | Move focus to next nav item                             |
| Shift+Tab   | Move focus to previous nav item                         |
| Enter       | Activate focused nav item                               |
| Space       | Activate focused nav item                               |
| Escape      | Close mobile menu if open                               |
| Arrow Left  | Move to previous nav item (when in nav)                 |
| Arrow Right | Move to next nav item (when in nav)                     |

---

## 6. Accessibility

### ARIA Attributes

```html
<header role="banner">
  <nav aria-label="Main navigation">
    <a href="/" aria-current="page">Home</a>
    <a href="/quick-start">Quick Start</a>
    <a href="/mcp-server" aria-describedby="mcp-badge">
      MCP Server
      <span id="mcp-badge" class="badge" aria-label="AI feature">AI</span>
    </a>
    <a href="/tour">Guided Tour</a>
    <a href="/reference">Reference</a>
  </nav>
  
  <button 
    aria-label="Toggle theme"
    aria-pressed="false"
    aria-describedby="theme-status"
  >
    <span id="theme-status" class="sr-only">Current theme: light</span>
  </button>
  
  <button
    aria-label="Open main menu"
    aria-expanded="false"
    aria-controls="mobile-menu"
  >
    <!-- hamburger icon -->
  </button>
</header>
```

### Semantic HTML

- `<header>` as landmark with `role="banner"`
- `<nav>` with `aria-label` for screen readers
- `<a>` elements for navigation (not buttons)
- `aria-current="page"` for active nav item
- Skip link to main content as first focusable element

### Focus Management

- Focus visible on all interactive elements
- Focus trapped in mobile menu when open
- Focus returns to hamburger button when menu closes
- Logical tab order left-to-right

### Screen Reader Announcements

- Theme change: "Theme changed to dark mode"
- Menu open: "Main menu opened"
- Menu close: "Main menu closed"
- Active page: "Current page: Quick Start"

---

## 7. Responsive Behavior

### Breakpoints

| Breakpoint | Layout                                           |
|------------|--------------------------------------------------|
| ≥1024px    | Full horizontal nav, theme toggle visible        |
| 768-1023px | Hamburger menu, compact spacing                  |
| <768px     | Hamburger menu, mobile-optimized touch targets   |

### Mobile Menu Behavior

```
┌─────────────────────────────────┐
│  [Logo]                   [X] 🌙 │
├─────────────────────────────────┤
│                                 │
│    Home                         │
│    ─────────────────            │
│    Quick Start                  │
│    ─────────────────            │
│    ★ MCP Server [AI]            │
│    ─────────────────            │
│    Guided Tour                  │
│    ─────────────────            │
│    Reference                    │
│                                 │
└─────────────────────────────────┘
```

- Full-screen overlay with backdrop blur
- Slide in from right (200ms ease-out)
- Each nav item: 56px height for touch
- MCP Server highlighted with star icon

---

## 8. Animation Specifications

### Scroll Shadow

```css
.header {
  transition: box-shadow 200ms ease-out;
}

.header--scrolled {
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}
```

### Mobile Menu

```css
.mobile-menu {
  transform: translateX(100%);
  transition: transform 200ms ease-out;
}

.mobile-menu--open {
  transform: translateX(0);
}

/* Backdrop */
.mobile-menu-backdrop {
  opacity: 0;
  transition: opacity 200ms ease-out;
}

.mobile-menu-backdrop--visible {
  opacity: 1;
  background: rgba(0, 0, 0, 0.5);
}
```

### Hamburger to X

```css
.hamburger-line {
  transition: transform 200ms ease-out, opacity 150ms ease-out;
}

.hamburger--open .line-1 {
  transform: rotate(45deg) translate(5px, 5px);
}

.hamburger--open .line-2 {
  opacity: 0;
}

.hamburger--open .line-3 {
  transform: rotate(-45deg) translate(5px, -5px);
}
```

### Nav Item Hover

```css
.nav-item {
  transition: color 100ms ease-out, background-color 100ms ease-out;
}

.nav-item--prominent {
  transition: transform 100ms ease-out, box-shadow 100ms ease-out;
}

.nav-item--prominent:hover {
  transform: scale(1.02);
}
```

---

## 9. Error States

| Scenario                  | Behavior                                          |
|---------------------------|---------------------------------------------------|
| JavaScript disabled       | Nav links work as standard links, no animations   |
| Slow network              | SSR content visible immediately                   |
| Theme storage unavailable | Fallback to system preference                     |

---

## 10. Implementation Notes

### Performance

- Use `will-change: transform` for animated elements
- Debounce scroll listener (16ms)
- Lazy load mobile menu component
- Use CSS transitions over JS animations where possible

### SSR Considerations

- Render with system theme initially
- Hydrate theme from localStorage on client
- Add `data-theme` attribute to `<html>` element
- Use CSS `color-scheme` property

### Testing Checklist

- [ ] All nav items keyboard accessible
- [ ] Focus visible on all interactive elements
- [ ] Screen reader announces active page
- [ ] Theme persists across page refreshes
- [ ] Mobile menu traps focus
- [ ] Escape closes mobile menu
- [ ] Touch targets ≥ 44x44px
- [ ] Color contrast ≥ 4.5:1
- [ ] Works with JavaScript disabled
- [ ] Reduced motion respected
