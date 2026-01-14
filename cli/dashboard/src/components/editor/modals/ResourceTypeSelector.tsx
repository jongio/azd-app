/**
 * Resource Type Selector
 * Visual grid for selecting Azure resource types
 */

import * as React from 'react'
import { Search } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { RESOURCE_TYPES, type ResourceType, type ResourceCategory } from '@/lib/editor/resource-types'

export interface ResourceTypeSelectorProps {
  /** Callback when resource type is selected */
  onSelect: (type: ResourceType) => void
  
  /** Filter by category */
  category?: ResourceCategory
}

/**
 * Resource Type Selector Component
 */
export function ResourceTypeSelector({ onSelect, category }: ResourceTypeSelectorProps) {
  const [searchQuery, setSearchQuery] = React.useState('')
  const [selectedCategory, setSelectedCategory] = React.useState<ResourceCategory | 'all'>(
    category || 'all'
  )

  // Filter resource types by search and category
  const filteredTypes = React.useMemo(() => {
    let types = RESOURCE_TYPES

    if (selectedCategory !== 'all') {
      types = types.filter(t => t.category === selectedCategory)
    }

    if (searchQuery) {
      const query = searchQuery.toLowerCase()
      types = types.filter(t =>
        t.displayName.toLowerCase().includes(query) ||
        t.description.toLowerCase().includes(query)
      )
    }

    return types
  }, [searchQuery, selectedCategory])

  // Group types by category
  const groupedTypes = React.useMemo(() => {
    const groups: Record<ResourceCategory, ResourceType[]> = {
      storage: [],
      database: [],
      messaging: [],
      compute: [],
      other: [],
    }

    for (const type of filteredTypes) {
      groups[type.category].push(type)
    }

    return groups
  }, [filteredTypes])

  const categories: Array<{ value: ResourceCategory | 'all'; label: string }> = [
    { value: 'all', label: 'All' },
    { value: 'storage', label: 'Storage' },
    { value: 'database', label: 'Database' },
    { value: 'messaging', label: 'Messaging' },
    { value: 'compute', label: 'Compute' },
    { value: 'other', label: 'Other' },
  ]

  return (
    <div className="space-y-4">
      {/* Search and Category Filter */}
      <div className="space-y-3">
        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
          <Input
            type="text"
            placeholder="Search resource types..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>

        {/* Category Tabs */}
        <div className="flex gap-2 flex-wrap">
          {categories.map((cat) => (
            <button
              key={cat.value}
              type="button"
              onClick={() => setSelectedCategory(cat.value)}
              className={cn(
                'px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
                selectedCategory === cat.value
                  ? 'bg-cyan-600 text-white'
                  : 'bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-700'
              )}
            >
              {cat.label}
            </button>
          ))}
        </div>
      </div>

      {/* Resource Type Grid */}
      <div className="max-h-96 overflow-y-auto">
        {selectedCategory === 'all' ? (
          // Show all categories
          <div className="space-y-4">
            {Object.entries(groupedTypes).map(([cat, types]) => {
              if (types.length === 0) return null

              return (
                <div key={cat}>
                  <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2 capitalize">
                    {cat}
                  </h3>
                  <div className="grid grid-cols-2 gap-3">
                    {types.map((type) => (
                      <ResourceTypeCard
                        key={type.id}
                        type={type}
                        onSelect={onSelect}
                      />
                    ))}
                  </div>
                </div>
              )
            })}
          </div>
        ) : (
          // Show selected category only
          <div className="grid grid-cols-2 gap-3">
            {filteredTypes.map((type) => (
              <ResourceTypeCard
                key={type.id}
                type={type}
                onSelect={onSelect}
              />
            ))}
          </div>
        )}

        {filteredTypes.length === 0 && (
          <div className="text-center py-8 text-slate-500 dark:text-slate-400">
            <p className="text-sm">No resource types found</p>
            <p className="text-xs mt-1">Try a different search or category</p>
          </div>
        )}
      </div>
    </div>
  )
}

interface ResourceTypeCardProps {
  type: ResourceType
  onSelect: (type: ResourceType) => void
}

function ResourceTypeCard({ type, onSelect }: ResourceTypeCardProps) {
  return (
    <button
      type="button"
      onClick={() => onSelect(type)}
      className={cn(
        'flex items-start gap-3 p-4 rounded-lg border-2 transition-all text-left',
        'border-slate-200 dark:border-slate-700',
        'hover:border-cyan-500 hover:bg-cyan-50 dark:hover:bg-cyan-950/20',
        'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2'
      )}
    >
      <span className="text-2xl shrink-0" aria-hidden="true">
        {type.icon || '📦'}
      </span>
      <div className="flex-1 min-w-0">
        <div className="font-semibold text-sm text-slate-900 dark:text-slate-100">
          {type.displayName}
        </div>
        <div className="text-xs text-slate-600 dark:text-slate-400 mt-0.5">
          {type.description}
        </div>
      </div>
    </button>
  )
}
