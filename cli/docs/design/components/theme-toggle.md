# Theme Toggle Component Specification

## Overview

The Theme Toggle component allows users to switch between light, dark, and system color schemes. It persists the user's preference and provides smooth transitions between themes.

---

## 1. Component Hierarchy

```
ThemeToggle (molecule)
├── ThemeToggleButton (atom)
│   ├── SunIcon (atom)
│   ├── MoonIcon (atom)
│   └── SystemIcon (atom) [optional]
├── ThemeDropdown (molecule) [optional, for 3-way toggle]
│   ├── ThemeOption (atom) × 3
│   └── ThemeIndicator (atom)
└── ThemeStatusAnnouncer (atom) [screen reader only]
```

---

## 2. Props Interface

### ThemeToggle

```typescript
interface ThemeToggleProps {
  /** Current theme value */
  theme: Theme;
  /** Callback when theme changes */
  onThemeChange: (theme: Theme) => void;
  /** Toggle variant */
  variant?: 'icon' | 'button' | 'dropdown';
  /** Size */
  size?: 'sm' | 'md' | 'lg';
  /** Show label (desktop only) */
  showLabel?: boolean;
  /** Custom class name */
  className?: string;
}

type Theme = 'light' | 'dark' | 'system';
```

### ThemeOption

```typescript
interface ThemeOptionProps {
  /** Theme value */
  value: Theme;
  /** Option label */
  label: string;
  /** Icon component */
  icon: React.ReactNode;
  /** Whether this option is selected */
  isSelected: boolean;
  /** Click handler */
  onSelect: () => void;
}
```

---

## 3. States

### Button States

| State    | Trigger            | Visual Changes                        |
|----------|-------------------|---------------------------------------|
| Default  | Initial render     | Current theme icon visible            |
| Hover    | Mouse enter        | Background highlight, slight glow     |
| Focus    | Keyboard focus     | Focus ring visible                    |
| Active   | Mouse down/touch   | Scale down slightly, icon animates    |
| Disabled | `disabled={true}`  | Muted icon, cursor not-allowed        |

### Icon States

| Theme   | Icon Displayed | Animation on Switch                   |
|---------|---------------|---------------------------------------|
| Light   | Sun ☀️         | Rotate in from moon position          |
| Dark    | Moon 🌙        | Rotate in from sun position           |
| System  | Monitor 🖥️     | Fade in (no rotation)                 |

### Dropdown States (if using dropdown variant)

| State    | Trigger          | Visual Changes                        |
|----------|------------------|---------------------------------------|
| Closed   | Initial/blur     | Only button visible                   |
| Open     | Click button     | Dropdown visible with all options     |
| Focused  | Option focus     | Option highlighted                    |

---

## 4. Visual Specifications

### Layout Variants

```
Icon Only (default):
┌─────┐
│ 🌙  │
└─────┘

Button with Label:
┌─────────────┐
│ 🌙  Dark    │
└─────────────┘

Dropdown:
┌─────┐
│ 🌙  │ ▼
├─────┴─────────┐
│ ☀️  Light     │
│ 🌙  Dark    ✓ │
│ 🖥️  System    │
└───────────────┘

Toggle Switch (alternative):
┌───────────────────┐
│ ☀️ ═══════●═══ 🌙 │
└───────────────────┘
```

### Dimensions

| Property         | sm      | md (default) | lg      |
|------------------|---------|--------------|---------|
| Button size      | 32px    | 40px         | 48px    |
| Icon size        | 16px    | 20px         | 24px    |
| Border radius    | 8px     | 8px          | 12px    |
| Dropdown width   | 120px   | 140px        | 160px   |
| Option height    | 36px    | 40px         | 44px    |

### Colors

#### Light Theme (showing dark mode option)

