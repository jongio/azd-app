/**
 * CollapsibleSection - Reusable collapsible/expandable section component
 * Used for help content, FAQs, and accordion-style UI
 */
import * as React from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface CollapsibleSectionProps {
  /** Unique identifier for accessibility */
  id: string
  /** Section title */
  title: string
  /** Whether section is expanded */
  isOpen: boolean
  /** Callback when toggle is clicked */
  onToggle: () => void
  /** Section content */
  children: React.ReactNode
  /** Additional class names */
  className?: string
}

/**
 * CollapsibleSection component for expandable content areas.
 * Features:
 * - Accessible ARIA attributes
 * - Smooth expand/collapse animation
 * - Keyboard navigation support
 * - Visual expand/collapse indicator
 */
export function CollapsibleSection({
  id,
  title,
  isOpen,
  onToggle,
  children,
  className,
}: Readonly<CollapsibleSectionProps>) {
  return (
    <div className={cn('border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden', className)}>
      <button
        type="button"
        onClick={onToggle}
        className={cn(
          'flex items-center justify-between w-full px-4 py-3',
          'bg-slate-50 dark:bg-slate-800/50',
          'hover:bg-slate-100 dark:hover:bg-slate-800',
          'text-left font-medium text-sm',
          'text-slate-900 dark:text-slate-100',
          'focus:outline-none focus:ring-2 focus:ring-inset focus:ring-cyan-500',
          'transition-colors duration-150',
        )}
        aria-expanded={isOpen}
        aria-controls={`collapsible-section-${id}`}
      >
        <span>{title}</span>
        {isOpen ? (
          <ChevronDown className="w-4 h-4 text-slate-500 transition-transform" aria-hidden="true" />
        ) : (
          <ChevronRight className="w-4 h-4 text-slate-500 transition-transform" aria-hidden="true" />
        )}
      </button>
      {isOpen && (
        <div
          id={`collapsible-section-${id}`}
          className="px-4 py-3 bg-white dark:bg-slate-900 animate-fade-in"
        >
          {children}
        </div>
      )}
    </div>
  )
}

export default CollapsibleSection
