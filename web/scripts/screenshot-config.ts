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
  type: 'click' | 'wait' | 'evaluate';
  selector?: string;
  script?: string;
  delay?: number;
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
  validateElements?: ValidationRule[];
  requireServices?: boolean;
  /** Actions to perform before taking screenshot (e.g., click buttons to change view) */
  actions?: ScreenshotAction[];
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
export const SCREENSHOT_CONFIGS: ScreenshotConfig[] = [
  // Console view (default landing page)
  {
    name: 'dashboard-console',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 900, height: 600 },
    delay: 2000,
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    // Console is the default view, no navigation needed
  },
  // Resources view - Grid (default for resources)
  {
    name: 'dashboard-resources-grid',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 900, height: 600 },
    delay: 1500,
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      { type: 'click', selector: '[role="tab"]:has-text("Services")', description: 'Click Services tab' },
      { type: 'wait', delay: 500, description: 'Wait for view to load' },
      // Ensure grid view is selected (click Grid button if visible)
      { type: 'click', selector: 'button:has-text("Grid")', description: 'Click Grid view button' },
      { type: 'wait', delay: 500, description: 'Wait for grid to render' },
    ],
  },
  // Resources view - Table
  {
    name: 'dashboard-resources-table',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 900, height: 600 },
    delay: 1500,
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      { type: 'click', selector: '[role="tab"]:has-text("Services")', description: 'Click Services tab' },
      { type: 'wait', delay: 500, description: 'Wait for view to load' },
      // Switch to table view
      { type: 'click', selector: 'button:has-text("Table")', description: 'Click Table view button' },
      { type: 'wait', delay: 500, description: 'Wait for table to render' },
    ],
  },
  // Azure Logs - Main view (Console tab with Azure mode enabled)
  {
    name: 'dashboard-azure-logs',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 900, height: 600 },
    delay: 2000,
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      // Console tab is the default view, no navigation needed
      // Switch to Azure mode
      { type: 'click', selector: 'button[aria-label="View Azure logs"]', description: 'Switch to Azure logs mode' },
      { type: 'wait', delay: 15000, description: 'Wait for Azure Log Analytics polling cycle (15s)' },
    ],
  },
  // Azure Logs - Time range selector view
  {
    name: 'dashboard-azure-logs-time-range',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 900, height: 600 },
    delay: 2000,
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      // Switch to Azure mode
      { type: 'click', selector: 'button[aria-label="View Azure logs"]', description: 'Switch to Azure logs mode' },
      { type: 'wait', delay: 15000, description: 'Wait for Azure Log Analytics polling cycle (15s)' },
      // Focus and hover the time range select to make it visible
      { type: 'evaluate', script: 'document.querySelector("select")?.focus()', description: 'Focus time range dropdown' },
      { type: 'wait', delay: 500, description: 'Wait for focus state' },
    ],
  },
  // Azure Logs - Service filter view
  {
    name: 'dashboard-azure-logs-filters',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 900, height: 600 },
    delay: 2000,
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      // Switch to Azure mode
      { type: 'click', selector: 'button[aria-label="View Azure logs"]', description: 'Switch to Azure logs mode' },
      { type: 'wait', delay: 15000, description: 'Wait for Azure Log Analytics polling cycle (15s)' },
      // The service filter is visible by default in the filters bar
      // Wait additional time to ensure filters are rendered
      { type: 'wait', delay: 500, description: 'Wait for service filters to render' },
    ],
  },
];
