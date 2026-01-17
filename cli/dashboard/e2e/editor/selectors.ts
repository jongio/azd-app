/**
 * Selector Registry for Azure YAML Editor E2E Tests
 * 
 * This file contains all selectors used in E2E tests, organized by component.
 * Selectors use multiple fallback patterns for reliability.
 * 
 * Priority order:
 * 1. data-testid (if available)
 * 2. aria-label/aria-labelledby
 * 3. role attributes
 * 4. Text content
 * 5. Class names (last resort)
 */

/**
 * Navigation Selectors
 */
export const navigation = {
  // Main navigation sidebar
  sidebar: '[role="navigation"][aria-label="Azure YAML Editor Navigation"], nav[aria-label*="Azure YAML Editor" i]',
  
  // Navigation tree
  tree: '[role="tree"][aria-label="Configuration structure"], [role="tree"]',
  
  // Navigation items (treeitems)
  item: (label: string) => 
    `[role="treeitem"][aria-label*="${label}" i], [role="treeitem"]:has-text("${label}")`,
  
  // Specific navigation sections
  overview: '[role="tab"][aria-selected]:has-text("Overview"), button:has-text("Overview")',
  services: '[role="tab"][aria-selected]:has-text("Services"), button:has-text("Services"), [role="treeitem"]:has-text("Services")',
  resources: '[role="tab"][aria-selected]:has-text("Resources"), button:has-text("Resources"), [role="treeitem"]:has-text("Resources")',
  hooks: '[role="treeitem"]:has-text("Hooks")',
  pipeline: '[role="treeitem"]:has-text("Pipeline")',
  infrastructure: '[role="treeitem"]:has-text("Infrastructure")',
  
  // Navigation search
  search: 'input[placeholder*="Search" i], input[type="search"]',
  searchClear: 'button[aria-label*="Clear" i], button:has-text("Clear")',
  
  // Add buttons in navigation
  addService: '[role="treeitem"][aria-label="New service"], button:has-text("New Service"), button[aria-label*="New service" i]',
  addResource: '[role="treeitem"][aria-label="New resource"], button:has-text("New Resource"), button[aria-label*="New resource" i]',
  
  // Collapse/expand
  collapseToggle: 'button[aria-label*="Collapse navigation" i], button[aria-label*="Expand navigation" i]',
  
  // Validation badges
  errorBadge: '[aria-label*="error" i]',
  warningBadge: '[aria-label*="warning" i]',
}

/**
 * Header Selectors
 */
export const header = {
  // Main header
  container: 'header, [role="banner"]',
  
  // Title
  title: 'h1:has-text("Edit azure.yaml"), h1:has-text("azure.yaml")',
  
  // Action buttons
  addService: 'button:has-text("Add Service"), button[aria-label*="Add service" i]',
  save: 'button:has-text("Save"), button:has-text("Saving...")',
  cancel: 'button:has-text("Cancel"), button:has-text("Discard")',
  preview: 'button:has-text("Preview"), button[title*="preview" i]',
  help: 'button:has-text("Help"), button[title*="Help" i], button[aria-label*="Help" i]',
  keyboardShortcuts: 'button[title*="Keyboard shortcuts" i], button[aria-label*="keyboard" i]',
  themeToggle: 'button[aria-label*="theme" i], button[aria-label*="light mode" i], button[aria-label*="dark mode" i]',
  back: 'button[aria-label="Back to Dashboard"], button[title="Back to Dashboard"]',
  
  // Status indicators
  unsaved: 'span:has-text("Unsaved")',
  saved: 'span:has-text("Saved")',
  loading: 'span:has-text("Loading...")',
  errorCount: 'span:has-text("error"), span:has-text("errors")',
  warningCount: 'span:has-text("warning"), span:has-text("warnings")',
}

/**
 * Tab Selectors
 */
export const tabs = {
  container: '[role="tablist"][aria-label="Azure YAML Editor views"], [role="tablist"]',
  tab: (label: string) => 
    `[role="tab"][aria-selected]:has-text("${label}"), [role="tab"]:has-text("${label}"), button:has-text("${label}")`,
  overview: '[role="tab"]:has-text("Overview")',
  services: '[role="tab"]:has-text("Services")',
  resources: '[role="tab"]:has-text("Resources")',
}

