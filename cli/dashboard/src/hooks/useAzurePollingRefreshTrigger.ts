import { useEffect, useRef, useState } from 'react'
import type { LogMode } from '@/components/ModeToggle'

export function useAzurePollingRefreshTrigger(params: {
  syncInterval: number | undefined
  isPaused: boolean
  logMode: LogMode
  azureRealtime: boolean
}): { secondsUntilRefresh: number; refreshTrigger: number } {
  const { syncInterval, isPaused, logMode, azureRealtime } = params

  const [secondsUntilRefresh, setSecondsUntilRefresh] = useState<number>(0)
  const [refreshTrigger, setRefreshTrigger] = useState<number>(0)

  const refreshTriggerRef = useRef<number>(0)
  const nextRefreshAtMsRef = useRef<number | null>(null)

  useEffect(() => {
    // Only run countdown in Azure mode with polling (not realtime)
    if (!syncInterval || isPaused || logMode !== 'azure' || azureRealtime) {
      nextRefreshAtMsRef.current = null
      setSecondsUntilRefresh(0)
      return
    }

    const intervalMs = Math.max(1000, syncInterval)

    const computeSecondsRemaining = (nowMs: number): number => {
      const nextAt = nextRefreshAtMsRef.current
      if (!nextAt) return Math.ceil(intervalMs / 1000)
      return Math.max(0, Math.ceil((nextAt - nowMs) / 1000))
    }

    const ensureNextRefreshAt = (nowMs: number): void => {
      nextRefreshAtMsRef.current ??= nowMs + intervalMs
    }

    const tick = (): void => {
      const nowMs = Date.now()
      ensureNextRefreshAt(nowMs)

      // If the tab was backgrounded/throttled and we missed one or more intervals,
      // advance the schedule and emit refresh triggers.
      let nextAt = nextRefreshAtMsRef.current ?? nowMs + intervalMs
      let bumped = false

      // Cap catch-up to avoid long loops if system time jumps.
      for (let i = 0; i < 10 && nowMs >= nextAt; i++) {
        refreshTriggerRef.current += 1
        bumped = true
        nextAt += intervalMs
      }

      nextRefreshAtMsRef.current = nextAt

      if (bumped) {
        setRefreshTrigger(refreshTriggerRef.current)
      }

      setSecondsUntilRefresh(computeSecondsRemaining(nowMs))
    }

    // Initialize immediately.
    tick()

    const intervalId = globalThis.setInterval(tick, 1000)

    const onVisibilityChange = (): void => {
      // When a tab becomes visible again, timers may have been throttled.
      // Force a tick so overdue refreshes trigger immediately.
      if (document.visibilityState === 'visible') {
        tick()
      }
    }

    document.addEventListener('visibilitychange', onVisibilityChange)

    return () => {
      globalThis.clearInterval(intervalId)
      document.removeEventListener('visibilitychange', onVisibilityChange)
    }
  }, [syncInterval, isPaused, logMode, azureRealtime])

  return { secondsUntilRefresh, refreshTrigger }
}
