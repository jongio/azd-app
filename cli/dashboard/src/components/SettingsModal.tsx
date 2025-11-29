import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { X, Save, RotateCcw } from 'lucide-react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ClassificationsEditor, type ClassificationChange } from './ClassificationsManager'
import { useLogClassifications } from '@/hooks/useLogClassifications'
import { usePreferences } from '@/hooks/usePreferences'

interface SettingsModalProps {
  isOpen: boolean
  onClose: () => void
}

export function SettingsModal({ isOpen, onClose }: SettingsModalProps) {
  const { classifications, addClassification, deleteClassification, reload } = useLogClassifications()
  const { preferences, updateUI } = usePreferences()
  const [activeTab, setActiveTab] = useState<string>('general')
  const [pendingChanges, setPendingChanges] = useState<ClassificationChange[]>([])
  const [isSaving, setIsSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)

  // Reset pending changes and reload classifications when modal opens
  useEffect(() => {
    if (isOpen) {
      setPendingChanges([])
      setSaveError(null)
      // Force reload classifications when modal opens
      void reload()
    }
  }, [isOpen, reload])

  if (!isOpen) return null

  const hasUnsavedChanges = pendingChanges.length > 0

  const handleAddChange = (change: ClassificationChange) => {
    setPendingChanges(prev => [...prev, change])
    setSaveError(null)
  }

  const handleRemoveChange = (index: number) => {
    setPendingChanges(prev => prev.filter((_, i) => i !== index))
  }

  const handleDiscardChanges = () => {
    setPendingChanges([])
    setSaveError(null)
  }

  const handleSaveChanges = async () => {
    if (pendingChanges.length === 0) return

    setIsSaving(true)
    setSaveError(null)

    try {
      // Process changes in order: deletes first (in reverse index order), then adds
      const deletes = pendingChanges
        .filter(c => c.type === 'delete')
        .sort((a, b) => (b.index ?? 0) - (a.index ?? 0)) // Reverse order to preserve indices
      
      const adds = pendingChanges.filter(c => c.type === 'add')

      // Process deletes (skip reload/notify until all done)
      for (const change of deletes) {
        if (change.index !== undefined) {
          await deleteClassification(change.index, true) // skipNotify = true
        }
      }

      // Process adds (skip reload/notify until all done)
      for (const change of adds) {
        await addClassification(change.classification.text, change.classification.level, true) // skipNotify = true
      }

      // Now reload once and notify all instances
      await reload()
      setPendingChanges([])
    } catch (err) {
      console.error('Failed to save changes:', err)
      setSaveError(err instanceof Error ? err.message : 'Failed to save changes')
      // Reload to get current state
      await reload()
    } finally {
      setIsSaving(false)
    }
  }

  const handleClose = () => {
    if (hasUnsavedChanges) {
      // Could show a confirmation dialog here, but for now just discard
      setPendingChanges([])
    }
    onClose()
  }

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50" onClick={handleClose}>
      <div className="rounded-lg shadow-xl max-w-3xl w-full max-h-[80vh] overflow-hidden border border-border bg-white dark:bg-neutral-900" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between p-4 border-b border-border">
          <h2 className="text-lg font-semibold text-foreground">Logs Dashboard Settings</h2>
          <Button variant="ghost" size="icon" onClick={handleClose}>
            <X className="w-4 h-4" />
          </Button>
        </div>

        <div className="p-4 overflow-y-auto max-h-[calc(80vh-140px)]">
          <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="general">General</TabsTrigger>
              <TabsTrigger value="classifications">
                Log Classifications
                {hasUnsavedChanges && (
                  <span className="ml-2 w-2 h-2 bg-yellow-500 rounded-full" title="Unsaved changes" />
                )}
              </TabsTrigger>
            </TabsList>
            
            <TabsContent value="general" className="mt-4 space-y-4">
              <div className="flex items-center justify-between">
                <label htmlFor="gridColumns" className="text-sm font-medium">Grid Columns</label>
                <input
                  id="gridColumns"
                  type="number"
                  min={1}
                  max={6}
                  value={preferences.ui.gridColumns}
                  onChange={(e) => {
                    const val = parseInt(e.target.value)
                    if (val >= 1 && val <= 6) {
                      updateUI({ gridColumns: val })
                    }
                  }}
                  className="w-20 px-3 py-1.5 text-sm border border-border rounded-md bg-white dark:bg-neutral-800 text-foreground text-center"
                />
              </div>
            </TabsContent>
            
            <TabsContent value="classifications" className="mt-4">
              <div className="space-y-4">
                <div className="text-sm text-muted-foreground">
                  <p>Log classifications are stored in your project's <code className="bg-muted px-1 rounded">azure.yaml</code> file.</p>
                  <p className="mt-1">Add classifications below or select text in the logs to classify it.</p>
                </div>
                
                {saveError && (
                  <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-600 dark:text-red-400">
                    {saveError}
                  </div>
                )}

                <ClassificationsEditor
                  classifications={classifications}
                  pendingChanges={pendingChanges}
                  onAddChange={handleAddChange}
                  onRemoveChange={handleRemoveChange}
                />
              </div>
            </TabsContent>
          </Tabs>
        </div>

        {/* Footer with Save/Discard buttons - only show when on classifications tab with changes */}
        {activeTab === 'classifications' && hasUnsavedChanges && (
          <div className="flex items-center justify-between p-4 border-t border-border bg-muted/30">
            <div className="text-sm text-muted-foreground">
              {pendingChanges.length} unsaved change{pendingChanges.length !== 1 ? 's' : ''}
            </div>
            <div className="flex gap-2">
              <Button 
                variant="outline" 
                size="sm" 
                onClick={handleDiscardChanges}
                disabled={isSaving}
              >
                <RotateCcw className="w-4 h-4 mr-2" />
                Discard
              </Button>
              <Button 
                size="sm" 
                onClick={() => { void handleSaveChanges() }}
                disabled={isSaving}
              >
                <Save className="w-4 h-4 mr-2" />
                {isSaving ? 'Saving...' : 'Save Changes'}
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
