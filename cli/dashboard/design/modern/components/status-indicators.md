# Modern Status Indicators Specification

> A comprehensive visual language for communicating service health, status, and real-time state changes.

## Design Philosophy

Status indicators in the Modern theme are designed to be **informative without being distracting**. They use a combination of:

- **Color** for quick visual scanning
- **Shape** for accessibility (not relying on color alone)
- **Animation** for conveying real-time state
- **Text** for clarity

---

## Status States

### Process Status

| State | Description | Color | Animation |
|-------|-------------|-------|-----------|
| Running | Process is active | Teal/Success | Gentle pulse |
| Starting | Process is initializing | Blue/Info | Expanding pulse |
| Stopping | Process is shutting down | Amber/Warning | Fade pulse |
| Stopped | Process is not running | Gray | None |
| Error | Process failed | Rose/Error | Alert flash |
| Restarting | Process is restarting | Blue/Info | Spinning |

### Health Status

| State | Description | Color | Animation |
|-------|-------------|-------|-----------|
| Healthy | All checks pass | Emerald | Heartbeat |
| Degraded | Partial/slow response | Amber | Breathing |
| Unhealthy | Checks failing | Rose | Rapid flash |
| Unknown | No health data | Gray | None |
| Starting | Initial checks | Blue | Wave pulse |

---

## Visual Treatments

### Status Dot (Minimal)

The simplest indicator - a colored circle.

```css
.modern-status-dot {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

/* Status colors */
.modern-status-dot--running,
.modern-status-dot--healthy {
  background: var(--modern-success-light);
}

.modern-status-dot--starting {
  background: var(--modern-info-light);
}

.modern-status-dot--degraded,
.modern-status-dot--stopping {
  background: var(--modern-warning-light);
}

.modern-status-dot--error,
.modern-status-dot--unhealthy {
  background: var(--modern-error-light);
}

.modern-status-dot--stopped,
.modern-status-dot--unknown {
  background: var(--foreground-muted);
}

/* Dark mode adjustments */
[data-theme="modern-dark"] .modern-status-dot--running,
[data-theme="modern-dark"] .modern-status-dot--healthy {
  background: var(--modern-success-dark);
}

[data-theme="modern-dark"] .modern-status-dot--starting {
  background: var(--modern-info-dark);
}

[data-theme="modern-dark"] .modern-status-dot--degraded {
  background: var(--modern-warning-dark);
}

[data-theme="modern-dark"] .modern-status-dot--error,
[data-theme="modern-dark"] .modern-status-dot--unhealthy {
  background: var(--modern-error-dark);
}
```

### Animated Status Dot

```css
/* Gentle heartbeat - Healthy/Running */
.modern-status-dot--animated-heartbeat {
  animation: modern-heartbeat 2s ease-in-out infinite;
}

@keyframes modern-heartbeat {
  0%, 100% {
    transform: scale(1);
    opacity: 0.9;
  }
  15% {
    transform: scale(1.2);
    opacity: 1;
  }
  30%, 100% {
    transform: scale(1);
    opacity: 0.9;
  }
}

/* Wave pulse - Starting */
.modern-status-dot--animated-pulse {
  animation: modern-wave-pulse 1.5s ease-out infinite;
}

@keyframes modern-wave-pulse {
  0% {
    box-shadow: 0 0 0 0 currentColor;
    opacity: 1;
  }
  70% {
    box-shadow: 0 0 0 8px transparent;
    opacity: 0.7;
  }
  100% {
    box-shadow: 0 0 0 0 transparent;
    opacity: 1;
  }
}

/* Breathing - Degraded */
.modern-status-dot--animated-breathe {
  animation: modern-breathe 2.5s ease-in-out infinite;
}

@keyframes modern-breathe {
  0%, 100% {
    transform: scale(1);
    opacity: 0.8;
  }
  50% {
    transform: scale(1.15);
    opacity: 1;
  }
}

/* Alert flash - Error/Unhealthy */
.modern-status-dot--animated-flash {
  animation: modern-alert-flash 1s ease-in-out infinite;
}

@keyframes modern-alert-flash {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.3;
  }
}

/* Spin - Restarting */
.modern-status-dot--animated-spin {
  animation: modern-spin 1s linear infinite;
}
```

