# Modern Detail Panel Specification

> A comprehensive slide-in panel for displaying service details, environment variables, and Azure deployment information.

## Design Concept

The Modern detail panel uses a **slide-in sheet pattern** with a soft backdrop blur. It provides tabbed navigation for organizing different aspects of service information while maintaining context of the main dashboard.

---

## Visual Design

### Panel Container

```css
.modern-detail-panel {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  width: 100%;
  max-width: 520px;
  background: var(--modern-bg-primary);
  border-left: 1px solid var(--modern-border-default);
  box-shadow: var(--shadow-2xl);
  z-index: var(--z-modal);
  display: flex;
  flex-direction: column;
  animation: modern-slide-in-right var(--duration-slow) var(--ease-out);
}

@keyframes modern-slide-in-right {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

/* Exit animation */
.modern-detail-panel--closing {
  animation: modern-slide-out-right var(--duration-normal) var(--ease-in) forwards;
}

@keyframes modern-slide-out-right {
  to {
    transform: translateX(100%);
    opacity: 0;
  }
}
```

### Backdrop

```css
.modern-detail-backdrop {
  position: fixed;
  inset: 0;
  background: var(--bg-overlay);
  backdrop-filter: blur(4px);
  z-index: calc(var(--z-modal) - 1);
  animation: modern-fade-in var(--duration-normal) var(--ease-out);
}

.modern-detail-backdrop--closing {
  animation: modern-fade-out var(--duration-fast) var(--ease-in) forwards;
}

@keyframes modern-fade-out {
  to { opacity: 0; }
}
```

---

## Panel Header

```css
.modern-detail-header {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-5) var(--space-6);
  border-bottom: 1px solid var(--modern-border-default);
  flex-shrink: 0;
}

.modern-detail-header-icon {
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-lg);
  background: var(--modern-bg-tertiary);
  flex-shrink: 0;
}

.modern-detail-header-icon svg {
  width: 22px;
  height: 22px;
  color: var(--foreground-tertiary);
}

/* Status-based icon styling */
.modern-detail-header--healthy .modern-detail-header-icon {
  background: var(--modern-success-bg-light);
}

.modern-detail-header--healthy .modern-detail-header-icon svg {
  color: var(--modern-success-light);
}

.modern-detail-header--error .modern-detail-header-icon {
  background: var(--modern-error-bg-light);
}

.modern-detail-header--error .modern-detail-header-icon svg {
  color: var(--modern-error-light);
}

.modern-detail-header-info {
  flex: 1;
  min-width: 0;
}

.modern-detail-title {
  font-size: var(--text-xl);
  font-weight: var(--font-semibold);
  color: var(--foreground);
  margin-bottom: var(--space-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modern-detail-subtitle {
  font-size: var(--text-sm);
  color: var(--foreground-muted);
}

.modern-detail-close {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--foreground-muted);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--duration-fast);
  flex-shrink: 0;
}

.modern-detail-close:hover {
  color: var(--foreground);
  background: var(--modern-bg-tertiary);
}

.modern-detail-close:active {
  transform: scale(0.95);
}

.modern-detail-close svg {
  width: 20px;
  height: 20px;
}
```

---

## Tab Navigation

```css
.modern-detail-tabs {
  display: flex;
  gap: var(--space-6);
  padding: 0 var(--space-6);
  border-bottom: 1px solid var(--modern-border-default);
  flex-shrink: 0;
}

.modern-detail-tab {
  position: relative;
  padding: var(--space-3) 0;
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--foreground-muted);
  cursor: pointer;
  transition: color var(--duration-fast);
}

.modern-detail-tab:hover {
  color: var(--foreground-secondary);
}

.modern-detail-tab--active {
  color: var(--foreground);
}

.modern-detail-tab--active::after {
  content: '';
  position: absolute;
  bottom: -1px;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--primary);
  border-radius: var(--radius-full) var(--radius-full) 0 0;
}

/* Tab focus state */
.modern-detail-tab:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
  border-radius: var(--radius-sm);
}
```

---

## Tab Content

```css
.modern-detail-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-5) var(--space-6);
}

/* Scrollbar styling */
.modern-detail-content::-webkit-scrollbar {
  width: 6px;
}

.modern-detail-content::-webkit-scrollbar-track {
  background: transparent;
}

.modern-detail-content::-webkit-scrollbar-thumb {
  background: var(--modern-border-strong);
  border-radius: var(--radius-full);
}
```

---

## Section Card

```css
.modern-detail-section {
  background: var(--modern-bg-secondary);
  border: 1px solid var(--modern-border-default);
  border-radius: var(--radius-lg);
  margin-bottom: var(--space-4);
  overflow: hidden;
}

.modern-detail-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-4);
  background: var(--modern-bg-tertiary);
  border-bottom: 1px solid var(--modern-border-default);
}

.modern-detail-section-title {
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  color: var(--foreground-muted);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wider);
}

.modern-detail-section-action {
  font-size: var(--text-xs);
  color: var(--primary);
  cursor: pointer;
  transition: color var(--duration-fast);
}

.modern-detail-section-action:hover {
  color: var(--primary-hover);
}

.modern-detail-section-content {
  padding: var(--space-4);
}
```

