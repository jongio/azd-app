/**
 * useCommandPalette Hook Tests
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import { useCommandPalette } from './useCommandPalette'

describe('useCommandPalette', () => {
  beforeEach(() => {
    // Clear any keyboard event listeners
    document.body.innerHTML = ''
  })
  
  afterEach(() => {
    // Cleanup
  })
  
  describe('State Management', () => {
    it('should start closed', () => {
      const { result } = renderHook(() => useCommandPalette())
      
      expect(result.current.isOpen).toBe(false)
    })
    
    it('should open palette', () => {
      const { result } = renderHook(() => useCommandPalette())
      
      act(() => {
        result.current.open()
      })
      
      expect(result.current.isOpen).toBe(true)
    })
    
    it('should close palette', () => {
      const { result } = renderHook(() => useCommandPalette())
      
      act(() => {
        result.current.open()
      })
      
      expect(result.current.isOpen).toBe(true)
      
      act(() => {
        result.current.close()
      })
      
      expect(result.current.isOpen).toBe(false)
    })
    
    it('should toggle palette', () => {
      const { result } = renderHook(() => useCommandPalette())
      
      act(() => {
        result.current.toggle()
      })
      
      expect(result.current.isOpen).toBe(true)
      
      act(() => {
        result.current.toggle()
      })
      
      expect(result.current.isOpen).toBe(false)
    })
  })
  
  describe('Keyboard Shortcuts', () => {
    it('should open on Cmd+K (Mac)', async () => {
      const { result } = renderHook(() => useCommandPalette())
      
      const event = new KeyboardEvent('keydown', {
        key: 'k',
        metaKey: true,
        bubbles: true,
      })
      
      act(() => {
        window.dispatchEvent(event)
      })
      
      await waitFor(() => {
        expect(result.current.isOpen).toBe(true)
      })
    })
    
    it('should open on Ctrl+K (Windows/Linux)', async () => {
      const { result } = renderHook(() => useCommandPalette())
      
      const event = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: true,
        bubbles: true,
      })
      
      act(() => {
        window.dispatchEvent(event)
      })
      
      await waitFor(() => {
        expect(result.current.isOpen).toBe(true)
      })
    })
    
    it('should toggle on repeated Cmd+K', async () => {
      const { result } = renderHook(() => useCommandPalette())
      
      const createEvent = () => new KeyboardEvent('keydown', {
        key: 'k',
        metaKey: true,
        bubbles: true,
      })
      
      // First press - open
      act(() => {
        window.dispatchEvent(createEvent())
      })
      
      await waitFor(() => {
        expect(result.current.isOpen).toBe(true)
      })
      
      // Second press - close
      act(() => {
        window.dispatchEvent(createEvent())
      })
      
      await waitFor(() => {
        expect(result.current.isOpen).toBe(false)
      })
    })
    
    it('should prevent default browser behavior', () => {
      renderHook(() => useCommandPalette())
      
      const event = new KeyboardEvent('keydown', {
        key: 'k',
        metaKey: true,
        bubbles: true,
        cancelable: true,
      })
      
      let defaultPrevented = false
      event.preventDefault = () => {
        defaultPrevented = true
      }
      
      window.dispatchEvent(event)
      
      expect(defaultPrevented).toBe(true)
    })
    
    it('should not trigger on other key combinations', () => {
      const { result } = renderHook(() => useCommandPalette())
      
      const event = new KeyboardEvent('keydown', {
        key: 'k',
        shiftKey: true,
        bubbles: true,
      })
      
      act(() => {
        window.dispatchEvent(event)
      })
      
      expect(result.current.isOpen).toBe(false)
    })
    
    it('should not trigger on different keys', () => {
      const { result } = renderHook(() => useCommandPalette())
      
      const event = new KeyboardEvent('keydown', {
        key: 'j',
        metaKey: true,
        bubbles: true,
      })
      
      act(() => {
        window.dispatchEvent(event)
      })
      
      expect(result.current.isOpen).toBe(false)
    })
  })
  
  describe('Cleanup', () => {
    it('should remove event listener on unmount', () => {
      const { unmount } = renderHook(() => useCommandPalette())
      
      unmount()
      
      // Event listener should be removed
      const event = new KeyboardEvent('keydown', {
        key: 'k',
        metaKey: true,
        bubbles: true,
      })
      
      // This should not throw or cause issues
      window.dispatchEvent(event)
    })
  })
})
