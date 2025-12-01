import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useClipboard } from './useClipboard'

describe('useClipboard', () => {
  const mockWriteText = vi.fn()

  beforeEach(() => {
    vi.useFakeTimers()
    Object.assign(navigator, {
      clipboard: {
        writeText: mockWriteText,
      },
    })
    mockWriteText.mockResolvedValue(undefined)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.clearAllMocks()
  })

  it('initializes with null copiedField', () => {
    const { result } = renderHook(() => useClipboard())

    expect(result.current.copiedField).toBeNull()
  })

  it('copies text to clipboard and sets copiedField', async () => {
    const { result } = renderHook(() => useClipboard())

    await act(async () => {
      await result.current.copyToClipboard('test-text', 'test-field')
    })

    expect(mockWriteText).toHaveBeenCalledWith('test-text')
    expect(result.current.copiedField).toBe('test-field')
  })

  it('clears copiedField after 2 seconds', async () => {
    const { result } = renderHook(() => useClipboard())

    await act(async () => {
      await result.current.copyToClipboard('test-text', 'test-field')
    })

    expect(result.current.copiedField).toBe('test-field')

    act(() => {
      vi.advanceTimersByTime(2000)
    })

    expect(result.current.copiedField).toBeNull()
  })

  it('does not clear copiedField before 2 seconds', async () => {
    const { result } = renderHook(() => useClipboard())

    await act(async () => {
      await result.current.copyToClipboard('test-text', 'test-field')
    })

    expect(result.current.copiedField).toBe('test-field')

    act(() => {
      vi.advanceTimersByTime(1999)
    })

    expect(result.current.copiedField).toBe('test-field')
  })

  it('updates copiedField when copying different fields', async () => {
    const { result } = renderHook(() => useClipboard())

    await act(async () => {
      await result.current.copyToClipboard('text-1', 'field-1')
    })

    expect(result.current.copiedField).toBe('field-1')

    await act(async () => {
      await result.current.copyToClipboard('text-2', 'field-2')
    })

    expect(result.current.copiedField).toBe('field-2')
  })

  it('handles empty text', async () => {
    const { result } = renderHook(() => useClipboard())

    await act(async () => {
      await result.current.copyToClipboard('', 'empty-field')
    })

    expect(mockWriteText).toHaveBeenCalledWith('')
    expect(result.current.copiedField).toBe('empty-field')
  })

  it('returns stable copyToClipboard function reference', () => {
    const { result, rerender } = renderHook(() => useClipboard())

    const firstRef = result.current.copyToClipboard
    rerender()
    const secondRef = result.current.copyToClipboard

    expect(firstRef).toBe(secondRef)
  })
})
