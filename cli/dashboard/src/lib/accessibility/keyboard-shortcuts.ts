/**
 * Keyboard shortcuts registry and handler
 */

export interface KeyboardShortcut {
  id: string
  key: string
  modifiers: {
    ctrl?: boolean
    shift?: boolean
    alt?: boolean
    meta?: boolean
  }
  description: string
  action: () => void
  category: 'navigation' | 'editor' | 'modal' | 'general'
}

/**
 * Keyboard shortcuts registry
 */
class KeyboardShortcutsRegistry {
  private shortcuts = new Map<string, KeyboardShortcut>()

  /**
   * Register a keyboard shortcut
   */
  register(shortcut: KeyboardShortcut) {
    const key = this.getShortcutKey(shortcut)
    this.shortcuts.set(key, shortcut)
  }

  /**
   * Unregister a keyboard shortcut
   */
  unregister(id: string) {
    for (const [key, shortcut] of this.shortcuts.entries()) {
      if (shortcut.id === id) {
        this.shortcuts.delete(key)
        return
      }
    }
  }

  /**
   * Handle keyboard event
   */
  handle(e: KeyboardEvent): boolean {
    const key = this.getEventKey(e)
    const shortcut = this.shortcuts.get(key)

    if (shortcut) {
      e.preventDefault()
      e.stopPropagation()
      shortcut.action()
      return true
    }

    return false
  }

  /**
   * Get all shortcuts
   */
  getAll(): KeyboardShortcut[] {
    return Array.from(this.shortcuts.values())
  }

  /**
   * Get shortcuts by category
   */
  getByCategory(category: KeyboardShortcut['category']): KeyboardShortcut[] {
    return this.getAll().filter((s) => s.category === category)
  }

  /**
   * Get shortcut key for matching
   */
  private getShortcutKey(shortcut: KeyboardShortcut): string {
    const parts: string[] = []
    if (shortcut.modifiers.ctrl) parts.push('ctrl')
    if (shortcut.modifiers.shift) parts.push('shift')
    if (shortcut.modifiers.alt) parts.push('alt')
    if (shortcut.modifiers.meta) parts.push('meta')
    parts.push(shortcut.key.toLowerCase())
    return parts.join('+')
  }

  /**
   * Get event key for matching
   */
  private getEventKey(e: KeyboardEvent): string {
    const parts: string[] = []
    if (e.ctrlKey) parts.push('ctrl')
    if (e.shiftKey) parts.push('shift')
    if (e.altKey) parts.push('alt')
    if (e.metaKey) parts.push('meta')
    parts.push(e.key.toLowerCase())
    return parts.join('+')
  }

  /**
   * Format shortcut for display
   */
  formatShortcut(shortcut: KeyboardShortcut): string {
    const parts: string[] = []
    const isMac = navigator.platform.toLowerCase().includes('mac')

    if (shortcut.modifiers.ctrl) parts.push(isMac ? 'Cmd' : 'Ctrl')
    if (shortcut.modifiers.shift) parts.push('Shift')
    if (shortcut.modifiers.alt) parts.push('Alt')
    if (shortcut.modifiers.meta) parts.push(isMac ? 'Cmd' : 'Win')

    // Format key name
    const key = shortcut.key.length === 1 
      ? shortcut.key.toUpperCase() 
      : shortcut.key.charAt(0).toUpperCase() + shortcut.key.slice(1)
    parts.push(key)

    return parts.join('+')
  }
}

// Singleton instance
let registry: KeyboardShortcutsRegistry | null = null

/**
 * Get shortcuts registry
 */
export function getShortcutsRegistry(): KeyboardShortcutsRegistry {
  if (!registry) {
    registry = new KeyboardShortcutsRegistry()
  }
  return registry
}

/**
 * Default editor shortcuts
 */
export const DEFAULT_SHORTCUTS: Omit<KeyboardShortcut, 'action'>[] = [
  {
    id: 'command-palette',
    key: 'k',
    modifiers: { ctrl: true },
    description: 'Open command palette',
    category: 'general',
  },
  {
    id: 'save-config',
    key: 's',
    modifiers: { ctrl: true },
    description: 'Save configuration',
    category: 'editor',
  },
  {
    id: 'toggle-preview',
    key: 'p',
    modifiers: { ctrl: true },
    description: 'Toggle preview pane',
    category: 'editor',
  },
  {
    id: 'close-modal',
    key: 'Escape',
    modifiers: {},
    description: 'Close modal/dialog/dropdown',
    category: 'modal',
  },
  {
    id: 'toggle-nav',
    key: 'b',
    modifiers: { ctrl: true },
    description: 'Toggle navigation sidebar',
    category: 'navigation',
  },
  {
    id: 'search-nav',
    key: 'f',
    modifiers: { ctrl: true },
    description: 'Search in navigation',
    category: 'navigation',
  },
  {
    id: 'add-service',
    key: 'n',
    modifiers: { ctrl: true },
    description: 'Add new service',
    category: 'editor',
  },
]

/**
 * React hook for keyboard shortcuts
 */
export function useKeyboardShortcuts(
  shortcuts: KeyboardShortcut[],
  enabled = true
) {
  React.useEffect(() => {
    if (!enabled) return

    const registry = getShortcutsRegistry()

    // Register shortcuts
    shortcuts.forEach((shortcut) => registry.register(shortcut))

    // Handle keyboard events
    const handleKeyDown = (e: KeyboardEvent) => {
      registry.handle(e)
    }

    document.addEventListener('keydown', handleKeyDown)

    return () => {
      // Unregister shortcuts
      shortcuts.forEach((shortcut) => registry.unregister(shortcut.id))
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [shortcuts, enabled])
}

// For React import
import * as React from 'react'
