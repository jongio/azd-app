/**
 * Dialog - Modal dialog component using native HTML dialog
 * Provides accessible modal functionality with backdrop, focus management, and keyboard support
 */
import * as React from 'react'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'

interface DialogIdsContextValue {
  titleId: string
  descriptionId: string
}

const DialogIdsContext = React.createContext<DialogIdsContextValue | null>(null)

export interface DialogProps {
  isOpen: boolean
  onClose: () => void
  children: React.ReactNode
  title?: string
  description?: string
  className?: string
  maxWidth?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl'
}

/**
 * Dialog Component
 * 
 * Modal dialog with backdrop, focus trap, and keyboard support.
 * Uses native HTML dialog element for accessibility.
 */
export function Dialog({
  isOpen,
  onClose,
  children,
  title,
  description,
  className,
  maxWidth = '2xl',
}: DialogProps) {
  const dialogRef = React.useRef<HTMLDialogElement>(null)
  const dialogId = React.useId()
  const titleId = `${dialogId}-title`
  const descriptionId = `${dialogId}-description`

  // Handle escape key
  React.useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose()
      }
    }

    if (isOpen) {
      document.addEventListener('keydown', handleEscape)
    }

    return () => {
      document.removeEventListener('keydown', handleEscape)
    }
  }, [isOpen, onClose])

  // Focus management
  React.useEffect(() => {
    if (isOpen && dialogRef.current) {
      const closeButton = dialogRef.current.querySelector<HTMLButtonElement>('[data-close-button]')
      closeButton?.focus()
    }
  }, [isOpen])

  // Handle backdrop click
  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose()
    }
  }

  if (!isOpen) {
    return null
  }

  const maxWidthClasses = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-lg',
    xl: 'max-w-xl',
    '2xl': 'max-w-2xl',
    '3xl': 'max-w-3xl',
    '4xl': 'max-w-4xl',
  }

  return (
    <DialogIdsContext.Provider value={{ titleId, descriptionId }}>
      <>
        {/* Backdrop - positioned below dialog (z-40) */}
        <div
          className="fixed inset-0 z-40 bg-black/50 dark:bg-black/70 animate-fade-in pointer-events-auto"
          data-testid="dialog-backdrop"
          onClick={handleBackdropClick}
          aria-hidden="true"
          style={{ pointerEvents: 'auto' }}
        />

        {/* Dialog - positioned above backdrop (z-50) */}
        <dialog
          ref={dialogRef}
          open
          role="dialog"
          aria-labelledby={titleId}
          aria-describedby={description ? descriptionId : undefined}
          aria-modal="true"
          aria-label={title}
          className={cn(
            'fixed left-1/2 top-1/2 z-50 -translate-x-1/2 -translate-y-1/2',
            'w-full',
            'pointer-events-auto',
            maxWidthClasses[maxWidth],
            'bg-white dark:bg-slate-900',
            'border border-slate-200 dark:border-slate-700',
            'rounded-2xl shadow-2xl',
            'flex flex-col',
            'max-h-[90vh]',
            'animate-scale-in',
            className
          )}
        >
          {children}
        </dialog>
      </>
    </DialogIdsContext.Provider>
  )
}

export interface DialogHeaderProps {
  children: React.ReactNode
  onClose?: () => void
  className?: string
}

export function DialogHeader({ children, onClose, className }: DialogHeaderProps) {
  return (
    <div className={cn('flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-slate-700 shrink-0', className)}>
      <div className="flex-1">{children}</div>
      {onClose && (
        <button
          type="button"
          data-close-button
          onClick={onClose}
          className="p-2 -mr-2 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
          aria-label="Close dialog"
        >
          <X className="w-5 h-5" />
        </button>
      )}
    </div>
  )
}

export interface DialogTitleProps {
  children: React.ReactNode
  id?: string
  className?: string
}

export function DialogTitle({ children, className, id }: DialogTitleProps) {
  const ids = React.useContext(DialogIdsContext)
  const resolvedId = id ?? ids?.titleId ?? 'dialog-title'

  return (
    <h2 id={resolvedId} className={cn('text-lg font-semibold text-slate-900 dark:text-slate-100', className)}>
      {children}
    </h2>
  )
}

export interface DialogDescriptionProps {
  children: React.ReactNode
  id?: string
  className?: string
}

export function DialogDescription({ children, className, id }: DialogDescriptionProps) {
  const ids = React.useContext(DialogIdsContext)
  const resolvedId = id ?? ids?.descriptionId ?? 'dialog-description'

  return (
    <p id={resolvedId} className={cn('text-sm text-slate-600 dark:text-slate-400 mt-0.5', className)}>
      {children}
    </p>
  )
}

export interface DialogContentProps {
  children: React.ReactNode
  className?: string
}

export function DialogContent({ children, className }: DialogContentProps) {
  return <div className={cn('flex-1 overflow-y-auto p-6', className)}>{children}</div>
}

export interface DialogFooterProps {
  children: React.ReactNode
  className?: string
}

export function DialogFooter({ children, className }: DialogFooterProps) {
  return (
    <div className={cn('px-6 py-4 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/60 shrink-0', className)}>
      {children}
    </div>
  )
}
