````markdown
# Lightbox Component Specification

## Overview

The Lightbox component provides a full-screen modal overlay for viewing images at their full resolution. It is triggered by clicking on Screenshot components and supports keyboard navigation, focus trapping, and accessible dismissal.

---

## 1. Component Hierarchy

```
Lightbox (organism) [portal to body]
├── LightboxBackdrop (atom)
├── LightboxDialog (molecule)
│   ├── LightboxHeader (molecule)
│   │   └── CloseButton (atom)
│   ├── LightboxContent (molecule)
│   │   ├── LightboxImage (atom)
│   │   └── LoadingSpinner (atom) [while loading]
│   └── LightboxFooter (molecule) [optional]
│       └── LightboxCaption (atom)
└── LightboxAnnouncer (atom) [screen reader only]
```

---

## 2. Props Interface

### Lightbox

```typescript
interface LightboxProps {
  /** Whether the lightbox is open */
  isOpen: boolean;
  /** Handler called when lightbox should close */
  onClose: () => void;
  /** Image source to display */
  src: string;
  /** Alt text for the image (required) */
  alt: string;
  /** Optional caption text */
  caption?: string;
  /** Current theme for themed images */
  theme?: 'light' | 'dark';
  /** Light theme variant source (for theme switching) */
  lightSrc?: string;
  /** Dark theme variant source (for theme switching) */
  darkSrc?: string;
  /** Callback when image finishes loading */
  onImageLoad?: () => void;
  /** Callback when image fails to load */
  onImageError?: (error: Error) => void;
  /** Custom class name */
  className?: string;
  /** Enable zoom functionality */
  enableZoom?: boolean;
}
```

### CloseButton (internal)

```typescript
interface CloseButtonProps {
  /** Click handler */
  onClick: () => void;
  /** Size variant */
  size?: 'sm' | 'md' | 'lg';
  /** Custom class name */
  className?: string;
}
```

### LightboxImage (internal)

```typescript
interface LightboxImageProps {
  /** Image source */
  src: string;
  /** Alt text */
  alt: string;
  /** Load handler */
  onLoad?: () => void;
  /** Error handler */
  onError?: (error: Error) => void;
  /** Enable zoom on click */
  enableZoom?: boolean;
  /** Current zoom level */
  zoomLevel?: number;
}
```

---

## 3. States

### Lightbox States

| State     | Trigger                | Visual Changes                            |
|-----------|------------------------|-------------------------------------------|
| Closed    | Initial / onClose      | Not rendered in DOM                       |
| Opening   | isOpen becomes true    | Fade in backdrop, scale/fade in image     |
| Open      | Animation complete     | Full visibility, focus trapped            |
| Loading   | Image loading          | Spinner visible, image hidden             |
| Loaded    | Image onload           | Image visible, spinner hidden             |
| Error     | Image onerror          | Error message with retry option           |
| Closing   | Escape / click outside | Fade out backdrop, scale/fade out image   |

### Image States

| State     | Trigger              | Visual Changes                              |
|-----------|---------------------|---------------------------------------------|
| Loading   | Lightbox opens       | Loading spinner centered                    |
| Loaded    | Image loaded         | Full-size image displayed                   |
| Error     | Load failed          | Error message with retry                    |
| Zoomed    | Click (if enabled)   | Image scaled up, scrollable                 |

### Close Button States

| State     | Trigger              | Visual Changes                              |
|-----------|---------------------|---------------------------------------------|
| Default   | Initial              | X icon visible                              |
| Hover     | Mouse enter          | Background highlight                        |
| Focus     | Keyboard focus       | Focus ring visible                          |
| Active    | Mouse down           | Scale down slightly                         |

---

## 4. Visual Specifications

### Layout

