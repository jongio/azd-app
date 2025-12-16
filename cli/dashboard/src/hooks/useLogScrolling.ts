import { useState, useEffect, useRef } from 'react'
import type { LogEntry } from '@/components/LogsPane'

export function useLogScrolling(
  logs: LogEntry[],
  autoScrollEnabled: boolean,
  isPaused: boolean
) {
  const logsContainerRef = useRef<HTMLDivElement>(null)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const [isHovering, setIsHovering] = useState(false)

  // Auto-scroll - scroll the container, not the page
  // Pause auto-scroll when user is hovering over the logs
  useEffect(() => {
    if (autoScrollEnabled && !isPaused && !isHovering && logsContainerRef.current) {
      const container = logsContainerRef.current
      container.scrollTop = container.scrollHeight
    }
  }, [logs, autoScrollEnabled, isPaused, isHovering])

  return {
    logsContainerRef,
    logsEndRef,
    isHovering,
    setIsHovering,
  }
}
