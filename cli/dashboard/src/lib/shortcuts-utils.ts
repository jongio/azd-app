/**
 * Helper functions for KeyboardShortcuts component
 */

export type ShortcutCategory = 'navigation' | 'actions' | 'views'

export interface Shortcut {
  key: string | string[]
  description: string
  category: ShortcutCategory
}

/**
 * All keyboard shortcuts organized by category
 */
export const shortcuts = [
  // Navigation
  { key: '1', description: 'Resources view', category: 'navigation' },
  { key: '2', description: 'Console view', category: 'navigation' },
  { key: '3', description: 'Metrics view', category: 'navigation' },
  { key: '4', description: 'Environment view', category: 'navigation' },
  { key: '5', description: 'Dependencies view', category: 'navigation' },
  
  // Actions
  { key: 'R', description: 'Refresh all services', category: 'actions' },
  { key: 'C', description: 'Clear console logs', category: 'actions' },
  { key: 'E', description: 'Export logs', category: 'actions' },
  { key: ['/', 'Ctrl+F'], description: 'Focus search input', category: 'actions' },
  
  // Views
  { key: 'T', description: 'Toggle table/grid view', category: 'views' },
  { key: '?', description: 'Show keyboard shortcuts', category: 'views' },
  { key: 'Esc', description: 'Close dialogs/modals', category: 'views' },
] as const satisfies readonly Shortcut[]

/**
 * Check if running on macOS
 * Uses modern userAgentData API with fallback to userAgent
 */
export function isMacPlatform(): boolean {
  if (typeof navigator === 'undefined') return false
  // Use modern userAgentData API if available
  const userAgentData = (navigator as Navigator & { userAgentData?: { platform?: string } }).userAgentData
  if (userAgentData?.platform) {
    return userAgentData.platform.toLowerCase() === 'macos'
  }
  // Fallback to userAgent string (navigator.platform is deprecated)
  return /mac/i.test(navigator.userAgent)
}

/**
 * Format key for display based on platform
 */
export function formatKey(key: string): string {
  const isMac = isMacPlatform()
  
  // Replace Ctrl with ⌘ on Mac
  if (key.startsWith('Ctrl+')) {
    return isMac ? `⌘${key.slice(5)}` : key
  }
  
  // Replace Alt with ⌥ on Mac
  if (key.startsWith('Alt+')) {
    return isMac ? `⌥${key.slice(4)}` : key
  }
  
  return key
}

/**
 * Get shortcuts by category
 */
export function getShortcutsByCategory(category: ShortcutCategory): Shortcut[] {
  return shortcuts.filter(s => s.category === category)
}

/**
 * Get category display name
 */
export function getCategoryDisplayName(category: ShortcutCategory): string {
  const names = {
    navigation: 'Navigation',
    actions: 'Actions',
    views: 'Views',
  } as const satisfies Record<ShortcutCategory, string>
  return names[category]
}

/**
 * Map view name to navigation key
 */
export const viewToKey = {
  resources: '1',
  console: '2',
  metrics: '3',
  environment: '4',
  dependencies: '5',
} as const satisfies Record<string, string>

/**
 * Map key to view name
 */
export const keyToView = {
  '1': 'resources',
  '2': 'console',
  '3': 'metrics',
  '4': 'environment',
  '5': 'dependencies',
} as const satisfies Record<string, string>

/**
 * Check if key event should trigger shortcut (not in input/textarea)
 */
export function shouldHandleShortcut(event: KeyboardEvent): boolean {
  const target = event.target as HTMLElement
  const tagName = target.tagName.toLowerCase()
  
  // Don't handle shortcuts when focused on input elements
  if (tagName === 'input' || tagName === 'textarea' || tagName === 'select') {
    return false
  }
  
  // Don't handle shortcuts when element is contenteditable
  if (target.isContentEditable) {
    return false
  }
  
  return true
}
