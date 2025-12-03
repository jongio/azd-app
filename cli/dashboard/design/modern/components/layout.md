# Modern Layout Specification

> Overall page structure and layout patterns for the Modern dashboard theme.

## Layout Philosophy

The Modern layout uses a **top navigation + content area** pattern instead of the traditional sidebar layout. This provides:

- Maximum horizontal space for content
- Cleaner visual hierarchy
- Better mobile responsiveness
- Modern, app-like feel

---

## Page Structure

```
┌─────────────────────────────────────────────────────────────┐
│                        Header Bar                           │
│  [Logo] [Nav Items...]                    [Status] [Actions]│
├─────────────────────────────────────────────────────────────┤
│                                                             │
│                                                             │
│                      Main Content                           │
│                                                             │
│                                                             │
│                                                             │
│                                                             │
│                                                             │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## CSS Grid Structure

```css
.modern-layout {
  display: grid;
  grid-template-rows: var(--header-height) 1fr;
  grid-template-columns: 1fr;
  min-height: 100vh;
  background: var(--background);
}

.modern-header {
  grid-row: 1;
  position: sticky;
  top: 0;
  z-index: var(--z-sticky);
}

.modern-main {
  grid-row: 2;
  overflow-y: auto;
  overflow-x: hidden;
}
```

---

## Layout Dimensions

### Header

| Property | Value | Notes |
|----------|-------|-------|
| Height | 64px | Fixed height |
| Padding | 0 24px | Horizontal padding |
| Gap | 16px | Between elements |
| Border | 1px bottom | Subtle separator |

### Content Area

| Property | Value | Notes |
|----------|-------|-------|
| Max Width | 1536px | Container constraint |
| Padding | 24px (default), 32px (lg) | Responsive padding |
| Background | `--background-tertiary` | Subtle texture |

---

## Layout Variants

### Standard Layout

The default layout with header and scrollable content.

```jsx
<div className="modern-layout">
  <ModernHeader />
  <main className="modern-main">
    <div className="modern-container">
      {/* Page content */}
    </div>
  </main>
</div>
```

### Split View Layout (Console)

For the console view with resizable panels.

```
┌─────────────────────────────────────────────────────────────┐
│                        Header Bar                           │
├─────────────────────────────────────────────────────────────┤
│                      Toolbar Strip                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│                   Log Panes Grid                            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

```css
.modern-split-layout {
  display: grid;
  grid-template-rows: var(--header-height) auto 1fr;
  min-height: 100vh;
}
```

### Fullscreen Mode

Removes header for maximum content space.

```css
.modern-fullscreen {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal);
  display: grid;
  grid-template-rows: 1fr;
  background: var(--background);
}
```

---

## Container System

### Container Widths

```css
.modern-container {
  width: 100%;
  max-width: var(--container-xl);
  margin: 0 auto;
  padding: 0 var(--space-6);
}

.modern-container-sm {
  max-width: var(--container-md);
}

.modern-container-lg {
  max-width: var(--container-2xl);
}

.modern-container-fluid {
  max-width: none;
  padding: 0 var(--space-4);
}
```

### Content Padding

```css
.modern-content-padding {
  padding: var(--space-6);
}

@media (min-width: 1024px) {
  .modern-content-padding {
    padding: var(--space-8);
  }
}
```

---

## Grid System

### Resource Grid

For service cards and metric displays.

```css
.modern-grid {
  display: grid;
  gap: var(--space-4);
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
}

/* Explicit column variants */
.modern-grid-2 {
  grid-template-columns: repeat(2, 1fr);
}

.modern-grid-3 {
  grid-template-columns: repeat(3, 1fr);
}

.modern-grid-4 {
  grid-template-columns: repeat(4, 1fr);
}

@media (max-width: 768px) {
  .modern-grid-2,
  .modern-grid-3,
  .modern-grid-4 {
    grid-template-columns: 1fr;
  }
}
```

### Dashboard Grid

For the main dashboard with status summary.

```css
.modern-dashboard-grid {
  display: grid;
  gap: var(--space-6);
  grid-template-columns: 1fr;
  grid-template-areas:
    "summary"
    "services"
    "activity";
}

@media (min-width: 1024px) {
  .modern-dashboard-grid {
    grid-template-columns: 280px 1fr;
    grid-template-areas:
      "summary services"
      "summary activity";
  }
}

@media (min-width: 1280px) {
  .modern-dashboard-grid {
    grid-template-columns: 300px 1fr 320px;
    grid-template-areas:
      "summary services activity"
      "summary services activity";
  }
}
```

