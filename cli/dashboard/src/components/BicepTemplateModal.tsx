/**
 * BicepTemplateModal - Modal dialog for displaying and copying Bicep diagnostic settings template
 * Shows unified template for all services with integration instructions
 */
import * as React from 'react'
import { X, ChevronRight, Download, AlertTriangle, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useEscapeKey } from '@/hooks/useEscapeKey'
import { useBicepTemplate } from '@/hooks/useBicepTemplate'
import { useToast } from '@/hooks/useToast'
import { useTimeout, useTimeoutMap } from '@/hooks/useTimeout'
import { CodeBlock } from '@/components/shared/CodeBlock'

// =============================================================================
// Types
// =============================================================================

/**
 * Props for BicepTemplateModal component.
 * 
 * @property isOpen - Controls visibility of the modal
 * @property onClose - Callback when user closes the modal
 * @property services - List of service names to include in template (optional, for display purposes)
 */
export interface BicepTemplateModalProps {
  isOpen: boolean
  onClose: () => void
  services?: string[]
}

// =============================================================================
// Helper Components
// =============================================================================

/**
 * Collapsible integration instructions section
 */
interface InstructionsSectionProps {
  summary: string
  steps: string[]
}

function InstructionsSection({ summary, steps }: Readonly<InstructionsSectionProps>) {
  const [isExpanded, setIsExpanded] = React.useState(false)

  return (
    <details
      open={isExpanded}
      onToggle={(e) => setIsExpanded((e.target as HTMLDetailsElement).open)}
      className="border-b border-slate-200 dark:border-slate-700"
    >
      <summary className="cursor-pointer p-4 hover:bg-slate-50 dark:hover:bg-slate-900/60 transition-colors flex items-center gap-2 select-none">
        <ChevronRight
          className={cn(
            'w-4 h-4 text-slate-500 transition-transform',
            isExpanded && 'rotate-90'
          )}
        />
        <span className="text-sm font-semibold text-slate-900 dark:text-slate-100">
          Integration Instructions
        </span>
      </summary>
      <div className="px-6 pb-4 space-y-3">
        <p className="text-sm text-slate-600 dark:text-slate-400">
          {summary}
        </p>
        <ol className="text-sm text-slate-700 dark:text-slate-300 space-y-2 ml-4 list-decimal">
          {steps.map((step) => (
            <li key={step} dangerouslySetInnerHTML={{ __html: step }} />
          ))}
        </ol>
      </div>
    </details>
  )
}

// =============================================================================
// Main Component
// =============================================================================