| Element                | Property    | Value               |
|------------------------|-------------|---------------------|
| Button bg              | background  | transparent         |
| Button hover bg        | background  | #f3f4f6             |
| Icon color             | color       | #4b5563             |
| Icon hover color       | color       | #111827             |
| Focus ring             | box-shadow  | 0 0 0 3px #93c5fd   |
| Dropdown bg            | background  | white               |
| Dropdown border        | border      | 1px solid #e5e7eb   |
| Dropdown shadow        | box-shadow  | 0 10px 15px rgba()  |
| Option hover bg        | background  | #f3f4f6             |
| Selected option        | color       | #2563eb             |
| Checkmark              | color       | #2563eb             |

#### Dark Theme (showing light mode option)

| Element                | Property    | Value               |
|------------------------|-------------|---------------------|
| Button bg              | background  | transparent         |
| Button hover bg        | background  | #334155             |
| Icon color             | color       | #94a3b8             |
| Icon hover color       | color       | #f1f5f9             |
| Focus ring             | box-shadow  | 0 0 0 3px #1e40af   |
| Dropdown bg            | background  | #1e293b             |
| Dropdown border        | border      | 1px solid #334155   |
| Option hover bg        | background  | #334155             |
| Selected option        | color       | #60a5fa             |
| Checkmark              | color       | #60a5fa             |

---

## 5. Icon Specifications

### Sun Icon

```svg
<svg 
  width="20" 
  height="20" 
  viewBox="0 0 24 24" 
  fill="none"
  stroke="currentColor" 
  stroke-width="2"
  stroke-linecap="round" 
  stroke-linejoin="round"
  aria-hidden="true"
>
  <circle cx="12" cy="12" r="5"></circle>
  <line x1="12" y1="1" x2="12" y2="3"></line>
  <line x1="12" y1="21" x2="12" y2="23"></line>
  <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
  <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
  <line x1="1" y1="12" x2="3" y2="12"></line>
  <line x1="21" y1="12" x2="23" y2="12"></line>
  <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
  <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
</svg>
```

### Moon Icon

```svg
<svg 
  width="20" 
  height="20" 
  viewBox="0 0 24 24" 
  fill="none"
  stroke="currentColor" 
  stroke-width="2"
  stroke-linecap="round" 
  stroke-linejoin="round"
  aria-hidden="true"
>
  <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
</svg>
```

### System Icon

```svg
<svg 
  width="20" 
  height="20" 
  viewBox="0 0 24 24" 
  fill="none"
  stroke="currentColor" 
  stroke-width="2"
  stroke-linecap="round" 
  stroke-linejoin="round"
  aria-hidden="true"
>
  <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
  <line x1="8" y1="21" x2="16" y2="21"></line>
  <line x1="12" y1="17" x2="12" y2="21"></line>
</svg>
```

---

## 6. Interactions

### Click/Tap Behavior

| Variant  | Action              | Result                               |
|----------|---------------------|--------------------------------------|
| Icon     | Click               | Cycle: light → dark → system → light |
| Button   | Click               | Cycle: light → dark → system → light |
| Dropdown | Click button        | Open dropdown                        |
| Dropdown | Click option        | Select theme, close dropdown         |
| Dropdown | Click outside       | Close dropdown                       |

### Keyboard

| Key         | Behavior                                        |
|-------------|------------------------------------------------|
| Tab         | Focus the toggle button                         |
| Enter       | Activate (cycle or open dropdown)               |
| Space       | Activate (cycle or open dropdown)               |
| Escape      | Close dropdown (if open)                        |
| Arrow Down  | Open dropdown / move to next option             |
| Arrow Up    | Move to previous option                         |
| Home        | Move to first option (Light)                    |
| End         | Move to last option (System)                    |

### Touch

| Action        | Behavior                                      |
|---------------|-----------------------------------------------|
| Tap           | Cycle through themes (icon/button)            |
| Tap           | Select option (dropdown)                      |
| Long press    | Show tooltip with current theme (optional)    |

---

## 7. Accessibility

### ARIA Attributes

#### Simple Toggle (Icon/Button)

