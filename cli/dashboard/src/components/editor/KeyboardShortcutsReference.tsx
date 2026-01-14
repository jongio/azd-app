/**
 * Keyboard shortcuts reference modal
 * Displays all available keyboard shortcuts
 */

import * as React from 'react'
import { X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { getShortcutsRegistry, type KeyboardShortcut } from '@/lib/accessibility'

export interface KeyboardShortcutsReferenceProps {
  isOpen: boolean
  onClose: () => void
}

/**
 * KeyboardShortcutsReference component
 */
export function KeyboardShortcutsReference({ isOpen, onClose }: KeyboardShortcutsReferenceProps) {
  const registry = getShortcutsRegistry()
  const shortcuts = React.useMemo(() => registry.getAll(), [registry])

  const shortcutsByCategory = React.useMemo(() => {
    const categories: Record<string, KeyboardShortcut[]> = {
      general: [],
      navigation: [],
      editor: [],
      modal: [],
    }

    shortcuts.forEach((shortcut) => {
      categories[shortcut.category].push(shortcut)
    })

    return categories
  }, [shortcuts])

  const containerRef = React.useRef<HTMLDivElement>(null)

  // Focus trap
  React.useEffect(() => {
    if (!isOpen || !containerRef.current) return

    const focusableElements = containerRef.current.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    )

    const firstElement = focusableElements[0]
    const lastElement = focusableElements[focusableElements.length - 1]

    firstElement?.focus()

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }

      if (e.key === 'Tab') {
        if (e.shiftKey) {
          if (document.activeElement === firstElement) {
            e.preventDefault()
            lastElement?.focus()
          }
        } else {
          if (document.activeElement === lastElement) {
            e.preventDefault()
            firstElement?.focus()
          }
        }
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, onClose])

  if (!isOpen) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={onClose}
      aria-hidden={!isOpen}
    >
      <div
        ref={containerRef}
        role="dialog"
        aria-label="Keyboard shortcuts reference"
        aria-modal="true"
        className="bg-background border border-border rounded-lg shadow-lg max-w-2xl w-full max-h-[80vh] overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border">
          <h2 className="text-xl font-semibold">Keyboard Shortcuts</h2>
          <Button
            variant="ghost"
            size="sm"
            onClick={onClose}
            aria-label="Close keyboard shortcuts reference"
          >
            <X className="w-4 h-4" />
          </Button>
        </div>

        {/* Content */}
        <div className="overflow-y-auto max-h-[calc(80vh-80px)] px-6 py-4">
          {Object.entries(shortcutsByCategory).map(([category, categoryShortcuts]) => {
            if (categoryShortcuts.length === 0) return null

            return (
              <div key={category} className="mb-6 last:mb-0">
                <h3 className="text-lg font-medium mb-3 capitalize">{category}</h3>
                <div className="space-y-2">
                  {categoryShortcuts.map((shortcut) => (
                    <div
                      key={shortcut.id}
                      className="flex items-center justify-between py-2 px-3 rounded-md hover:bg-muted/50"
                    >
                      <span className="text-sm text-muted-foreground">{shortcut.description}</span>
                      <kbd className="px-2 py-1 text-xs font-semibold bg-muted border border-border rounded-md">
                        {registry.formatShortcut(shortcut)}
                      </kbd>
                    </div>
                  ))}
                </div>
              </div>
            )
          })}

          {/* Additional Help */}
          <div className="mt-8 pt-6 border-t border-border">
            <h3 className="text-lg font-medium mb-3">Navigation</h3>
            <div className="space-y-2 text-sm text-muted-foreground">
              <div className="flex items-center justify-between py-2 px-3 rounded-md hover:bg-muted/50">
                <span>Navigate through form fields</span>
                <kbd className="px-2 py-1 text-xs font-semibold bg-muted border border-border rounded-md">
                  Tab
                </kbd>
              </div>
              <div className="flex items-center justify-between py-2 px-3 rounded-md hover:bg-muted/50">
                <span>Navigate backwards</span>
                <kbd className="px-2 py-1 text-xs font-semibold bg-muted border border-border rounded-md">
                  Shift+Tab
                </kbd>
              </div>
              <div className="flex items-center justify-between py-2 px-3 rounded-md hover:bg-muted/50">
                <span>Submit form / Execute action</span>
                <kbd className="px-2 py-1 text-xs font-semibold bg-muted border border-border rounded-md">
                  Enter
                </kbd>
              </div>
              <div className="flex items-center justify-between py-2 px-3 rounded-md hover:bg-muted/50">
                <span>Navigate lists and menus</span>
                <kbd className="px-2 py-1 text-xs font-semibold bg-muted border border-border rounded-md">
                  Arrow Keys
                </kbd>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
