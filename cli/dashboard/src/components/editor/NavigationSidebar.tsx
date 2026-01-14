/**
 * Navigation Sidebar - Hierarchical navigation for Azure YAML Editor
 * 
 * Features:
 * - Tree structure reflecting azure.yaml organization
 * - Active section highlighting
 * - Validation badges (error/warning indicators)
 * - Keyboard navigation (arrow keys, enter)
 * - Search/filter functionality
 * - Collapsible sections
 */

import { useState, useCallback, useRef, useEffect } from 'react'
import { ChevronRight, Plus } from 'lucide-react'
import { cn } from '@/lib/utils'
import { NavigationItem } from './NavigationItem'
import { NavigationSearch } from './NavigationSearch'
import type { NavigationNode, ValidationIssue } from '@/lib/editor/navigation-types'

export interface NavigationSidebarProps {
  /** Navigation tree structure */
  nodes: NavigationNode[]
  /** Currently active section path */
  activeSection: string
  /** Validation issues mapped by section path */
  validationIssues?: Map<string, ValidationIssue[]>
  /** Callback when navigation item is clicked */
  onNavigate: (path: string) => void
  /** Callback when add button is clicked */
  onAdd?: (type: 'service' | 'resource', parentPath: string) => void
  /** Whether sidebar is collapsed */
  isCollapsed?: boolean
  /** Callback to toggle sidebar */
  onToggleCollapse?: () => void
  /** Custom className */
  className?: string
}

/**
 * Navigation Sidebar Component
 */
