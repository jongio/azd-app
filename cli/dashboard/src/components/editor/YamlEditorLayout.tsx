/**
 * YAML Editor Layout Component
 * 
 * Main layout structure with sidebar, content area, and preview pane.
 * Handles resizing and responsive behavior.
 */

import * as React from 'react'
import { cn } from '@/lib/utils'

export interface YamlEditorLayoutProps {
  /** Header content */
  header: React.ReactNode
  
  /** Sidebar (navigation) */
  sidebar: React.ReactNode
  
  /** Main content area (form) */
  content: React.ReactNode
  
  /** Preview pane (optional, controlled by visibility) */
  preview?: React.ReactNode
  
  /** Footer/Quick actions bar */
  footer?: React.ReactNode
  
  /** Whether sidebar is collapsed */
  isSidebarCollapsed?: boolean
  
  /** Whether preview is visible */
  isPreviewVisible?: boolean
  
  /** Custom className */
  className?: string
}

/**
 * YAML Editor Layout Component
 * 
 * Three-column layout: Sidebar | Content | Preview
 * Improved UX with better spacing, resizable preview, and modern design
 */
export function YamlEditorLayout({
  header,
  sidebar,
  content,
  preview,
  footer,
  isSidebarCollapsed = false,
  isPreviewVisible = true,
  className,
}: YamlEditorLayoutProps) {
  const [previewWidth, setPreviewWidth] = React.useState(() => {
    if (typeof window === 'undefined') return 40
    try {
      const stored = localStorage.getItem('azd-editor-preview-width')
      return stored ? Math.max(20, Math.min(60, Number.parseInt(stored, 10))) : 40
    } catch {
      return 40
    }
  })

  const [isResizing, setIsResizing] = React.useState(false)

  // Handle preview pane resize
  const handleMouseDown = React.useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    setIsResizing(true)
  }, [])

  React.useEffect(() => {
    if (!isResizing) return

    const handleMouseMove = (e: MouseEvent) => {
      const containerWidth = window.innerWidth
      const sidebarWidth = isSidebarCollapsed ? 48 : 256
      const availableWidth = containerWidth - sidebarWidth
      const newWidth = ((containerWidth - e.clientX) / availableWidth) * 100
      const clampedWidth = Math.max(20, Math.min(60, newWidth))
      setPreviewWidth(clampedWidth)
      try {
        localStorage.setItem('azd-editor-preview-width', String(clampedWidth))
      } catch {
        // Ignore localStorage errors
      }
    }

    const handleMouseUp = () => {
      setIsResizing(false)
    }

    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)

    return () => {
      window.removeEventListener('mousemove', handleMouseMove)
      window.removeEventListener('mouseup', handleMouseUp)
    }
  }, [isResizing, isSidebarCollapsed])

  return (
    <div className={cn(
      'flex flex-col h-screen overflow-hidden',
      'bg-slate-50 dark:bg-slate-900',
      className
    )}>
      {/* Header */}
      <div className="flex-shrink-0 border-b border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
        {header}
      </div>

      {/* Main Content Area */}
      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar */}
        <aside
          className={cn(
            'flex-shrink-0 transition-all duration-300 ease-in-out',
            'border-r border-slate-200 dark:border-slate-700',
            'bg-white dark:bg-slate-800',
            isSidebarCollapsed ? 'w-12' : 'w-64'
          )}
        >
          {sidebar}
        </aside>

        {/* Content Area (Form) */}
        <main
          className={cn(
            'flex-1 overflow-y-auto overflow-x-hidden min-w-0',
            'bg-slate-50 dark:bg-slate-900',
            isPreviewVisible && 'border-r border-slate-200 dark:border-slate-700'
          )}
          style={isPreviewVisible ? { flex: `1 1 ${100 - previewWidth}%` } : undefined}
        >
          <div className="container mx-auto py-8 px-8 max-w-6xl">
            {content}
          </div>
        </main>

        {/* Preview Pane with Resizer */}
        {isPreviewVisible && (
          <>
            {/* Resizer */}
            <div
              onMouseDown={handleMouseDown}
              className={cn(
                'w-1 flex-shrink-0 cursor-col-resize bg-slate-200 dark:bg-slate-700',
                'hover:bg-cyan-500 dark:hover:bg-cyan-600',
                'transition-colors duration-150',
                isResizing && 'bg-cyan-500 dark:bg-cyan-600'
              )}
              style={{ width: '4px' }}
            >
              <div className="w-full h-full flex items-center justify-center">
                <div className={cn(
                  'w-0.5 h-12 bg-slate-400 dark:bg-slate-500 rounded-full',
                  'opacity-0 group-hover:opacity-100',
                  isResizing && 'opacity-100'
                )} />
              </div>
            </div>

            {/* Preview Pane - Takes remaining space */}
            <aside
              className="flex flex-col min-w-0 bg-white dark:bg-slate-800 border-l border-slate-200 dark:border-slate-700 h-full"
              style={{
                flex: `0 0 ${previewWidth}%`,
                minWidth: '300px',
                maxWidth: `${previewWidth}%`,
              }}
            >
              {preview}
            </aside>
          </>
        )}
      </div>

      {/* Footer (Quick Actions) */}
      {footer && (
        <div className="flex-shrink-0 border-t border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
          {footer}
        </div>
      )}
    </div>
  )
}
