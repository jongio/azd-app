# Layout Component Specification

## Overview

The Layout component provides the structural foundation for all pages on the azd-app marketing website. It orchestrates the Header, Sidebar (optional), Footer, and main content area, ensuring consistent spacing, responsive behavior, and proper landmark structure for accessibility.

---

## 1. Component Hierarchy

```
Layout (template)
├── SkipLinks (atom)
├── Header (organism)
├── LayoutContainer (molecule)
│   ├── Sidebar (organism) [optional, for docs pages]
│   └── MainContent (molecule)
│       ├── Breadcrumb (molecule) [optional]
│       ├── PageContent (slot)
│       └── PageNavigation (molecule) [prev/next for docs]
├── Footer (organism)
└── MobileMenu (organism) [rendered via portal]
```

---

## 2. Props Interface

### Layout

```typescript
interface LayoutProps {
  /** Page variant */
  variant?: 'default' | 'docs' | 'full-width' | 'landing';
  /** Page title for meta */
  title: string;
  /** Page description for meta */
  description?: string;
  /** Show sidebar */
  showSidebar?: boolean;
  /** Sidebar navigation structure */
  sidebarNavigation?: SidebarNavigation;
  /** Current path for active states */
  currentPath: string;
  /** Breadcrumb items */
  breadcrumb?: BreadcrumbItem[];
  /** Previous page link (docs) */
  prevPage?: PageLink;
  /** Next page link (docs) */
  nextPage?: PageLink;
  /** Page content */
  children: React.ReactNode;
  /** Custom class for main content */
  contentClassName?: string;
}

interface BreadcrumbItem {
  label: string;
  href?: string;
}

interface PageLink {
  label: string;
  href: string;
}
```

### SkipLinks

```typescript
interface SkipLinksProps {
  /** Links to skip to */
  links: SkipLink[];
}

interface SkipLink {
  label: string;
  href: string; // e.g., "#main-content"
}
```

---

## 3. Layout Variants

### Default Layout

Standard pages with header, content, and footer.

```
┌─────────────────────────────────────────────┐
│                   Header                     │
├─────────────────────────────────────────────┤
│                                             │
│               Main Content                  │
│            (max-width: 1280px)              │
│                                             │
├─────────────────────────────────────────────┤
│                   Footer                     │
└─────────────────────────────────────────────┘
```

### Docs Layout

Documentation pages with sidebar navigation.

```
┌─────────────────────────────────────────────┐
│                   Header                     │
├──────────┬──────────────────────────────────┤
│          │                                  │
│ Sidebar  │         Main Content             │
│ (256px)  │       (max-width: 768px)         │
│          │                                  │
│          │                                  │
├──────────┴──────────────────────────────────┤
│                   Footer                     │
└─────────────────────────────────────────────┘
```

### Full-Width Layout

Edge-to-edge content (e.g., hero sections).

```
┌─────────────────────────────────────────────┐
│                   Header                     │
├─────────────────────────────────────────────┤
│                                             │
│             Main Content                    │
│              (100% width)                   │
│                                             │
├─────────────────────────────────────────────┤
│                   Footer                     │
└─────────────────────────────────────────────┘
```

### Landing Layout

Home page with prominent hero and no sidebar.

```
┌─────────────────────────────────────────────┐
│              Header (transparent)            │
├─────────────────────────────────────────────┤
│                                             │
│                 Hero Section                │
│              (full viewport)                │
│                                             │
├─────────────────────────────────────────────┤
│                                             │
│               Feature Sections              │
│                                             │
├─────────────────────────────────────────────┤
│                   Footer                     │
└─────────────────────────────────────────────┘
```

---

## 4. Visual Specifications

### Dimensions

| Property                | Desktop     | Tablet      | Mobile      |
|-------------------------|-------------|-------------|-------------|
| Header height           | 64px        | 64px        | 56px        |
| Sidebar width           | 256px       | 64px (icons)| Hidden      |
| Content max-width       | 1280px      | 100%        | 100%        |
| Content max-width (docs)| 768px       | 100%        | 100%        |
| Content padding         | 48px 24px   | 32px 16px   | 24px 16px   |
| Footer height           | auto        | auto        | auto        |
| Main min-height         | calc(100vh - header - footer) |

### Spacing

```css
/* Content area spacing */
--layout-content-padding-x: 24px;
--layout-content-padding-y: 48px;

/* Sidebar gap */
--layout-sidebar-gap: 24px;

/* Section spacing within content */
--layout-section-gap: 64px;

/* Mobile adjustments */
@media (max-width: 767px) {
  --layout-content-padding-x: 16px;
  --layout-content-padding-y: 24px;
  --layout-section-gap: 32px;
}
```

### Colors

| Element              | Light Theme | Dark Theme  |
|----------------------|-------------|-------------|
| Page background      | #ffffff     | #0f172a     |
| Content background   | #ffffff     | #0f172a     |
| Sidebar background   | #f9fafb     | #1e293b     |

