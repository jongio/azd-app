/**
 * Command Palette
 * Global command palette with fuzzy search and keyboard navigation
 * 
 * Features:
 * - Modal overlay triggered by Cmd/Ctrl+K
 * - Fuzzy search with Fuse.js
 * - Keyboard navigation (arrow keys, enter, escape)
 * - Grouped results by category
 * - Recent command history
 * - Auto-focus search input
 */

import * as React from 'react'
import { X, Search, ArrowRight, Zap, Edit3, HelpCircle, Clock } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useEscapeKey } from '@/hooks/useEscapeKey'
import { VirtualList } from '@/lib/performance'
import { CommandSearch, groupResultsByCategory, filterRecentCommands, getHighlightedParts } from '@/lib/editor/command-search'
import { loadHistory, addToHistory, clearHistory } from '@/lib/editor/command-history'
import type { Command, CommandSearchResult, CommandHistory } from '@/lib/editor/command-types'

export interface CommandPaletteProps {
  /** Whether palette is open */
  isOpen: boolean
  
  /** Callback when palette should close */
  onClose: () => void
  
  /** Available commands */
  commands: Command[]
  
  /** Callback when navigation command is executed */
  onNavigate?: (path: string) => void
  
  /** Callback when field jump command is executed */
  onJumpToField?: (fieldPath: string) => void
  
  /** Callback when help command is executed */
  onOpenHelp?: (topic: string) => void
  
  /** Maximum results per category */
  maxPerCategory?: number
  
  /** Custom className */
  className?: string
}

// Category display configuration
const CATEGORY_CONFIG = {
  navigation: { label: 'Navigation', Icon: ArrowRight, color: 'text-blue-500' },
  action: { label: 'Actions', Icon: Zap, color: 'text-purple-500' },
  field: { label: 'Fields', Icon: Edit3, color: 'text-green-500' },
  // Use a non-overlapping label so tests and accessibility queries stay unique
  help: { label: 'Guides', Icon: HelpCircle, color: 'text-orange-500' },
} as const

/**
 * Command Palette Component
 */
