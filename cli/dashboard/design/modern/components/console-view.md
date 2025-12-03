# Modern Console View Specification

> An immersive, feature-rich log viewing experience with multi-pane support and powerful filtering.

## Design Concept

The Modern console view is designed as a **focused workspace** that maximizes log visibility while providing quick access to filtering and navigation tools. It uses a dark-tinted color scheme even in light mode to reduce eye strain during extended log viewing.

---

## Layout Structure

### Console Container

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  [Toolbar: Actions | Filters | View Controls | Search]                       │
├─────────────────────────────────────────────────────────────────────────────┤
│  [Service Filters Bar]                                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│ ┌───────────────────────────────┐ ┌───────────────────────────────┐         │
│ │ Service A                   ▽│ │ Service B                   ▽│         │
│ ├───────────────────────────────┤ ├───────────────────────────────┤         │
│ │ > 10:42:31 Starting server... │ │ > 10:42:32 Connected to DB... │         │
│ │ > 10:42:32 Listening on :3000 │ │ > 10:42:33 Migration done     │         │
│ │ > 10:42:35 Request GET /api   │ │ > 10:42:34 Ready             │         │
│ │ > 10:42:36 Response 200 12ms  │ │                               │         │
│ │                               │ │                               │         │
│ └───────────────────────────────┘ └───────────────────────────────┘         │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Full Container Styles

```css
.modern-console {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--modern-console-bg);
  overflow: hidden;
}

/* Console-specific color scheme - darker for readability */
.modern-console {
  --modern-console-bg: #0d1117;
  --modern-console-surface: #161b22;
  --modern-console-border: #30363d;
  --modern-console-text: #c9d1d9;
  --modern-console-text-muted: #8b949e;
}

/* Light mode console - still uses dark palette */
[data-theme="modern-light"] .modern-console {
  --modern-console-bg: #1e2127;
  --modern-console-surface: #282c34;
  --modern-console-border: #3e4451;
  --modern-console-text: #abb2bf;
  --modern-console-text-muted: #5c6370;
}
```

---

## Toolbar

### Container

```css
.modern-console-toolbar {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-3) var(--space-4);
  background: var(--modern-console-surface);
  border-bottom: 1px solid var(--modern-console-border);
  flex-shrink: 0;
}

.modern-console-toolbar-section {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.modern-console-toolbar-divider {
  width: 1px;
  height: 24px;
  background: var(--modern-console-border);
}
```

### Toolbar Button

```css
.modern-console-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-2);
  font-size: var(--text-xs);
  font-weight: var(--font-medium);
  color: var(--modern-console-text-muted);
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.modern-console-btn:hover {
  color: var(--modern-console-text);
  background: rgba(255, 255, 255, 0.05);
}

.modern-console-btn--active {
  color: var(--modern-primary-dark);
  background: rgba(34, 211, 238, 0.1);
  border-color: rgba(34, 211, 238, 0.3);
}

.modern-console-btn-icon {
  width: 16px;
  height: 16px;
}

/* Primary variant */
.modern-console-btn--primary {
  background: var(--primary);
  color: var(--primary-foreground);
  border-color: transparent;
}

.modern-console-btn--primary:hover {
  background: var(--primary-hover);
  color: var(--primary-foreground);
}

/* Danger variant */
.modern-console-btn--danger:hover {
  color: var(--modern-error-dark);
  background: rgba(251, 113, 133, 0.1);
}

/* Disabled */
.modern-console-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
```

### Search Input

```css
.modern-console-search {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-3);
  background: rgba(0, 0, 0, 0.2);
  border: 1px solid var(--modern-console-border);
  border-radius: var(--radius-md);
  min-width: 200px;
  transition: all var(--duration-fast);
}

.modern-console-search:focus-within {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px rgba(34, 211, 238, 0.2);
}

.modern-console-search-icon {
  width: 16px;
  height: 16px;
  color: var(--modern-console-text-muted);
  flex-shrink: 0;
}

.modern-console-search-input {
  flex: 1;
  font-size: var(--text-sm);
  color: var(--modern-console-text);
  background: transparent;
  border: none;
  outline: none;
}

.modern-console-search-input::placeholder {
  color: var(--modern-console-text-muted);
}

.modern-console-search-clear {
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--modern-console-text-muted);
  cursor: pointer;
  border-radius: var(--radius-sm);
}

.modern-console-search-clear:hover {
  color: var(--modern-console-text);
  background: rgba(255, 255, 255, 0.1);
}
```

### Grid Column Selector

```css
.modern-console-columns {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1);
  background: rgba(0, 0, 0, 0.2);
  border-radius: var(--radius-md);
}

.modern-console-column-btn {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  color: var(--modern-console-text-muted);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.modern-console-column-btn:hover {
  color: var(--modern-console-text);
  background: rgba(255, 255, 255, 0.1);
}

.modern-console-column-btn--active {
  color: var(--modern-primary-dark);
  background: rgba(34, 211, 238, 0.15);
}
```

