import { useEffect, useRef } from 'react'
import { X, Keyboard } from 'lucide-react'
import { useEscapeKey } from '../../hooks/useEscapeKey'
import {
  shortcuts,
  formatKey,
  getCategoryDisplayName,
  type ShortcutCategory,
  type Shortcut,
} from '../../lib/shortcuts-utils'

export interface KeyboardShortcutsProps {
  isOpen: boolean
  onClose: () => void
}

interface KeyBadgeProps {
  keyName: string
}

/**
 * Visual key representation badge
 */
function KeyBadge({ keyName }: KeyBadgeProps) {
  const formattedKey = formatKey(keyName)
  return (
    <kbd className="inline-flex items-center justify-center min-w-[28px] h-7 px-2 rounded bg-muted border border-border font-mono text-sm text-foreground">
      {formattedKey}
    </kbd>
  )
}

interface ShortcutRowProps {
  shortcut: Shortcut
}

/**
 * Single shortcut row with key(s) and description
 */
function ShortcutRow({ shortcut }: ShortcutRowProps) {
  const keys = Array.isArray(shortcut.key) ? shortcut.key : [shortcut.key]
  
  return (
    <div className="flex items-center justify-between py-2">
      <span className="text-sm text-foreground">{shortcut.description}</span>
      <div className="flex items-center gap-1">
        {keys.map((key, index) => (
          <span key={key} className="flex items-center gap-1">
            {index > 0 && <span className="text-muted-foreground text-xs mx-1">or</span>}
            <KeyBadge keyName={key} />
          </span>
        ))}
      </div>
    </div>
  )
}

interface ShortcutGroupProps {
  category: ShortcutCategory
  shortcuts: Shortcut[]
}

/**
 * Group of shortcuts with category heading
 */
function ShortcutGroup({ category, shortcuts }: ShortcutGroupProps) {
  return (
    <div className="mb-6 last:mb-0">
      <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider mb-3">
        {getCategoryDisplayName(category)}
      </h3>
      <div className="space-y-1 divide-y divide-border">
        {shortcuts.map((shortcut) => {
          const keyString = Array.isArray(shortcut.key) 
            ? shortcut.key.join('-') 
            : shortcut.key
          return (
            <ShortcutRow 
              key={`${category}-${keyString}`} 
              shortcut={shortcut} 
            />
          )
        })}
      </div>
    </div>
  )
}

/**
 * Modal displaying all keyboard shortcuts
 */
export function KeyboardShortcuts({ isOpen, onClose }: KeyboardShortcutsProps) {
  const modalRef = useRef<HTMLDivElement>(null)
  const previousActiveElement = useRef<HTMLElement | null>(null)
  
  // Close on escape
  useEscapeKey(onClose, isOpen)
  
  // Focus trap and restoration
  useEffect(() => {
    if (isOpen) {
      // Store currently focused element
      previousActiveElement.current = document.activeElement as HTMLElement
      
      // Focus the modal after opening
      setTimeout(() => {
        modalRef.current?.focus()
      }, 0)
    } else {
      // Restore focus when closing
      previousActiveElement.current?.focus()
    }
  }, [isOpen])
  
  // Handle tab trapping
  const handleKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === 'Tab') {
      const focusableElements = modalRef.current?.querySelectorAll(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      )
      
      if (!focusableElements || focusableElements.length === 0) {
        event.preventDefault()
        return
      }
      
      const firstElement = focusableElements[0] as HTMLElement
      const lastElement = focusableElements[focusableElements.length - 1] as HTMLElement
      
      if (event.shiftKey && document.activeElement === firstElement) {
        event.preventDefault()
        lastElement.focus()
      } else if (!event.shiftKey && document.activeElement === lastElement) {
        event.preventDefault()
        firstElement.focus()
      }
    }
  }
  
  // Group shortcuts by category
  const navigationShortcuts = shortcuts.filter(s => s.category === 'navigation')
  const actionShortcuts = shortcuts.filter(s => s.category === 'actions')
  const viewShortcuts = shortcuts.filter(s => s.category === 'views')
  
  if (!isOpen) {
    return null
  }
  
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      role="presentation"
    >
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-background/80 backdrop-blur-sm animate-in fade-in duration-200"
        onClick={onClose}
        data-testid="shortcuts-backdrop"
      />
      
      {/* Modal */}
      <div
        ref={modalRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="shortcuts-title"
        tabIndex={-1}
        onKeyDown={handleKeyDown}
        className="relative z-50 w-full max-w-lg mx-4 bg-popover border border-border rounded-lg shadow-lg animate-in zoom-in-95 duration-200 outline-none"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border">
          <div className="flex items-center gap-2">
            <Keyboard className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
            <h2 id="shortcuts-title" className="text-lg font-semibold">
              Keyboard Shortcuts
            </h2>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-md hover:bg-muted transition-colors"
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        
        {/* Content */}
        <div className="px-6 py-4 max-h-[60vh] overflow-y-auto">
          <ShortcutGroup category="navigation" shortcuts={navigationShortcuts} />
          <ShortcutGroup category="actions" shortcuts={actionShortcuts} />
          <ShortcutGroup category="views" shortcuts={viewShortcuts} />
        </div>
        
        {/* Footer */}
        <div className="px-6 py-3 border-t border-border bg-muted/50 rounded-b-lg">
          <p className="text-xs text-muted-foreground text-center">
            Press <KeyBadge keyName="?" /> anytime to show this dialog
          </p>
        </div>
      </div>
    </div>
  )
}

export default KeyboardShortcuts
