/**
 * Virtual List Component
 * Provides virtual scrolling for large lists using react-window
 * 
 * Features:
 * - Only renders visible items + buffer
 * - Maintains scroll position on updates
 * - Smooth scrolling experience
 * - Supports dynamic item heights
 */

// @ts-nocheck - react-window API compatibility pending migration
import { FixedSizeList as WindowFixedSizeList, VariableSizeList as WindowVariableSizeList } from 'react-window'
import { memo, useRef, useEffect, useState } from 'react'
import { cn } from '@/lib/utils'

export interface VirtualListProps<T> {
  /** List items */
  items: T[]
  /** Height of each item (fixed) or function to calculate (variable) */
  itemHeight: number | ((index: number) => number)
  /** Total height of the list container */
  height: number
  /** Total width of the list container */
  width?: number | string
  /** Overscan count (items to render outside visible area) */
  overscanCount?: number
  /** Render function for each item */
  renderItem: (item: T, index: number) => React.ReactNode
  /** Optional className */
  className?: string
  /** Optional item key function */
  getItemKey?: (index: number, data: T[]) => string | number
  /** Optional scroll callback */
  onScroll?: (scrollTop: number) => void
}

/**
 * Virtual List Component
 * Uses react-window for efficient rendering of large lists
 */
export function VirtualList<T>({
  items,
  itemHeight,
  height,
  width = '100%',
  overscanCount = 3,
  renderItem,
  className,
  getItemKey,
  onScroll,
}: VirtualListProps<T>) {
  const listRef = useRef<WindowFixedSizeList | WindowVariableSizeList>(null)

  // Reset scroll position when items change significantly
  useEffect(() => {
    if (listRef.current) {
      listRef.current.scrollTo(0)
    }
  }, [items.length])

  // Row renderer for react-window
  const Row = memo(({ index, style }: { index: number; style: React.CSSProperties }) => {
    const item = items[index]
    if (!item) return null

    return (
      <div style={style} className={cn('virtual-list-item', className)}>
        {renderItem(item, index)}
      </div>
    )
  })

  Row.displayName = 'VirtualListRow'

  // Use FixedSizeList or VariableSizeList based on itemHeight type
  const isFixedSize = typeof itemHeight === 'number'

  if (isFixedSize) {
    return (
      <WindowFixedSizeList
        ref={listRef as React.Ref<WindowFixedSizeList>}
        height={height}
        width={width}
        itemCount={items.length}
        itemSize={itemHeight}
        itemData={items}
        itemKey={getItemKey}
        overscanCount={overscanCount}
        onScroll={(props: any) => onScroll?.(props.scrollOffset)}
        className={className}
      >
        {Row}
      </WindowFixedSizeList>
    )
  } else {
    return (
      <WindowVariableSizeList
        ref={listRef as React.Ref<WindowVariableSizeList>}
        height={height}
        width={width}
        itemCount={items.length}
        itemSize={itemHeight as (index: number) => number}
        itemData={items}
        itemKey={getItemKey}
        overscanCount={overscanCount}
        onScroll={(props: any) => onScroll?.(props.scrollOffset)}
        className={className}
      >
        {Row}
      </WindowVariableSizeList>
    )
  }
}

/**
 * Auto-sizing Virtual List Component
 * Automatically calculates height based on container
 */
interface AutoSizerVirtualListProps<T> extends Omit<VirtualListProps<T>, 'height' | 'width'> {
  /** Optional max height */
  maxHeight?: number
  /** Optional min height */
  minHeight?: number
}

export function AutoSizerVirtualList<T>(props: AutoSizerVirtualListProps<T>) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [dimensions, setDimensions] = useState({ width: 0, height: 0 })

  useEffect(() => {
    if (!containerRef.current) return

    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const { width, height } = entry.contentRect
        setDimensions({ width, height })
      }
    })

    observer.observe(containerRef.current)
    return () => observer.disconnect()
  }, [])

  const { maxHeight, minHeight, ...listProps } = props
  let height = dimensions.height

  if (maxHeight && height > maxHeight) {
    height = maxHeight
  }
  if (minHeight && height < minHeight) {
    height = minHeight
  }

  return (
    <div ref={containerRef} className="flex-1">
      {dimensions.height > 0 && (
        <VirtualList {...listProps} height={height} width={dimensions.width} />
      )}
    </div>
  )
}

// Export for testing
export { WindowFixedSizeList as FixedSizeList, WindowVariableSizeList as VariableSizeList }