export function CommandPalette({
  isOpen,
  onClose,
  commands,
  onNavigate,
  onJumpToField,
  onOpenHelp,
  maxPerCategory = 5,
  className,
}: CommandPaletteProps) {
  const [query, setQuery] = React.useState('')
  const [selectedIndex, setSelectedIndex] = React.useState(0)
  const [history, setHistory] = React.useState<CommandHistory>(() => loadHistory())
  
  const searchRef = React.useRef<HTMLInputElement>(null)
  const containerRef = React.useRef<HTMLDivElement>(null)
  const [searchEngine, setSearchEngine] = React.useState<CommandSearch>(() => new CommandSearch(commands))
  
  // Initialize or refresh search engine when command list changes
  React.useEffect(() => {
    setSearchEngine(new CommandSearch(commands))
  }, [commands])
  
  // Auto-focus search input when opened
  React.useEffect(() => {
    if (isOpen && searchRef.current) {
      searchRef.current.focus()
    }
  }, [isOpen])
  
  // Reset state when closed
  React.useEffect(() => {
    if (!isOpen) {
      setQuery('')
      setSelectedIndex(0)
    }
  }, [isOpen])
  
  // Search results
  const searchResults = React.useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase()
    const results = searchEngine.search(query)

    if (!normalizedQuery) {
      return results
    }

    const queryParts = normalizedQuery.split(/\s+/).filter(Boolean)

    const filtered = results.filter((result) => {
      const text = `${result.command.label} ${result.command.description ?? ''} ${(result.command.keywords ?? []).join(' ')}`.toLowerCase()

      if (queryParts.some((part) => !text.includes(part))) {
        return false
      }

      if (!normalizedQuery.includes('help') && result.command.category === 'help') {
        return false
      }

      return true
    })

    return filtered.length > 0 ? filtered : results
  }, [searchEngine, query])
  
  // Grouped results
  const groupedResults = React.useMemo(() => {
    return groupResultsByCategory(searchResults, maxPerCategory)
  }, [searchResults, maxPerCategory])
  
  // Recent commands (shown when no query)
  const recentResults = React.useMemo(() => {
    if (query.trim() || history.recent.length === 0) {
      return []
    }
    
    return filterRecentCommands(searchResults, history.recent)
  }, [searchResults, history.recent, query])
  
  // Flatten results for keyboard navigation
  const flatResults = React.useMemo(() => {
    if (recentResults.length > 0) {
      return recentResults
    }
    
    const flat: CommandSearchResult[] = []
    const categoryOrder: (keyof typeof CATEGORY_CONFIG)[] = ['navigation', 'action', 'field', 'help']
    
    for (const category of categoryOrder) {
      const results = groupedResults.get(category)
      if (results) {
        flat.push(...results)
      }
    }
    
    return flat
  }, [groupedResults, recentResults])
  
  // Reset selected index when results change
  React.useEffect(() => {
    setSelectedIndex(0)
  }, [flatResults])
  
  // Scroll selected item into view
  React.useEffect(() => {
    if (!containerRef.current) return
    
    const selectedElement = containerRef.current.querySelector('[data-selected="true"]')
    if (selectedElement) {
      selectedElement.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
    }
  }, [selectedIndex])
  
  // Execute command
  const executeCommand = React.useCallback((command: Command) => {
    // Add to history
    setHistory((prev) => addToHistory(command.id, prev))
    
    // Execute action
    switch (command.action.type) {
      case 'navigate':
        onNavigate?.(command.action.path)
        break
      
      case 'execute':
        command.action.handler()
        break
      
      case 'jump-to-field':
        onJumpToField?.(command.action.fieldPath)
        break
      
      case 'open-help':
        onOpenHelp?.(command.action.topic)
        break
    }
    
    // Close palette
    onClose()
  }, [onNavigate, onJumpToField, onOpenHelp, onClose])
  
  // Keyboard navigation
  const handleKeyDown = React.useCallback((e: React.KeyboardEvent) => {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        setSelectedIndex((prev) => Math.min(prev + 1, flatResults.length - 1))
        break
      
      case 'ArrowUp':
        e.preventDefault()
        setSelectedIndex((prev) => Math.max(prev - 1, 0))
        break
      
      case 'Enter':
        e.preventDefault()
        if (flatResults[selectedIndex]) {
          executeCommand(flatResults[selectedIndex].command)
        }
        break
      
      case 'Escape':
        e.preventDefault()
        onClose()
        break
    }
  }, [flatResults, selectedIndex, executeCommand, onClose])
  
  // Close on backdrop click
  const handleBackdropClick = React.useCallback(() => {
    onClose()
  }, [onClose])
  
  // Close on Escape key
  useEscapeKey(onClose, isOpen)
  
  // Clear history handler
  const handleClearHistory = React.useCallback(() => {
    setHistory(clearHistory())
  }, [])
  
  // Use virtual scrolling for large result sets
  const useVirtualScrolling = flatResults.length > 20
  
  if (!isOpen) {
    return null
  }
  
  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-50 bg-black/50 dark:bg-black/70 animate-fade-in"
        onClick={handleBackdropClick}
        aria-hidden="true"
      />
      
      {/* Command Palette */}
      <div
        className={cn(
          'fixed left-1/2 top-[20%] z-50 -translate-x-1/2',
          'w-full max-w-2xl',
          'bg-white dark:bg-slate-900',
          'border border-slate-200 dark:border-slate-700',
          'rounded-xl shadow-2xl',
          'flex flex-col',
          'max-h-[70vh]',
          'animate-scale-in',
          className
        )}
        onKeyDown={handleKeyDown}
        role="dialog"
        aria-modal="true"
        aria-label="Command palette"
        tabIndex={-1}
      >
        {/* Search Input */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-slate-200 dark:border-slate-700">
          <Search className="w-5 h-5 text-slate-400" aria-hidden="true" />
          <input
            ref={searchRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search commands..."
            className={cn(
              'flex-1 bg-transparent border-none outline-none',
              'text-slate-900 dark:text-slate-100',
              'placeholder:text-slate-400 dark:placeholder:text-slate-500',
              'text-base'
            )}
            aria-label="Search commands"
            autoComplete="off"
            spellCheck={false}
          />
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
            aria-label="Close command palette"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
        
        {/* Results */}
        <div ref={containerRef} className="flex-1 overflow-y-auto px-2 py-2">
          {/* Recent Commands */}
          {recentResults.length > 0 && (
            <div className="mb-3">
              <div className="flex items-center justify-between px-3 py-2">
                <div className="flex items-center gap-2 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wide">
                  <Clock className="w-3.5 h-3.5" />
                  Recent
                </div>
                <button
                  type="button"
                  onClick={handleClearHistory}
                  className="text-xs text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 transition-colors"
                >
                  Clear
                </button>
              </div>
              {useVirtualScrolling ? (
                <VirtualList
                  items={recentResults}
                  itemHeight={48}
                  height={Math.min(recentResults.length * 48, 400)}
                  width="100%"
                  overscanCount={3}
                  renderItem={(result, index) => (
                    <CommandResultItem
                      result={result}
                      isSelected={index === selectedIndex}
                      onClick={() => executeCommand(result.command)}
                    />
                  )}
                  getItemKey={(index, items) => items[index].command.id}
                />
              ) : (
                <div className="space-y-0.5">
                  {recentResults.map((result, index) => (
                    <CommandResultItem
                      key={result.command.id}
                      result={result}
                      isSelected={index === selectedIndex}
                      onClick={() => executeCommand(result.command)}
                    />
                  ))}
                </div>
              )}
            </div>
          )}
          
          {/* Grouped Results */}
          {recentResults.length === 0 && Array.from(groupedResults.entries()).map(([category, results]) => {
            const config = CATEGORY_CONFIG[category as keyof typeof CATEGORY_CONFIG]
            if (!config) return null
            
            const categoryStartIndex = flatResults.findIndex(r => r.command.category === category)
            
            return (
              <div key={category} className="mb-3">
                <div className="flex items-center gap-2 px-3 py-2 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wide">
                  <config.Icon className={cn('w-3.5 h-3.5', config.color)} />
                  {config.label}
                  {results.length < searchResults.filter(r => r.command.category === category).length && (
                    <span className="text-slate-400">
                      ({results.length} of {searchResults.filter(r => r.command.category === category).length})
                    </span>
                  )}
                </div>
                {useVirtualScrolling ? (
                  <VirtualList
                    items={results}
                    itemHeight={48}
                    height={Math.min(results.length * 48, 300)}
                    width="100%"
                    overscanCount={2}
                    renderItem={(result, index) => (
                      <CommandResultItem
                        result={result}
                        isSelected={categoryStartIndex + index === selectedIndex}
                        onClick={() => executeCommand(result.command)}
                      />
                    )}
                    getItemKey={(index, items) => items[index].command.id}
                  />
                ) : (
                  <div className="space-y-0.5">
                    {results.map((result, index) => (
                      <CommandResultItem
                        key={result.command.id}
                        result={result}
                        isSelected={categoryStartIndex + index === selectedIndex}
                        onClick={() => executeCommand(result.command)}
                      />
                    ))}
                  </div>
                )}
              </div>
            )
          })}
          
          {/* Empty State */}
          {flatResults.length === 0 && query && (
            <div className="flex flex-col items-center justify-center py-12 text-slate-500 dark:text-slate-400">
              <Search className="w-12 h-12 mb-3 opacity-30" />
              <p className="text-sm font-medium">No commands found</p>
              <p className="text-xs text-slate-400 dark:text-slate-500 mt-1">
                Try a different search term
              </p>
            </div>
          )}
        </div>
        
        {/* Footer */}
        <div className="flex items-center justify-between px-4 py-2 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/60 text-xs text-slate-500 dark:text-slate-400">
          <div className="flex items-center gap-4">
            <span>
              <kbd className="px-1.5 py-0.5 bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 rounded">↑↓</kbd> Navigate
            </span>
            <span>
              <kbd className="px-1.5 py-0.5 bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 rounded">Enter</kbd> Select
            </span>
            <span>
              <kbd className="px-1.5 py-0.5 bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 rounded">Esc</kbd> Close
            </span>
          </div>
          <div>
            {flatResults.length} {flatResults.length === 1 ? 'result' : 'results'}
          </div>
        </div>
      </div>
    </>
  )
}

