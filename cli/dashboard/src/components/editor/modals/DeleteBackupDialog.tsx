/**
 * Delete Backup Confirmation Dialog
 * Confirms backup deletion with warning
 */

import { Trash2 } from 'lucide-react'
import {
  Dialog,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogContent,
  DialogFooter,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'

export interface DeleteBackupDialogProps {
  /** Whether dialog is open */
  isOpen: boolean
  /** Callback to close dialog */
  onClose: () => void
  /** Callback when delete is confirmed */
  onConfirm: () => void | Promise<void>
  /** Backup timestamp */
  timestamp: string
  /** Whether delete is in progress */
  isDeleting?: boolean
}

/**
 * Delete Backup Confirmation Dialog
 */
export function DeleteBackupDialog({
  isOpen,
  onClose,
  onConfirm,
  timestamp,
  isDeleting = false,
}: DeleteBackupDialogProps) {
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
      console.error('Failed to delete backup:', error)
      alert('Failed to delete backup. Please try again.')
    }
  }

  return (
    <Dialog isOpen={isOpen} onClose={onClose} maxWidth="md">
      <DialogHeader onClose={onClose}>
        <DialogTitle>Delete Backup</DialogTitle>
        <DialogDescription>
          This action cannot be undone
        </DialogDescription>
      </DialogHeader>

      <DialogContent>
        <div className="flex items-start gap-4">
          <div className="flex-shrink-0">
            <div className="w-12 h-12 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center">
              <Trash2 className="w-6 h-6 text-red-600 dark:text-red-400" />
            </div>
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100 mb-2">
              Delete backup from {formatTimestamp(timestamp)}?
            </h3>
            <p className="text-sm text-slate-600 dark:text-slate-400 mb-3">
              Are you sure you want to delete this backup? You won't be able to restore this configuration after deletion.
            </p>
            <div className="rounded-lg bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 p-3">
              <p className="text-sm text-red-800 dark:text-red-300">
                <strong>Warning:</strong> This action cannot be undone. The backup file will be permanently removed.
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
            disabled={isDeleting}
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
            disabled={isDeleting}
            className={cn(
              'px-6 py-2 rounded-lg text-sm font-semibold shadow-sm',
              'bg-red-600 text-white hover:bg-red-700',
              'focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2',
              'disabled:opacity-50 disabled:cursor-not-allowed',
              'transition-colors duration-150'
            )}
          >
            {isDeleting ? 'Deleting...' : 'Confirm Delete'}
          </button>
        </div>
      </DialogFooter>
    </Dialog>
  )
}
