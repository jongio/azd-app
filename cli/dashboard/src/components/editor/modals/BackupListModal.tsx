/**
 * Backup List Modal
 * Displays list of backups with preview, restore, view, and delete actions
 */

import * as React from 'react'
import { Clock, Eye, RotateCcw, Trash2, Search, FileText } from 'lucide-react'
import {
  Dialog,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogContent,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'
import type { BackupInfo } from '@/lib/editor/config-api'

export interface BackupListModalProps {
  /** Whether modal is open */
  isOpen: boolean
  /** Callback to close modal */
  onClose: () => void
  /** List of backups */
  backups: BackupInfo[]
  /** Callback when restore is clicked */
  onRestore: (timestamp: string) => void
  /** Callback when view is clicked */
  onView: (timestamp: string) => void
  /** Callback when delete is clicked */
  onDelete: (timestamp: string) => void
  /** Callback to get preview content */
  onGetPreview: (timestamp: string) => Promise<string>
  /** Whether an operation is in progress */
  isLoading?: boolean
}

interface BackupPreview {
  timestamp: string
  content: string
}

/**
 * Backup List Modal
 */
export function BackupListModal({
  isOpen,
  onClose,
  backups,
  onRestore,
  onView,
  onDelete,
  onGetPreview,
  isLoading = false,
}: BackupListModalProps) {
  const [searchQuery, setSearchQuery] = React.useState('')
  const [previews, setPreviews] = React.useState<Map<string, BackupPreview>>(new Map())
  const [loadingPreviews, setLoadingPreviews] = React.useState<Set<string>>(new Set())

  // Track which previews have been requested to avoid duplicates
  const requestedPreviews = React.useRef<Set<string>>(new Set())

  // Load preview for a backup
  const loadPreview = React.useCallback(async (timestamp: string) => {
    // Skip if already requested
    if (requestedPreviews.current.has(timestamp)) {
      return
    }
    
    requestedPreviews.current.add(timestamp)
    setLoadingPreviews(prev => new Set(prev).add(timestamp))
    
    try {
      const content = await onGetPreview(timestamp)
      const lines = content.split('\n').slice(0, 10).join('\n')
      setPreviews(prev => new Map(prev).set(timestamp, { timestamp, content: lines }))
    } catch {
      // Preview loading failure is non-critical - will just not show preview
    } finally {
      setLoadingPreviews(prev => {
        const next = new Set(prev)
        next.delete(timestamp)
        return next
      })
    }
  }, [onGetPreview])

  // Load previews for visible backups
  React.useEffect(() => {
    if (isOpen && backups.length > 0) {
      // Load first 5 previews
      backups.slice(0, 5).forEach(backup => {
        void loadPreview(backup.timestamp)
      })
    }
  }, [isOpen, backups, loadPreview])

  // Filter backups by search query
  const filteredBackups = React.useMemo(() => {
    if (!searchQuery) return backups

    const query = searchQuery.toLowerCase()
    return backups.filter(backup => {
      const formattedDate = formatTimestamp(backup.timestamp).toLowerCase()
      return formattedDate.includes(query) || backup.timestamp.includes(query)
    })
  }, [backups, searchQuery])

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

  // Format file size for display
  function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
  }

  // Get preview content
  function getPreviewContent(timestamp: string): string {
    const preview = previews.get(timestamp)
    return preview?.content || ''
  }

  return (
    <Dialog isOpen={isOpen} onClose={onClose} maxWidth="4xl">
      <DialogHeader onClose={onClose}>
        <DialogTitle>Backups</DialogTitle>
        <DialogDescription>
          Manage configuration backups
        </DialogDescription>
      </DialogHeader>

      <DialogContent className="p-0">
        {/* Search Bar */}
        <div className="px-6 pt-6 pb-4 border-b border-slate-200 dark:border-slate-700">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search backups by date..."
              className={cn(
                'w-full pl-10 pr-4 py-2 rounded-lg border',
                'bg-white dark:bg-slate-900',
                'border-slate-300 dark:border-slate-700',
                'text-slate-900 dark:text-slate-100',
                'placeholder:text-slate-400',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500',
                'transition-colors duration-150'
              )}
            />
          </div>
        </div>

        {/* Backup List */}
        <div className="max-h-[60vh] overflow-y-auto">
          {filteredBackups.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 px-6">
              <FileText className="w-12 h-12 text-slate-300 dark:text-slate-600 mb-4" />
              <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100 mb-2">
                {searchQuery ? 'No backups found' : 'No backups available'}
              </h3>
              <p className="text-sm text-slate-600 dark:text-slate-400 text-center max-w-sm">
                {searchQuery
                  ? 'Try adjusting your search query'
                  : 'Backups are created automatically when you save changes to azure.yaml'}
              </p>
            </div>
          ) : (
            <div className="divide-y divide-slate-200 dark:divide-slate-700">
              {filteredBackups.map((backup) => (
                <div
                  key={backup.timestamp}
                  className={cn(
                    'p-6 hover:bg-slate-50 dark:hover:bg-slate-800/50',
                    'transition-colors duration-150'
                  )}
                >
                  <div className="flex items-start gap-4">
                    {/* Icon */}
                    <div className="flex-shrink-0">
                      <div className="w-10 h-10 rounded-lg bg-cyan-100 dark:bg-cyan-900/30 flex items-center justify-center">
                        <Clock className="w-5 h-5 text-cyan-600 dark:text-cyan-400" />
                      </div>
                    </div>

                    {/* Content */}
                    <div className="flex-1 min-w-0">
                      {/* Header */}
                      <div className="flex items-start justify-between gap-4 mb-2">
                        <div>
                          <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100">
                            {formatTimestamp(backup.timestamp)}
                          </h3>
                          <p className="text-sm text-slate-600 dark:text-slate-400">
                            {formatSize(backup.size)}
                          </p>
                        </div>

                        {/* Actions */}
                        <div className="flex items-center gap-2">
                          <button
                            type="button"
                            onClick={() => onView(backup.timestamp)}
                            disabled={isLoading}
                            className={cn(
                              'p-2 rounded-lg text-slate-600 dark:text-slate-400',
                              'hover:bg-slate-200 dark:hover:bg-slate-700',
                              'focus:outline-none focus:ring-2 focus:ring-cyan-500',
                              'disabled:opacity-50 disabled:cursor-not-allowed',
                              'transition-colors duration-150'
                            )}
                            aria-label="View backup"
                            title="View full backup"
                          >
                            <Eye className="w-4 h-4" />
                          </button>
                          <button
                            type="button"
                            onClick={() => onRestore(backup.timestamp)}
                            disabled={isLoading}
                            className={cn(
                              'p-2 rounded-lg text-cyan-600 dark:text-cyan-400',
                              'hover:bg-cyan-100 dark:hover:bg-cyan-900/30',
                              'focus:outline-none focus:ring-2 focus:ring-cyan-500',
                              'disabled:opacity-50 disabled:cursor-not-allowed',
                              'transition-colors duration-150'
                            )}
                            aria-label="Restore backup"
                            title="Restore this backup"
                          >
                            <RotateCcw className="w-4 h-4" />
                          </button>
                          <button
                            type="button"
                            onClick={() => onDelete(backup.timestamp)}
                            disabled={isLoading}
                            className={cn(
                              'p-2 rounded-lg text-red-600 dark:text-red-400',
                              'hover:bg-red-100 dark:hover:bg-red-900/30',
                              'focus:outline-none focus:ring-2 focus:ring-red-500',
                              'disabled:opacity-50 disabled:cursor-not-allowed',
                              'transition-colors duration-150'
                            )}
                            aria-label="Delete backup"
                            title="Delete this backup"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </div>

                      {/* Preview */}
                      {loadingPreviews.has(backup.timestamp) ? (
                        <div className="mt-3 p-3 rounded-lg bg-slate-100 dark:bg-slate-800">
                          <p className="text-sm text-slate-400">Loading preview...</p>
                        </div>
                      ) : getPreviewContent(backup.timestamp) ? (
                        <div className="mt-3 p-3 rounded-lg bg-slate-100 dark:bg-slate-800 font-mono text-xs">
                          <pre className="text-slate-700 dark:text-slate-300 whitespace-pre-wrap break-all">
                            {getPreviewContent(backup.timestamp)}
                          </pre>
                          <p className="text-slate-400 dark:text-slate-500 mt-2 italic">
                            First 10 lines • Click "View" to see full content
                          </p>
                        </div>
                      ) : null}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
