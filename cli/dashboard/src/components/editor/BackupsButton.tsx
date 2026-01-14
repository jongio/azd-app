/**
 * Backups Button
 * Button to open backup management modal
 */

import { Clock } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface BackupsButtonProps {
  /** Callback when button is clicked */
  onClick: () => void
  /** Whether button is disabled */
  disabled?: boolean
  /** Optional className */
  className?: string
}

/**
 * Backups Button Component
 */
export function BackupsButton({ onClick, disabled = false, className }: BackupsButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={cn(
        'flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-semibold',
        'text-slate-700 dark:text-slate-300',
        'border border-slate-200 dark:border-slate-700',
        'hover:bg-slate-100 dark:hover:bg-slate-800',
        'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
        'disabled:opacity-50 disabled:cursor-not-allowed',
        'transition-colors duration-150',
        className
      )}
      aria-label="Manage backups"
    >
      <Clock className="w-4 h-4" />
      Backups
    </button>
  )
}
