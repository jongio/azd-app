/**
 * YAML Editor Header Component
 * 
 * Top bar with title, actions, and controls.
 * Features save/cancel buttons, preview toggle, help, theme toggle, etc.
 */

import * as React from 'react'
import { 
  Save, 
  X, 
  Eye, 
  EyeOff, 
  HelpCircle, 
  Keyboard,
  Loader2,
  CheckCircle2,
  AlertCircle,
  ArrowLeft,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { useEditorState } from './useEditorState'
import { useSchema } from '@/contexts/SchemaContext'
import { Sun, Moon } from 'lucide-react'

export interface YamlEditorHeaderProps {
  /** Custom className */
  className?: string
  /** Handler to open Add Service modal */
  onOpenAddService?: () => void
}

/**
 * YAML Editor Header Component
 */
export function YamlEditorHeader({ className, onOpenAddService }: YamlEditorHeaderProps) {
  const {
    isDirty,
    isSaving,
    isLoading,
    error,
    isPreviewVisible,
    isAddServiceModalOpen,
    saveConfig,
    discardChanges,
    togglePreview,
    toggleHelpPanel,
    openKeyboardShortcuts,
    validationErrors,
  } = useEditorState()

  const { rawSchema } = useSchema()
  const [saveSuccess, setSaveSuccess] = React.useState(false)

  // Handle save - validate against schema before saving
  const handleSave = React.useCallback(async () => {
    const success = await saveConfig(rawSchema as Record<string, unknown> | undefined)
    
    if (success) {
      setSaveSuccess(true)
      setTimeout(() => setSaveSuccess(false), 2000)
    }
  }, [saveConfig, rawSchema])

  // Handle cancel
  const handleCancel = React.useCallback(() => {
    if (window.confirm('Discard all changes?')) {
      discardChanges()
    }
  }, [discardChanges])

  // Handle close/back to dashboard
  const handleClose = React.useCallback(() => {
    if (isDirty) {
      if (!window.confirm('You have unsaved changes. Are you sure you want to leave?')) {
        return
      }
    }
    window.location.href = '/'
  }, [isDirty])

  // Count errors
  const errorCount = validationErrors.filter(e => e.level === 'error').length
  const warningCount = validationErrors.filter(e => e.level === 'warning').length

  // Theme toggle - syncs with dashboard theme
  const [isDark, setIsDark] = React.useState(() => {
    if (typeof window === 'undefined') return false
    return document.documentElement.classList.contains('dark')
  })

  React.useEffect(() => {
    const checkTheme = () => {
      setIsDark(document.documentElement.classList.contains('dark'))
    }

    const observer = new MutationObserver(checkTheme)
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class', 'data-theme']
    })
    
    return () => {
      observer.disconnect()
    }
  }, [])

  const handleThemeToggle = React.useCallback(() => {
    const newTheme = isDark ? 'light' : 'dark'
    document.documentElement.classList.toggle('dark', newTheme === 'dark')
    document.documentElement.setAttribute('data-theme', newTheme)
    localStorage.setItem('dashboard-theme', newTheme)
    setIsDark(!isDark)
  }, [isDark])

  return (
    <header 
      className={cn(
        'flex items-center justify-between px-6 py-3',
        'border-b border-slate-200 dark:border-slate-700',
        'bg-white dark:bg-slate-800',
        className
      )}
    >
      {/* Left: Back Button, Title and Status */}
      <div className="flex items-center gap-3">
        {/* Back to Dashboard Button */}
        <button
          type="button"
          onClick={handleClose}
          className={cn(
            'p-2 rounded-lg transition-all duration-200',
            'text-slate-500 dark:text-slate-400',
            'hover:text-slate-700 dark:hover:text-slate-200',
            'hover:bg-slate-100 dark:hover:bg-slate-800',
            'focus-visible:outline-none focus-visible:ring-2',
            'focus-visible:ring-cyan-500 focus-visible:ring-offset-2',
            'active:scale-95'
          )}
          aria-label="Back to Dashboard"
          title="Back to Dashboard"
        >
          <ArrowLeft className="w-5 h-5" />
        </button>

        <div className="h-6 w-px bg-slate-200 dark:bg-slate-700" />

        <h1 className="text-lg font-semibold flex items-center gap-2 text-slate-900 dark:text-slate-100">
          <span className="text-xl" role="img" aria-label="YAML">📝</span>
          <span>Edit azure.yaml</span>
        </h1>

        {/* Status Indicators */}
        {isLoading && (
          <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
            <Loader2 className="w-4 h-4 animate-spin text-cyan-500" />
            <span>Loading...</span>
          </div>
        )}

        {saveSuccess && (
          <div className="flex items-center gap-2 text-sm text-emerald-600 dark:text-emerald-400">
            <CheckCircle2 className="w-4 h-4" />
            <span>Saved</span>
          </div>
        )}

        {error && (
          <div className="flex items-center gap-2 text-sm text-rose-600 dark:text-rose-400">
            <AlertCircle className="w-4 h-4" />
            <span className="max-w-xs truncate">{error}</span>
          </div>
        )}

        {isDirty && !isSaving && !saveSuccess && (
          <span className="px-2.5 py-1 text-xs font-medium bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400 rounded-md border border-amber-200 dark:border-amber-800">
            Unsaved
          </span>
        )}

        {/* Validation Status */}
        {errorCount > 0 && (
          <span className="px-2.5 py-1 text-xs font-medium bg-rose-100 dark:bg-rose-900/30 text-rose-700 dark:text-rose-400 rounded-md border border-rose-200 dark:border-rose-800">
            {errorCount} error{errorCount !== 1 ? 's' : ''}
          </span>
        )}

        {errorCount === 0 && warningCount > 0 && (
          <span className="px-2.5 py-1 text-xs font-medium bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400 rounded-md border border-amber-200 dark:border-amber-800">
            {warningCount} warning{warningCount !== 1 ? 's' : ''}
          </span>
        )}
      </div>

      {/* Right: Actions */}
      <div className="flex items-center gap-2">
        {/* Add Service */}
        <Button
          variant="default"
          size="sm"
          onClick={() => {
            onOpenAddService?.()
          }}
          aria-hidden={isAddServiceModalOpen}
        >
          Add Service
        </Button>

        {/* Preview Toggle */}
        <Button
          variant={isPreviewVisible ? 'default' : 'outline'}
          size="sm"
          onClick={togglePreview}
          title={isPreviewVisible ? 'Hide preview (Ctrl+P)' : 'Show preview (Ctrl+P)'}
        >
          {isPreviewVisible ? (
            <>
              <Eye className="w-4 h-4 mr-2" />
              Preview
            </>
          ) : (
            <>
              <EyeOff className="w-4 h-4 mr-2" />
              Preview
            </>
          )}
        </Button>

        {/* Help Button */}
        <Button
          variant="outline"
          size="sm"
          onClick={() => toggleHelpPanel()}
          title="Help (F1)"
        >
          <HelpCircle className="w-4 h-4 mr-2" />
          Help
        </Button>

        {/* Keyboard Shortcuts Button */}
        <Button
          variant="outline"
          size="sm"
          onClick={openKeyboardShortcuts}
          title="Keyboard shortcuts (?)"
        >
          <Keyboard className="w-4 h-4" />
        </Button>

        {/* Theme Toggle - syncs with dashboard */}
        <button
          type="button"
          onClick={handleThemeToggle}
          className={cn(
            'p-2 rounded-lg transition-all duration-200',
            'text-slate-500 dark:text-slate-400',
            'hover:text-slate-700 dark:hover:text-slate-200',
            'hover:bg-slate-100 dark:hover:bg-slate-800',
            'focus-visible:outline-none focus-visible:ring-2',
            'focus-visible:ring-cyan-500 focus-visible:ring-offset-2',
            'active:scale-95'
          )}
          aria-label={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
          title={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
        >
          {isDark ? (
            <Sun className="w-[18px] h-[18px]" />
          ) : (
            <Moon className="w-[18px] h-[18px]" />
          )}
        </button>

        {/* Divider */}
        <div className="w-px h-6 bg-border" />

        {/* Cancel Button */}
        <Button
          variant="outline"
          size="sm"
          onClick={handleCancel}
          disabled={!isDirty || isSaving}
          title="Discard changes (Esc)"
        >
          <X className="w-4 h-4 mr-2" />
          Cancel
        </Button>

        {/* Save Button */}
        <Button
          variant="default"
          size="sm"
          onClick={handleSave}
          disabled={!isDirty || isSaving || isLoading || errorCount > 0}
          title="Save changes (Ctrl+S)"
        >
          {isSaving ? (
            <>
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              Saving...
            </>
          ) : (
            <>
              <Save className="w-4 h-4 mr-2" />
              Save
            </>
          )}
        </Button>
      </div>
    </header>
  )
}