---

## Filter Bar

```css
.modern-console-filters {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-6);
  padding: var(--space-3) var(--space-4);
  background: var(--modern-console-surface);
  border-bottom: 1px solid var(--modern-console-border);
}

.modern-console-filter-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.modern-console-filter-label {
  font-size: var(--text-xs);
  font-weight: var(--font-medium);
  color: var(--modern-console-text-muted);
}

.modern-console-filter-options {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

/* Filter checkbox */
.modern-console-filter-checkbox {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  cursor: pointer;
}

.modern-console-filter-checkbox input {
  width: 14px;
  height: 14px;
  accent-color: var(--primary);
}

.modern-console-filter-checkbox span {
  font-size: var(--text-xs);
  color: var(--modern-console-text);
}

/* Level-specific colors */
.modern-console-filter-checkbox--info span {
  color: var(--modern-info-dark);
}

.modern-console-filter-checkbox--warning span {
  color: var(--modern-warning-dark);
}

.modern-console-filter-checkbox--error span {
  color: var(--modern-error-dark);
}

/* Health status colors */
.modern-console-filter-checkbox--healthy span {
  color: var(--modern-success-dark);
}

.modern-console-filter-checkbox--degraded span {
  color: var(--modern-warning-dark);
}

.modern-console-filter-checkbox--unhealthy span {
  color: var(--modern-error-dark);
}
```

---

## Log Pane Grid

```css
.modern-console-grid {
  flex: 1;
  display: grid;
  gap: var(--space-2);
  padding: var(--space-2);
  overflow: hidden;
}

/* Column variants */
.modern-console-grid--1 {
  grid-template-columns: 1fr;
}

.modern-console-grid--2 {
  grid-template-columns: repeat(2, 1fr);
}

.modern-console-grid--3 {
  grid-template-columns: repeat(3, 1fr);
}

.modern-console-grid--4 {
  grid-template-columns: repeat(4, 1fr);
}

@media (max-width: 768px) {
  .modern-console-grid {
    grid-template-columns: 1fr !important;
  }
}
```

---

## Log Pane

### Container

```css
.modern-log-pane {
  display: flex;
  flex-direction: column;
  background: var(--modern-console-surface);
  border: 1px solid var(--modern-console-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
  min-height: 200px;
}

/* Collapsed state */
.modern-log-pane--collapsed {
  min-height: 0;
}

.modern-log-pane--collapsed .modern-log-pane-content {
  display: none;
}
```

### Pane Header

```css
.modern-log-pane-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--modern-console-border);
  cursor: pointer;
  transition: background var(--duration-fast);
}

.modern-log-pane-header:hover {
  background: rgba(255, 255, 255, 0.02);
}

/* Status-based header backgrounds */
.modern-log-pane--error .modern-log-pane-header {
  background: rgba(251, 113, 133, 0.08);
  border-bottom-color: rgba(251, 113, 133, 0.2);
}

.modern-log-pane--warning .modern-log-pane-header {
  background: rgba(251, 191, 36, 0.08);
  border-bottom-color: rgba(251, 191, 36, 0.2);
}

.modern-log-pane--healthy .modern-log-pane-header {
  background: rgba(52, 211, 153, 0.05);
}

/* Header elements */
.modern-log-pane-status {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

.modern-log-pane-title {
  flex: 1;
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--modern-console-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modern-log-pane-meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-xs);
  color: var(--modern-console-text-muted);
}

.modern-log-pane-chevron {
  width: 16px;
  height: 16px;
  color: var(--modern-console-text-muted);
  transition: transform var(--duration-fast);
}

.modern-log-pane--collapsed .modern-log-pane-chevron {
  transform: rotate(-90deg);
}
```

### Pane Actions (on hover)

```css
.modern-log-pane-actions {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  opacity: 0;
  transition: opacity var(--duration-fast);
}

.modern-log-pane-header:hover .modern-log-pane-actions {
  opacity: 1;
}

.modern-log-pane-action {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--modern-console-text-muted);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.modern-log-pane-action:hover {
  color: var(--modern-console-text);
  background: rgba(255, 255, 255, 0.1);
}

.modern-log-pane-action svg {
  width: 14px;
  height: 14px;
}
```

### Log Content

```css
.modern-log-pane-content {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  font-family: var(--modern-font-mono);
  font-size: 12px;
  line-height: 1.6;
}

/* Scrollbar styling */
.modern-log-pane-content::-webkit-scrollbar {
  width: 8px;
}

.modern-log-pane-content::-webkit-scrollbar-track {
  background: transparent;
}

.modern-log-pane-content::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-full);
}

.modern-log-pane-content::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.2);
}
```

