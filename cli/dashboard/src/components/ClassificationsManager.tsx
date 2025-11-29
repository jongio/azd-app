import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Trash2, Info, AlertTriangle, XCircle, Plus, X, Check } from 'lucide-react'
import type { LogClassification } from '@/hooks/useLogClassifications'

/**
 * Pending change to a classification (add, edit, or delete)
 */
export interface ClassificationChange {
  type: 'add' | 'edit' | 'delete'
  index?: number  // Original index for edit/delete
  classification: LogClassification
  originalClassification?: LogClassification  // For edit - to show what changed
}

interface ClassificationsEditorProps {
  classifications: LogClassification[]
  pendingChanges: ClassificationChange[]
  onAddChange: (change: ClassificationChange) => void
  onRemoveChange: (index: number) => void
}

interface ConfirmDeleteDialogProps {
  isOpen: boolean
  classification: LogClassification | null
  onConfirm: () => void
  onCancel: () => void
}

function ConfirmDeleteDialog({ isOpen, classification, onConfirm, onCancel }: ConfirmDeleteDialogProps) {
  if (!isOpen || !classification) return null

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-60" onClick={onCancel}>
      <div 
        className="rounded-lg shadow-xl max-w-md w-full border border-border bg-white dark:bg-neutral-900 p-6"
        onClick={(e) => e.stopPropagation()}
      >
        <h3 className="text-lg font-semibold text-foreground mb-4">Delete Classification?</h3>
        <p className="text-sm text-muted-foreground mb-2">
          This will mark the following classification for deletion:
        </p>
        <div className="p-3 bg-muted/50 rounded-lg border border-border mb-4">
          <span className="font-mono text-sm">"{classification.text}"</span>
          <span className="text-xs text-muted-foreground ml-2">→ {classification.level}</span>
        </div>
        <p className="text-sm text-muted-foreground mb-4">
          The change won't take effect until you click <strong>Save Changes</strong>.
        </p>
        <div className="flex gap-2 justify-end">
          <Button variant="outline" onClick={onCancel}>Cancel</Button>
          <Button className="bg-red-600 hover:bg-red-700 text-white" onClick={onConfirm}>Mark for Deletion</Button>
        </div>
      </div>
    </div>
  )
}

interface AddClassificationFormProps {
  onAdd: (classification: LogClassification) => void
  onCancel: () => void
}

function AddClassificationForm({ onAdd, onCancel }: AddClassificationFormProps) {
  const [text, setText] = useState('')
  const [level, setLevel] = useState<'info' | 'warning' | 'error'>('info')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (text.trim()) {
      onAdd({ text: text.trim(), level })
      setText('')
      setLevel('info')
    }
  }

  return (
    <form onSubmit={handleSubmit} className="p-3 bg-muted/50 rounded-lg border border-border space-y-3">
      <div>
        <label htmlFor="classification-text" className="text-xs font-medium text-muted-foreground">
          Text to match
        </label>
        <input
          id="classification-text"
          type="text"
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="e.g., Connection refused"
          className="mt-1 w-full px-3 py-2 text-sm border border-border rounded-md bg-white dark:bg-neutral-800 text-foreground"
          autoFocus
        />
      </div>
      <div>
        <label className="text-xs font-medium text-muted-foreground">Classification level</label>
        <div className="mt-1 flex gap-2">
          {(['info', 'warning', 'error'] as const).map((l) => (
            <button
              key={l}
              type="button"
              onClick={() => setLevel(l)}
              className={`flex-1 px-3 py-2 text-sm rounded-md border transition-colors ${
                level === l
                  ? l === 'error'
                    ? 'bg-red-100 border-red-300 text-red-700 dark:bg-red-900/30 dark:border-red-700 dark:text-red-400'
                    : l === 'warning'
                    ? 'bg-yellow-100 border-yellow-300 text-yellow-700 dark:bg-yellow-900/30 dark:border-yellow-700 dark:text-yellow-400'
                    : 'bg-blue-100 border-blue-300 text-blue-700 dark:bg-blue-900/30 dark:border-blue-700 dark:text-blue-400'
                  : 'bg-white dark:bg-neutral-800 border-border text-muted-foreground hover:bg-muted'
              }`}
            >
              {l.charAt(0).toUpperCase() + l.slice(1)}
            </button>
          ))}
        </div>
      </div>
      <div className="flex gap-2 justify-end">
        <Button type="button" variant="ghost" size="sm" onClick={onCancel}>
          <X className="w-4 h-4 mr-1" /> Cancel
        </Button>
        <Button type="submit" size="sm" disabled={!text.trim()}>
          <Check className="w-4 h-4 mr-1" /> Add
        </Button>
      </div>
    </form>
  )
}

