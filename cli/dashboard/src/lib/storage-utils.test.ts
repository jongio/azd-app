/**
 * @vitest-environment jsdom
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  getStorageItem,
  setStorageItem,
  removeStorageItem,
  isBoolean,
  isRecordOfBooleans,
  createStringArrayValidator,
} from './storage-utils'

describe('storage-utils', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    vi.spyOn(console, 'error').mockImplementation(() => {})
  })

  describe('getStorageItem', () => {
    it('returns default value when key does not exist', () => {
      const result = getStorageItem('nonexistent', 'default')
      expect(result).toBe('default')
    })

    it('returns parsed value when key exists', () => {
      localStorage.setItem('test-key', JSON.stringify({ foo: 'bar' }))
      const result = getStorageItem('test-key', {})
      expect(result).toEqual({ foo: 'bar' })
    })

    it('returns default value when JSON parsing fails', () => {
      localStorage.setItem('test-key', 'invalid json{')
      const result = getStorageItem('test-key', 'default')
      expect(result).toBe('default')
      expect(console.warn).toHaveBeenCalled()
    })

    it('returns default value when validator fails', () => {
      localStorage.setItem('test-key', JSON.stringify('not a boolean'))
      const result = getStorageItem('test-key', true, isBoolean)
      expect(result).toBe(true)
      expect(console.warn).toHaveBeenCalled()
    })

    it('returns parsed value when validator passes', () => {
      localStorage.setItem('test-key', JSON.stringify(false))
      const result = getStorageItem('test-key', true, isBoolean)
      expect(result).toBe(false)
    })

    it('handles arrays correctly', () => {
      localStorage.setItem('test-key', JSON.stringify(['a', 'b', 'c']))
      const result = getStorageItem<string[]>('test-key', [])
      expect(result).toEqual(['a', 'b', 'c'])
    })

    it('handles numbers correctly', () => {
      localStorage.setItem('test-key', JSON.stringify(42))
      const result = getStorageItem('test-key', 0)
      expect(result).toBe(42)
    })
  })

  describe('setStorageItem', () => {
    it('stores JSON stringified value', () => {
      setStorageItem('test-key', { foo: 'bar' })
      const stored = localStorage.getItem('test-key')
      expect(stored).toBe(JSON.stringify({ foo: 'bar' }))
    })

    it('stores arrays correctly', () => {
      setStorageItem('test-key', ['a', 'b', 'c'])
      const stored = localStorage.getItem('test-key')
      expect(stored).toBe(JSON.stringify(['a', 'b', 'c']))
    })

    it('stores primitive values correctly', () => {
      setStorageItem('test-number', 42)
      setStorageItem('test-string', 'hello')
      setStorageItem('test-boolean', true)
      
      expect(localStorage.getItem('test-number')).toBe('42')
      expect(localStorage.getItem('test-string')).toBe('"hello"')
      expect(localStorage.getItem('test-boolean')).toBe('true')
    })
  })

  describe('removeStorageItem', () => {
    it('removes item from localStorage', () => {
      localStorage.setItem('test-key', 'value')
      expect(localStorage.getItem('test-key')).toBe('value')
      
      removeStorageItem('test-key')
      expect(localStorage.getItem('test-key')).toBeNull()
    })

    it('does not throw when key does not exist', () => {
      expect(() => removeStorageItem('nonexistent')).not.toThrow()
    })
  })

  describe('isBoolean', () => {
    it('returns true for boolean values', () => {
      expect(isBoolean(true)).toBe(true)
      expect(isBoolean(false)).toBe(true)
    })

    it('returns false for non-boolean values', () => {
      expect(isBoolean('true')).toBe(false)
      expect(isBoolean(1)).toBe(false)
      expect(isBoolean(null)).toBe(false)
      expect(isBoolean(undefined)).toBe(false)
      expect(isBoolean({})).toBe(false)
    })
  })

  describe('isRecordOfBooleans', () => {
    it('returns true for valid Record<string, boolean>', () => {
      expect(isRecordOfBooleans({ a: true, b: false })).toBe(true)
      expect(isRecordOfBooleans({})).toBe(true)
    })

    it('returns false for invalid values', () => {
      expect(isRecordOfBooleans(null)).toBe(false)
      expect(isRecordOfBooleans(undefined)).toBe(false)
      expect(isRecordOfBooleans([])).toBe(false)
      expect(isRecordOfBooleans({ a: 'true' })).toBe(false)
      expect(isRecordOfBooleans({ a: 1 })).toBe(false)
      expect(isRecordOfBooleans('string')).toBe(false)
    })
  })

  describe('createStringArrayValidator', () => {
    const isValidStatus = createStringArrayValidator(['healthy', 'degraded', 'unhealthy'] as const)

    it('returns true for valid string arrays', () => {
      expect(isValidStatus(['healthy'])).toBe(true)
      expect(isValidStatus(['healthy', 'degraded'])).toBe(true)
      expect(isValidStatus(['healthy', 'degraded', 'unhealthy'])).toBe(true)
      expect(isValidStatus([])).toBe(true)
    })

    it('returns false for arrays with invalid values', () => {
      expect(isValidStatus(['invalid'])).toBe(false)
      expect(isValidStatus(['healthy', 'invalid'])).toBe(false)
    })

    it('returns false for non-arrays', () => {
      expect(isValidStatus(null)).toBe(false)
      expect(isValidStatus(undefined)).toBe(false)
      expect(isValidStatus('healthy')).toBe(false)
      expect(isValidStatus({})).toBe(false)
    })

    it('returns false for arrays with non-string values', () => {
      expect(isValidStatus([1, 2, 3])).toBe(false)
      expect(isValidStatus([true, false])).toBe(false)
    })
  })
})