---

## 5. Skip Links

### Implementation

```html
<div class="skip-links">
  <a href="#main-content" class="skip-link">
    Skip to main content
  </a>
  <a href="#navigation" class="skip-link">
    Skip to navigation
  </a>
</div>
```

### Styling

```css
.skip-links {
  position: absolute;
  top: 0;
  left: 0;
  z-index: 100;
}

.skip-link {
  position: absolute;
  top: -100%;
  left: 16px;
  padding: 12px 24px;
  background: var(--color-interactive-default);
  color: white;
  font-weight: 600;
  text-decoration: none;
  border-radius: 0 0 8px 8px;
  transition: top 0.2s ease-out;
}

.skip-link:focus {
  top: 0;
  outline: 2px solid var(--color-border-focus);
  outline-offset: 2px;
}
```

---

## 6. Responsive Grid

### Desktop (≥1024px)

```css
.layout--docs {
  display: grid;
  grid-template-columns: 256px 1fr;
  grid-template-rows: auto 1fr auto;
  grid-template-areas:
    "header header"
    "sidebar main"
    "footer footer";
  min-height: 100vh;
}

.layout__header { grid-area: header; }
.layout__sidebar { grid-area: sidebar; }
.layout__main { grid-area: main; }
.layout__footer { grid-area: footer; }
```

### Tablet (768-1023px)

```css
@media (max-width: 1023px) {
  .layout--docs {
    grid-template-columns: 64px 1fr;
  }
  
  /* Sidebar collapsed to icons */
  .layout__sidebar {
    width: 64px;
  }
}
```

### Mobile (<768px)

```css
@media (max-width: 767px) {
  .layout--docs {
    display: flex;
    flex-direction: column;
  }
  
  /* Sidebar hidden, accessed via mobile menu */
  .layout__sidebar {
    display: none;
  }
}
```

---

## 7. Sticky Header Behavior

### CSS Implementation

```css
.layout__header {
  position: sticky;
  top: 0;
  z-index: var(--z-sticky);
  transition: box-shadow 0.2s ease-out;
}

.layout__header--scrolled {
  box-shadow: var(--shadow-md);
}
```

### JavaScript Enhancement

```typescript
// Detect scroll for shadow effect
let lastScrollY = 0;
let ticking = false;

function onScroll(): void {
  if (!ticking) {
    requestAnimationFrame(() => {
      const header = document.querySelector('.layout__header');
      if (window.scrollY > 20) {
        header?.classList.add('layout__header--scrolled');
      } else {
        header?.classList.remove('layout__header--scrolled');
      }
      ticking = false;
    });
    ticking = true;
  }
}

window.addEventListener('scroll', onScroll, { passive: true });
```

---

## 8. Content Containers

### MaxWidth Container

```css
.container {
  width: 100%;
  max-width: var(--container-max-width, 1280px);
  margin-left: auto;
  margin-right: auto;
  padding-left: var(--layout-content-padding-x);
  padding-right: var(--layout-content-padding-x);
}

.container--narrow {
  --container-max-width: 768px;
}

.container--wide {
  --container-max-width: 1536px;
}
```

### Content Wrapper

```css
.layout__content {
  flex: 1;
  padding: var(--layout-content-padding-y) var(--layout-content-padding-x);
}

.layout--docs .layout__content {
  max-width: 768px;
}
```

---

## 9. Accessibility

### ARIA Landmarks

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Page Title | azd-app</title>
</head>
<body class="layout layout--docs">
  <!-- Skip links -->
  <div class="skip-links">
    <a href="#main-content" class="skip-link">Skip to main content</a>
    <a href="#sidebar-nav" class="skip-link">Skip to navigation</a>
  </div>
  
  <!-- Header landmark -->
  <header class="layout__header" role="banner">
    <nav aria-label="Main navigation" id="navigation">
      <!-- Header content -->
    </nav>
  </header>
  
  <!-- Sidebar (complementary) -->
  <aside class="layout__sidebar" aria-label="Documentation navigation">
    <nav id="sidebar-nav" aria-label="Documentation sections">
      <!-- Sidebar content -->
    </nav>
  </aside>
  
  <!-- Main content landmark -->
  <main id="main-content" class="layout__main" role="main">
    <article class="layout__content">
      <!-- Breadcrumb -->
      <nav aria-label="Breadcrumb">
        <ol>
          <li><a href="/">Home</a></li>
          <li><a href="/docs">Docs</a></li>
          <li aria-current="page">Quick Start</li>
        </ol>
      </nav>
      
      <!-- Page content -->
      <h1>Page Title</h1>
      <!-- ... -->
      
      <!-- Page navigation -->
      <nav aria-label="Page navigation">
        <a href="/prev" rel="prev">Previous: Installation</a>
        <a href="/next" rel="next">Next: Configuration</a>
      </nav>
    </article>
  </main>
  
  <!-- Footer landmark -->
  <footer class="layout__footer" role="contentinfo">
    <!-- Footer content -->
  </footer>
  
  <!-- Mobile menu (portal) -->
  <div id="mobile-menu-portal"></div>
