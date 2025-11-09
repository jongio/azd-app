import { X, Command, Search, Terminal, Activity } from 'lucide-react'
import { useEscapeKey } from '@/hooks/useEscapeKey'

interface KeyboardShortcutsProps {
  isOpen: boolean
  onClose: () => void
}

interface Shortcut {
  keys: string[]
  description: string
  category: 'Navigation' | 'Actions' | 'Views'
}

const shortcuts: Shortcut[] = [
  // Navigation
  { keys: ['1'], description: 'Go to Resources view', category: 'Navigation' },
  { keys: ['2'], description: 'Go to Console view', category: 'Navigation' },
  { keys: ['3'], description: 'Go to Metrics view', category: 'Navigation' },
  { keys: ['4'], description: 'Go to Environment view', category: 'Navigation' },
  { keys: ['5'], description: 'Go to Actions view', category: 'Navigation' },
  { keys: ['6'], description: 'Go to Dependencies view', category: 'Navigation' },
  
  // Actions
  { keys: ['R'], description: 'Refresh all services', category: 'Actions' },
  { keys: ['C'], description: 'Clear console logs', category: 'Actions' },
  { keys: ['E'], description: 'Export logs', category: 'Actions' },
  { keys: ['/', 'Ctrl', 'F'], description: 'Focus search', category: 'Actions' },
  
  // Views
  { keys: ['T'], description: 'Toggle table/grid view', category: 'Views' },
  { keys: ['?'], description: 'Show keyboard shortcuts', category: 'Views' },
  { keys: ['Esc'], description: 'Close dialogs', category: 'Views' },
]

function KeyBadge({ keys }: { keys: string[] }) {
  return (
    <div className="flex items-center gap-1">
      {keys.map((key, index) => (
        <span key={index}>
          <kbd className="px-2 py-1 text-xs font-semibold text-foreground bg-[#0d0d0d] border border-[#2a2a2a] rounded shadow-sm">
            {key}
          </kbd>
          {index < keys.length - 1 && (
            <span className="mx-1 text-muted-foreground">+</span>
          )}
        </span>
      ))}
    </div>
  )
}

export function KeyboardShortcuts({ isOpen, onClose }: KeyboardShortcutsProps) {
  useEscapeKey(isOpen, onClose, false)
  
  if (!isOpen) return null

  const categories = ['Navigation', 'Actions', 'Views'] as const
  const groupedShortcuts = categories.map(category => ({
    category,
    shortcuts: shortcuts.filter(s => s.category === category)
  }))

  const categoryIcons = {
    Navigation: Activity,
    Actions: Command,
    Views: Search
  }

  return (
    <div 
      className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4"
      onClick={onClose}
    >
      <div 
        className="glass max-w-2xl w-full rounded-2xl border border-white/10 shadow-2xl overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-white/10 bg-[#0d0d0d]">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-primary/10">
              <Terminal className="w-5 h-5 text-primary" />
            </div>
            <div>
              <h2 className="text-xl font-semibold text-foreground">Keyboard Shortcuts</h2>
              <p className="text-sm text-muted-foreground mt-0.5">
                Quick access to common actions
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-2 hover:bg-white/5 rounded-lg transition-colors"
          >
            <X className="w-5 h-5 text-muted-foreground" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 max-h-[60vh] overflow-y-auto">
          <div className="space-y-6">
            {groupedShortcuts.map(({ category, shortcuts }) => {
              const Icon = categoryIcons[category]
              return (
                <div key={category}>
                  <div className="flex items-center gap-2 mb-3">
                    <Icon className="w-4 h-4 text-primary" />
                    <h3 className="text-sm font-semibold text-foreground uppercase tracking-wide">
                      {category}
                    </h3>
                  </div>
                  <div className="space-y-2">
                    {shortcuts.map((shortcut, index) => (
                      <div
                        key={index}
                        className="flex items-center justify-between p-3 rounded-lg hover:bg-white/5 transition-colors"
                      >
                        <span className="text-sm text-muted-foreground">
                          {shortcut.description}
                        </span>
                        <KeyBadge keys={shortcut.keys} />
                      </div>
                    ))}
                  </div>
                </div>
              )
            })}
          </div>
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-white/10 bg-[#0d0d0d] flex items-center justify-center">
          <p className="text-xs text-muted-foreground">
            Press <kbd className="px-1.5 py-0.5 text-xs bg-white/5 rounded border border-white/10">?</kbd> to toggle this panel
          </p>
        </div>
      </div>
    </div>
  )
}