export function ClassificationsEditor({ 
  classifications, 
  pendingChanges, 
  onAddChange, 
  onRemoveChange 
}: ClassificationsEditorProps) {
  const [showAddForm, setShowAddForm] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState<{ index: number; classification: LogClassification } | null>(null)

  const getLevelIcon = (level: string) => {
    switch (level) {
      case 'error':
        return <XCircle className="w-4 h-4 text-red-500" />
      case 'warning':
        return <AlertTriangle className="w-4 h-4 text-yellow-500" />
      default:
        return <Info className="w-4 h-4 text-blue-500" />
    }
  }

  const getLevelBadgeClass = (level: string) => {
    switch (level) {
      case 'error':
        return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
      case 'warning':
        return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
      default:
        return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
    }
  }

  // Check if an index is marked for deletion in pending changes
  const isMarkedForDeletion = (index: number) => {
    return pendingChanges.some(c => c.type === 'delete' && c.index === index)
  }

  // Get the pending change index for an item (to allow undoing)
  const getPendingChangeIndex = (originalIndex: number, type: 'delete' | 'edit') => {
    return pendingChanges.findIndex(c => c.type === type && c.index === originalIndex)
  }

  // Get pending adds
  const pendingAdds = pendingChanges.filter(c => c.type === 'add')

  const handleDeleteClick = (index: number, classification: LogClassification) => {
    setDeleteConfirm({ index, classification })
  }

  const handleConfirmDelete = () => {
    if (deleteConfirm) {
      onAddChange({
        type: 'delete',
        index: deleteConfirm.index,
        classification: deleteConfirm.classification
      })
      setDeleteConfirm(null)
    }
  }

  const handleAddClassification = (classification: LogClassification) => {
    onAddChange({
      type: 'add',
      classification
    })
    setShowAddForm(false)
  }

  return (
    <div className="space-y-4">
      {/* Existing classifications */}
      <div className="space-y-2">
        {classifications.length === 0 && pendingAdds.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            <p className="text-sm">No classifications defined.</p>
            <p className="text-xs mt-2">
              Click "Add Classification" below or select text in the logs.
            </p>
          </div>
        ) : (
          <>
            {classifications.map((classification, index) => {
              const markedForDeletion = isMarkedForDeletion(index)
              const pendingDeleteIndex = getPendingChangeIndex(index, 'delete')
              
              return (
                <div
                  key={index}
                  className={`flex items-center justify-between p-3 rounded-lg border transition-all ${
                    markedForDeletion
                      ? 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800 opacity-60'
                      : 'bg-muted/50 border-border'
                  }`}
                >
                  <div className="flex items-center gap-3 flex-1 min-w-0">
                    {getLevelIcon(classification.level)}
                    <span 
                      className={`font-mono text-sm truncate flex-1 ${markedForDeletion ? 'line-through' : ''}`} 
                      title={classification.text}
                    >
                      "{classification.text}"
                    </span>
                    <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${getLevelBadgeClass(classification.level)}`}>
                      {classification.level}
                    </span>
                  </div>
                  {markedForDeletion ? (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => onRemoveChange(pendingDeleteIndex)}
                      className="shrink-0 ml-2 text-muted-foreground hover:text-foreground"
                      title="Undo deletion"
                    >
                      Undo
                    </Button>
                  ) : (
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleDeleteClick(index, classification)}
                      className="shrink-0 ml-2 text-muted-foreground hover:text-destructive"
                      title="Delete classification"
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  )}
                </div>
              )
            })}
          </>
        )}

        {/* Pending adds */}
        {pendingAdds.map((change, changeIndex) => {
          const actualIndex = pendingChanges.indexOf(change)
          return (
            <div
              key={`pending-${changeIndex}`}
              className="flex items-center justify-between p-3 bg-green-50 dark:bg-green-900/20 rounded-lg border border-green-200 dark:border-green-800"
            >
              <div className="flex items-center gap-3 flex-1 min-w-0">
                <span className="text-xs text-green-600 dark:text-green-400 font-medium">NEW</span>
                {getLevelIcon(change.classification.level)}
                <span className="font-mono text-sm truncate flex-1" title={change.classification.text}>
                  "{change.classification.text}"
                </span>
                <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${getLevelBadgeClass(change.classification.level)}`}>
                  {change.classification.level}
                </span>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onRemoveChange(actualIndex)}
                className="shrink-0 ml-2 text-muted-foreground hover:text-foreground"
                title="Remove"
              >
                Undo
              </Button>
            </div>
          )
        })}
      </div>

      {/* Add form or button */}
      {showAddForm ? (
        <AddClassificationForm
          onAdd={handleAddClassification}
          onCancel={() => setShowAddForm(false)}
        />
      ) : (
        <Button
          variant="outline"
          size="sm"
          onClick={() => setShowAddForm(true)}
          className="w-full"
        >
          <Plus className="w-4 h-4 mr-2" /> Add Classification
        </Button>
      )}

      {/* Confirm delete dialog */}
      <ConfirmDeleteDialog
        isOpen={deleteConfirm !== null}
        classification={deleteConfirm?.classification ?? null}
        onConfirm={handleConfirmDelete}
        onCancel={() => setDeleteConfirm(null)}
      />
    </div>
  )
}

// Keep the simple list for backward compatibility if needed
// ClassificationsList is deprecated - use ClassificationsEditor instead
export function ClassificationsList(_props: { classifications: LogClassification[], onDelete: (index: number) => void }) {
  return null
}
