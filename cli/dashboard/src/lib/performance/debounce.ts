/**
 * Debounce and Throttle Utilities
 * Performance optimization utilities for rate-limiting function calls
 */

import { useCallback, useEffect, useRef } from 'react'

/**
 * Debounce a function call
 * Delays execution until after specified wait time has elapsed since last call
 */
export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeoutId: ReturnType<typeof setTimeout> | null = null

  return function debounced(...args: Parameters<T>) {
    if (timeoutId !== null) {
      clearTimeout(timeoutId)
    }

    timeoutId = setTimeout(() => {
      func(...args)
    }, wait)
  }
}

/**
 * Debounce hook with cancellation
 * Returns debounced function and cancel function
 */
export function useDebounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): [(...args: Parameters<T>) => void, () => void] {
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const funcRef = useRef(func)

  // Update function ref when it changes
  useEffect(() => {
    funcRef.current = func
  }, [func])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (timeoutRef.current !== null) {
        clearTimeout(timeoutRef.current)
      }
    }
  }, [])

  const debouncedFunc = useCallback(
    (...args: Parameters<T>) => {
      if (timeoutRef.current !== null) {
        clearTimeout(timeoutRef.current)
      }

      timeoutRef.current = setTimeout(() => {
        funcRef.current(...args)
      }, wait)
    },
    [wait]
  )

  const cancel = useCallback(() => {
    if (timeoutRef.current !== null) {
      clearTimeout(timeoutRef.current)
      timeoutRef.current = null
    }
  }, [])

  return [debouncedFunc, cancel]
}

/**
 * Debounced value hook
 * Returns a debounced version of the value
 */
export function useDebouncedValue<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = React.useState<T>(value)

  React.useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value)
    }, delay)

    return () => {
      clearTimeout(handler)
    }
  }, [value, delay])

  return debouncedValue
}

/**
 * Throttle a function call
 * Ensures function is called at most once per specified interval
 */
export function throttle<T extends (...args: any[]) => any>(
  func: T,
  limit: number
): (...args: Parameters<T>) => void {
  let inThrottle: boolean = false

  return function throttled(...args: Parameters<T>) {
    if (!inThrottle) {
      func(...args)
      inThrottle = true
      setTimeout(() => {
        inThrottle = false
      }, limit)
    }
  }
}

/**
 * Throttle hook
 */
export function useThrottle<T extends (...args: any[]) => any>(
  func: T,
  limit: number
): (...args: Parameters<T>) => void {
  const funcRef = useRef(func)
  const inThrottleRef = useRef(false)

  useEffect(() => {
    funcRef.current = func
  }, [func])

  return useCallback(
    (...args: Parameters<T>) => {
      if (!inThrottleRef.current) {
        funcRef.current(...args)
        inThrottleRef.current = true
        setTimeout(() => {
          inThrottleRef.current = false
        }, limit)
      }
    },
    [limit]
  )
}

/**
 * Debounced callback hook
 * Similar to useCallback but with debouncing
 */
export function useDebouncedCallback<T extends (...args: any[]) => any>(
  callback: T,
  delay: number,
  deps: React.DependencyList = []
): [(...args: Parameters<T>) => void, () => void] {
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const cancel = useCallback(() => {
    if (timeoutRef.current !== null) {
      clearTimeout(timeoutRef.current)
      timeoutRef.current = null
    }
  }, [])

  const debouncedCallback = useCallback(
    (...args: Parameters<T>) => {
      cancel()
      timeoutRef.current = setTimeout(() => {
        callback(...args)
      }, delay)
    },
    [callback, delay, cancel, ...deps]
  )

  useEffect(() => {
    return cancel
  }, [cancel])

  return [debouncedCallback, cancel]
}

// Import React for useState
import * as React from 'react'
