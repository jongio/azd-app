/**
 * Command History Tests
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { loadHistory, saveHistory, addToHistory, clearHistory, removeFromHistory } from './command-history'

describe('command-history', () => {
  beforeEach(() => {
    localStorage.clear()
  })
  
  afterEach(() => {
    localStorage.clear()
  })
  
  describe('loadHistory', () => {
    it('should return empty history when no data exists', () => {
      const history = loadHistory()
      
      expect(history.recent).toEqual([])
      expect(history.lastUpdated).toBeGreaterThan(0)
    })
    
    it('should load existing history from localStorage', () => {
      const testHistory = {
        recent: ['cmd1', 'cmd2'],
        lastUpdated: Date.now(),
      }
      
      localStorage.setItem('azd-command-palette-history', JSON.stringify(testHistory))
      
      const history = loadHistory()
      
      expect(history.recent).toEqual(['cmd1', 'cmd2'])
      expect(history.lastUpdated).toBe(testHistory.lastUpdated)
    })
    
    it('should return empty history for invalid JSON', () => {
      localStorage.setItem('azd-command-palette-history', 'invalid json')
      
      const history = loadHistory()
      
      expect(history.recent).toEqual([])
    })
    
    it('should return empty history for invalid structure', () => {
      localStorage.setItem('azd-command-palette-history', JSON.stringify({ invalid: 'structure' }))
      
      const history = loadHistory()
      
      expect(history.recent).toEqual([])
    })
  })
  
  describe('saveHistory', () => {
    it('should save history to localStorage', () => {
      const history = {
        recent: ['cmd1', 'cmd2', 'cmd3'],
        lastUpdated: Date.now(),
      }
      
      saveHistory(history)
      
      const stored = localStorage.getItem('azd-command-palette-history')
      expect(stored).toBeTruthy()
      
      const parsed = JSON.parse(stored!)
      expect(parsed.recent).toEqual(['cmd1', 'cmd2', 'cmd3'])
    })
    
    it('should handle localStorage errors gracefully', () => {
      // Mock localStorage.setItem to throw
      const originalSetItem = localStorage.setItem
      localStorage.setItem = () => {
        throw new Error('Quota exceeded')
      }
      
      const history = {
        recent: ['cmd1'],
        lastUpdated: Date.now(),
      }
      
      // Should not throw
      expect(() => saveHistory(history)).not.toThrow()
      
      // Restore
      localStorage.setItem = originalSetItem
    })
  })
  
  describe('addToHistory', () => {
    it('should add command to front of history', () => {
      const history = {
        recent: ['cmd1', 'cmd2'],
        lastUpdated: Date.now(),
      }
      
      const updated = addToHistory('cmd3', history)
      
      expect(updated.recent).toEqual(['cmd3', 'cmd1', 'cmd2'])
    })
    
    it('should move existing command to front', () => {
      const history = {
        recent: ['cmd1', 'cmd2', 'cmd3'],
        lastUpdated: Date.now(),
      }
      
      const updated = addToHistory('cmd2', history)
      
      expect(updated.recent).toEqual(['cmd2', 'cmd1', 'cmd3'])
    })
    
    it('should limit history to 10 items', () => {
      const history = {
        recent: Array.from({ length: 10 }, (_, i) => `cmd${i}`),
        lastUpdated: Date.now(),
      }
      
      const updated = addToHistory('new', history)
      
      expect(updated.recent).toHaveLength(10)
      expect(updated.recent[0]).toBe('new')
      expect(updated.recent.includes('cmd9')).toBe(false)
    })
    
    it('should update lastUpdated timestamp', () => {
      const oldTimestamp = Date.now() - 10000
      const history = {
        recent: ['cmd1'],
        lastUpdated: oldTimestamp,
      }
      
      const updated = addToHistory('cmd2', history)
      
      expect(updated.lastUpdated).toBeGreaterThan(oldTimestamp)
    })
    
    it('should persist to localStorage', () => {
      const history = {
        recent: ['cmd1'],
        lastUpdated: Date.now(),
      }
      
      addToHistory('cmd2', history)
      
      const stored = localStorage.getItem('azd-command-palette-history')
      expect(stored).toBeTruthy()
      
      const parsed = JSON.parse(stored!)
      expect(parsed.recent).toEqual(['cmd2', 'cmd1'])
    })
  })
  
  describe('clearHistory', () => {
    it('should clear all history', () => {
      const history = {
        recent: ['cmd1', 'cmd2', 'cmd3'],
        lastUpdated: Date.now(),
      }
      
      saveHistory(history)
      
      const cleared = clearHistory()
      
      expect(cleared.recent).toEqual([])
    })
    
    it('should persist cleared history to localStorage', () => {
      const history = {
        recent: ['cmd1', 'cmd2'],
        lastUpdated: Date.now(),
      }
      
      saveHistory(history)
      clearHistory()
      
      const stored = localStorage.getItem('azd-command-palette-history')
      expect(stored).toBeTruthy()
      
      const parsed = JSON.parse(stored!)
      expect(parsed.recent).toEqual([])
    })
  })
  
  describe('removeFromHistory', () => {
    it('should remove specific command from history', () => {
      const history = {
        recent: ['cmd1', 'cmd2', 'cmd3'],
        lastUpdated: Date.now(),
      }
      
      const updated = removeFromHistory('cmd2', history)
      
      expect(updated.recent).toEqual(['cmd1', 'cmd3'])
    })
    
    it('should handle removing non-existent command', () => {
      const history = {
        recent: ['cmd1', 'cmd2'],
        lastUpdated: Date.now(),
      }
      
      const updated = removeFromHistory('cmd3', history)
      
      expect(updated.recent).toEqual(['cmd1', 'cmd2'])
    })
    
    it('should update lastUpdated timestamp', () => {
      const oldTimestamp = Date.now() - 10000
      const history = {
        recent: ['cmd1', 'cmd2'],
        lastUpdated: oldTimestamp,
      }
      
      const updated = removeFromHistory('cmd1', history)
      
      expect(updated.lastUpdated).toBeGreaterThan(oldTimestamp)
    })
    
    it('should persist to localStorage', () => {
      const history = {
        recent: ['cmd1', 'cmd2', 'cmd3'],
        lastUpdated: Date.now(),
      }
      
      saveHistory(history)
      removeFromHistory('cmd2', history)
      
      const stored = localStorage.getItem('azd-command-palette-history')
      expect(stored).toBeTruthy()
      
      const parsed = JSON.parse(stored!)
      expect(parsed.recent).toEqual(['cmd1', 'cmd3'])
    })
  })
})