export function NavigationSidebar({
  nodes,
  activeSection,
  validationIssues = new Map(),
  onNavigate,
  onAdd,
  isCollapsed = false,
  onToggleCollapse,
  className,
}: NavigationSidebarProps) {
  const [searchQuery, setSearchQuery] = useState('')
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set(['services', 'resources']))
  const [focusedIndex, setFocusedIndex] = useState(0)
  const navRef = useRef<HTMLDivElement>(null)

  // Filter nodes based on search query
  const filteredNodes = useCallback((nodes: NavigationNode[]): NavigationNode[] => {
    if (!searchQuery) return nodes

    const query = searchQuery.toLowerCase()

    const filterNode = (node: NavigationNode): NavigationNode | null => {
      const matchesQuery = node.label.toLowerCase().includes(query)
      
      if (node.children) {
        const filteredChildren = node.children
          .map(filterNode)
          .filter((n): n is NavigationNode => n !== null)
        
        if (filteredChildren.length > 0 || matchesQuery) {
          return {
            ...node,
            children: filteredChildren,
          }
        }
      }

      return matchesQuery ? node : null
    }

    return nodes.map(filterNode).filter((n): n is NavigationNode => n !== null)
  }, [searchQuery])

  const filtered = filteredNodes(nodes)

  // Flatten nodes for keyboard navigation
  const flattenNodes = useCallback((nodes: NavigationNode[], parentPath = ''): { node: NavigationNode; path: string; depth: number }[] => {
    const result: { node: NavigationNode; path: string; depth: number }[] = []
    
    const flatten = (nodes: NavigationNode[], depth = 0, parentPath = '') => {
      nodes.forEach((node) => {
        const path = parentPath ? `${parentPath}.${node.id}` : node.id
        result.push({ node, path, depth })
        
        if (node.children && expandedPaths.has(path)) {
          flatten(node.children, depth + 1, path)
        }
      })
    }
    
    flatten(nodes, 0, parentPath)
    return result
  }, [expandedPaths])

  const flatNodes = flattenNodes(filtered)

  // Toggle expanded state
  const toggleExpanded = useCallback((path: string) => {
    setExpandedPaths((prev) => {
      const next = new Set(prev)
      if (next.has(path)) {
        next.delete(path)
      } else {
        next.add(path)
      }
      return next
    })
  }, [])

  // Keyboard navigation
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (flatNodes.length === 0) return

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        setFocusedIndex((prev) => Math.min(prev + 1, flatNodes.length - 1))
        break

      case 'ArrowUp':
        e.preventDefault()
        setFocusedIndex((prev) => Math.max(prev - 1, 0))
        break

      case 'ArrowRight': {
        e.preventDefault()
        const { path, node } = flatNodes[focusedIndex]
        if (node.children && !expandedPaths.has(path)) {
          toggleExpanded(path)
        }
        break
      }

      case 'ArrowLeft': {
        e.preventDefault()
        const { path, node } = flatNodes[focusedIndex]
        if (node.children && expandedPaths.has(path)) {
          toggleExpanded(path)
        }
        break
      }

      case 'Enter': {
        e.preventDefault()
        const { path, node } = flatNodes[focusedIndex]
        if (node.children) {
          toggleExpanded(path)
        } else {
          onNavigate(path)
        }
        break
      }

      case 'Escape':
        e.preventDefault()
        setSearchQuery('')
        break
    }
  }, [flatNodes, focusedIndex, expandedPaths, toggleExpanded, onNavigate])

  // Auto-expand parent of active section
  useEffect(() => {
    if (!activeSection) return

    const parts = activeSection.split('.')
    const pathsToExpand = parts.reduce((acc: string[], index) => {
      const path = parts.slice(0, Number(index) + 1).join('.')
      acc.push(path)
      return acc
    }, [])

    setExpandedPaths((prev) => {
      const next = new Set(prev)
      pathsToExpand.forEach((path) => next.add(path))
      return next
    })
  }, [activeSection])

  // Render navigation tree
  const renderNodes = (nodes: NavigationNode[], depth = 0, parentPath = ''): React.ReactElement[] => {
    return nodes.map((node) => {
      const path = parentPath ? `${parentPath}.${node.id}` : node.id
      const isExpanded = expandedPaths.has(path)
      const isActive = activeSection === path
      const hasChildren = node.children && node.children.length > 0
      const issues = validationIssues.get(path) || []
      const errors = issues.filter((i) => i.level === 'error')
      const warnings = issues.filter((i) => i.level === 'warning')

      return (
        <div key={path}>
          <NavigationItem
            label={node.label}
            icon={node.icon}
            depth={depth}
            isActive={isActive}
            isExpanded={isExpanded}
            hasChildren={hasChildren}
            errorCount={errors.length}
            warningCount={warnings.length}
            onClick={() => {
              if (hasChildren) {
                toggleExpanded(path)
              } else {
                onNavigate(path)
              }
            }}
            onToggle={hasChildren ? () => toggleExpanded(path) : undefined}
          />
          
          {/* Render children if expanded */}
          {hasChildren && isExpanded && (
            <div role="group" aria-label={`${node.label} items`}>
              {renderNodes(node.children!, depth + 1, path)}
            </div>
          )}

          {/* Render add button for services/resources */}
          {isExpanded && node.type === 'section' && (node.id === 'services' || node.id === 'resources') && onAdd && (
            <button
              onClick={() => onAdd(node.id === 'services' ? 'service' : 'resource', path)}
              className={cn(
                'flex items-center gap-2 px-3 py-1.5 w-full text-sm text-muted-foreground hover:text-foreground hover:bg-accent transition-colors',
                'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring'
              )}
              style={{ paddingLeft: `${(depth + 1) * 12 + 12}px` }}
              aria-label={`Add ${node.id === 'services' ? 'service' : 'resource'}`}
            >
              <Plus className="w-3.5 h-3.5" />
              <span>Add {node.id === 'services' ? 'Service' : 'Resource'}</span>
            </button>
          )}
        </div>
      )
    })
  }

  if (isCollapsed) {
    return (
      <div className={cn('border-r border-border bg-background', className)}>
        <button
          onClick={onToggleCollapse}
          className="p-2 hover:bg-accent transition-colors"
          aria-label="Expand navigation"
        >
          <ChevronRight className="w-5 h-5" />
        </button>
      </div>
    )
  }

  return (
    <nav
      ref={navRef}
      className={cn('flex flex-col border-r border-border bg-background', className)}
      onKeyDown={handleKeyDown}
      tabIndex={0}
      role="navigation"
      aria-label="Azure YAML Editor Navigation"
    >
      {/* Header */}
      <div className="flex items-center justify-between p-3 border-b border-border">
        <h2 className="text-sm font-semibold">Configuration</h2>
        {onToggleCollapse && (
          <button
            onClick={onToggleCollapse}
            className="p-1 rounded hover:bg-accent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            aria-label="Collapse navigation"
          >
            <ChevronRight className="w-4 h-4" />
          </button>
        )}
      </div>

      {/* Search */}
      <NavigationSearch
        value={searchQuery}
        onChange={setSearchQuery}
        onClear={() => setSearchQuery('')}
      />

      {/* Navigation Tree */}
      <div className="flex-1 overflow-y-auto overflow-x-hidden py-2">
        {filtered.length > 0 ? (
          <div role="tree" aria-label="Configuration structure">
            {renderNodes(filtered)}
          </div>
        ) : (
          <div className="px-3 py-6 text-sm text-muted-foreground text-center">
            No results found for "{searchQuery}"
          </div>
        )}
      </div>

      {/* Footer hint */}
      <div className="px-3 py-2 border-t border-border">
        <p className="text-xs text-muted-foreground">
          Use arrow keys to navigate
        </p>
      </div>
    </nav>
  )
}
