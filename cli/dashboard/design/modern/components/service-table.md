# Modern Service Table Specification

> A sophisticated data table optimized for scanning and interaction, featuring inline actions and real-time status updates.

## Design Concept

The Modern table uses **subtle row styling with hover states** and **inline expandable details**, providing a clean, scannable interface that can handle many services without visual clutter.

---

## Visual Design

### Table Container

```css
.modern-table-container {
  background: var(--modern-bg-primary);
  border: 1px solid var(--modern-border-default);
  border-radius: var(--radius-xl);
  overflow: hidden;
  box-shadow: var(--shadow-sm);
}
```

### Table Header Section

```css
.modern-table-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--modern-border-default);
  background: var(--modern-bg-secondary);
}

.modern-table-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
  color: var(--foreground);
}

.modern-table-count {
  font-size: var(--text-sm);
  font-weight: var(--font-normal);
  color: var(--foreground-muted);
  padding: var(--space-0.5) var(--space-2);
  background: var(--modern-bg-tertiary);
  border-radius: var(--radius-full);
}
```

---

## Table Structure

### Base Table

```css
.modern-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}
```

### Column Headers

```css
.modern-table thead {
  background: var(--modern-bg-secondary);
  border-bottom: 1px solid var(--modern-border-default);
}

.modern-table th {
  padding: var(--space-3) var(--space-4);
  text-align: left;
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  color: var(--foreground-muted);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wider);
  white-space: nowrap;
}

/* Sortable column header */
.modern-table th[data-sortable] {
  cursor: pointer;
  user-select: none;
  transition: color var(--duration-fast);
}

.modern-table th[data-sortable]:hover {
  color: var(--foreground);
}

.modern-table-sort-icon {
  display: inline-block;
  width: 14px;
  height: 14px;
  margin-left: var(--space-1);
  opacity: 0.3;
  transition: opacity var(--duration-fast);
}

.modern-table th[data-sorted] .modern-table-sort-icon {
  opacity: 1;
  color: var(--primary);
}

.modern-table th[data-sorted="desc"] .modern-table-sort-icon {
  transform: rotate(180deg);
}
```

### Table Body

```css
.modern-table tbody tr {
  border-bottom: 1px solid var(--modern-border-subtle);
  transition: background var(--duration-fast);
}

.modern-table tbody tr:last-child {
  border-bottom: none;
}

.modern-table tbody tr:hover {
  background: var(--modern-bg-tertiary);
}

.modern-table tbody tr:focus-within {
  background: rgba(var(--modern-primary-rgb), 0.05);
}

/* Clickable row */
.modern-table tbody tr[role="button"] {
  cursor: pointer;
}
```

### Table Cells

```css
.modern-table td {
  padding: var(--space-3) var(--space-4);
  vertical-align: middle;
  color: var(--foreground);
}

/* First column emphasis */
.modern-table td:first-child {
  font-weight: var(--font-medium);
}
```

---

## Column Types

### Name Column

```css
.modern-table-name-cell {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-width: 180px;
}

.modern-table-service-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  background: var(--modern-bg-tertiary);
  flex-shrink: 0;
}

.modern-table-service-icon svg {
  width: 16px;
  height: 16px;
  color: var(--foreground-tertiary);
}

.modern-table-service-info {
  min-width: 0;
}

.modern-table-service-name {
  font-weight: var(--font-medium);
  color: var(--foreground);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modern-table-service-meta {
  font-size: var(--text-xs);
  color: var(--foreground-muted);
  margin-top: 2px;
}
```

### Status Column

```css
.modern-table-status-cell {
  min-width: 120px;
}

.modern-table-status {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
}

.modern-table-status-dot {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

.modern-table-status--running .modern-table-status-dot {
  background: var(--modern-success-light);
  animation: modern-breathe 2s ease-in-out infinite;
}

.modern-table-status--starting .modern-table-status-dot {
  background: var(--modern-info-light);
  animation: modern-pulse-glow 1.5s ease-in-out infinite;
}

.modern-table-status--error .modern-table-status-dot {
  background: var(--modern-error-light);
  animation: modern-alert-flash 1s ease-in-out infinite;
}

.modern-table-status--degraded .modern-table-status-dot {
  background: var(--modern-warning-light);
  animation: modern-pulse-glow 2s ease-in-out infinite;
}

.modern-table-status--stopped .modern-table-status-dot {
  background: var(--foreground-muted);
}

.modern-table-status-text {
  font-weight: var(--font-medium);
}

.modern-table-status--running .modern-table-status-text {
  color: var(--modern-success-light);
}

.modern-table-status--error .modern-table-status-text {
  color: var(--modern-error-light);
}

.modern-table-status--degraded .modern-table-status-text {
  color: var(--modern-warning-light);
}

.modern-table-status--stopped .modern-table-status-text {
  color: var(--foreground-muted);
}
```

