# Sidebar Component Specification

## Overview

The Sidebar component provides secondary navigation for documentation pages. It displays a hierarchical structure of documentation sections and pages, with collapsible sections and active state indicators.

---

## 1. Component Hierarchy

```
Sidebar (organism)
├── SidebarContainer (molecule)
│   ├── SidebarHeader (molecule)
│   │   ├── SidebarTitle (atom)
│   │   └── CollapseButton (atom) [tablet only]
│   ├── SidebarSearch (molecule) [optional]
│   │   ├── SearchInput (atom)
│   │   └── SearchResults (molecule)
│   ├── SidebarNav (molecule)
│   │   ├── SidebarSection (molecule) × n
│   │   │   ├── SectionHeader (atom)
│   │   │   └── SidebarLink (atom) × n
│   │   └── SidebarLink (atom) × n [top-level items]
│   └── SidebarFooter (molecule) [optional]
└── SidebarOverlay (atom) [mobile only]
```

---

## 2. Props Interface

### Sidebar

```typescript
interface SidebarProps {
  /** Navigation structure */
  navigation: SidebarNavigation;
  /** Current active path */
  currentPath: string;
  /** Whether sidebar is open (mobile) */
  isOpen?: boolean;
  /** Callback when sidebar open state changes */
  onOpenChange?: (isOpen: boolean) => void;
  /** Whether sidebar is collapsed (tablet) */
  isCollapsed?: boolean;
  /** Callback when collapse state changes */
  onCollapseChange?: (isCollapsed: boolean) => void;
  /** Enable search functionality */
  searchEnabled?: boolean;
  /** Custom class name */
  className?: string;
}

interface SidebarNavigation {
  sections: SidebarSection[];
}

interface SidebarSection {
  /** Section title */
  title: string;
  /** Unique identifier */
  id: string;
  /** Whether section is collapsible */
  collapsible?: boolean;
  /** Default expanded state */
  defaultExpanded?: boolean;
  /** Section items */
  items: SidebarItem[];
}

interface SidebarItem {
  /** Item label */
  label: string;
  /** Link destination */
  href: string;
  /** Optional icon */
  icon?: string;
  /** Optional badge (e.g., "New", "Beta") */
  badge?: {
    text: string;
    variant: 'default' | 'new' | 'beta' | 'deprecated';
  };
  /** Nested items */
  items?: SidebarItem[];
}
```

### SidebarSection

```typescript
interface SidebarSectionProps {
  /** Section title */
  title: string;
  /** Whether section is expanded */
  isExpanded: boolean;
  /** Toggle handler */
  onToggle: () => void;
  /** Whether section is collapsible */
  collapsible: boolean;
  /** Child items */
  children: React.ReactNode;
}
```

### SidebarLink

```typescript
interface SidebarLinkProps {
  /** Link label */
  label: string;
  /** Link destination */
  href: string;
  /** Whether link is active */
  isActive: boolean;
  /** Nesting depth (0, 1, 2) */
  depth: number;
  /** Optional icon */
  icon?: string;
  /** Optional badge */
  badge?: {
    text: string;
    variant: 'default' | 'new' | 'beta' | 'deprecated';
  };
  /** onClick handler */
  onClick?: () => void;
}
```

---

## 3. States

### Sidebar States

| State       | Trigger                    | Visual Changes                    |
|-------------|----------------------------|-----------------------------------|
| Default     | Desktop view               | Full width, visible               |
| Collapsed   | User collapses (tablet)    | Icons only, tooltips on hover     |
| Open        | Hamburger click (mobile)   | Overlay visible, slides in        |
| Closed      | Outside click (mobile)     | Hidden, overlay fades             |
| Loading     | Async navigation load      | Skeleton placeholders             |

### SidebarSection States

| State     | Trigger           | Visual Changes                        |
|-----------|-------------------|---------------------------------------|
| Expanded  | Click header      | Items visible, chevron rotated down   |
| Collapsed | Click header      | Items hidden, chevron pointing right  |
| HasActive | Child is active   | Subtle highlight on section header    |

### SidebarLink States

