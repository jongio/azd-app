# Footer Component Specification

## Overview

The Footer component provides site-wide footer content including navigation links, social media links, newsletter signup (optional), and copyright information. It adapts from a multi-column layout on desktop to a stacked layout on mobile.

---

## 1. Component Hierarchy

```
Footer (organism)
├── FooterContainer (molecule)
│   ├── FooterBrand (molecule)
│   │   ├── Logo (atom)
│   │   └── Tagline (atom)
│   ├── FooterNav (molecule)
│   │   ├── FooterNavSection (molecule) × 3-4
│   │   │   ├── FooterNavTitle (atom)
│   │   │   └── FooterNavLink (atom) × n
│   ├── FooterSocial (molecule)
│   │   └── SocialIcon (atom) × n
│   └── FooterBottom (molecule)
│       ├── Copyright (atom)
│       ├── LegalLinks (molecule)
│       └── ThemeToggle (atom) [optional duplicate]
```

---

## 2. Props Interface

### Footer

```typescript
interface FooterProps {
  /** Footer navigation sections */
  navigation: FooterNavigation[];
  /** Social media links */
  socialLinks: SocialLink[];
  /** Copyright text */
  copyright: string;
  /** Legal/policy links */
  legalLinks: LegalLink[];
  /** Show newsletter signup */
  showNewsletter?: boolean;
  /** Custom class name */
  className?: string;
}

interface FooterNavigation {
  /** Section title */
  title: string;
  /** Section links */
  links: {
    label: string;
    href: string;
    external?: boolean;
    badge?: string;
  }[];
}

interface SocialLink {
  /** Platform name */
  platform: 'github' | 'twitter' | 'discord' | 'linkedin' | 'youtube';
  /** Profile URL */
  href: string;
  /** Accessible label */
  label: string;
}

interface LegalLink {
  /** Link text */
  label: string;
  /** Link destination */
  href: string;
}
```

### FooterNavSection

```typescript
interface FooterNavSectionProps {
  /** Section title */
  title: string;
  /** Section links */
  links: {
    label: string;
    href: string;
    external?: boolean;
    badge?: string;
  }[];
}
```

### SocialIcon

```typescript
interface SocialIconProps {
  /** Platform identifier */
  platform: 'github' | 'twitter' | 'discord' | 'linkedin' | 'youtube';
  /** Link URL */
  href: string;
  /** Accessible label */
  label: string;
  /** Icon size */
  size?: 'sm' | 'md' | 'lg';
}
```

---

## 3. States

### FooterNavLink States

| State    | Trigger          | Visual Changes                        |
|----------|------------------|---------------------------------------|
| Default  | Initial render   | Normal text color                     |
| Hover    | Mouse enter      | Underline, slightly brighter color    |
| Focus    | Keyboard focus   | Focus ring visible                    |
| Active   | Mouse down       | Slightly dimmed                       |

### SocialIcon States

| State    | Trigger          | Visual Changes                        |
|----------|------------------|---------------------------------------|
| Default  | Initial render   | Muted icon color                      |
| Hover    | Mouse enter      | Brand color for platform, scale up    |
| Focus    | Keyboard focus   | Focus ring, brand color               |
| Active   | Mouse down       | Scale down slightly                   |

---

## 4. Visual Specifications

### Layout

```
Desktop (≥1024px):
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│  [Logo]                Resources    Documentation    Community      │
│  azd-app               ─────────    ─────────────    ─────────      │
│  Local dev made        Quick Start  CLI Reference    GitHub         │
│  simple                Installation MCP Server       Discord        │
│                        Configuration Guided Tour     Twitter        │
│                        Changelog     FAQ             YouTube        │
│                                                                     │
│  [GitHub] [Discord] [Twitter]                                       │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│  © 2024 azd-app        Privacy  Terms  Cookies              🌙      │
└─────────────────────────────────────────────────────────────────────┘

Mobile (<768px):
┌─────────────────────────────────┐
│                                 │
│           [Logo]                │
│           azd-app               │
│    Local dev made simple        │
│                                 │
│        Resources ▼              │
│     Documentation ▼             │
│       Community ▼               │
│                                 │
│  [GitHub] [Discord] [Twitter]   │
│                                 │
├─────────────────────────────────┤
│     © 2024 azd-app              │
│  Privacy · Terms · Cookies      │
└─────────────────────────────────┘
```

### Dimensions

| Property              | Desktop    | Tablet     | Mobile     |
|-----------------------|------------|------------|------------|
| Padding top/bottom    | 48px       | 40px       | 32px       |
| Container max-width   | 1280px     | 100%       | 100%       |
| Container padding     | 0 24px     | 0 16px     | 0 16px     |
| Column gap            | 64px       | 32px       | 0          |
| Section gap           | 32px       | 24px       | 16px       |
| Link gap              | 12px       | 12px       | 12px       |
| Social icon size      | 24px       | 24px       | 28px       |
| Social icon gap       | 16px       | 16px       | 20px       |
| Bottom bar height     | 56px       | 56px       | auto       |
| Bottom bar padding    | 16px 0     | 16px 0     | 16px       |