</body>
</html>
```

### Landmark Regions

| Element  | Role          | aria-label                  |
|----------|---------------|-----------------------------|
| header   | banner        | (implicit)                  |
| nav      | navigation    | "Main navigation"           |
| aside    | complementary | "Documentation navigation"  |
| main     | main          | (implicit)                  |
| footer   | contentinfo   | (implicit)                  |

---

## 10. Breadcrumb Component

### HTML Structure

```html
<nav aria-label="Breadcrumb" class="breadcrumb">
  <ol class="breadcrumb__list">
    <li class="breadcrumb__item">
      <a href="/" class="breadcrumb__link">Home</a>
      <span class="breadcrumb__separator" aria-hidden="true">/</span>
    </li>
    <li class="breadcrumb__item">
      <a href="/docs" class="breadcrumb__link">Documentation</a>
      <span class="breadcrumb__separator" aria-hidden="true">/</span>
    </li>
    <li class="breadcrumb__item">
      <span aria-current="page" class="breadcrumb__current">Quick Start</span>
    </li>
  </ol>
</nav>
```

### Styling

```css
.breadcrumb {
  margin-bottom: 24px;
  font-size: 14px;
}

.breadcrumb__list {
  display: flex;
  flex-wrap: wrap;
  list-style: none;
  margin: 0;
  padding: 0;
}

.breadcrumb__item {
  display: flex;
  align-items: center;
}

.breadcrumb__link {
  color: var(--color-text-secondary);
  text-decoration: none;
}

.breadcrumb__link:hover {
  color: var(--color-text-primary);
  text-decoration: underline;
}

.breadcrumb__separator {
  margin: 0 8px;
  color: var(--color-text-muted);
}

.breadcrumb__current {
  color: var(--color-text-primary);
  font-weight: 500;
}
```

---

## 11. Page Navigation (Prev/Next)

### HTML Structure

```html
<nav aria-label="Page navigation" class="page-nav">
  <a href="/docs/installation" class="page-nav__link page-nav__link--prev" rel="prev">
    <span class="page-nav__direction">Previous</span>
    <span class="page-nav__title">Installation</span>
  </a>
  <a href="/docs/configuration" class="page-nav__link page-nav__link--next" rel="next">
    <span class="page-nav__direction">Next</span>
    <span class="page-nav__title">Configuration</span>
  </a>
</nav>
```

### Styling

```css
.page-nav {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-top: 64px;
  padding-top: 32px;
  border-top: 1px solid var(--color-border-default);
}

.page-nav__link {
  display: flex;
  flex-direction: column;
  padding: 16px;
  border: 1px solid var(--color-border-default);
  border-radius: 8px;
  text-decoration: none;
  transition: border-color 0.1s ease-out, background-color 0.1s ease-out;
}

.page-nav__link:hover {
  border-color: var(--color-border-strong);
  background-color: var(--color-bg-secondary);
}

.page-nav__link--prev {
  align-items: flex-start;
}

.page-nav__link--next {
  align-items: flex-end;
  text-align: right;
}

.page-nav__direction {
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
}

.page-nav__title {
  font-size: 16px;
  font-weight: 500;
  color: var(--color-interactive-default);
}

@media (max-width: 480px) {
  .page-nav {
    grid-template-columns: 1fr;
  }
  
  .page-nav__link--next {
    align-items: flex-start;
    text-align: left;
  }
}
```

---

## 12. Error States

| Scenario                | Behavior                                     |
|-------------------------|---------------------------------------------|
| JavaScript disabled     | Layout works, mobile menu via CSS fallback  |
| Slow network            | SSR content visible immediately             |
| Missing sidebar data    | Sidebar hidden, full-width content          |
| 404 page               | Use default layout without sidebar          |

---

## 13. Performance

### Critical CSS

Inline critical layout CSS in `<head>`:

```css
/* Critical layout styles */
.layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.layout__main {
  flex: 1;
}

.layout__header {
  position: sticky;
  top: 0;
  z-index: 20;
  background: var(--color-bg-primary);
}
```

### Content-Visibility

```css
.layout__footer {
  content-visibility: auto;
  contain-intrinsic-size: 0 300px;
}
```

---

## 14. Testing Checklist

- [ ] Skip links work and are visible on focus
- [ ] All landmark regions properly identified
- [ ] Heading hierarchy correct (single h1)
- [ ] Tab order follows visual layout
- [ ] Sidebar collapses correctly at breakpoints
- [ ] Mobile layout stacks properly
- [ ] Footer always at bottom (min-height works)
- [ ] Sticky header works correctly
- [ ] Page nav accessible with keyboard
- [ ] Breadcrumb announces correctly
- [ ] Works with JavaScript disabled
- [ ] Reduced motion preferences respected
