/**
 * TableSelector - Multi-select component for Log Analytics tables
 * Groups tables by category with search, recommended badges, and select all actions.
 */
import * as React from 'react'
import { Search, Check, ChevronDown, ChevronRight, Star, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TableInfo, TableCategory } from '@/hooks/useLogConfig'

// =============================================================================
// Types
// =============================================================================

export interface TableSelectorProps {
  /** Available tables to select from */
  tables: TableInfo[]
  /** Table categories for grouping */
  categories: TableCategory[]
  /** Currently selected table names */
  selectedTables: string[]
  /** Callback when selection changes */
  onSelectionChange: (tables: string[]) => void
  /** Recommended tables (highlighted) */
  recommendedTables?: string[]
  /** Whether the selector is disabled */
  disabled?: boolean
  /** Loading state */
  isLoading?: boolean
  /** Additional class names */
  className?: string
}

// =============================================================================
// Helper Functions
// =============================================================================

function groupTablesByCategory(
  tables: TableInfo[],
  categories: TableCategory[]
): Map<string, TableInfo[]> {
  const grouped = new Map<string, TableInfo[]>()
  
  // Initialize with known categories
  for (const cat of categories) {
    grouped.set(cat.name, [])
  }
  grouped.set('other', [])
  
  // Group tables
  for (const table of tables) {
    const category = table.category || 'other'
    const list = grouped.get(category) || grouped.get('other')!
    list.push(table)
  }
  
  // Remove empty categories
  for (const [key, value] of grouped) {
    if (value.length === 0) {
      grouped.delete(key)
    }
  }
  
  return grouped
}

function getCategoryDisplayName(
  categoryName: string,
  categories: TableCategory[]
): string {
  const cat = categories.find(c => c.name === categoryName)
  return cat?.displayName || categoryName.charAt(0).toUpperCase() + categoryName.slice(1)
}

// =============================================================================
// TableSelector Component
// =============================================================================

