/**
 * Toast System Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act, cleanup } from '@testing-library/react'
import { useToast } from './toast-system'

describe('toast-system', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    cleanup() // Clear React state between tests
  })

  describe('useToast', () => {
    it('should initialize with empty toasts', () => {
      const { result } = renderHook(() => useToast())
      
      // Clear any existing toasts first
      act(() => {
        result.current.clearAllToasts()
      })
      
      expect(result.current.toasts).toEqual([])
    })

    it('should add toast', () => {
      const { result } = renderHook(() => useToast())

      // Clear first
      act(() => {
        result.current.clearAllToasts()
      })

      act(() => {
        result.current.showToast({
          type: 'success',
          message: 'Test message',
        })
      })

      expect(result.current.toasts).toHaveLength(1)
      expect(result.current.toasts[0]).toMatchObject({
        type: 'success',
        message: 'Test message',
        duration: 5000,
      })
      expect(result.current.toasts[0].id).toBeTruthy()
    })

    it('should add multiple toasts', () => {
      const { result } = renderHook(() => useToast())

      act(() => {
        result.current.clearAllToasts()
        result.current.showToast({ type: 'success', message: 'First' })
        result.current.showToast({ type: 'error', message: 'Second' })
      })

      expect(result.current.toasts).toHaveLength(2)
    })

    it('should dismiss toast', () => {
      const { result } = renderHook(() => useToast())

      act(() => {
        result.current.clearAllToasts()
      })

      let toastId: string
      act(() => {
        toastId = result.current.showToast({ type: 'success', message: 'Test' })
      })

      expect(result.current.toasts).toHaveLength(1)

      act(() => {
        result.current.dismissToast(toastId)
      })

      expect(result.current.toasts).toHaveLength(0)
    })

    it('should auto-dismiss toast after duration', () => {
      const { result } = renderHook(() => useToast())

      act(() => {
        result.current.clearAllToasts()
      })

      act(() => {
        result.current.showToast({
          type: 'success',
          message: 'Test',
          duration: 3000,
        })
      })

      expect(result.current.toasts).toHaveLength(1)

      act(() => {
        vi.advanceTimersByTime(3000)
      })

      expect(result.current.toasts).toHaveLength(0)
    })

    it('should not auto-dismiss if duration is 0', () => {
      const { result } = renderHook(() => useToast())

      act(() => {
        result.current.clearAllToasts()
      })

      act(() => {
        result.current.showToast({
          type: 'error',
          message: 'Test',
          duration: 0,
        })
      })

      expect(result.current.toasts).toHaveLength(1)

      act(() => {
        vi.advanceTimersByTime(10000)
      })

      expect(result.current.toasts).toHaveLength(1)
    })

    it('should clear all toasts', () => {
      const { result } = renderHook(() => useToast())

      act(() => {
        result.current.clearAllToasts()
        result.current.showToast({ type: 'success', message: 'First' })
        result.current.showToast({ type: 'error', message: 'Second' })
      })

      expect(result.current.toasts).toHaveLength(2)

      act(() => {
        result.current.clearAllToasts()
      })

      expect(result.current.toasts).toHaveLength(0)
    })

    describe('convenience methods', () => {
      it('should show success toast', () => {
        const { result } = renderHook(() => useToast())

        act(() => {
          result.current.clearAllToasts()
          result.current.success('Success!', 'Description')
        })

        expect(result.current.toasts[0]).toMatchObject({
          type: 'success',
          message: 'Success!',
          description: 'Description',
        })
      })

      it('should show error toast', () => {
        const { result } = renderHook(() => useToast())

        act(() => {
          result.current.clearAllToasts()
          result.current.error('Error!', 'Description')
        })

        expect(result.current.toasts[0]).toMatchObject({
          type: 'error',
          message: 'Error!',
          description: 'Description',
        })
      })

      it('should show warning toast', () => {
        const { result } = renderHook(() => useToast())

        act(() => {
          result.current.clearAllToasts()
          result.current.warning('Warning!', 'Description')
        })

        expect(result.current.toasts[0]).toMatchObject({
          type: 'warning',
          message: 'Warning!',
          description: 'Description',
        })
      })

      it('should show info toast', () => {
        const { result } = renderHook(() => useToast())

        act(() => {
          result.current.clearAllToasts()
          result.current.info('Info!', 'Description')
        })

        expect(result.current.toasts[0]).toMatchObject({
          type: 'info',
          message: 'Info!',
          description: 'Description',
        })
      })
    })

    it('should support toast with action', () => {
      const { result } = renderHook(() => useToast())
      const actionFn = vi.fn()

      act(() => {
        result.current.clearAllToasts()
        result.current.showToast({
          type: 'success',
          message: 'Test',
          action: {
            label: 'Undo',
            onClick: actionFn,
          },
        })
      })

      expect(result.current.toasts[0].action).toMatchObject({
        label: 'Undo',
        onClick: actionFn,
      })
    })
  })
})
