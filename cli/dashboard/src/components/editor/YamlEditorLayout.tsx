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
  return (
    <div className={cn('flex flex-col h-screen overflow-hidden bg-background', className)}>
      {/* Header */}
      {header}

      {/* Main Content Area */}
      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar */}
        <aside
          className={cn(
            'transition-all duration-300 ease-in-out overflow-hidden',
            isSidebarCollapsed ? 'w-12' : 'w-64'
          )}
        >
          {sidebar}
        </aside>

        {/* Content Area (Form) */}
        <main
          className={cn(
            'flex-1 overflow-y-auto overflow-x-hidden',
            'bg-muted/30'
          )}
        >
          <div className="container mx-auto py-6 px-6 max-w-5xl">
            {content}
          </div>
        </main>

        {/* Preview Pane */}
        {isPreviewVisible && preview}
      </div>

      {/* Footer (Quick Actions) */}
      {footer}
    </div>
  )
}
