/**
 * View Backup Modal
 * Displays full content of a backup file
 */

import * as React from 'react'
import { Copy, Download, Check } from 'lucide-react'
import {
  Dialog,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogContent,
  DialogFooter,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'

export interface ViewBackupModalProps {
  /** Whether modal is open */
  isOpen: boolean
  /** Callback to close modal */
  onClose: () => void
  /** Backup timestamp */
  timestamp: string
  /** Backup content */
  content: string
  /** Whether content is loading */
  isLoading?: boolean
  /** Testing flag to force copy failure */
  forceCopyError?: boolean
}

/**
 * View Backup Modal
 */
export function ViewBackupModal({
  isOpen,
  onClose,
  timestamp,
  content,
  isLoading = false,
  forceCopyError = false,
}: ViewBackupModalProps) {
  const [copied, setCopied] = React.useState(false)

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

  // Copy content to clipboard
  const handleCopy = React.useCallback(async () => {
    if (forceCopyError) {
      const error = new Error('Copy failed (forced)')
      console.error('Failed to copy:', error)
      alert('Failed to copy to clipboard')
      return
    }

    if (!navigator.clipboard || typeof navigator.clipboard.writeText !== 'function') {
      const error = new Error('Clipboard API unavailable')
      console.error('Failed to copy:', error)
      alert('Failed to copy to clipboard')
      return
    }

    try {
      await navigator.clipboard.writeText(content)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (error) {
      console.error('Failed to copy:', error)
      alert('Failed to copy to clipboard')
    }
  }, [content, forceCopyError])

  // Download content as file
  const handleDownload = React.useCallback(() => {
    try {
      const blob = new Blob([content], { type: 'text/yaml' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `azure.yaml.backup.${timestamp}`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    } catch (error) {
      console.error('Failed to download:', error)
      alert('Failed to download backup')
    }
  }, [content, timestamp])

  return (
    <Dialog isOpen={isOpen} onClose={onClose} maxWidth="4xl">
      <DialogHeader onClose={onClose}>
        <DialogTitle>View Backup</DialogTitle>
        <DialogDescription>
          Backup from {formatTimestamp(timestamp)}
        </DialogDescription>
      </DialogHeader>

      <DialogContent className="p-0">
        {isLoading ? (
          <div className="flex items-center justify-center py-16">
            <div className="text-center">
              <div className="w-12 h-12 border-4 border-slate-300 dark:border-slate-700 border-t-cyan-500 rounded-full animate-spin mx-auto mb-4" />
              <p className="text-sm text-slate-600 dark:text-slate-400">Loading backup...</p>
            </div>
          </div>
        ) : (
          <div className="max-h-[60vh] overflow-y-auto">
            <pre className={cn(
              'p-6 font-mono text-sm whitespace-pre',
              'text-slate-700 dark:text-slate-300',
              'bg-slate-50 dark:bg-slate-900'
            )}>
              {content}
            </pre>
          </div>
        )}
      </DialogContent>

      <DialogFooter>
        <div className="flex items-center justify-between w-full gap-4">
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => void handleCopy()}
              disabled={isLoading}
              className={cn(
                'flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-semibold',
                'text-slate-700 dark:text-slate-300',
                'border border-slate-200 dark:border-slate-700',
                'hover:bg-slate-100 dark:hover:bg-slate-800',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
                'disabled:opacity-50 disabled:cursor-not-allowed',
                'transition-colors duration-150'
              )}
            >
              {copied ? (
                <>
                  <Check className="w-4 h-4" />
                  Copied!
                </>
              ) : (
                <>
                  <Copy className="w-4 h-4" />
                  Copy
                </>
              )}
            </button>
            <button
              type="button"
              onClick={handleDownload}
              disabled={isLoading}
              className={cn(
                'flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-semibold',
                'text-slate-700 dark:text-slate-300',
                'border border-slate-200 dark:border-slate-700',
                'hover:bg-slate-100 dark:hover:bg-slate-800',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
                'disabled:opacity-50 disabled:cursor-not-allowed',
                'transition-colors duration-150'
              )}
            >
              <Download className="w-4 h-4" />
              Download
            </button>
          </div>
          <button
            type="button"
            onClick={onClose}
            className={cn(
              'px-6 py-2 rounded-lg text-sm font-semibold shadow-sm',
              'bg-cyan-600 text-white hover:bg-cyan-700',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
              'transition-colors duration-150'
            )}
          >
            Close
          </button>
        </div>
      </DialogFooter>
    </Dialog>
  )
}