---

## Info Row

```css
.modern-info-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: var(--space-2) 0;
}

.modern-info-row:not(:last-child) {
  border-bottom: 1px solid var(--modern-border-subtle);
}

.modern-info-label {
  font-size: var(--text-sm);
  color: var(--foreground-muted);
  flex-shrink: 0;
}

.modern-info-value {
  font-size: var(--text-sm);
  color: var(--foreground);
  text-align: right;
  word-break: break-word;
}

.modern-info-value--mono {
  font-family: var(--modern-font-mono);
}

.modern-info-value--primary {
  color: var(--primary);
}

.modern-info-value--success {
  color: var(--modern-success-light);
}

.modern-info-value--error {
  color: var(--modern-error-light);
}
```

### Info Row with Copy

```css
.modern-info-copy {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.modern-info-copy-btn {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--foreground-muted);
  border-radius: var(--radius-sm);
  cursor: pointer;
  opacity: 0;
  transition: all var(--duration-fast);
}

.modern-info-row:hover .modern-info-copy-btn {
  opacity: 1;
}

.modern-info-copy-btn:hover {
  color: var(--foreground);
  background: var(--modern-bg-tertiary);
}

.modern-info-copy-btn--copied {
  color: var(--modern-success-light) !important;
  opacity: 1 !important;
}

.modern-info-copy-btn svg {
  width: 14px;
  height: 14px;
}
```

---

## Status Display

```css
.modern-detail-status {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.modern-detail-status-dot {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
}

.modern-detail-status-text {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
}

/* Status variants */
.modern-detail-status--running .modern-detail-status-dot {
  background: var(--modern-success-light);
  animation: modern-breathe 2s ease-in-out infinite;
}

.modern-detail-status--running .modern-detail-status-text {
  color: var(--modern-success-light);
}

.modern-detail-status--error .modern-detail-status-dot {
  background: var(--modern-error-light);
  animation: modern-alert-flash 1s ease-in-out infinite;
}

.modern-detail-status--error .modern-detail-status-text {
  color: var(--modern-error-light);
}
```

---

## URL Display

```css
.modern-detail-url {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  background: var(--modern-bg-tertiary);
  border-radius: var(--radius-md);
  text-decoration: none;
  transition: all var(--duration-fast);
}

.modern-detail-url:hover {
  background: var(--modern-bg-secondary);
  border-color: var(--modern-border-strong);
}

.modern-detail-url-text {
  flex: 1;
  font-size: var(--text-xs);
  font-family: var(--modern-font-mono);
  color: var(--foreground-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modern-detail-url-icon {
  width: 14px;
  height: 14px;
  color: var(--foreground-muted);
  flex-shrink: 0;
  transition: all var(--duration-fast);
}

.modern-detail-url:hover .modern-detail-url-icon {
  color: var(--primary);
  transform: translate(2px, -2px);
}
```

---

## Environment Variables

```css
.modern-env-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.modern-env-item {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.modern-env-key {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  font-size: var(--text-xs);
  font-family: var(--modern-font-mono);
  color: var(--foreground-muted);
}

.modern-env-key-icon {
  width: 12px;
  height: 12px;
  color: var(--modern-warning-light);
}

.modern-env-value-wrapper {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.modern-env-value {
  flex: 1;
  padding: var(--space-1) var(--space-2);
  font-size: var(--text-sm);
  font-family: var(--modern-font-mono);
  color: var(--foreground-secondary);
  background: var(--modern-bg-tertiary);
  border: 1px solid var(--modern-border-default);
  border-radius: var(--radius-sm);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modern-env-value--masked {
  letter-spacing: 0.15em;
}

/* Toggle visibility button */
.modern-env-toggle {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--foreground-muted);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.modern-env-toggle:hover {
  color: var(--foreground);
  background: var(--modern-bg-tertiary);
}
```

---

## Actions Section

```css
.modern-detail-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  padding: var(--space-4);
  border-top: 1px solid var(--modern-border-default);
  flex-shrink: 0;
}

.modern-detail-action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-4);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.modern-detail-action-btn--primary {
  background: var(--primary);
  color: var(--primary-foreground);
}

.modern-detail-action-btn--primary:hover {
  background: var(--primary-hover);
}

.modern-detail-action-btn--secondary {
  background: var(--modern-bg-tertiary);
  color: var(--foreground);
  border: 1px solid var(--modern-border-default);
}

.modern-detail-action-btn--secondary:hover {
  background: var(--modern-bg-secondary);
  border-color: var(--modern-border-strong);
}

.modern-detail-action-btn svg {
  width: 16px;
  height: 16px;
}
```

---

## Azure Deployment Section