/**
 * Command Result Item Component
 */
interface CommandResultItemProps {
  result: CommandSearchResult
  isSelected: boolean
  onClick: () => void
}

function CommandResultItem({ result, isSelected, onClick }: CommandResultItemProps) {
  const { command } = result
  const config = CATEGORY_CONFIG[command.category]
  const parts = getHighlightedParts(command.label, result.matches)
  
  return (
    <button
      type="button"
      onClick={onClick}
      data-selected={isSelected ? 'true' : undefined}
      className={cn(
        'w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left transition-colors',
        isSelected
          ? 'bg-cyan-100 dark:bg-cyan-900/30 text-cyan-900 dark:text-cyan-100'
          : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800'
      )}
    >
      {/* Icon */}
      {config && <config.Icon className={cn('w-4 h-4 shrink-0', isSelected ? 'text-cyan-600 dark:text-cyan-400' : config.color)} />}
      
      {/* Label and Description */}
      <div className="flex-1 min-w-0">
        <div className="font-medium text-sm">
          {parts.map((part, index) => part.highlight ? (
            <mark key={index} className="bg-amber-200 text-amber-900 rounded px-0.5">
              {part.text}
            </mark>
          ) : (
            <span key={index}>{part.text}</span>
          ))}
        </div>
        {command.description && (
          <div className="text-xs text-slate-500 dark:text-slate-400 truncate mt-0.5">
            {command.description}
          </div>
        )}
      </div>
      
      {/* Shortcut */}
      {command.shortcut && (
        <div className="shrink-0 text-xs text-slate-400 dark:text-slate-500">
          <kbd className="px-1.5 py-0.5 bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded">
            {command.shortcut}
          </kbd>
        </div>
      )}
    </button>
  )
}
