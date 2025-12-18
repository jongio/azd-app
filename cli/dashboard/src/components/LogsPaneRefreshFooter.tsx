import type { ReactNode } from 'react'
import { RotateCw } from 'lucide-react'

export interface LogsPaneRefreshFooterProps {
  isCollapsed: boolean
  syncInterval?: number
  isPaused: boolean
  secondsUntilRefresh: number
}

export function LogsPaneRefreshFooter({
  isCollapsed,
  syncInterval,
  isPaused,
  secondsUntilRefresh,
}: Readonly<LogsPaneRefreshFooterProps>): ReactNode {
  if (isCollapsed || !syncInterval || isPaused || secondsUntilRefresh <= 0) {
    return null
  }

  // Format countdown: HH:MM:SS if >= 1 hour, MM:SS if > 60 seconds, otherwise "{n}s"
  const formatCountdown = (seconds: number): string => {
    if (seconds > 60) {
      const hours = Math.floor(seconds / 3600)
      const minutes = Math.floor((seconds % 3600) / 60)
      const secs = seconds % 60
      
      if (hours > 0) {
        return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`
      }
      return `${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`
    }
    return `${seconds}s`
  }

  return (
    <div className="flex items-center justify-center gap-2 px-3 py-1.5 text-xs border-t border-border bg-muted/30">
      <RotateCw className="w-3 h-3 text-muted-foreground" />
      <span className="text-muted-foreground">
        Next refresh in <span className="font-medium text-foreground">{formatCountdown(secondsUntilRefresh)}</span>
      </span>
    </div>
  )
}
