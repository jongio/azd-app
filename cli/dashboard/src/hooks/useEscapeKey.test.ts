import { describe, it, expect, vi, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useEscapeKey } from './useEscapeKey'

describe('useEscapeKey', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('calls onEscape when Escape key is pressed', () => {
    const onEscape = vi.fn()
    renderHook(() => useEscapeKey(onEscape))

    const event = new KeyboardEvent('keydown', { key: 'Escape' })
    document.dispatchEvent(event)

    expect(onEscape).toHaveBeenCalledTimes(1)
  })

  it('does not call onEscape for other keys', () => {
    const onEscape = vi.fn()
    renderHook(() => useEscapeKey(onEscape))

    const enterEvent = new KeyboardEvent('keydown', { key: 'Enter' })
    document.dispatchEvent(enterEvent)

    const spaceEvent = new KeyboardEvent('keydown', { key: ' ' })
    document.dispatchEvent(spaceEvent)

    const aEvent = new KeyboardEvent('keydown', { key: 'a' })
    document.dispatchEvent(aEvent)

    expect(onEscape).not.toHaveBeenCalled()
  })

  it('does not add listener when isActive is false', () => {
    const onEscape = vi.fn()
    renderHook(() => useEscapeKey(onEscape, false))

    const event = new KeyboardEvent('keydown', { key: 'Escape' })
    document.dispatchEvent(event)

    expect(onEscape).not.toHaveBeenCalled()
  })

  it('adds listener when isActive is true (default)', () => {
    const onEscape = vi.fn()
    renderHook(() => useEscapeKey(onEscape, true))

    const event = new KeyboardEvent('keydown', { key: 'Escape' })
    document.dispatchEvent(event)

    expect(onEscape).toHaveBeenCalledTimes(1)
  })

  it('removes listener on unmount', () => {
    const onEscape = vi.fn()
    const { unmount } = renderHook(() => useEscapeKey(onEscape))

    unmount()

    const event = new KeyboardEvent('keydown', { key: 'Escape' })
    document.dispatchEvent(event)

    expect(onEscape).not.toHaveBeenCalled()
  })

  it('removes listener when isActive changes to false', () => {
    const onEscape = vi.fn()
    const { rerender } = renderHook(
      ({ isActive }) => useEscapeKey(onEscape, isActive),
      { initialProps: { isActive: true } }
    )

    // Initially active
    const event1 = new KeyboardEvent('keydown', { key: 'Escape' })
    document.dispatchEvent(event1)
    expect(onEscape).toHaveBeenCalledTimes(1)

    // Deactivate
    rerender({ isActive: false })

    const event2 = new KeyboardEvent('keydown', { key: 'Escape' })
    document.dispatchEvent(event2)
    expect(onEscape).toHaveBeenCalledTimes(1) // Still 1, not 2
  })

  it('adds listener when isActive changes to true', () => {
    const onEscape = vi.fn()
    const { rerender } = renderHook(
      ({ isActive }) => useEscapeKey(onEscape, isActive),
      { initialProps: { isActive: false } }
    )

    // Initially inactive
    const event1 = new KeyboardEvent('keydown', { key: 'Escape' })
    document.dispatchEvent(event1)
    expect(onEscape).not.toHaveBeenCalled()

    // Activate
    rerender({ isActive: true })

    const event2 = new KeyboardEvent('keydown', { key: 'Escape' })
    document.dispatchEvent(event2)
    expect(onEscape).toHaveBeenCalledTimes(1)
  })

  it('updates handler when onEscape callback changes', () => {
    const onEscape1 = vi.fn()
    const onEscape2 = vi.fn()
    
    const { rerender } = renderHook(
      ({ onEscape }) => useEscapeKey(onEscape),
      { initialProps: { onEscape: onEscape1 } }
    )

    const event1 = new KeyboardEvent('keydown', { key: 'Escape' })
    document.dispatchEvent(event1)
    expect(onEscape1).toHaveBeenCalledTimes(1)
    expect(onEscape2).not.toHaveBeenCalled()

    // Change callback
    rerender({ onEscape: onEscape2 })

    const event2 = new KeyboardEvent('keydown', { key: 'Escape' })
    document.dispatchEvent(event2)
    expect(onEscape1).toHaveBeenCalledTimes(1) // Still 1
    expect(onEscape2).toHaveBeenCalledTimes(1)
  })

  it('handles multiple Escape key presses', () => {
    const onEscape = vi.fn()
    renderHook(() => useEscapeKey(onEscape))

    for (let i = 0; i < 5; i++) {
      const event = new KeyboardEvent('keydown', { key: 'Escape' })
      document.dispatchEvent(event)
    }

    expect(onEscape).toHaveBeenCalledTimes(5)
  })
})
