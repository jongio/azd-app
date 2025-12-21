import { useState, useEffect, useRef } from 'react'

const LOADING_INDICATOR_DELAY_MS = 150
const LOADING_INDICATOR_MIN_VISIBLE_MS = 250

export interface UseSmoothedLoadingIndicatorOptions {
  /** Whether to show loading immediately without delay. Use for critical operations where user expects feedback. */
  immediate?: boolean
}

export function useSmoothedLoadingIndicator(
  isActive: boolean, 
  options?: UseSmoothedLoadingIndicatorOptions
): boolean {
  const { immediate = false } = options ?? {}
  const [isVisible, setIsVisible] = useState(false)
  const delayTimeoutRef = useRef<ReturnType<typeof globalThis.setTimeout> | null>(null)
  const hideTimeoutRef = useRef<ReturnType<typeof globalThis.setTimeout> | null>(null)
  const shownAtRef = useRef<number | null>(null)

  useEffect(() => {
    return () => {
      if (delayTimeoutRef.current !== null) {
        globalThis.clearTimeout(delayTimeoutRef.current)
        delayTimeoutRef.current = null
      }
      if (hideTimeoutRef.current !== null) {
        globalThis.clearTimeout(hideTimeoutRef.current)
        hideTimeoutRef.current = null
      }
    }
  }, [])

  useEffect(() => {
    if (delayTimeoutRef.current !== null) {
      globalThis.clearTimeout(delayTimeoutRef.current)
      delayTimeoutRef.current = null
    }

    if (hideTimeoutRef.current !== null) {
      globalThis.clearTimeout(hideTimeoutRef.current)
      hideTimeoutRef.current = null
    }

    if (isActive) {
      if (isVisible) {
        return
      }

      // Show immediately if requested, otherwise use delay to prevent flashing
      const delay = immediate ? 0 : LOADING_INDICATOR_DELAY_MS

      delayTimeoutRef.current = globalThis.setTimeout(() => {
        delayTimeoutRef.current = null
        shownAtRef.current = Date.now()
        setIsVisible(true)
      }, delay)

      return
    }

    if (!isVisible) {
      return
    }

    const shownAt = shownAtRef.current ?? Date.now()
    const elapsedMs = Date.now() - shownAt
    const remainingMs = LOADING_INDICATOR_MIN_VISIBLE_MS - elapsedMs
    if (remainingMs <= 0) {
      shownAtRef.current = null
      queueMicrotask(() => setIsVisible(false))
      return
    }

    hideTimeoutRef.current = globalThis.setTimeout(() => {
      hideTimeoutRef.current = null
      shownAtRef.current = null
      setIsVisible(false)
    }, remainingMs)
  }, [isActive, isVisible, immediate])

  return isVisible
}
