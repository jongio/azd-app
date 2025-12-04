````markdown
# Screenshot Component Specification

## Overview

The Screenshot component displays auto-generated screenshots from Playwright tests, with support for dark/light theme variants. It includes click-to-expand functionality that opens a lightbox for full-size viewing.

---

## 1. Component Hierarchy

```
Screenshot (organism)
├── ScreenshotContainer (molecule)
│   ├── ScreenshotImage (atom)
│   │   └── <img> (themed variant)
│   ├── ScreenshotOverlay (atom)
│   │   ├── ExpandIcon (atom)
│   │   └── "View full size" label
│   └── LoadingPlaceholder (atom) [while loading]
├── ScreenshotCaption (atom) [optional]
└── Lightbox (molecule) [portal, when open]
    ├── LightboxBackdrop (atom)
    ├── LightboxContent (molecule)
    │   ├── LightboxImage (atom)
    │   └── LightboxCaption (atom)
    └── LightboxControls (molecule)
        └── CloseButton (atom)
```

---

## 2. Props Interface

### Screenshot

```typescript
interface ScreenshotProps {
  /** Light theme screenshot path (relative to /public/screenshots/) */
  lightSrc: string;
  /** Dark theme screenshot path (relative to /public/screenshots/) */
  darkSrc: string;
  /** Alt text for accessibility (required) */
  alt: string;
  /** Optional caption displayed below the screenshot */
  caption?: string;
  /** Width of the image (CSS value or number for pixels) */
  width?: string | number;
  /** Height of the image (CSS value or number for pixels) */
  height?: string | number;
  /** Aspect ratio to maintain (e.g., "16/9") */
  aspectRatio?: string;
  /** Disable lightbox functionality */
  disableLightbox?: boolean;
  /** Custom class name */
  className?: string;
  /** Priority loading for above-the-fold images */
  priority?: boolean;
  /** Callback when lightbox opens */
  onLightboxOpen?: () => void;
  /** Callback when lightbox closes */
  onLightboxClose?: () => void;
}
```

### ScreenshotImage (internal)

```typescript
interface ScreenshotImageProps {
  /** Current theme ('light' | 'dark') */
  theme: 'light' | 'dark';
  /** Light theme image source */
  lightSrc: string;
  /** Dark theme image source */
  darkSrc: string;
  /** Alt text */
  alt: string;
  /** Loading state callback */
  onLoad?: () => void;
  /** Error state callback */
  onError?: (error: Error) => void;
}
```

### Lightbox

```typescript
interface LightboxProps {
  /** Whether lightbox is open */
  isOpen: boolean;
  /** Image source to display */
  src: string;
  /** Alt text for the image */
  alt: string;
  /** Optional caption */
  caption?: string;
  /** Close handler */
  onClose: () => void;
}
```

---

## 3. States

### Screenshot States

| State     | Trigger              | Visual Changes                              |
|-----------|---------------------|---------------------------------------------|
| Loading   | Initial render       | Skeleton placeholder with shimmer animation |
| Loaded    | Image loaded         | Image visible, overlay ready on hover       |
| Error     | Image load failed    | Error placeholder with retry option         |
| Hover     | Mouse enter          | Overlay visible with expand icon            |
| Focus     | Keyboard focus       | Focus ring, overlay visible                 |
| Active    | Click/tap            | Opens lightbox                              |

### Lightbox States

| State     | Trigger              | Visual Changes                              |
|-----------|---------------------|---------------------------------------------|
| Opening   | Click screenshot     | Backdrop fade in, image scale/fade in       |
| Open      | Animation complete   | Full-size image, close button visible       |
| Closing   | Escape/click outside | Backdrop fade out, image scale/fade out     |
| Closed    | Animation complete   | Removed from DOM                            |

### Theme Sync States

