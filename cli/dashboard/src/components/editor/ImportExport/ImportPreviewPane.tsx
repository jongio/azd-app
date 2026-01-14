/**
 * Import Preview Pane Component
 * Shows diff preview before confirming import
 */

import * as React from 'react'
import {
  Dialog,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogContent,
  DialogFooter,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'
import type { ImportPreview } from '@/lib/editor/import-export-types'
import { Check, X, Edit, Eye } from 'lucide-react'

export interface ImportPreviewPaneProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void
  preview: ImportPreview
  isSubmitting?: boolean
}

/**
 * Import Preview Pane Component
 */
export function ImportPreviewPane({
  isOpen,
  onClose,
  onConfirm,
  preview,
  isSubmitting = false,
}: ImportPreviewPaneProps) {
  const [viewMode, setViewMode] = React.useState<'diff' | 'merged'>('diff')

  // Count changes by type
  const stats = React.useMemo(() => {
    const added = preview.diff.filter(d => d.type === 'added').length
    const removed = preview.diff.filter(d => d.type === 'removed').length
    const changed = preview.diff.filter(d => d.type === 'changed').length
    const unchanged = preview.diff.filter(d => d.type === 'unchanged').length

    return { added, removed, changed, unchanged }
  }, [preview.diff])

  return (
    <Dialog isOpen={isOpen} onClose={onClose} maxWidth="4xl">
      <DialogHeader onClose={onClose}>
        <DialogTitle>Import Preview</DialogTitle>
        <DialogDescription>
          Review changes before importing
        </DialogDescription>
      </DialogHeader>

      <DialogContent className="p-0">
        <div className="space-y-4">
          {/* View Mode Selector */}
          <div className="px-6 pt-4 border-b border-slate-200 dark:border-slate-700">
            <div className="flex items-center gap-2 mb-4">
              <button
                type="button"
                onClick={() => setViewMode('diff')}
                className={cn(
                  'px-4 py-2 rounded-lg text-sm font-semibold transition-colors',
                  viewMode === 'diff'
                    ? 'bg-cyan-600 text-white'
                    : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800'
                )}
              >
                <Eye className="w-4 h-4 inline-block mr-2" />
                Diff View
              </button>
              <button
                type="button"
                onClick={() => setViewMode('merged')}
                className={cn(
                  'px-4 py-2 rounded-lg text-sm font-semibold transition-colors',
                  viewMode === 'merged'
                    ? 'bg-cyan-600 text-white'
                    : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800'
                )}
              >
                <Edit className="w-4 h-4 inline-block mr-2" />
                Merged Result
              </button>
            </div>

            {/* Stats */}
            <div className="flex items-center gap-4 pb-4">
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-green-500" />
                <span className="text-sm text-slate-600 dark:text-slate-400">
                  {stats.added} added
                </span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-red-500" />
                <span className="text-sm text-slate-600 dark:text-slate-400">
                  {stats.removed} removed
                </span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-amber-500" />
                <span className="text-sm text-slate-600 dark:text-slate-400">
                  {stats.changed} changed
                </span>
              </div>
            </div>
          </div>

          {/* Content */}
          <div className="px-6 pb-4">
            {viewMode === 'diff' ? (
              <div className="space-y-3">
                {preview.diff.map((item, index) => {
                  if (item.type === 'unchanged') return null

                  const Icon = item.type === 'added' ? Check : item.type === 'removed' ? X : Edit
                  const color = item.type === 'added' ? 'green' : item.type === 'removed' ? 'red' : 'amber'

                  return (
                    <div
                      key={index}
                      className={cn(
                        'p-3 rounded-lg border',
                        `border-${color}-200 dark:border-${color}-800`,
                        `bg-${color}-50 dark:bg-${color}-900/20`
                      )}
                    >
                      <div className="flex items-start gap-3">
                        <Icon className={cn('w-4 h-4 flex-shrink-0 mt-0.5', `text-${color}-600 dark:text-${color}-400`)} />
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-medium text-slate-900 dark:text-slate-100 mb-1">
                            {item.path}
                          </div>
                          <div className="font-mono text-xs space-y-1">
                            {item.type === 'changed' && (
                              <>
                                <div className="text-red-700 dark:text-red-300">
                                  - {JSON.stringify(item.currentValue)}
                                </div>
                                <div className="text-green-700 dark:text-green-300">
                                  + {JSON.stringify(item.importedValue)}
                                </div>
                              </>
                            )}
                            {item.type === 'added' && (
                              <div className="text-green-700 dark:text-green-300">
                                + {JSON.stringify(item.importedValue)}
                              </div>
                            )}
                            {item.type === 'removed' && (
                              <div className="text-red-700 dark:text-red-300">
                                - {JSON.stringify(item.currentValue)}
                              </div>
                            )}
                          </div>
                        </div>
                      </div>
                    </div>
                  )
                })}
              </div>
            ) : (
              <pre className="p-4 rounded-lg bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 font-mono text-xs overflow-x-auto max-h-96 text-slate-900 dark:text-slate-100">
                {preview.merged}
              </pre>
            )}
          </div>
        </div>
      </DialogContent>

      <DialogFooter>
        <div className="flex items-center justify-end w-full gap-3">
          <button
            type="button"
            onClick={onClose}
            disabled={isSubmitting}
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
            onClick={onConfirm}
            disabled={isSubmitting}
            className={cn(
              'px-6 py-2 rounded-lg text-sm font-semibold shadow-sm',
              'bg-cyan-600 text-white hover:bg-cyan-700',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
              'disabled:opacity-50 disabled:cursor-not-allowed',
              'transition-colors duration-150'
            )}
          >
            {isSubmitting ? 'Importing...' : 'Confirm Import'}
          </button>
        </div>
      </DialogFooter>
    </Dialog>
  )
}