### Health Check Indicator

```css
.modern-table-health-badge {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-0.5) var(--space-2);
  font-size: var(--text-xs);
  border-radius: var(--radius-full);
  background: var(--modern-bg-tertiary);
}

.modern-table-health-badge-icon {
  width: 12px;
  height: 12px;
}

/* Health badge variants */
.modern-table-health-badge--http {
  color: var(--modern-info-light);
}

.modern-table-health-badge--port {
  color: var(--foreground-tertiary);
}

.modern-table-health-badge--process {
  color: var(--foreground-muted);
}
```

### Time Column

```css
.modern-table-time-cell {
  color: var(--foreground-tertiary);
  font-size: var(--text-xs);
  white-space: nowrap;
}

.modern-table-time-relative {
  display: block;
}

.modern-table-time-absolute {
  display: none;
  font-family: var(--modern-font-mono);
  font-size: 10px;
  color: var(--foreground-muted);
}

.modern-table tbody tr:hover .modern-table-time-absolute {
  display: block;
}
```

### URL Column

```css
.modern-table-url-cell {
  min-width: 200px;
  max-width: 300px;
}

.modern-table-url {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-2);
  font-size: var(--text-xs);
  font-family: var(--modern-font-mono);
  color: var(--foreground-secondary);
  background: var(--modern-bg-tertiary);
  border-radius: var(--radius-sm);
  text-decoration: none;
  max-width: 100%;
  transition: all var(--duration-fast);
}

.modern-table-url:hover {
  color: var(--primary);
  background: rgba(var(--modern-primary-rgb), 0.1);
}

.modern-table-url-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modern-table-url-icon {
  width: 12px;
  height: 12px;
  flex-shrink: 0;
  opacity: 0;
  transform: translate(-2px, 2px);
  transition: all var(--duration-fast);
}

.modern-table-url:hover .modern-table-url-icon {
  opacity: 1;
  transform: translate(0, 0);
}

/* No URL placeholder */
.modern-table-url--none {
  color: var(--foreground-muted);
  font-style: italic;
}
```

### Source/Project Column

```css
.modern-table-source-cell {
  color: var(--foreground-tertiary);
  font-size: var(--text-xs);
  max-width: 200px;
}

.modern-table-source-path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  direction: rtl;
  text-align: left;
}
```

### Actions Column

```css
.modern-table-actions-cell {
  width: 100px;
  text-align: right;
}

.modern-table-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-1);
  opacity: 0;
  transform: translateX(10px);
  transition: all var(--duration-fast);
}

.modern-table tbody tr:hover .modern-table-actions {
  opacity: 1;
  transform: translateX(0);
}

/* Always show on focus */
.modern-table tbody tr:focus-within .modern-table-actions {
  opacity: 1;
  transform: translateX(0);
}

.modern-table-action-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  color: var(--foreground-tertiary);
  transition: all var(--duration-fast);
}

.modern-table-action-btn:hover {
  background: var(--modern-bg-secondary);
  color: var(--foreground);
}

.modern-table-action-btn--danger:hover {
  background: var(--modern-error-bg-light);
  color: var(--modern-error-light);
}

.modern-table-action-btn svg {
  width: 16px;
  height: 16px;
}
```

---

## Expandable Row Details

```css
.modern-table-row-details {
  background: var(--modern-bg-tertiary);
  border-top: 1px solid var(--modern-border-subtle);
}

.modern-table-row-details td {
  padding: var(--space-4);
}

.modern-table-details-content {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: var(--space-4);
  max-width: 900px;
}

.modern-table-details-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.modern-table-details-label {
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  color: var(--foreground-muted);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wider);
}

.modern-table-details-value {
  font-size: var(--text-sm);
  color: var(--foreground);
}
```

---

## Responsive Behavior

### Tablet (< 1024px)

