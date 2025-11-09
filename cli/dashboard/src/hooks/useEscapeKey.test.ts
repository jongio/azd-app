import { describe, it, expect, vi } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useEscapeKey } from './useEscapeKey'

describe('useEscapeKey', () => {
  it('should call callback when Escape key is pressed and isOpen is true', () => {
    const mockCallback = vi.fn()
    renderHook(() => useEscapeKey(true, mockCallback))

    const event = new KeyboardEvent('keydown', { key: 'Escape' })
    window.dispatchEvent(event)

    expect(mockCallback).toHaveBeenCalledTimes(1)
  })

  it('should not call callback when isOpen is false', () => {
    const mockCallback = vi.fn()
    renderHook(() => useEscapeKey(false, mockCallback))

    const event = new KeyboardEvent('keydown', { key: 'Escape' })
    window.dispatchEvent(event)

    expect(mockCallback).not.toHaveBeenCalled()
  })

  it('should not call callback when other keys are pressed', () => {
    const mockCallback = vi.fn()
    renderHook(() => useEscapeKey(true, mockCallback))

    const event = new KeyboardEvent('keydown', { key: 'Enter' })
    window.dispatchEvent(event)

    expect(mockCallback).not.toHaveBeenCalled()
  })

  it('should not call callback when defaultPrevented is true and respectPreventDefault is true', () => {
    const mockCallback = vi.fn()
    renderHook(() => useEscapeKey(true, mockCallback, true))

    const event = new KeyboardEvent('keydown', { key: 'Escape', cancelable: true })
    event.preventDefault()
    window.dispatchEvent(event)

    expect(mockCallback).not.toHaveBeenCalled()
  })

  it('should call callback even when defaultPrevented is true if respectPreventDefault is false', () => {
    const mockCallback = vi.fn()
    renderHook(() => useEscapeKey(true, mockCallback, false))

    const event = new KeyboardEvent('keydown', { key: 'Escape', cancelable: true })
    event.preventDefault()
    window.dispatchEvent(event)

    expect(mockCallback).toHaveBeenCalledTimes(1)
  })

  it('should clean up event listener on unmount', () => {
    const mockCallback = vi.fn()
    const { unmount } = renderHook(() => useEscapeKey(true, mockCallback))

    unmount()

    const event = new KeyboardEvent('keydown', { key: 'Escape' })
    window.dispatchEvent(event)

    expect(mockCallback).not.toHaveBeenCalled()
  })

  it('should update callback when it changes', () => {
    const mockCallback1 = vi.fn()
    const mockCallback2 = vi.fn()
    const { rerender } = renderHook(
      ({ callback }) => useEscapeKey(true, callback),
      { initialProps: { callback: mockCallback1 } }
    )

    const event = new KeyboardEvent('keydown', { key: 'Escape' })
    window.dispatchEvent(event)
    expect(mockCallback1).toHaveBeenCalledTimes(1)

    rerender({ callback: mockCallback2 })
    window.dispatchEvent(event)
    expect(mockCallback2).toHaveBeenCalledTimes(1)
  })
})