export function BicepTemplateModal({
  isOpen,
  onClose,
  services: _services,
}: Readonly<BicepTemplateModalProps>) {
  const dialogRef = React.useRef<HTMLDialogElement>(null)
  const {
    isLoading,
    error,
    template,
    services,
    instructions,
    fetchTemplate,
  } = useBicepTemplate()
  const { showToast } = useToast()
  const [copied, setCopied] = React.useState(false)
  const { setTimeout: setTimeoutSafe } = useTimeout()

  useEscapeKey(onClose, isOpen)

  // Focus management
  React.useEffect(() => {
    if (isOpen && dialogRef.current) {
      const closeButton = dialogRef.current.querySelector<HTMLButtonElement>('[data-close-button]')
      closeButton?.focus()
    }
  }, [isOpen])

  // Handle copy all
  const handleCopyAll = async () => {
    if (!template) return

    try {
      await navigator.clipboard.writeText(template)
      setCopied(true)
      showToast('Template copied to clipboard', 'success')
      setTimeoutSafe(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy template:', err)
      showToast('Failed to copy template', 'error')
    }
  }

  // Handle download
  const handleDownload = () => {
    if (!template) return

    try {
      const blob = new Blob([template], { type: 'text/plain' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'diagnostic-settings.bicep'
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
      showToast('Template downloaded', 'success')
    } catch (err) {
      console.error('Failed to download template:', err)
      showToast('Failed to download template', 'error')
    }
  }

  // Handle backdrop click
  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose()
    }
  }

  if (!isOpen) {
    return null
  }

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-50 bg-black/50 dark:bg-black/70 animate-fade-in"
        onClick={handleBackdropClick}
        aria-hidden="true"
      />

      {/* Dialog */}
      <dialog
        ref={dialogRef}
        open
        aria-labelledby="bicep-template-title"
        aria-modal="true"
        className={cn(
          'fixed left-1/2 top-1/2 z-50 -translate-x-1/2 -translate-y-1/2',
          'w-full max-w-4xl',
          'bg-white dark:bg-slate-900',
          'border border-slate-200 dark:border-slate-700',
          'rounded-2xl shadow-2xl',
          'flex flex-col',
          'max-h-[85vh]',
          'animate-scale-in',
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-slate-700 shrink-0">
          <div>
            <h2
              id="bicep-template-title"
              className="text-lg font-semibold text-slate-900 dark:text-slate-100"
            >
              Diagnostic Settings Template
            </h2>
            {services.length > 0 && (
              <p className="text-sm text-slate-600 dark:text-slate-400 mt-0.5">
                Bicep template for {services.length} {services.length === 1 ? 'service' : 'services'}
              </p>
            )}
          </div>
          <button
            type="button"
            data-close-button
            onClick={onClose}
            className="p-2 -mr-2 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
            aria-label="Close template"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto">
          {/* Loading State */}
          {isLoading && (
            <div className="p-8 flex flex-col items-center justify-center gap-3">
              <Loader2 className="w-8 h-8 animate-spin text-cyan-500" />
              <p className="text-sm text-slate-600 dark:text-slate-400">Generating template...</p>
            </div>
          )}

          {/* Error State */}
          {error && !isLoading && (
            <div className="p-6">
              <div className="rounded-lg border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-4">
                <div className="flex items-start gap-3 mb-3">
                  <AlertTriangle className="w-5 h-5 text-red-600 dark:text-red-400 shrink-0 mt-0.5" />
                  <div className="flex-1">
                    <p className="text-sm font-medium text-red-800 dark:text-red-300">
                      Failed to generate template
                    </p>
                    <p className="text-sm text-red-700 dark:text-red-400 mt-1">{error}</p>
                  </div>
                </div>
                <button
                  type="button"
                  onClick={() => { void fetchTemplate() }}
                  className={cn(
                    'inline-flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-medium',
                    'bg-red-100 dark:bg-red-900 text-red-700 dark:text-red-300',
                    'hover:bg-red-200 dark:hover:bg-red-800',
                    'focus:outline-none focus:ring-2 focus:ring-red-500',
                  )}
                >
                  Retry
                </button>
              </div>
            </div>
          )}

          {/* Success State */}
          {template && !isLoading && !error && (
            <>
              {/* Instructions */}
              {instructions && (
                <InstructionsSection
                  summary={instructions.summary}
                  steps={instructions.steps}
                />
              )}

              {/* Template Section */}
              <div className="p-6 space-y-3">
                <div className="flex items-center justify-between">
                  <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
                    Template (Bicep)
                  </h3>
                  <button
                    type="button"
                    onClick={() => void handleCopyAll()}
                    className={cn(
                      'inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium',
                      'bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300',
                      'border border-slate-200 dark:border-slate-700',
                      'hover:bg-slate-200 dark:hover:bg-slate-700',
                      'focus:outline-none focus:ring-2 focus:ring-cyan-500',
                      'transition-colors duration-150',
                    )}
                    aria-label={copied ? 'Copied' : 'Copy all code'}
                  >
                    {copied ? '✓ Copied' : 'Copy All'}
                  </button>
                </div>

                {/* Code Block */}
                <div className="relative max-h-96 overflow-y-auto rounded-lg border border-slate-200 dark:border-slate-700">
                  <CodeBlock
                    code={template}
                    language="bicep"
                    onCopy={() => showToast('Code copied to clipboard', 'success')}
                    className="m-0"
                  />
                </div>
              </div>
            </>
          )}
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/60 shrink-0">
          <div className="flex items-center justify-between gap-3">
            <button
              type="button"
              onClick={handleDownload}
              disabled={!template || isLoading}
              className={cn(
                'inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold',
                'text-slate-700 dark:text-slate-300',
                'border border-slate-200 dark:border-slate-700',
                'hover:bg-slate-100 dark:hover:bg-slate-800',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2 dark:focus:ring-offset-slate-900',
                'disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-transparent',
                'transition-colors duration-150',
              )}
            >
              <Download className="w-4 h-4" />
              Download .bicep
            </button>

            <button
              type="button"
              onClick={onClose}
              className={cn(
                'inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold shadow-sm',
                'bg-cyan-600 text-white hover:bg-cyan-700',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2 dark:focus:ring-offset-slate-900',
                'transition-colors duration-150',
              )}
            >
              Close
            </button>
          </div>
        </div>
      </dialog>

      {/* Toast Container - positioned fixed for toasts */}
      <ToastContainer />
    </>
  )
}

/**
 * Simple toast container for displaying notifications
 */
function ToastContainer() {
  const { toasts, removeToast } = useToast()
  const { set: setToastTimeout, clearAll: clearToastTimeouts } = useTimeoutMap()

  // Auto-dismiss toasts after 3 seconds
  React.useEffect(() => {
    clearToastTimeouts()

    toasts.forEach(toast => {
      setToastTimeout(toast.id, () => removeToast(toast.id), 3000)
    })

    return () => clearToastTimeouts()
  }, [toasts, removeToast, clearToastTimeouts, setToastTimeout])

  return (
    <div className="fixed bottom-4 right-4 z-50 space-y-2 pointer-events-none" style={{ zIndex: 60 }}>
      {toasts.map(toast => (
        <div
          key={toast.id}
          className={cn(
            'px-4 py-3 rounded-lg shadow-lg border pointer-events-auto',
            'animate-fade-in',
            'max-w-sm',
            toast.type === 'success' && 'bg-emerald-50 dark:bg-emerald-950 border-emerald-200 dark:border-emerald-800 text-emerald-800 dark:text-emerald-300',
            toast.type === 'error' && 'bg-red-50 dark:bg-red-950 border-red-200 dark:border-red-800 text-red-800 dark:text-red-300',
            toast.type === 'info' && 'bg-blue-50 dark:bg-blue-950 border-blue-200 dark:border-blue-800 text-blue-800 dark:text-blue-300',
          )}
        >
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">{toast.message}</span>
            <button
              type="button"
              onClick={() => removeToast(toast.id)}
              className="ml-auto p-1 rounded hover:bg-black/5 dark:hover:bg-white/5"
              aria-label="Dismiss"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>
      ))}
    </div>
  )
}

export default BicepTemplateModal
