# azd-app Design System

## Overview

This design system defines the foundational tokens, patterns, and guidelines for the azd-app marketing website. It ensures visual consistency, accessibility compliance (WCAG 2.1 AA), and a cohesive user experience across all components.

---

## 1. Color Tokens

### Brand Colors (Azure-inspired)

```css
--color-azure-50: #eff6ff;
--color-azure-100: #dbeafe;
--color-azure-200: #bfdbfe;
--color-azure-300: #93c5fd;
--color-azure-400: #60a5fa;
--color-azure-500: #3b82f6;
--color-azure-600: #2563eb;
--color-azure-700: #1d4ed8;
--color-azure-800: #1e40af;
--color-azure-900: #1e3a8a;
```

### Semantic Colors

#### Light Theme

```css
--color-bg-primary: #ffffff;
--color-bg-secondary: #f9fafb;
--color-bg-tertiary: #f3f4f6;
--color-bg-accent: #eff6ff;

--color-text-primary: #111827;
--color-text-secondary: #4b5563;
--color-text-tertiary: #6b7280;
--color-text-muted: #9ca3af;
--color-text-inverse: #ffffff;

--color-border-default: #e5e7eb;
--color-border-strong: #d1d5db;
--color-border-focus: #3b82f6;

--color-interactive-default: #2563eb;
--color-interactive-hover: #1d4ed8;
--color-interactive-active: #1e40af;

--color-mcp-badge-bg: #fef3c7;
--color-mcp-badge-text: #92400e;
--color-mcp-badge-border: #f59e0b;
```

#### Dark Theme

```css
--color-bg-primary: #0f172a;
--color-bg-secondary: #1e293b;
--color-bg-tertiary: #334155;
--color-bg-accent: #1e3a5f;

--color-text-primary: #f1f5f9;
--color-text-secondary: #cbd5e1;
--color-text-tertiary: #94a3b8;
--color-text-muted: #64748b;
--color-text-inverse: #0f172a;

--color-border-default: #334155;
--color-border-strong: #475569;
--color-border-focus: #60a5fa;

--color-interactive-default: #3b82f6;
--color-interactive-hover: #60a5fa;
--color-interactive-active: #93c5fd;

--color-mcp-badge-bg: #422006;
--color-mcp-badge-text: #fcd34d;
--color-mcp-badge-border: #f59e0b;
```

### State Colors

```css
--color-success: #10b981;
--color-warning: #f59e0b;
--color-error: #ef4444;
--color-info: #3b82f6;
```

---

## 2. Typography Tokens

### Font Families

```css
--font-family-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
--font-family-mono: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
```

### Font Sizes (rem-based for accessibility)

```css
--font-size-xs: 0.75rem;     /* 12px */
--font-size-sm: 0.875rem;    /* 14px */
--font-size-base: 1rem;      /* 16px */
--font-size-lg: 1.125rem;    /* 18px */
--font-size-xl: 1.25rem;     /* 20px */
--font-size-2xl: 1.5rem;     /* 24px */
--font-size-3xl: 1.875rem;   /* 30px */
--font-size-4xl: 2.25rem;    /* 36px */
--font-size-5xl: 3rem;       /* 48px */
```

### Font Weights

```css
--font-weight-normal: 400;
--font-weight-medium: 500;
--font-weight-semibold: 600;
--font-weight-bold: 700;
```

### Line Heights

```css
--line-height-tight: 1.25;
--line-height-normal: 1.5;
--line-height-relaxed: 1.75;
```

---

## 3. Spacing Tokens

Based on 4px grid system:

```css
--spacing-0: 0;
--spacing-1: 0.25rem;   /* 4px */
--spacing-2: 0.5rem;    /* 8px */
--spacing-3: 0.75rem;   /* 12px */
--spacing-4: 1rem;      /* 16px */
--spacing-5: 1.25rem;   /* 20px */
--spacing-6: 1.5rem;    /* 24px */
--spacing-8: 2rem;      /* 32px */
--spacing-10: 2.5rem;   /* 40px */
--spacing-12: 3rem;     /* 48px */
--spacing-16: 4rem;     /* 64px */
--spacing-20: 5rem;     /* 80px */
--spacing-24: 6rem;     /* 96px */
```

---

## 4. Breakpoints

```css
--breakpoint-sm: 640px;   /* Mobile landscape */
--breakpoint-md: 768px;   /* Tablet */
--breakpoint-lg: 1024px;  /* Desktop */
--breakpoint-xl: 1280px;  /* Large desktop */
--breakpoint-2xl: 1536px; /* Extra large */
```

### Responsive Behavior Matrix

