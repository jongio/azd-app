import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useClipboard } from './useClipboard'

// Mock navigator.clipboard
const mockWriteText = vi.fn()
Object.assign(navigator, {
  clipboard: {
    writeText: mockWriteText,
  },
})

describe('useClipboard', () => {
  beforeEach(() => {
    mockWriteText.mockClear()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.runOnlyPendingTimers()
    vi.useRealTimers()
  })

  it('should initialize with copiedField as null', () => {
    const { result } = renderHook(() => useClipboard())
    expect(result.current.copiedField).toBe(null)
  })

  it('should copy text to clipboard', async () => {
    mockWriteText.mockResolvedValue(undefined)
    const { result } = renderHook(() => useClipboard())

    await act(async () => {
      await result.current.copyToClipboard('test text', 'testField')
    })

    expect(mockWriteText).toHaveBeenCalledWith('test text')
    expect(result.current.copiedField).toBe('testField')
  })

  it('should reset copiedField after timeout', async () => {
    mockWriteText.mockResolvedValue(undefined)
    const { result } = renderHook(() => useClipboard())

    await act(async () => {
      await result.current.copyToClipboard('test text', 'testField')
    })

    expect(result.current.copiedField).toBe('testField')

    act(() => {
      vi.advanceTimersByTime(2000)
    })

    expect(result.current.copiedField).toBe(null)
  })

  it('should handle clipboard write errors', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    mockWriteText.mockRejectedValue(new Error('Clipboard error'))
    const { result } = renderHook(() => useClipboard())

    await act(async () => {
      await result.current.copyToClipboard('test text', 'testField')
    })

    expect(consoleError).toHaveBeenCalled()
    expect(result.current.copiedField).toBe(null)
    consoleError.mockRestore()
  })

  it('should use custom timeout', async () => {
    mockWriteText.mockResolvedValue(undefined)
    const { result } = renderHook(() => useClipboard(1000))

    await act(async () => {
      await result.current.copyToClipboard('test text', 'testField')
    })

    expect(result.current.copiedField).toBe('testField')

    act(() => {
      vi.advanceTimersByTime(1000)
    })

    expect(result.current.copiedField).toBe(null)
  })
})