```
Lightbox Open:
┌─────────────────────────────────────────────────────────────────────┐
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ [×]│
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│░░░░░░░░░░┌────────────────────────────────────────────┐░░░░░░░░░░░░░│
│░░░░░░░░░░│                                            │░░░░░░░░░░░░░│
│░░░░░░░░░░│                                            │░░░░░░░░░░░░░│
│░░░░░░░░░░│              Full Size Image               │░░░░░░░░░░░░░│
│░░░░░░░░░░│                                            │░░░░░░░░░░░░░│
│░░░░░░░░░░│                                            │░░░░░░░░░░░░░│
│░░░░░░░░░░│                                            │░░░░░░░░░░░░░│
│░░░░░░░░░░└────────────────────────────────────────────┘░░░░░░░░░░░░░│
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│                    Caption: Dashboard overview                       │
└─────────────────────────────────────────────────────────────────────┘

Loading State:
┌─────────────────────────────────────────────────────────────────────┐
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ [×]│
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│░░░░░░░░░░░░░░░░░░░░░░░ ◌ Loading... ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
└─────────────────────────────────────────────────────────────────────┘

Error State:
┌─────────────────────────────────────────────────────────────────────┐
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ [×]│
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│░░░░░░░░░░░░░░░ ⚠️ Failed to load image ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│░░░░░░░░░░░░░░░░░░░░ [ Retry ] ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
│░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
└─────────────────────────────────────────────────────────────────────┘
```

### Dimensions

| Property                 | Value                         |
|--------------------------|-------------------------------|
| Backdrop                 | 100vw × 100vh                 |
| Image max-width          | calc(100vw - 48px)            |
| Image max-height         | calc(100vh - 120px)           |
| Close button size        | 48px × 48px                   |
| Close button position    | 16px from top-right           |
| Caption padding          | 16px 24px                     |
| Caption max-width        | 600px                         |
| Caption font-size        | 14px (--font-size-sm)         |
| Loading spinner size     | 48px                          |
| Border radius (image)    | 4px                           |
| Padding (mobile)         | 16px                          |
| Padding (desktop)        | 48px                          |

### Colors

| Element                | Light Theme          | Dark Theme           |
|------------------------|----------------------|----------------------|
| Backdrop               | rgba(0, 0, 0, 0.9)   | rgba(0, 0, 0, 0.95)  |
| Close button bg        | rgba(255,255,255,0.1)| rgba(255,255,255,0.1)|
| Close button hover     | rgba(255,255,255,0.2)| rgba(255,255,255,0.2)|
| Close button icon      | white                | white                |
| Close button focus     | 0 0 0 3px #60a5fa    | 0 0 0 3px #60a5fa    |
| Caption text           | #e5e7eb              | #e5e7eb              |
| Caption bg             | rgba(0, 0, 0, 0.5)   | rgba(0, 0, 0, 0.5)   |
| Loading spinner        | white                | white                |
| Error text             | #fca5a5              | #fca5a5              |
| Error bg               | rgba(127, 29, 29, 0.5)| rgba(127, 29, 29, 0.5)|

---

## 5. Interactions

### Opening the Lightbox

| Trigger              | Action                                       |
|---------------------|----------------------------------------------|
| Screenshot click    | Lightbox opens with corresponding image       |
| Screenshot Enter    | Lightbox opens with corresponding image       |
| Screenshot Space    | Lightbox opens with corresponding image       |

### Closing the Lightbox

| Trigger              | Action                                       |
|---------------------|----------------------------------------------|
| Escape key          | Close lightbox                                |
| Click backdrop      | Close lightbox                                |
| Click close button  | Close lightbox                                |
| Enter on close btn  | Close lightbox                                |
| Space on close btn  | Close lightbox                                |

### Navigation (Future Enhancement)

| Trigger              | Action                                       |
|---------------------|----------------------------------------------|
| Arrow Left          | Previous image (if gallery mode)             |
| Arrow Right         | Next image (if gallery mode)                 |
| Arrow keys          | Pan zoomed image                             |

### Zoom (Optional)

| Trigger              | Action                                       |
|---------------------|----------------------------------------------|
| Click image         | Toggle zoom in/out                           |
| Scroll wheel        | Zoom in/out incrementally                    |
| Pinch gesture       | Zoom in/out (touch)                          |
| Double-tap          | Toggle zoom (touch)                          |

---

## 6. Focus Management

### Focus Trap