---

### Status Badge (With Label)

```css
.modern-status-badge {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-2);
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  border-radius: var(--radius-full);
  white-space: nowrap;
}

/* Variants */
.modern-status-badge--running,
.modern-status-badge--healthy {
  background: var(--modern-success-bg-light);
  color: var(--modern-success-light);
}

.modern-status-badge--starting {
  background: var(--modern-info-bg-light);
  color: var(--modern-info-light);
}

.modern-status-badge--degraded,
.modern-status-badge--stopping {
  background: var(--modern-warning-bg-light);
  color: var(--modern-warning-light);
}

.modern-status-badge--error,
.modern-status-badge--unhealthy {
  background: var(--modern-error-bg-light);
  color: var(--modern-error-light);
}

.modern-status-badge--stopped,
.modern-status-badge--unknown {
  background: var(--modern-bg-tertiary);
  color: var(--foreground-muted);
}

/* Dark mode */
[data-theme="modern-dark"] .modern-status-badge--running,
[data-theme="modern-dark"] .modern-status-badge--healthy {
  background: var(--modern-success-bg-dark);
  color: var(--modern-success-dark);
}

[data-theme="modern-dark"] .modern-status-badge--starting {
  background: var(--modern-info-bg-dark);
  color: var(--modern-info-dark);
}

[data-theme="modern-dark"] .modern-status-badge--degraded {
  background: var(--modern-warning-bg-dark);
  color: var(--modern-warning-dark);
}

[data-theme="modern-dark"] .modern-status-badge--error,
[data-theme="modern-dark"] .modern-status-badge--unhealthy {
  background: var(--modern-error-bg-dark);
  color: var(--modern-error-dark);
}

[data-theme="modern-dark"] .modern-status-badge--stopped,
[data-theme="modern-dark"] .modern-status-badge--unknown {
  background: var(--modern-bg-tertiary);
  color: var(--foreground-muted);
}
```

---

### Status Indicator with Icon

```css
.modern-status-indicator {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
}

.modern-status-indicator-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.modern-status-indicator-text {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
}

/* Icon animations */
.modern-status-indicator--starting .modern-status-indicator-icon {
  animation: modern-spin 1s linear infinite;
}

.modern-status-indicator--stopping .modern-status-indicator-icon {
  animation: modern-fade-pulse 1.5s ease-in-out infinite;
}

@keyframes modern-fade-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
```

---

### Health Summary Pill

Aggregate status indicator for the header.

```css
.modern-health-pill {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-3);
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
}

.modern-health-pill:hover {
  box-shadow: var(--shadow-sm);
}

/* All healthy */
.modern-health-pill--healthy {
  background: var(--modern-success-bg-light);
  color: var(--modern-success-light);
}

/* Some degraded */
.modern-health-pill--degraded {
  background: var(--modern-warning-bg-light);
  color: var(--modern-warning-light);
}

/* Some unhealthy */
.modern-health-pill--unhealthy {
  background: var(--modern-error-bg-light);
  color: var(--modern-error-light);
}

/* Disconnected */
.modern-health-pill--disconnected {
  background: var(--modern-bg-tertiary);
  color: var(--foreground-muted);
}

/* Content structure */
.modern-health-pill-dot {
  width: 6px;
  height: 6px;
  border-radius: var(--radius-full);
  background: currentColor;
}

.modern-health-pill--healthy .modern-health-pill-dot {
  animation: modern-heartbeat 2s ease-in-out infinite;
}

.modern-health-pill--unhealthy .modern-health-pill-dot {
  animation: modern-alert-flash 1s ease-in-out infinite;
}

.modern-health-pill-count {
  font-variant-numeric: tabular-nums;
}

.modern-health-pill-chevron {
  width: 14px;
  height: 14px;
  opacity: 0.6;
  transition: transform var(--duration-fast);
}

.modern-health-pill[aria-expanded="true"] .modern-health-pill-chevron {
  transform: rotate(180deg);
}
```