| Component      | Mobile (<768px)    | Tablet (768-1023px) | Desktop (≥1024px) |
|----------------|--------------------|--------------------|-------------------|
| Header         | Hamburger menu     | Hamburger menu     | Full nav visible  |
| Sidebar        | Hidden (overlay)   | Collapsible        | Always visible    |
| Footer         | Stacked columns    | 2-column grid      | 4-column grid     |
| Theme Toggle   | Icon only          | Icon only          | Icon + label      |

---

## 5. Animation Tokens

### Duration

```css
--duration-instant: 0ms;
--duration-fast: 100ms;
--duration-normal: 200ms;
--duration-slow: 300ms;
--duration-slower: 500ms;
```

### Easing

```css
--ease-default: cubic-bezier(0.4, 0, 0.2, 1);
--ease-in: cubic-bezier(0.4, 0, 1, 1);
--ease-out: cubic-bezier(0, 0, 0.2, 1);
--ease-in-out: cubic-bezier(0.4, 0, 0.2, 1);
--ease-bounce: cubic-bezier(0.68, -0.55, 0.265, 1.55);
```

### Predefined Animations

```css
/* Theme transition */
--animation-theme-switch: transform 0.3s var(--ease-bounce);

/* Menu slide */
--animation-menu-slide: transform 0.2s var(--ease-out);

/* Fade in */
--animation-fade-in: opacity 0.2s var(--ease-out);
```

---

## 6. Shadow Tokens

```css
--shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.05);
--shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1);
--shadow-lg: 0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1);
--shadow-xl: 0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1);

/* Focus ring */
--shadow-focus: 0 0 0 3px rgba(59, 130, 246, 0.5);
```

---

## 7. Border Radius Tokens

```css
--radius-none: 0;
--radius-sm: 0.125rem;   /* 2px */
--radius-md: 0.375rem;   /* 6px */
--radius-lg: 0.5rem;     /* 8px */
--radius-xl: 0.75rem;    /* 12px */
--radius-2xl: 1rem;      /* 16px */
--radius-full: 9999px;
```

---

## 8. Z-Index Scale

```css
--z-dropdown: 10;
--z-sticky: 20;
--z-fixed: 30;
--z-modal-backdrop: 40;
--z-modal: 50;
--z-popover: 60;
--z-tooltip: 70;
```

---

## 9. Focus States

All interactive elements must have visible focus indicators for keyboard navigation:

```css
/* Default focus ring */
.focus-ring {
  outline: 2px solid var(--color-border-focus);
  outline-offset: 2px;
}

/* Focus visible (keyboard only) */
.focus-visible:focus:not(:focus-visible) {
  outline: none;
}

.focus-visible:focus-visible {
  outline: 2px solid var(--color-border-focus);
  outline-offset: 2px;
}
```

---

## 10. Motion Preferences

Respect user's reduced motion preference:

```css
@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

---

## 11. Dark Mode Implementation

### CSS Variable Approach

```css
:root {
  /* Light theme is default */
  color-scheme: light;
}

[data-theme="dark"] {
  color-scheme: dark;
}

@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
    /* Apply dark theme when system prefers dark and no explicit light preference */
    color-scheme: dark;
  }
}
```

### Theme Transition

```css
:root {
  transition: background-color var(--duration-normal) var(--ease-default),
              color var(--duration-normal) var(--ease-default);
}
```

---

## 12. Accessibility Guidelines

### Color Contrast Requirements

| Element Type    | Minimum Contrast (AA) | Enhanced (AAA) |
|-----------------|----------------------|----------------|
| Normal text     | 4.5:1               | 7:1            |
| Large text      | 3:1                 | 4.5:1          |
| UI components   | 3:1                 | N/A            |
| Focus indicator | 3:1                 | N/A            |

### Touch Targets

- Minimum size: 44x44 CSS pixels
- Adequate spacing between targets: 8px minimum

### Keyboard Navigation

- All interactive elements must be focusable
- Logical tab order (follows visual layout)
- Skip links for main content
- Focus trapping in modals/overlays

---

## 13. Component Tokens

### Button Sizes

```css
--button-height-sm: 2rem;      /* 32px */
--button-height-md: 2.5rem;    /* 40px */
--button-height-lg: 3rem;      /* 48px */

--button-padding-sm: 0.75rem;  /* 12px */
--button-padding-md: 1rem;     /* 16px */
--button-padding-lg: 1.5rem;   /* 24px */
```

### Nav Item Sizes

```css
--nav-item-height: 2.5rem;     /* 40px */
--nav-item-padding-x: 1rem;    /* 16px */
--sidebar-width: 16rem;        /* 256px */
--sidebar-width-collapsed: 4rem; /* 64px */
```

### Header Dimensions

```css
--header-height: 4rem;         /* 64px */
--header-height-mobile: 3.5rem; /* 56px */
```

### Footer Dimensions

```css
--footer-padding-y: 3rem;      /* 48px */
--footer-gap: 2rem;            /* 32px */
```