/**
 * Modal/Dialog Selectors
 */
export const modals = {
  // Generic modal
  dialog: '[role="dialog"]',
  
  // Add Service Modal
  addService: {
    container: '[role="dialog"]:has-text("Add Service"), [role="dialog"]:has-text("Service")',
    tabs: {
      wellknown: 'button:has-text("Well-Known"), [role="tab"]:has-text("Well-Known")',
      application: 'button:has-text("Application"), [role="tab"]:has-text("Application")',
      container: 'button:has-text("Container"), [role="tab"]:has-text("Container")',
    },
    nameInput: 'input[id="app-service-name"], input[name="name"], input[placeholder*="name" i]',
    hostSelect: 'select[name="host"], [name="host"]',
    projectInput: 'input[name="project"], input[placeholder*="project" i]',
    imageInput: 'input[name="image"], input[placeholder*="image" i]',
    languageSelect: 'select[name="language"], [name="language"]',
    saveButton: 'button:has-text("Add Service"):not([aria-hidden="true"]), button:has-text("Add"):not([aria-hidden="true"]):not(:has-text("Add Service")), button[type="submit"]:not([aria-hidden="true"])',
    cancelButton: 'button:has-text("Cancel"), button:has-text("Close")',
  },
  
  // Add Resource Modal
  addResource: {
    container: '[role="dialog"]:has-text("Add Resource"), [role="dialog"]:has-text("Resource")',
    nameInput: 'input[name="name"], input[placeholder*="name" i]',
    typeSelect: 'select[name="type"], [name="type"]',
    saveButton: 'button:has-text("Add Resource"), button:has-text("Add"), button[type="submit"]',
    cancelButton: 'button:has-text("Cancel"), button:has-text("Close")',
  },
  
  // Import Modal
  import: {
    container: '[role="dialog"]:has-text("Import"), [role="dialog"]:has-text("Import Configuration")',
    fileTab: 'button:has-text("File"), [role="tab"]:has-text("File")',
    pasteTab: 'button:has-text("Paste"), [role="tab"]:has-text("Paste")',
    templateTab: 'button:has-text("Template"), [role="tab"]:has-text("Template")',
    fileInput: 'input[type="file"]',
    pasteTextarea: 'textarea',
    importButton: 'button:has-text("Import"), button:has-text("Confirm")',
    cancelButton: 'button:has-text("Cancel"), button:has-text("Close")',
  },
  
  // Export Modal
  export: {
    container: '[role="dialog"]:has-text("Export"), [role="dialog"]:has-text("Export Configuration")',
    downloadButton: 'button:has-text("Download"), button:has-text("Export")',
    cancelButton: 'button:has-text("Cancel"), button:has-text("Close")',
  },
  
  // Command Palette
  commandPalette: {
    container: '[role="dialog"][aria-label="Command palette"], [role="dialog"]:has-text("Command")',
    search: 'input[aria-label="Search commands"], input[placeholder*="Search" i]',
    close: 'button[aria-label="Close command palette"], button:has-text("Close")',
  },
  
  // Delete Service Dialog
  deleteService: {
    container: '[role="dialog"]:has-text("Delete"), [role="dialog"]:has-text("Remove")',
    confirmButton: 'button:has-text("Delete"), button:has-text("Confirm")',
    cancelButton: 'button:has-text("Cancel")',
  },
  
  // Generic modal buttons
  close: 'button[aria-label*="Close" i], button:has-text("Close"), button:has-text("×")',
  backdrop: '[data-testid="dialog-backdrop"], .backdrop, [aria-hidden="true"][class*="backdrop"]',
}

/**
 * Form Field Selectors
 */