| State    | Trigger            | Visual Changes                        |
|----------|-------------------|---------------------------------------|
| Default  | Initial render     | Normal text color                     |
| Hover    | Mouse enter        | Background highlight                  |
| Active   | Current page       | Primary color, left border, bold      |
| Focus    | Keyboard focus     | Focus ring visible                    |
| Disabled | `disabled={true}`  | Muted color, no interaction           |

---

## 4. Visual Specifications

### Layout

```
Desktop (≥1024px):
┌──────────────────────┬─────────────────────────────────────────┐
│  Documentation       │                                         │
│  ─────────────────   │                                         │
│  📖 Getting Started  │                                         │
│    · Quick Start     │            Main Content                 │
│    · Installation    │                                         │
│    · Configuration   │                                         │
│  ─────────────────   │                                         │
│  🤖 MCP Server       │                                         │
│    · Setup           │                                         │
│    · Commands        │                                         │
│    · Integration     │                                         │
│  ─────────────────   │                                         │
│  📚 Commands         │                                         │
│    · run             │                                         │
│    · logs            │                                         │
│    · health          │                                         │
└──────────────────────┴─────────────────────────────────────────┘

Tablet - Collapsed (768-1023px):
┌────┬─────────────────────────────────────────────────────────┐
│ 📖 │                                                         │
│ 🤖 │                    Main Content                         │
│ 📚 │                                                         │
└────┴─────────────────────────────────────────────────────────┘

Mobile (<768px):
┌─────────────────────────────────────┐
│         Main Content                │
│  (Sidebar in overlay when open)     │
└─────────────────────────────────────┘
```

### Dimensions

| Property              | Desktop    | Tablet (Collapsed) | Mobile      |
|-----------------------|------------|-------------------|-------------|
| Width                 | 256px      | 64px              | 280px       |
| Min height            | 100vh      | 100vh             | 100vh       |
| Padding               | 16px       | 8px               | 16px        |
| Section gap           | 24px       | 16px              | 24px        |
| Item height           | 40px       | 48px              | 48px        |
| Item indent (depth 1) | 16px       | 0                 | 16px        |
| Item indent (depth 2) | 32px       | 0                 | 32px        |

### Typography

| Element          | Font Size | Weight    | Line Height |
|------------------|-----------|-----------|-------------|
| Section title    | 12px      | Semibold  | 1.25        |
| Link (depth 0)   | 14px      | Medium    | 1.5         |
| Link (depth 1)   | 14px      | Normal    | 1.5         |
| Link (depth 2)   | 13px      | Normal    | 1.5         |
| Badge text       | 10px      | Semibold  | 1           |

### Colors

#### Light Theme

| Element                | Property    | Value               |
|------------------------|-------------|---------------------|
| Sidebar bg             | background  | #f9fafb             |
| Section title          | color       | #6b7280             |
| Link text              | color       | #374151             |
| Link hover bg          | background  | #f3f4f6             |
| Link active text       | color       | #2563eb             |
| Link active bg         | background  | #eff6ff             |
| Link active border     | border-left | 2px solid #2563eb   |
| Border                 | border-right| 1px solid #e5e7eb   |
| Badge (new)            | background  | #dcfce7             |
| Badge (new) text       | color       | #166534             |
| Badge (beta)           | background  | #dbeafe             |
| Badge (beta) text      | color       | #1e40af             |
| Badge (deprecated)     | background  | #fee2e2             |
| Badge (deprecated) text| color       | #991b1b             |

#### Dark Theme

| Element                | Property    | Value               |
|------------------------|-------------|---------------------|
| Sidebar bg             | background  | #1e293b             |
| Section title          | color       | #94a3b8             |
| Link text              | color       | #cbd5e1             |
| Link hover bg          | background  | #334155             |
| Link active text       | color       | #60a5fa             |
| Link active bg         | background  | #1e3a5f             |
| Link active border     | border-left | 2px solid #60a5fa   |
| Border                 | border-right| 1px solid #334155   |

---

## 5. Interactions

### Desktop

| Action                  | Behavior                                        |
|-------------------------|------------------------------------------------|
| Click section header    | Toggle section expanded/collapsed               |
| Click link              | Navigate to page, update active state           |
| Hover link              | Show hover background                           |
| Scroll                  | Sidebar scrolls independently of content        |

