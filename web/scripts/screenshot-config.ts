/**
 * Screenshot Configuration Module
 * 
 * Defines all configuration types, validation rules, and screenshot targets.
 */

export interface ValidationRule {
  selector: string;
  description: string;
  minCount?: number;  // Minimum number of elements expected
  mustBeVisible?: boolean;  // Element must be in viewport
  textContent?: string | RegExp;  // Expected text content
}

export interface ScreenshotAction {
  type: 'click' | 'wait' | 'evaluate' | 'type' | 'keyboard';
  selector?: string;  // For click, type, or wait (wait for selector)
  script?: string;
  delay?: number;  // For wait action (timeout in ms, or delay in ms if no selector)
  text?: string;  // For type action
  key?: string;  // For keyboard action (e.g., 'k', 'Enter', 'Escape')
  modifier?: string;  // For keyboard action (e.g., 'Control', 'Meta', 'Shift')
  description: string;
}

export interface ScreenshotConfig {
  name: string;
  url: string;
  selector?: string;
  viewport: { width: number; height: number };
  waitFor?: string;
  delay?: number;
  clip?: { x: number; y: number; width: number; height: number };
  /** Selector to calculate clip region from (alternative to fixed clip) */
  clipSelector?: string;
  validateElements?: ValidationRule[];
  requireServices?: boolean;
  /** Actions to perform before taking screenshot (e.g., click buttons to change view) */
  actions?: ScreenshotAction[];
  /** Color scheme for screenshot: 'light' or 'dark'. Defaults to 'dark' for backward compatibility. */
  colorScheme?: 'light' | 'dark';
  /** Capture multiple theme variants. If specified, overrides colorScheme and captures both variants. */
  variants?: ('light' | 'dark')[];
}

// Required UI elements that must be present in the dashboard
export const REQUIRED_ELEMENTS: ValidationRule[] = [
  { 
    selector: 'header[role="banner"]', 
    description: 'Header navigation',
    mustBeVisible: true 
  },
  { 
    selector: '[role="tablist"] [role="tab"]', 
    description: 'Navigation tabs',
    minCount: 3
  },
];

// Elements that indicate a healthy dashboard with services
export const SERVICE_ELEMENTS: ValidationRule[] = [
  { 
    selector: 'table tbody tr, [class*="ServiceCard"], main > div, [class*="logs"]', 
    description: 'Service rows, cards, or main content',
    minCount: 1
  },
];

// Error states that should NOT be present
export const ERROR_SELECTORS = [
  { selector: 'text="Error Loading Services"', description: 'Service loading error' },
  { selector: 'text="No Services Running"', description: 'No services message' },
  { selector: 'text="Failed to connect"', description: 'Connection error' },
  { selector: 'text="Reconnecting"', description: 'Reconnecting state - dashboard not connected' },
  { selector: 'text="Connection lost"', description: 'Connection lost message' },
  { selector: '[class*="error"]', description: 'Error styling', checkClass: true },
];