```typescript
interface FocusTrapConfig {
  /** Container element to trap focus within */
  container: HTMLElement;
  /** Initial element to focus */
  initialFocus?: HTMLElement;
  /** Element to return focus to on close */
  returnFocus: HTMLElement;
}

function createFocusTrap(config: FocusTrapConfig) {
  const focusableSelectors = [
    'button',
    '[href]',
    'input',
    'select',
    'textarea',
    '[tabindex]:not([tabindex="-1"])',
  ];
  
  const focusableElements = config.container.querySelectorAll(
    focusableSelectors.join(',')
  );
  
  const firstFocusable = focusableElements[0] as HTMLElement;
  const lastFocusable = focusableElements[focusableElements.length - 1] as HTMLElement;
  
  function handleKeyDown(event: KeyboardEvent) {
    if (event.key !== 'Tab') return;
    
    if (event.shiftKey) {
      if (document.activeElement === firstFocusable) {
        event.preventDefault();
        lastFocusable.focus();
      }
    } else {
      if (document.activeElement === lastFocusable) {
        event.preventDefault();
        firstFocusable.focus();
      }
    }
  }
  
  // Set initial focus
  (config.initialFocus || firstFocusable)?.focus();
  
  // Add listener
  config.container.addEventListener('keydown', handleKeyDown);
  
  return {
    release() {
      config.container.removeEventListener('keydown', handleKeyDown);
      config.returnFocus.focus();
    }
  };
}
```

### Focus Flow

1. **Open**: Focus moves to close button
2. **Tab**: Cycles through close button → image (if zoom enabled) → close button
3. **Shift+Tab**: Reverse cycle
4. **Close**: Focus returns to triggering Screenshot element

---

## 7. Keyboard Navigation

| Key           | Action                                         |
|---------------|------------------------------------------------|
| Escape        | Close lightbox                                 |
| Tab           | Move to next focusable element                 |
| Shift+Tab     | Move to previous focusable element             |
| Enter         | Activate focused element                       |
| Space         | Activate focused element                       |
| Arrow keys    | Pan zoomed image (when zoom enabled)           |
| +/=           | Zoom in (when zoom enabled)                    |
| -             | Zoom out (when zoom enabled)                   |
| 0             | Reset zoom (when zoom enabled)                 |

---

## 8. Accessibility

### Semantic HTML Structure

```html
<div 
  class="lightbox"
  role="dialog"
  aria-modal="true"
  aria-labelledby="lightbox-title"
  aria-describedby="lightbox-description"
>
  <!-- Backdrop (click to close) -->
  <div 
    class="lightbox-backdrop" 
    aria-hidden="true"
    data-close-trigger
  ></div>
  
  <!-- Dialog content -->
  <div class="lightbox-dialog">
    <!-- Header with close button -->
    <div class="lightbox-header">
      <button
        type="button"
        class="lightbox-close"
        aria-label="Close image viewer"
        autofocus
      >
        <svg aria-hidden="true"><!-- X icon --></svg>
      </button>
    </div>
    
    <!-- Image container -->
    <div class="lightbox-content">
      <img
        id="lightbox-image"
        src="/screenshots/dashboard-dark.png"
        alt="Dashboard showing service status and health metrics"
        class="lightbox-image"
      />
      
      <!-- Screen reader only title -->
      <h2 id="lightbox-title" class="sr-only">
        Image viewer
      </h2>
    </div>
    
    <!-- Caption -->
    <div class="lightbox-footer">
      <p id="lightbox-description" class="lightbox-caption">
        Dashboard overview showing running services
      </p>
    </div>
  </div>
  
  <!-- Screen reader announcements -->
  <div aria-live="polite" class="sr-only" id="lightbox-announcer">
    <!-- Dynamic announcements -->
  </div>
</div>
```

### ARIA Attributes

| Element          | Attribute                    | Purpose                          |
|------------------|------------------------------|----------------------------------|
| Container        | role="dialog"                | Semantic dialog                  |
| Container        | aria-modal="true"            | Indicates modal behavior         |
| Container        | aria-labelledby              | References title                 |
| Container        | aria-describedby             | References caption               |
| Close button     | aria-label                   | Describe action                  |
| Image            | alt                          | Image description                |
| Backdrop         | aria-hidden="true"           | Hide from screen readers         |
| Announcer        | aria-live="polite"           | Dynamic announcements            |

### Screen Reader Announcements

```typescript
// On open
announceToScreenReader("Image viewer opened. Dashboard screenshot. Press Escape to close.");

// On image load
announceToScreenReader("Image loaded. Dashboard showing service status and health metrics.");

// On error
announceToScreenReader("Failed to load image. Retry button available.");

// On close
announceToScreenReader("Image viewer closed.");
```

### Body Scroll Lock

```typescript
function lockBodyScroll(): void {
  const scrollY = window.scrollY;
  document.body.style.position = 'fixed';
  document.body.style.top = `-${scrollY}px`;
  document.body.style.width = '100%';
  document.body.dataset.scrollY = String(scrollY);
}

function unlockBodyScroll(): void {
  const scrollY = document.body.dataset.scrollY;
  document.body.style.position = '';
  document.body.style.top = '';
  document.body.style.width = '';
  if (scrollY) {
    window.scrollTo(0, parseInt(scrollY));
  }
}
```

