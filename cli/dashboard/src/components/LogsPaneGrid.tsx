import { ReactNode, Children, useMemo, isValidElement, useState, useRef, useLayoutEffect } from 'react'
import { cn } from '@/lib/utils'

interface LogsPaneGridProps {
  children: ReactNode
  columns: number
  collapsedPanes?: Record<string, boolean>
  /** When true, auto-calculates columns and rows to fit all panes on screen */
  autoFit?: boolean
}

/**
 * Calculate optimal columns to fit all panes on screen.
 * Aims to maximize pane size while fitting all on screen.
 */
function calculateAutoFitColumns(serviceCount: number, containerWidth: number, containerHeight: number): number {
  if (serviceCount <= 0) return 1
  if (serviceCount === 1) return 1
  if (serviceCount === 2) return 2
  
  // Try different column counts and pick the one that gives best pane size
  const gap = 16
  const padding = 16
  const minPaneWidth = 250
  const minPaneHeight = 120
  
  let bestColumns = 2
  let bestPaneArea = 0
  
  for (let cols = 2; cols <= Math.min(6, serviceCount); cols++) {
    const rows = Math.ceil(serviceCount / cols)
    
    // Calculate pane dimensions for this column count
    const totalHGaps = (cols - 1) * gap
    const totalVGaps = (rows - 1) * gap
    const availableWidth = containerWidth - (padding * 2) - totalHGaps
    const availableHeight = containerHeight - (padding * 2) - totalVGaps
    
    const paneWidth = availableWidth / cols
    const paneHeight = availableHeight / rows
    
    // Skip if panes would be too small
    if (paneWidth < minPaneWidth || paneHeight < minPaneHeight) continue
    
    // Calculate area - prefer layouts with larger panes
    const paneArea = paneWidth * paneHeight
    
    if (paneArea > bestPaneArea) {
      bestPaneArea = paneArea
      bestColumns = cols
    }
  }
  
  return bestColumns
}

/** Gap between grid items in pixels */
const GRID_GAP = 16
/** Padding on the grid container */
const GRID_PADDING = 16
/** Collapsed pane header height */
const COLLAPSED_HEIGHT = 48

export function LogsPaneGrid({ children, columns, collapsedPanes = {}, autoFit = false }: LogsPaneGridProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [containerSize, setContainerSize] = useState({ width: 1200, height: 800 })
  
  // Measure container size - use useLayoutEffect for synchronous measurement
  useLayoutEffect(() => {
    const updateSize = () => {
      if (containerRef.current) {
        // Get the parent's dimensions since we want to fill it
        const parent = containerRef.current.parentElement
        if (parent) {
          const rect = parent.getBoundingClientRect()
          setContainerSize({ width: rect.width, height: rect.height })
        }
      }
    }
    
    // Initial measurement
    updateSize()
    
    // Debounced resize handler
    let timeoutId: ReturnType<typeof setTimeout>
    const debouncedResize = () => {
      clearTimeout(timeoutId)
      timeoutId = setTimeout(updateSize, 50)
    }
    
    window.addEventListener('resize', debouncedResize)
    
    // Use ResizeObserver for accurate container size changes
    let resizeObserver: ResizeObserver | null = null
    if (typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver(debouncedResize)
      const parent = containerRef.current?.parentElement
      if (parent) {
        resizeObserver.observe(parent)
      }
    }
    
    return () => {
      clearTimeout(timeoutId)
      window.removeEventListener('resize', debouncedResize)
      resizeObserver?.disconnect()
    }
  }, [])
  
  const childArray = Children.toArray(children)
  const childCount = childArray.length
  
  // Calculate effective columns - either manual or auto-fit
  const effectiveColumns = autoFit 
    ? calculateAutoFitColumns(childCount, containerSize.width, containerSize.height)
    : columns
  
  const rows = Math.ceil(childCount / effectiveColumns)
  
  // Calculate which rows are collapsed
  const rowCollapsedState = useMemo(() => {
    const states: boolean[] = []
    
    for (let row = 0; row < rows; row++) {
      const startIdx = row * effectiveColumns
      const endIdx = Math.min(startIdx + effectiveColumns, childCount)
      const rowChildren = childArray.slice(startIdx, endIdx)
      
      // Check if ALL panes in this row are collapsed
      const allCollapsed = rowChildren.every((child) => {
        if (isValidElement(child)) {
          const serviceName = (child.props as { serviceName?: string }).serviceName
          return serviceName ? collapsedPanes[serviceName] : false
        }
        return false
      })
      states.push(allCollapsed)
    }
    
    return states
  }, [childArray, childCount, rows, effectiveColumns, collapsedPanes])
  
  // Calculate exact pixel heights for each row in autoFit mode
  const gridTemplateRows = useMemo(() => {
    if (!autoFit) {
      // Manual mode: use minmax for scrollable behavior
      return rowCollapsedState
        .map(collapsed => collapsed ? 'auto' : 'minmax(200px, 1fr)')
        .join(' ')
    }
    
    // Auto-fit mode: calculate exact pixel heights
    const collapsedCount = rowCollapsedState.filter(c => c).length
    const expandedCount = rows - collapsedCount
    
    if (expandedCount === 0) {
      return rowCollapsedState.map(() => `${COLLAPSED_HEIGHT}px`).join(' ')
    }
    
    // Calculate available height for expanded rows
    // Total height - padding - gaps - collapsed row heights
    const totalGaps = (rows - 1) * GRID_GAP
    const totalCollapsedHeight = collapsedCount * COLLAPSED_HEIGHT
    const availableForExpanded = containerSize.height - (GRID_PADDING * 2) - totalGaps - totalCollapsedHeight
    
    // Divide equally among expanded rows
    const expandedRowHeight = Math.floor(availableForExpanded / expandedCount)
    
    return rowCollapsedState
      .map(collapsed => collapsed ? `${COLLAPSED_HEIGHT}px` : `${expandedRowHeight}px`)
      .join(' ')
  }, [autoFit, rows, rowCollapsedState, containerSize.height])
  
  // In autoFit mode, use calculated height; otherwise let it be flexible
  const gridStyle: React.CSSProperties = {
    gridTemplateColumns: `repeat(${effectiveColumns}, minmax(0, 1fr))`,
    gridTemplateRows: gridTemplateRows,
    alignItems: 'stretch',
  }
  
  if (autoFit) {
    // Fixed height based on parent container
    gridStyle.height = `${containerSize.height}px`
  } else {
    gridStyle.height = '100%'
    gridStyle.minHeight = 0
  }
  
  return (
    <div
      ref={containerRef}
      className={cn(
        "grid gap-4 w-full p-4 box-border",
        autoFit ? "overflow-hidden" : "overflow-auto"
      )}
      style={gridStyle}
    >
      {children}
    </div>
  )
}