| State          | Trigger                    | Behavior                          |
|----------------|---------------------------|-----------------------------------|
| Light theme    | data-theme="light"        | Display lightSrc image            |
| Dark theme     | data-theme="dark"         | Display darkSrc image             |
| Theme change   | User toggles theme        | Crossfade to new theme variant    |

---

## 4. Visual Specifications

### Layout

```
Inline Screenshot:
┌─────────────────────────────────────────────────────┐
│                                                     │
│                  ┌─────────────────┐                │
│                  │   Screenshot    │                │
│                  │     Image       │ ← Hover shows  │
│                  │                 │   overlay      │
│                  │      🔍 View    │                │
│                  └─────────────────┘                │
│                                                     │
│           Caption: Dashboard overview               │
└─────────────────────────────────────────────────────┘

Hover State:
┌─────────────────────────────────────────────────────┐
│  ┌───────────────────────────────────────────────┐  │
│  │░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│  │
│  │░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│  │
│  │░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│  │
│  │░░░░░░░░░░░░ 🔍 View full size ░░░░░░░░░░░░░░░│  │
│  │░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│  │
│  │░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│  │
│  └───────────────────────────────────────────────┘  │
│                  Caption text here                   │
└─────────────────────────────────────────────────────┘

Loading State:
┌─────────────────────────────────────────────────────┐
│  ┌───────────────────────────────────────────────┐  │
│  │▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒│  │
│  │▒▒▒▒▒▒▒▒▒▒▒▒▒ Shimmer ▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒│  │
│  │▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒│  │
│  └───────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘

Error State:
┌─────────────────────────────────────────────────────┐
│  ┌───────────────────────────────────────────────┐  │
│  │                                               │  │
│  │          ⚠️  Failed to load image             │  │
│  │             [ Retry ]                         │  │
│  │                                               │  │
│  └───────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

### Dimensions

| Property                 | Value                         |
|--------------------------|-------------------------------|
| Container border-radius  | 8px (--radius-lg)             |
| Container border         | 1px solid var(--border-color) |
| Image max-width          | 100%                          |
| Default aspect-ratio     | auto (native)                 |
| Caption padding          | 12px 0                        |
| Caption font-size        | 14px (--font-size-sm)         |
| Overlay padding          | 16px                          |
| Expand icon size         | 24px                          |
| Shadow (hover)           | 0 4px 12px rgba(0,0,0,0.15)   |

### Colors

#### Light Theme

| Element                | Property    | Value                      |
|------------------------|-------------|----------------------------|
| Container border       | border      | 1px solid #e5e7eb          |
| Container bg           | background  | #f9fafb                    |
| Overlay bg             | background  | rgba(0, 0, 0, 0.6)         |
| Overlay text           | color       | white                      |
| Caption text           | color       | #6b7280                    |
| Loading skeleton       | background  | #e5e7eb                    |
| Loading shimmer        | background  | linear-gradient #f3f4f6    |
| Error bg               | background  | #fef2f2                    |
| Error text             | color       | #dc2626                    |
| Focus ring             | box-shadow  | 0 0 0 3px #93c5fd          |

#### Dark Theme

| Element                | Property    | Value                      |
|------------------------|-------------|----------------------------|
| Container border       | border      | 1px solid #334155          |
| Container bg           | background  | #1e293b                    |
| Overlay bg             | background  | rgba(0, 0, 0, 0.7)         |
| Overlay text           | color       | white                      |
| Caption text           | color       | #94a3b8                    |
| Loading skeleton       | background  | #334155                    |
| Loading shimmer        | background  | linear-gradient #475569    |
| Error bg               | background  | #450a0a                    |
| Error text             | color       | #f87171                    |
| Focus ring             | box-shadow  | 0 0 0 3px #1e40af          |

---

## 5. Interactions

### Click/Tap Behavior

| Element       | Action              | Result                              |
|---------------|---------------------|-------------------------------------|
| Screenshot    | Click/tap           | Open lightbox (if not disabled)     |
| Screenshot    | Right-click         | Browser context menu                |
| Overlay       | Click               | Open lightbox                       |
| Caption       | Click               | No action (not clickable)           |

### Hover Behavior

| Element       | Hover Effect                                    |
|---------------|------------------------------------------------|
| Container     | Shadow increases, cursor becomes pointer        |
| Image         | Slight scale (1.02)                            |
| Overlay       | Fades in with expand icon and text             |

### Keyboard Navigation

| Key           | Action                                         |
|---------------|------------------------------------------------|
| Tab           | Focus the screenshot container                 |
| Enter         | Open lightbox                                  |
| Space         | Open lightbox                                  |

---

## 6. Theme Switching

### Theme Detection

```typescript
// Get current theme
function getCurrentTheme(): 'light' | 'dark' {
  return document.documentElement.getAttribute('data-theme') as 'light' | 'dark' 
    || 'light';
}