### Tablet (Collapsed)

| Action                  | Behavior                                        |
|-------------------------|------------------------------------------------|
| Click expand button     | Expand sidebar to full width                    |
| Hover collapsed icon    | Show tooltip with section/link name             |
| Click icon              | Navigate or expand section                      |

### Mobile

| Action                  | Behavior                                        |
|-------------------------|------------------------------------------------|
| Click hamburger         | Open sidebar overlay                            |
| Click link              | Navigate and close sidebar                      |
| Click overlay backdrop  | Close sidebar                                   |
| Swipe left              | Close sidebar                                   |

### Keyboard

| Key          | Behavior                                           |
|--------------|---------------------------------------------------|
| Tab          | Move focus to next focusable element               |
| Shift+Tab    | Move focus to previous focusable element           |
| Enter        | Activate focused item (navigate or toggle)         |
| Space        | Toggle section if focused on section header        |
| Arrow Down   | Move focus to next item in sidebar                 |
| Arrow Up     | Move focus to previous item in sidebar             |
| Arrow Right  | Expand section if focused on collapsed section     |
| Arrow Left   | Collapse section if focused on expanded section    |
| Home         | Move focus to first item                           |
| End          | Move focus to last visible item                    |
| Escape       | Close sidebar (mobile)                             |

---

## 6. Accessibility

### ARIA Attributes

```html
<aside 
  aria-label="Documentation navigation"
  class="sidebar"
>
  <nav aria-label="Documentation sections">
    <div role="group" aria-labelledby="section-getting-started">
      <button
        id="section-getting-started"
        aria-expanded="true"
        aria-controls="section-getting-started-items"
        class="section-header"
      >
        <span>Getting Started</span>
        <svg aria-hidden="true"><!-- chevron --></svg>
      </button>
      
      <ul id="section-getting-started-items" role="list">
        <li>
          <a 
            href="/docs/quick-start"
            aria-current="page"
          >
            Quick Start
          </a>
        </li>
        <li>
          <a href="/docs/installation">
            Installation
            <span class="badge" aria-label="New feature">New</span>
          </a>
        </li>
      </ul>
    </div>
  </nav>
</aside>

<!-- Mobile overlay -->
<div 
  class="sidebar-overlay"
  aria-hidden="true"
  inert
></div>
```

### Semantic HTML

- `<aside>` as landmark for sidebar
- `<nav>` with `aria-label` for navigation
- `<button>` for collapsible section headers
- `<a>` for navigation links
- `<ul>` / `<li>` for list structure
- `aria-current="page"` for active link

### Focus Management

- Visible focus indicators on all interactive elements
- Focus trapped in mobile overlay when open
- Focus returns to trigger when overlay closes
- Logical tab order top-to-bottom

### Screen Reader Announcements

- Section toggle: "Getting Started, expanded" / "Getting Started, collapsed"
- Active page: "Current page: Quick Start"
- Badge content: "Installation, New feature"

---

## 7. Responsive Behavior

### Breakpoints

| Breakpoint | Sidebar Behavior                                    |
|------------|-----------------------------------------------------|
| ≥1024px    | Always visible, full width (256px)                  |
| 768-1023px | Collapsible (64px collapsed, 256px expanded)        |
| <768px     | Hidden by default, overlay when open (280px)        |

### Collapse Animation (Tablet)

```css
.sidebar {
  width: 256px;
  transition: width 200ms ease-out;
}

.sidebar--collapsed {
  width: 64px;
}

.sidebar__content {
  opacity: 1;
  transition: opacity 150ms ease-out;
}

.sidebar--collapsed .sidebar__content {
  opacity: 0;
  pointer-events: none;
}

.sidebar__icons {
  opacity: 0;
  transition: opacity 150ms ease-out 50ms;
}

.sidebar--collapsed .sidebar__icons {
  opacity: 1;
}
```

### Mobile Overlay

