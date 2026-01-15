/**
 * useCommandPalette Hook
 * Manages command palette state and keyboard shortcuts
 */

import { useState, useEffect, useCallback } from 'react'

export interface UseCommandPaletteReturn {
  /** Whether palette is open */
  isOpen: boolean
  
  /** Open the palette */
  open: () => void
  
  /** Close the palette */
  close: () => void
  
  /** Toggle palette open/closed */
  toggle: () => void
}

/**
 * Hook to manage command palette state with Cmd/Ctrl+K shortcut
 */
export function useCommandPalette(): UseCommandPaletteReturn {
  const [isOpen, setIsOpen] = useState(false)
  
  const open = useCallback(() => setIsOpen(true), [])
  const close = useCallback(() => setIsOpen(false), [])
  const toggle = useCallback(() => setIsOpen((prev) => !prev), [])
  
  // Global keyboard shortcut (Cmd/Ctrl+K)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const key = e.key.toLowerCase()

      // Cmd+K (Mac) or Ctrl+K (Windows/Linux) toggles the palette
      if ((e.metaKey || e.ctrlKey) && key === 'k') {
        e.preventDefault()
        toggle()
      }
    }
    
    window.addEventListener('keydown', handleKeyDown)

    return () => {
      window.removeEventListener('keydown', handleKeyDown)
    }
  }, [toggle])
  
  return { isOpen, open, close, toggle }
}