// Listen for theme changes
function observeThemeChanges(callback: (theme: 'light' | 'dark') => void) {
  const observer = new MutationObserver((mutations) => {
    for (const mutation of mutations) {
      if (mutation.attributeName === 'data-theme') {
        const theme = getCurrentTheme();
        callback(theme);
      }
    }
  });
  
  observer.observe(document.documentElement, { attributes: true });
  return () => observer.disconnect();
}
```

### Image Switching

```typescript
// Select appropriate image source
function getThemedSrc(theme: 'light' | 'dark', lightSrc: string, darkSrc: string): string {
  return theme === 'dark' ? darkSrc : lightSrc;
}

// Full path resolution
function resolveScreenshotPath(src: string): string {
  return src.startsWith('/') ? src : `/screenshots/${src}`;
}
```

### Crossfade Animation

```css
.screenshot-image {
  transition: opacity 0.2s ease-out;
}

.screenshot-image--switching {
  opacity: 0;
}

.screenshot-image--visible {
  opacity: 1;
}
```

---

## 7. Accessibility

### Semantic HTML Structure

```html
<figure class="screenshot" role="figure">
  <button
    type="button"
    class="screenshot-button"
    aria-label="View full size: Dashboard overview screenshot"
    aria-haspopup="dialog"
  >
    <picture class="screenshot-picture">
      <!-- Light theme image (preloaded, hidden in dark mode) -->
      <source 
        srcset="/screenshots/dashboard-light.png" 
        media="(prefers-color-scheme: light)"
      />
      <!-- Dark theme image -->
      <source 
        srcset="/screenshots/dashboard-dark.png" 
        media="(prefers-color-scheme: dark)"
      />
      <img
        src="/screenshots/dashboard-light.png"
        alt="Dashboard showing service status and health metrics"
        loading="lazy"
        decoding="async"
        class="screenshot-image"
      />
    </picture>
    
    <div class="screenshot-overlay" aria-hidden="true">
      <svg class="expand-icon" aria-hidden="true"><!-- expand --></svg>
      <span>View full size</span>
    </div>
  </button>
  
  <figcaption class="screenshot-caption">
    Dashboard overview showing running services
  </figcaption>
</figure>
```

### ARIA Attributes

| Element          | Attribute                    | Purpose                          |
|------------------|------------------------------|----------------------------------|
| Container        | role="figure"                | Semantic grouping                |
| Button           | aria-label                   | Describe action and content      |
| Button           | aria-haspopup="dialog"       | Indicates lightbox will open     |
| Image            | alt                          | Required descriptive text        |
| Overlay          | aria-hidden="true"           | Decorative, hide from SR         |
| Caption          | <figcaption>                 | Associated with figure           |

### Screen Reader Announcements

```typescript
// On focus
"View full size: Dashboard overview screenshot, button"

// On activation
"Opening image viewer"

// Alt text read when image loads
"Dashboard showing service status and health metrics"
```

### Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  .screenshot-image {
    transition: none;
  }
  
  .screenshot-overlay {
    transition: opacity 0.1s ease-out;
  }
  
  .screenshot:hover .screenshot-image {
    transform: none;
  }
}
```