---

### Connection Status Indicator

```css
.modern-connection-status {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-2);
  font-size: var(--text-xs);
  border-radius: var(--radius-full);
}

/* Connected */
.modern-connection-status--connected {
  color: var(--modern-success-light);
}

/* Reconnecting */
.modern-connection-status--reconnecting {
  color: var(--modern-warning-light);
}

.modern-connection-status--reconnecting .modern-connection-icon {
  animation: modern-spin 1.5s linear infinite;
}

/* Disconnected */
.modern-connection-status--disconnected {
  color: var(--modern-error-light);
}

.modern-connection-icon {
  width: 14px;
  height: 14px;
}
```

---

### Progress/Loading Indicators

```css
/* Indeterminate progress bar */
.modern-progress-indeterminate {
  height: 2px;
  background: var(--modern-bg-tertiary);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.modern-progress-indeterminate::after {
  content: '';
  display: block;
  width: 40%;
  height: 100%;
  background: var(--primary);
  border-radius: var(--radius-full);
  animation: modern-progress-slide 1.5s ease-in-out infinite;
}

@keyframes modern-progress-slide {
  0% { transform: translateX(-100%); }
  50% { transform: translateX(150%); }
  100% { transform: translateX(300%); }
}

/* Skeleton loader */
.modern-skeleton {
  background: linear-gradient(
    90deg,
    var(--modern-bg-tertiary) 0%,
    var(--modern-bg-secondary) 50%,
    var(--modern-bg-tertiary) 100%
  );
  background-size: 200% 100%;
  animation: modern-shimmer 1.5s ease-in-out infinite;
  border-radius: var(--radius-sm);
}

@keyframes modern-shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* Spinner */
.modern-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--modern-bg-tertiary);
  border-top-color: var(--primary);
  border-radius: var(--radius-full);
  animation: modern-spin 0.8s linear infinite;
}

.modern-spinner--sm {
  width: 14px;
  height: 14px;
  border-width: 1.5px;
}

.modern-spinner--lg {
  width: 32px;
  height: 32px;
  border-width: 3px;
}
```

---

### Status Change Transition

When status changes, use a brief flash animation.

```css
.modern-status-transition {
  animation: modern-status-change 600ms ease-out;
}

@keyframes modern-status-change {
  0% {
    transform: scale(1);
  }
  25% {
    transform: scale(1.3);
    filter: brightness(1.2);
  }
  50% {
    transform: scale(0.9);
  }
  100% {
    transform: scale(1);
    filter: brightness(1);
  }
}
```

---

## Icon Reference

### Status Icons (Lucide)

| Status | Icon | Name |
|--------|------|------|
| Running/Healthy | ✓ circle | `CheckCircle` |
| Starting | ◷ | `Loader2` |
| Stopping | ⏹ | `Square` |
| Stopped | ◯ | `Circle` |
| Error/Unhealthy | ✕ circle | `XCircle` |
| Degraded | ⚠ | `AlertTriangle` |
| Unknown | ? circle | `HelpCircle` |
| Restarting | ↻ | `RefreshCw` |

### Health Check Type Icons

| Type | Icon | Name |
|------|------|------|
| HTTP | 🌐 | `Globe` |
| Port | 🔌 | `Plug` |
| Process | ⚡ | `Cpu` |

---

## Accessibility Considerations

### Color Independence

Status is never communicated by color alone:

