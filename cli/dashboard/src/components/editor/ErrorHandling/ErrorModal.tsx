/**
 * ErrorModal - Display critical errors with actions
 */

import { AlertCircle, Copy } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import type { ModalErrorOptions } from '../../../lib/errors'

function ensureClipboardWritable(): void {
  const nav = (globalThis as typeof globalThis & { navigator?: Navigator }).navigator
  if (!nav) {
    return
  }

  try {
    const currentClipboard = (nav as Navigator).clipboard
    Object.defineProperty(nav, 'clipboard', {
      configurable: true,
      writable: true,
      value: currentClipboard,
    })
  } catch {
    // Ignore if the environment disallows redefining clipboard
  }
}

ensureClipboardWritable()

function getClipboard(): Navigator['clipboard'] | undefined {
  ensureClipboardWritable()
  return (globalThis as typeof globalThis & { navigator?: Navigator }).navigator?.clipboard
}

interface ErrorModalProps extends ModalErrorOptions {
  isOpen: boolean
  onClose: () => void
}

export function ErrorModal({
  isOpen,
  onClose,
  title,
  message,
  technicalDetails,
  actions = [],
  dismissible = true,
}: ErrorModalProps) {
  const [showDetails, setShowDetails] = useState(false)
  const [copied, setCopied] = useState(false)
  const resetCopyTimerRef = useRef<number | null>(null)

  const handleCopyDetails = async () => {
    const details = `${title}\n\n${message}\n\nTechnical Details:\n${technicalDetails || 'None'}`

    try {
      const clipboard = getClipboard()
      if (!clipboard?.writeText) {
        throw new Error('Clipboard API unavailable')
      }

      await clipboard.writeText(details)
    } catch (error) {
      // Silently fail - copy is not critical
    }

    setCopied(true)

    if (resetCopyTimerRef.current) {
      clearTimeout(resetCopyTimerRef.current)
    }

    resetCopyTimerRef.current = window.setTimeout(() => {
      setCopied(false)
      resetCopyTimerRef.current = null
    }, 2000)
  }

  useEffect(() => {
    return () => {
      if (resetCopyTimerRef.current) {
        clearTimeout(resetCopyTimerRef.current)
      }
    }
  }, [])

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (dismissible && e.target === e.currentTarget) {
      onClose()
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (dismissible && e.key === 'Escape') {
      onClose()
    }
  }

  if (!isOpen) {
    return null
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      style={{ animation: 'fade-in 0.2s ease-out' }}
      onClick={handleBackdropClick}
      onKeyDown={handleKeyDown}
      role="dialog"
      aria-modal="true"
      aria-labelledby="error-modal-title"
      tabIndex={-1}
    >
      <div
        className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full mx-4"
        style={{ animation: 'fade-in-up 0.2s ease-out' }}
      >
        {/* Header */}
        <div className="flex items-start gap-3 p-6 border-b border-gray-200 dark:border-gray-700">
          <AlertCircle className="w-6 h-6 text-red-600 dark:text-red-400 flex-shrink-0" />
          <h2 id="error-modal-title" className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            {title}
          </h2>
        </div>

        {/* Body */}
        <div className="p-6 space-y-4">
          <p className="text-sm text-gray-700 dark:text-gray-300">{message}</p>

          {technicalDetails && (
            <div className="space-y-2">
              <button
                onClick={() => setShowDetails(!showDetails)}
                className="text-sm text-blue-600 dark:text-blue-400 hover:underline"
              >
                {showDetails ? 'Hide' : 'Show'} technical details
              </button>

              {showDetails && (
                <div className="relative">
                  <pre className="text-xs bg-gray-100 dark:bg-gray-900 p-3 rounded overflow-x-auto text-gray-800 dark:text-gray-200">
                    {technicalDetails}
                  </pre>
                  <button
                    onClick={handleCopyDetails}
                    className="absolute top-2 right-2 p-1 rounded bg-white dark:bg-gray-800 shadow-sm hover:bg-gray-50 dark:hover:bg-gray-700"
                    aria-label="Copy error details"
                  >
                    <Copy className="w-4 h-4" />
                  </button>
                  {copied && (
                    <span className="absolute top-2 right-12 text-xs text-green-600 dark:text-green-400">
                      Copied!
                    </span>
                  )}
                </div>
              )}
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-2 p-6 border-t border-gray-200 dark:border-gray-700">
          {actions.map((action, index) => (
            <button
              key={index}
              onClick={() => {
                action.onClick()
                if (dismissible) {
                  onClose()
                }
              }}
              className={`
                px-4 py-2 rounded-md text-sm font-medium transition-colors
                ${
                  action.variant === 'primary'
                    ? 'bg-blue-600 text-white hover:bg-blue-700'
                    : action.variant === 'danger'
                      ? 'bg-red-600 text-white hover:bg-red-700'
                      : 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100 hover:bg-gray-300 dark:hover:bg-gray-600'
                }
              `}
            >
              {action.label}
            </button>
          ))}
          {dismissible && actions.length === 0 && (
            <button
              onClick={onClose}
              className="px-4 py-2 rounded-md text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-100 hover:bg-gray-300 dark:hover:bg-gray-600"
            >
              Close
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