---

## 8. Responsive Design

### Breakpoint Behavior

| Breakpoint        | Changes                                     |
|-------------------|---------------------------------------------|
| Mobile (<640px)   | Full-width, caption below                   |
| Tablet (640-1023px)| Inline with text wrap, hover overlay       |
| Desktop (≥1024px) | Standard layout with hover effects         |

### Mobile Considerations

```css
@media (max-width: 640px) {
  .screenshot {
    /* Bleed to edges on mobile */
    margin-left: calc(-1 * var(--spacing-4));
    margin-right: calc(-1 * var(--spacing-4));
    border-radius: 0;
    border-left: none;
    border-right: none;
  }
  
  .screenshot-button {
    width: 100%;
  }
  
  /* Always show overlay hint on mobile (touch devices) */
  .screenshot-overlay {
    opacity: 0.8;
    background: linear-gradient(
      to top,
      rgba(0, 0, 0, 0.6) 0%,
      transparent 50%
    );
  }
  
  .screenshot-overlay span {
    position: absolute;
    bottom: 12px;
    right: 12px;
    font-size: 12px;
  }
  
  .screenshot-caption {
    padding-left: var(--spacing-4);
    padding-right: var(--spacing-4);
  }
}
```

### Touch Targets

- Screenshot container: Touch target is entire image
- Minimum touch area: 44x44px (guaranteed by image size)

---

## 9. Animation Specifications

### Loading Skeleton

```css
.screenshot-skeleton {
  background: var(--skeleton-bg);
  animation: skeleton-shimmer 1.5s ease-in-out infinite;
}

@keyframes skeleton-shimmer {
  0% {
    background-position: -200% 0;
  }
  100% {
    background-position: 200% 0;
  }
}

.screenshot-skeleton {
  background: linear-gradient(
    90deg,
    var(--skeleton-bg) 0%,
    var(--skeleton-shimmer) 50%,
    var(--skeleton-bg) 100%
  );
  background-size: 200% 100%;
}
```

### Hover Effects

```css
.screenshot-button {
  transition: box-shadow 0.2s ease-out;
}

.screenshot-button:hover {
  box-shadow: var(--shadow-lg);
}

.screenshot-image {
  transition: transform 0.2s ease-out;
}

.screenshot-button:hover .screenshot-image {
  transform: scale(1.02);
}

.screenshot-overlay {
  opacity: 0;
  transition: opacity 0.2s ease-out;
}

.screenshot-button:hover .screenshot-overlay,
.screenshot-button:focus .screenshot-overlay {
  opacity: 1;
}
```

### Image Load Animation

```css
.screenshot-image {
  opacity: 0;
  transition: opacity 0.3s ease-out;
}

.screenshot-image--loaded {
  opacity: 1;
}
```

---

## 10. File Path Conventions

### Screenshot Storage

```
web/public/screenshots/
├── dashboard-light.png
├── dashboard-dark.png
├── terminal-light.png
├── terminal-dark.png
├── mcp-server-light.png
├── mcp-server-dark.png
└── ...
```

### Naming Convention

```
{feature}-{variant}.{extension}

feature: Descriptive name (dashboard, terminal, settings)
variant: light | dark
extension: png | jpg | webp

Examples:
- dashboard-overview-light.png
- dashboard-overview-dark.png
- service-logs-light.png
- service-logs-dark.png
```

### Component Usage

```tsx
// In Astro/MDX
<Screenshot
  lightSrc="dashboard-overview-light.png"
  darkSrc="dashboard-overview-dark.png"
  alt="Dashboard showing running services and health status"
  caption="The azd app dashboard provides real-time service monitoring"
/>
```

---

## 11. Error Handling

