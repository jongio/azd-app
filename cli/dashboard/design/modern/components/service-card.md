# Modern Service Card Specification

> Compact, information-dense cards for the grid view that balance visual appeal with functional clarity.

## Design Concept

The Modern service card is a **layered card with depth** that uses subtle gradients, refined shadows, and strategic color accents to communicate status at a glance. The design prioritizes scannability while providing rich interaction affordances.

---

## Visual Design

### Anatomy

```
┌──────────────────────────────────────────────────────────┐
│  ┌──────┐                                                │
│  │ Icon │  Service Name                    [ Status ]   │
│  └──────┘  Language • Framework                          │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  [ Start ]  [ Stop ]  [ Restart ]  [ Open ↗ ]           │
│                                                          │
├──────────────────────────────────────────────────────────┤
│  Port: 3000        Response: 12ms        Uptime: 2h 15m │
│                                                          │
│  http://localhost:3000/                           [ ↗ ] │
└──────────────────────────────────────────────────────────┘
```

### Dimensions

| Property | Value |
|----------|-------|
| Min Width | 280px |
| Max Width | 400px |
| Padding | 20px |
| Border Radius | 16px |
| Gap (internal) | 16px |

---

## Card Container

### Base Style

```css
.modern-service-card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-5);
  background: var(--modern-bg-primary);
  border: 1px solid var(--modern-border-default);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-sm);
  transition: all var(--duration-normal) var(--ease-out);
  cursor: pointer;
  overflow: hidden;
}

/* Subtle gradient overlay */
.modern-service-card::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(
    135deg,
    rgba(var(--modern-primary-rgb), 0.02) 0%,
    transparent 50%,
    rgba(var(--modern-primary-rgb), 0.02) 100%
  );
  opacity: 0;
  transition: opacity var(--duration-normal);
  pointer-events: none;
}
```

### Hover State

```css
.modern-service-card:hover {
  border-color: var(--modern-border-strong);
  box-shadow: var(--shadow-lg);
  transform: translateY(-2px);
}

.modern-service-card:hover::before {
  opacity: 1;
}
```

### Active/Press State

```css
.modern-service-card:active {
  transform: translateY(0);
  box-shadow: var(--shadow-sm);
}
```

### Focus State

```css
.modern-service-card:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}
```

---

## Card Header

### Layout

```css
.modern-card-header {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
}
```

### Service Icon

```css
.modern-card-icon {
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-lg);
  background: var(--modern-bg-tertiary);
  flex-shrink: 0;
  transition: all var(--duration-normal);
}

.modern-card-icon svg {
  width: 22px;
  height: 22px;
  color: var(--foreground-tertiary);
}

/* Status-based icon styling */
.modern-card--running .modern-card-icon {
  background: linear-gradient(135deg, 
    rgba(var(--modern-success-rgb), 0.15), 
    rgba(var(--modern-success-rgb), 0.05)
  );
}

.modern-card--running .modern-card-icon svg {
  color: var(--modern-success-light);
}

.modern-card--error .modern-card-icon {
  background: linear-gradient(135deg, 
    rgba(var(--modern-error-rgb), 0.15), 
    rgba(var(--modern-error-rgb), 0.05)
  );
}

.modern-card--error .modern-card-icon svg {
  color: var(--modern-error-light);
}
```

### Service Info

```css
.modern-card-info {
  flex: 1;
  min-width: 0;
}

.modern-card-title {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--foreground);
  line-height: 1.3;
  margin-bottom: var(--space-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modern-card-subtitle {
  font-size: var(--text-xs);
  color: var(--foreground-muted);
}

.modern-card-subtitle-divider {
  margin: 0 var(--space-1);
  color: var(--modern-border-strong);
}
```

### Status Badge

```css
.modern-card-status {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-2);
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

/* Running/Healthy */
.modern-card-status--running {
  background: var(--modern-success-bg-light);
  color: var(--modern-success-light);
}

/* Starting */
.modern-card-status--starting {
  background: var(--modern-info-bg-light);
  color: var(--modern-info-light);
}

/* Warning/Degraded */
.modern-card-status--degraded {
  background: var(--modern-warning-bg-light);
  color: var(--modern-warning-light);
}

/* Error/Unhealthy */
.modern-card-status--error {
  background: var(--modern-error-bg-light);
  color: var(--modern-error-light);
}

/* Stopped */
.modern-card-status--stopped {
  background: var(--modern-bg-tertiary);
  color: var(--foreground-muted);
}
```

