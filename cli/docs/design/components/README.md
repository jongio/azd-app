# Component Specifications Index

## Overview

This directory contains comprehensive component specifications for the azd-app marketing website layout and navigation system. All components are designed for WCAG 2.1 AA accessibility compliance.

---

## Document Structure

### Design System Foundation

| Document | Description |
|----------|-------------|
| [design-system.md](../design-system.md) | Core design tokens, colors, typography, spacing, animations |

### Component Specifications

| Component | File | Description |
|-----------|------|-------------|
| Layout | [layout.md](layout.md) | Page structure, grid, landmarks, skip links |
| Header | [header.md](header.md) | Fixed header with main navigation |
| Sidebar | [sidebar.md](sidebar.md) | Documentation sidebar navigation |
| Footer | [footer.md](footer.md) | Site footer with links and social |
| Theme Toggle | [theme-toggle.md](theme-toggle.md) | Dark/light/system theme switcher |
| Mobile Menu | [mobile-menu.md](mobile-menu.md) | Full-screen mobile navigation overlay |
| Navigation Structure | [navigation-structure.md](navigation-structure.md) | Information architecture and URLs |
| Code Block | [code-block.md](code-block.md) | Syntax-highlighted code with copy button |
| Terminal | [terminal.md](terminal.md) | Terminal output with optional typing animation |
| Copy Button | [copy-button.md](copy-button.md) | Clipboard copy with feedback |
| Language Indicator | [language-indicator.md](language-indicator.md) | Programming language badge |
| Syntax Highlighting | [syntax-highlighting-tokens.md](syntax-highlighting-tokens.md) | Color tokens for all languages |
| Screenshot | [screenshot.md](screenshot.md) | Auto-generated screenshot display with theme variants |
| Lightbox | [lightbox.md](lightbox.md) | Full-screen image viewer modal |

### Page Specifications

| Page | File | Description |
|------|------|-------------|
| Landing Page | [landing-page.md](landing-page.md) | Homepage with hero, features, demo, install sections |
| Quick Start | [quick-start-page.md](quick-start-page.md) | 4-step onboarding with "Fix the Bug" AI challenge |
| Guided Tour | [guided-tour.md](guided-tour.md) | 8-step progressive tutorial with localStorage persistence |

### Verification

| Document | Description |
|----------|-------------|
| [accessibility-verification.md](accessibility-verification.md) | WCAG compliance matrix and testing |

---

## Quick Reference

### Navigation Items

```
Home | Quick Start | MCP Server ✨ | Guided Tour | Reference
```

### Breakpoints

| Name | Width | Layout Changes |
|------|-------|----------------|
| Mobile | <768px | Hamburger menu, stacked footer |
| Tablet | 768-1023px | Hamburger menu, 2-col footer |
| Desktop | ≥1024px | Full nav, sidebar visible |

### Key Tokens

```css
--header-height: 64px;
--sidebar-width: 256px;
--content-max-width: 1280px;
--color-azure-600: #2563eb;

/* Syntax Highlighting */
--syntax-keyword: #f472b6;
--syntax-string: #a5f3fc;
--syntax-function: #a78bfa;
--syntax-variable: #93c5fd;
--syntax-comment: #64748b;
```

### MCP Server Prominence

The MCP Server navigation item receives special treatment:
- Badge with "AI" label
- Gradient/glow border
- Star/sparkles icon
- Enhanced hover state
- `aria-describedby` for badge announcement

---

## Implementation Order

Recommended order for implementing components:

1. **Design System** - Set up CSS custom properties
2. **Layout** - Establish page structure
3. **Header** - Main navigation (desktop first)
4. **Footer** - Static content
5. **Theme Toggle** - Persistence logic
6. **Mobile Menu** - Mobile navigation
7. **Sidebar** - Documentation pages
8. **Copy Button** - Reusable atom for code components
9. **Language Indicator** - Language badge atom
10. **Code Block** - Syntax highlighting with copy
11. **Terminal** - Command output with animation
12. **Screenshot** - Themed screenshot display
13. **Lightbox** - Full-screen image viewer

---

## Component Dependencies

```
Layout
├── Header
│   ├── ThemeToggle
│   └── MobileMenuButton
├── Sidebar (optional)
├── MainContent
│   ├── CodeBlock
│   │   ├── LanguageIndicator
│   │   └── CopyButton
│   ├── Terminal
│   │   └── CopyButton
│   ├── Screenshot
│   │   └── Lightbox (portal)
│   └── TourLayout (for /tour pages)
│       ├── TourProgressSidebar
│       │   ├── TourStepItem
│       │   └── TourProgress
│       ├── TourStepPage
│       │   ├── TourStepHeader
│       │   ├── CompletionCheckbox
│       │   ├── TryItYourself
│       │   ├── LearnMoreSection
│       │   └── TourNavigation
│       └── TourCompletionPage
├── Footer
│   └── ThemeToggle (optional)
└── MobileMenu (portal)
    └── ThemeToggle
```

---

## Accessibility Highlights

### Required for All Components

- Visible focus indicators (2px outline, 3:1 contrast)
- Semantic HTML elements
- ARIA attributes where needed
- Keyboard navigation support
- Screen reader announcements

### Critical Elements

| Element | Requirement |
|---------|-------------|
| Skip links | First focusable element |
| Focus trap | Mobile menu and overlays |
| Live regions | Theme changes, menu state |
| `aria-current` | Active navigation items |
| `aria-expanded` | Collapsible sections |

---

## Responsive Summary

### Mobile (<768px)
- Hamburger menu for navigation
- Sidebar hidden (accessible via mobile menu)
- Single-column footer with accordion sections
- Touch targets ≥44x44px

### Tablet (768-1023px)
- Hamburger menu continues
- Sidebar collapsed to icons (64px)
- Two-column footer grid

### Desktop (≥1024px)
- Full horizontal navigation
- Sidebar always visible (256px)
- Four-column footer grid

---

## Testing Checklist

Before component completion, verify:

- [ ] Passes axe-core with 0 violations
- [ ] Keyboard navigation works
- [ ] Screen reader tested
- [ ] Color contrast verified
- [ ] Touch targets adequate
- [ ] Reduced motion respected
- [ ] Works without JavaScript
- [ ] Responsive at all breakpoints

### Code Component Specific

- [ ] Syntax highlighting colors contrast ≥4.5:1
- [ ] Copy button announces status to screen readers
- [ ] Terminal animation respects reduced motion
- [ ] All supported languages render correctly
- [ ] Line highlighting visible in both themes

### Screenshot & Lightbox Specific

- [ ] Light/dark theme variants load correctly
- [ ] Theme switch updates displayed image
- [ ] Lightbox opens on click/Enter/Space
- [ ] Lightbox closes on Escape
- [ ] Lightbox closes on backdrop click
- [ ] Focus trapped in lightbox
- [ ] Focus returns to trigger on close
- [ ] Alt text is meaningful and announced
- [ ] Caption associated with image

### Guided Tour Specific

- [ ] Progress saves to localStorage on step completion
- [ ] Progress restores on page reload and new tabs
- [ ] Reset progress clears all data
- [ ] Previous/Next navigation works correctly
- [ ] Step indicator dots update correctly
- [ ] Learn More sections expand/collapse
- [ ] Completion checkbox toggles state and saves
- [ ] Mobile sidebar opens/closes correctly
- [ ] Resume prompt appears for returning users
- [ ] All 8 step pages render correctly
- [ ] Completion page shows all achievements