### Typography

| Element          | Font Size | Weight    | Line Height |
|------------------|-----------|-----------|-------------|
| Logo text        | 20px      | Bold      | 1.25        |
| Tagline          | 14px      | Normal    | 1.5         |
| Section title    | 14px      | Semibold  | 1.25        |
| Link             | 14px      | Normal    | 1.5         |
| Copyright        | 14px      | Normal    | 1.5         |
| Legal link       | 12px      | Normal    | 1.5         |

### Colors

#### Light Theme

| Element                | Property    | Value               |
|------------------------|-------------|---------------------|
| Footer bg              | background  | #f9fafb             |
| Section title          | color       | #111827             |
| Link text              | color       | #6b7280             |
| Link hover             | color       | #111827             |
| Social icon            | color       | #9ca3af             |
| Social icon hover      | color       | platform specific   |
| Border (top, divider)  | border      | 1px solid #e5e7eb   |
| Copyright              | color       | #6b7280             |
| Legal link             | color       | #9ca3af             |

#### Dark Theme

| Element                | Property    | Value               |
|------------------------|-------------|---------------------|
| Footer bg              | background  | #0f172a             |
| Section title          | color       | #f1f5f9             |
| Link text              | color       | #94a3b8             |
| Link hover             | color       | #f1f5f9             |
| Social icon            | color       | #64748b             |
| Social icon hover      | color       | platform specific   |
| Border (top, divider)  | border      | 1px solid #334155   |
| Copyright              | color       | #94a3b8             |
| Legal link             | color       | #64748b             |

#### Platform Brand Colors

| Platform  | Color    |
|-----------|----------|
| GitHub    | #333     |
| Twitter   | #1DA1F2  |
| Discord   | #5865F2  |
| LinkedIn  | #0A66C2  |
| YouTube   | #FF0000  |

---

## 5. Interactions

### Links

| Action          | Behavior                                      |
|-----------------|-----------------------------------------------|
| Click link      | Navigate to destination                       |
| Click external  | Open in new tab with `rel="noopener"`         |
| Hover link      | Show underline, brighten color                |

### Social Icons

| Action          | Behavior                                      |
|-----------------|-----------------------------------------------|
| Click icon      | Open social profile in new tab                |
| Hover icon      | Show platform brand color, scale to 1.1       |

### Mobile Sections (Accordion)

| Action               | Behavior                                   |
|----------------------|-------------------------------------------|
| Click section title  | Toggle section expanded/collapsed          |
| Focus section        | Show focus ring                            |

### Keyboard

| Key         | Behavior                                        |
|-------------|------------------------------------------------|
| Tab         | Move focus to next link/icon                    |
| Shift+Tab   | Move focus to previous link/icon                |
| Enter       | Activate focused link/icon                      |
| Space       | Toggle accordion section (mobile)               |

---

## 6. Accessibility

### ARIA Attributes

```html
<footer role="contentinfo" aria-label="Site footer">
  <div class="footer-brand">
    <a href="/" aria-label="azd-app home">
      <img src="/logo.svg" alt="" />
      <span>azd-app</span>
    </a>
    <p>Local dev made simple</p>
  </div>
  
  <nav aria-label="Footer navigation">
    <div role="group" aria-labelledby="footer-resources">
      <h2 id="footer-resources">Resources</h2>
      <ul role="list">
        <li>
          <a href="/quick-start">Quick Start</a>
        </li>
        <li>
          <a href="/changelog">
            Changelog
            <span class="badge" aria-label="New">New</span>
          </a>
        </li>
      </ul>
    </div>
    
    <!-- Additional sections -->
  </nav>
  
  <nav aria-label="Social media links">
    <ul role="list">
      <li>
        <a 
          href="https://github.com/jongio/azd-app"
          aria-label="azd-app on GitHub"
          rel="noopener noreferrer"
          target="_blank"
        >
          <svg aria-hidden="true"><!-- github icon --></svg>
        </a>
      </li>
      <!-- Additional social links -->
    </ul>
  </nav>
  
  <div class="footer-bottom">
    <p>© 2024 azd-app. All rights reserved.</p>
    <nav aria-label="Legal links">
      <ul role="list">
        <li><a href="/privacy">Privacy</a></li>
        <li><a href="/terms">Terms</a></li>
        <li><a href="/cookies">Cookies</a></li>
      </ul>
    </nav>
  </div>
</footer>
```

### Semantic HTML

- `<footer>` with `role="contentinfo"`
- Multiple `<nav>` elements with unique `aria-label`s
- `<h2>` for section titles (in document outline)
- `<ul>` / `<li>` for link lists
- External links with `rel="noopener noreferrer"`

### Focus Management

- All links/icons keyboard focusable
- Visible focus indicators
- Logical tab order (columns, then bottom bar)

### Screen Reader Announcements

- "Footer navigation, Resources section"
- "azd-app on GitHub, external link"
- "Changelog, New"

---

## 7. Responsive Behavior

### Breakpoints

