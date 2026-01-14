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
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { ThemeToggle } from '@/lib/editor/theme-provider'
import { cn } from '@/lib/utils'
import { useEditorState } from './useEditorState'

export interface YamlEditorHeaderProps {
  /** Custom className */
  className?: string
}

/**
 * YAML Editor Header Component
 */
export function YamlEditorHeader({ className }: YamlEditorHeaderProps) {
  const {
    isDirty,
    isSaving,
    isLoading,
    error,
    isPreviewVisible,
    saveConfig,
    discardChanges,
    togglePreview,
    toggleHelpPanel,
    openKeyboardShortcuts,
    validationErrors,
  } = useEditorState()

  const [saveSuccess, setSaveSuccess] = React.useState(false)

  // Handle save
  const handleSave = React.useCallback(async () => {
    const success = await saveConfig()
    
    if (success) {
      setSaveSuccess(true)
      setTimeout(() => setSaveSuccess(false), 2000)
    }
  }, [saveConfig])

  // Handle cancel
  const handleCancel = React.useCallback(() => {
    if (window.confirm('Discard all changes?')) {
      discardChanges()
    }
  }, [discardChanges])

  // Count errors
  const errorCount = validationErrors.filter(e => e.level === 'error').length
  const warningCount = validationErrors.filter(e => e.level === 'warning').length

  return (
    <header 
      className={cn(
        'flex items-center justify-between px-6 py-3 border-b border-border bg-background',
        className
      )}
    >
      {/* Left: Title and Status */}
      <div className="flex items-center gap-4">
        <h1 className="text-xl font-semibold flex items-center gap-2">
          <span className="text-2xl" role="img" aria-label="YAML">📝</span>
          <span>Azure YAML Editor</span>
        </h1>

        {/* Status Indicators */}
        {isLoading && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="w-4 h-4 animate-spin" />
            <span>Loading...</span>
          </div>
        )}

        {saveSuccess && (
          <div className="flex items-center gap-2 text-sm text-green-600 dark:text-green-400">
            <CheckCircle2 className="w-4 h-4" />
            <span>Saved successfully</span>
          </div>
        )}

        {error && (
          <div className="flex items-center gap-2 text-sm text-rose-600 dark:text-rose-400">
            <AlertCircle className="w-4 h-4" />
            <span>{error}</span>
          </div>
        )}

        {isDirty && !isSaving && !saveSuccess && (
          <span className="px-2 py-1 text-xs font-medium bg-amber-100 dark:bg-amber-900/20 text-amber-600 dark:text-amber-400 rounded">
            Unsaved changes
          </span>
        )}

        {/* Validation Status */}
        {errorCount > 0 && (
          <span className="px-2 py-1 text-xs font-medium bg-rose-100 dark:bg-rose-900/20 text-rose-600 dark:text-rose-400 rounded">
            {errorCount} error{errorCount !== 1 ? 's' : ''}
          </span>
        )}

        {errorCount === 0 && warningCount > 0 && (
          <span className="px-2 py-1 text-xs font-medium bg-amber-100 dark:bg-amber-900/20 text-amber-600 dark:text-amber-400 rounded">
            {warningCount} warning{warningCount !== 1 ? 's' : ''}
          </span>
        )}
      </div>

      {/* Right: Actions */}
      <div className="flex items-center gap-2">
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

        {/* Theme Toggle */}
        <ThemeToggle />

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
