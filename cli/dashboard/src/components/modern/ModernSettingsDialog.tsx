/**
 * ModernSettingsDialog - Settings dialog with modern styling
 * Includes log classifications and other settings
 * Follows design patterns from ModernServiceDetailPanel
 */
import * as React from 'react'
import { X, AlertCircle, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useEscapeKey } from '@/hooks/useEscapeKey'
import { useLogClassifications } from '@/hooks/useLogClassifications'
import { ClassificationsEditor, type ClassificationChange } from '@/components/ClassificationsManager'

// =============================================================================
// Types
// =============================================================================

export interface ModernSettingsDialogProps {
  /** Whether dialog is open */
  isOpen: boolean
  /** Close handler */
  onClose: () => void
  /** Additional class names */
  className?: string
}

// =============================================================================
// Section Card Component
// =============================================================================

interface SectionCardProps {
  title: string
  description?: string
  children: React.ReactNode
}

function SectionCard({ title, description, children }: SectionCardProps) {
  return (
    <div className="mb-4 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 overflow-hidden">
      <div className="px-4 py-3 bg-slate-100 dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
          {title}
        </h4>
        {description && (
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
            {description}
          </p>
        )}
      </div>
      <div className="p-4">{children}</div>
    </div>
  )
}

// =============================================================================
// ModernSettingsDialog Component
// =============================================================================

export function ModernSettingsDialog({
  isOpen,
  onClose,
  className,
}: ModernSettingsDialogProps) {
  const dialogRef = React.useRef<HTMLDivElement>(null)
  const { classifications, addClassification, deleteClassification } = useLogClassifications()
  const [pendingChanges, setPendingChanges] = React.useState<ClassificationChange[]>([])
  const [saveStatus, setSaveStatus] = React.useState<'idle' | 'saving' | 'saved' | 'error'>('idle')
  const statusTimeoutRef = React.useRef<ReturnType<typeof setTimeout> | null>(null)

  const hasPendingChanges = pendingChanges.length > 0

  useEscapeKey(onClose, isOpen)

  // Cleanup timers on unmount
  React.useEffect(() => {
    return () => {
      if (statusTimeoutRef.current) {
        clearTimeout(statusTimeoutRef.current)
      }
    }
  }, [])

  // Reset pending changes when dialog opens
  React.useEffect(() => {
    if (isOpen) {
      setPendingChanges([])
      setSaveStatus('idle')
    }
  }, [isOpen])

  // Focus management
  React.useEffect(() => {
    if (isOpen && dialogRef.current) {
      const closeButton = dialogRef.current.querySelector<HTMLButtonElement>('[data-close-button]')
      closeButton?.focus()
    }
  }, [isOpen])

  // Handle adding a pending change
  const handleAddChange = (change: ClassificationChange) => {
    setPendingChanges(prev => [...prev, change])
  }

  // Handle removing a pending change
  const handleRemoveChange = (index: number) => {
    setPendingChanges(prev => prev.filter((_, i) => i !== index))
  }

  // Save all pending changes
  const handleSaveChanges = async () => {
    if (pendingChanges.length === 0) return

    setSaveStatus('saving')

    try {
      // Process deletes first (in reverse order to maintain indices)
      const deletes = pendingChanges.filter(c => c.type === 'delete')
      const sortedDeletes = [...deletes].sort((a, b) => (b.index ?? 0) - (a.index ?? 0))
      
      for (const change of sortedDeletes) {
        if (change.index !== undefined) {
          await deleteClassification(change.index)
        }
      }

      // Process adds
      const adds = pendingChanges.filter(c => c.type === 'add')
      for (const change of adds) {
        await addClassification(change.classification.text, change.classification.level)
      }

      setPendingChanges([])
      setSaveStatus('saved')

      // Reset status after a moment
      if (statusTimeoutRef.current) {
        clearTimeout(statusTimeoutRef.current)
      }
      statusTimeoutRef.current = setTimeout(() => setSaveStatus('idle'), 2000)
    } catch (err) {
      console.error('Failed to save changes:', err)
      setSaveStatus('error')
      
      if (statusTimeoutRef.current) {
        clearTimeout(statusTimeoutRef.current)
      }
      statusTimeoutRef.current = setTimeout(() => setSaveStatus('idle'), 3000)
    }
  }

  // Discard all pending changes
  const handleDiscardChanges = () => {
    setPendingChanges([])
    setSaveStatus('idle')
  }

  if (!isOpen) {
    return null
  }

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-40 bg-black/40 backdrop-blur-sm animate-fade-in"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Dialog */}
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="settings-title"
        className={cn(
          'fixed left-1/2 top-1/2 z-50 -translate-x-1/2 -translate-y-1/2',
          'w-full max-w-lg',
          'bg-white dark:bg-slate-900',
          'border border-slate-200 dark:border-slate-700',
          'rounded-2xl shadow-2xl',
          'flex flex-col',
          'max-h-[90vh]',
          'animate-scale-in',
          className
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-slate-700 shrink-0">
          <h2
            id="settings-title"
            className="text-lg font-semibold text-slate-900 dark:text-slate-100"
          >
            Settings
          </h2>
          <button
            type="button"
            data-close-button
            onClick={onClose}
            className="p-2 -mr-2 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
            aria-label="Close settings"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {/* Log Classifications */}
          <SectionCard
            title="Log Classifications"
            description="Customize how log messages are classified"
          >
            <ClassificationsEditor
              classifications={classifications}
              pendingChanges={pendingChanges}
              onAddChange={handleAddChange}
              onRemoveChange={handleRemoveChange}
            />
          </SectionCard>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t border-slate-200 dark:border-slate-700 shrink-0">
          {/* Status indicator */}
          <div className="flex items-center gap-2 text-sm">
            {saveStatus === 'saved' && (
              <span className="flex items-center gap-1 text-emerald-600 dark:text-emerald-400">
                <Check className="w-4 h-4" />
                Saved
              </span>
            )}
            {saveStatus === 'error' && (
              <span className="flex items-center gap-1 text-red-600 dark:text-red-400">
                <AlertCircle className="w-4 h-4" />
                Error saving
              </span>
            )}
            {hasPendingChanges && saveStatus === 'idle' && (
              <span className="text-amber-600 dark:text-amber-400">
                {pendingChanges.length} unsaved {pendingChanges.length === 1 ? 'change' : 'changes'}
              </span>
            )}
          </div>

          {/* Action buttons */}
          <div className="flex items-center gap-2">
            {hasPendingChanges && (
              <>
                <button
                  type="button"
                  onClick={handleDiscardChanges}
                  className="px-4 py-2 text-sm font-medium text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
                >
                  Discard
                </button>
                <button
                  type="button"
                  onClick={() => void handleSaveChanges()}
                  disabled={saveStatus === 'saving'}
                  className={cn(
                    "px-4 py-2 text-sm font-medium rounded-lg transition-colors",
                    "bg-cyan-500 hover:bg-cyan-600 text-white",
                    saveStatus === 'saving' && "opacity-50 cursor-not-allowed"
                  )}
                >
                  {saveStatus === 'saving' ? 'Saving...' : 'Save Changes'}
                </button>
              </>
            )}
            {!hasPendingChanges && (
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
              >
                Close
              </button>
            )}
          </div>
        </div>
      </div>
    </>
  )
}