### Status Indicator Dot

```css
.modern-card-status-dot {
  width: 6px;
  height: 6px;
  border-radius: var(--radius-full);
  background: currentColor;
}

.modern-card-status--running .modern-card-status-dot {
  animation: modern-breathe 2s ease-in-out infinite;
}

.modern-card-status--starting .modern-card-status-dot {
  animation: modern-pulse-glow 1.5s ease-in-out infinite;
}

.modern-card-status--error .modern-card-status-dot {
  animation: modern-alert-flash 1s ease-in-out infinite;
}
```

---

## Alert Banner

Shown when service has an error or is degraded.

```css
.modern-card-alert {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
  padding: var(--space-3);
  border-radius: var(--radius-md);
  font-size: var(--text-xs);
  line-height: 1.4;
}

.modern-card-alert--error {
  background: var(--modern-error-bg-light);
  border: 1px solid rgba(var(--modern-error-rgb), 0.2);
  color: var(--modern-error-light);
}

.modern-card-alert--warning {
  background: var(--modern-warning-bg-light);
  border: 1px solid rgba(var(--modern-warning-rgb), 0.2);
  color: var(--modern-warning-light);
}

.modern-card-alert-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  margin-top: 1px;
}

.modern-card-alert-message {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
```

---

## Action Buttons

### Container

```css
.modern-card-actions {
  display: flex;
  gap: var(--space-2);
}
```

### Action Button

```css
.modern-card-action {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-1);
  padding: var(--space-2);
  font-size: var(--text-xs);
  font-weight: var(--font-medium);
  color: var(--foreground-secondary);
  background: var(--modern-bg-tertiary);
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
}

.modern-card-action:hover {
  background: var(--modern-bg-secondary);
  border-color: var(--modern-border-default);
  color: var(--foreground);
}

.modern-card-action:active {
  transform: scale(0.98);
}

.modern-card-action-icon {
  width: 14px;
  height: 14px;
}

/* Primary action variant */
.modern-card-action--primary {
  background: var(--primary);
  color: var(--primary-foreground);
}

.modern-card-action--primary:hover {
  background: var(--primary-hover);
  border-color: transparent;
  color: var(--primary-foreground);
}

/* Danger action variant */
.modern-card-action--danger:hover {
  background: var(--modern-error-bg-light);
  border-color: rgba(var(--modern-error-rgb), 0.3);
  color: var(--modern-error-light);
}

/* Disabled state */
.modern-card-action:disabled {
  opacity: 0.4;
  cursor: not-allowed;
  pointer-events: none;
}
```

### Icon-only Action

```css
.modern-card-action--icon-only {
  flex: 0 0 auto;
  width: 32px;
  padding: 0;
}
```

---

## Metrics Row

### Container

```css
.modern-card-metrics {
  display: flex;
  gap: var(--space-4);
  padding: var(--space-3) 0;
  border-top: 1px solid var(--modern-border-subtle);
}
```

### Metric Item

```css
.modern-card-metric {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--space-0.5);
}

.modern-card-metric-label {
  font-size: 10px;
  font-weight: var(--font-medium);
  color: var(--foreground-muted);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wider);
}

.modern-card-metric-value {
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--foreground);
  font-variant-numeric: tabular-nums;
}

/* Mono style for numerical values */
.modern-card-metric-value--mono {
  font-family: var(--modern-font-mono);
}
```

---

## URL Section

```css
.modern-card-url {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  background: var(--modern-bg-tertiary);
  border-radius: var(--radius-md);
  text-decoration: none;
  transition: all var(--duration-fast) var(--ease-out);
}

.modern-card-url:hover {
  background: var(--modern-bg-secondary);
}

.modern-card-url-text {
  flex: 1;
  font-size: var(--text-xs);
  font-family: var(--modern-font-mono);
  color: var(--foreground-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modern-card-url-icon {
  width: 14px;
  height: 14px;
  color: var(--foreground-muted);
  flex-shrink: 0;
  transition: transform var(--duration-fast);
}

.modern-card-url:hover .modern-card-url-icon {
  transform: translate(2px, -2px);
  color: var(--primary);
}

/* Azure URL variant */
.modern-card-url--azure {
  background: linear-gradient(
    135deg,
    rgba(var(--modern-primary-rgb), 0.08),
    rgba(var(--modern-primary-rgb), 0.04)
  );
  border: 1px solid rgba(var(--modern-primary-rgb), 0.15);
}

.modern-card-url--azure:hover {
  border-color: var(--primary);
}
```