---

## Spacing Patterns

### Section Spacing

```css
.modern-section {
  margin-bottom: var(--space-8);
}

.modern-section-header {
  margin-bottom: var(--space-4);
}

.modern-section-title {
  font-size: var(--text-xl);
  font-weight: var(--font-semibold);
  color: var(--foreground);
}
```

### Stack Layouts

```css
.modern-stack {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.modern-stack-sm {
  gap: var(--space-2);
}

.modern-stack-lg {
  gap: var(--space-6);
}
```

### Inline Layouts

```css
.modern-inline {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.modern-inline-sm {
  gap: var(--space-2);
}

.modern-inline-lg {
  gap: var(--space-4);
}

.modern-inline-between {
  justify-content: space-between;
}
```

---

## Responsive Breakpoints

### Layout Behavior

| Breakpoint | Layout Changes |
|------------|---------------|
| < 640px | Stack navigation, single column, condensed padding |
| 640-768px | Compact header, 2-column grid |
| 768-1024px | Full header, 2-3 column grid |
| 1024-1280px | Standard layout, 3-4 column grid |
| > 1280px | Extended layout, sidebar visibility |

### Mobile Adaptations

```css
@media (max-width: 640px) {
  :root {
    --header-height: 56px;
  }
  
  .modern-container {
    padding: 0 var(--space-4);
  }
  
  .modern-content-padding {
    padding: var(--space-4);
  }
}
```

---

## Scroll Behavior

### Main Content

```css
.modern-main {
  overflow-y: auto;
  scroll-behavior: smooth;
  scrollbar-width: thin;
  scrollbar-color: var(--border-strong) transparent;
}

.modern-main::-webkit-scrollbar {
  width: 8px;
}

.modern-main::-webkit-scrollbar-track {
  background: transparent;
}

.modern-main::-webkit-scrollbar-thumb {
  background-color: var(--border-strong);
  border-radius: var(--radius-full);
  border: 2px solid transparent;
  background-clip: padding-box;
}

.modern-main::-webkit-scrollbar-thumb:hover {
  background-color: var(--foreground-muted);
}
```

### Horizontal Scroll Containers

```css
.modern-scroll-x {
  overflow-x: auto;
  overflow-y: hidden;
  scroll-snap-type: x mandatory;
}

.modern-scroll-x > * {
  scroll-snap-align: start;
}
```

---

## Overlay Layouts

### Modal Backdrop

```css
.modern-backdrop {
  position: fixed;
  inset: 0;
  z-index: var(--z-overlay);
  background: var(--bg-overlay);
  backdrop-filter: blur(4px);
  animation: modern-fade-in var(--duration-normal) var(--ease-out);
}
```

### Slide Panel

```css
.modern-panel-overlay {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal);
  display: flex;
  justify-content: flex-end;
}

.modern-panel {
  width: 100%;
  max-width: 500px;
  height: 100%;
  background: var(--background);
  box-shadow: var(--shadow-xl);
  animation: modern-slide-in-right var(--duration-slow) var(--ease-out);
}
```

---

## Layout Component Structure

```tsx
// ModernLayout.tsx
interface ModernLayoutProps {
  children: React.ReactNode;
  variant?: 'standard' | 'split' | 'fullscreen';
}

export function ModernLayout({ children, variant = 'standard' }: ModernLayoutProps) {
  return (
    <div className={`modern-layout modern-layout--${variant}`}>
      {variant !== 'fullscreen' && <ModernHeader />}
      <main className="modern-main">
        {children}
      </main>
    </div>
  );
}
```

---

## Accessibility

### Landmarks

- Header: `<header role="banner">`
- Navigation: `<nav role="navigation" aria-label="Main">`
- Main: `<main role="main">`
- Regions: Use `role="region"` with `aria-label`

### Skip Links

```jsx
<a href="#main-content" className="modern-skip-link">
  Skip to main content
</a>

// Style hidden until focused
.modern-skip-link {
  position: absolute;
  top: -40px;
  left: 0;
  padding: var(--space-2) var(--space-4);
  background: var(--primary);
  color: var(--primary-foreground);
  z-index: var(--z-tooltip);
  transition: top var(--duration-fast);
}

.modern-skip-link:focus {
  top: 0;
}
```

### Focus Management

- Trap focus in modals
- Return focus on panel close
- Logical tab order