```html
<button
  type="button"
  class="theme-toggle"
  aria-label="Toggle theme"
  aria-pressed="mixed"
  aria-describedby="theme-status"
>
  <svg class="icon sun" aria-hidden="true"><!-- sun --></svg>
  <svg class="icon moon" aria-hidden="true"><!-- moon --></svg>
  <svg class="icon system" aria-hidden="true"><!-- monitor --></svg>
</button>

<span id="theme-status" class="sr-only" aria-live="polite">
  Current theme: dark
</span>
```

#### Dropdown Variant

```html
<div class="theme-toggle-dropdown">
  <button
    type="button"
    class="theme-toggle-button"
    aria-haspopup="listbox"
    aria-expanded="false"
    aria-label="Select theme, current: dark"
  >
    <svg aria-hidden="true"><!-- current icon --></svg>
    <span class="sr-only">Theme</span>
  </button>
  
  <ul
    role="listbox"
    aria-label="Theme options"
    tabindex="-1"
    hidden
  >
    <li role="option" aria-selected="false" data-value="light">
      <svg aria-hidden="true"><!-- sun --></svg>
      <span>Light</span>
    </li>
    <li role="option" aria-selected="true" data-value="dark">
      <svg aria-hidden="true"><!-- moon --></svg>
      <span>Dark</span>
      <svg aria-hidden="true"><!-- checkmark --></svg>
    </li>
    <li role="option" aria-selected="false" data-value="system">
      <svg aria-hidden="true"><!-- monitor --></svg>
      <span>System</span>
    </li>
  </ul>
</div>
```

### Screen Reader Announcements

- On toggle: "Theme changed to dark mode"
- On dropdown open: "Theme options menu, 3 options"
- On selection: "Dark theme selected"
- System theme: "System theme, currently light" (reflects actual)

---

## 8. Animation Specifications

### Icon Transition (Rotate)

```css
.theme-toggle {
  position: relative;
  overflow: hidden;
}

.theme-toggle .icon {
  position: absolute;
  transition: transform 0.3s cubic-bezier(0.68, -0.55, 0.265, 1.55),
              opacity 0.15s ease-out;
}

/* Light theme - show sun */
[data-theme="light"] .theme-toggle .sun {
  transform: rotate(0deg);
  opacity: 1;
}

[data-theme="light"] .theme-toggle .moon {
  transform: rotate(90deg);
  opacity: 0;
}

/* Dark theme - show moon */
[data-theme="dark"] .theme-toggle .moon {
  transform: rotate(0deg);
  opacity: 1;
}

[data-theme="dark"] .theme-toggle .sun {
  transform: rotate(-90deg);
  opacity: 0;
}
```

### Button Press

```css
.theme-toggle:active {
  transform: scale(0.95);
}
```

### Dropdown Animation

```css
.theme-dropdown {
  transform-origin: top right;
  transform: scale(0.95);
  opacity: 0;
  visibility: hidden;
  transition: transform 0.15s ease-out,
              opacity 0.15s ease-out,
              visibility 0s 0.15s;
}

.theme-dropdown--open {
  transform: scale(1);
  opacity: 1;
  visibility: visible;
  transition: transform 0.15s ease-out,
              opacity 0.15s ease-out;
}
```

### Page Theme Transition

```css
:root {
  --theme-transition: background-color 0.2s ease-out,
                      color 0.2s ease-out,
                      border-color 0.2s ease-out;
}

body {
  transition: var(--theme-transition);
}

/* Prevent transition on page load */
body.no-transition,
body.no-transition * {
  transition: none !important;
}
```

### Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  .theme-toggle .icon {
    transition: opacity 0.1s ease-out;
    transform: none !important;
  }
  
  .theme-dropdown {
    transition: opacity 0.1s ease-out;
    transform: scale(1) !important;
  }
  
  body {
    transition: none;
  }
}
```

---

## 9. Theme Persistence

### Local Storage

```typescript
const STORAGE_KEY = 'azd-app:theme';