```jsx
<div className="modern-status-indicator">
  <CheckCircle className="modern-status-indicator-icon" aria-hidden="true" />
  <span className="modern-status-dot modern-status-dot--healthy" aria-hidden="true" />
  <span className="modern-status-indicator-text">Running</span>
  <span className="sr-only">Service is running and healthy</span>
</div>
```

### Motion Preferences

```css
@media (prefers-reduced-motion: reduce) {
  .modern-status-dot--animated-heartbeat,
  .modern-status-dot--animated-pulse,
  .modern-status-dot--animated-breathe,
  .modern-status-dot--animated-flash,
  .modern-status-dot--animated-spin {
    animation: none;
  }
  
  .modern-spinner {
    animation-duration: 1.5s;
  }
  
  .modern-progress-indeterminate::after {
    animation-duration: 3s;
  }
}
```

### Tooltips for Context

```jsx
<span 
  className="modern-status-dot modern-status-dot--degraded modern-status-dot--animated-breathe"
  title="Service responding slowly - avg response time 2.3s"
  role="img"
  aria-label="Degraded"
/>
```

### Focus Indicators

```css
.modern-status-badge:focus-visible,
.modern-health-pill:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}
```

---

## Component Composition

```tsx
interface StatusIndicatorProps {
  status: ProcessStatus;
  health?: HealthStatus;
  variant?: 'dot' | 'badge' | 'full';
  animated?: boolean;
  showLabel?: boolean;
}

export function StatusIndicator({
  status,
  health,
  variant = 'dot',
  animated = true,
  showLabel = false
}: StatusIndicatorProps) {
  const effectiveStatus = getEffectiveStatus(status, health);
  const { color, text, icon: Icon, animation } = STATUS_CONFIG[effectiveStatus];
  
  if (variant === 'dot') {
    return (
      <span
        className={cn(
          'modern-status-dot',
          `modern-status-dot--${effectiveStatus}`,
          animated && animation && `modern-status-dot--animated-${animation}`
        )}
        role="img"
        aria-label={text}
        title={text}
      />
    );
  }
  
  if (variant === 'badge') {
    return (
      <span className={cn('modern-status-badge', `modern-status-badge--${effectiveStatus}`)}>
        <span className="modern-status-dot" aria-hidden="true" />
        <span>{text}</span>
      </span>
    );
  }
  
  return (
    <div className={cn('modern-status-indicator', `modern-status-indicator--${effectiveStatus}`)}>
      <Icon className="modern-status-indicator-icon" aria-hidden="true" />
      <span className="modern-status-indicator-text">{text}</span>
    </div>
  );
}
```

---

## Status Configuration Map

```tsx
const STATUS_CONFIG = {
  running: {
    color: 'success',
    text: 'Running',
    icon: CheckCircle,
    animation: 'heartbeat'
  },
  healthy: {
    color: 'success',
    text: 'Healthy',
    icon: CheckCircle,
    animation: 'heartbeat'
  },
  starting: {
    color: 'info',
    text: 'Starting',
    icon: Loader2,
    animation: 'pulse'
  },
  stopping: {
    color: 'warning',
    text: 'Stopping',
    icon: Square,
    animation: 'breathe'
  },
  stopped: {
    color: 'muted',
    text: 'Stopped',
    icon: Circle,
    animation: null
  },
  degraded: {
    color: 'warning',
    text: 'Degraded',
    icon: AlertTriangle,
    animation: 'breathe'
  },
  error: {
    color: 'error',
    text: 'Error',
    icon: XCircle,
    animation: 'flash'
  },
  unhealthy: {
    color: 'error',
    text: 'Unhealthy',
    icon: XCircle,
    animation: 'flash'
  },
  unknown: {
    color: 'muted',
    text: 'Unknown',
    icon: HelpCircle,
    animation: null
  },
  restarting: {
    color: 'info',
    text: 'Restarting',
    icon: RefreshCw,
    animation: 'spin'
  }
} as const;
```