export function TableSelector({
  tables,
  categories,
  selectedTables,
  onSelectionChange,
  recommendedTables = [],
  disabled = false,
  isLoading = false,
  className,
}: TableSelectorProps) {
  const safeTables = React.useMemo(() => Array.isArray(tables) ? tables : [], [tables])
  const safeCategories = React.useMemo(() => Array.isArray(categories) ? categories : [], [categories])
  const safeSelectedTables = Array.isArray(selectedTables) ? selectedTables : []
  const safeRecommendedTables = Array.isArray(recommendedTables) ? recommendedTables : []

  const [searchQuery, setSearchQuery] = React.useState('')
  const [expandedCategories, setExpandedCategories] = React.useState<Set<string>>(
    new Set<string>()
  )

  // Initialize expanded categories once categories are available.
  React.useEffect(() => {
    setExpandedCategories((prev) => {
      if (prev.size > 0) return prev
      return new Set(safeCategories.map((c) => c.name))
    })
  }, [safeCategories])

  // Filter tables by search query
  const filteredTables = React.useMemo(() => {
    if (!searchQuery.trim()) {
      return safeTables
    }
    const query = searchQuery.toLowerCase()
    return safeTables.filter(
      t => t.name.toLowerCase().includes(query) ||
           t.description?.toLowerCase().includes(query) ||
           t.category?.toLowerCase().includes(query)
    )
  }, [safeTables, searchQuery])

  // Group filtered tables by category
  const groupedTables = React.useMemo(
    () => groupTablesByCategory(filteredTables, safeCategories),
    [filteredTables, safeCategories]
  )

  // Toggle category expansion
  const toggleCategory = (category: string) => {
    setExpandedCategories(prev => {
      const next = new Set(prev)
      if (next.has(category)) {
        next.delete(category)
      } else {
        next.add(category)
      }
      return next
    })
  }

  // Toggle single table selection
  const toggleTable = (tableName: string) => {
    if (disabled) return
    
    if (safeSelectedTables.includes(tableName)) {
      onSelectionChange(safeSelectedTables.filter(t => t !== tableName))
    } else {
      onSelectionChange([...safeSelectedTables, tableName])
    }
  }

  // Select all tables in a category
  const selectAllInCategory = (category: string) => {
    if (disabled) return
    
    const categoryTables = groupedTables.get(category) || []
    const categoryTableNames = categoryTables.map(t => t.name)
    const newSelection = new Set(safeSelectedTables)
    
    for (const name of categoryTableNames) {
      newSelection.add(name)
    }
    
    onSelectionChange(Array.from(newSelection))
  }

  // Deselect all tables in a category
  const deselectAllInCategory = (category: string) => {
    if (disabled) return
    
    const categoryTables = groupedTables.get(category) || []
    const categoryTableNames = new Set(categoryTables.map(t => t.name))
    
    onSelectionChange(safeSelectedTables.filter(t => !categoryTableNames.has(t)))
  }

  // Select all tables
  const selectAll = () => {
    if (disabled) return
    onSelectionChange(safeTables.map(t => t.name))
  }

  // Clear all selections
  const clearAll = () => {
    if (disabled) return
    onSelectionChange([])
  }

  // Select recommended tables
  const selectRecommended = () => {
    if (disabled) return
    onSelectionChange(safeRecommendedTables)
  }

  // Check if all tables in a category are selected
  const isCategoryFullySelected = (category: string): boolean => {
    const categoryTables = groupedTables.get(category) || []
    return categoryTables.every(t => selectedTables.includes(t.name))
  }

  // Check if some (but not all) tables in a category are selected
  const isCategoryPartiallySelected = (category: string): boolean => {
    const categoryTables = groupedTables.get(category) || []
    const selectedCount = categoryTables.filter(t => selectedTables.includes(t.name)).length
    return selectedCount > 0 && selectedCount < categoryTables.length
  }

  return (
    <div className={cn('space-y-3', className)}>
      {/* Header Actions */}
      <div className="flex items-center justify-between gap-2">
        <div className="text-sm text-slate-500 dark:text-slate-400">
          {selectedTables?.length ?? 0} of {tables?.length ?? 0} tables selected
        </div>
        <div className="flex items-center gap-2">
          {safeRecommendedTables.length > 0 && (
            <button
              type="button"
              onClick={selectRecommended}
              disabled={disabled}
              className={cn(
                'flex items-center gap-1 px-2 py-1 text-xs font-medium rounded',
                'text-amber-600 dark:text-amber-400',
                'hover:bg-amber-50 dark:hover:bg-amber-900/20',
                'disabled:opacity-50 disabled:cursor-not-allowed',
              )}
            >
              <Star className="w-3 h-3" />
              Recommended
            </button>
          )}
          <button
            type="button"
            onClick={selectAll}
            disabled={disabled}
            className={cn(
              'px-2 py-1 text-xs font-medium rounded',
              'text-slate-600 dark:text-slate-300',
              'hover:bg-slate-100 dark:hover:bg-slate-700',
              'disabled:opacity-50 disabled:cursor-not-allowed',
            )}
          >
            Select All
          </button>
          <button
            type="button"
            onClick={clearAll}
            disabled={disabled || (selectedTables?.length ?? 0) === 0}
            className={cn(
              'px-2 py-1 text-xs font-medium rounded',
              'text-slate-600 dark:text-slate-300',
              'hover:bg-slate-100 dark:hover:bg-slate-700',
              'disabled:opacity-50 disabled:cursor-not-allowed',
            )}
          >
            Clear
          </button>
        </div>
      </div>

      {/* Search Input */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder="Search tables..."
          disabled={disabled}
          className={cn(
            'w-full pl-9 pr-8 py-2 text-sm rounded-md',
            'bg-white dark:bg-slate-800',
            'border border-slate-200 dark:border-slate-600',
            'placeholder:text-slate-400 dark:placeholder:text-slate-500',
            'focus:outline-none focus:ring-2 focus:ring-cyan-500',
            'disabled:opacity-50 disabled:cursor-not-allowed',
          )}
        />
        {searchQuery && (
          <button
            type="button"
            onClick={() => setSearchQuery('')}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
          >
            <X className="w-4 h-4" />
          </button>
        )}
      </div>

      {/* Loading State */}
      {isLoading && (
        <div className="py-8 text-center text-slate-500 dark:text-slate-400">
          Loading tables...
        </div>
      )}

      {/* Table List */}
      {!isLoading && (
        <div className="border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden">
          <div className="max-h-[300px] overflow-y-auto">
            {Array.from(groupedTables.entries()).map(([category, categoryTables]) => (
              <div key={category} className="border-b border-slate-200 dark:border-slate-700 last:border-b-0">
                {/* Category Header */}
                <button
                  type="button"
                  onClick={() => toggleCategory(category)}
                  className={cn(
                    'w-full flex items-center justify-between px-3 py-2',
                    'bg-slate-50 dark:bg-slate-800/50',
                    'hover:bg-slate-100 dark:hover:bg-slate-800',
                    'text-left text-sm font-medium text-slate-700 dark:text-slate-200',
                  )}
                >
                  <div className="flex items-center gap-2">
                    {expandedCategories.has(category) ? (
                      <ChevronDown className="w-4 h-4" />
                    ) : (
                      <ChevronRight className="w-4 h-4" />
                    )}
                    <span>{getCategoryDisplayName(category, categories)}</span>
                    <span className="text-xs text-slate-400 dark:text-slate-500">
                      ({categoryTables.length})
                    </span>
                  </div>
                  
                  {/* Category selection indicator */}
                  <div className="flex items-center gap-2">
                    {isCategoryFullySelected(category) && (
                      <span className="text-xs text-cyan-600 dark:text-cyan-400">All selected</span>
                    )}
                    {isCategoryPartiallySelected(category) && (
                      <span className="text-xs text-slate-400">Some selected</span>
                    )}
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation()
                        if (isCategoryFullySelected(category)) {
                          deselectAllInCategory(category)
                        } else {
                          selectAllInCategory(category)
                        }
                      }}
                      disabled={disabled}
                      className={cn(
                        'p-1 rounded',
                        'hover:bg-slate-200 dark:hover:bg-slate-700',
                        'disabled:opacity-50 disabled:cursor-not-allowed',
                      )}
                      title={isCategoryFullySelected(category) ? 'Deselect all' : 'Select all'}
                    >
                      <Check className={cn(
                        'w-4 h-4',
                        isCategoryFullySelected(category) 
                          ? 'text-cyan-600 dark:text-cyan-400' 
                          : 'text-slate-400'
                      )} />
                    </button>
                  </div>
                </button>

                {/* Tables in Category */}
                {expandedCategories.has(category) && (
                  <div>
                    {categoryTables.map((table) => {
                      const isSelected = selectedTables.includes(table.name)
                      const isRecommended = safeRecommendedTables.includes(table.name)
                      
                      return (
                        <button
                          key={table.name}
                          type="button"
                          onClick={() => toggleTable(table.name)}
                          disabled={disabled}
                          className={cn(
                            'w-full flex items-start gap-3 px-3 py-2 text-left',
                            'hover:bg-slate-50 dark:hover:bg-slate-800/30',
                            'disabled:opacity-50 disabled:cursor-not-allowed',
                            isSelected && 'bg-cyan-50 dark:bg-cyan-900/20',
                          )}
                        >
                          {/* Checkbox */}
                          <div className={cn(
                            'mt-0.5 w-4 h-4 rounded border flex-shrink-0',
                            'flex items-center justify-center',
                            isSelected
                              ? 'bg-cyan-600 border-cyan-600'
                              : 'border-slate-300 dark:border-slate-600',
                          )}>
                            {isSelected && <Check className="w-3 h-3 text-white" />}
                          </div>
                          
                          {/* Table Info */}
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <span className={cn(
                                'font-mono text-sm',
                                isSelected 
                                  ? 'text-cyan-700 dark:text-cyan-300' 
                                  : 'text-slate-700 dark:text-slate-200',
                              )}>
                                {table.name}
                              </span>
                              {isRecommended && (
                                <span className={cn(
                                  'inline-flex items-center gap-1 px-1.5 py-0.5 text-[10px] font-semibold rounded',
                                  'bg-amber-100 dark:bg-amber-900/50 text-amber-700 dark:text-amber-300',
                                )}>
                                  <Star className="w-2.5 h-2.5" />
                                  Recommended
                                </span>
                              )}
                            </div>
                            {table.description && (
                              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 line-clamp-1">
                                {table.description}
                              </p>
                            )}
                          </div>
                        </button>
                      )
                    })}
                  </div>
                )}
              </div>
            ))}
          </div>

          {/* Empty State */}
          {filteredTables.length === 0 && !isLoading && (
            <div className="py-8 text-center text-slate-500 dark:text-slate-400">
              {searchQuery ? 'No tables match your search' : 'No tables available'}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default TableSelector
