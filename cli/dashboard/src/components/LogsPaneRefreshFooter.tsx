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

  return (
    <div className="flex items-center justify-center gap-2 px-3 py-1.5 text-xs border-t border-border bg-muted/30">
      <RotateCw className="w-3 h-3 text-muted-foreground" />
      <span className="text-muted-foreground">
        Next refresh in <span className="font-medium text-foreground">{secondsUntilRefresh}s</span>
      </span>
    </div>
  )
}