---

## Log Entry

```css
.modern-log-entry {
  display: flex;
  padding: var(--space-0.5) var(--space-3);
  transition: background var(--duration-fast);
}

.modern-log-entry:hover {
  background: rgba(255, 255, 255, 0.02);
}

/* Level-based left border */
.modern-log-entry--info {
  border-left: 2px solid transparent;
}

.modern-log-entry--warning {
  border-left: 2px solid var(--modern-warning-dark);
  background: rgba(251, 191, 36, 0.05);
}

.modern-log-entry--error {
  border-left: 2px solid var(--modern-error-dark);
  background: rgba(251, 113, 133, 0.05);
}

/* Timestamp */
.modern-log-timestamp {
  flex-shrink: 0;
  width: 80px;
  color: var(--modern-console-text-muted);
  font-size: 11px;
}

/* Message */
.modern-log-message {
  flex: 1;
  color: var(--modern-console-text);
  word-break: break-word;
  white-space: pre-wrap;
}

/* Search highlight */
.modern-log-highlight {
  background: rgba(34, 211, 238, 0.3);
  color: white;
  padding: 0 2px;
  border-radius: 2px;
}

/* Level badge (optional) */
.modern-log-level {
  flex-shrink: 0;
  width: 50px;
  font-size: 10px;
  font-weight: var(--font-semibold);
  text-transform: uppercase;
}

.modern-log-level--info {
  color: var(--modern-info-dark);
}

.modern-log-level--warning {
  color: var(--modern-warning-dark);
}

.modern-log-level--error {
  color: var(--modern-error-dark);
}
```

---

## Unified View Mode

Alternative single-stream view.

```css
.modern-console-unified {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modern-console-unified-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-2);
  background: var(--modern-console-bg);
}

/* Unified log entry with service indicator */
.modern-log-entry-unified {
  display: flex;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-3);
}

.modern-log-service-badge {
  flex-shrink: 0;
  padding: var(--space-0.5) var(--space-2);
  font-size: 10px;
  font-weight: var(--font-semibold);
  background: rgba(255, 255, 255, 0.05);
  border-radius: var(--radius-sm);
  color: var(--modern-console-text-muted);
  min-width: 80px;
  text-align: center;
}

/* Service color coding */
.modern-log-service-badge[data-service="api"] {
  background: rgba(34, 211, 238, 0.15);
  color: var(--modern-primary-dark);
}

.modern-log-service-badge[data-service="web"] {
  background: rgba(52, 211, 153, 0.15);
  color: var(--modern-success-dark);
}

.modern-log-service-badge[data-service="worker"] {
  background: rgba(251, 191, 36, 0.15);
  color: var(--modern-warning-dark);
}
```

---

## Fullscreen Mode

```css
.modern-console--fullscreen {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal);
  border-radius: 0;
}

.modern-console--fullscreen .modern-console-toolbar {
  padding: var(--space-3) var(--space-6);
}

.modern-console--fullscreen .modern-console-grid {
  padding: var(--space-4);
}
```

---

## Empty State

```css
.modern-console-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-4);
  color: var(--modern-console-text-muted);
}

.modern-console-empty-icon {
  width: 48px;
  height: 48px;
  opacity: 0.5;
}

.modern-console-empty-title {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--modern-console-text);
}

.modern-console-empty-message {
  font-size: var(--text-sm);
  text-align: center;
  max-width: 400px;
}
```

---

## Paused Indicator

```css
.modern-console-paused-badge {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-3);
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  color: var(--modern-warning-dark);
  background: rgba(251, 191, 36, 0.15);
  border-radius: var(--radius-full);
}

.modern-console-paused-badge svg {
  width: 14px;
  height: 14px;
}
```

---

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Space` | Toggle pause |
| `Ctrl+Shift+L` | Toggle view mode |
| `Ctrl+Shift+F` / `F11` | Toggle fullscreen |
| `Escape` | Exit fullscreen |
| `Ctrl+F` | Focus search |
| `1-6` | Set grid columns |

---

## Accessibility

- All interactive elements are keyboard accessible
- Log entries use semantic markup
- Level indicators use text labels, not just color
- Search results announced to screen readers
- Fullscreen mode maintains focus
- High contrast in log viewer (dark background)

```jsx
<div 
  role="log"
  aria-live="polite"
  aria-label={`Logs for ${serviceName}`}
  className="modern-log-pane-content"
>
  {logs.map(log => (
    <div 
      key={log.id}
      className={`modern-log-entry modern-log-entry--${log.level}`}
      aria-label={`${log.level}: ${log.message}`}
    >
      <time className="modern-log-timestamp">{log.timestamp}</time>
      <span className="modern-log-message">{log.message}</span>
    </div>
  ))}
</div>
```