```css
.modern-azure-badge {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-2);
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  background: var(--modern-success-bg-light);
  color: var(--modern-success-light);
  border-radius: var(--radius-full);
}

.modern-azure-not-deployed {
  text-align: center;
  padding: var(--space-6);
  color: var(--foreground-muted);
}

.modern-azure-not-deployed-icon {
  width: 32px;
  height: 32px;
  margin: 0 auto var(--space-2);
  opacity: 0.5;
}

.modern-azure-not-deployed p {
  font-size: var(--text-sm);
  margin-bottom: var(--space-1);
}

.modern-azure-not-deployed code {
  display: inline-block;
  padding: var(--space-1) var(--space-2);
  font-size: var(--text-xs);
  font-family: var(--modern-font-mono);
  background: var(--modern-bg-tertiary);
  border-radius: var(--radius-sm);
  color: var(--primary);
}

/* Azure Portal link */
.modern-azure-portal-link {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-4);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--primary-foreground);
  background: var(--primary);
  border-radius: var(--radius-md);
  text-decoration: none;
  transition: all var(--duration-fast);
}

.modern-azure-portal-link:hover {
  background: var(--primary-hover);
}

.modern-azure-portal-link svg {
  width: 16px;
  height: 16px;
}
```

---

## Dark Mode Adjustments

```css
[data-theme="modern-dark"] .modern-detail-panel {
  background: var(--modern-bg-secondary);
  border-left-color: var(--modern-border-default);
}

[data-theme="modern-dark"] .modern-detail-section {
  background: var(--modern-bg-tertiary);
}

[data-theme="modern-dark"] .modern-detail-section-header {
  background: rgba(0, 0, 0, 0.2);
}
```

---

## Responsive Behavior

```css
@media (max-width: 600px) {
  .modern-detail-panel {
    max-width: 100%;
    border-left: none;
  }
  
  .modern-detail-header,
  .modern-detail-content {
    padding-left: var(--space-4);
    padding-right: var(--space-4);
  }
  
  .modern-detail-tabs {
    padding: 0 var(--space-4);
    gap: var(--space-4);
    overflow-x: auto;
  }
}
```

---

## Accessibility

### Focus Management

```jsx
// Trap focus within panel when open
useEffect(() => {
  if (isOpen) {
    const firstFocusable = panelRef.current?.querySelector('[tabindex="0"], button, a');
    firstFocusable?.focus();
  }
}, [isOpen]);

// Return focus on close
const handleClose = () => {
  onClose();
  triggerRef.current?.focus();
};
```

### ARIA Attributes

```jsx
<>
  {/* Backdrop */}
  <div
    className="modern-detail-backdrop"
    onClick={handleClose}
    aria-hidden="true"
  />
  
  {/* Panel */}
  <div
    ref={panelRef}
    role="dialog"
    aria-modal="true"
    aria-labelledby="panel-title"
    className="modern-detail-panel"
  >
    <header className="modern-detail-header">
      <h2 id="panel-title" className="modern-detail-title">
        {service.name}
      </h2>
      <button
        onClick={handleClose}
        className="modern-detail-close"
        aria-label="Close panel"
      >
        <X />
      </button>
    </header>
    
    {/* Tabs */}
    <div role="tablist" className="modern-detail-tabs">
      {tabs.map(tab => (
        <button
          key={tab.id}
          role="tab"
          aria-selected={activeTab === tab.id}
          aria-controls={`panel-${tab.id}`}
          className={cn(
            'modern-detail-tab',
            activeTab === tab.id && 'modern-detail-tab--active'
          )}
          onClick={() => setActiveTab(tab.id)}
        >
          {tab.label}
        </button>
      ))}
    </div>
    
    {/* Content */}
    <div
      role="tabpanel"
      id={`panel-${activeTab}`}
      aria-labelledby={activeTab}
      className="modern-detail-content"
    >
      {/* Tab content */}
    </div>
  </div>
</>
```

### Keyboard Navigation

| Key | Action |
|-----|--------|
| `Escape` | Close panel |
| `Tab` | Navigate focusable elements |
| `Arrow Left/Right` | Navigate tabs |
| `Enter/Space` | Activate tab, buttons |

---

## Component Structure

```tsx
interface ModernDetailPanelProps {
  service: Service | null;
  isOpen: boolean;
  onClose: () => void;
  healthStatus?: HealthCheckResult;
}

export function ModernDetailPanel({
  service,
  isOpen,
  onClose,
  healthStatus
}: ModernDetailPanelProps) {
  const [activeTab, setActiveTab] = useState<'overview' | 'local' | 'azure' | 'env'>('overview');
  const panelRef = useRef<HTMLDivElement>(null);
  
  // Close on escape
  useEscapeKey(onClose, isOpen);
  
  if (!isOpen || !service) return null;
  
  return (
    <>
      <div 
        className="modern-detail-backdrop" 
        onClick={onClose}
        aria-hidden="true"
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="panel-title"
        className="modern-detail-panel"
      >
        <DetailHeader service={service} onClose={onClose} />
        <DetailTabs activeTab={activeTab} onChange={setActiveTab} />
        <div className="modern-detail-content">
          {activeTab === 'overview' && <OverviewTab service={service} health={healthStatus} />}
          {activeTab === 'local' && <LocalTab service={service} health={healthStatus} />}
          {activeTab === 'azure' && <AzureTab service={service} />}
          {activeTab === 'env' && <EnvironmentTab service={service} />}
        </div>
      </div>
    </>
  );
}
```