---

## 9. Animation Specifications

### Opening Animation

```css
/* Backdrop fade in */
.lightbox-backdrop {
  opacity: 0;
  transition: opacity 0.3s ease-out;
}

.lightbox--open .lightbox-backdrop {
  opacity: 1;
}

/* Dialog scale and fade */
.lightbox-dialog {
  opacity: 0;
  transform: scale(0.95);
  transition: opacity 0.3s ease-out,
              transform 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.lightbox--open .lightbox-dialog {
  opacity: 1;
  transform: scale(1);
}
```

### Closing Animation

```css
/* Reverse of opening */
.lightbox--closing .lightbox-backdrop {
  opacity: 0;
}

.lightbox--closing .lightbox-dialog {
  opacity: 0;
  transform: scale(0.95);
}
```

### Image Load Animation

```css
.lightbox-image {
  opacity: 0;
  transition: opacity 0.3s ease-out;
}

.lightbox-image--loaded {
  opacity: 1;
}
```

### Loading Spinner

```css
.lightbox-spinner {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
```

### Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  .lightbox-backdrop,
  .lightbox-dialog,
  .lightbox-image {
    transition: opacity 0.1s ease-out;
  }
  
  .lightbox-dialog {
    transform: none !important;
  }
  
  .lightbox-spinner {
    animation: none;
    /* Use pulsing opacity instead */
    animation: pulse 1.5s ease-in-out infinite;
  }
  
  @keyframes pulse {
    0%, 100% { opacity: 0.4; }
    50% { opacity: 1; }
  }
}
```

---

## 10. Theme Switching in Lightbox

### Behavior

When the user switches themes while the lightbox is open, the displayed image should update to match.

```typescript
// Listen for theme changes while lightbox is open
useEffect(() => {
  if (!isOpen) return;
  
  const observer = new MutationObserver((mutations) => {
    for (const mutation of mutations) {
      if (mutation.attributeName === 'data-theme') {
        const newTheme = document.documentElement.getAttribute('data-theme');
        const newSrc = newTheme === 'dark' ? darkSrc : lightSrc;
        setCurrentSrc(newSrc);
      }
    }
  });
  
  observer.observe(document.documentElement, { attributes: true });
  
  return () => observer.disconnect();
}, [isOpen, darkSrc, lightSrc]);
```

### Crossfade on Theme Change

```css
.lightbox-image {
  transition: opacity 0.2s ease-out;
}

.lightbox-image--switching {
  opacity: 0.7;
}
```

---

## 11. Portal Rendering

The lightbox should be rendered as a portal to the document body to ensure proper stacking context.

### Astro Implementation

```astro
---
// Lightbox.astro
interface Props {
  id: string;
}

const { id } = Astro.props;
---

<div id={id} class="lightbox" data-lightbox hidden>
  <div class="lightbox-backdrop" data-close-trigger></div>
  <div class="lightbox-dialog">
    <button class="lightbox-close" aria-label="Close image viewer">
      <svg><!-- X icon --></svg>
    </button>
    <div class="lightbox-content">
      <img class="lightbox-image" src="" alt="" />
    </div>
    <div class="lightbox-footer">
      <p class="lightbox-caption"></p>
    </div>
  </div>
</div>

<script>
  // Move to body on mount
  document.addEventListener('DOMContentLoaded', () => {
    const lightbox = document.getElementById('{id}');
    if (lightbox && lightbox.parentElement !== document.body) {
      document.body.appendChild(lightbox);
    }
  });
</script>
```

### React Implementation

```tsx
import { createPortal } from 'react-dom';

function Lightbox({ isOpen, ...props }: LightboxProps) {
  if (!isOpen) return null;
  
  return createPortal(
    <LightboxContent {...props} />,
    document.body
  );
}
```

---

## 12. Error Handling

| Scenario               | Behavior                                     |
|------------------------|---------------------------------------------|
| Image fails to load    | Show error state with retry button          |
| Theme variant missing  | Use available variant                       |
| Both variants missing  | Show error message                          |
| Network timeout        | Show error after 30s, offer retry           |

### Error State UI

```html
<div class="lightbox-error">
  <svg class="lightbox-error-icon" aria-hidden="true">
    <!-- Warning icon -->
  </svg>
  <p class="lightbox-error-message">Failed to load image</p>
  <button 
    type="button" 
    class="lightbox-retry-button"
    aria-label="Retry loading image"
  >
    Retry
  </button>