export const forms = {
  // Text inputs
  textInput: (name: string) => 
    `input[type="text"][name="${name}"], input[name="${name}"], input[placeholder*="${name}" i]`,
  
  // Number inputs
  numberInput: (name: string) => 
    `input[type="number"][name="${name}"], input[type="number"][placeholder*="${name}" i]`,
  
  // Textareas
  textarea: (name?: string) => 
    name 
      ? `textarea[name="${name}"], textarea[placeholder*="${name}" i]`
      : 'textarea',
  
  // Selects/Dropdowns
  select: (name: string) => 
    `select[name="${name}"], [name="${name}"][role="combobox"], select[aria-label*="${name}" i]`,
  
  // Checkboxes
  checkbox: (name: string) => 
    `input[type="checkbox"][name="${name}"], input[type="checkbox"][aria-label*="${name}" i]`,
  
  // Switches/Toggles
  switch: (name: string) => 
    `button[role="switch"][name="${name}"], input[type="checkbox"][name="${name}"], button[role="switch"][aria-label*="${name}" i]`,
  
  // Field labels
  label: (text: string) => 
    `label:has-text("${text}"), [aria-label*="${text}" i]`,
  
  // Field errors
  error: (fieldName?: string) => 
    fieldName
      ? `[aria-label*="error" i]:near(input[name="${fieldName}"], 100), .error:has-text("${fieldName}")`
      : '[aria-label*="error" i], .error, [class*="error"]',
  
  // Required indicator
  required: 'span[aria-label="required"], span:has-text("*")',
  
  // Help icon
  help: 'button[aria-label*="Help" i], [title*="help" i]',
}

/**
 * Array Field Selectors
 */
export const arrays = {
  addButton: 'button:has-text("Add"), button[aria-label*="Add" i]',
  removeButton: (index?: number) => 
    index !== undefined
      ? `button[aria-label="Remove item ${index + 1}"], button[aria-label*="Remove" i]:nth(${index})`
      : 'button[aria-label*="Remove" i], button:has-text("Remove")',
  item: (index: number) => 
    `[role="group"]:nth(${index}), .array-item:nth(${index})`,
}

/**
 * Object Field Selectors
 */
export const objects = {
  expandButton: 'button[aria-label*="Expand" i], button[aria-label*="Collapse" i]',
  header: 'button:has-text("Config"), [class*="object-header"]',
}

/**
 * Preview Pane Selectors
 */
export const preview = {
  container: '[class*="preview"], [role="region"][aria-label*="preview" i]',
  content: '[class*="preview"] pre, [class*="preview"] code, [class*="preview"] textarea',
  toggle: 'button:has-text("Preview"), button[title*="preview" i]',
  resizeHandle: '[role="separator"][aria-label="Resize preview pane"], [aria-label*="Resize" i]',
}

/**
 * YAML Editor Selectors
 */
export const yamlEditor = {
  textarea: 'textarea[class*="yaml"], textarea, [role="textbox"]',
  content: 'textarea, [role="textbox"]',
}

/**
 * Validation Selectors
 */
export const validation = {
  panel: '[class*="validation"], [aria-label*="validation" i]',
  errors: '[aria-label*="error" i], .error, [class*="error"]',
  warnings: '[aria-label*="warning" i], .warning, [class*="warning"]',
  summary: '[class*="validation-summary"], [aria-label*="validation" i]',
  errorCount: 'span:has-text("error"), span:has-text("errors")',
  warningCount: 'span:has-text("warning"), span:has-text("warnings")',
}

/**
 * Quick Actions Bar Selectors
 */
export const quickActions = {
  container: '[class*="quick-actions"], footer',
  import: 'button[aria-label="Import configuration"], button:has-text("Import")',
  export: 'button[aria-label="Export configuration"], button:has-text("Export")',
  addService: (serviceName: string) => 
    `button[aria-label*="${serviceName}" i], button:has-text("${serviceName}")`,
}

/**
 * Help Panel Selectors
 */
export const help = {
  panel: '[class*="help"], [role="complementary"][aria-label*="help" i]',
  close: 'button:has-text("Close"), button[aria-label*="Close" i]',
}

/**
 * Keyboard Shortcuts Selectors
 */
export const keyboard = {
  dialog: '[role="dialog"]:has-text("Keyboard Shortcuts"), [role="dialog"]:has-text("Shortcuts")',
  close: 'button:has-text("Close"), button[aria-label*="Close" i]',
}

/**
 * Helper function to combine multiple selectors
 */
export function combineSelectors(...selectors: string[]): string {
  return selectors.join(', ')
}

/**
 * Helper function to create a locator with fallbacks
 */
export function createLocator(page: any, selectors: string | string[]) {
  const selectorArray = Array.isArray(selectors) ? selectors : [selectors]
  return page.locator(combineSelectors(...selectorArray)).first()
}
