/**
 * Import Modal Component
 * Modal dialog for importing configurations with multiple sources and merge strategies
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
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { TemplateTab } from './TemplateTab'
import { FileUploadTab } from './FileUploadTab'
import { PasteYamlTab } from './PasteYamlTab'
import { MergeStrategySelector } from './MergeStrategySelector'
import { ImportPreviewPane } from './ImportPreviewPane'
import { CherryPickSelector } from './CherryPickSelector'
import { cn } from '@/lib/utils'
import { parseYaml } from '@/lib/editor/yaml-utils'
import { mergeConfigurations, generateDiff, extractCherryPickSections, applyCherryPick } from '@/lib/editor/import-export-utils'
import type { ImportSource, MergeStrategy, ImportPreview, CherryPickSection } from '@/lib/editor/import-export-types'
import { AlertTriangle } from 'lucide-react'

export interface ImportModalProps {
  /** Whether modal is open */
  isOpen: boolean
  /** Callback to close modal */
  onClose: () => void
  /** Callback when import is confirmed */
  onImport: (config: Record<string, unknown>) => void | Promise<void>
  /** Current configuration */
  currentConfig: Record<string, unknown>
}

/**
 * Import Modal Component
 */
export function ImportModal({
  isOpen,
  onClose,
  onImport,
  currentConfig,
}: ImportModalProps) {
  const [activeTab, setActiveTab] = React.useState<ImportSource>('template')
  const [mergeStrategy, setMergeStrategy] = React.useState<MergeStrategy>('merge')
  const [importedYaml, setImportedYaml] = React.useState<string>('')
  const [importedConfig, setImportedConfig] = React.useState<Record<string, unknown> | null>(null)
  const [parseError, setParseError] = React.useState<string | null>(null)
  const [validationErrors, setValidationErrors] = React.useState<string[]>([])
  const [showPreview, setShowPreview] = React.useState(false)
  const [cherryPickSections, setCherryPickSections] = React.useState<CherryPickSection[]>([])
  const [isSubmitting, setIsSubmitting] = React.useState(false)
  const [showReplaceWarning, setShowReplaceWarning] = React.useState(false)

  // Reset state when modal closes
  React.useEffect(() => {
    if (!isOpen) {
      setActiveTab('template')
      setMergeStrategy('merge')
      setImportedYaml('')
      setImportedConfig(null)
      setParseError(null)
      setValidationErrors([])
      setShowPreview(false)
      setCherryPickSections([])
      setIsSubmitting(false)
      setShowReplaceWarning(false)
    }
  }, [isOpen])

  // Parse imported YAML when it changes
  React.useEffect(() => {
    if (!importedYaml) {
      setImportedConfig(null)
      setParseError(null)
      return
    }

    const result = parseYaml(importedYaml)
    if (result.success && result.data) {
      setImportedConfig(result.data as Record<string, unknown>)
      setParseError(null)

      // Extract cherry-pick sections if strategy is cherry-pick
      if (mergeStrategy === 'cherry-pick') {
        const sections = extractCherryPickSections(result.data as Record<string, unknown>)
        setCherryPickSections(sections)
      }
    } else {
      setImportedConfig(null)
      setParseError(result.error || 'Failed to parse YAML')
    }
  }, [importedYaml, mergeStrategy])

  // Update cherry-pick sections when merge strategy changes
  React.useEffect(() => {
    if (mergeStrategy === 'cherry-pick' && importedConfig) {
      const sections = extractCherryPickSections(importedConfig)
      setCherryPickSections(sections)
    } else {
      setCherryPickSections([])
    }
  }, [mergeStrategy, importedConfig])

  // Generate preview
  const preview = React.useMemo<ImportPreview | null>(() => {
    if (!importedConfig) return null

    const merged = mergeStrategy === 'cherry-pick'
      ? applyCherryPick(currentConfig, importedConfig, cherryPickSections)
      : mergeConfigurations(currentConfig, importedConfig, mergeStrategy)

    const diff = generateDiff(currentConfig, importedConfig)

    return {
      current: JSON.stringify(currentConfig, null, 2),
      imported: importedYaml,
      merged: JSON.stringify(merged, null, 2),
      diff,
    }
  }, [currentConfig, importedConfig, importedYaml, mergeStrategy, cherryPickSections])

  // Handle preview click
  const handlePreviewClick = () => {
    if (mergeStrategy === 'replace') {
      setShowReplaceWarning(true)
    } else {
      setShowPreview(true)
    }
  }

  // Handle replace confirmation
  const handleReplaceConfirm = () => {
    setShowReplaceWarning(false)
    setShowPreview(true)
  }

  // Handle import
  const handleImport = async () => {
    if (!importedConfig) return

    try {
      setIsSubmitting(true)

      const merged = mergeStrategy === 'cherry-pick'
        ? applyCherryPick(currentConfig, importedConfig, cherryPickSections)
        : mergeConfigurations(currentConfig, importedConfig, mergeStrategy)

      await onImport(merged)
      onClose()
    } catch {
      setValidationErrors(['Failed to import configuration. Please try again.'])
    } finally {
      setIsSubmitting(false)
    }
  }

  // Handle cherry-pick selection toggle
  const handleCherryPickToggle = (id: string) => {
    setCherryPickSections(prev =>
      prev.map(section =>
        section.id === id ? { ...section, selected: !section.selected } : section
      )
    )
  }

  const canImport = importedConfig !== null && parseError === null && !isSubmitting

  return (
    <>
      <Dialog isOpen={isOpen && !showPreview} onClose={onClose} maxWidth="4xl">
        <DialogHeader onClose={onClose}>
          <DialogTitle>Import Configuration</DialogTitle>
          <DialogDescription>
            Import azure.yaml configuration from a template, file, or paste directly
          </DialogDescription>
        </DialogHeader>

        <DialogContent className="p-0">
          <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as ImportSource)} className="w-full">
            <div className="px-6 pt-4 border-b border-slate-200 dark:border-slate-700">
              <TabsList className="w-full justify-start">
                <TabsTrigger value="template">Templates</TabsTrigger>
                <TabsTrigger value="file">File Upload</TabsTrigger>
                <TabsTrigger value="paste">Paste YAML</TabsTrigger>
              </TabsList>
            </div>

            <div className="px-6 py-4 space-y-4">
              <TabsContent value="template" className="mt-0">
                <TemplateTab onSelectTemplate={setImportedYaml} />
              </TabsContent>

              <TabsContent value="file" className="mt-0">
                <FileUploadTab onFileLoad={setImportedYaml} />
              </TabsContent>

              <TabsContent value="paste" className="mt-0">
                <PasteYamlTab value={importedYaml} onChange={setImportedYaml} />
              </TabsContent>

              {/* Parse Error */}
              {parseError && (
                <div className="p-4 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
                  <div className="flex items-start gap-3">
                    <AlertTriangle className="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
                    <div>
                      <h4 className="text-sm font-semibold text-red-900 dark:text-red-100">
                        Parse Error
                      </h4>
                      <p className="text-sm text-red-700 dark:text-red-300 mt-1">{parseError}</p>
                    </div>
                  </div>
                </div>
              )}

              {/* Merge Strategy Selector */}
              {importedConfig && !parseError && (
                <div className="space-y-4">
                  <MergeStrategySelector value={mergeStrategy} onChange={setMergeStrategy} />

                  {/* Cherry-Pick Selector */}
                  {mergeStrategy === 'cherry-pick' && cherryPickSections.length > 0 && (
                    <CherryPickSelector
                      sections={cherryPickSections}
                      onToggle={handleCherryPickToggle}
                    />
                  )}
                </div>
              )}

              {/* Validation Errors */}
              {validationErrors.length > 0 && (
                <div className="p-4 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800">
                  <div className="flex items-start gap-3">
                    <AlertTriangle className="w-5 h-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />
                    <div>
                      <h4 className="text-sm font-semibold text-amber-900 dark:text-amber-100">
                        Validation Issues
                      </h4>
                      <ul className="text-sm text-amber-700 dark:text-amber-300 mt-1 space-y-1">
                        {validationErrors.map((error, i) => (
                          <li key={i}>• {error}</li>
                        ))}
                      </ul>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </Tabs>
        </DialogContent>

        <DialogFooter>
          <div className="flex items-center justify-end w-full gap-3">
            <button
              type="button"
              onClick={onClose}
              className={cn(
                'px-4 py-2 rounded-lg text-sm font-semibold',
                'text-slate-700 dark:text-slate-300',
                'border border-slate-200 dark:border-slate-700',
                'hover:bg-slate-100 dark:hover:bg-slate-800',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
                'transition-colors duration-150'
              )}
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handlePreviewClick}
              disabled={!canImport}
              className={cn(
                'px-6 py-2 rounded-lg text-sm font-semibold shadow-sm',
                'bg-cyan-600 text-white hover:bg-cyan-700',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
                'disabled:opacity-50 disabled:cursor-not-allowed',
                'transition-colors duration-150'
              )}
            >
              Preview
            </button>
          </div>
        </DialogFooter>
      </Dialog>

      {/* Replace Warning Modal */}
      {showReplaceWarning && (
        <Dialog isOpen={showReplaceWarning} onClose={() => setShowReplaceWarning(false)} maxWidth="md">
          <DialogHeader onClose={() => setShowReplaceWarning(false)}>
            <DialogTitle>Confirm Replace</DialogTitle>
          </DialogHeader>

          <DialogContent>
            <div className="flex items-start gap-3">
              <AlertTriangle className="w-6 h-6 text-amber-600 dark:text-amber-400 flex-shrink-0" />
              <div>
                <p className="text-sm text-slate-700 dark:text-slate-300">
                  You are about to <strong>replace</strong> your entire current configuration. This action cannot be undone (though a backup will be created).
                </p>
                <p className="text-sm text-slate-700 dark:text-slate-300 mt-2">
                  Are you sure you want to proceed?
                </p>
              </div>
            </div>
          </DialogContent>

          <DialogFooter>
            <div className="flex items-center justify-end w-full gap-3">
              <button
                type="button"
                onClick={() => setShowReplaceWarning(false)}
                className={cn(
                  'px-4 py-2 rounded-lg text-sm font-semibold',
                  'text-slate-700 dark:text-slate-300',
                  'border border-slate-200 dark:border-slate-700',
                  'hover:bg-slate-100 dark:hover:bg-slate-800',
                  'transition-colors duration-150'
                )}
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleReplaceConfirm}
                className={cn(
                  'px-6 py-2 rounded-lg text-sm font-semibold shadow-sm',
                  'bg-amber-600 text-white hover:bg-amber-700',
                  'transition-colors duration-150'
                )}
              >
                Continue
              </button>
            </div>
          </DialogFooter>
        </Dialog>
      )}

      {/* Preview Modal */}
      {showPreview && preview && (
        <ImportPreviewPane
          isOpen={showPreview}
          onClose={() => setShowPreview(false)}
          onConfirm={handleImport}
          preview={preview}
          isSubmitting={isSubmitting}
        />
      )}
    </>
  )
}