</div>
```

---

## 13. CSS Custom Properties

```css
/* Lightbox Component Tokens */
--lightbox-backdrop-bg: rgba(0, 0, 0, 0.9);
--lightbox-backdrop-bg-dark: rgba(0, 0, 0, 0.95);
--lightbox-close-bg: rgba(255, 255, 255, 0.1);
--lightbox-close-hover-bg: rgba(255, 255, 255, 0.2);
--lightbox-close-color: white;
--lightbox-close-size: 48px;
--lightbox-close-icon-size: 24px;
--lightbox-close-position: 16px;
--lightbox-image-max-width: calc(100vw - 48px);
--lightbox-image-max-height: calc(100vh - 120px);
--lightbox-image-border-radius: 4px;
--lightbox-caption-bg: rgba(0, 0, 0, 0.5);
--lightbox-caption-color: #e5e7eb;
--lightbox-caption-padding: 16px 24px;
--lightbox-caption-max-width: 600px;
--lightbox-spinner-size: 48px;
--lightbox-spinner-color: white;
--lightbox-focus-ring: 0 0 0 3px #60a5fa;
--lightbox-transition-duration: 0.3s;
--lightbox-transition-timing: cubic-bezier(0.16, 1, 0.3, 1);
--lightbox-padding-mobile: 16px;
--lightbox-padding-desktop: 48px;

/* Z-index */
--lightbox-z-index: 9999;
```

---

## 14. Responsive Design

### Breakpoint Behavior

| Breakpoint        | Changes                                     |
|-------------------|---------------------------------------------|
| Mobile (<640px)   | Close button larger, image full-width       |
| Tablet (640-1023px)| Standard padding, close in corner          |
| Desktop (≥1024px) | Generous padding, caption below             |

### Mobile Considerations

```css
@media (max-width: 640px) {
  .lightbox-dialog {
    padding: var(--lightbox-padding-mobile);
  }
  
  .lightbox-close {
    /* Larger touch target on mobile */
    width: 56px;
    height: 56px;
    top: 8px;
    right: 8px;
  }
  
  .lightbox-image {
    max-width: calc(100vw - 32px);
    max-height: calc(100vh - 100px);
  }
  
  .lightbox-caption {
    font-size: 13px;
    padding: 12px 16px;
  }
  
  /* Swipe to close on mobile */
  .lightbox-dialog {
    touch-action: pan-y;
  }
}
```

### Touch Gestures

- **Swipe down**: Close lightbox
- **Pinch**: Zoom in/out (if enabled)
- **Double-tap**: Toggle zoom (if enabled)

---

## 15. Testing Checklist

### Functionality

- [ ] Opens when Screenshot is clicked
- [ ] Opens when Screenshot receives Enter key
- [ ] Displays correct image
- [ ] Theme variant switches correctly
- [ ] Caption displays when provided
- [ ] Loading state shows spinner
- [ ] Error state shows retry option
- [ ] Retry button works

### Closing Behavior

- [ ] Escape key closes lightbox
- [ ] Click on backdrop closes lightbox
- [ ] Click on close button closes lightbox
- [ ] Focus returns to triggering element

### Accessibility

- [ ] Focus moves to close button on open
- [ ] Focus is trapped within lightbox
- [ ] Tab cycles through focusable elements
- [ ] Screen reader announces open/close
- [ ] Alt text is announced
- [ ] Caption is associated properly
- [ ] Reduced motion preference respected

### Focus Management

- [ ] Focus trapped in lightbox
- [ ] Tab cycles correctly
- [ ] Shift+Tab cycles correctly
- [ ] Focus returns on close

### Responsive

- [ ] Works on mobile devices
- [ ] Touch targets adequate (≥44x44px)
- [ ] Caption readable on small screens
- [ ] Image scales appropriately

### Performance

- [ ] Renders in portal to body
- [ ] Body scroll locked when open
- [ ] No layout shift on open/close
- [ ] Animation is smooth

### Cross-browser

- [ ] Works in Chrome, Firefox, Safari, Edge
- [ ] Focus trap works in all browsers
- [ ] Animations work in all browsers
- [ ] Portal rendering works

````
