# Light/Dark Mode Switcher Specification

## Overview
Add a light/dark mode switcher to the dashboard with a moon/sun icon toggle in the top right corner. User preference persisted in browser local storage.

## Functional Requirements

### 1. Theme Toggle UI
- **Location**: Top right corner of dashboard header
- **Icon Design**:
  - Light mode: Sun icon
  - Dark mode: Moon icon
  - Icon swaps based on current theme
- **Interaction**: Single click toggles between light and dark mode
- **Visual Feedback**: Icon changes immediately, smooth transition between themes

### 2. Theme Persistence
- **Storage**: Browser local storage
- **Key**: `dashboard-theme` (options: `light`, `dark`)
- **Default**: `light` mode on first visit
- **Loading**: Restore theme on page load from local storage
- **Scope**: Per-browser, per-device (localStorage is browser-specific)

### 3. Theme Application
- **Scope**: Apply theme to entire dashboard application
- **CSS Variables**: Use CSS custom properties for theme colors (light/dark variants)
- **Transitions**: Smooth color transitions when theme changes (400-500ms recommended)
- **System Preference**: Optional: Detect system theme preference as fallback if no local storage setting

### 4. Color Scheme

#### Light Mode
- Background: White/light gray
- Text: Dark gray/black
- Borders: Light gray
- Accents: Brand colors (maintained from current design)
- Logs pane: Light background with dark text
- Hover states: Subtle shadows and color shifts

#### Dark Mode
- Background: Dark gray/charcoal
- Text: Light gray/white
- Borders: Dark gray
- Accents: Brightened brand colors for contrast
- Logs pane: Dark background with light text
- Hover states: Subtle shadows and color shifts

### 5. Components Affected
- Header/navbar
- Logs pane
- Multi-pane grid
- Buttons and interactive elements
- Tables/data display areas
- Cards and panels
- Input fields and forms
- Scrollbars (if customized)

## Technical Requirements

### State Management
- Store current theme in React state
- Load from local storage on app initialization
- Sync local storage on theme change

### CSS Architecture
- Use CSS custom properties for theme colors
- Support both light and dark theme variants
- Apply to root element or dedicated theme provider
- Ensure specificity doesn't override base styles

### Accessibility (WCAG 2.1 AA)
- Icon button must have:
  - Accessible label describing the toggle action
  - Proper keyboard focus states
  - Keyboard navigation support (Enter/Space to activate)
- Sufficient color contrast in both themes (WCAG AA minimum 4.5:1)
- No color-only differentiation for critical information
- Reduced motion support: Respect user motion preferences
- Screen reader announces theme change

### Visual States
- **Default**: Current theme icon displayed
- **Hover**: Icon container background highlight
- **Active/Focus**: Focus ring around icon button
- **Disabled**: (N/A for this toggle)
- **Loading**: Smooth transition to new theme

## Acceptance Criteria

1. ✓ Moon/sun icon toggle visible in top right corner
2. ✓ Single click toggles between light and dark mode
3. ✓ Theme change applies immediately to all components
4. ✓ Theme preference saved to `dashboard-theme` in local storage
5. ✓ Theme preference restored on page reload
6. ✓ Smooth transition between themes (no jarring color changes)
7. ✓ Icon updates to reflect current theme
8. ✓ Keyboard accessible (Tab + Enter/Space)
9. ✓ Proper accessible labels for screen readers
10. ✓ Sufficient color contrast in both modes (WCAG AA)
11. ✓ All dashboard components properly styled in both themes
12. ✓ No layout shifts when switching themes

## Success Metrics

- User can toggle theme with single click
- Theme persists across page reloads
- All components visually compatible with both themes
- Accessibility tests pass (WCAG 2.1 AA)
- No console errors or warnings
- Smooth visual transitions

## Browser Support

- Modern browsers with CSS custom properties support
- localStorage support
- No polyfills required for MVP

## Future Enhancements

- System preference detection
- Scheduled theme changes (auto-dark at night)
- Additional theme options (e.g., sepia, high contrast)
- Theme preview before applying
- Keyboard shortcut for theme toggle