---

## Dark Mode Adjustments

```css
[data-theme="modern-dark"] .modern-service-card {
  background: var(--modern-bg-secondary);
  border-color: var(--modern-border-default);
  box-shadow: var(--shadow-sm-dark);
}

[data-theme="modern-dark"] .modern-service-card:hover {
  box-shadow: var(--shadow-lg-dark);
  border-color: var(--modern-border-strong);
}

[data-theme="modern-dark"] .modern-card-icon {
  background: var(--modern-bg-tertiary);
}

[data-theme="modern-dark"] .modern-card-action {
  background: var(--modern-bg-tertiary);
}

[data-theme="modern-dark"] .modern-card-action:hover {
  background: var(--modern-bg-elevated);
}
```

---

## Loading State

```css
.modern-service-card--loading {
  pointer-events: none;
}

.modern-service-card--loading .modern-card-title,
.modern-service-card--loading .modern-card-subtitle,
.modern-service-card--loading .modern-card-metric-value {
  background: linear-gradient(
    90deg,
    var(--modern-bg-tertiary) 0%,
    var(--modern-bg-secondary) 50%,
    var(--modern-bg-tertiary) 100%
  );
  background-size: 200% 100%;
  animation: modern-shimmer 1.5s infinite;
  color: transparent;
  border-radius: var(--radius-sm);
}
```

---

## Component Structure

```tsx
interface ModernServiceCardProps {
  service: Service;
  healthStatus?: HealthCheckResult;
  onClick?: () => void;
}

export function ModernServiceCard({ 
  service, 
  healthStatus, 
  onClick 
}: ModernServiceCardProps) {
  const status = getEffectiveStatus(service, healthStatus);
  const hasError = service.error || healthStatus?.error;
  
  return (
    <article
      className={cn(
        'modern-service-card',
        `modern-card--${status}`,
        hasError && 'modern-card--has-error'
      )}
      onClick={onClick}
      role="button"
      tabIndex={0}
      aria-label={`${service.name} service - ${status}`}
    >
      {/* Header */}
      <header className="modern-card-header">
        <div className="modern-card-icon">
          <Server />
        </div>
        <div className="modern-card-info">
          <h3 className="modern-card-title">{service.name}</h3>
          <p className="modern-card-subtitle">
            {service.language}
            <span className="modern-card-subtitle-divider">•</span>
            {service.framework}
          </p>
        </div>
        <StatusBadge status={status} health={healthStatus?.status} />
      </header>
      
      {/* Error Alert */}
      {hasError && (
        <div className="modern-card-alert modern-card-alert--error">
          <XCircle className="modern-card-alert-icon" />
          <span className="modern-card-alert-message">
            {service.error || healthStatus?.error}
          </span>
        </div>
      )}
      
      {/* Actions */}
      <div className="modern-card-actions" onClick={e => e.stopPropagation()}>
        <ServiceActions service={service} variant="card" />
      </div>
      
      {/* Metrics */}
      <div className="modern-card-metrics">
        <div className="modern-card-metric">
          <span className="modern-card-metric-label">Port</span>
          <span className="modern-card-metric-value modern-card-metric-value--mono">
            {service.local?.port || '—'}
          </span>
        </div>
        <div className="modern-card-metric">
          <span className="modern-card-metric-label">Response</span>
          <span className="modern-card-metric-value modern-card-metric-value--mono">
            {formatResponseTime(healthStatus?.responseTime)}
          </span>
        </div>
        <div className="modern-card-metric">
          <span className="modern-card-metric-label">Uptime</span>
          <span className="modern-card-metric-value modern-card-metric-value--mono">
            {formatUptime(service.local?.startTime)}
          </span>
        </div>
      </div>
      
      {/* URL */}
      {service.local?.url && (
        <a 
          href={service.local.url}
          target="_blank"
          rel="noopener noreferrer"
          className="modern-card-url"
          onClick={e => e.stopPropagation()}
        >
          <span className="modern-card-url-text">{service.local.url}</span>
          <ExternalLink className="modern-card-url-icon" />
        </a>
      )}
    </article>
  );
}
```

---

## Accessibility

- Card is focusable and keyboard-navigable
- Status communicated via text, not just color
- Interactive elements have sufficient contrast
- Actions are accessible via keyboard
- Clear focus indicators
- Proper heading hierarchy (h3 for title)
