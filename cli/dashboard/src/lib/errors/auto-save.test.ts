/**
 * Auto-Save Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  saveDraft,
  loadDraft,
  clearDraft,
  getDraftAge,
  formatDraftAge,
  isDraftStale,
  cleanupStaleDrafts,
  AutoSaveManager,
} from './auto-save'

describe('auto-save', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.useFakeTimers()
  })

  afterEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  describe('saveDraft', () => {
    it('should save draft to localStorage', () => {
      const config = { name: 'test-app', services: {} }
      const result = saveDraft(config, true)

      expect(result).toBe(true)
      const stored = localStorage.getItem('azure-yaml-editor-draft')
      expect(stored).toBeTruthy()

      const parsed = JSON.parse(stored!)
      expect(parsed.config).toEqual(config)
      expect(parsed.dirty).toBe(true)
    })

    it('should reject drafts larger than 5MB', () => {
      const largConfig = { data: 'x'.repeat(6 * 1024 * 1024) }
      const result = saveDraft(largConfig, true)

      expect(result).toBe(false)
      expect(localStorage.getItem('azure-yaml-editor-draft')).toBeNull()
    })

    it('should handle localStorage errors', () => {
      const originalSetItem = localStorage.setItem
      localStorage.setItem = vi.fn(() => {
        throw new Error('Quota exceeded')
      })

      const result = saveDraft({ name: 'test' }, true)

      expect(result).toBe(false)

      localStorage.setItem = originalSetItem
    })
  })

  describe('loadDraft', () => {
    it('should load draft from localStorage', () => {
      const config = { name: 'test-app' }
      saveDraft(config, true)

      const draft = loadDraft()
      expect(draft).toBeTruthy()
      expect(draft!.config).toEqual(config)
      expect(draft!.dirty).toBe(true)
    })

    it('should return null if no draft exists', () => {
      const draft = loadDraft()
      expect(draft).toBeNull()
    })

    it('should handle invalid JSON', () => {
      localStorage.setItem('azure-yaml-editor-draft', 'invalid json')
      const draft = loadDraft()
      expect(draft).toBeNull()
    })

    it('should handle localStorage errors', () => {
      const originalGetItem = localStorage.getItem
      localStorage.getItem = vi.fn(() => {
        throw new Error('Storage error')
      })

      const draft = loadDraft()

      expect(draft).toBeNull()

      localStorage.getItem = originalGetItem
    })
  })

  describe('clearDraft', () => {
    it('should clear draft from localStorage', () => {
      saveDraft({ name: 'test' }, true)
      expect(localStorage.getItem('azure-yaml-editor-draft')).toBeTruthy()

      clearDraft()
      expect(localStorage.getItem('azure-yaml-editor-draft')).toBeNull()
    })

    it('should handle errors gracefully', () => {
      const originalRemoveItem = localStorage.removeItem
      localStorage.removeItem = vi.fn(() => {
        throw new Error('Storage error')
      })

      clearDraft()

      localStorage.removeItem = originalRemoveItem
    })
  })

  describe('getDraftAge', () => {
    it('should calculate draft age correctly', () => {
      const timestamp = Date.now() - 5000 // 5 seconds ago
      const draft = { config: {}, timestamp, dirty: true }

      const age = getDraftAge(draft)
      expect(age).toBeGreaterThanOrEqual(5000)
      expect(age).toBeLessThan(6000)
    })
  })

  describe('formatDraftAge', () => {
    it('should format seconds', () => {
      expect(formatDraftAge(30 * 1000)).toBe('30 seconds ago')
      expect(formatDraftAge(1 * 1000)).toBe('1 second ago')
    })

    it('should format minutes', () => {
      expect(formatDraftAge(5 * 60 * 1000)).toBe('5 minutes ago')
      expect(formatDraftAge(1 * 60 * 1000)).toBe('1 minute ago')
    })

    it('should format hours', () => {
      expect(formatDraftAge(3 * 60 * 60 * 1000)).toBe('3 hours ago')
      expect(formatDraftAge(1 * 60 * 60 * 1000)).toBe('1 hour ago')
    })

    it('should format days', () => {
      expect(formatDraftAge(2 * 24 * 60 * 60 * 1000)).toBe('2 days ago')
      expect(formatDraftAge(1 * 24 * 60 * 60 * 1000)).toBe('1 day ago')
    })
  })

  describe('isDraftStale', () => {
    it('should detect stale draft', () => {
      const eightDaysAgo = Date.now() - 8 * 24 * 60 * 60 * 1000
      const draft = { config: {}, timestamp: eightDaysAgo, dirty: true }

      expect(isDraftStale(draft)).toBe(true)
    })

    it('should detect fresh draft', () => {
      const oneDayAgo = Date.now() - 1 * 24 * 60 * 60 * 1000
      const draft = { config: {}, timestamp: oneDayAgo, dirty: true }

      expect(isDraftStale(draft)).toBe(false)
    })

    it('should handle edge case (exactly 7 days)', () => {
      const sevenDaysAgo = Date.now() - 7 * 24 * 60 * 60 * 1000
      const draft = { config: {}, timestamp: sevenDaysAgo, dirty: true }

      expect(isDraftStale(draft)).toBe(false)
    })
  })

  describe('cleanupStaleDrafts', () => {
    it('should remove stale draft', () => {
      const eightDaysAgo = Date.now() - 8 * 24 * 60 * 60 * 1000
      const staleDraft = { config: {}, timestamp: eightDaysAgo, dirty: true }
      localStorage.setItem('azure-yaml-editor-draft', JSON.stringify(staleDraft))

      cleanupStaleDrafts()

      expect(localStorage.getItem('azure-yaml-editor-draft')).toBeNull()
    })

    it('should keep fresh draft', () => {
      const oneDayAgo = Date.now() - 1 * 24 * 60 * 60 * 1000
      const freshDraft = { config: {}, timestamp: oneDayAgo, dirty: true }
      localStorage.setItem('azure-yaml-editor-draft', JSON.stringify(freshDraft))

      cleanupStaleDrafts()

      expect(localStorage.getItem('azure-yaml-editor-draft')).toBeTruthy()
    })
  })

  describe('AutoSaveManager', () => {
    it('should start auto-save', () => {
      const manager = new AutoSaveManager()
      const getConfig = vi.fn(() => ({ name: 'test' }))
      const isDirty = vi.fn(() => true)

      manager.start(getConfig, isDirty)

      // Advance 30 seconds
      vi.advanceTimersByTime(30000)

      expect(getConfig).toHaveBeenCalled()
      expect(isDirty).toHaveBeenCalled()

      manager.stop()
    })

    it('should stop auto-save', () => {
      const manager = new AutoSaveManager()
      const getConfig = vi.fn(() => ({ name: 'test' }))
      const isDirty = vi.fn(() => true)

      manager.start(getConfig, isDirty)
      manager.stop()

      // Advance time - should not save
      vi.advanceTimersByTime(60000)

      // getConfig may be called during start, but not after stop
      const callCount = getConfig.mock.calls.length
      vi.advanceTimersByTime(30000)
      expect(getConfig.mock.calls.length).toBe(callCount)
    })

    it('should only save when dirty', () => {
      const manager = new AutoSaveManager()
      const getConfig = vi.fn(() => ({ name: 'test' }))
      const isDirty = vi.fn(() => false)

      manager.start(getConfig, isDirty)

      vi.advanceTimersByTime(30000)

      expect(isDirty).toHaveBeenCalled()
      expect(getConfig).not.toHaveBeenCalled()

      manager.stop()
    })

    it('should trigger immediate save (debounced)', () => {
      const manager = new AutoSaveManager()
      const config = { name: 'test' }

      manager.triggerSave(config, true)

      // Not saved yet (debounced)
      expect(loadDraft()).toBeNull()

      // After debounce delay
      vi.advanceTimersByTime(1000)

      const draft = loadDraft()
      expect(draft).toBeTruthy()
      expect(draft!.config).toEqual(config)

      manager.stop()
    })

    it('should debounce multiple trigger saves', () => {
      const manager = new AutoSaveManager()

      manager.triggerSave({ name: 'test1' }, true)
      vi.advanceTimersByTime(500)
      manager.triggerSave({ name: 'test2' }, true)
      vi.advanceTimersByTime(500)
      manager.triggerSave({ name: 'test3' }, true)
      vi.advanceTimersByTime(1000)

      const draft = loadDraft()
      expect(draft!.config).toEqual({ name: 'test3' })

      manager.stop()
    })
  })
})