// Screenshot configurations
// 
// To capture both light and dark variants, use the `variants` property:
//   variants: ['light', 'dark']  // Will create dashboard-console-light.png and dashboard-console-dark.png
// 
// To capture a single variant, use the `colorScheme` property:
//   colorScheme: 'light'  // Will create dashboard-console.png in light mode
//
export const SCREENSHOT_CONFIGS: ScreenshotConfig[] = [
  // Console view (default landing page)
  {
    name: 'dashboard-console',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
    colorScheme: 'light', // All screenshots should be in light mode
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    // Console is the default view, no navigation needed
    // Note: Can add variants: ['light', 'dark'] to capture both themes
  },
  // Mobile view - Dashboard in mobile viewport
  {
    name: 'dashboard-mobile',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 375, height: 667 }, // iPhone SE size
    delay: 2000,
    colorScheme: 'light', // All screenshots should be in light mode
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    // Console is the default view, no navigation needed
  },
  // Resources view - Grid (default for resources)
  {
    name: 'dashboard-resources-grid',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 1500,
    colorScheme: 'light', // All screenshots should be in light mode
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      { type: 'click', selector: '[role="tab"]:has-text("Console")', description: 'Reset to Console tab' },
      { type: 'wait', delay: 300, description: 'Wait for reset' },
      { type: 'click', selector: '[role="tab"]:has-text("Services")', description: 'Click Services tab' },
      { type: 'wait', delay: 500, description: 'Wait for view to load' },
      // Ensure grid view is selected (click Grid button if visible)
      { type: 'click', selector: 'button:has-text("Grid")', description: 'Click Grid view button' },
      { type: 'wait', delay: 500, description: 'Wait for grid to render' },
    ],
    // Capture the full services grid area with better focus - using fixed clip for more consistent results
    clip: { x: 0, y: 80, width: 1800, height: 1000 }, // Capture from below header to show grid cards
  },
  // Resources view - Table
  {
    name: 'dashboard-resources-table',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 1500,
    colorScheme: 'light', // All screenshots should be in light mode
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      { type: 'click', selector: '[role="tab"]:has-text("Console")', description: 'Reset to Console tab' },
      { type: 'wait', delay: 300, description: 'Wait for reset' },
      { type: 'click', selector: '[role="tab"]:has-text("Services")', description: 'Click Services tab' },
      { type: 'wait', delay: 300, description: 'Wait for view to load' },
      // Switch to table view
      { type: 'click', selector: 'button:has-text("Table")', description: 'Click Table view button' },
      { type: 'wait', delay: 300, description: 'Wait for table to render' },
    ],
    // Focus on the table area, excluding header
    clipSelector: 'table, [role="table"], [class*="table"]',
  },
  // Azure Logs - Main view (Console tab with Azure mode enabled)
  {
    name: 'dashboard-azure-logs',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
    colorScheme: 'light', // All screenshots should be in light mode
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      { type: 'click', selector: '[role="tab"]:has-text("Console")', description: 'Reset to Console tab' },
      { type: 'wait', delay: 300, description: 'Wait for reset' },
      // Console tab is the default view, no navigation needed
      // Switch to Azure mode
      { type: 'click', selector: 'button[aria-label="View Azure logs"]', description: 'Switch to Azure logs mode' },
      { type: 'wait', delay: 5000, description: 'Wait for Azure Log Analytics data to load' },
    ],
    // Focus on the logs pane area
    clipSelector: '[class*="LogsPane"], [class*="logs"], [class*="console"], main > div:has(table), main > div:has([class*="log"])',
  },
  // Azure Logs - Time range selector view
  {
    name: 'dashboard-azure-logs-time-range',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
    colorScheme: 'light', // All screenshots should be in light mode
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      { type: 'click', selector: '[role="tab"]:has-text("Console")', description: 'Reset to Console tab' },
      { type: 'wait', delay: 300, description: 'Wait for reset' },
      // Switch to Azure mode
      { type: 'click', selector: 'button[aria-label="View Azure logs"]', description: 'Switch to Azure logs mode' },
      { type: 'wait', delay: 8000, description: 'Wait for Azure logs to fully load' },
      // Focus and hover the time range select to make it visible
      { type: 'evaluate', script: 'document.querySelector("select")?.focus()', description: 'Focus time range dropdown' },
      { type: 'wait', delay: 300, description: 'Wait for focus state' },
    ],
    // Focus on the time range selector area (top of logs pane)
    clipSelector: 'select, [class*="time"], [class*="range"], [class*="filter"], header:has(select)',
  },
  // Azure Logs - Service filter view
  {
    name: 'dashboard-azure-logs-filters',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
    colorScheme: 'light', // All screenshots should be in light mode
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      { type: 'click', selector: '[role="tab"]:has-text("Console")', description: 'Reset to Console tab' },
      { type: 'wait', delay: 300, description: 'Wait for reset' },
      // Switch to Azure mode
      { type: 'click', selector: 'button[aria-label="View Azure logs"]', description: 'Switch to Azure logs mode' },
      { type: 'wait', delay: 8000, description: 'Wait for Azure logs to fully load' },
      // The service filter is visible by default in the filters bar
      // Wait additional time to ensure filters are rendered
      { type: 'wait', delay: 300, description: 'Wait for service filters to render' },
    ],
    // Focus on the filters bar area - use clip to capture more vertical space (taller)
    // Capture from top of filters to show full filter controls area
    clip: { x: 0, y: 60, width: 1800, height: 400 }, // Capture top 400px starting from below header
  },
  // Services view with health status indicators - MARKETING HERO SHOT
  {
    name: 'dashboard-services-health',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
    colorScheme: 'light', // All screenshots should be in light mode
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      { type: 'click', selector: '[role="tab"]:has-text("Console")', description: 'Reset to Console tab' },
      { type: 'wait', delay: 300, description: 'Wait for reset' },
      { type: 'click', selector: '[role="tab"]:has-text("Services")', description: 'Click Services tab' },
      { type: 'wait', delay: 800, description: 'Wait for services view to load with health indicators' },
      // Ensure grid view for better visual impact
      { type: 'click', selector: 'button:has-text("Grid")', description: 'Switch to grid view for better visual' },
      { type: 'wait', delay: 500, description: 'Wait for grid to render' },
    ],
    // Capture full services grid showing health cards - the "everything running" hero shot
    clip: { x: 0, y: 80, width: 1800, height: 1000 },
  },
  // Console with local logs and filters visible
  {
    name: 'console-local-logs',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
    colorScheme: 'light', // All screenshots should be in light mode
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      { type: 'click', selector: '[role="tab"]:has-text("Console")', description: 'Reset to Console tab' },
      { type: 'wait', delay: 300, description: 'Wait for reset' },
      // Console tab is the default view, ensure we're showing local logs
      { type: 'wait', delay: 500, description: 'Wait for initial view to load' },
      // Explicitly click the Local logs button to ensure Local mode is selected
      { type: 'click', selector: 'button[aria-label="View local logs"]', description: 'Switch to Local logs mode' },
      { type: 'wait', delay: 500, description: 'Wait for local logs to populate' },
    ],
    // Focus on the logs pane area
    clipSelector: '[class*="LogsPane"], [class*="logs"], [class*="console"], main > div:has(table), main > div:has([class*="log"])',
  },
  // Console with search term highlighted - DEBUGGING IN ACTION SHOT
  {
    name: 'console-log-search',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
    colorScheme: 'light', // All screenshots should be in light mode
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      { type: 'click', selector: '[role="tab"]:has-text("Console")', description: 'Reset to Console tab' },
      { type: 'wait', delay: 300, description: 'Wait for reset' },
      // Ensure local logs mode is selected to show logs
      { type: 'click', selector: 'button[aria-label="View local logs"]', description: 'Switch to Local logs mode' },
      { type: 'wait', delay: 1000, description: 'Wait for local logs mode to activate' },
      // Wait longer for logs to fully populate from all services
      { type: 'wait', delay: 5000, description: 'Wait for all logs to fully populate from services' },
      // Search for a common term that will show results (use a broader term that's likely to match)
      { type: 'click', selector: 'input[type="text"], input[placeholder*="Search"], input[placeholder*="Filter"], input[aria-label*="Search" i]', description: 'Click search input' },
      { type: 'wait', delay: 300, description: 'Wait for input to focus' },
      // Use a more common search term that will match log entries
      { type: 'type', selector: 'input[type="text"], input[placeholder*="Search"], input[placeholder*="Filter"], input[aria-label*="Search" i]', text: 'log', description: 'Search for "log" to show matching entries' },
      { type: 'wait', delay: 2000, description: 'Wait for search results to highlight and render' },
      // Verify logs are visible before capturing
      { type: 'wait', selector: '[class*="log"], [class*="Log"], table tbody tr, [role="row"]', delay: 3000, description: 'Wait for log entries to be visible' },
    ],
    // Full console area showing search in action
    clip: { x: 0, y: 60, width: 1800, height: 1000 },
  },
  // Health tab or status view showing service health details
  {
    name: 'health-view',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 1500,
    colorScheme: 'light', // All screenshots should be in light mode
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      { type: 'click', selector: '[role="tab"]:has-text("Console")', description: 'Reset to Console tab' },
      { type: 'wait', delay: 300, description: 'Wait for reset' },
      // Try to find and click health tab/view
      { type: 'click', selector: '[role="tab"]:has-text("Services")', description: 'Navigate to Services tab (health is shown here)' },
      { type: 'wait', delay: 500, description: 'Wait for services/health view to load' },
      // The health information is typically integrated into the services view
    ],
    // Focus on health status indicators and service cards
    clipSelector: '[class*="health"], [class*="ServiceCard"], [class*="status"], table:has([class*="health"])',
  },
  // Editor - Main view with navigation sidebar and preview pane
  {
    name: 'editor-main-view',
    url: '', // Will be set dynamically from azd app run output (will be /editor route)
    viewport: { width: 1800, height: 1200 },
    delay: 3000,
    colorScheme: 'light', // Editor screenshots should be in light mode
    validateElements: [
      { selector: '[role="navigation"][aria-label*="Azure YAML Editor" i], [class*="editor"], [role="tree"], nav', description: 'Editor navigation or sidebar', minCount: 1 },
    ],
    requireServices: false,
    actions: [
      { type: 'wait', delay: 2000, description: 'Wait for editor to load' },
      // Expand Services section in navigation
      { type: 'click', selector: '[role="treeitem"]:has-text("Services"), [role="treeitem"][aria-label*="Services" i], [role="button"]:has-text("Services")', description: 'Expand Services section in navigation' },
      { type: 'wait', delay: 1000, description: 'Wait for Services to expand' },
      // Click on a service/application to show UI controls in middle section
      { type: 'evaluate', script: `
        (() => {
          // Find Services section that's expanded
          const servicesSection = Array.from(document.querySelectorAll('[role="treeitem"], [role="button"]')).find(
            el => el.textContent?.includes('Services') && (el.getAttribute('aria-expanded') === 'true' || el.closest('[role="tree"]')?.querySelector('[role="treeitem"]:not(:has-text("Services"))'))
          );
          if (!servicesSection) {
            // Try to find and expand Services
            const servicesBtn = Array.from(document.querySelectorAll('[role="treeitem"], [role="button"]')).find(
              el => el.textContent?.includes('Services')
            );
            if (servicesBtn && servicesBtn instanceof HTMLElement) {
              servicesBtn.click();
              return false; // Will retry
            }
            return false;
          }
          
          // Find first service item (not Services itself, and not Overview/Resources/Hooks)
          const serviceNames = ['web', 'api', 'containerapp-api', 'appservice-web', 'functions-worker'];
          for (const name of serviceNames) {
            const service = Array.from(document.querySelectorAll('[role="treeitem"], [role="button"]')).find(
              el => {
                const text = el.textContent?.toLowerCase() || '';
                return text.includes(name.toLowerCase()) && 
                       !text.includes('services') &&
                       el !== servicesSection &&
                       el.closest('[role="tree"]')?.contains(servicesSection);
              }
            );
            if (service && service instanceof HTMLElement) {
              service.click();
              return true;
            }
          }
          
          // Fallback: find any treeitem that's a child of Services
          const allItems = Array.from(document.querySelectorAll('[role="treeitem"], [role="button"]'));
          const serviceItem = allItems.find(el => {
            const text = el.textContent?.toLowerCase() || '';
            return !text.includes('services') && 
                   !text.includes('overview') && 
                   !text.includes('resources') && 
                   !text.includes('hooks') &&
                   el.closest('[role="tree"]')?.contains(servicesSection) &&
                   servicesSection.compareDocumentPosition(el) & Node.DOCUMENT_POSITION_FOLLOWING;
          });
          if (serviceItem && serviceItem instanceof HTMLElement) {
            serviceItem.click();
            return true;
          }
          return false;
        })();
      `, description: 'Click on a service/application to show UI controls in middle section' },
      { type: 'wait', delay: 1500, description: 'Wait for service selection and UI to render' },
    ],
    // Full page to show overall layout
  },
  // Editor - Navigation sidebar view (focused on sidebar only)
  {
    name: 'editor-navigation',
    url: '', // Will be set dynamically from azd app run output (will be /editor route)
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
    colorScheme: 'light', // Editor screenshots should be in light mode
    validateElements: [
      { selector: '[aria-label="Azure YAML Editor Navigation"], [role="navigation"]', description: 'Editor navigation sidebar', minCount: 1 },
      { selector: '[role="button"], [role="treeitem"]', description: 'Navigation items', minCount: 3 },
    ],
    requireServices: false,
    actions: [
      { type: 'wait', delay: 2000, description: 'Wait for editor to load' },
      // Expand Services section to show navigation tree - try treeitem first, then button
      { type: 'click', selector: '[role="treeitem"]:has-text("Services"), [role="treeitem"][aria-label*="Services" i], [role="button"]:has-text("Services")', description: 'Expand Services section in navigation' },
      { type: 'wait', delay: 500, description: 'Wait for Services to expand' },
      { type: 'wait', delay: 500, description: 'Wait for navigation tree to fully render' },
    ],
    // Use selector to capture only the navigation sidebar
    selector: '[aria-label="Azure YAML Editor Navigation"], nav[role="navigation"]',
  },
  // Editor - Form view for editing service properties (focused on form area)
  {
    name: 'editor-form-view',
    url: '', // Will be set dynamically from azd app run output (will be /editor route)
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
    colorScheme: 'light', // Editor screenshots should be in light mode
    validateElements: [
      { selector: 'form, [class*="form"], input, select, textarea', description: 'Form elements', minCount: 1 },
    ],
    requireServices: false,
    actions: [
      { type: 'wait', delay: 2000, description: 'Wait for editor to load' },
      // Expand Services in navigation first - try treeitem first, then button
      { type: 'click', selector: '[role="treeitem"]:has-text("Services"), [role="treeitem"][aria-label*="Services" i], [role="button"]:has-text("Services")', description: 'Expand Services section in navigation' },
      { type: 'wait', delay: 1000, description: 'Wait for Services to expand' },
      // Click on a service to show form - use evaluate to find first available service and click it
      { type: 'evaluate', script: `
        (() => {
          // Find Services section that's expanded
          const servicesSection = Array.from(document.querySelectorAll('[role="treeitem"], [role="button"]')).find(
            el => el.textContent?.includes('Services') && (el.getAttribute('aria-expanded') === 'true' || el.closest('[role="tree"]')?.querySelector('[role="treeitem"]:not(:has-text("Services"))'))
          );
          if (!servicesSection) {
            // Try to find and expand Services
            const servicesBtn = Array.from(document.querySelectorAll('[role="treeitem"], [role="button"]')).find(
              el => el.textContent?.includes('Services')
            );
            if (servicesBtn && servicesBtn instanceof HTMLElement) {
              servicesBtn.click();
              // Wait a bit for expansion
              return false; // Will retry
            }
            return false;
          }
          
          // Find first service item (not Services itself, and not Overview/Resources/Hooks)
          const serviceNames = ['web', 'api', 'containerapp-api', 'appservice-web', 'functions-worker'];
          for (const name of serviceNames) {
            const service = Array.from(document.querySelectorAll('[role="treeitem"], [role="button"]')).find(
              el => {
                const text = el.textContent?.toLowerCase() || '';
                return text.includes(name.toLowerCase()) && 
                       !text.includes('services') &&
                       el !== servicesSection &&
                       el.closest('[role="tree"]')?.contains(servicesSection);
              }
            );
            if (service && service instanceof HTMLElement) {
              service.click();
              return true;
            }
          }
          
          // Fallback: find any treeitem that's a child of Services
          const allItems = Array.from(document.querySelectorAll('[role="treeitem"], [role="button"]'));
          const serviceItem = allItems.find(el => {
            const text = el.textContent?.toLowerCase() || '';
            return !text.includes('services') && 
                   !text.includes('overview') && 
                   !text.includes('resources') && 
                   !text.includes('hooks') &&
                   el.closest('[role="tree"]')?.contains(servicesSection) &&
                   servicesSection.compareDocumentPosition(el) & Node.DOCUMENT_POSITION_FOLLOWING;
          });
          if (serviceItem && serviceItem instanceof HTMLElement) {
            serviceItem.click();
            return true;
          }
          return false;
        })();
      `, description: 'Select a service to show form' },
      { type: 'wait', delay: 2000, description: 'Wait for service selection' },
      // Wait for form elements to actually appear - try multiple selectors
      { type: 'wait', selector: 'input, select, textarea, [class*="form"], [class*="field"]', delay: 3000, description: 'Wait for form elements to appear' },
      { type: 'wait', delay: 1000, description: 'Additional wait for form to fully render' },
      // If no form found, try Overview which should always have a form
      { type: 'evaluate', script: `
        (() => {
          // Check if form elements exist
          const hasForm = document.querySelectorAll('input, select, textarea, [class*="form"]').length > 0;
          if (!hasForm) {
            // Navigate to Overview which should have form fields
            const overview = Array.from(document.querySelectorAll('[role="treeitem"], [role="button"], [role="tab"]')).find(
              el => el.textContent?.toLowerCase().includes('overview')
            );
            if (overview && overview instanceof HTMLElement) {
              overview.click();
              return true;
            }
          }
          return false;
        })();
      `, description: 'Fallback to Overview if service form not found' },
      { type: 'wait', delay: 1000, description: 'Wait for Overview form to load' },
    ],
    // Use fixed clip starting from x: 0 to capture full form width (including left edge)
    // The form content area starts from the left, so we need to capture from x: 0
    clip: { x: 0, y: 100, width: 1800, height: 900 }, // Full width to capture complete form
  },
  // Editor - Validation view showing errors and warnings (focused on validation area)
  {
    name: 'editor-validation',
    url: '', // Will be set dynamically from azd app run output (will be /editor route)
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
    colorScheme: 'light', // Editor screenshots should be in light mode
    validateElements: [
      { selector: '[class*="error"], [class*="warning"], [class*="validation"], [class*="Validation"], [role="alert"]', description: 'Validation indicators', minCount: 1 },
    ],
    requireServices: false,
    actions: [
      { type: 'wait', delay: 2000, description: 'Wait for editor to load' },
      // Expand Services in navigation first
      { type: 'click', selector: '[role="treeitem"]:has-text("Services"), [role="treeitem"][aria-label*="Services" i], [role="button"]:has-text("Services")', description: 'Expand Services section' },
      { type: 'wait', delay: 1000, description: 'Wait for expansion' },
      // Click on a service - use evaluate to find first available service
      { type: 'evaluate', script: `
        (() => {
          // Find Services section that's expanded
          const servicesSection = Array.from(document.querySelectorAll('[role="treeitem"], [role="button"]')).find(
            el => el.textContent?.includes('Services')
          );
          if (!servicesSection) return false;
          
          // Ensure it's expanded
          if (servicesSection.getAttribute('aria-expanded') !== 'true') {
            if (servicesSection instanceof HTMLElement) {
              servicesSection.click();
              // Wait a bit
              return false;
            }
          }
          
          // Find first service item
          const serviceNames = ['web', 'api', 'containerapp-api', 'appservice-web', 'functions-worker'];
          for (const name of serviceNames) {
            const service = Array.from(document.querySelectorAll('[role="treeitem"], [role="button"]')).find(
              el => {
                const text = el.textContent?.toLowerCase() || '';
                return text.includes(name.toLowerCase()) && 
                       !text.includes('services') &&
                       el !== servicesSection &&
                       el.closest('[role="tree"]')?.contains(servicesSection);
              }
            );
            if (service && service instanceof HTMLElement) {
              service.click();
              return true;
            }
          }
          
          // Fallback: find any treeitem that's a child of Services
          const allItems = Array.from(document.querySelectorAll('[role="treeitem"], [role="button"]'));
          const serviceItem = allItems.find(el => {
            const text = el.textContent?.toLowerCase() || '';
            return !text.includes('services') && 
                   !text.includes('overview') && 
                   !text.includes('resources') && 
                   !text.includes('hooks') &&
                   el.closest('[role="tree"]')?.contains(servicesSection) &&
                   servicesSection.compareDocumentPosition(el) & Node.DOCUMENT_POSITION_FOLLOWING;
          });
          if (serviceItem && serviceItem instanceof HTMLElement) {
            serviceItem.click();
            return true;
          }
          return false;
        })();
      `, description: 'Select a service' },
      { type: 'wait', delay: 2000, description: 'Wait for form to load' },
      // Wait for form to appear
      { type: 'wait', selector: 'input, select, textarea', delay: 3000, description: 'Wait for form elements' },
      // Create invalid state by clearing a required field - try multiple field types
      { type: 'evaluate', script: `
        (() => {
          // Try to find and clear a required field - prefer name or project fields
          const fields = Array.from(document.querySelectorAll('input[type="text"], input:not([type]), input[name*="name" i], input[name*="project" i]'));
          for (const field of fields) {
            if (field instanceof HTMLInputElement && field.value) {
              field.focus();
              field.select();
              field.value = '';
              // Trigger input event
              field.dispatchEvent(new Event('input', { bubbles: true }));
              field.dispatchEvent(new Event('change', { bubbles: true }));
              field.blur();
              return true;
            }
          }
          return false;
        })();
      `, description: 'Clear a required field to trigger validation' },
      { type: 'wait', delay: 1500, description: 'Wait for validation to run' },
      // Navigate to Overview to see ValidationSummaryPanel which shows all validation errors
      { type: 'evaluate', script: `
        (() => {
          // Find Overview button/tab in navigation
          const overview = Array.from(document.querySelectorAll('[role="treeitem"], [role="button"], [role="tab"]')).find(
            el => el.textContent?.toLowerCase().includes('overview')
          );
          if (overview && overview instanceof HTMLElement) {
            overview.click();
            return true;
          }
          return false;
        })();
      `, description: 'Navigate to Overview to see validation summary' },
      { type: 'wait', delay: 1000, description: 'Wait for overview and validation panel to load' },
    ],
    // Focus on validation summary panel or form area showing errors
    // Use selector to capture ValidationSummaryPanel if available, otherwise form area
    selector: '[aria-label="Validation Summary"], [class*="error"], [role="alert"]',
  },
  // Editor - Quick actions bar (focused on bottom bar)
  {
    name: 'editor-quick-actions',
    url: '', // Will be set dynamically from azd app run output (will be /editor route)
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
    colorScheme: 'light', // Editor screenshots should be in light mode
    validateElements: [
      { selector: 'text="Quick Actions", [aria-label*="Import"], [aria-label*="Export"], [aria-label*="Add"]', description: 'Quick action buttons', minCount: 1 },
    ],
    requireServices: false,
    actions: [
      { type: 'wait', delay: 2000, description: 'Wait for editor to load' },
      { type: 'wait', delay: 1000, description: 'Wait for quick actions to render' },
    ],
    // Use clip to focus on bottom portion where QuickActionsBar is fixed
    // Reduced height to make it less tall - capture just the quick actions bar
    clip: { x: 0, y: 1100, width: 1800, height: 100 }, // Bottom 100px of 1200px viewport (reduced from 200px)
  },
  // Editor - Command palette (focused on modal)
  {
    name: 'editor-command-palette',
    url: '', // Will be set dynamically from azd app run output (will be /editor route)
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
    colorScheme: 'light', // Editor screenshots should be in light mode
    validateElements: [
      { selector: '[aria-label="Command palette"], input[aria-label="Search commands"]', description: 'Command palette', minCount: 1 },
    ],
    requireServices: false,
    actions: [
      { type: 'wait', delay: 2000, description: 'Wait for editor to load' },
      // Open command palette using Playwright keyboard API
      { type: 'keyboard', key: 'k', modifier: 'Control', description: 'Open command palette with Ctrl+K' },
      { type: 'wait', delay: 1000, description: 'Wait for command palette to open' },
    ],
    // Use selector to capture only the command palette modal
    selector: '[aria-label="Command palette"], [role="dialog"]:has-text("Command"), [role="dialog"]:has([aria-label="Search commands"])',
  },
];