| Scenario               | Behavior                                     |
|------------------------|---------------------------------------------|
| Image fails to load    | Show error state with retry button          |
| Missing dark variant   | Fall back to light variant                  |
| Missing light variant  | Fall back to dark variant                   |
| Both variants missing  | Show error placeholder                      |
| Slow network           | Continue showing skeleton, no timeout       |

### Error Recovery

```typescript
interface ImageLoadState {
  status: 'loading' | 'loaded' | 'error';
  error?: Error;
  retryCount: number;
}

const MAX_RETRIES = 2;

async function loadImage(src: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve();
    img.onerror = () => reject(new Error('Failed to load image'));
    img.src = src;
  });
}

function handleRetry(): void {
  if (state.retryCount < MAX_RETRIES) {
    setState({ status: 'loading', retryCount: state.retryCount + 1 });
    loadImage(currentSrc);
  }
}
```

---

## 12. Performance Optimization

### Lazy Loading

```tsx
<img
  src={resolvedSrc}
  alt={alt}
  loading={priority ? "eager" : "lazy"}
  decoding="async"
/>
```

### Preloading Critical Images

```html
<!-- In <head> for above-the-fold screenshots -->
<link 
  rel="preload" 
  as="image" 
  href="/screenshots/hero-light.png"
  media="(prefers-color-scheme: light)"
/>
<link 
  rel="preload" 
  as="image" 
  href="/screenshots/hero-dark.png"
  media="(prefers-color-scheme: dark)"
/>
```

### Image Optimization

- Use WebP format when supported
- Provide multiple sizes for responsive images
- Compress images during build

```tsx
// With responsive images
<picture>
  <source 
    type="image/webp"
    srcset="/screenshots/dashboard-light.webp"
    media="(prefers-color-scheme: light)"
  />
  <source 
    type="image/webp"
    srcset="/screenshots/dashboard-dark.webp"
    media="(prefers-color-scheme: dark)"
  />
  <img src="/screenshots/dashboard-light.png" alt="..." />
</picture>
```

---

## 13. CSS Custom Properties

```css
/* Screenshot Component Tokens */
--screenshot-border-radius: 0.5rem;
--screenshot-border-color: var(--border-color);
--screenshot-bg: var(--color-bg-secondary);
--screenshot-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
--screenshot-shadow-hover: 0 4px 12px rgba(0, 0, 0, 0.15);
--screenshot-overlay-bg: rgba(0, 0, 0, 0.6);
--screenshot-overlay-text: white;
--screenshot-caption-color: var(--color-text-secondary);
--screenshot-skeleton-bg: var(--color-bg-tertiary);
--screenshot-skeleton-shimmer: var(--color-bg-secondary);
--screenshot-focus-ring: 0 0 0 3px var(--color-focus);
--screenshot-transition: 0.2s ease-out;
```

---

## 14. Testing Checklist

### Functionality

- [ ] Light theme screenshot loads correctly
- [ ] Dark theme screenshot loads correctly
- [ ] Theme switching updates screenshot
- [ ] Click opens lightbox
- [ ] Hover shows overlay
- [ ] Loading state displays skeleton
- [ ] Error state shows retry option
- [ ] Retry button works

### Accessibility

- [ ] Meaningful alt text provided
- [ ] Keyboard navigation works (Tab, Enter, Space)
- [ ] Focus indicator visible
- [ ] Screen reader announces actions
- [ ] Caption associated with image
- [ ] Reduced motion preference respected

### Responsive

- [ ] Full-width on mobile
- [ ] Overlay visible on touch devices
- [ ] Touch targets adequate (≥44x44px)
- [ ] Caption readable on all sizes

### Performance

- [ ] Lazy loading works
- [ ] Priority loading works for hero images
- [ ] No layout shift on image load
- [ ] Theme switch is smooth

### Cross-browser

- [ ] Works in Chrome, Firefox, Safari, Edge
- [ ] Picture element fallback works
- [ ] CSS transitions work

````
