/**
 * Memoization Utilities
 * Provides utilities for memoizing expensive computations
 * 
 * Features:
 * - Schema parsing memoization
 * - Validation result caching
 * - Selector memoization
 */

import { useMemo, useRef } from 'react'
import type { ParsedSchema } from '@/lib/schema'

/**
 * Deep equality check for objects
 */
function deepEqual(a: any, b: any): boolean {
  if (a === b) return true
  if (a == null || b == null) return false
  if (typeof a !== 'object' || typeof b !== 'object') return false

  const keysA = Object.keys(a)
  const keysB = Object.keys(b)

  if (keysA.length !== keysB.length) return false

  for (const key of keysA) {
    if (!keysB.includes(key)) return false
    if (!deepEqual(a[key], b[key])) return false
  }

  return true
}

/**
 * Memoize a function with custom equality check
 */
export function memoize<T extends (...args: any[]) => any>(
  fn: T,
  equalityFn: (a: Parameters<T>, b: Parameters<T>) => boolean = (a, b) => deepEqual(a, b)
): T {
  const cache = new Map<string, ReturnType<T>>()
  let lastArgs: Parameters<T> | null = null
  let lastResult: ReturnType<T> | null = null

  return ((...args: Parameters<T>) => {
    // Check if args match last call
    if (lastArgs && equalityFn(args, lastArgs)) {
      return lastResult
    }

    // Check cache
    const key = JSON.stringify(args)
    if (cache.has(key)) {
      lastArgs = args
      lastResult = cache.get(key)!
      return lastResult
    }

    // Compute result
    const result = fn(...args)
    cache.set(key, result)
    lastArgs = args
    lastResult = result

    // Limit cache size
    if (cache.size > 100) {
      const firstKey = cache.keys().next().value
      if (firstKey !== undefined) {
        cache.delete(firstKey)
      }
    }

    return result
  }) as T
}

/**
 * Hook: Memoize with deep comparison
 */
export function useMemoDeep<T>(factory: () => T, deps: React.DependencyList): T {
  const ref = useRef<{ deps: React.DependencyList; value: T } | undefined>(undefined)

  if (!ref.current || !deepEqual(ref.current.deps, deps)) {
    ref.current = { deps, value: factory() }
  }

  return ref.current.value
}

/**
 * Hook: Memoized schema parsing
 */
export function useMemoizedSchemaParsing(
  schema: any,
  parseFunc: (schema: any) => ParsedSchema
): ParsedSchema | null {
  return useMemo(() => {
    if (!schema) {
      return null
    }

    return parseFunc(schema)
  }, [schema, parseFunc])
}

/**
 * Hook: Memoized validation
 */
export function useMemoizedValidation<T>(
  value: T,
  validateFunc: (value: T) => any,
  debounceMs: number = 500
): any {
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const lastValueRef = useRef<T | undefined>(undefined)
  const lastResultRef = useRef<any | undefined>(undefined)

  return useMemo(() => {
    // Cancel pending validation
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
    }

    // Return cached result if value hasn't changed
    if (lastValueRef.current === value && lastResultRef.current) {
      return lastResultRef.current
    }

    // Schedule new validation
    return new Promise((resolve) => {
      timeoutRef.current = setTimeout(() => {
        const result = validateFunc(value)
        lastValueRef.current = value
        lastResultRef.current = result
        resolve(result)
      }, debounceMs)
    })
  }, [value, validateFunc, debounceMs])
}

/**
 * LRU Cache implementation
 */
export class LRUCache<K, V> {
  private maxSize: number
  private cache: Map<K, V>

  constructor(maxSize: number = 100) {
    this.maxSize = maxSize
    this.cache = new Map()
  }

  get(key: K): V | undefined {
    if (!this.cache.has(key)) {
      return undefined
    }

    // Move to end (most recently used)
    const value = this.cache.get(key)!
    this.cache.delete(key)
    this.cache.set(key, value)
    return value
  }

  set(key: K, value: V): void {
    // Remove if exists (will re-add at end)
    if (this.cache.has(key)) {
      this.cache.delete(key)
    }

    // Add to end
    this.cache.set(key, value)

    // Evict oldest if over size
    if (this.cache.size > this.maxSize) {
      const firstKey = this.cache.keys().next().value
      if (firstKey !== undefined) {
        this.cache.delete(firstKey)
      }
    }
  }

  has(key: K): boolean {
    return this.cache.has(key)
  }

  clear(): void {
    this.cache.clear()
  }

  get size(): number {
    return this.cache.size
  }
}

/**
 * Create a memoized selector
 */
export function createSelector<T, R>(
  selector: (state: T) => R,
  equalityFn: (a: R, b: R) => boolean = (a, b) => a === b
): (state: T) => R {
  let lastState: T | undefined
  let lastResult: R | undefined

  return (state: T) => {
    if (lastState === state && lastResult !== undefined) {
      return lastResult
    }

    const result = selector(state)

    if (lastResult !== undefined && equalityFn(result, lastResult)) {
      return lastResult
    }

    lastState = state
    lastResult = result
    return result
  }
}
