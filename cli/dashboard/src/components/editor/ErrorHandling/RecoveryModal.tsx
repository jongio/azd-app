/**
 * RecoveryModal - Draft recovery interface
 */

import { AlertTriangle, Clock } from 'lucide-react'
import { formatDraftAge, getDraftAge } from '../../../lib/errors'

export interface DraftData {
  config: any
  timestamp: number
  dirty: boolean
}

interface RecoveryModalProps {
  isOpen: boolean
  draft: DraftData | null
  currentConfig: any
  onRestore: () => void
  onDiscard: () => void
  onCancel: () => void
}

export function RecoveryModal({
  isOpen,
  draft,
  currentConfig,
  onRestore,
  onDiscard,
  onCancel,
}: RecoveryModalProps) {
  if (!isOpen || !draft) {
    return null
  }

  const age = getDraftAge(draft)
  const formattedAge = formatDraftAge(age)

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onCancel()
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      onCancel()
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      style={{ animation: 'fade-in 0.2s ease-out' }}
      onClick={handleBackdropClick}
      onKeyDown={handleKeyDown}
      role="dialog"
      aria-modal="true"
      aria-labelledby="recovery-modal-title"
      tabIndex={-1}
    >
      <div
        className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-2xl w-full mx-4"
        style={{ animation: 'fade-in-up 0.2s ease-out' }}
      >
        {/* Header */}
        <div className="flex items-start gap-3 p-6 border-b border-gray-200 dark:border-gray-700">
          <AlertTriangle className="w-6 h-6 text-yellow-600 dark:text-yellow-400 flex-shrink-0" />
          <div>
            <h2
              id="recovery-modal-title"
              className="text-lg font-semibold text-gray-900 dark:text-gray-100"
            >
              Unsaved Changes Detected
            </h2>
            <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
              A draft was saved {formattedAge}. Would you like to restore it?
            </p>
          </div>
        </div>

        {/* Body */}
        <div className="p-6 space-y-4">
          <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
            <Clock className="w-4 h-4" />
            <span>Draft saved: {new Date(draft.timestamp).toLocaleString()}</span>
          </div>

          <div className="grid grid-cols-2 gap-4">
            {/* Current Config Preview */}
            <div>
              <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
                Current Configuration
              </h3>
              <div className="bg-gray-100 dark:bg-gray-900 p-3 rounded text-xs overflow-auto max-h-48">
                <pre className="text-gray-800 dark:text-gray-200">
                  {JSON.stringify(currentConfig, null, 2).slice(0, 500)}
                  {JSON.stringify(currentConfig).length > 500 && '\n...'}
                </pre>
              </div>
            </div>

            {/* Draft Preview */}
            <div>
              <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
                Draft Configuration
              </h3>
              <div className="bg-gray-100 dark:bg-gray-900 p-3 rounded text-xs overflow-auto max-h-48">
                <pre className="text-gray-800 dark:text-gray-200">
                  {JSON.stringify(draft.config, null, 2).slice(0, 500)}
                  {JSON.stringify(draft.config).length > 500 && '\n...'}
                </pre>
              </div>
            </div>
          </div>

          <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 p-4 rounded">
            <p className="text-sm text-blue-800 dark:text-blue-200">
              <strong>What will happen:</strong>
            </p>
            <ul className="mt-2 text-sm text-blue-700 dark:text-blue-300 space-y-1 list-disc list-inside">
              <li>
                <strong>Restore Draft:</strong> Load the draft configuration and continue editing
              </li>
              <li>
                <strong>Discard Draft:</strong> Delete the draft and continue with current
                configuration
              </li>
              <li>
                <strong>Cancel:</strong> Close this dialog without making changes
              </li>
            </ul>
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-2 p-6 border-t border-gray-200 dark:border-gray-700">
          <button
            onClick={onCancel}
            className="px-4 py-2 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100 hover:bg-gray-300 dark:hover:bg-gray-600"
          >
            Cancel
          </button>
          <button
            onClick={onDiscard}
            className="px-4 py-2 rounded-md text-sm font-medium bg-red-600 text-white hover:bg-red-700"
          >
            Discard Draft
          </button>
          <button
            onClick={onRestore}
            className="px-4 py-2 rounded-md text-sm font-medium bg-blue-600 text-white hover:bg-blue-700"
          >
            Restore Draft
          </button>
        </div>
      </div>
    </div>
  )
}
