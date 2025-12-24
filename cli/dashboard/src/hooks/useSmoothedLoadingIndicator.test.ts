import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useSmoothedLoadingIndicator } from './useSmoothedLoadingIndicator'

describe('useSmoothedLoadingIndicator', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  describe('default behavior (delayed)', () => {
    it('should not show loading state immediately', () => {
      const { result } = renderHook(() => useSmoothedLoadingIndicator(true))
      
      expect(result.current).toBe(false)
    })

    it('should show loading state after 150ms delay', () => {
      const { result } = renderHook(() => useSmoothedLoadingIndicator(true))
      
      expect(result.current).toBe(false)
      
      act(() => {
        vi.advanceTimersByTime(150)
      })
      
      expect(result.current).toBe(true)
    })

    it('should not show loading if operation completes before delay', () => {
      const { result, rerender } = renderHook(
        ({ isActive }) => useSmoothedLoadingIndicator(isActive),
        { initialProps: { isActive: true } }
      )
      
      // Complete operation before delay
      act(() => {
        vi.advanceTimersByTime(100)
        rerender({ isActive: false })
      })
      
      // Advance past delay
      act(() => {
        vi.advanceTimersByTime(100)
      })
      
      expect(result.current).toBe(false)
    })

    it('should keep loading visible for at least 250ms', () => {
      const { result, rerender } = renderHook(
        ({ isActive }) => useSmoothedLoadingIndicator(isActive),
        { initialProps: { isActive: true } }
      )
      
      // Show loading
      act(() => {
        vi.advanceTimersByTime(150)
      })
      expect(result.current).toBe(true)
      
      // Complete operation immediately
      act(() => {
        rerender({ isActive: false })
      })
      
      // Should still be visible
      expect(result.current).toBe(true)
      
      // Should hide after 250ms total
      act(() => {
        vi.advanceTimersByTime(250)
      })
      expect(result.current).toBe(false)
    })
  })

  describe('immediate mode', () => {
    it('should show loading state immediately when immediate=true', () => {
      const { result } = renderHook(() => 
        useSmoothedLoadingIndicator(true, { immediate: true })
      )
      
      // Should show immediately (on next tick)
      act(() => {
        vi.advanceTimersByTime(1)
      })
      expect(result.current).toBe(true)
    })

    it('should not delay showing loading when immediate=true', () => {
      const { result } = renderHook(() => 
        useSmoothedLoadingIndicator(true, { immediate: true })
      )
      
      // Advance time by a small amount
      act(() => {
        vi.advanceTimersByTime(1)
      })
      expect(result.current).toBe(true)
    })

    it('should still enforce minimum visible duration with immediate=true', () => {
      const { result, rerender } = renderHook(
        ({ isActive }) => useSmoothedLoadingIndicator(isActive, { immediate: true }),
        { initialProps: { isActive: true } }
      )
      
      // Show loading immediately
      act(() => {
        vi.advanceTimersByTime(1)
      })
      expect(result.current).toBe(true)
      
      // Complete operation immediately
      act(() => {
        rerender({ isActive: false })
      })
      
      // Should still be visible
      expect(result.current).toBe(true)
      
      // Should hide after minimum visible duration (250ms from when it was shown)
      act(() => {
        vi.advanceTimersByTime(250)
      })
      expect(result.current).toBe(false)
    })

    it('should hide immediately if operation completes before becoming visible', () => {
      const { result, rerender } = renderHook(
        ({ isActive }) => useSmoothedLoadingIndicator(isActive, { immediate: true }),
        { initialProps: { isActive: true } }
      )
      
      // Complete before showing
      act(() => {
        rerender({ isActive: false })
      })
      
      // Should never show
      act(() => {
        vi.advanceTimersByTime(1000)
      })
      expect(result.current).toBe(false)
    })
  })

  describe('mode transitions', () => {
    it('should switch from delayed to immediate mode correctly', () => {
      const { result, rerender } = renderHook(
        ({ immediate }) => useSmoothedLoadingIndicator(true, { immediate }),
        { initialProps: { immediate: false } }
      )
      
      // Start with delay - should not show immediately
      expect(result.current).toBe(false)
      
      // Switch to immediate mode
      act(() => {
        rerender({ immediate: true })
      })
      
      // Should show immediately
      act(() => {
        vi.advanceTimersByTime(1)
      })
      expect(result.current).toBe(true)
    })

    it('should handle rapid activation/deactivation', () => {
      const { result, rerender } = renderHook(
        ({ isActive }) => useSmoothedLoadingIndicator(isActive, { immediate: true }),
        { initialProps: { isActive: true } }
      )
      
      // Activate
      act(() => {
        vi.advanceTimersByTime(1)
      })
      expect(result.current).toBe(true)
      
      // Rapid deactivate/activate
      act(() => {
        rerender({ isActive: false })
        rerender({ isActive: true })
      })
      
      // Should remain visible
      expect(result.current).toBe(true)
    })
  })

  describe('cleanup', () => {
    it('should cleanup timers on unmount', () => {
      const clearTimeoutSpy = vi.spyOn(globalThis, 'clearTimeout')
      
      const { unmount } = renderHook(() => useSmoothedLoadingIndicator(true))
      
      unmount()
      
      expect(clearTimeoutSpy).toHaveBeenCalled()
    })
  })
})
