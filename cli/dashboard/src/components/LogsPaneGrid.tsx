import { ReactNode, Children, useMemo, isValidElement } from 'react'
import { cn } from '@/lib/utils'

interface LogsPaneGridProps {
  children: ReactNode
  columns: number
  collapsedPanes?: Record<string, boolean>
}

export function LogsPaneGrid({ children, columns, collapsedPanes = {} }: LogsPaneGridProps) {
  const childArray = Children.toArray(children)
  const childCount = childArray.length
  const rows = Math.ceil(childCount / columns)
  
  // Calculate which rows have all panes collapsed vs expanded
  const gridTemplateRows = useMemo(() => {
    const rowTemplates: string[] = []
    
    for (let row = 0; row < rows; row++) {
      const startIdx = row * columns
      const endIdx = Math.min(startIdx + columns, childCount)
      const rowChildren = childArray.slice(startIdx, endIdx)
      
      // Check if ALL panes in this row are collapsed
      const allCollapsed = rowChildren.every((child) => {
        if (isValidElement(child)) {
          const serviceName = (child.props as { serviceName?: string }).serviceName
          return serviceName ? collapsedPanes[serviceName] : false
        }
        return false
      })
      
      // If all panes in row are collapsed, use auto height; otherwise use 1fr
      rowTemplates.push(allCollapsed ? 'auto' : '1fr')
    }
    
    return rowTemplates.join(' ')
  }, [childArray, childCount, rows, columns, collapsedPanes])
  
  return (
    <div
      className={cn("grid gap-4 w-full h-full p-4 overflow-auto box-border")}
      style={{
        gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
        gridTemplateRows: gridTemplateRows,
        alignItems: 'stretch'
      } as React.CSSProperties}
    >
      {children}
    </div>
  )
}
