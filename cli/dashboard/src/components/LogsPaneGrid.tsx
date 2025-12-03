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
      
      // If all panes in row are collapsed, use auto height; otherwise use minmax for flexible height
      // minmax(200px, 1fr) ensures a minimum height but allows growth to fill available space
      rowTemplates.push(allCollapsed ? 'auto' : 'minmax(200px, 1fr)')
    }
    
    return rowTemplates.join(' ')
  }, [childArray, childCount, rows, columns, collapsedPanes])
  
  return (
    <div
      className={cn("grid gap-4 w-full p-4 box-border overflow-hidden")}
      style={{
        gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
        gridTemplateRows: gridTemplateRows,
        alignItems: 'stretch',
        height: '100%',
        minHeight: 0, // Critical for flex children to shrink properly
      } as React.CSSProperties}
    >
      {children}
    </div>
  )
}