```css
@media (max-width: 1024px) {
  .modern-table-source-cell {
    display: none;
  }
  
  .modern-table-actions {
    opacity: 1;
    transform: none;
  }
}
```

### Mobile (< 768px)

Convert to card-style list on mobile.

```css
@media (max-width: 768px) {
  .modern-table-container {
    border: none;
    border-radius: 0;
    box-shadow: none;
    background: transparent;
  }
  
  .modern-table thead {
    display: none;
  }
  
  .modern-table tbody tr {
    display: block;
    margin-bottom: var(--space-3);
    background: var(--modern-bg-primary);
    border: 1px solid var(--modern-border-default);
    border-radius: var(--radius-lg);
    padding: var(--space-4);
  }
  
  .modern-table tbody td {
    display: flex;
    justify-content: space-between;
    padding: var(--space-2) 0;
    border-bottom: 1px solid var(--modern-border-subtle);
  }
  
  .modern-table tbody td:last-child {
    border-bottom: none;
  }
  
  .modern-table tbody td::before {
    content: attr(data-label);
    font-size: var(--text-xs);
    font-weight: var(--font-medium);
    color: var(--foreground-muted);
    text-transform: uppercase;
  }
  
  .modern-table-actions {
    opacity: 1;
    transform: none;
    justify-content: flex-start;
    padding-top: var(--space-2);
  }
}
```

---

## Dark Mode

```css
[data-theme="modern-dark"] .modern-table-container {
  background: var(--modern-bg-secondary);
  border-color: var(--modern-border-default);
}

[data-theme="modern-dark"] .modern-table-header,
[data-theme="modern-dark"] .modern-table thead {
  background: var(--modern-bg-tertiary);
}

[data-theme="modern-dark"] .modern-table tbody tr:hover {
  background: var(--modern-bg-tertiary);
}

[data-theme="modern-dark"] .modern-table-url {
  background: var(--modern-bg-tertiary);
}

[data-theme="modern-dark"] .modern-table-service-icon {
  background: var(--modern-bg-elevated);
}
```

---

## Empty State

```css
.modern-table-empty {
  text-align: center;
  padding: var(--space-12) var(--space-8);
}

.modern-table-empty-icon {
  width: 48px;
  height: 48px;
  margin: 0 auto var(--space-4);
  color: var(--foreground-muted);
  opacity: 0.5;
}

.modern-table-empty-title {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--foreground);
  margin-bottom: var(--space-2);
}

.modern-table-empty-message {
  font-size: var(--text-sm);
  color: var(--foreground-muted);
  margin-bottom: var(--space-4);
}

.modern-table-empty-code {
  display: inline-block;
  padding: var(--space-2) var(--space-4);
  font-family: var(--modern-font-mono);
  font-size: var(--text-sm);
  background: var(--modern-bg-tertiary);
  border-radius: var(--radius-md);
  color: var(--primary);
}
```

---

## Component Structure

```tsx
interface ModernServiceTableProps {
  services: Service[];
  healthReport?: HealthReportEvent | null;
  onServiceClick?: (service: Service) => void;
  onViewLogs?: (serviceName: string) => void;
}

export function ModernServiceTable({
  services,
  healthReport,
  onServiceClick,
  onViewLogs
}: ModernServiceTableProps) {
  return (
    <div className="modern-table-container">
      <div className="modern-table-header">
        <div className="modern-table-title">
          <span>Services</span>
          <span className="modern-table-count">{services.length}</span>
        </div>
      </div>
      
      {services.length === 0 ? (
        <TableEmptyState />
      ) : (
        <table className="modern-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Status</th>
              <th>Started</th>
              <th>Source</th>
              <th>Local URL</th>
              <th>Azure URL</th>
              <th className="text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            {services.map(service => (
              <ModernServiceTableRow
                key={service.name}
                service={service}
                healthStatus={getServiceHealth(service.name, healthReport)}
                onClick={() => onServiceClick?.(service)}
                onViewLogs={() => onViewLogs?.(service.name)}
              />
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
```

---

## Accessibility

- Table uses proper `<table>`, `<thead>`, `<tbody>` structure
- Column headers use `scope="col"`
- Row headers use `scope="row"` where applicable
- Sortable columns have `aria-sort` attribute
- Actions are keyboard accessible
- Expandable rows use `aria-expanded`
- Mobile view maintains semantic meaning with data-labels
