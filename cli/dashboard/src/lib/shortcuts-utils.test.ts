import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import {
  shortcuts,
  isMacPlatform,
  formatKey,
  getShortcutsByCategory,
  getCategoryDisplayName,
  viewToKey,
  keyToView,
  shouldHandleShortcut,
} from './shortcuts-utils'

describe('shortcuts-utils', () => {
  describe('shortcuts', () => {
    it('should have navigation shortcuts', () => {
      const navShortcuts = shortcuts.filter(s => s.category === 'navigation')
      expect(navShortcuts.length).toBeGreaterThan(0)
    })

    it('should have action shortcuts', () => {
      const actionShortcuts = shortcuts.filter(s => s.category === 'actions')
      expect(actionShortcuts.length).toBeGreaterThan(0)
    })

    it('should have view shortcuts', () => {
      const viewShortcuts = shortcuts.filter(s => s.category === 'views')
      expect(viewShortcuts.length).toBeGreaterThan(0)
    })

    it('should include number keys for navigation', () => {
      const navShortcuts = shortcuts.filter(s => s.category === 'navigation')
      const keys = navShortcuts.map(s => s.key)
      expect(keys).toContain('1')
      expect(keys).toContain('2')
      expect(keys).toContain('3')
    })

    it('should include ? for showing shortcuts modal', () => {
      const shortcut = shortcuts.find(s => s.key === '?')
      expect(shortcut).toBeDefined()
      expect(shortcut?.description).toContain('keyboard shortcuts')
    })
  })

  describe('isMacPlatform', () => {
    const originalNavigator = global.navigator

    afterEach(() => {
      Object.defineProperty(global, 'navigator', {
        value: originalNavigator,
        writable: true,
      })
    })

    it('should return false when navigator is undefined', () => {
      Object.defineProperty(global, 'navigator', {
        value: undefined,
        writable: true,
      })
      expect(isMacPlatform()).toBe(false)
    })

    it('should return true on Mac platform', () => {
      Object.defineProperty(global, 'navigator', {
        value: { userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36' },
        writable: true,
      })
      expect(isMacPlatform()).toBe(true)
    })

    it('should return true on Mac when using userAgentData API', () => {
      Object.defineProperty(global, 'navigator', {
        value: { userAgentData: { platform: 'macOS' }, userAgent: '' },
        writable: true,
      })
      expect(isMacPlatform()).toBe(true)
    })

    it('should return false on Windows platform', () => {
      Object.defineProperty(global, 'navigator', {
        value: { userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36' },
        writable: true,
      })
      expect(isMacPlatform()).toBe(false)
    })

    it('should return false on Linux platform', () => {
      Object.defineProperty(global, 'navigator', {
        value: { userAgent: 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36' },
        writable: true,
      })
      expect(isMacPlatform()).toBe(false)
    })
  })

  describe('formatKey', () => {
    const originalNavigator = global.navigator

    beforeEach(() => {
      // Default to non-Mac for tests
      Object.defineProperty(global, 'navigator', {
        value: { userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36' },
        writable: true,
      })
    })

    afterEach(() => {
      Object.defineProperty(global, 'navigator', {
        value: originalNavigator,
        writable: true,
      })
    })

    it('should return key unchanged for simple keys', () => {
      expect(formatKey('A')).toBe('A')
      expect(formatKey('1')).toBe('1')
      expect(formatKey('?')).toBe('?')
    })

    it('should keep Ctrl on Windows', () => {
      expect(formatKey('Ctrl+F')).toBe('Ctrl+F')
      expect(formatKey('Ctrl+C')).toBe('Ctrl+C')
    })

    it('should replace Ctrl with ⌘ on Mac', () => {
      Object.defineProperty(global, 'navigator', {
        value: { userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36' },
        writable: true,
      })
      expect(formatKey('Ctrl+F')).toBe('⌘F')
      expect(formatKey('Ctrl+C')).toBe('⌘C')
    })

    it('should keep Alt on Windows', () => {
      expect(formatKey('Alt+Tab')).toBe('Alt+Tab')
    })

    it('should replace Alt with ⌥ on Mac', () => {
      Object.defineProperty(global, 'navigator', {
        value: { userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36' },
        writable: true,
      })
      expect(formatKey('Alt+Tab')).toBe('⌥Tab')
    })
  })

  describe('getShortcutsByCategory', () => {
    it('should filter navigation shortcuts', () => {
      const result = getShortcutsByCategory('navigation')
      expect(result.every(s => s.category === 'navigation')).toBe(true)
    })

    it('should filter action shortcuts', () => {
      const result = getShortcutsByCategory('actions')
      expect(result.every(s => s.category === 'actions')).toBe(true)
    })

    it('should filter view shortcuts', () => {
      const result = getShortcutsByCategory('views')
      expect(result.every(s => s.category === 'views')).toBe(true)
    })

    it('should return correct count for each category', () => {
      const navCount = shortcuts.filter(s => s.category === 'navigation').length
      const actionCount = shortcuts.filter(s => s.category === 'actions').length
      const viewCount = shortcuts.filter(s => s.category === 'views').length
      
      expect(getShortcutsByCategory('navigation').length).toBe(navCount)
      expect(getShortcutsByCategory('actions').length).toBe(actionCount)
      expect(getShortcutsByCategory('views').length).toBe(viewCount)
    })
  })

  describe('getCategoryDisplayName', () => {
    it('should return Navigation for navigation category', () => {
      expect(getCategoryDisplayName('navigation')).toBe('Navigation')
    })

    it('should return Actions for actions category', () => {
      expect(getCategoryDisplayName('actions')).toBe('Actions')
    })

    it('should return Views for views category', () => {
      expect(getCategoryDisplayName('views')).toBe('Views')
    })
  })

  describe('viewToKey', () => {
    it('should map console to 1', () => {
      expect(viewToKey.console).toBe('1')
    })

    it('should map resources to 2', () => {
      expect(viewToKey.resources).toBe('2')
    })

    it('should map environment to 3', () => {
      expect(viewToKey.environment).toBe('3')
    })
  })

  describe('keyToView', () => {
    it('should map 1 to console', () => {
      expect(keyToView['1']).toBe('console')
    })

    it('should map 2 to resources', () => {
      expect(keyToView['2']).toBe('resources')
    })

    it('should map 3 to environment', () => {
      expect(keyToView['3']).toBe('environment')
    })
  })

  describe('shouldHandleShortcut', () => {
    it('should return true for regular elements', () => {
      const event = {
        target: document.createElement('div'),
      } as unknown as KeyboardEvent
      expect(shouldHandleShortcut(event)).toBe(true)
    })

    it('should return false for input elements', () => {
      const event = {
        target: document.createElement('input'),
      } as unknown as KeyboardEvent
      expect(shouldHandleShortcut(event)).toBe(false)
    })

    it('should return false for textarea elements', () => {
      const event = {
        target: document.createElement('textarea'),
      } as unknown as KeyboardEvent
      expect(shouldHandleShortcut(event)).toBe(false)
    })

    it('should return false for select elements', () => {
      const event = {
        target: document.createElement('select'),
      } as unknown as KeyboardEvent
      expect(shouldHandleShortcut(event)).toBe(false)
    })

    it('should return false for contenteditable elements', () => {
      const div = document.createElement('div')
      // Need to set both contentEditable property and mock isContentEditable getter
      div.contentEditable = 'true'
      Object.defineProperty(div, 'isContentEditable', { value: true })
      const event = {
        target: div,
      } as unknown as KeyboardEvent
      expect(shouldHandleShortcut(event)).toBe(false)
    })

    it('should return true for buttons', () => {
      const event = {
        target: document.createElement('button'),
      } as unknown as KeyboardEvent
      expect(shouldHandleShortcut(event)).toBe(true)
    })

    it('should return true for spans', () => {
      const event = {
        target: document.createElement('span'),
      } as unknown as KeyboardEvent
      expect(shouldHandleShortcut(event)).toBe(true)
    })
  })
})
