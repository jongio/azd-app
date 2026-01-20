/**
 * Restore Backup Confirmation Dialog
 * Confirms backup restoration with warning
 */

import { RotateCcw } from 'lucide-react'
import {
  Dialog,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogContent,
  DialogFooter,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'

export interface RestoreConfirmationDialogProps {
  /** Whether dialog is open */
  isOpen: boolean
  /** Callback to close dialog */
  onClose: () => void
  /** Callback when restore is confirmed */
  onConfirm: () => void | Promise<void>
  /** Backup timestamp */
  timestamp: string
  /** Whether restore is in progress */
  isRestoring?: boolean
}

/**
 * Restore Backup Confirmation Dialog
 */
export function RestoreConfirmationDialog({
  isOpen,
  onClose,
  onConfirm,
  timestamp,
  isRestoring = false,
}: RestoreConfirmationDialogProps) {
  // Format timestamp for display
  function formatTimestamp(timestamp: string): string {
    try {
      const date = new Date(timestamp)
      return date.toLocaleString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: 'numeric',
        minute: '2-digit',
        hour12: true,
      })
    } catch {
      return timestamp
    }
  }

  const handleConfirm = async () => {
    try {
      await onConfirm()
      onClose()
    } catch (error) {
      alert('Failed to restore backup. Please try again.')
    }
  }

  return (
    <Dialog isOpen={isOpen} onClose={onClose} maxWidth="md">
      <DialogHeader onClose={onClose}>
        <DialogTitle>Restore Backup</DialogTitle>
        <DialogDescription>
          Confirm backup restoration
        </DialogDescription>
      </DialogHeader>

      <DialogContent>
        <div className="flex items-start gap-4">
          <div className="flex-shrink-0">
            <div className="w-12 h-12 rounded-full bg-cyan-100 dark:bg-cyan-900/30 flex items-center justify-center">
              <RotateCcw className="w-6 h-6 text-cyan-600 dark:text-cyan-400" />
            </div>
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100 mb-2">
              Restore backup from {formatTimestamp(timestamp)}?
            </h3>
            <p className="text-sm text-slate-600 dark:text-slate-400 mb-3">
              This will replace your current azure.yaml configuration with the backup from {formatTimestamp(timestamp)}.
            </p>
            <div className="rounded-lg bg-cyan-50 dark:bg-cyan-950/30 border border-cyan-200 dark:border-cyan-800 p-3">
              <p className="text-sm text-cyan-800 dark:text-cyan-300">
                <strong>Note:</strong> Your current configuration will be backed up first before the restore operation.
              </p>
            </div>
          </div>
        </div>
      </DialogContent>

      <DialogFooter>
        <div className="flex items-center justify-end gap-3 w-full">
          <button
            type="button"
            onClick={onClose}
            disabled={isRestoring}
            className={cn(
              'px-4 py-2 rounded-lg text-sm font-semibold',
              'text-slate-700 dark:text-slate-300',
              'border border-slate-200 dark:border-slate-700',
              'hover:bg-slate-100 dark:hover:bg-slate-800',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
              'disabled:opacity-50 disabled:cursor-not-allowed',
              'transition-colors duration-150'
            )}
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => void handleConfirm()}
            disabled={isRestoring}
            className={cn(
              'px-6 py-2 rounded-lg text-sm font-semibold shadow-sm',
              'bg-cyan-600 text-white hover:bg-cyan-700',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
              'disabled:opacity-50 disabled:cursor-not-allowed',
              'transition-colors duration-150'
            )}
          >
            {isRestoring ? 'Restoring...' : 'Restore Backup'}
          </button>
        </div>
      </DialogFooter>
    </Dialog>
  )
}
