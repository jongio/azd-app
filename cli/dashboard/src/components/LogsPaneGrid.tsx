import { ReactNode, Children } from 'react'
import { cn } from '@/lib/utils'

interface LogsPaneGridProps {
  children: ReactNode
  columns: number
}

export function LogsPaneGrid({ children, columns }: LogsPaneGridProps) {
  // Calculate number of rows needed
  const childCount = Children.count(children)
  const rows = Math.ceil(childCount / columns)
  
  // Each pane takes equal height, filling the available container space
  // Use minmax to ensure panes don't shrink below a reasonable size but fit within viewport
  const paneMinHeight = '150px'
  const paneHeight = `minmax(${paneMinHeight}, calc((100% - ${(rows - 1) * 16}px) / ${rows}))`
  
  return (
    <div
      className={cn("grid gap-4 w-full h-full p-4 overflow-auto box-border")}
      style={{
        gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
        gridAutoRows: paneHeight
      } as React.CSSProperties}
    >
      {children}
    </div>
  )
}