| Breakpoint | Layout                                          |
|------------|------------------------------------------------|
| ≥1024px    | 4-column grid (brand + 3 nav sections)         |
| 768-1023px | 2-column grid, 2 sections per row              |
| <768px     | Stacked, accordion sections                     |

### Grid Configuration

```css
/* Desktop */
@media (min-width: 1024px) {
  .footer-content {
    display: grid;
    grid-template-columns: 1.5fr 1fr 1fr 1fr;
    gap: 64px;
  }
}

/* Tablet */
@media (min-width: 768px) and (max-width: 1023px) {
  .footer-content {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 32px;
  }
  
  .footer-brand {
    grid-column: 1 / -1;
    margin-bottom: 16px;
  }
}

/* Mobile */
@media (max-width: 767px) {
  .footer-content {
    display: flex;
    flex-direction: column;
    gap: 0;
  }
}
```

### Mobile Accordion

```html
<div class="footer-section" data-expanded="false">
  <button 
    class="footer-section__header"
    aria-expanded="false"
    aria-controls="section-resources"
  >
    <span>Resources</span>
    <svg aria-hidden="true" class="chevron"><!-- icon --></svg>
  </button>
  <div id="section-resources" class="footer-section__content">
    <!-- links -->
  </div>
</div>
```

---

## 8. Animation Specifications

### Link Hover

```css
.footer-link {
  transition: color 100ms ease-out;
  text-decoration: none;
}

.footer-link:hover {
  color: var(--color-text-primary);
  text-decoration: underline;
  text-underline-offset: 2px;
}
```

### Social Icon Hover

```css
.social-icon {
  transition: color 100ms ease-out, transform 100ms ease-out;
}

.social-icon:hover {
  transform: scale(1.1);
}

.social-icon--github:hover {
  color: #333;
}

.social-icon--twitter:hover {
  color: #1DA1F2;
}

/* etc. */
```

### Mobile Accordion

```css
.footer-section__content {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows 200ms ease-out;
}

.footer-section[data-expanded="true"] .footer-section__content {
  grid-template-rows: 1fr;
}

.footer-section__content-inner {
  overflow: hidden;
}

.footer-section .chevron {
  transition: transform 200ms ease-out;
}

.footer-section[data-expanded="true"] .chevron {
  transform: rotate(180deg);
}
```

---

## 9. External Link Handling

### Visual Indicator

External links should have a visual indicator:

```html
<a href="https://github.com/..." target="_blank" rel="noopener noreferrer">
  GitHub
  <svg 
    class="external-icon" 
    aria-hidden="true"
    width="12" 
    height="12"
  >
    <!-- external link icon -->
  </svg>
  <span class="sr-only">(opens in new tab)</span>
</a>
```

### Styling

```css
.external-icon {
  display: inline-block;
  margin-left: 4px;
  opacity: 0.5;
  vertical-align: middle;
}

.footer-link:hover .external-icon {
  opacity: 1;
}
```

---

## 10. Content Structure

### Recommended Navigation Sections

```typescript
const footerNavigation: FooterNavigation[] = [
  {
    title: 'Resources',
    links: [
      { label: 'Quick Start', href: '/quick-start' },
      { label: 'Installation', href: '/installation' },
      { label: 'Configuration', href: '/configuration' },
      { label: 'Changelog', href: '/changelog', badge: 'New' },
    ],
  },
  {
    title: 'Documentation',
    links: [
      { label: 'CLI Reference', href: '/reference' },
      { label: 'MCP Server', href: '/mcp-server', badge: 'AI' },
      { label: 'Guided Tour', href: '/tour' },
      { label: 'FAQ', href: '/faq' },
    ],
  },
  {
    title: 'Community',
    links: [
      { label: 'GitHub', href: 'https://github.com/jongio/azd-app', external: true },
      { label: 'Discord', href: 'https://discord.gg/...', external: true },
      { label: 'Twitter', href: 'https://twitter.com/...', external: true },
      { label: 'Contributing', href: '/contributing' },
    ],
  },
];
```

---

## 11. Error States

| Scenario                  | Behavior                                     |
|---------------------------|----------------------------------------------|
| Missing social links      | Hide social section entirely                 |
| Invalid URL               | Link still renders, error handled at runtime |
| JavaScript disabled       | Accordion sections expanded, all links work  |

---

## 12. Implementation Notes

### Performance

- Lazy load social icons (SVG sprites or icon font)
- Use semantic HTML for SEO
- Minimal JavaScript (accordion only on mobile)

### SSR Considerations

- Render all content server-side
- Accordion state can be CSS-only with `:target`
- No hydration needed for static content

### Testing Checklist

- [ ] All links keyboard accessible
- [ ] External links open in new tab
- [ ] Focus visible on all interactive elements
- [ ] Screen reader announces link destinations
- [ ] Mobile accordion works with keyboard
- [ ] Touch targets ≥ 44x44px on mobile
- [ ] Color contrast ≥ 4.5:1
- [ ] Works with JavaScript disabled
- [ ] Social icons have accessible labels
- [ ] Reduced motion respected