```css
.sidebar-overlay {
  position: fixed;
  inset: 0;
  z-index: 40;
  background: rgba(0, 0, 0, 0.5);
  opacity: 0;
  visibility: hidden;
  transition: opacity 200ms ease-out, visibility 0ms 200ms;
}

.sidebar-overlay--visible {
  opacity: 1;
  visibility: visible;
  transition: opacity 200ms ease-out;
}

.sidebar--mobile {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: 280px;
  z-index: 50;
  transform: translateX(-100%);
  transition: transform 200ms ease-out;
}

.sidebar--mobile.sidebar--open {
  transform: translateX(0);
}
```

---

## 8. Animation Specifications

### Section Toggle

```css
.section-items {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows 200ms ease-out;
}

.section-items--expanded {
  grid-template-rows: 1fr;
}

.section-items__inner {
  overflow: hidden;
}

.section-chevron {
  transition: transform 200ms ease-out;
}

.section--expanded .section-chevron {
  transform: rotate(90deg);
}
```

### Link Hover

```css
.sidebar-link {
  transition: background-color 100ms ease-out, color 100ms ease-out;
}
```

### Active Indicator

```css
.sidebar-link {
  position: relative;
}

.sidebar-link::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%) scaleY(0);
  width: 2px;
  height: 24px;
  background: var(--color-interactive-default);
  transition: transform 150ms ease-out;
}

.sidebar-link--active::before {
  transform: translateY(-50%) scaleY(1);
}
```

---

## 9. Scroll Behavior

### Sticky Positioning

```css
.sidebar {
  position: sticky;
  top: 64px; /* header height */
  height: calc(100vh - 64px);
  overflow-y: auto;
  overscroll-behavior: contain;
}
```

### Scroll Indicators

```css
.sidebar::before,
.sidebar::after {
  content: '';
  position: sticky;
  display: block;
  height: 24px;
  pointer-events: none;
}

.sidebar::before {
  top: 0;
  background: linear-gradient(
    to bottom,
    var(--color-bg-secondary),
    transparent
  );
}

.sidebar::after {
  bottom: 0;
  background: linear-gradient(
    to top,
    var(--color-bg-secondary),
    transparent
  );
}
```

### Scroll Into View

When a page loads with an active item not visible:

```javascript
// Scroll active item into view smoothly
const activeItem = sidebar.querySelector('[aria-current="page"]');
if (activeItem) {
  activeItem.scrollIntoView({ 
    behavior: 'smooth', 
    block: 'center' 
  });
}
```

---

## 10. Search Integration (Optional)

### Search Input

```html
<div class="sidebar-search">
  <label for="sidebar-search" class="sr-only">
    Search documentation
  </label>
  <input
    type="search"
    id="sidebar-search"
    placeholder="Search..."
    aria-describedby="search-hint"
  />
  <span id="search-hint" class="sr-only">
    Type to search documentation pages
  </span>
</div>
```

### Search Results

- Filter visible items as user types
- Highlight matching text
- Show "No results" message when empty
- Keyboard navigation through results

---

## 11. Error States

| Scenario              | Behavior                                         |
|-----------------------|-------------------------------------------------|
| Navigation load error | Show retry button with error message             |
| Empty section         | Hide section or show "Coming soon" message       |
| Invalid link          | 404 handling at page level                       |
| JavaScript disabled   | All links work, sections always expanded         |

---

## 12. Implementation Notes

### Performance

- Virtualize long lists (>50 items)
- Lazy load collapsed section content
- Debounce search input (150ms)
- Use CSS for animations

### State Persistence

- Remember expanded/collapsed sections in localStorage
- Key: `azd-app:sidebar:sections`
- Restore on page load

### SSR Considerations

- Render all sections expanded initially
- Hydrate collapsed state from localStorage
- Ensure links work without JavaScript

### Testing Checklist

- [ ] All links keyboard accessible
- [ ] Section toggle works with keyboard
- [ ] Screen reader announces states correctly
- [ ] Mobile overlay traps focus
- [ ] Escape closes mobile sidebar
- [ ] Touch targets ≥ 44x44px on mobile
- [ ] Color contrast ≥ 4.5:1
- [ ] Works with JavaScript disabled
- [ ] Reduced motion respected
- [ ] Active item scrolled into view on load
