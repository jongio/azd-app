/**
 * Navigation Search - Search/filter component for navigation sidebar
 * 
 * Features:
 * - Search input with clear button
 * - Keyboard accessible (Cmd/Ctrl+F to focus)
 * - Visual feedback
 */

import { useEffect, useRef } from 'react'
import { Search, X } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface NavigationSearchProps {
  /** Current search value */
  value: string
  /** Change handler */
  onChange: (value: string) => void
  /** Clear handler */
  onClear: () => void
  /** Custom className */
  className?: string
}

/**
 * Navigation Search Component
 */
export function NavigationSearch({
  value,
  onChange,
  onClear,
  className,
}: NavigationSearchProps) {
  const inputRef = useRef<HTMLInputElement>(null)

  // Global keyboard shortcut: Cmd/Ctrl+F
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'f') {
        e.preventDefault()
        inputRef.current?.focus()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  return (
    <div className={cn('px-3 py-2 border-b border-border', className)}>
      <div className="relative">
        {/* Search icon */}
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />

        {/* Search input */}
        <input
          ref={inputRef}
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="Search... (Ctrl+F)"
          className={cn(
            'w-full h-8 pl-8 pr-8 text-sm bg-background border border-input rounded-md',
            'placeholder:text-muted-foreground',
            'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-0',
            'transition-colors'
          )}
          aria-label="Search navigation"
        />

        {/* Clear button */}
        {value && (
          <button
            onClick={onClear}
            className={cn(
              'absolute right-2 top-1/2 -translate-y-1/2 p-0.5 rounded',
              'text-muted-foreground hover:text-foreground hover:bg-accent',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
              'transition-colors'
            )}
            aria-label="Clear search"
            tabIndex={-1}
          >
            <X className="w-3.5 h-3.5" />
          </button>
        )}
      </div>
    </div>
  )
}
