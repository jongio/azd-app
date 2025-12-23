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
  type: 'click' | 'wait' | 'evaluate' | 'type';
  selector?: string;
  script?: string;
  delay?: number;
  text?: string;  // For type action
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
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
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
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      { type: 'click', selector: '[role="tab"]:has-text("Console")', description: 'Reset to Console tab' },
      { type: 'wait', delay: 300, description: 'Wait for reset' },
      { type: 'click', selector: '[role="tab"]:has-text("Services")', description: 'Click Services tab' },
      { type: 'wait', delay: 300, description: 'Wait for view to load' },
      // Ensure grid view is selected (click Grid button if visible)
      { type: 'click', selector: 'button:has-text("Grid")', description: 'Click Grid view button' },
      { type: 'wait', delay: 300, description: 'Wait for grid to render' },
    ],
  },
  // Resources view - Table
  {
    name: 'dashboard-resources-table',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 1500,
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
  },
  // Azure Logs - Main view (Console tab with Azure mode enabled)
  {
    name: 'dashboard-azure-logs',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
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
  },
  // Azure Logs - Time range selector view
  {
    name: 'dashboard-azure-logs-time-range',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
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
  },
  // Azure Logs - Service filter view
  {
    name: 'dashboard-azure-logs-filters',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
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
  },
  // Services view with health status indicators
  {
    name: 'dashboard-services-health',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 1500,
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      { type: 'click', selector: '[role="tab"]:has-text("Console")', description: 'Reset to Console tab' },
      { type: 'wait', delay: 300, description: 'Wait for reset' },
      { type: 'click', selector: '[role="tab"]:has-text("Services")', description: 'Click Services tab' },
      { type: 'wait', delay: 500, description: 'Wait for services view to load with health indicators' },
    ],
  },
  // Console with local logs and filters visible
  {
    name: 'console-local-logs',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
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
  },
  // Console with search term highlighted
  {
    name: 'console-log-search',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 2000,
    validateElements: REQUIRED_ELEMENTS,
    requireServices: true,
    actions: [
      { type: 'click', selector: '[role="tab"]:has-text("Console")', description: 'Reset to Console tab' },
      { type: 'wait', delay: 300, description: 'Wait for reset' },
      // Console tab is default view
      { type: 'wait', delay: 2000, description: 'Wait for all logs to fully populate' },
      // Find and interact with search input - use type action for better React compatibility
      { type: 'click', selector: 'input[type="text"], input[placeholder*="Search"], input[placeholder*="Filter"]', description: 'Click search input' },
      { type: 'wait', delay: 200, description: 'Wait for input to focus' },
      { type: 'type', selector: 'input[type="text"], input[placeholder*="Search"], input[placeholder*="Filter"]', text: 'health', description: 'Type search term "health"' },
      { type: 'wait', delay: 1000, description: 'Wait for search results to highlight and render' },
    ],
  },
  // Health tab or status view showing service health details
  {
    name: 'health-view',
    url: '', // Will be set dynamically from azd app run output
    viewport: { width: 1800, height: 1200 },
    delay: 1500,
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
  },
];
