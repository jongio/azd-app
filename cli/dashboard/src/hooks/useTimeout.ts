import * as React from 'react'

type TimerHandle = ReturnType<typeof globalThis.setTimeout>

type TimeoutCallback = () => void

type TimeoutMapKey = string | number

/**
 * Small helper to manage timeouts safely and clear them on unmount.
 */
export function useTimeout() {
  const timeoutsRef = React.useRef<Set<TimerHandle>>(new Set())

  const setTimeoutSafe = React.useCallback((fn: TimeoutCallback, delay: number) => {
    const id = globalThis.setTimeout(() => {
      timeoutsRef.current.delete(id)
      fn()
    }, delay)
    timeoutsRef.current.add(id)
    return id
  }, [])

  const clearTimeoutSafe = React.useCallback((id: TimerHandle) => {
    globalThis.clearTimeout(id)
    timeoutsRef.current.delete(id)
  }, [])

  const clearAllTimeouts = React.useCallback(() => {
    timeoutsRef.current.forEach(globalThis.clearTimeout)
    timeoutsRef.current.clear()
  }, [])

  React.useEffect(() => () => clearAllTimeouts(), [clearAllTimeouts])

  return { setTimeout: setTimeoutSafe, clearTimeout: clearTimeoutSafe, clearAllTimeouts }
}

/**
 * Manages keyed timeouts (e.g., one per toast) with auto-cleanup on unmount.
 */
export function useTimeoutMap() {
  const timeoutsRef = React.useRef<Map<TimeoutMapKey, TimerHandle>>(new Map())

  const set = React.useCallback((key: TimeoutMapKey, fn: TimeoutCallback, delay: number) => {
    const current = timeoutsRef.current.get(key)
    if (current) {
      clearTimeout(current)
    }
    const id = globalThis.setTimeout(() => {
      timeoutsRef.current.delete(key)
      fn()
    }, delay)
    timeoutsRef.current.set(key, id)
    return id
  }, [])

  const clear = React.useCallback((key: TimeoutMapKey) => {
    const existing = timeoutsRef.current.get(key)
    if (existing) {
      globalThis.clearTimeout(existing)
      timeoutsRef.current.delete(key)
    }
  }, [])

  const clearAll = React.useCallback(() => {
    timeoutsRef.current.forEach(globalThis.clearTimeout)
    timeoutsRef.current.clear()
  }, [])

  React.useEffect(() => () => clearAll(), [clearAll])

  return { set, clear, clearAll }
}