// Save preference
function setTheme(theme: Theme): void {
  localStorage.setItem(STORAGE_KEY, theme);
  applyTheme(theme);
}

// Load preference
function getStoredTheme(): Theme | null {
  return localStorage.getItem(STORAGE_KEY) as Theme | null;
}

// Get effective theme (resolve 'system')
function getEffectiveTheme(stored: Theme | null): 'light' | 'dark' {
  if (stored === 'system' || stored === null) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches 
      ? 'dark' 
      : 'light';
  }
  return stored;
}
```

### Initialization Script (Prevent Flash)

```html
<script>
  // Run immediately in <head> to prevent flash
  (function() {
    const stored = localStorage.getItem('azd-app:theme');
    const theme = stored === 'dark' || 
      (stored !== 'light' && window.matchMedia('(prefers-color-scheme: dark)').matches)
      ? 'dark' 
      : 'light';
    document.documentElement.setAttribute('data-theme', theme);
    document.documentElement.style.colorScheme = theme;
  })();
</script>
```

### System Preference Listener

```typescript
// Listen for system preference changes
const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');

mediaQuery.addEventListener('change', (e) => {
  const stored = getStoredTheme();
  if (stored === 'system' || stored === null) {
    applyTheme(e.matches ? 'dark' : 'light');
  }
});
```

---

## 10. Apply Theme Function

```typescript
function applyTheme(theme: 'light' | 'dark'): void {
  // Temporarily disable transitions for instant switch on load
  const body = document.body;
  body.classList.add('no-transition');
  
  // Apply theme
  document.documentElement.setAttribute('data-theme', theme);
  document.documentElement.style.colorScheme = theme;
  
  // Update meta theme-color for mobile browsers
  const metaTheme = document.querySelector('meta[name="theme-color"]');
  if (metaTheme) {
    metaTheme.setAttribute('content', theme === 'dark' ? '#1e293b' : '#ffffff');
  }
  
  // Re-enable transitions
  requestAnimationFrame(() => {
    body.classList.remove('no-transition');
  });
  
  // Announce to screen readers
  announceThemeChange(theme);
}

function announceThemeChange(theme: string): void {
  const announcer = document.getElementById('theme-status');
  if (announcer) {
    announcer.textContent = `Theme changed to ${theme} mode`;
  }
}
```

---

## 11. Error Handling

| Scenario                | Behavior                                     |
|-------------------------|---------------------------------------------|
| localStorage unavailable| Use system preference, don't persist        |
| Invalid stored value    | Clear and use system preference             |
| matchMedia unavailable  | Default to light theme                      |
| JavaScript disabled     | Use CSS prefers-color-scheme only           |

### CSS-Only Fallback

```css
/* Works without JavaScript */
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
    /* Dark theme variables */
    --color-bg-primary: #0f172a;
    --color-text-primary: #f1f5f9;
    /* etc. */
  }
}
```

---

## 12. Implementation Notes

### Component Placement

- Header: Primary location, always visible
- Footer: Optional secondary placement
- Sidebar: Optional for documentation pages
- Mobile menu: Include in hamburger menu

### Performance

- Inline critical theme script in `<head>`
- Use CSS custom properties for theme colors
- Minimize JavaScript for theme switching
- Avoid layout shift on theme change

### Testing Checklist

- [ ] Toggle is keyboard accessible
- [ ] Focus visible on toggle button
- [ ] Screen reader announces theme changes
- [ ] Theme persists across page refreshes
- [ ] Theme persists across browser sessions
- [ ] System preference changes update theme
- [ ] No flash of wrong theme on page load
- [ ] Dropdown closes on Escape
- [ ] Dropdown closes on click outside
- [ ] Color contrast ≥ 4.5:1 in both themes
- [ ] Reduced motion respected
- [ ] Works with JavaScript disabled (system pref)
- [ ] Touch targets ≥ 44x44px on mobile
