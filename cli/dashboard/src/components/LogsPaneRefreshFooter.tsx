import type { ReactNode } from 'react'
import type { LogMode } from './ModeToggle'

export interface LogsPaneRefreshFooterProps {
  isCollapsed: boolean
  isPaused: boolean
  logMode: LogMode
}

export function LogsPaneRefreshFooter({
  isCollapsed,
  isPaused,
  logMode,
}: Readonly<LogsPaneRefreshFooterProps>): ReactNode {
  if (isCollapsed) {
    return null
  }

  // For Azure mode, show paused status
  // Connection status is managed by WebSocket internally
  const showPausedIndicator = isPaused && logMode === 'azure'

  if (!showPausedIndicator) {
    return null
  }

  return (
    <div className="flex items-center justify-center gap-2 px-3 py-1.5 text-xs border-t border-border bg-muted/30">
      <span className="w-2 h-2 rounded-full bg-yellow-500" />
      <span className="text-muted-foreground">
        Paused - log streaming stopped
      </span>
    </div>
  )
}
