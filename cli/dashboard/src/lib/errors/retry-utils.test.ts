/**
 * Retry Utilities Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  retryWithBackoff,
  createRetryState,
  formatRetryStatus,
} from './retry-utils'

describe('retry-utils', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  describe('retryWithBackoff', () => {
    it('should succeed on first attempt', async () => {
      const fn = vi.fn().mockResolvedValue('success')
      const result = await retryWithBackoff(fn)

      expect(result).toBe('success')
      expect(fn).toHaveBeenCalledTimes(1)
    })

    it('should retry on failure', async () => {
      const fn = vi
        .fn()
        .mockRejectedValueOnce(new Error('Network timeout'))
        .mockResolvedValueOnce('success')

      const promise = retryWithBackoff(fn, { maxRetries: 3 })

      // Advance timer for first retry
      await vi.advanceTimersByTimeAsync(1000)

      const result = await promise
      expect(result).toBe('success')
      expect(fn).toHaveBeenCalledTimes(2)
    })

    it('should retry with exponential backoff', async () => {
      const fn = vi
        .fn()
        .mockRejectedValueOnce(new Error('Network timeout'))
        .mockRejectedValueOnce(new Error('Network timeout'))
        .mockResolvedValueOnce('success')

      const promise = retryWithBackoff(fn, { maxRetries: 3, initialDelay: 1000 })

      // First retry: 1000ms
      await vi.advanceTimersByTimeAsync(1000)
      // Second retry: 2000ms
      await vi.advanceTimersByTimeAsync(2000)

      const result = await promise
      expect(result).toBe('success')
      expect(fn).toHaveBeenCalledTimes(3)
    })

    it('should respect maxDelay', async () => {
      const fn = vi
        .fn()
        .mockRejectedValueOnce(new Error('Network timeout'))
        .mockRejectedValueOnce(new Error('Network timeout'))
        .mockResolvedValueOnce('success')

      const promise = retryWithBackoff(fn, {
        maxRetries: 3,
        initialDelay: 1000,
        backoffMultiplier: 10,
        maxDelay: 1500,
      })

      // First retry: 1000ms
      await vi.advanceTimersByTimeAsync(1000)
      // Second retry: capped at 1500ms (not 10000ms)
      await vi.advanceTimersByTimeAsync(1500)

      const result = await promise
      expect(result).toBe('success')
    })

    it('should fail after max retries', async () => {
      const fn = vi.fn().mockRejectedValue(new Error('Network timeout'))

      const promise = retryWithBackoff(fn, { maxRetries: 2, initialDelay: 100 })
      // Prevent unhandled rejection warnings while advancing fake timers
      promise.catch(() => {})

      await vi.advanceTimersByTimeAsync(100) // First retry
      await vi.advanceTimersByTimeAsync(200) // Second retry

      await expect(promise).rejects.toThrow('Network timeout')
      expect(fn).toHaveBeenCalledTimes(3) // Initial + 2 retries
    })

    it('should not retry on 4xx errors', async () => {
      const error = { status: 404, message: 'Not found' }
      const fn = vi.fn().mockRejectedValue(error)

      await expect(retryWithBackoff(fn)).rejects.toEqual(error)
      expect(fn).toHaveBeenCalledTimes(1) // No retries
    })

    it('should retry on 5xx errors', async () => {
      const error = { status: 500, message: 'Server error' }
      const fn = vi.fn().mockRejectedValueOnce(error).mockResolvedValueOnce('success')

      const promise = retryWithBackoff(fn)
      await vi.advanceTimersByTimeAsync(1000)

      const result = await promise
      expect(result).toBe('success')
      expect(fn).toHaveBeenCalledTimes(2)
    })

    it('should retry on 429 rate limit', async () => {
      const error = { status: 429, message: 'Too many requests' }
      const fn = vi.fn().mockRejectedValueOnce(error).mockResolvedValueOnce('success')

      const promise = retryWithBackoff(fn)
      await vi.advanceTimersByTimeAsync(1000)

      const result = await promise
      expect(result).toBe('success')
    })

    it('should use custom shouldRetry', async () => {
      const fn = vi.fn().mockRejectedValue(new Error('Custom error'))
      const shouldRetry = vi.fn().mockReturnValue(false)

      await expect(retryWithBackoff(fn, { shouldRetry })).rejects.toThrow('Custom error')
      expect(shouldRetry).toHaveBeenCalledWith(expect.any(Error))
      expect(fn).toHaveBeenCalledTimes(1) // No retries
    })

    it('should call onRetry callback', async () => {
      const fn = vi
        .fn()
        .mockRejectedValueOnce(new Error('Error'))
        .mockResolvedValueOnce('success')
      const onRetry = vi.fn()

      const promise = retryWithBackoff(fn, { onRetry })
      
      // Advance timer for retry and await promise
      await vi.advanceTimersByTimeAsync(1000)
      await promise

      expect(onRetry).toHaveBeenCalledWith(1, expect.any(Error))
    })
  })

  describe('createRetryState', () => {
    it('should initialize with default state', () => {
      const state = createRetryState()
      expect(state.getStatus()).toEqual({
        attempt: 0,
        maxAttempts: 0,
        isRetrying: false,
      })
    })

    it('should set retrying state', () => {
      const state = createRetryState()
      const error = new Error('Test')
      state.setRetrying(2, 3, error)

      expect(state.getStatus()).toEqual({
        attempt: 2,
        maxAttempts: 3,
        isRetrying: true,
        error,
      })
    })

    it('should set complete state', () => {
      const state = createRetryState()
      state.setRetrying(2, 3)
      state.setComplete()

      expect(state.getStatus().isRetrying).toBe(false)
      expect(state.getStatus().attempt).toBe(2) // Preserved
    })

    it('should reset state', () => {
      const state = createRetryState()
      state.setRetrying(2, 3)
      state.reset()

      expect(state.getStatus()).toEqual({
        attempt: 0,
        maxAttempts: 0,
        isRetrying: false,
      })
    })
  })

  describe('formatRetryStatus', () => {
    it('should format retrying status', () => {
      const status = { attempt: 2, maxAttempts: 3, isRetrying: true }
      expect(formatRetryStatus(status)).toBe('Retrying... 2/3')
    })

    it('should return empty for non-retrying', () => {
      const status = { attempt: 0, maxAttempts: 0, isRetrying: false }
      expect(formatRetryStatus(status)).toBe('')
    })
  })
})

